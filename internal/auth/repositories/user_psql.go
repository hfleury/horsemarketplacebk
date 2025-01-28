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

// Insert insert a new user in the database
func (ur *UserRepoPsql) Insert(ctx context.Context, user *models.User) (*models.User, error) {
	query := `
		INSERT INTO authentic.users (username, email, password_hash)
		VALUES
		($1, $2, $3)
		RETURNING id, username, email, password_hash, is_active, is_verified, last_login, created_at, updated_at;
	`

	err := ur.psql.QueryRow(ctx, query, user.Username, user.Email, user.PasswordHash).Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
		&user.IsVerified,
		&user.LastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		ur.logger.Log(ctx, config.ErrorLevel, "Failed to insert user", map[string]any{
			"error": err.Error(),
			"query": query,
		})
		return nil, err
	}

	ur.logger.Log(ctx, config.InfoLevel, "User inserted successfully", map[string]any{
		"user_id": user.Id,
	})

	return user, nil
}

// IsUsernameTaken check if the username already exist in the database
func (ur *UserRepoPsql) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	query := `
		SELECT 1
		FROM authentic.users
		WHERE username = $1
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

// IsEmailTaken check if the email already exist in the database
func (ur *UserRepoPsql) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	query := `
		SELECT 1
		FROM authentic.users
		WHERE email = $1
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

// SelectUserByUsername get the user by the username
func (ur *UserRepoPsql) SelectUserByUsername(ctx context.Context, user *models.User) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, is_active, is_verified, last_login, created_at, updated_at
		FROM authentic.users
		WHERE username = $1;
	`
	err := ur.psql.QueryRow(ctx, query, user.Username).Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
		&user.IsVerified,
		&user.LastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		ur.logger.Log(ctx, config.ErrorLevel, "Failed to get user by username", map[string]any{
			"error": err.Error(),
			"query": query,
		})
		return nil, err
	}

	return user, nil
}

// SelectUserByEmail get the user by email
func (ur *UserRepoPsql) SelectUserByEmail(ctx context.Context, user *models.User) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, is_active, is_verified, last_login, created_at, updated_at
		FROM authentic.users
		WHERE email = $1;
	`
	err := ur.psql.QueryRow(ctx, query, user.Email).Scan(
		&user.Id,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.IsActive,
		&user.IsVerified,
		&user.LastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		ur.logger.Log(ctx, config.ErrorLevel, "Failed to get user by username", map[string]any{
			"error": err.Error(),
			"query": query,
		})
		return nil, err
	}

	return user, nil
}
