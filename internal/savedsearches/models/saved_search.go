package models

import (
	"time"

	"github.com/google/uuid"
)

// SearchCriteria uses the same keys as the GET /api/v1/products query
// params, so a saved search can be replayed against the listing search.
type SearchCriteria struct {
	Query      *string  `json:"q,omitempty"`
	CategoryID *string  `json:"category_id,omitempty"`
	Breed      *string  `json:"breed,omitempty"`
	Gender     *string  `json:"gender,omitempty"`
	Discipline *string  `json:"discipline,omitempty"`
	MinAge     *int     `json:"min_age,omitempty"`
	MaxAge     *int     `json:"max_age,omitempty"`
	MinHeight  *int     `json:"min_height,omitempty"`
	MaxHeight  *int     `json:"max_height,omitempty"`
	MinPrice   *float64 `json:"min_price,omitempty"`
	MaxPrice   *float64 `json:"max_price,omitempty"`
	Lat        *float64 `json:"lat,omitempty"`
	Lng        *float64 `json:"lng,omitempty"`
	RadiusKm   *float64 `json:"radius_km,omitempty"`
}

type SavedSearch struct {
	ID             uuid.UUID      `json:"id"`
	UserID         uuid.UUID      `json:"user_id"`
	Name           string         `json:"name"`
	Criteria       SearchCriteria `json:"criteria"`
	LocationLabel  *string        `json:"location_label"`
	AlertsEnabled  bool           `json:"alerts_enabled"`
	LastNotifiedAt *time.Time     `json:"last_notified_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// SavedSearchRequest is the body for both create and full-replace update.
type SavedSearchRequest struct {
	Name          string          `json:"name" binding:"required"`
	Criteria      *SearchCriteria `json:"criteria" binding:"required"`
	LocationLabel *string         `json:"location_label"`
	AlertsEnabled *bool           `json:"alerts_enabled"`
}
