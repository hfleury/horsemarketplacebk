package mockmessaging

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/messaging/models"
	"github.com/stretchr/testify/mock"
)

type MockConversationRepository struct {
	mock.Mock
}

func (m *MockConversationRepository) Create(ctx context.Context, conversation *models.Conversation) (*models.Conversation, error) {
	args := m.Called(ctx, conversation)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Conversation), args.Error(1)
}

func (m *MockConversationRepository) FindByID(ctx context.Context, id string) (*models.Conversation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Conversation), args.Error(1)
}

func (m *MockConversationRepository) FindByProductAndBuyer(ctx context.Context, productID, buyerID string) (*models.Conversation, error) {
	args := m.Called(ctx, productID, buyerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Conversation), args.Error(1)
}

func (m *MockConversationRepository) MarkReadByBuyer(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockConversationRepository) MarkReadBySeller(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockConversationRepository) FindByUserID(ctx context.Context, userID string, page, limit int) ([]*models.ConversationSummary, int, error) {
	args := m.Called(ctx, userID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.ConversationSummary), args.Int(1), args.Error(2)
}

func (m *MockConversationRepository) CountUnreadByUserID(ctx context.Context, userID string) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}
