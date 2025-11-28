package models

// LoginResponse represents the safe response returned after successful login
type LoginResponse struct {
	Token     string       `json:"token"`
	User      UserResponse `json:"user"`
	ExpiresAt string       `json:"expires_at,omitempty"` // Optional: ISO 8601 format
}

// UserResponse represents safe user data for API responses
type UserResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}
