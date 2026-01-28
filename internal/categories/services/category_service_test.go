package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/categories/models"
	"github.com/hfleury/horsemarketplacebk/internal/categories/services"
	mockcategories "github.com/hfleury/horsemarketplacebk/internal/mocks/categories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAllCategories_TreeStructure(t *testing.T) {
	mockRepo := new(mockcategories.MockCategoryRepository)
	logger := config.NewZerologService()
	service := services.NewCategoryService(mockRepo, logger)

	// Create IDs
	parentID := uuid.New()
	child1ID := uuid.New()
	child2ID := uuid.New()

	// Setup data
	parentName := "Parent"
	child1Name := "Child 1"
	child2Name := "Child 2"
	now := time.Now()

	parent := &models.Category{Id: &parentID, Name: &parentName, CreatedAt: &now, UpdatedAt: &now}
	child1 := &models.Category{Id: &child1ID, Name: &child1Name, ParentID: &parentID, CreatedAt: &now, UpdatedAt: &now}
	child2 := &models.Category{Id: &child2ID, Name: &child2Name, ParentID: &parentID, CreatedAt: &now, UpdatedAt: &now}

	flatList := []*models.Category{parent, child1, child2}

	mockRepo.On("FindAll", mock.Anything).Return(flatList, nil)

	// Execute
	result, err := service.GetAllCategories(context.Background())

	// Verify
	assert.NoError(t, err)
	assert.Len(t, result, 1) // Only 1 root
	assert.Equal(t, "Parent", *result[0].Name)
	assert.Len(t, result[0].SubCategories, 2) // 2 children

	// Check children names (order depends on implementation, but likely creation order from list)
	assert.Equal(t, "Child 1", *result[0].SubCategories[0].Name)
	assert.Equal(t, "Child 2", *result[0].SubCategories[1].Name)
}

func TestCreateCategory_Success(t *testing.T) {
	mockRepo := new(mockcategories.MockCategoryRepository)
	logger := config.NewZerologService()
	service := services.NewCategoryService(mockRepo, logger)

	req := models.CreateCategoryRequest{
		Name: "New Category",
	}

	mockRepo.On("FindByName", mock.Anything, "New Category").Return(nil, nil)
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *models.Category) bool {
		return *c.Name == "New Category"
	})).Return(&models.Category{Name: &req.Name}, nil)

	_, err := service.CreateCategory(context.Background(), req)
	assert.NoError(t, err)
}
