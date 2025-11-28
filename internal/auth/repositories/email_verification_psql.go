package repositories

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/db"
)

type EmailVerificationRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewEmailVerificationRepoPsql(psql db.Database, logger config.Logging) *EmailVerificationRepoPsql {
	return &EmailVerificationRepoPsql{psql: psql, logger: logger}
}

func (er *EmailVerificationRepoPsql) Create(ctx context.Context, ev *models.EmailVerification) (*models.EmailVerification, error) {
	// parse expiry expected as time.Time pointer
	err := er.psql.QueryRow(ctx, `INSERT INTO authentic.email_verifications (user_id, verification_token, email, requested_at, expires_at) VALUES ($1,$2,$3,$4,$5) RETURNING id, user_id, verification_token, email, is_verified, requested_at, expires_at, created_at, updated_at`, ev.UserId, ev.VerificationToken, ev.Email, ev.RequestedAt, ev.ExpiresAt).Scan(
		&ev.Id,
		&ev.UserId,
		&ev.VerificationToken,
		&ev.Email,
		&ev.IsVerified,
		&ev.RequestedAt,
		&ev.ExpiresAt,
		&ev.CreatedAt,
		&ev.UpdatedAt,
	)
	if err != nil {
		er.logger.Log(ctx, config.ErrorLevel, "failed to create email verification", map[string]any{"error": err.Error()})
		return nil, err
	}
	return ev, nil
}

func (er *EmailVerificationRepoPsql) SelectByToken(ctx context.Context, token string) (*models.EmailVerification, error) {
	query := `SELECT id, user_id, verification_token, email, is_verified, requested_at, expires_at, created_at, updated_at FROM authentic.email_verifications WHERE verification_token = $1 LIMIT 1`
	ev := &models.EmailVerification{}
	err := er.psql.QueryRow(ctx, query, token).Scan(&ev.Id, &ev.UserId, &ev.VerificationToken, &ev.Email, &ev.IsVerified, &ev.RequestedAt, &ev.ExpiresAt, &ev.CreatedAt, &ev.UpdatedAt)
	if err != nil {
		er.logger.Log(ctx, config.ErrorLevel, "failed to select email verification by token", map[string]any{"error": err.Error()})
		return nil, err
	}
	return ev, nil
}

func (er *EmailVerificationRepoPsql) MarkVerified(ctx context.Context, token string) error {
	// set is_verified = true and updated_at
	_, err := er.psql.Execute(ctx, `UPDATE authentic.email_verifications SET is_verified = NOW(), updated_at = NOW() WHERE verification_token = $1`, token)
	if err != nil {
		er.logger.Log(ctx, config.ErrorLevel, "failed to mark email verification as verified", map[string]any{"error": err.Error()})
	}
	return err
}
