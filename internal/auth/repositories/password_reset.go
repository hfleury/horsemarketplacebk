//go:generate mockgen -source=password_reset.go -destination=../mocks/auth/repositories/mock_password_reset.go -package=mockrepositories
package repositories

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
)

type PasswordResetRepository interface {
	Create(ctx context.Context, pr *models.PasswordReset) (*models.PasswordReset, error)
	SelectByToken(ctx context.Context, token string) (*models.PasswordReset, error)
	MarkAsUsed(ctx context.Context, token string) error
	GetLatestByUserID(ctx context.Context, userID string) (*models.PasswordReset, error)
}
