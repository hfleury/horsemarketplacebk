package system

import (
	"context"
	"database/sql"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
)

type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	IsProductApprovalRequired(ctx context.Context) (bool, error)
}

type SettingsRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewSettingsRepoPsql(psql db.Database, logger config.Logging) *SettingsRepoPsql {
	return &SettingsRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *SettingsRepoPsql) Get(ctx context.Context, key string) (string, error) {
	query := `SELECT value FROM authentic.system_settings WHERE key = $1`
	var value string
	err := r.psql.QueryRow(ctx, query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (r *SettingsRepoPsql) Set(ctx context.Context, key string, value string) error {
	query := `
		INSERT INTO authentic.system_settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
	`
	_, err := r.psql.Execute(ctx, query, key, value)
	return err
}

func (r *SettingsRepoPsql) IsProductApprovalRequired(ctx context.Context) (bool, error) {
	val, err := r.Get(ctx, "product_approval_required")
	if err != nil {
		return true, err // Default to safe "true" on error or fallback?
	}
	return val == "true", nil
}
