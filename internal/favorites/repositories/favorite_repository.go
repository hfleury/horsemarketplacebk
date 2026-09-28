package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
)

type FavoriteRepository interface {
	Add(ctx context.Context, userID, productID uuid.UUID) error
	Remove(ctx context.Context, userID, productID uuid.UUID) error
}

type FavoriteRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewFavoriteRepoPsql(psql db.Database, logger config.Logging) *FavoriteRepoPsql {
	return &FavoriteRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

// Add is idempotent: favoriting an already-favorited listing is a no-op.
func (r *FavoriteRepoPsql) Add(ctx context.Context, userID, productID uuid.UUID) error {
	query := `
		INSERT INTO catalog.favorites (user_id, product_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, product_id) DO NOTHING
	`
	if _, err := r.psql.Execute(ctx, query, userID, productID); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to add favorite", map[string]any{"error": err.Error()})
		return err
	}
	return nil
}

// Remove is idempotent: removing a listing that isn't a favorite is a no-op.
func (r *FavoriteRepoPsql) Remove(ctx context.Context, userID, productID uuid.UUID) error {
	query := `DELETE FROM catalog.favorites WHERE user_id = $1 AND product_id = $2`
	if _, err := r.psql.Execute(ctx, query, userID, productID); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to remove favorite", map[string]any{"error": err.Error()})
		return err
	}
	return nil
}
