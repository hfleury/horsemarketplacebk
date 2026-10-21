package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
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
	FindSimilar(ctx context.Context, id string, limit int) ([]*models.Product, error)
}

// ListingPublishedNotifier is told when a listing becomes visible to buyers.
type ListingPublishedNotifier interface {
	NotifyListingPublished(ctx context.Context, productID string) error
}

type ProductServiceImp struct {
	repo            repositories.ProductRepository
	settingsRepo    system.SettingsRepository
	logger          config.Logging
	geocodingClient geocoding.Client
	publishNotifier ListingPublishedNotifier
	tracker         analytics.Tracker
}

func NewProductService(repo repositories.ProductRepository, settingsRepo system.SettingsRepository, logger config.Logging, geocodingClient geocoding.Client) *ProductServiceImp {
	return &ProductServiceImp{
		repo:            repo,
		settingsRepo:    settingsRepo,
		logger:          logger,
		geocodingClient: geocodingClient,
		tracker:         analytics.NoopClient{},
	}
}

func (s *ProductServiceImp) SetPublishNotifier(notifier ListingPublishedNotifier) {
	s.publishNotifier = notifier
}

func (s *ProductServiceImp) SetTracker(tracker analytics.Tracker) {
	s.tracker = tracker
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

	created, err := s.repo.Create(ctx, product)
	if err != nil {
		return nil, err
	}

	s.tracker.Track(ctx, analytics.Event{
		Name:       analytics.EventListingCreated,
		DistinctID: created.UserID.String(),
		Properties: created.AnalyticsProperties(),
	})
	s.onListingStatusChanged(ctx, "", created, false)
	return created, nil
}

// onListingStatusChanged runs the side effects of a listing becoming visible to
// buyers. Every code path that changes a listing's status must call it.
func (s *ProductServiceImp) onListingStatusChanged(ctx context.Context, previousStatus models.ProductStatus, product *models.Product, byAdmin bool) {
	if product.Status != models.StatusPublished || previousStatus == models.StatusPublished {
		return
	}
	s.trackListingPublished(ctx, product, byAdmin)
	s.notifyListingPublished(ctx, product.ID.String())
}

func (s *ProductServiceImp) trackListingPublished(ctx context.Context, product *models.Product, byAdmin bool) {
	properties := product.AnalyticsProperties()
	properties["by_admin"] = byAdmin
	s.tracker.Track(ctx, analytics.Event{
		Name:       analytics.EventListingPublished,
		DistinctID: product.UserID.String(),
		Properties: properties,
	})
}

// notifyListingPublished is a best-effort side effect of publishing. A
// notification failure is logged and swallowed — the status change has
// already been saved and must not be reported as failed.
func (s *ProductServiceImp) notifyListingPublished(ctx context.Context, productID string) {
	if s.publishNotifier == nil {
		return
	}
	if err := s.publishNotifier.NotifyListingPublished(ctx, productID); err != nil {
		s.logger.Log(ctx, config.ErrorLevel, "Failed to notify that a listing was published", map[string]any{"error": err.Error(), "productId": productID})
	}
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
	product, err := s.repo.FindByID(ctx, id)
	if err != nil || product == nil {
		return product, err
	}

	s.incrementViewCount(ctx, id)

	return product, nil
}

// incrementViewCount is a best-effort side effect of viewing a product.
// Failures are logged and swallowed — a successful fetch must never fail because the counter write did.
func (s *ProductServiceImp) incrementViewCount(ctx context.Context, id string) {
	if err := s.repo.IncrementViewCount(ctx, id); err != nil {
		s.logger.Log(ctx, config.WarnLevel, "Failed to increment product view count", map[string]any{"error": err.Error(), "productId": id})
	}
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

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return err
	}

	updated := *p
	updated.Status = status
	s.onListingStatusChanged(ctx, p.Status, &updated, isAdmin)
	return nil
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

func (s *ProductServiceImp) FindSimilar(ctx context.Context, id string, limit int) ([]*models.Product, error) {
	source, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, nil
	}
	return s.repo.FindSimilar(ctx, source, limit)
}
