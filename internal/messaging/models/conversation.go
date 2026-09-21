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

type ConversationSummary struct {
	ID                   uuid.UUID  `json:"id"`
	ProductID            uuid.UUID  `json:"product_id"`
	ProductTitle         string     `json:"product_title"`
	ProductThumbnailURL  *string    `json:"product_thumbnail_url"`
	CounterpartyID       uuid.UUID  `json:"counterparty_id"`
	CounterpartyUsername string     `json:"counterparty_username"`
	LastMessageBody      *string    `json:"last_message_body"`
	LastMessageSenderID  *uuid.UUID `json:"last_message_sender_id"`
	LastMessageAt        *time.Time `json:"last_message_at"`
	IsUnread             bool       `json:"is_unread"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type PaginatedConversationSummaries struct {
	Items []*ConversationSummary `json:"items"`
	Total int                    `json:"total"`
	Page  int                    `json:"page"`
	Limit int                    `json:"limit"`
}
