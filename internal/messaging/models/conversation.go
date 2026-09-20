package models

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID               uuid.UUID  `json:"id"`
	ProductID        uuid.UUID  `json:"product_id"`
	BuyerID          uuid.UUID  `json:"buyer_id"`
	SellerID         uuid.UUID  `json:"seller_id"`
	BuyerLastReadAt  *time.Time `json:"buyer_last_read_at"`
	SellerLastReadAt *time.Time `json:"seller_last_read_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type CreateConversationRequest struct {
	ProductID string `json:"product_id" binding:"required"`
}
