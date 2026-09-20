package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID             int64     `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"created_at"`
}

type SendMessageRequest struct {
	Body string `json:"body"`
}

type ListMessagesResponse struct {
	Messages []*Message `json:"messages"`
	HasMore  bool       `json:"has_more"`
}
