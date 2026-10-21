package services_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	authModels "github.com/hfleury/horsemarketplacebk/internal/auth/models"
	tierModels "github.com/hfleury/horsemarketplacebk/internal/boosttiers/models"
	mockanalytics "github.com/hfleury/horsemarketplacebk/internal/mocks/analytics"
	mockboosttiers "github.com/hfleury/horsemarketplacebk/internal/mocks/boosttiers"
	mockemail "github.com/hfleury/horsemarketplacebk/internal/mocks/email"
	mockpayments "github.com/hfleury/horsemarketplacebk/internal/mocks/payments"
	mockproducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	"github.com/hfleury/horsemarketplacebk/internal/payments/models"
	"github.com/hfleury/horsemarketplacebk/internal/payments/providers"
	"github.com/hfleury/horsemarketplacebk/internal/payments/services"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	testProviderCode = "fake"
	testFrontendURL  = "https://hast.example"
	testReference    = "ref_123"
	sellerEmail      = "seller@example.se"
)

var fixedNow = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

// fakeProvider is a hand fake, following the fakeEnqueuer precedent.
type fakeProvider struct {
	checkoutErr  error
	lastSession  providers.CheckoutSession
	event        providers.Event
	parseErr     error
	parseInvoked bool
}

func (p *fakeProvider) Code() string { return testProviderCode }

func (p *fakeProvider) CreateCheckout(_ context.Context, session providers.CheckoutSession) (providers.CheckoutResult, error) {
	p.lastSession = session
	if p.checkoutErr != nil {
		return providers.CheckoutResult{}, p.checkoutErr
	}
	return providers.CheckoutResult{Reference: testReference, RedirectURL: "https://pay.example/checkout"}, nil
}

func (p *fakeProvider) ParseEvent(context.Context, http.Header, []byte) (providers.Event, error) {
	p.parseInvoked = true
	return p.event, p.parseErr
}

type fakeUsers struct {
	user *authModels.User
	err  error
}

func (f *fakeUsers) SelectUserByID(context.Context, string) (*authModels.User, error) {
	return f.user, f.err
}

type paymentFixture struct {
	service    *services.PaymentService
	repo       *mockpayments.MockPaymentRepository
	listings   *mockproducts.MockProductRepo
	tiers      *mockboosttiers.MockBoostTierRepository
	provider   *fakeProvider
	users      *fakeUsers
	sender     *mockemail.MockSender
	tracker    *mockanalytics.FakeTracker
	userID     uuid.UUID
	providerID uuid.UUID
	listing    *productModels.Product
	tier       *tierModels.BoostTier
}

func newPaymentFixture() *paymentFixture {
	userID := uuid.New()
	email := sellerEmail
	f := &paymentFixture{
		repo:       new(mockpayments.MockPaymentRepository),
		listings:   new(mockproducts.MockProductRepo),
		tiers:      new(mockboosttiers.MockBoostTierRepository),
		provider:   &fakeProvider{},
		users:      &fakeUsers{user: &authModels.User{Email: &email}},
		sender:     mockemail.NewMockSender(),
		tracker:    mockanalytics.NewFakeTracker(),
		userID:     userID,
		providerID: uuid.New(),
		listing: &productModels.Product{
			ID: uuid.New(), UserID: userID, Status: productModels.StatusPublished, Title: "Lovely mare",
		},
		tier: &tierModels.BoostTier{Code: "medium", Name: "Medium", PriceSEK: 99.99, DurationDays: 14, IsActive: true},
	}
	f.service = services.NewPaymentService(
		f.repo, f.listings, f.tiers, providers.NewRegistry(f.provider), f.users,
		testFrontendURL, config.NewZerologService(), func() time.Time { return fixedNow },
	)
	f.service.SetEmailSender(f.sender)
	f.service.SetTracker(f.tracker)
	return f
}

func (f *paymentFixture) checkoutRequest() models.CheckoutRequest {
	return models.CheckoutRequest{ProductID: f.listing.ID.String(), TierCode: f.tier.Code, Provider: testProviderCode}
}

// expectCheckoutLookups stubs the listing, provider row and tier lookups.
func (f *paymentFixture) expectCheckoutLookups() {
	f.listings.On("FindByID", mock.Anything, f.listing.ID.String()).Return(f.listing, nil)
	f.repo.On("FindProviderIDByCode", mock.Anything, testProviderCode).Return(&f.providerID, nil)
	f.tiers.On("FindActiveByCode", mock.Anything, f.tier.Code).Return(f.tier, nil)
}

func (f *paymentFixture) expectPurchaseCreated() {
	f.repo.On("CreatePendingWithBoost", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	f.repo.On("SetProviderReference", mock.Anything, mock.Anything, testReference).Return(nil)
}

func (f *paymentFixture) capturedPurchase() (*models.Payment, *models.ProductBoost) {
	for _, call := range f.repo.Calls {
		if call.Method == "CreatePendingWithBoost" {
			return call.Arguments.Get(1).(*models.Payment), call.Arguments.Get(2).(*models.ProductBoost)
		}
	}
	return nil, nil
}

func TestPaymentService_StartCheckout_Success(t *testing.T) {
	f := newPaymentFixture()
	f.expectCheckoutLookups()
	f.expectPurchaseCreated()

	response, err := f.service.StartCheckout(context.Background(), f.userID.String(), f.checkoutRequest())

	assert.NoError(t, err)
	payment, boost := f.capturedPurchase()
	assert.Equal(t, payment.ID, response.PaymentID)
	assert.Equal(t, "https://pay.example/checkout", response.CheckoutURL)

	assert.Equal(t, f.userID, *payment.UserID)
	assert.Equal(t, f.providerID, payment.ProviderID)
	assert.Equal(t, 99.99, payment.AmountSEK)
	assert.Equal(t, models.StatusPending, payment.Status)

	assert.Equal(t, payment.ID, boost.PaymentID)
	assert.Equal(t, f.listing.ID, boost.ProductID)
	assert.Equal(t, "medium", boost.Tier)
	assert.Equal(t, fixedNow, boost.StartsAt)
	assert.Equal(t, fixedNow.AddDate(0, 0, 14), boost.ExpiresAt)

	session := f.provider.lastSession
	assert.Equal(t, payment.ID, session.PaymentID)
	assert.Equal(t, int64(9999), session.AmountOre)
	assert.Equal(t, "Medium boost: Lovely mare", session.Description)
	assert.Equal(t, testFrontendURL+"/boost/success?payment_id="+payment.ID.String(), session.SuccessURL)
	assert.Equal(t, testFrontendURL+"/boost/cancel?payment_id="+payment.ID.String(), session.CancelURL)
	f.repo.AssertCalled(t, "SetProviderReference", mock.Anything, payment.ID, testReference)
}

func TestPaymentService_StartCheckout_StartDate(t *testing.T) {
	future := fixedNow.Add(72 * time.Hour)
	past := fixedNow.Add(-time.Hour)
	inStockholm := time.Date(2026, 12, 24, 9, 0, 0, 0, time.FixedZone("CET", 3600))

	tests := []struct {
		name      string
		requested *time.Time
		want      time.Time
	}{
		{name: "not given starts now", requested: nil, want: fixedNow},
		{name: "in the past starts now", requested: &past, want: fixedNow},
		{name: "exactly now", requested: &fixedNow, want: fixedNow},
		{name: "in the future is kept", requested: &future, want: future},
		{name: "stored in UTC", requested: &inStockholm, want: inStockholm.UTC()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPaymentFixture()
			f.expectCheckoutLookups()
			f.expectPurchaseCreated()
			req := f.checkoutRequest()
			req.StartsAt = tt.requested

			_, err := f.service.StartCheckout(context.Background(), f.userID.String(), req)

			assert.NoError(t, err)
			_, boost := f.capturedPurchase()
			assert.Equal(t, tt.want, boost.StartsAt)
			assert.Equal(t, tt.want.AddDate(0, 0, 14), boost.ExpiresAt)
		})
	}
}

func TestPaymentService_StartCheckout_ListingStatus(t *testing.T) {
	tests := []struct {
		status  productModels.ProductStatus
		wantErr error
	}{
		{status: productModels.StatusPublished},
		{status: productModels.StatusPendingApproval},
		{status: productModels.StatusDraft, wantErr: services.ErrListingNotBoostable},
		{status: productModels.StatusSold, wantErr: services.ErrListingNotBoostable},
		{status: productModels.StatusArchived, wantErr: services.ErrListingNotBoostable},
		{status: productModels.StatusDeleted, wantErr: services.ErrListingNotBoostable},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			f := newPaymentFixture()
			f.listing.Status = tt.status
			f.expectCheckoutLookups()
			f.expectPurchaseCreated()

			_, err := f.service.StartCheckout(context.Background(), f.userID.String(), f.checkoutRequest())

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
			f.repo.AssertNotCalled(t, "CreatePendingWithBoost", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestPaymentService_StartCheckout_Rejections(t *testing.T) {
	repoErr := errors.New("db down")

	tests := []struct {
		name    string
		setup   func(f *paymentFixture, req *models.CheckoutRequest, userID *string)
		wantErr error
	}{
		{
			name:    "invalid user id",
			setup:   func(_ *paymentFixture, _ *models.CheckoutRequest, userID *string) { *userID = "nope" },
			wantErr: services.ErrInvalidPaymentRequest,
		},
		{
			name:    "invalid product id",
			setup:   func(_ *paymentFixture, req *models.CheckoutRequest, _ *string) { req.ProductID = "nope" },
			wantErr: services.ErrInvalidPaymentRequest,
		},
		{
			name: "listing not found",
			setup: func(f *paymentFixture, _ *models.CheckoutRequest, _ *string) {
				f.listings.On("FindByID", mock.Anything, mock.Anything).Return(nil, nil)
			},
			wantErr: services.ErrListingNotFound,
		},
		{
			name: "listing lookup fails",
			setup: func(f *paymentFixture, _ *models.CheckoutRequest, _ *string) {
				f.listings.On("FindByID", mock.Anything, mock.Anything).Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
		{
			name: "not the owner",
			setup: func(f *paymentFixture, _ *models.CheckoutRequest, _ *string) {
				f.listing.UserID = uuid.New()
				f.listings.On("FindByID", mock.Anything, mock.Anything).Return(f.listing, nil)
			},
			wantErr: services.ErrNotListingOwner,
		},
		{
			name: "unknown provider",
			setup: func(f *paymentFixture, req *models.CheckoutRequest, _ *string) {
				req.Provider = "stripe"
				f.listings.On("FindByID", mock.Anything, mock.Anything).Return(f.listing, nil)
			},
			wantErr: services.ErrUnknownProvider,
		},
		{
			name: "provider lookup fails",
			setup: func(f *paymentFixture, _ *models.CheckoutRequest, _ *string) {
				f.listings.On("FindByID", mock.Anything, mock.Anything).Return(f.listing, nil)
				f.repo.On("FindProviderIDByCode", mock.Anything, testProviderCode).Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
		{
			name: "unknown or inactive tier",
			setup: func(f *paymentFixture, _ *models.CheckoutRequest, _ *string) {
				f.listings.On("FindByID", mock.Anything, mock.Anything).Return(f.listing, nil)
				f.repo.On("FindProviderIDByCode", mock.Anything, testProviderCode).Return(&f.providerID, nil)
				f.tiers.On("FindActiveByCode", mock.Anything, mock.Anything).Return(nil, nil)
			},
			wantErr: services.ErrTierNotFound,
		},
		{
			name: "tier lookup fails",
			setup: func(f *paymentFixture, _ *models.CheckoutRequest, _ *string) {
				f.listings.On("FindByID", mock.Anything, mock.Anything).Return(f.listing, nil)
				f.repo.On("FindProviderIDByCode", mock.Anything, testProviderCode).Return(&f.providerID, nil)
				f.tiers.On("FindActiveByCode", mock.Anything, mock.Anything).Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
		{
			name: "creating the pending payment fails",
			setup: func(f *paymentFixture, _ *models.CheckoutRequest, _ *string) {
				f.expectCheckoutLookups()
				f.repo.On("CreatePendingWithBoost", mock.Anything, mock.Anything, mock.Anything).Return(repoErr)
			},
			wantErr: repoErr,
		},
		{
			name: "saving the provider reference fails",
			setup: func(f *paymentFixture, _ *models.CheckoutRequest, _ *string) {
				f.expectCheckoutLookups()
				f.repo.On("CreatePendingWithBoost", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				f.repo.On("SetProviderReference", mock.Anything, mock.Anything, testReference).Return(repoErr)
			},
			wantErr: repoErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPaymentFixture()
			req := f.checkoutRequest()
			userID := f.userID.String()
			tt.setup(f, &req, &userID)

			response, err := f.service.StartCheckout(context.Background(), userID, req)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, response)
		})
	}
}

func TestPaymentService_StartCheckout_RegisteredProviderWithoutRow(t *testing.T) {
	f := newPaymentFixture()
	f.listings.On("FindByID", mock.Anything, mock.Anything).Return(f.listing, nil)
	f.repo.On("FindProviderIDByCode", mock.Anything, testProviderCode).Return(nil, nil)

	_, err := f.service.StartCheckout(context.Background(), f.userID.String(), f.checkoutRequest())

	assert.ErrorContains(t, err, "has no database row")
	assert.NotErrorIs(t, err, services.ErrUnknownProvider)
}

func TestPaymentService_StartCheckout_ProviderFailure(t *testing.T) {
	tests := []struct {
		name    string
		markErr error
	}{
		{name: "payment marked failed"},
		{name: "marking failed also fails", markErr: errors.New("db down")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPaymentFixture()
			f.provider.checkoutErr = errors.New("provider timeout")
			f.expectCheckoutLookups()
			f.repo.On("CreatePendingWithBoost", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			f.repo.On("MarkCheckoutFailed", mock.Anything, mock.Anything).Return(tt.markErr)

			_, err := f.service.StartCheckout(context.Background(), f.userID.String(), f.checkoutRequest())

			assert.ErrorIs(t, err, services.ErrProviderUnavailable)
			payment, _ := f.capturedPurchase()
			f.repo.AssertCalled(t, "MarkCheckoutFailed", mock.Anything, payment.ID)
			f.repo.AssertNotCalled(t, "SetProviderReference", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func (f *paymentFixture) confirmedPayment() *models.ConfirmedPayment {
	return &models.ConfirmedPayment{
		PaymentID: uuid.New(),
		UserID:    &f.userID,
		AmountSEK: 99,
		Boost: models.ProductBoost{
			ProductID: f.listing.ID,
			Tier:      "medium",
			StartsAt:  fixedNow,
			ExpiresAt: fixedNow.AddDate(0, 0, 14),
		},
	}
}

func (f *paymentFixture) handleEvent(outcome providers.Outcome) error {
	f.provider.event = providers.Event{Reference: testReference, Outcome: outcome}
	return f.service.HandleProviderEvent(context.Background(), testProviderCode, http.Header{}, []byte(`{}`))
}

func (f *paymentFixture) expectProviderRow() {
	f.repo.On("FindProviderIDByCode", mock.Anything, testProviderCode).Return(&f.providerID, nil)
}

func TestPaymentService_HandleProviderEvent_Paid(t *testing.T) {
	f := newPaymentFixture()
	f.expectProviderRow()
	confirmed := f.confirmedPayment()
	f.repo.On("ConfirmPaid", mock.Anything, f.providerID, testReference).Return(confirmed, nil)
	f.listings.On("FindByID", mock.Anything, f.listing.ID.String()).Return(f.listing, nil)

	err := f.handleEvent(providers.OutcomePaid)

	assert.NoError(t, err)
	assert.Len(t, f.sender.Sent, 1)
	assert.Equal(t, sellerEmail, f.sender.Sent[0].To)
	assert.Equal(t, `Your boost for "Lovely mare" is confirmed`, f.sender.Sent[0].Subject)
	assert.Contains(t, f.sender.Sent[0].Body, "99 SEK (incl. moms)")

	events := f.tracker.Events()
	assert.Len(t, events, 1)
	assert.Equal(t, "boost_paid", events[0].Name)
	assert.Equal(t, f.userID.String(), events[0].DistinctID)
	assert.Equal(t, map[string]any{
		"listing_id": f.listing.ID.String(), "tier": "medium", "amount_sek": 99.0, "provider": testProviderCode,
	}, events[0].Properties)
}

func TestPaymentService_HandleProviderEvent_PaidNotApplied(t *testing.T) {
	tests := []struct {
		name    string
		exists  bool
		wantErr error
	}{
		{name: "duplicate delivery is safe", exists: true},
		{name: "unknown reference is retried", exists: false, wantErr: services.ErrPaymentNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPaymentFixture()
			f.expectProviderRow()
			f.repo.On("ConfirmPaid", mock.Anything, f.providerID, testReference).Return(nil, nil)
			f.repo.On("PaymentExists", mock.Anything, f.providerID, testReference).Return(tt.exists, nil)

			err := f.handleEvent(providers.OutcomePaid)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Empty(t, f.sender.Sent)
			assert.Empty(t, f.tracker.Events())
		})
	}
}

func TestPaymentService_HandleProviderEvent_Transitions(t *testing.T) {
	tests := []struct {
		name         string
		outcome      providers.Outcome
		from, to     models.PaymentStatus
		transitioned bool
		exists       bool
		wantErr      error
	}{
		{name: "failed while pending", outcome: providers.OutcomeFailed, from: models.StatusPending, to: models.StatusFailed, transitioned: true},
		{name: "failed again is safe", outcome: providers.OutcomeFailed, from: models.StatusPending, to: models.StatusFailed, exists: true},
		{name: "failed for unknown reference", outcome: providers.OutcomeFailed, from: models.StatusPending, to: models.StatusFailed, wantErr: services.ErrPaymentNotFound},
		{name: "refunded while paid", outcome: providers.OutcomeRefunded, from: models.StatusPaid, to: models.StatusRefunded, transitioned: true},
		{name: "refunded while pending changes nothing", outcome: providers.OutcomeRefunded, from: models.StatusPaid, to: models.StatusRefunded, exists: true},
		{name: "refunded for unknown reference", outcome: providers.OutcomeRefunded, from: models.StatusPaid, to: models.StatusRefunded, wantErr: services.ErrPaymentNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPaymentFixture()
			f.expectProviderRow()
			f.repo.On("TransitionStatus", mock.Anything, f.providerID, testReference, tt.from, tt.to).Return(tt.transitioned, nil)
			f.repo.On("PaymentExists", mock.Anything, f.providerID, testReference).Return(tt.exists, nil)

			err := f.handleEvent(tt.outcome)

			assert.ErrorIs(t, err, tt.wantErr)
			if tt.transitioned {
				f.repo.AssertNotCalled(t, "PaymentExists", mock.Anything, mock.Anything, mock.Anything)
			}
			f.repo.AssertNotCalled(t, "ConfirmPaid", mock.Anything, mock.Anything, mock.Anything)
			assert.Empty(t, f.sender.Sent)
		})
	}
}

func TestPaymentService_HandleProviderEvent_Errors(t *testing.T) {
	repoErr := errors.New("db down")

	tests := []struct {
		name    string
		setup   func(f *paymentFixture)
		outcome providers.Outcome
		wantErr error
	}{
		{
			name:    "parse error passes through",
			setup:   func(f *paymentFixture) { f.provider.parseErr = providers.ErrInvalidSignature },
			outcome: providers.OutcomePaid,
			wantErr: providers.ErrInvalidSignature,
		},
		{
			name:    "unsupported outcome",
			setup:   func(f *paymentFixture) { f.expectProviderRow() },
			outcome: providers.Outcome("chargeback"),
			wantErr: providers.ErrMalformedEvent,
		},
		{
			name: "provider lookup fails",
			setup: func(f *paymentFixture) {
				f.repo.On("FindProviderIDByCode", mock.Anything, testProviderCode).Return(nil, repoErr)
			},
			outcome: providers.OutcomePaid,
			wantErr: repoErr,
		},
		{
			name: "confirm fails",
			setup: func(f *paymentFixture) {
				f.expectProviderRow()
				f.repo.On("ConfirmPaid", mock.Anything, mock.Anything, mock.Anything).Return(nil, repoErr)
			},
			outcome: providers.OutcomePaid,
			wantErr: repoErr,
		},
		{
			name: "existence check fails",
			setup: func(f *paymentFixture) {
				f.expectProviderRow()
				f.repo.On("ConfirmPaid", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				f.repo.On("PaymentExists", mock.Anything, mock.Anything, mock.Anything).Return(false, repoErr)
			},
			outcome: providers.OutcomePaid,
			wantErr: repoErr,
		},
		{
			name: "transition fails",
			setup: func(f *paymentFixture) {
				f.expectProviderRow()
				f.repo.On("TransitionStatus", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, repoErr)
			},
			outcome: providers.OutcomeFailed,
			wantErr: repoErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPaymentFixture()
			tt.setup(f)

			err := f.handleEvent(tt.outcome)

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestPaymentService_HandleProviderEvent_Ignored(t *testing.T) {
	f := newPaymentFixture()

	err := f.handleEvent(providers.OutcomeIgnored)

	assert.NoError(t, err)
	assert.Empty(t, f.repo.Calls)
}

func TestPaymentService_HandleProviderEvent_UnknownProvider(t *testing.T) {
	f := newPaymentFixture()

	err := f.service.HandleProviderEvent(context.Background(), "stripe", http.Header{}, nil)

	assert.ErrorIs(t, err, services.ErrUnknownProvider)
	assert.False(t, f.provider.parseInvoked)
	assert.Empty(t, f.repo.Calls)
}

func TestPaymentService_HandleProviderEvent_PaidEmailIsBestEffort(t *testing.T) {
	sendErr := errors.New("smtp down")

	tests := []struct {
		name      string
		setup     func(f *paymentFixture, confirmed *models.ConfirmedPayment)
		wantSent  int
		wantEvent bool
	}{
		{
			name: "send fails",
			setup: func(f *paymentFixture, _ *models.ConfirmedPayment) {
				f.sender.ErrFor = map[string]error{sellerEmail: sendErr}
			},
			wantEvent: true,
		},
		{
			name:      "seller has no email",
			setup:     func(f *paymentFixture, _ *models.ConfirmedPayment) { f.users.user = &authModels.User{} },
			wantEvent: true,
		},
		{
			name:      "seller not found",
			setup:     func(f *paymentFixture, _ *models.ConfirmedPayment) { f.users.user = nil },
			wantEvent: true,
		},
		{
			name:      "seller lookup fails",
			setup:     func(f *paymentFixture, _ *models.ConfirmedPayment) { f.users.err = errors.New("db down") },
			wantEvent: true,
		},
		{
			name: "listing lookup fails",
			setup: func(f *paymentFixture, _ *models.ConfirmedPayment) {
				f.listings.ExpectedCalls = nil
				f.listings.On("FindByID", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))
			},
			wantEvent: true,
		},
		{
			name: "listing gone",
			setup: func(f *paymentFixture, _ *models.ConfirmedPayment) {
				f.listings.ExpectedCalls = nil
				f.listings.On("FindByID", mock.Anything, mock.Anything).Return(nil, nil)
			},
			wantEvent: true,
		},
		{
			name:  "buyer account deleted",
			setup: func(_ *paymentFixture, confirmed *models.ConfirmedPayment) { confirmed.UserID = nil },
		},
		{
			name:      "no sender configured",
			setup:     func(f *paymentFixture, _ *models.ConfirmedPayment) { f.service.SetEmailSender(nil) },
			wantEvent: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPaymentFixture()
			f.expectProviderRow()
			confirmed := f.confirmedPayment()
			f.listings.On("FindByID", mock.Anything, mock.Anything).Return(f.listing, nil)
			tt.setup(f, confirmed)
			f.repo.On("ConfirmPaid", mock.Anything, mock.Anything, mock.Anything).Return(confirmed, nil)

			err := f.handleEvent(providers.OutcomePaid)

			assert.NoError(t, err)
			assert.Len(t, f.sender.Sent, tt.wantSent)
			assert.Equal(t, tt.wantEvent, len(f.tracker.Events()) == 1)
		})
	}
}

func TestPaymentService_GetPaymentStatus(t *testing.T) {
	repoErr := errors.New("db down")
	paymentID := uuid.New()
	found := &models.PaymentStatusResponse{PaymentID: paymentID, Status: models.StatusPaid}

	tests := []struct {
		name      string
		userID    string
		paymentID string
		repoValue *models.PaymentStatusResponse
		repoErr   error
		want      *models.PaymentStatusResponse
		wantErr   error
	}{
		{name: "found", paymentID: paymentID.String(), repoValue: found, want: found},
		{name: "missing or someone else's", paymentID: paymentID.String(), wantErr: services.ErrPaymentNotFound},
		{name: "repository error", paymentID: paymentID.String(), repoErr: repoErr, wantErr: repoErr},
		{name: "invalid payment id", paymentID: "nope", wantErr: services.ErrInvalidPaymentRequest},
		{name: "invalid user id", userID: "nope", paymentID: paymentID.String(), wantErr: services.ErrInvalidPaymentRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPaymentFixture()
			userID := tt.userID
			if userID == "" {
				userID = f.userID.String()
			}
			f.repo.On("FindStatusForUser", mock.Anything, paymentID, f.userID).Return(tt.repoValue, tt.repoErr)

			status, err := f.service.GetPaymentStatus(context.Background(), userID, tt.paymentID)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, status)
		})
	}
}

func TestPaymentService_ListProviders(t *testing.T) {
	assert.Equal(t, []string{testProviderCode}, newPaymentFixture().service.ListProviders())
}

// The email shows Swedish local time, which moves between CEST and CET.
func TestPaymentService_HandleProviderEvent_ConfirmationEmailContent(t *testing.T) {
	tests := []struct {
		name       string
		amountSEK  float64
		startsAt   time.Time
		expiresAt  time.Time
		wantAmount string
		wantWindow string
	}{
		{
			name:       "summer time",
			amountSEK:  49,
			startsAt:   time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
			expiresAt:  time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
			wantAmount: "49 SEK (incl. moms)",
			wantWindow: "from 2026-07-01 12:00 to 2026-07-08 12:00 (Swedish time)",
		},
		{
			name:       "across the switch to winter time",
			amountSEK:  199.5,
			startsAt:   time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
			expiresAt:  time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC),
			wantAmount: "199.5 SEK (incl. moms)",
			wantWindow: "from 2026-10-21 14:00 to 2026-11-20 13:00 (Swedish time)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newPaymentFixture()
			f.expectProviderRow()
			confirmed := f.confirmedPayment()
			confirmed.AmountSEK = tt.amountSEK
			confirmed.Boost.StartsAt = tt.startsAt
			confirmed.Boost.ExpiresAt = tt.expiresAt
			f.repo.On("ConfirmPaid", mock.Anything, mock.Anything, mock.Anything).Return(confirmed, nil)
			f.listings.On("FindByID", mock.Anything, mock.Anything).Return(f.listing, nil)

			assert.NoError(t, f.handleEvent(providers.OutcomePaid))

			body := f.sender.Sent[0].Body
			assert.Contains(t, body, tt.wantAmount)
			assert.Contains(t, body, tt.wantWindow)
			assert.Contains(t, body, `the "medium" boost on "Lovely mare"`)
			assert.Contains(t, body, testFrontendURL+"/listings/"+f.listing.ID.String())
		})
	}
}
