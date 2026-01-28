//go:generate mockgen -source=user.go -destination=internal/mocks/auth/repositories/mock_user_psql.go -package=mockrepositories
package repositories

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
)

type UserRepository interface {
	IsUsernameTaken(ctx context.Context, username string) (bool, error)
	IsEmailTaken(ctx context.Context, email string) (bool, error)
	Insert(ctx context.Context, user *models.User) (*models.User, error)
	SelectUserByUsername(ctx context.Context, user *models.User) (*models.User, error)
	SelectUserByEmail(ctx context.Context, user *models.User) (*models.User, error)
	SelectUserByID(ctx context.Context, id string) (*models.User, error)
	// SetVerified updates the user's verified status
	SetVerified(ctx context.Context, id string, verified bool) error
	// FindAll returns all users
	FindAll(ctx context.Context) ([]*models.User, error)
	// UpdateStatus updates the user's active status
	UpdateStatus(ctx context.Context, id string, isActive bool) error
}
