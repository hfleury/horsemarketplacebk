package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/boosttiers/models"
	"github.com/hfleury/horsemarketplacebk/internal/db"
)

type BoostTierRepository interface {
	ListAll(ctx context.Context) ([]models.BoostTier, error)
	ListActive(ctx context.Context) ([]models.BoostTier, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.BoostTier, error)
	// FindActiveByCode returns nil when no active tier has that code.
	FindActiveByCode(ctx context.Context, code string) (*models.BoostTier, error)
	// Create returns false when the code is already taken.
	Create(ctx context.Context, tier *models.BoostTier) (bool, error)
	// Update returns false when no tier has the given ID. It never changes the code.
	Update(ctx context.Context, tier *models.BoostTier) (bool, error)
}

const (
	boostTierColumns = `id, code, name, price_sek, duration_days, is_active, created_at, updated_at`
	boostTierOrder   = ` ORDER BY price_sek ASC, code ASC`
)

type BoostTierRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewBoostTierRepoPsql(psql db.Database, logger config.Logging) *BoostTierRepoPsql {
	return &BoostTierRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *BoostTierRepoPsql) ListAll(ctx context.Context) ([]models.BoostTier, error) {
	return r.list(ctx, `SELECT `+boostTierColumns+` FROM catalog.boost_tiers`+boostTierOrder)
}

func (r *BoostTierRepoPsql) ListActive(ctx context.Context) ([]models.BoostTier, error) {
	return r.list(ctx, `SELECT `+boostTierColumns+` FROM catalog.boost_tiers WHERE is_active`+boostTierOrder)
}

func (r *BoostTierRepoPsql) FindByID(ctx context.Context, id uuid.UUID) (*models.BoostTier, error) {
	query := `SELECT ` + boostTierColumns + ` FROM catalog.boost_tiers WHERE id = $1`
	return r.findOne(ctx, "Failed to find boost tier", query, id)
}

func (r *BoostTierRepoPsql) FindActiveByCode(ctx context.Context, code string) (*models.BoostTier, error) {
	query := `SELECT ` + boostTierColumns + ` FROM catalog.boost_tiers WHERE code = $1 AND is_active`
	return r.findOne(ctx, "Failed to find active boost tier", query, code)
}

func (r *BoostTierRepoPsql) Create(ctx context.Context, tier *models.BoostTier) (bool, error) {
	query := `
		INSERT INTO catalog.boost_tiers (id, code, name, price_sek, duration_days, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (code) DO NOTHING
		RETURNING created_at, updated_at
	`
	err := r.psql.QueryRow(ctx, query,
		tier.ID, tier.Code, tier.Name, tier.PriceSEK, tier.DurationDays, tier.IsActive,
	).Scan(&tier.CreatedAt, &tier.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to create boost tier", map[string]any{"error": err.Error()})
		return false, err
	}
	return true, nil
}

func (r *BoostTierRepoPsql) Update(ctx context.Context, tier *models.BoostTier) (bool, error) {
	query := `
		UPDATE catalog.boost_tiers
		SET name = $2, price_sek = $3, duration_days = $4, is_active = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING ` + boostTierColumns
	updated, err := r.findOne(ctx, "Failed to update boost tier", query,
		tier.ID, tier.Name, tier.PriceSEK, tier.DurationDays, tier.IsActive,
	)
	if err != nil || updated == nil {
		return false, err
	}
	*tier = *updated
	return true, nil
}

func (r *BoostTierRepoPsql) list(ctx context.Context, query string) ([]models.BoostTier, error) {
	rows, err := r.psql.Query(ctx, query)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to list boost tiers", map[string]any{"error": err.Error()})
		return nil, err
	}
	defer rows.Close()

	tiers := []models.BoostTier{}
	for rows.Next() {
		tier, err := scanBoostTier(rows)
		if err != nil {
			return nil, err
		}
		tiers = append(tiers, *tier)
	}
	return tiers, rows.Err()
}

func (r *BoostTierRepoPsql) findOne(ctx context.Context, failureMessage, query string, args ...any) (*models.BoostTier, error) {
	tier, err := scanBoostTier(r.psql.QueryRow(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, failureMessage, map[string]any{"error": err.Error()})
		return nil, err
	}
	return tier, nil
}

func scanBoostTier(row interface{ Scan(...any) error }) (*models.BoostTier, error) {
	var tier models.BoostTier
	err := row.Scan(&tier.ID, &tier.Code, &tier.Name, &tier.PriceSEK, &tier.DurationDays, &tier.IsActive, &tier.CreatedAt, &tier.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &tier, nil
}
