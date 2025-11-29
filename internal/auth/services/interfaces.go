package services

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
)

// UserServiceInterface defines the methods of the user service used by handlers and tests.
type UserServiceInterface interface {
	CreateUser(ctx context.Context, userRequest models.UserCreateResquest) (*models.User, error)
	SelectUserByUsername(ctx context.Context, user *models.UserGetRequest) (*models.User, error)
	Login(ctx context.Context, userLogin models.UserLogin) (*models.LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	Refresh(ctx context.Context, refreshToken string) (string, string, string, error)
	VerifyEmail(ctx context.Context, token string) error
	ResendVerification(ctx context.Context, email string) error
}
