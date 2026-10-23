package repositories

import (
	"fmt"
	"testing"

	"github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/stretchr/testify/assert"
)

const locationClause = "ST_DWithin(ST_SetSRID(ST_MakePoint(p.longitude, p.latitude), 4326)::geography, ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography, $%d)"

func stringPtr(value string) *string  { return &value }
func intPtr(value int) *int           { return &value }
func floatPtr(value float64) *float64 { return &value }

func TestBuildSearchConditions(t *testing.T) {
	stockholm := &models.LocationFilter{Lat: 59.33, Lng: 18.07, RadiusKm: 50}

	tests := []struct {
		name               string
		seedArgs           []any
		categoryID         string
		query              string
		filter             *models.HorseFilter
		locationFilter     *models.LocationFilter
		expectedConditions []string
		expectedArgs       []any
	}{
		{
			name: "no filters produce no conditions",
		},
		{
			name:               "category",
			categoryID:         "cat-1",
			expectedConditions: []string{"p.category_id = $1"},
			expectedArgs:       []any{"cat-1"},
		},
		{
			name:               "free-text query reuses one placeholder for title and description",
			query:              "arab",
			expectedConditions: []string{"(p.description ILIKE $1 OR p.title ILIKE $1)"},
			expectedArgs:       []any{"%arab%"},
		},
		{
			name:               "breed",
			filter:             &models.HorseFilter{Breed: stringPtr("Arabian")},
			expectedConditions: []string{"h.breed = $1"},
			expectedArgs:       []any{"Arabian"},
		},
		{
			name:               "gender",
			filter:             &models.HorseFilter{Gender: stringPtr("mare")},
			expectedConditions: []string{"h.gender = $1"},
			expectedArgs:       []any{"mare"},
		},
		{
			name:               "discipline maps to orientation",
			filter:             &models.HorseFilter{Discipline: stringPtr("dressage")},
			expectedConditions: []string{"h.orientation = $1"},
			expectedArgs:       []any{"dressage"},
		},
		{
			name:               "min age",
			filter:             &models.HorseFilter{MinAge: intPtr(3)},
			expectedConditions: []string{"h.age >= $1"},
			expectedArgs:       []any{3},
		},
		{
			name:               "max age",
			filter:             &models.HorseFilter{MaxAge: intPtr(12)},
			expectedConditions: []string{"h.age <= $1"},
			expectedArgs:       []any{12},
		},
		{
			name:               "min height",
			filter:             &models.HorseFilter{MinHeight: intPtr(150)},
			expectedConditions: []string{"h.height >= $1"},
			expectedArgs:       []any{150},
		},
		{
			name:               "max height",
			filter:             &models.HorseFilter{MaxHeight: intPtr(170)},
			expectedConditions: []string{"h.height <= $1"},
			expectedArgs:       []any{170},
		},
		{
			name:               "min price",
			filter:             &models.HorseFilter{MinPrice: floatPtr(1000)},
			expectedConditions: []string{"p.price_sek >= $1"},
			expectedArgs:       []any{1000.0},
		},
		{
			name:               "max price",
			filter:             &models.HorseFilter{MaxPrice: floatPtr(50000)},
			expectedConditions: []string{"p.price_sek <= $1"},
			expectedArgs:       []any{50000.0},
		},
		{
			name:               "location",
			locationFilter:     stockholm,
			expectedConditions: []string{sprintfLocation(1, 2, 3)},
			expectedArgs:       []any{18.07, 59.33, 50000.0},
		},
		{
			name:               "empty-string text filters are skipped",
			filter:             &models.HorseFilter{Breed: stringPtr(""), Gender: stringPtr(""), Discipline: stringPtr("")},
			expectedConditions: nil,
		},
		{
			name:       "all criteria combined are numbered sequentially",
			categoryID: "cat-1",
			query:      "arab",
			filter: &models.HorseFilter{
				Breed: stringPtr("Arabian"), Gender: stringPtr("mare"), Discipline: stringPtr("dressage"),
				MinAge: intPtr(3), MaxAge: intPtr(12), MinHeight: intPtr(150), MaxHeight: intPtr(170),
				MinPrice: floatPtr(1000), MaxPrice: floatPtr(50000),
			},
			locationFilter: stockholm,
			expectedConditions: []string{
				"p.category_id = $1",
				"(p.description ILIKE $2 OR p.title ILIKE $2)",
				"h.breed = $3",
				"h.gender = $4",
				"h.orientation = $5",
				"h.age >= $6",
				"h.age <= $7",
				"h.height >= $8",
				"h.height <= $9",
				"p.price_sek >= $10",
				"p.price_sek <= $11",
				sprintfLocation(12, 13, 14),
			},
			expectedArgs: []any{"cat-1", "%arab%", "Arabian", "mare", "dressage", 3, 12, 150, 170, 1000.0, 50000.0, 18.07, 59.33, 50000.0},
		},
		{
			name:               "pre-seeded args shift every placeholder",
			seedArgs:           []any{"product-1"},
			categoryID:         "cat-1",
			locationFilter:     stockholm,
			expectedConditions: []string{"p.category_id = $2", sprintfLocation(3, 4, 5)},
			expectedArgs:       []any{"product-1", "cat-1", 18.07, 59.33, 50000.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditions, args := buildSearchConditions(tt.seedArgs, tt.categoryID, tt.query, tt.filter, tt.locationFilter)

			assert.Equal(t, tt.expectedConditions, conditions)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestBoostedFirstOrderBy(t *testing.T) {
	expected := " ORDER BY (p.status = 'published' AND EXISTS (SELECT 1 FROM catalog.product_boosts b" +
		" JOIN catalog.payments pay ON pay.id = b.payment_id" +
		" WHERE b.product_id = p.id AND pay.status = 'paid' AND b.starts_at <= NOW() AND b.expires_at > NOW()))" +
		" DESC, p.created_at DESC, p.id DESC"

	assert.Equal(t, expected, boostedFirstOrderBy)
}

func sprintfLocation(lngPlaceholder, latPlaceholder, radiusPlaceholder int) string {
	return fmt.Sprintf(locationClause, lngPlaceholder, latPlaceholder, radiusPlaceholder)
}
