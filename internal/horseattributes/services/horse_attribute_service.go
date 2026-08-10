package services

import (
	"context"
	"errors"
	"strings"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/horseattributes/models"
	"github.com/hfleury/horsemarketplacebk/internal/horseattributes/repositories"
)

var allowedHorseAttributeTypes = map[string]bool{
	"breed":      true,
	"discipline": true,
	"gender":     true,
}

type HorseAttributeService struct {
	repo   repositories.HorseAttributeRepository
	logger config.Logging
}

func NewHorseAttributeService(repo repositories.HorseAttributeRepository, logger config.Logging) *HorseAttributeService {
	return &HorseAttributeService{
		repo:   repo,
		logger: logger,
	}
}

func (s *HorseAttributeService) CreateOption(ctx context.Context, req models.CreateHorseAttributeOptionRequest) (*models.HorseAttributeOption, error) {
	attrType := strings.TrimSpace(req.Type)
	if !allowedHorseAttributeTypes[attrType] {
		return nil, errors.New("type must be one of: breed, discipline, gender")
	}

	value := strings.TrimSpace(req.Value)
	if value == "" {
		return nil, errors.New("value is required")
	}

	existing, err := s.repo.FindAll(ctx, attrType)
	if err != nil {
		return nil, err
	}
	for _, opt := range existing {
		if opt.Value != nil && strings.EqualFold(*opt.Value, value) {
			return nil, errors.New("value already exists for this type")
		}
	}

	option := &models.HorseAttributeOption{
		Type:  &attrType,
		Value: &value,
	}

	return s.repo.Create(ctx, option)
}

func (s *HorseAttributeService) DeleteOption(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *HorseAttributeService) ListOptions(ctx context.Context, attrType string) ([]*models.HorseAttributeOption, error) {
	return s.repo.FindAll(ctx, attrType)
}
