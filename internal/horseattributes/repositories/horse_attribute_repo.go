package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/horseattributes/models"
)

type HorseAttributeRepository interface {
	Create(ctx context.Context, option *models.HorseAttributeOption) (*models.HorseAttributeOption, error)
	Delete(ctx context.Context, id string) error
	FindAll(ctx context.Context, attrType string) ([]*models.HorseAttributeOption, error)
}

type HorseAttributeRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewHorseAttributeRepoPsql(psql db.Database, logger config.Logging) *HorseAttributeRepoPsql {
	return &HorseAttributeRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *HorseAttributeRepoPsql) Create(ctx context.Context, option *models.HorseAttributeOption) (*models.HorseAttributeOption, error) {
	query := `
		INSERT INTO catalog.horse_attribute_options (id, type, value)
		VALUES ($1, $2, $3)
		RETURNING id, type, value, created_at, updated_at
	`
	id := uuid.New()
	if option.ID != nil {
		id = *option.ID
	}

	row := r.psql.QueryRow(ctx, query, id, option.Type, option.Value)

	var created models.HorseAttributeOption
	err := row.Scan(
		&created.ID,
		&created.Type,
		&created.Value,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to create horse attribute option", map[string]any{"error": err.Error()})
		return nil, err
	}

	return &created, nil
}

func (r *HorseAttributeRepoPsql) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM catalog.horse_attribute_options WHERE id = $1`
	result, err := r.psql.Execute(ctx, query, id)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to delete horse attribute option", map[string]any{"error": err.Error()})
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("horse attribute option not found")
	}

	return nil
}

func (r *HorseAttributeRepoPsql) FindAll(ctx context.Context, attrType string) ([]*models.HorseAttributeOption, error) {
	query := `
		SELECT id, type, value, created_at, updated_at
		FROM catalog.horse_attribute_options
	`
	args := []any{}
	if attrType != "" {
		query += ` WHERE type = $1`
		args = append(args, attrType)
	}
	query += ` ORDER BY type ASC, value ASC`

	rows, err := r.psql.Query(ctx, query, args...)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find all horse attribute options", map[string]any{"error": err.Error()})
		return nil, err
	}
	defer rows.Close()

	var options []*models.HorseAttributeOption
	for rows.Next() {
		var opt models.HorseAttributeOption
		err := rows.Scan(
			&opt.ID,
			&opt.Type,
			&opt.Value,
			&opt.CreatedAt,
			&opt.UpdatedAt,
		)
		if err != nil {
			continue
		}
		options = append(options, &opt)
	}

	return options, nil
}
