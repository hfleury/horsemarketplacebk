package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	authRepositories "github.com/hfleury/horsemarketplacebk/internal/auth/repositories"
	"github.com/hfleury/horsemarketplacebk/internal/email"
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
	userRepo         authRepositories.UserRepository
	logger           config.Logging
	frontendURL      string
	emailSender      email.Sender
}

func NewMessagingService(conversationRepo repositories.ConversationRepository, messageRepo repositories.MessageRepository, productRepo productRepositories.ProductRepository, userRepo authRepositories.UserRepository, logger config.Logging, frontendURL string) *MessagingService {
	return &MessagingService{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		productRepo:      productRepo,
		userRepo:         userRepo,
		logger:           logger,
		frontendURL:      frontendURL,
	}
}

// SetEmailSender allows wiring an email.Sender after construction without
// changing existing constructor call sites.
func (s *MessagingService) SetEmailSender(sender email.Sender) {
	s.emailSender = sender
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

func (s *MessagingService) SendMessage(ctx context.Context, conversationID string, req models.SendMessageRequest, senderUserID string) (*models.MessageResponse, error) {
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

	created, err := s.messageRepo.Create(ctx, message)
	if err != nil {
		return nil, err
	}

	s.notifyRecipient(ctx, conversation, senderUserID)

	return &models.MessageResponse{
		ID:             created.ID,
		ConversationID: created.ConversationID,
		SenderID:       created.SenderID,
		Body:           created.Body,
		CreatedAt:      created.CreatedAt,
		IsMine:         true,
	}, nil
}

func (s *MessagingService) notifyRecipient(ctx context.Context, conversation *models.Conversation, senderUserID string) {
	if s.emailSender == nil {
		return
	}

	recipientID := conversation.SellerID
	if senderUserID != conversation.BuyerID.String() {
		recipientID = conversation.BuyerID
	}

	recipient, err := s.userRepo.SelectUserByID(ctx, recipientID.String())
	if err != nil {
		s.logger.Log(ctx, config.ErrorLevel, "failed to look up message notification recipient", map[string]any{"error": err.Error()})
		return
	}
	if recipient == nil || recipient.Email == nil {
		return
	}

	body := fmt.Sprintf("You have a new message on HorseMarketplace.\n\nView and reply: %s/inbox", s.frontendURL)
	if err := s.emailSender.Send(ctx, *recipient.Email, "New message on HorseMarketplace", body); err != nil {
		s.logger.Log(ctx, config.ErrorLevel, "failed to send message notification email", map[string]any{"error": err.Error()})
	}
}

func (s *MessagingService) ListMessages(ctx context.Context, conversationID string, requesterUserID string, afterID int64, limit int) ([]*models.MessageResponse, bool, error) {
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

	responses := make([]*models.MessageResponse, len(messages))
	for i, msg := range messages {
		responses[i] = &models.MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			SenderID:       msg.SenderID,
			Body:           msg.Body,
			CreatedAt:      msg.CreatedAt,
			IsMine:         msg.SenderID.String() == requesterUserID,
		}
	}

	return responses, hasMore, nil
}

func (s *MessagingService) ListConversations(ctx context.Context, userID string, page, limit int, sellerOnly bool) (*models.PaginatedConversationSummaries, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	items, total, err := s.conversationRepo.FindByUserID(ctx, userID, page, limit, sellerOnly)
	if err != nil {
		return nil, err
	}

	return &models.PaginatedConversationSummaries{Items: items, Total: total, Page: page, Limit: limit}, nil
}

func (s *MessagingService) CountUnreadConversations(ctx context.Context, userID string) (int, error) {
	return s.conversationRepo.CountUnreadByUserID(ctx, userID)
}
