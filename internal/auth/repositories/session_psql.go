package repositories

import (
	"context"
	"time"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
)

type SessionRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewSessionRepoPsql(psql db.Database, logger config.Logging) *SessionRepoPsql {
	return &SessionRepoPsql{psql: psql, logger: logger}
}

func (s *SessionRepoPsql) Create(ctx context.Context, userID string, sessionToken string, expiresAt string) error {
	// expiresAt expected as RFC3339
	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return err
	}
	_, err = s.psql.Execute(ctx, `INSERT INTO authentic.user_sessions (user_id, session_token, is_active, last_activity, created_at, expires_at) VALUES ($1,$2,$3,$4,$5,$6)`, userID, sessionToken, true, time.Now().UTC(), time.Now().UTC(), t)
	return err
}

func (s *SessionRepoPsql) Validate(ctx context.Context, sessionToken string) (string, bool, string, error) {
	var userID string
	var isActive bool
	var expiresAt time.Time
	row := s.psql.QueryRow(ctx, `SELECT user_id, is_active, expires_at FROM authentic.user_sessions WHERE session_token = $1`, sessionToken)
	err := row.Scan(&userID, &isActive, &expiresAt)
	if err != nil {
		return "", false, "", err
	}
	return userID, isActive, expiresAt.Format(time.RFC3339), nil
}

func (s *SessionRepoPsql) Revoke(ctx context.Context, sessionToken string) error {
	_, err := s.psql.Execute(ctx, `UPDATE authentic.user_sessions SET is_active = false WHERE session_token = $1`, sessionToken)
	return err
}

// Rotate creates new session and revokes the old token in a single transaction
func (s *SessionRepoPsql) Rotate(ctx context.Context, userID string, oldToken string, newToken string, newExpiry string) error {
	tx, err := s.psql.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	// ensure rollback on failure
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// parse expiry
	t, err := time.Parse(time.RFC3339, newExpiry)
	if err != nil {
		tx.Rollback()
		return err
	}

	// insert new session
	if _, err = tx.ExecContext(ctx, `INSERT INTO authentic.user_sessions (user_id, session_token, is_active, last_activity, created_at, expires_at) VALUES ($1,$2,$3,$4,$5,$6)`, userID, newToken, true, time.Now().UTC(), time.Now().UTC(), t); err != nil {
		tx.Rollback()
		return err
	}

	// revoke old session
	if _, err = tx.ExecContext(ctx, `UPDATE authentic.user_sessions SET is_active = false WHERE session_token = $1`, oldToken); err != nil {
		tx.Rollback()
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// RevokeAllForUser revokes all sessions for a given user
func (s *SessionRepoPsql) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := s.psql.Execute(ctx, `UPDATE authentic.user_sessions SET is_active = false WHERE user_id = $1`, userID)
	return err
}
