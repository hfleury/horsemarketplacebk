package media

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type MediaRepository interface {
	Create(ctx context.Context, media *Media) (*Media, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Media, error)
	UpdateVariants(ctx context.Context, id uuid.UUID, variants any) error
}

type PostgresMediaRepository struct {
	DB *sql.DB
}

func NewPostgresMediaRepository(db *sql.DB) *PostgresMediaRepository {
	return &PostgresMediaRepository{DB: db}
}

func (r *PostgresMediaRepository) Create(ctx context.Context, m *Media) (*Media, error) {
	query := `
		INSERT INTO authentic.media (
			file_name, original_name, mime_type, size_bytes, url, bucket_name, region, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()

	err := r.DB.QueryRowContext(ctx, query,
		m.FileName, m.OriginalName, m.MimeType, m.SizeBytes, m.URL, m.BucketName, m.Region, m.CreatedAt, m.UpdatedAt,
	).Scan(&m.ID)

	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *PostgresMediaRepository) FindByID(ctx context.Context, id uuid.UUID) (*Media, error) {
	query := `
		SELECT id, file_name, original_name, mime_type, size_bytes, url, bucket_name, region, created_at, updated_at
		FROM authentic.media
		WHERE id = $1
	`
	m := &Media{}
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&m.ID, &m.FileName, &m.OriginalName, &m.MimeType, &m.SizeBytes, &m.URL, &m.BucketName, &m.Region, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}
	return m, nil
	return m, nil
}

func (r *PostgresMediaRepository) UpdateVariants(ctx context.Context, id uuid.UUID, variants any) error {
	query := `
		UPDATE authentic.media
		SET variants = $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.DB.ExecContext(ctx, query, variants, id)
	return err
}
