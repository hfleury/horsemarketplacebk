package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
)

type FavoriteRepository interface {
	Add(ctx context.Context, userID, productID uuid.UUID) error
	Remove(ctx context.Context, userID, productID uuid.UUID) error
	ListProductIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
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

// ListProductIDs returns the user's favorited product IDs, newest-favorited
// first, hiding soft-deleted listings like FindFavoritedByUserID does.
func (r *FavoriteRepoPsql) ListProductIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT f.product_id
		FROM catalog.favorites f
		JOIN catalog.products p ON p.id = f.product_id
		WHERE f.user_id = $1 AND p.status <> $2
		ORDER BY f.created_at DESC
	`
	rows, err := r.psql.Query(ctx, query, userID, productModels.StatusDeleted)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to list favorite product IDs", map[string]any{"error": err.Error()})
		return nil, err
	}
	defer rows.Close()

	productIDs := []uuid.UUID{}
	for rows.Next() {
		var productID uuid.UUID
		if err := rows.Scan(&productID); err != nil {
			return nil, err
		}
		productIDs = append(productIDs, productID)
	}
	return productIDs, rows.Err()
}
