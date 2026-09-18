package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/reports/models"
)

type ReportRepository interface {
	Create(ctx context.Context, report *models.Report) (*models.Report, error)
	FindByID(ctx context.Context, id string) (*models.Report, error)
	FindAll(ctx context.Context, status string) ([]*models.Report, error)
	UpdateStatus(ctx context.Context, id string, status models.ReportStatus, reviewedBy uuid.UUID) error
}

type ReportRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewReportRepoPsql(psql db.Database, logger config.Logging) *ReportRepoPsql {
	return &ReportRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *ReportRepoPsql) Create(ctx context.Context, report *models.Report) (*models.Report, error) {
	query := `
		INSERT INTO catalog.reports (id, product_id, reporter_user_id, reason, description, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, product_id, reporter_user_id, reason, description, status, reviewed_by, reviewed_at, created_at, updated_at
	`
	id := uuid.New()

	row := r.psql.QueryRow(ctx, query, id, report.ProductID, report.ReporterUserID, report.Reason, report.Description, report.Status)

	var created models.Report
	err := row.Scan(
		&created.ID,
		&created.ProductID,
		&created.ReporterUserID,
		&created.Reason,
		&created.Description,
		&created.Status,
		&created.ReviewedBy,
		&created.ReviewedAt,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to create report", map[string]any{"error": err.Error()})
		return nil, err
	}

	return &created, nil
}

func (r *ReportRepoPsql) FindByID(ctx context.Context, id string) (*models.Report, error) {
	query := `
		SELECT id, product_id, reporter_user_id, reason, description, status, reviewed_by, reviewed_at, created_at, updated_at
		FROM catalog.reports
		WHERE id = $1
	`
	row := r.psql.QueryRow(ctx, query, id)

	var report models.Report
	err := row.Scan(
		&report.ID,
		&report.ProductID,
		&report.ReporterUserID,
		&report.Reason,
		&report.Description,
		&report.Status,
		&report.ReviewedBy,
		&report.ReviewedAt,
		&report.CreatedAt,
		&report.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find report", map[string]any{"error": err.Error(), "id": id})
		return nil, err
	}

	return &report, nil
}

func (r *ReportRepoPsql) FindAll(ctx context.Context, status string) ([]*models.Report, error) {
	query := `
		SELECT id, product_id, reporter_user_id, reason, description, status, reviewed_by, reviewed_at, created_at, updated_at
		FROM catalog.reports
	`
	args := []any{}
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.psql.Query(ctx, query, args...)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find all reports", map[string]any{"error": err.Error()})
		return nil, err
	}
	defer rows.Close()

	var reports []*models.Report
	for rows.Next() {
		var report models.Report
		err := rows.Scan(
			&report.ID,
			&report.ProductID,
			&report.ReporterUserID,
			&report.Reason,
			&report.Description,
			&report.Status,
			&report.ReviewedBy,
			&report.ReviewedAt,
			&report.CreatedAt,
			&report.UpdatedAt,
		)
		if err != nil {
			continue
		}
		reports = append(reports, &report)
	}

	return reports, nil
}

func (r *ReportRepoPsql) UpdateStatus(ctx context.Context, id string, status models.ReportStatus, reviewedBy uuid.UUID) error {
	query := `UPDATE catalog.reports SET status = $1, reviewed_by = $2, reviewed_at = NOW(), updated_at = NOW() WHERE id = $3`
	_, err := r.psql.Execute(ctx, query, status, reviewedBy, id)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to update report status", map[string]any{"error": err.Error(), "id": id})
		return err
	}
	return nil
}
