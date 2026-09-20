package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/messaging/models"
)

type ConversationRepository interface {
	Create(ctx context.Context, conversation *models.Conversation) (*models.Conversation, error)
	FindByID(ctx context.Context, id string) (*models.Conversation, error)
	FindByProductAndBuyer(ctx context.Context, productID, buyerID string) (*models.Conversation, error)
	MarkReadByBuyer(ctx context.Context, id string) error
	MarkReadBySeller(ctx context.Context, id string) error
}

type ConversationRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewConversationRepoPsql(psql db.Database, logger config.Logging) *ConversationRepoPsql {
	return &ConversationRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *ConversationRepoPsql) Create(ctx context.Context, conversation *models.Conversation) (*models.Conversation, error) {
	query := `
		INSERT INTO catalog.conversations (id, product_id, buyer_id, seller_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, product_id, buyer_id, seller_id, buyer_last_read_at, seller_last_read_at, created_at, updated_at
	`
	id := uuid.New()

	row := r.psql.QueryRow(ctx, query, id, conversation.ProductID, conversation.BuyerID, conversation.SellerID)

	var created models.Conversation
	err := row.Scan(
		&created.ID,
		&created.ProductID,
		&created.BuyerID,
		&created.SellerID,
		&created.BuyerLastReadAt,
		&created.SellerLastReadAt,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to create conversation", map[string]any{"error": err.Error()})
		return nil, err
	}

	return &created, nil
}

func (r *ConversationRepoPsql) FindByID(ctx context.Context, id string) (*models.Conversation, error) {
	query := `
		SELECT id, product_id, buyer_id, seller_id, buyer_last_read_at, seller_last_read_at, created_at, updated_at
		FROM catalog.conversations
		WHERE id = $1
	`
	row := r.psql.QueryRow(ctx, query, id)

	var conversation models.Conversation
	err := row.Scan(
		&conversation.ID,
		&conversation.ProductID,
		&conversation.BuyerID,
		&conversation.SellerID,
		&conversation.BuyerLastReadAt,
		&conversation.SellerLastReadAt,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find conversation", map[string]any{"error": err.Error(), "id": id})
		return nil, err
	}

	return &conversation, nil
}

func (r *ConversationRepoPsql) FindByProductAndBuyer(ctx context.Context, productID, buyerID string) (*models.Conversation, error) {
	query := `
		SELECT id, product_id, buyer_id, seller_id, buyer_last_read_at, seller_last_read_at, created_at, updated_at
		FROM catalog.conversations
		WHERE product_id = $1 AND buyer_id = $2
	`
	row := r.psql.QueryRow(ctx, query, productID, buyerID)

	var conversation models.Conversation
	err := row.Scan(
		&conversation.ID,
		&conversation.ProductID,
		&conversation.BuyerID,
		&conversation.SellerID,
		&conversation.BuyerLastReadAt,
		&conversation.SellerLastReadAt,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find conversation by product and buyer", map[string]any{"error": err.Error(), "product_id": productID, "buyer_id": buyerID})
		return nil, err
	}

	return &conversation, nil
}

func (r *ConversationRepoPsql) MarkReadByBuyer(ctx context.Context, id string) error {
	query := `UPDATE catalog.conversations SET buyer_last_read_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.psql.Execute(ctx, query, id)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to mark conversation read by buyer", map[string]any{"error": err.Error(), "id": id})
		return err
	}
	return nil
}

func (r *ConversationRepoPsql) MarkReadBySeller(ctx context.Context, id string) error {
	query := `UPDATE catalog.conversations SET seller_last_read_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.psql.Execute(ctx, query, id)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to mark conversation read by seller", map[string]any{"error": err.Error(), "id": id})
		return err
	}
	return nil
}
