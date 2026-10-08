package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/favorites/models"
	"github.com/hfleury/horsemarketplacebk/internal/favorites/repositories"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	productRepositories "github.com/hfleury/horsemarketplacebk/internal/products/repositories"
)

var (
	ErrInvalidProductID         = errors.New("invalid product id")
	ErrProductNotFound          = errors.New("product not found")
	ErrCannotFavoriteOwnListing = errors.New("cannot favorite your own listing")
	ErrListingNotPublished      = errors.New("only published listings can be favorited")
)

type FavoriteService struct {
	repo        repositories.FavoriteRepository
	productRepo productRepositories.ProductRepository
	logger      config.Logging
}

func NewFavoriteService(repo repositories.FavoriteRepository, productRepo productRepositories.ProductRepository, logger config.Logging) *FavoriteService {
	return &FavoriteService{
		repo:        repo,
		productRepo: productRepo,
		logger:      logger,
	}
}

func (s *FavoriteService) Add(ctx context.Context, userID, productID string) (*models.FavoriteStatus, error) {
	productUUID, userUUID, err := parseIDs(productID, userID)
	if err != nil {
		return nil, err
	}

	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	if product.UserID == userUUID {
		return nil, ErrCannotFavoriteOwnListing
	}
	if product.Status != productModels.StatusPublished {
		return nil, ErrListingNotPublished
	}

	if err := s.repo.Add(ctx, userUUID, productUUID); err != nil {
		return nil, err
	}

	return &models.FavoriteStatus{ProductID: productUUID, Favorited: true}, nil
}

// Remove skips the product lookup so a user can still un-favorite a listing
// that has since been sold or deleted.
func (s *FavoriteService) Remove(ctx context.Context, userID, productID string) (*models.FavoriteStatus, error) {
	productUUID, userUUID, err := parseIDs(productID, userID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Remove(ctx, userUUID, productUUID); err != nil {
		return nil, err
	}

	return &models.FavoriteStatus{ProductID: productUUID, Favorited: false}, nil
}

func (s *FavoriteService) List(ctx context.Context, userID string, page, limit int) (*productModels.PaginatedProducts, error) {
	items, total, err := s.productRepo.FindFavoritedByUserID(ctx, userID, page, limit)
	if err != nil {
		return nil, err
	}

	return &productModels.PaginatedProducts{Items: items, Total: total, Page: page, Limit: limit}, nil
}

func (s *FavoriteService) ListIDs(ctx context.Context, userID string) ([]uuid.UUID, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListProductIDs(ctx, userUUID)
}

// parseIDs rejects a malformed product ID up front so it surfaces as a 400
// rather than a Postgres error.
func parseIDs(productID, userID string) (productUUID, userUUID uuid.UUID, err error) {
	productUUID, err = uuid.Parse(productID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidProductID
	}
	userUUID, err = uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return productUUID, userUUID, nil
}
