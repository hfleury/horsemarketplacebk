package repositories

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/db"
)

type PasswordResetRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewPasswordResetRepoPsql(psql db.Database, logger config.Logging) *PasswordResetRepoPsql {
	return &PasswordResetRepoPsql{psql: psql, logger: logger}
}

func (r *PasswordResetRepoPsql) Create(ctx context.Context, pr *models.PasswordReset) (*models.PasswordReset, error) {
	query := `
		INSERT INTO authentic.password_resets (user_id, reset_token, expires_at, is_used)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, reset_token, requested_at, expires_at, is_used, created_at, updated_at
	`
	err := r.psql.QueryRow(ctx, query, pr.UserId, pr.ResetToken, pr.ExpiresAt, pr.IsUsed).Scan(
		&pr.Id,
		&pr.UserId,
		&pr.ResetToken,
		&pr.RequestedAt,
		&pr.ExpiresAt,
		&pr.IsUsed,
		&pr.CreatedAt,
		&pr.UpdatedAt,
	)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "failed to create password reset record", map[string]any{"error": err.Error()})
		return nil, err
	}
	return pr, nil
}

func (r *PasswordResetRepoPsql) SelectByToken(ctx context.Context, token string) (*models.PasswordReset, error) {
	query := `
		SELECT id, user_id, reset_token, requested_at, expires_at, is_used, created_at, updated_at
		FROM authentic.password_resets
		WHERE reset_token = $1 LIMIT 1
	`
	pr := &models.PasswordReset{}
	err := r.psql.QueryRow(ctx, query, token).Scan(
		&pr.Id,
		&pr.UserId,
		&pr.ResetToken,
		&pr.RequestedAt,
		&pr.ExpiresAt,
		&pr.IsUsed,
		&pr.CreatedAt,
		&pr.UpdatedAt,
	)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "failed to select password reset by token", map[string]any{"error": err.Error()})
		return nil, err
	}
	return pr, nil
}

func (r *PasswordResetRepoPsql) MarkAsUsed(ctx context.Context, token string) error {
	query := `
		UPDATE authentic.password_resets
		SET is_used = true, updated_at = NOW()
		WHERE reset_token = $1
	`
	_, err := r.psql.Execute(ctx, query, token)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "failed to mark password reset as used", map[string]any{"error": err.Error()})
	}
	return err
}

func (r *PasswordResetRepoPsql) GetLatestByUserID(ctx context.Context, userID string) (*models.PasswordReset, error) {
	query := `
		SELECT id, user_id, reset_token, requested_at, expires_at, is_used, created_at, updated_at
		FROM authentic.password_resets
		WHERE user_id = $1
		ORDER BY requested_at DESC LIMIT 1
	`
	pr := &models.PasswordReset{}
	err := r.psql.QueryRow(ctx, query, userID).Scan(
		&pr.Id,
		&pr.UserId,
		&pr.ResetToken,
		&pr.RequestedAt,
		&pr.ExpiresAt,
		&pr.IsUsed,
		&pr.CreatedAt,
		&pr.UpdatedAt,
	)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "failed to select latest password reset by user ID", map[string]any{"error": err.Error()})
		return nil, err
	}
	return pr, nil
}
