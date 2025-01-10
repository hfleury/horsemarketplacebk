package models

import (
	"time"

	"github.com/google/uuid"
)

type MultiFactorAuth struct {
	Id        *uuid.UUID `json:"id"`
	UserId    *uuid.UUID `json:"user_id"`
	MfaType   *string    `json:"mfa_type"`
	MfaSecret *string    `json:"mfa_secret"`
	IsEnabled *bool      `json:"is_enabled"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}
