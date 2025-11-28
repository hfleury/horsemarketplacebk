//go:generate mockgen -source=internal/auth/repositories/user.go -destination=internal/auth/repositories/mock_user_psql.go -package=repositories
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
}
