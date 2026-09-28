package models

import "github.com/google/uuid"

type FavoriteStatus struct {
	ProductID uuid.UUID `json:"product_id"`
	Favorited bool      `json:"favorited"`
}
