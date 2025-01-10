package models

import (
	"time"

	"github.com/google/uuid"
)

type PasswordReset struct {
	Id          *uuid.UUID `json:"id"`
	UserId      *uuid.UUID `json:"user_id"`
	ResetToken  *string    `json:"reset_token"`
	RequestedAt *time.Time `json:"requested_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsUsed      *bool      `json:"is_used"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}
