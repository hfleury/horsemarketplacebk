package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/messaging/models"
	"github.com/hfleury/horsemarketplacebk/internal/messaging/repositories"
	productRepositories "github.com/hfleury/horsemarketplacebk/internal/products/repositories"
)

var (
	ErrProductNotFound            = errors.New("product not found")
	ErrCannotMessageOwnListing    = errors.New("cannot message your own listing")
	ErrConversationNotFound       = errors.New("conversation not found")
	ErrNotConversationParticipant = errors.New("not a participant of this conversation")
	ErrEmptyMessageBody           = errors.New("message body cannot be empty")
)

type MessagingService struct {
	conversationRepo repositories.ConversationRepository
	messageRepo      repositories.MessageRepository
	productRepo      productRepositories.ProductRepository
	logger           config.Logging
}

func NewMessagingService(conversationRepo repositories.ConversationRepository, messageRepo repositories.MessageRepository, productRepo productRepositories.ProductRepository, logger config.Logging) *MessagingService {
	return &MessagingService{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		productRepo:      productRepo,
		logger:           logger,
	}
}

func (s *MessagingService) CreateConversation(ctx context.Context, req models.CreateConversationRequest, buyerUserID string) (*models.Conversation, error) {
	product, err := s.productRepo.FindByID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	if product.UserID.String() == buyerUserID {
		return nil, ErrCannotMessageOwnListing
	}

	existing, err := s.conversationRepo.FindByProductAndBuyer(ctx, req.ProductID, buyerUserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	buyerID, err := uuid.Parse(buyerUserID)
	if err != nil {
		return nil, err
	}

	conversation := &models.Conversation{
		ID:        uuid.New(),
		ProductID: product.ID,
		BuyerID:   buyerID,
		SellerID:  product.UserID,
	}

	return s.conversationRepo.Create(ctx, conversation)
}

func (s *MessagingService) SendMessage(ctx context.Context, conversationID string, req models.SendMessageRequest, senderUserID string) (*models.Message, error) {
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, ErrEmptyMessageBody
	}

	conversation, err := s.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conversation == nil {
		return nil, ErrConversationNotFound
	}

	if senderUserID != conversation.BuyerID.String() && senderUserID != conversation.SellerID.String() {
		return nil, ErrNotConversationParticipant
	}

	senderID, err := uuid.Parse(senderUserID)
	if err != nil {
		return nil, err
	}

	message := &models.Message{
		ConversationID: conversation.ID,
		SenderID:       senderID,
		Body:           body,
	}

	return s.messageRepo.Create(ctx, message)
}

func (s *MessagingService) ListMessages(ctx context.Context, conversationID string, requesterUserID string, afterID int64, limit int) ([]*models.Message, bool, error) {
	conversation, err := s.conversationRepo.FindByID(ctx, conversationID)
	if err != nil {
		return nil, false, err
	}
	if conversation == nil {
		return nil, false, ErrConversationNotFound
	}

	if requesterUserID != conversation.BuyerID.String() && requesterUserID != conversation.SellerID.String() {
		return nil, false, ErrNotConversationParticipant
	}

	messages, hasMore, err := s.messageRepo.FindByConversationID(ctx, conversationID, afterID, limit)
	if err != nil {
		return nil, false, err
	}

	var markReadErr error
	if requesterUserID == conversation.BuyerID.String() {
		markReadErr = s.conversationRepo.MarkReadByBuyer(ctx, conversationID)
	} else {
		markReadErr = s.conversationRepo.MarkReadBySeller(ctx, conversationID)
	}
	if markReadErr != nil {
		s.logger.Log(ctx, config.ErrorLevel, "Failed to mark conversation as read", map[string]any{"error": markReadErr.Error(), "conversation_id": conversationID})
	}

	return messages, hasMore, nil
}
