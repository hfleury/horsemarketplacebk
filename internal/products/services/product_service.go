package services

import (
	"context"
	"errors"
	"time"

	"github.com/hfleury/horsemarketplacebk/config"
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
	FindAll(ctx context.Context, filters map[string]any) ([]*models.Product, error)
	UpdateStatus(ctx context.Context, id string, status models.ProductStatus, userID string, isAdmin bool) error
	Delete(ctx context.Context, id string, userID string, isAdmin bool) error
	// Specific searches
	Search(ctx context.Context, query string, categoryID string, fieldMap map[string]string) ([]*models.Product, error)
}

type ProductServiceImp struct {
	repo         repositories.ProductRepository
	settingsRepo system.SettingsRepository
	logger       config.Logging
}

func NewProductService(repo repositories.ProductRepository, settingsRepo system.SettingsRepository, logger config.Logging) *ProductServiceImp {
	return &ProductServiceImp{
		repo:         repo,
		settingsRepo: settingsRepo,
		logger:       logger,
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

	return s.repo.Create(ctx, product)
}

func (s *ProductServiceImp) FindByID(ctx context.Context, id string) (*models.Product, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ProductServiceImp) FindAll(ctx context.Context, filters map[string]any) ([]*models.Product, error) {
	return s.repo.FindAll(ctx, filters)
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

func (s *ProductServiceImp) Search(ctx context.Context, query string, categoryID string, fieldMap map[string]string) ([]*models.Product, error) {
	if categoryID != "" {
		return s.repo.FindByCategory(ctx, categoryID)
	}
	if query != "" {
		// Detect if it's a field search query e.g. "Model=R6"
		// or just simple text search
		return s.repo.FindByTextInDescription(ctx, query)
	}
	// Iterate field map if provided
	for k, v := range fieldMap {
		return s.repo.FindByField(ctx, k, v)
	}

	return s.repo.FindAll(ctx, nil)
}
