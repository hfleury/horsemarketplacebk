package models_test

import (
	"testing"

	productmodels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/models"
	"github.com/stretchr/testify/assert"
)

func stringPtr(value string) *string  { return &value }
func intPtr(value int) *int           { return &value }
func floatPtr(value float64) *float64 { return &value }

func TestSearchCriteria_ToProductFilters(t *testing.T) {
	tests := []struct {
		name                   string
		criteria               models.SearchCriteria
		expectedCategoryID     string
		expectedQuery          string
		expectedHorseFilter    *productmodels.HorseFilter
		expectedLocationFilter *productmodels.LocationFilter
	}{
		{
			name: "empty criteria map to zero values",
		},
		{
			name:          "query only",
			criteria:      models.SearchCriteria{Query: stringPtr("arab")},
			expectedQuery: "arab",
		},
		{
			name:               "category only",
			criteria:           models.SearchCriteria{CategoryID: stringPtr("cat-1")},
			expectedCategoryID: "cat-1",
		},
		{
			name:                "breed",
			criteria:            models.SearchCriteria{Breed: stringPtr("Arabian")},
			expectedHorseFilter: &productmodels.HorseFilter{Breed: stringPtr("Arabian")},
		},
		{
			name:                "gender",
			criteria:            models.SearchCriteria{Gender: stringPtr("mare")},
			expectedHorseFilter: &productmodels.HorseFilter{Gender: stringPtr("mare")},
		},
		{
			name:                "discipline",
			criteria:            models.SearchCriteria{Discipline: stringPtr("dressage")},
			expectedHorseFilter: &productmodels.HorseFilter{Discipline: stringPtr("dressage")},
		},
		{
			name:                "min age",
			criteria:            models.SearchCriteria{MinAge: intPtr(3)},
			expectedHorseFilter: &productmodels.HorseFilter{MinAge: intPtr(3)},
		},
		{
			name:                "max age",
			criteria:            models.SearchCriteria{MaxAge: intPtr(12)},
			expectedHorseFilter: &productmodels.HorseFilter{MaxAge: intPtr(12)},
		},
		{
			name:                "min height",
			criteria:            models.SearchCriteria{MinHeight: intPtr(150)},
			expectedHorseFilter: &productmodels.HorseFilter{MinHeight: intPtr(150)},
		},
		{
			name:                "max height",
			criteria:            models.SearchCriteria{MaxHeight: intPtr(170)},
			expectedHorseFilter: &productmodels.HorseFilter{MaxHeight: intPtr(170)},
		},
		{
			name:                "price only still yields a horse filter carrying just the price",
			criteria:            models.SearchCriteria{MinPrice: floatPtr(1000), MaxPrice: floatPtr(50000)},
			expectedHorseFilter: &productmodels.HorseFilter{MinPrice: floatPtr(1000), MaxPrice: floatPtr(50000)},
		},
		{
			name:                   "full location",
			criteria:               models.SearchCriteria{Lat: floatPtr(59.33), Lng: floatPtr(18.07), RadiusKm: floatPtr(50)},
			expectedLocationFilter: &productmodels.LocationFilter{Lat: 59.33, Lng: 18.07, RadiusKm: 50},
		},
		{
			name:     "partial location is ignored",
			criteria: models.SearchCriteria{Lat: floatPtr(59.33)},
		},
		{
			name: "all fields",
			criteria: models.SearchCriteria{
				Query: stringPtr("arab"), CategoryID: stringPtr("cat-1"),
				Breed: stringPtr("Arabian"), Gender: stringPtr("mare"), Discipline: stringPtr("dressage"),
				MinAge: intPtr(3), MaxAge: intPtr(12), MinHeight: intPtr(150), MaxHeight: intPtr(170),
				MinPrice: floatPtr(1000), MaxPrice: floatPtr(50000),
				Lat: floatPtr(59.33), Lng: floatPtr(18.07), RadiusKm: floatPtr(50),
			},
			expectedCategoryID: "cat-1",
			expectedQuery:      "arab",
			expectedHorseFilter: &productmodels.HorseFilter{
				Breed: stringPtr("Arabian"), Gender: stringPtr("mare"), Discipline: stringPtr("dressage"),
				MinAge: intPtr(3), MaxAge: intPtr(12), MinHeight: intPtr(150), MaxHeight: intPtr(170),
				MinPrice: floatPtr(1000), MaxPrice: floatPtr(50000),
			},
			expectedLocationFilter: &productmodels.LocationFilter{Lat: 59.33, Lng: 18.07, RadiusKm: 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categoryID, query, horseFilter, locationFilter := tt.criteria.ToProductFilters()

			assert.Equal(t, tt.expectedCategoryID, categoryID)
			assert.Equal(t, tt.expectedQuery, query)
			assert.Equal(t, tt.expectedHorseFilter, horseFilter)
			assert.Equal(t, tt.expectedLocationFilter, locationFilter)
		})
	}
}
