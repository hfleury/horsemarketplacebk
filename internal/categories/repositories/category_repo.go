package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/categories/models"
	"github.com/hfleury/horsemarketplacebk/internal/db"
)

type CategoryRepository interface {
	Create(ctx context.Context, category *models.Category) (*models.Category, error)
	Update(ctx context.Context, category *models.Category) (*models.Category, error)
	Delete(ctx context.Context, id string) error
	FindByID(ctx context.Context, id string) (*models.Category, error)
	FindAll(ctx context.Context) ([]*models.Category, error)
	FindByName(ctx context.Context, name string) (*models.Category, error)
}

type CategoryRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewCategoryRepoPsql(psql db.Database, logger config.Logging) *CategoryRepoPsql {
	return &CategoryRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *CategoryRepoPsql) Create(ctx context.Context, category *models.Category) (*models.Category, error) {
	query := `
		INSERT INTO authentic.categories (id, name, picture_url, parent_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, picture_url, parent_id, created_at, updated_at
	`
	id := uuid.New()
	if category.Id != nil {
		id = *category.Id
	}

	row := r.psql.QueryRow(ctx, query, id, category.Name, category.PictureURL, category.ParentID)

	var created models.Category
	err := row.Scan(
		&created.Id,
		&created.Name,
		&created.PictureURL,
		&created.ParentID,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to create category", map[string]any{"error": err.Error()})
		return nil, err
	}

	return &created, nil
}

func (r *CategoryRepoPsql) Update(ctx context.Context, category *models.Category) (*models.Category, error) {
	query := `
		UPDATE authentic.categories
		SET name = $2, picture_url = $3, parent_id = $4, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, picture_url, parent_id, created_at, updated_at
	`

	row := r.psql.QueryRow(ctx, query, category.Id, category.Name, category.PictureURL, category.ParentID)

	var updated models.Category
	err := row.Scan(
		&updated.Id,
		&updated.Name,
		&updated.PictureURL,
		&updated.ParentID,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to update category", map[string]any{"error": err.Error()})
		return nil, err
	}

	return &updated, nil
}

func (r *CategoryRepoPsql) Delete(ctx context.Context, id string) error {
	// First check if there are subcategories
	checkQuery := `SELECT 1 FROM authentic.categories WHERE parent_id = $1 LIMIT 1`
	var exists int
	err := r.psql.QueryRow(ctx, checkQuery, id).Scan(&exists)
	if err == nil {
		return errors.New("cannot delete category with subcategories")
	} else if err != sql.ErrNoRows {
		return err
	}

	query := `DELETE FROM authentic.categories WHERE id = $1`
	result, err := r.psql.Execute(ctx, query, id)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to delete category", map[string]any{"error": err.Error()})
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("category not found")
	}

	return nil
}

func (r *CategoryRepoPsql) FindByID(ctx context.Context, id string) (*models.Category, error) {
	query := `
		SELECT id, name, picture_url, parent_id, created_at, updated_at
		FROM authentic.categories
		WHERE id = $1
	`
	row := r.psql.QueryRow(ctx, query, id)

	var cat models.Category
	err := row.Scan(
		&cat.Id,
		&cat.Name,
		&cat.PictureURL,
		&cat.ParentID,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil if not found, let service handle 404
		}
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find category by id", map[string]any{"error": err.Error()})
		return nil, err
	}

	return &cat, nil
}

func (r *CategoryRepoPsql) FindAll(ctx context.Context) ([]*models.Category, error) {
	query := `
		SELECT id, name, picture_url, parent_id, created_at, updated_at
		FROM authentic.categories
		ORDER BY created_at ASC
	`
	rows, err := r.psql.Query(ctx, query)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find all categories", map[string]any{"error": err.Error()})
		return nil, err
	}
	defer rows.Close()

	var categories []*models.Category
	for rows.Next() {
		var cat models.Category
		err := rows.Scan(
			&cat.Id,
			&cat.Name,
			&cat.PictureURL,
			&cat.ParentID,
			&cat.CreatedAt,
			&cat.UpdatedAt,
		)
		if err != nil {
			continue
		}
		categories = append(categories, &cat)
	}

	return categories, nil
}

func (r *CategoryRepoPsql) FindByName(ctx context.Context, name string) (*models.Category, error) {
	query := `
		SELECT id, name, picture_url, parent_id, created_at, updated_at
		FROM authentic.categories
		WHERE name = $1
	`
	row := r.psql.QueryRow(ctx, query, name)

	var cat models.Category
	err := row.Scan(
		&cat.Id,
		&cat.Name,
		&cat.PictureURL,
		&cat.ParentID,
		&cat.CreatedAt,
		&cat.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &cat, nil
}
