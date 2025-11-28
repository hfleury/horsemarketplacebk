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
