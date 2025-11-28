//go:generate mockgen -source=internal/auth/repositories/session.go -destination=internal/auth/repositories/mock_session.go -package=repositories
package repositories

import "context"

// SessionRepository defines operations for user sessions (refresh tokens)
type SessionRepository interface {
	Create(ctx context.Context, userID string, sessionToken string, expiresAt string) error
	Validate(ctx context.Context, sessionToken string) (userID string, isActive bool, expiresAt string, err error)
	Revoke(ctx context.Context, sessionToken string) error
}
