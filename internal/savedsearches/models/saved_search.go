package models

import (
	"time"

	"github.com/google/uuid"
	productmodels "github.com/hfleury/horsemarketplacebk/internal/products/models"
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

// ToProductFilters maps the criteria onto the arguments of the product
// search, so a saved search matches exactly what browse would return.
func (c SearchCriteria) ToProductFilters() (categoryID, query string, horseFilter *productmodels.HorseFilter, locationFilter *productmodels.LocationFilter) {
	return valueOrEmpty(c.CategoryID), valueOrEmpty(c.Query), c.horseFilter(), c.locationFilter()
}

func (c SearchCriteria) horseFilter() *productmodels.HorseFilter {
	filter := productmodels.HorseFilter{
		Breed: c.Breed, Gender: c.Gender, Discipline: c.Discipline,
		MinAge: c.MinAge, MaxAge: c.MaxAge,
		MinHeight: c.MinHeight, MaxHeight: c.MaxHeight,
		MinPrice: c.MinPrice, MaxPrice: c.MaxPrice,
	}
	if filter == (productmodels.HorseFilter{}) {
		return nil
	}
	return &filter
}

func (c SearchCriteria) locationFilter() *productmodels.LocationFilter {
	if c.Lat == nil || c.Lng == nil || c.RadiusKm == nil {
		return nil
	}
	return &productmodels.LocationFilter{Lat: *c.Lat, Lng: *c.Lng, RadiusKm: *c.RadiusKm}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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

// AlertSubscription is an alert-enabled saved search together with the
// contact details of its owner.
type AlertSubscription struct {
	Search   SavedSearch
	Email    string
	Username string
}
