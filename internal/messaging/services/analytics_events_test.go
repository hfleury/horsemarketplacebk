package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
	"github.com/hfleury/horsemarketplacebk/internal/messaging/models"
	mockanalytics "github.com/hfleury/horsemarketplacebk/internal/mocks/analytics"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateConversation_NewConversationTracksConversationStarted(t *testing.T) {
	service, mockConversationRepo, _, mockProductRepo := newMessagingService()
	tracker := mockanalytics.NewFakeTracker()
	service.SetTracker(tracker)
	categoryID := uuid.New()
	product := &productModels.Product{ID: uuid.New(), UserID: uuid.New(), CategoryID: &categoryID, Type: productModels.TypeHorse, Status: productModels.StatusPublished}
	buyerID := uuid.New().String()

	mockProductRepo.On("FindByID", mock.Anything, product.ID.String()).Return(product, nil)
	mockConversationRepo.On("FindByProductAndBuyer", mock.Anything, product.ID.String(), buyerID).Return(nil, nil)
	mockConversationRepo.On("Create", mock.Anything, mock.Anything).Return(&models.Conversation{ID: uuid.New()}, nil)

	_, err := service.CreateConversation(context.Background(), models.CreateConversationRequest{ProductID: product.ID.String()}, buyerID)

	require.NoError(t, err)
	assert.Equal(t, []analytics.Event{{
		Name:       analytics.EventConversationStarted,
		DistinctID: buyerID,
		Properties: product.AnalyticsProperties(),
	}}, tracker.Events())
}

func TestCreateConversation_ExistingOrFailedConversationTracksNothing(t *testing.T) {
	tests := []struct {
		name     string
		existing *models.Conversation
		create   error
	}{
		{name: "existing conversation", existing: &models.Conversation{ID: uuid.New()}},
		{name: "create fails", create: errors.New("db error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockConversationRepo, _, mockProductRepo := newMessagingService()
			tracker := mockanalytics.NewFakeTracker()
			service.SetTracker(tracker)
			product := &productModels.Product{ID: uuid.New(), UserID: uuid.New()}
			buyerID := uuid.New().String()

			mockProductRepo.On("FindByID", mock.Anything, product.ID.String()).Return(product, nil)
			mockConversationRepo.On("FindByProductAndBuyer", mock.Anything, product.ID.String(), buyerID).Return(tt.existing, nil)
			mockConversationRepo.On("Create", mock.Anything, mock.Anything).Return(nil, tt.create)

			_, _ = service.CreateConversation(context.Background(), models.CreateConversationRequest{ProductID: product.ID.String()}, buyerID)

			assert.Empty(t, tracker.Events())
		})
	}
}

func TestSendMessage_TracksMessageSentWithSenderRole(t *testing.T) {
	buyerID, sellerID := uuid.New(), uuid.New()
	tests := []struct {
		name     string
		senderID uuid.UUID
		role     string
	}{
		{name: "buyer", senderID: buyerID, role: "buyer"},
		{name: "seller", senderID: sellerID, role: "seller"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, mockConversationRepo, mockMessageRepo, _ := newMessagingService()
			tracker := mockanalytics.NewFakeTracker()
			service.SetTracker(tracker)
			conversation := &models.Conversation{ID: uuid.New(), ProductID: uuid.New(), BuyerID: buyerID, SellerID: sellerID}

			mockConversationRepo.On("FindByID", mock.Anything, conversation.ID.String()).Return(conversation, nil)
			mockMessageRepo.On("Create", mock.Anything, mock.Anything).Return(&models.Message{ID: 1}, nil)

			_, err := service.SendMessage(context.Background(), conversation.ID.String(), models.SendMessageRequest{Body: "private text"}, tt.senderID.String())

			require.NoError(t, err)
			assert.Equal(t, []analytics.Event{{
				Name:       analytics.EventMessageSent,
				DistinctID: tt.senderID.String(),
				Properties: map[string]any{
					"conversation_id": conversation.ID.String(),
					"listing_id":      conversation.ProductID.String(),
					"sender_role":     tt.role,
				},
			}}, tracker.Events())
		})
	}
}
