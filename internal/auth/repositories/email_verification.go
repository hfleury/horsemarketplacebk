package repositories

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
)

//go:generate mockgen -source=internal/auth/repositories/email_verification.go -destination=internal/auth/repositories/mock_email_verification.go -package=repositories

type EmailVerificationRepository interface {
	Create(ctx context.Context, ev *models.EmailVerification) (*models.EmailVerification, error)
	SelectByToken(ctx context.Context, token string) (*models.EmailVerification, error)
	MarkVerified(ctx context.Context, token string) error
	// GetLatestByEmail returns the most recent email verification record for the given email
	GetLatestByEmail(ctx context.Context, email string) (*models.EmailVerification, error)
}
