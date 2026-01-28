package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	Id            *uuid.UUID  `json:"id"`
	Name          *string     `json:"name"`
	PictureURL    *string     `json:"picture_url,omitempty"`
	ParentID      *uuid.UUID  `json:"parent_id,omitempty"`
	SubCategories []*Category `json:"subcategories,omitempty"`
	CreatedAt     *time.Time  `json:"created_at"`
	UpdatedAt     *time.Time  `json:"updated_at"`
}

type CreateCategoryRequest struct {
	Name       string  `json:"name" binding:"required"`
	PictureURL *string `json:"picture_url"`
	ParentID   *string `json:"parent_id"`
}

type UpdateCategoryRequest struct {
	Name       *string `json:"name"`
	PictureURL *string `json:"picture_url"`
	ParentID   *string `json:"parent_id"`
}
