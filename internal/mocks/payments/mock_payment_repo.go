package mockpayments

import (
	"context"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/payments/models"
	"github.com/stretchr/testify/mock"
)

type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) FindProviderIDByCode(ctx context.Context, code string) (*uuid.UUID, error) {
	args := m.Called(ctx, code)
	providerID, _ := args.Get(0).(*uuid.UUID)
	return providerID, args.Error(1)
}

func (m *MockPaymentRepository) CreatePendingWithBoost(ctx context.Context, payment *models.Payment, boost *models.ProductBoost) error {
	args := m.Called(ctx, payment, boost)
	return args.Error(0)
}

func (m *MockPaymentRepository) SetProviderReference(ctx context.Context, paymentID uuid.UUID, reference string) error {
	args := m.Called(ctx, paymentID, reference)
	return args.Error(0)
}

func (m *MockPaymentRepository) MarkCheckoutFailed(ctx context.Context, paymentID uuid.UUID) error {
	args := m.Called(ctx, paymentID)
	return args.Error(0)
}

func (m *MockPaymentRepository) ConfirmPaid(ctx context.Context, providerID uuid.UUID, reference string) (*models.ConfirmedPayment, error) {
	args := m.Called(ctx, providerID, reference)
	confirmed, _ := args.Get(0).(*models.ConfirmedPayment)
	return confirmed, args.Error(1)
}

func (m *MockPaymentRepository) TransitionStatus(ctx context.Context, providerID uuid.UUID, reference string, from, to models.PaymentStatus) (bool, error) {
	args := m.Called(ctx, providerID, reference, from, to)
	return args.Bool(0), args.Error(1)
}

func (m *MockPaymentRepository) PaymentExists(ctx context.Context, providerID uuid.UUID, reference string) (bool, error) {
	args := m.Called(ctx, providerID, reference)
	return args.Bool(0), args.Error(1)
}

func (m *MockPaymentRepository) FindStatusForUser(ctx context.Context, paymentID, userID uuid.UUID) (*models.PaymentStatusResponse, error) {
	args := m.Called(ctx, paymentID, userID)
	status, _ := args.Get(0).(*models.PaymentStatusResponse)
	return status, args.Error(1)
}
