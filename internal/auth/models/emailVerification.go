package models

import (
	"time"

	"github.com/google/uuid"
)

type EmailVerification struct {
	Id                *uuid.UUID `json:"id"`
	UserId            *uuid.UUID `json:"user_id"`
	VerificationToken *string    `json:"verification_token"`
	Email             *string    `json:"email"`
	IsVerified        *bool      `json:"is_verified"`
	RequestedAt       *time.Time `json:"requested_at"`
	ExpiresAt         *time.Time `json:"expires_at"`
	CreatedAt         *time.Time `json:"created_at"`
	UpdatedAt         *time.Time `json:"updated_at"`
}
