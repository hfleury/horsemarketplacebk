package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id           *uuid.UUID `json:"id"`
	Username     *string    `json:"username"`
	Email        *string    `json:"email"`
	PasswordHash *string    `json:"password_hash"`
	IsActive     *bool      `json:"is_active"`
	IsVerified   *bool      `json:"is_verified"`
	LastLogin    *time.Time `json:"last_login"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

type UserCreateResquest struct {
	Username     *string `json:"username"`
	Email        *string `json:"email"`
	PasswordHash *string `json:"password_hash"`
}

type UserGetRequest struct {
	Username *string `json:"username"`
	Email    *string `json:"email"`
}
