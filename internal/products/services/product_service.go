package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/geocoding"
	"github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/hfleury/horsemarketplacebk/internal/products/repositories"
	"github.com/hfleury/horsemarketplacebk/internal/system"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrUnauthorized    = errors.New("unauthorized to modify this product")
)

type ProductService interface {
	Create(ctx context.Context, product *models.Product) (*models.Product, error)
	FindByID(ctx context.Context, id string) (*models.Product, error)
	FindAll(ctx context.Context, filters map[string]any, page, limit int) (*models.PaginatedProducts, error)
	UpdateStatus(ctx context.Context, id string, status models.ProductStatus, userID string, isAdmin bool) error
	Delete(ctx context.Context, id string, userID string, isAdmin bool) error
	// Specific searches
	Search(ctx context.Context, query string, categoryID string, fieldMap map[string]string, horseFilter *models.HorseFilter, locationFilter *models.LocationFilter, page, limit int) (*models.PaginatedProducts, error)
}

type ProductServiceImp struct {
	repo            repositories.ProductRepository
	settingsRepo    system.SettingsRepository
	logger          config.Logging
	geocodingClient geocoding.Client
}

func NewProductService(repo repositories.ProductRepository, settingsRepo system.SettingsRepository, logger config.Logging, geocodingClient geocoding.Client) *ProductServiceImp {
	return &ProductServiceImp{
		repo:            repo,
		settingsRepo:    settingsRepo,
		logger:          logger,
		geocodingClient: geocodingClient,
	}
}

func (s *ProductServiceImp) Create(ctx context.Context, product *models.Product) (*models.Product, error) {
	// 1. Determine initial status
	approvalRequired, err := s.settingsRepo.IsProductApprovalRequired(ctx)
	if err != nil {
		// Log warning, default to approval required for safety?
		approvalRequired = true
	}

	if product.Status == models.StatusPublished {
		if approvalRequired {
			product.Status = models.StatusPendingApproval
		} else {
			product.Status = models.StatusPublished
		}
	} else {
		product.Status = models.StatusDraft
	}

	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()

	if (product.Latitude == nil || product.Longitude == nil) && product.City != nil && *product.City != "" {
		s.geocodeProductLocation(ctx, product)
	}

	return s.repo.Create(ctx, product)
}

// geocodeProductLocation is a best-effort fallback for when the frontend
// couldn't supply coordinates from the browser's geolocation API. Failures
// are logged and swallowed — creation must never fail because geocoding did.
func (s *ProductServiceImp) geocodeProductLocation(ctx context.Context, product *models.Product) {
	parts := []string{}
	if product.Area != nil && *product.Area != "" {
		parts = append(parts, *product.Area)
	}
	parts = append(parts, *product.City, "Sweden")
	query := strings.Join(parts, ", ")

	coordinates, err := s.geocodingClient.Geocode(ctx, query)
	if err != nil {
		s.logger.Log(ctx, config.WarnLevel, "Failed to geocode product location", map[string]any{"error": err.Error(), "query": query})
		return
	}

	product.Latitude = &coordinates.Lat
	product.Longitude = &coordinates.Lng
}

func (s *ProductServiceImp) FindByID(ctx context.Context, id string) (*models.Product, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ProductServiceImp) FindAll(ctx context.Context, filters map[string]any, page, limit int) (*models.PaginatedProducts, error) {
	items, total, err := s.repo.FindAll(ctx, filters, page, limit)
	if err != nil {
		return nil, err
	}
	return &models.PaginatedProducts{Items: items, Total: total, Page: page, Limit: limit}, nil
}

func (s *ProductServiceImp) UpdateStatus(ctx context.Context, id string, status models.ProductStatus, userID string, isAdmin bool) error {
	// 1. Get existing product
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProductNotFound
	}

	// 2. Check permissions
	// Admin can do anything.
	// User can only change to:
	// - Draft -> Published (triggers approval check again)
	// - Published -> Draft
	// - Any -> Sold
	// - Any -> Deleted

	if !isAdmin {
		if p.UserID.String() != userID {
			return ErrUnauthorized
		}

		// If user is trying to publish, check approval again
		if status == models.StatusPublished {
			approvalRequired, _ := s.settingsRepo.IsProductApprovalRequired(ctx)
			if approvalRequired {
				status = models.StatusPendingApproval // Redirect status
			}
		}

		// Prevent user from approving their own product (setting status to Published or Live directly if they are pending)
		// But valid transitions are allowed.
	}

	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *ProductServiceImp) Delete(ctx context.Context, id string, userID string, isAdmin bool) error {
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrProductNotFound
	}

	if !isAdmin && p.UserID.String() != userID {
		return ErrUnauthorized
	}

	return s.repo.Delete(ctx, id)
}

func (s *ProductServiceImp) Search(ctx context.Context, query string, categoryID string, fieldMap map[string]string, horseFilter *models.HorseFilter, locationFilter *models.LocationFilter, page, limit int) (*models.PaginatedProducts, error) {
	// Iterate field map if provided (vehicle/equipment model/make filtering — unchanged, single-key-wins behavior preserved)
	for k, v := range fieldMap {
		items, err := s.repo.FindByField(ctx, k, v)
		if err != nil {
			return nil, err
		}
		return &models.PaginatedProducts{Items: items, Total: len(items), Page: 1, Limit: len(items)}, nil
	}

	items, total, err := s.repo.SearchByFilter(ctx, categoryID, query, horseFilter, locationFilter, page, limit)
	if err != nil {
		return nil, err
	}
	return &models.PaginatedProducts{Items: items, Total: total, Page: page, Limit: limit}, nil
}
