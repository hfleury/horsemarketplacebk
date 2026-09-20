package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	mockmessaging "github.com/hfleury/horsemarketplacebk/internal/mocks/messaging"
	mockProducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	"github.com/hfleury/horsemarketplacebk/internal/messaging/models"
	"github.com/hfleury/horsemarketplacebk/internal/messaging/services"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newMessagingService() (*services.MessagingService, *mockmessaging.MockConversationRepository, *mockmessaging.MockMessageRepository, *mockProducts.MockProductRepo) {
	mockConversationRepo := new(mockmessaging.MockConversationRepository)
	mockMessageRepo := new(mockmessaging.MockMessageRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewMessagingService(mockConversationRepo, mockMessageRepo, mockProductRepo, logger)
	return service, mockConversationRepo, mockMessageRepo, mockProductRepo
}

func TestCreateConversation_Success(t *testing.T) {
	service, mockConversationRepo, _, mockProductRepo := newMessagingService()

	productID := uuid.New()
	sellerID := uuid.New()
	buyerID := uuid.New()

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(&productModels.Product{ID: productID, UserID: sellerID}, nil)
	mockConversationRepo.On("FindByProductAndBuyer", mock.Anything, productID.String(), buyerID.String()).Return(nil, nil)
	mockConversationRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *models.Conversation) bool {
		return c.ProductID == productID && c.BuyerID == buyerID && c.SellerID == sellerID
	})).Return(&models.Conversation{ID: uuid.New(), ProductID: productID, BuyerID: buyerID, SellerID: sellerID}, nil)

	req := models.CreateConversationRequest{ProductID: productID.String()}

	conversation, err := service.CreateConversation(context.Background(), req, buyerID.String())

	assert.NoError(t, err)
	assert.Equal(t, productID, conversation.ProductID)
	mockConversationRepo.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
}

func TestCreateConversation_FindOrCreateHit(t *testing.T) {
	service, mockConversationRepo, _, mockProductRepo := newMessagingService()

	productID := uuid.New()
	sellerID := uuid.New()
	buyerID := uuid.New()
	existing := &models.Conversation{ID: uuid.New(), ProductID: productID, BuyerID: buyerID, SellerID: sellerID}

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(&productModels.Product{ID: productID, UserID: sellerID}, nil)
	mockConversationRepo.On("FindByProductAndBuyer", mock.Anything, productID.String(), buyerID.String()).Return(existing, nil)

	req := models.CreateConversationRequest{ProductID: productID.String()}

	conversation, err := service.CreateConversation(context.Background(), req, buyerID.String())

	assert.NoError(t, err)
	assert.Equal(t, existing, conversation)
	mockConversationRepo.AssertNotCalled(t, "Create")
	mockConversationRepo.AssertExpectations(t)
}

func TestCreateConversation_ProductNotFound(t *testing.T) {
	service, mockConversationRepo, _, mockProductRepo := newMessagingService()

	productID := uuid.New()
	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(nil, nil)

	req := models.CreateConversationRequest{ProductID: productID.String()}

	_, err := service.CreateConversation(context.Background(), req, uuid.New().String())

	assert.ErrorIs(t, err, services.ErrProductNotFound)
	mockConversationRepo.AssertNotCalled(t, "FindByProductAndBuyer")
	mockConversationRepo.AssertNotCalled(t, "Create")
}

func TestCreateConversation_CannotMessageOwnListing(t *testing.T) {
	service, mockConversationRepo, _, mockProductRepo := newMessagingService()

	productID := uuid.New()
	ownerID := uuid.New()

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(&productModels.Product{ID: productID, UserID: ownerID}, nil)

	req := models.CreateConversationRequest{ProductID: productID.String()}

	_, err := service.CreateConversation(context.Background(), req, ownerID.String())

	assert.ErrorIs(t, err, services.ErrCannotMessageOwnListing)
	mockConversationRepo.AssertNotCalled(t, "FindByProductAndBuyer")
	mockConversationRepo.AssertNotCalled(t, "Create")
}

func TestSendMessage_SuccessFromBuyer(t *testing.T) {
	service, mockConversationRepo, mockMessageRepo, _ := newMessagingService()

	conversationID := uuid.New()
	buyerID := uuid.New()
	sellerID := uuid.New()
	conversation := &models.Conversation{ID: conversationID, BuyerID: buyerID, SellerID: sellerID}

	mockConversationRepo.On("FindByID", mock.Anything, conversationID.String()).Return(conversation, nil)
	mockMessageRepo.On("Create", mock.Anything, mock.MatchedBy(func(m *models.Message) bool {
		return m.ConversationID == conversationID && m.SenderID == buyerID && m.Body == "hello"
	})).Return(&models.Message{ID: 1, ConversationID: conversationID, SenderID: buyerID, Body: "hello"}, nil)

	req := models.SendMessageRequest{Body: "hello"}

	message, err := service.SendMessage(context.Background(), conversationID.String(), req, buyerID.String())

	assert.NoError(t, err)
	assert.Equal(t, "hello", message.Body)
	mockConversationRepo.AssertExpectations(t)
	mockMessageRepo.AssertExpectations(t)
}

func TestSendMessage_SuccessFromSeller(t *testing.T) {
	service, mockConversationRepo, mockMessageRepo, _ := newMessagingService()

	conversationID := uuid.New()
	buyerID := uuid.New()
	sellerID := uuid.New()
	conversation := &models.Conversation{ID: conversationID, BuyerID: buyerID, SellerID: sellerID}

	mockConversationRepo.On("FindByID", mock.Anything, conversationID.String()).Return(conversation, nil)
	mockMessageRepo.On("Create", mock.Anything, mock.MatchedBy(func(m *models.Message) bool {
		return m.ConversationID == conversationID && m.SenderID == sellerID && m.Body == "hi there"
	})).Return(&models.Message{ID: 2, ConversationID: conversationID, SenderID: sellerID, Body: "hi there"}, nil)

	req := models.SendMessageRequest{Body: "hi there"}

	message, err := service.SendMessage(context.Background(), conversationID.String(), req, sellerID.String())

	assert.NoError(t, err)
	assert.Equal(t, "hi there", message.Body)
	mockConversationRepo.AssertExpectations(t)
	mockMessageRepo.AssertExpectations(t)
}

func TestSendMessage_EmptyBody(t *testing.T) {
	service, mockConversationRepo, mockMessageRepo, _ := newMessagingService()

	req := models.SendMessageRequest{Body: "   "}

	_, err := service.SendMessage(context.Background(), uuid.New().String(), req, uuid.New().String())

	assert.ErrorIs(t, err, services.ErrEmptyMessageBody)
	mockConversationRepo.AssertNotCalled(t, "FindByID")
	mockMessageRepo.AssertNotCalled(t, "Create")
}

func TestSendMessage_ConversationNotFound(t *testing.T) {
	service, mockConversationRepo, mockMessageRepo, _ := newMessagingService()

	conversationID := uuid.New()
	mockConversationRepo.On("FindByID", mock.Anything, conversationID.String()).Return(nil, nil)

	req := models.SendMessageRequest{Body: "hello"}

	_, err := service.SendMessage(context.Background(), conversationID.String(), req, uuid.New().String())

	assert.ErrorIs(t, err, services.ErrConversationNotFound)
	mockMessageRepo.AssertNotCalled(t, "Create")
}

func TestSendMessage_NotParticipant(t *testing.T) {
	service, mockConversationRepo, mockMessageRepo, _ := newMessagingService()

	conversationID := uuid.New()
	conversation := &models.Conversation{ID: conversationID, BuyerID: uuid.New(), SellerID: uuid.New()}
	mockConversationRepo.On("FindByID", mock.Anything, conversationID.String()).Return(conversation, nil)

	req := models.SendMessageRequest{Body: "hello"}

	_, err := service.SendMessage(context.Background(), conversationID.String(), req, uuid.New().String())

	assert.ErrorIs(t, err, services.ErrNotConversationParticipant)
	mockMessageRepo.AssertNotCalled(t, "Create")
}

func TestListMessages_SuccessMarksReadByBuyer(t *testing.T) {
	service, mockConversationRepo, mockMessageRepo, _ := newMessagingService()

	conversationID := uuid.New()
	buyerID := uuid.New()
	sellerID := uuid.New()
	conversation := &models.Conversation{ID: conversationID, BuyerID: buyerID, SellerID: sellerID}
	expected := []*models.Message{{ID: 1, ConversationID: conversationID, Body: "hi"}}

	mockConversationRepo.On("FindByID", mock.Anything, conversationID.String()).Return(conversation, nil)
	mockMessageRepo.On("FindByConversationID", mock.Anything, conversationID.String(), int64(0), 20).Return(expected, false, nil)
	mockConversationRepo.On("MarkReadByBuyer", mock.Anything, conversationID.String()).Return(nil)

	messages, hasMore, err := service.ListMessages(context.Background(), conversationID.String(), buyerID.String(), 0, 20)

	assert.NoError(t, err)
	assert.Equal(t, expected, messages)
	assert.False(t, hasMore)
	mockConversationRepo.AssertExpectations(t)
	mockConversationRepo.AssertNotCalled(t, "MarkReadBySeller")
}

func TestListMessages_SuccessMarksReadBySeller(t *testing.T) {
	service, mockConversationRepo, mockMessageRepo, _ := newMessagingService()

	conversationID := uuid.New()
	buyerID := uuid.New()
	sellerID := uuid.New()
	conversation := &models.Conversation{ID: conversationID, BuyerID: buyerID, SellerID: sellerID}
	expected := []*models.Message{{ID: 1, ConversationID: conversationID, Body: "hi"}, {ID: 2, ConversationID: conversationID, Body: "there"}}

	mockConversationRepo.On("FindByID", mock.Anything, conversationID.String()).Return(conversation, nil)
	mockMessageRepo.On("FindByConversationID", mock.Anything, conversationID.String(), int64(0), 1).Return(expected, true, nil)
	mockConversationRepo.On("MarkReadBySeller", mock.Anything, conversationID.String()).Return(nil)

	messages, hasMore, err := service.ListMessages(context.Background(), conversationID.String(), sellerID.String(), 0, 1)

	assert.NoError(t, err)
	assert.Equal(t, expected, messages)
	assert.True(t, hasMore)
	mockConversationRepo.AssertExpectations(t)
	mockConversationRepo.AssertNotCalled(t, "MarkReadByBuyer")
}

func TestListMessages_MarkReadErrorStillReturnsMessages(t *testing.T) {
	service, mockConversationRepo, mockMessageRepo, _ := newMessagingService()

	conversationID := uuid.New()
	buyerID := uuid.New()
	sellerID := uuid.New()
	conversation := &models.Conversation{ID: conversationID, BuyerID: buyerID, SellerID: sellerID}
	expected := []*models.Message{{ID: 1, ConversationID: conversationID, Body: "hi"}}

	mockConversationRepo.On("FindByID", mock.Anything, conversationID.String()).Return(conversation, nil)
	mockMessageRepo.On("FindByConversationID", mock.Anything, conversationID.String(), int64(0), 20).Return(expected, false, nil)
	mockConversationRepo.On("MarkReadByBuyer", mock.Anything, conversationID.String()).Return(errors.New("update failed"))

	messages, hasMore, err := service.ListMessages(context.Background(), conversationID.String(), buyerID.String(), 0, 20)

	assert.NoError(t, err)
	assert.Equal(t, expected, messages)
	assert.False(t, hasMore)
}

func TestListMessages_NotParticipant(t *testing.T) {
	service, mockConversationRepo, mockMessageRepo, _ := newMessagingService()

	conversationID := uuid.New()
	conversation := &models.Conversation{ID: conversationID, BuyerID: uuid.New(), SellerID: uuid.New()}
	mockConversationRepo.On("FindByID", mock.Anything, conversationID.String()).Return(conversation, nil)

	_, _, err := service.ListMessages(context.Background(), conversationID.String(), uuid.New().String(), 0, 20)

	assert.ErrorIs(t, err, services.ErrNotConversationParticipant)
	mockMessageRepo.AssertNotCalled(t, "FindByConversationID")
	mockConversationRepo.AssertNotCalled(t, "MarkReadByBuyer")
	mockConversationRepo.AssertNotCalled(t, "MarkReadBySeller")
}
