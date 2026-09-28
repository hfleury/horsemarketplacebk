package models

import (
	"time"

	"github.com/google/uuid"
)

type CallbackRequest struct {
	ID          uuid.UUID `json:"id"`
	ProductID   uuid.UUID `json:"product_id"`
	BuyerID     uuid.UUID `json:"buyer_id"`
	PhoneNumber string    `json:"phone_number"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateCallbackRequestRequest struct {
	ProductID   string `json:"product_id" binding:"required"`
	PhoneNumber string `json:"phone_number" binding:"required"`
}
