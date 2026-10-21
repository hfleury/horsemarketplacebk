package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProductAnalyticsProperties(t *testing.T) {
	productID := uuid.New()
	categoryID := uuid.New()

	tests := []struct {
		name       string
		categoryID *uuid.UUID
		want       map[string]any
	}{
		{
			name: "without category",
			want: map[string]any{"listing_id": productID.String(), "listing_type": "horse", "status": "published"},
		},
		{
			name:       "with category",
			categoryID: &categoryID,
			want: map[string]any{
				"listing_id": productID.String(), "listing_type": "horse", "status": "published",
				"category_id": categoryID.String(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := &Product{ID: productID, CategoryID: tt.categoryID, Type: TypeHorse, Status: StatusPublished}
			assert.Equal(t, tt.want, product.AnalyticsProperties())
		})
	}
}
