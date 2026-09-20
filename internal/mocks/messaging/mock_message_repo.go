package mockmessaging

import (
	"context"

	"github.com/hfleury/horsemarketplacebk/internal/messaging/models"
	"github.com/stretchr/testify/mock"
)

type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) Create(ctx context.Context, message *models.Message) (*models.Message, error) {
	args := m.Called(ctx, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

func (m *MockMessageRepository) FindByConversationID(ctx context.Context, conversationID string, afterID int64, limit int) ([]*models.Message, bool, error) {
	args := m.Called(ctx, conversationID, afterID, limit)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).([]*models.Message), args.Bool(1), args.Error(2)
}
