package models

// LoginResponse represents the safe response returned after successful login
type LoginResponse struct {
	Token     string       `json:"token"`
	User      UserResponse `json:"user"`
	ExpiresAt       string `json:"expires_at,omitempty"`
	RefreshToken    string `json:"refresh_token,omitempty"`
	RefreshExpiresAt string `json:"refresh_expires_at,omitempty"`
}

// UserResponse represents safe user data for API responses
type UserResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}
