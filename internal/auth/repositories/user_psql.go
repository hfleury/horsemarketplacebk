package repositories

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/db"
)

type UserRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewUserRepoPsql(psql db.Database, logger config.Logging) *UserRepoPsql {
	return &UserRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (ur *UserRepoPsql) Insert(user *models.User) (*models.User, error) {
	return nil, nil
}

func (ur *UserRepoPsql) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	query := `
		SELECT 1
		FROM authentic.users us
		WHERE us.username = $1
		LIMIT 1
	`
	row, err := ur.psql.Query(ctx, query, username)
	if err != nil {
		ur.logger.Log(ctx, config.ErrorLevel, "Query to check username failed", map[string]any{
			"Error": err.Error(),
		})
		return true, err
	}
	defer row.Close()

	return row.Next(), nil
}

func (ur *UserRepoPsql) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	query := `
		SELECT 1
		FROM authentic.users us
		WHERE us.email = $1
		LIMIT 1
	`

	row, err := ur.psql.Query(ctx, query, email)
	if err != nil {
		ur.logger.Log(ctx, config.ErrorLevel, "Query to check email failed", map[string]any{
			"Error": err.Error(),
		})
	}
	defer row.Close()

	return row.Next(), nil
}
