package models

import (
	"time"

	"github.com/google/uuid"
)

const defaultRole = "user"

// LoginResponse represents the safe response returned after successful login
type LoginResponse struct {
	Token            string       `json:"token"`
	User             UserResponse `json:"user"`
	ExpiresAt        string       `json:"expires_at,omitempty"`
	RefreshToken     string       `json:"refresh_token,omitempty"`
	RefreshExpiresAt string       `json:"refresh_expires_at,omitempty"`
}

// UserResponse represents safe user data for API responses
type UserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// AdminUserResponse is the user shape returned to the admin users list.
type AdminUserResponse struct {
	ID        *uuid.UUID `json:"id"`
	Username  *string    `json:"username"`
	Email     *string    `json:"email"`
	Role      *string    `json:"role"`
	IsActive  *bool      `json:"is_active"`
	CreatedAt *time.Time `json:"created_at"`
}

func NewUserResponse(user *User) *UserResponse {
	response := &UserResponse{
		Username: stringValue(user.Username),
		Email:    stringValue(user.Email),
		Role:     stringValue(user.Role),
	}
	if user.Id != nil {
		response.ID = user.Id.String()
	}
	if response.Role == "" {
		response.Role = defaultRole
	}
	return response
}

func NewAdminUserResponse(user *User) *AdminUserResponse {
	return &AdminUserResponse{
		ID:        user.Id,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
