package repositories

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/messaging/models"
)

type MessageRepository interface {
	Create(ctx context.Context, message *models.Message) (*models.Message, error)
	FindByConversationID(ctx context.Context, conversationID string, afterID int64, limit int) (items []*models.Message, hasMore bool, err error)
}

type MessageRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewMessageRepoPsql(psql db.Database, logger config.Logging) *MessageRepoPsql {
	return &MessageRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *MessageRepoPsql) Create(ctx context.Context, message *models.Message) (*models.Message, error) {
	query := `
		INSERT INTO catalog.messages (conversation_id, sender_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, conversation_id, sender_id, body, created_at
	`
	row := r.psql.QueryRow(ctx, query, message.ConversationID, message.SenderID, message.Body)

	var created models.Message
	err := row.Scan(
		&created.ID,
		&created.ConversationID,
		&created.SenderID,
		&created.Body,
		&created.CreatedAt,
	)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to create message", map[string]any{"error": err.Error()})
		return nil, err
	}

	return &created, nil
}

func (r *MessageRepoPsql) FindByConversationID(ctx context.Context, conversationID string, afterID int64, limit int) ([]*models.Message, bool, error) {
	query := `
		SELECT id, conversation_id, sender_id, body, created_at
		FROM catalog.messages
		WHERE conversation_id = $1 AND id > $2
		ORDER BY id ASC
		LIMIT $3
	`
	rows, err := r.psql.Query(ctx, query, conversationID, afterID, limit+1)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find messages by conversation id", map[string]any{"error": err.Error(), "conversation_id": conversationID})
		return nil, false, err
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		var message models.Message
		err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.SenderID,
			&message.Body,
			&message.CreatedAt,
		)
		if err != nil {
			continue
		}
		messages = append(messages, &message)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	return messages, hasMore, nil
}
