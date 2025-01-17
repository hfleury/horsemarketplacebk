package repositories

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
)

type UserRepository interface {
	IsUsernameTaken(ctx context.Context, username string) (bool, error)
	IsEmailTaken(ctx context.Context, email string) (bool, error)
	Insert(ctx context.Context, user *models.User) (*models.User, error)
}
