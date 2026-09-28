package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/callbacks/models"
	"github.com/hfleury/horsemarketplacebk/internal/db"
)

type CallbackRequestRepository interface {
	Create(ctx context.Context, request *models.CallbackRequest) (*models.CallbackRequest, error)
}

type CallbackRequestRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewCallbackRequestRepoPsql(psql db.Database, logger config.Logging) *CallbackRequestRepoPsql {
	return &CallbackRequestRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *CallbackRequestRepoPsql) Create(ctx context.Context, request *models.CallbackRequest) (*models.CallbackRequest, error) {
	query := `
		INSERT INTO catalog.callback_requests (id, product_id, buyer_id, phone_number)
		VALUES ($1, $2, $3, $4)
		RETURNING id, product_id, buyer_id, phone_number, created_at
	`
	id := uuid.New()

	row := r.psql.QueryRow(ctx, query, id, request.ProductID, request.BuyerID, request.PhoneNumber)

	var created models.CallbackRequest
	err := row.Scan(
		&created.ID,
		&created.ProductID,
		&created.BuyerID,
		&created.PhoneNumber,
		&created.CreatedAt,
	)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to create callback request", map[string]any{"error": err.Error()})
		return nil, err
	}

	return &created, nil
}
