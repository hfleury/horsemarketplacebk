package models

import (
	"time"

	"github.com/google/uuid"
)

type UserSession struct {
	Id           *uuid.UUID `json:"id"`
	UserId       *uuid.UUID `json:"user_id"`
	SessionToken *string    `json:"session_token"`
	IsActive     *bool      `json:"is_active"`
	LastActivity *time.Time `json:"last_activity"`
	CreatedAt    *time.Time `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
}
