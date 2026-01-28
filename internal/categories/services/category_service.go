package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/categories/models"
	"github.com/hfleury/horsemarketplacebk/internal/categories/repositories"
)

type CategoryService struct {
	repo   repositories.CategoryRepository
	logger config.Logging
}

func NewCategoryService(repo repositories.CategoryRepository, logger config.Logging) *CategoryService {
	return &CategoryService{
		repo:   repo,
		logger: logger,
	}
}

func (s *CategoryService) CreateCategory(ctx context.Context, req models.CreateCategoryRequest) (*models.Category, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("category name is required")
	}

	// Check if parent exists if parentID is provided
	var parentUUID *uuid.UUID
	if req.ParentID != nil && *req.ParentID != "" {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, errors.New("invalid parent id format")
		}
		parent, err := s.repo.FindByID(ctx, id.String())
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, errors.New("parent category not found")
		}
		parentUUID = &id
	}

	// Check if name is taken
	existing, err := s.repo.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("category name already exists")
	}

	cat := &models.Category{
		Name:       &name,
		PictureURL: req.PictureURL,
		ParentID:   parentUUID,
	}

	return s.repo.Create(ctx, cat)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, id string, req models.UpdateCategoryRequest) (*models.Category, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("category not found")
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New("category name cannot be empty")
		}
		// Check uniqueness if name changed
		if name != *existing.Name {
			check, err := s.repo.FindByName(ctx, name)
			if err != nil {
				return nil, err
			}
			if check != nil {
				return nil, errors.New("category name already exists")
			}
		}
		existing.Name = &name
	}

	if req.PictureURL != nil {
		existing.PictureURL = req.PictureURL
	}

	if req.ParentID != nil {
		if *req.ParentID == "" {
			existing.ParentID = nil
		} else {
			// prevent self-parenting
			if *req.ParentID == id {
				return nil, errors.New("category cannot be its own parent")
			}
			pid, err := uuid.Parse(*req.ParentID)
			if err != nil {
				return nil, errors.New("invalid parent id format")
			}
			// check if parent exists
			p, err := s.repo.FindByID(ctx, pid.String())
			if err != nil {
				return nil, err
			}
			if p == nil {
				return nil, errors.New("parent category not found")
			}
			existing.ParentID = &pid
		}
	}

	return s.repo.Update(ctx, existing)
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *CategoryService) GetAllCategories(ctx context.Context) ([]*models.Category, error) {
	all, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	return buildCategoryTree(all), nil
}

func (s *CategoryService) GetCategoryByName(ctx context.Context, name string) (*models.Category, error) {
	all, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[uuid.UUID]*models.Category)
	for _, c := range all {
		if c.Id != nil {
			nodeMap[*c.Id] = c
			c.SubCategories = []*models.Category{} // init
		}
	}

	var target *models.Category
	// First pass: link children and find target
	for _, c := range all {
		if c.Name != nil && strings.EqualFold(*c.Name, name) {
			target = c
		}
		if c.ParentID != nil {
			if parent, exists := nodeMap[*c.ParentID]; exists {
				parent.SubCategories = append(parent.SubCategories, c)
			}
		}
	}

	return target, nil
}

func buildCategoryTree(categories []*models.Category) []*models.Category {
	categoryMap := make(map[uuid.UUID]*models.Category)
	var roots []*models.Category

	for _, c := range categories {
		if c.Id != nil {
			c.SubCategories = []*models.Category{}
			categoryMap[*c.Id] = c
		}
	}

	for _, c := range categories {
		if c.ParentID != nil {
			if parent, exists := categoryMap[*c.ParentID]; exists {
				parent.SubCategories = append(parent.SubCategories, c)
			} else {
				roots = append(roots, c)
			}
		} else {
			roots = append(roots, c)
		}
	}

	return roots
}
