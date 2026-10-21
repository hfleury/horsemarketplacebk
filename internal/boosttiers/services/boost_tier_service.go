package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/models"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/repositories"
)

var (
	ErrInvalidBoostTier   = errors.New("invalid boost tier")
	ErrInvalidBoostTierID = errors.New("invalid boost tier id")
	ErrBoostTierNotFound  = errors.New("boost tier not found")
	ErrBoostTierCodeTaken = errors.New("boost tier code already exists")
)

type BoostTierService struct {
	repo   repositories.BoostTierRepository
	logger config.Logging
}

func NewBoostTierService(repo repositories.BoostTierRepository, logger config.Logging) *BoostTierService {
	return &BoostTierService{
		repo:   repo,
		logger: logger,
	}
}

func (s *BoostTierService) ListAll(ctx context.Context) ([]models.BoostTier, error) {
	return s.repo.ListAll(ctx)
}

func (s *BoostTierService) ListActive(ctx context.Context) ([]models.BoostTier, error) {
	return s.repo.ListActive(ctx)
}

func (s *BoostTierService) Create(ctx context.Context, req models.CreateBoostTierRequest) (*models.BoostTier, error) {
	if err := validateCode(req.Code); err != nil {
		return nil, err
	}
	name, err := validateTierFields(req.Name, req.PriceSEK, req.DurationDays)
	if err != nil {
		return nil, err
	}

	tier := &models.BoostTier{
		ID:           uuid.New(),
		Code:         req.Code,
		Name:         name,
		PriceSEK:     req.PriceSEK,
		DurationDays: req.DurationDays,
		IsActive:     true,
	}
	created, err := s.repo.Create(ctx, tier)
	if err != nil {
		return nil, err
	}
	if !created {
		return nil, ErrBoostTierCodeTaken
	}
	return tier, nil
}

func (s *BoostTierService) Update(ctx context.Context, id string, req models.UpdateBoostTierRequest) (*models.BoostTier, error) {
	tierID, err := uuid.Parse(id)
	if err != nil {
		return nil, ErrInvalidBoostTierID
	}
	name, err := validateTierFields(req.Name, req.PriceSEK, req.DurationDays)
	if err != nil {
		return nil, err
	}
	if req.IsActive == nil {
		return nil, fmt.Errorf("%w: is_active is required", ErrInvalidBoostTier)
	}

	tier := &models.BoostTier{
		ID:           tierID,
		Name:         name,
		PriceSEK:     req.PriceSEK,
		DurationDays: req.DurationDays,
		IsActive:     *req.IsActive,
	}
	updated, err := s.repo.Update(ctx, tier)
	if err != nil {
		return nil, err
	}
	if !updated {
		return nil, ErrBoostTierNotFound
	}
	return tier, nil
}
