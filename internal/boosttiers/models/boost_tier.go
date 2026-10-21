package models

import (
	"time"

	"github.com/google/uuid"
)

// BoostTier is a boost plan sellers can buy. PriceSEK includes 25% moms.
type BoostTier struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	PriceSEK     float64   `json:"price_sek"`
	DurationDays int       `json:"duration_days"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateBoostTierRequest has no binding tags: the service validates every field,
// so a missing or zero value gets a specific message instead of "Invalid payload".
type CreateBoostTierRequest struct {
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	PriceSEK     float64 `json:"price_sek"`
	DurationDays int     `json:"duration_days"`
}

// UpdateBoostTierRequest replaces every editable field. Code is immutable, so
// it is not accepted here.
type UpdateBoostTierRequest struct {
	Name         string  `json:"name"`
	PriceSEK     float64 `json:"price_sek"`
	DurationDays int     `json:"duration_days"`
	IsActive     *bool   `json:"is_active"`
}
