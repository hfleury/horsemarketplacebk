package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/horseattributes/models"
	"github.com/hfleury/horsemarketplacebk/internal/horseattributes/services"
	mockhorseattributes "github.com/hfleury/horsemarketplacebk/internal/mocks/horseattributes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateOption_InvalidType(t *testing.T) {
	mockRepo := new(mockhorseattributes.MockHorseAttributeRepository)
	logger := config.NewZerologService()
	service := services.NewHorseAttributeService(mockRepo, logger)

	req := models.CreateHorseAttributeOptionRequest{Type: "color", Value: "Bay"}

	_, err := service.CreateOption(context.Background(), req)

	assert.Error(t, err)
	mockRepo.AssertNotCalled(t, "FindAll")
	mockRepo.AssertNotCalled(t, "Create")
}

func TestCreateOption_EmptyValue(t *testing.T) {
	mockRepo := new(mockhorseattributes.MockHorseAttributeRepository)
	logger := config.NewZerologService()
	service := services.NewHorseAttributeService(mockRepo, logger)

	req := models.CreateHorseAttributeOptionRequest{Type: "breed", Value: "   "}

	_, err := service.CreateOption(context.Background(), req)

	assert.Error(t, err)
	mockRepo.AssertNotCalled(t, "FindAll")
	mockRepo.AssertNotCalled(t, "Create")
}

func TestCreateOption_Duplicate(t *testing.T) {
	mockRepo := new(mockhorseattributes.MockHorseAttributeRepository)
	logger := config.NewZerologService()
	service := services.NewHorseAttributeService(mockRepo, logger)

	existingValue := "Warmblood"
	existing := []*models.HorseAttributeOption{{Value: &existingValue}}
	mockRepo.On("FindAll", mock.Anything, "breed").Return(existing, nil)

	req := models.CreateHorseAttributeOptionRequest{Type: "breed", Value: "Warmblood"}

	_, err := service.CreateOption(context.Background(), req)

	assert.Error(t, err)
	mockRepo.AssertNotCalled(t, "Create")
}

func TestCreateOption_Success(t *testing.T) {
	mockRepo := new(mockhorseattributes.MockHorseAttributeRepository)
	logger := config.NewZerologService()
	service := services.NewHorseAttributeService(mockRepo, logger)

	mockRepo.On("FindAll", mock.Anything, "discipline").Return([]*models.HorseAttributeOption{}, nil)
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(o *models.HorseAttributeOption) bool {
		return *o.Type == "discipline" && *o.Value == "Dressage"
	})).Return(&models.HorseAttributeOption{}, nil)

	req := models.CreateHorseAttributeOptionRequest{Type: "discipline", Value: "Dressage"}

	_, err := service.CreateOption(context.Background(), req)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestDeleteOption_NotFoundPropagates(t *testing.T) {
	mockRepo := new(mockhorseattributes.MockHorseAttributeRepository)
	logger := config.NewZerologService()
	service := services.NewHorseAttributeService(mockRepo, logger)

	mockRepo.On("Delete", mock.Anything, "missing-id").Return(errors.New("horse attribute option not found"))

	err := service.DeleteOption(context.Background(), "missing-id")

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}
