package repositories

import (
	"context"
	"database/sql"
	"fmt"

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
	FindByUserID(ctx context.Context, userID string, page, limit int, sellerOnly bool) ([]*models.ConversationSummary, int, error)
	CountUnreadByUserID(ctx context.Context, userID string) (int, error)
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

func (r *ConversationRepoPsql) FindByUserID(ctx context.Context, userID string, page, limit int, sellerOnly bool) ([]*models.ConversationSummary, int, error) {
	offset := (page - 1) * limit

	whereClause := "c.buyer_id = $1 OR c.seller_id = $1"
	if sellerOnly {
		whereClause = "c.seller_id = $1"
	}

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM catalog.conversations c WHERE %s`, whereClause)
	if err := r.psql.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to count conversations for user", map[string]any{"error": err.Error(), "user_id": userID})
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT
			c.id, c.product_id, p.title, thumb.url,
			CASE WHEN c.buyer_id = $1 THEN c.seller_id ELSE c.buyer_id END,
			counterparty.username,
			lm.body, lm.sender_id, lm.created_at,
			CASE WHEN c.buyer_id = $1 THEN (c.buyer_last_read_at IS NULL OR c.buyer_last_read_at < COALESCE(lm.created_at, c.created_at))
			     ELSE (c.seller_last_read_at IS NULL OR c.seller_last_read_at < COALESCE(lm.created_at, c.created_at)) END,
			c.created_at, c.updated_at
		FROM catalog.conversations c
		JOIN catalog.products p ON p.id = c.product_id
		JOIN auth.users counterparty ON counterparty.id = CASE WHEN c.buyer_id = $1 THEN c.seller_id ELSE c.buyer_id END
		LEFT JOIN LATERAL (
			SELECT body, sender_id, created_at
			FROM catalog.messages
			WHERE conversation_id = c.id
			ORDER BY id DESC
			LIMIT 1
		) lm ON true
		LEFT JOIN LATERAL (
			SELECT m.url
			FROM catalog.product_media pm
			JOIN media.media m ON m.id = pm.media_id
			WHERE pm.product_id = c.product_id AND pm.is_primary = true
			LIMIT 1
		) thumb ON true
		WHERE %s
		ORDER BY COALESCE(lm.created_at, c.created_at) DESC
		LIMIT $2 OFFSET $3
	`, whereClause)
	rows, err := r.psql.Query(ctx, query, userID, limit, offset)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find conversations for user", map[string]any{"error": err.Error(), "user_id": userID})
		return nil, 0, err
	}
	defer rows.Close()

	var summaries []*models.ConversationSummary
	for rows.Next() {
		var s models.ConversationSummary
		if err := rows.Scan(
			&s.ID, &s.ProductID, &s.ProductTitle, &s.ProductThumbnailURL,
			&s.CounterpartyID, &s.CounterpartyUsername,
			&s.LastMessageBody, &s.LastMessageSenderID, &s.LastMessageAt,
			&s.IsUnread,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			r.logger.Log(ctx, config.ErrorLevel, "Failed to scan conversation summary", map[string]any{"error": err.Error(), "user_id": userID})
			return nil, 0, err
		}
		summaries = append(summaries, &s)
	}

	return summaries, total, nil
}

func (r *ConversationRepoPsql) CountUnreadByUserID(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM catalog.conversations c
		LEFT JOIN LATERAL (
			SELECT created_at
			FROM catalog.messages
			WHERE conversation_id = c.id
			ORDER BY id DESC
			LIMIT 1
		) lm ON true
		WHERE (c.buyer_id = $1 OR c.seller_id = $1)
		AND (
			CASE WHEN c.buyer_id = $1 THEN (c.buyer_last_read_at IS NULL OR c.buyer_last_read_at < COALESCE(lm.created_at, c.created_at))
			     ELSE (c.seller_last_read_at IS NULL OR c.seller_last_read_at < COALESCE(lm.created_at, c.created_at)) END
		)
	`
	var count int
	if err := r.psql.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to count unread conversations for user", map[string]any{"error": err.Error(), "user_id": userID})
		return 0, err
	}
	return count, nil
}
