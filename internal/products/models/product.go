package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/categories/models"
	media "github.com/hfleury/horsemarketplacebk/internal/media"
)

type ProductStatus string
type ProductType string

const (
	StatusDraft           ProductStatus = "draft"
	StatusPublished       ProductStatus = "published"
	StatusPendingApproval ProductStatus = "pending_approval"
	StatusSold            ProductStatus = "sold"
	StatusArchived        ProductStatus = "archived"
	StatusDeleted         ProductStatus = "deleted"

	TypeHorse     ProductType = "horse"
	TypeVehicle   ProductType = "vehicle"
	TypeEquipment ProductType = "equipment"
	TypeService   ProductType = "service"
	TypeProperty  ProductType = "property"
)

type Product struct {
	ID              uuid.UUID     `json:"id"`
	UserID          uuid.UUID     `json:"user_id"`
	CategoryID      *uuid.UUID    `json:"category_id"`
	Type            ProductType   `json:"type"`
	Status          ProductStatus `json:"status"`
	Title           string        `json:"title"`
	PriceSEK        *float64      `json:"price_sek"`
	Description     *string       `json:"description"`
	City            *string       `json:"city"`
	Area            *string       `json:"area"`
	TransactionType *string       `json:"transaction_type"`
	ViewsCount      int           `json:"views_count"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`

	// Relations
	Category *models.Category `json:"category,omitempty"`
	Media    []ProductMedia   `json:"media,omitempty"`

	// Specific Data (One of these should be populated based on Type)
	Horse     *Horse     `json:"horse,omitempty"`
	Vehicle   *Vehicle   `json:"vehicle,omitempty"`
	Equipment *Equipment `json:"equipment,omitempty"`
}

type ProductMedia struct {
	ProductID uuid.UUID    `json:"product_id"`
	MediaID   uuid.UUID    `json:"media_id"`
	Order     int          `json:"order"`
	IsPrimary bool         `json:"is_primary"`
	Media     *media.Media `json:"media,omitempty"`
}

// Helper to marshal specific data for JSON responses if needed,
// though embedding pointers above is usually sufficient for JSON APIs.
