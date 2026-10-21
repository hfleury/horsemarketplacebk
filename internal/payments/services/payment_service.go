package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
	authModels "github.com/hfleury/horsemarketplacebk/internal/auth/models"
	tierModels "github.com/hfleury/horsemarketplacebk/internal/boosttiers/models"
	"github.com/hfleury/horsemarketplacebk/internal/email"
	"github.com/hfleury/horsemarketplacebk/internal/payments/models"
	"github.com/hfleury/horsemarketplacebk/internal/payments/providers"
	"github.com/hfleury/horsemarketplacebk/internal/payments/repositories"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
)

var (
	ErrInvalidPaymentRequest = errors.New("invalid payment request")
	ErrListingNotFound       = errors.New("listing not found")
	ErrNotListingOwner       = errors.New("only the listing's owner can boost it")
	ErrListingNotBoostable   = errors.New("only published or pending listings can be boosted")
	ErrTierNotFound          = errors.New("boost tier not found")
	ErrUnknownProvider       = providers.ErrUnknownProvider
	ErrProviderUnavailable   = errors.New("payment provider unavailable")
	ErrPaymentNotFound       = errors.New("payment not found")
)

// listingFinder must not be the product service: its FindByID counts a view.
type listingFinder interface {
	FindByID(ctx context.Context, id string) (*productModels.Product, error)
}

type activeTierFinder interface {
	FindActiveByCode(ctx context.Context, code string) (*tierModels.BoostTier, error)
}

type userFinder interface {
	SelectUserByID(ctx context.Context, id string) (*authModels.User, error)
}

type PaymentService struct {
	repo        repositories.PaymentRepository
	listings    listingFinder
	tiers       activeTierFinder
	registry    *providers.Registry
	users       userFinder
	frontendURL string
	logger      config.Logging
	now         func() time.Time
	emailSender email.Sender
	tracker     analytics.Tracker
}

func NewPaymentService(
	repo repositories.PaymentRepository,
	listings listingFinder,
	tiers activeTierFinder,
	registry *providers.Registry,
	users userFinder,
	frontendURL string,
	logger config.Logging,
	now func() time.Time,
) *PaymentService {
	return &PaymentService{
		repo:        repo,
		listings:    listings,
		tiers:       tiers,
		registry:    registry,
		users:       users,
		frontendURL: frontendURL,
		logger:      logger,
		now:         now,
		tracker:     analytics.NoopClient{},
	}
}

func (s *PaymentService) SetEmailSender(sender email.Sender) {
	s.emailSender = sender
}

func (s *PaymentService) SetTracker(tracker analytics.Tracker) {
	s.tracker = tracker
}

func (s *PaymentService) ListProviders() []string {
	return s.registry.Codes()
}

// StartCheckout records a pending payment with its boost window, then opens
// the provider's checkout. The boost only counts once the provider confirms.
func (s *PaymentService) StartCheckout(ctx context.Context, userID string, req models.CheckoutRequest) (*models.CheckoutResponse, error) {
	userUUID, err := parseUUID("user id", userID)
	if err != nil {
		return nil, err
	}
	listing, err := s.findBoostableListing(ctx, userUUID, req.ProductID)
	if err != nil {
		return nil, err
	}
	provider, providerID, err := s.resolveProvider(ctx, req.Provider)
	if err != nil {
		return nil, err
	}
	tier, err := s.resolveTier(ctx, req.TierCode)
	if err != nil {
		return nil, err
	}

	payment, boost := s.newPendingPurchase(userUUID, providerID, listing.ID, tier, req.StartsAt)
	if err := s.repo.CreatePendingWithBoost(ctx, payment, boost); err != nil {
		return nil, err
	}
	return s.openCheckout(ctx, provider, payment, tier, listing.Title)
}

func (s *PaymentService) GetPaymentStatus(ctx context.Context, userID, paymentID string) (*models.PaymentStatusResponse, error) {
	userUUID, err := parseUUID("user id", userID)
	if err != nil {
		return nil, err
	}
	paymentUUID, err := parseUUID("payment id", paymentID)
	if err != nil {
		return nil, err
	}

	status, err := s.repo.FindStatusForUser(ctx, paymentUUID, userUUID)
	if err != nil {
		return nil, err
	}
	if status == nil {
		return nil, ErrPaymentNotFound
	}
	return status, nil
}

func (s *PaymentService) findBoostableListing(ctx context.Context, userID uuid.UUID, productID string) (*productModels.Product, error) {
	if _, err := parseUUID("product id", productID); err != nil {
		return nil, err
	}
	listing, err := s.listings.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if listing == nil {
		return nil, ErrListingNotFound
	}
	if listing.UserID != userID {
		return nil, ErrNotListingOwner
	}
	if !isBoostable(listing.Status) {
		return nil, ErrListingNotBoostable
	}
	return listing, nil
}

func isBoostable(status productModels.ProductStatus) bool {
	return status == productModels.StatusPublished || status == productModels.StatusPendingApproval
}

func (s *PaymentService) resolveProvider(ctx context.Context, code string) (providers.Provider, uuid.UUID, error) {
	provider, err := s.registry.Get(code)
	if err != nil {
		return nil, uuid.Nil, err
	}
	providerID, err := s.findProviderID(ctx, code)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return provider, providerID, nil
}

// findProviderID fails loudly when a registered provider has no seed row:
// that is a deployment mistake, not a client error.
func (s *PaymentService) findProviderID(ctx context.Context, code string) (uuid.UUID, error) {
	providerID, err := s.repo.FindProviderIDByCode(ctx, code)
	if err != nil {
		return uuid.Nil, err
	}
	if providerID == nil {
		return uuid.Nil, fmt.Errorf("payment provider %q is registered but has no database row", code)
	}
	return *providerID, nil
}

func (s *PaymentService) resolveTier(ctx context.Context, code string) (*tierModels.BoostTier, error) {
	tier, err := s.tiers.FindActiveByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if tier == nil {
		return nil, fmt.Errorf("%w: %q", ErrTierNotFound, code)
	}
	return tier, nil
}

func (s *PaymentService) newPendingPurchase(userID, providerID, productID uuid.UUID, tier *tierModels.BoostTier, requestedStart *time.Time) (*models.Payment, *models.ProductBoost) {
	payment := &models.Payment{
		ID:         uuid.New(),
		UserID:     &userID,
		ProviderID: providerID,
		AmountSEK:  tier.PriceSEK,
		Status:     models.StatusPending,
	}
	startsAt := s.resolveStartsAt(requestedStart)
	boost := &models.ProductBoost{
		ID:        uuid.New(),
		ProductID: productID,
		PaymentID: payment.ID,
		Tier:      tier.Code,
		StartsAt:  startsAt,
		ExpiresAt: startsAt.AddDate(0, 0, tier.DurationDays),
	}
	return payment, boost
}

// resolveStartsAt treats a missing or past start as "now".
func (s *PaymentService) resolveStartsAt(requested *time.Time) time.Time {
	now := s.now().UTC()
	if requested == nil || requested.Before(now) {
		return now
	}
	return requested.UTC()
}

func (s *PaymentService) openCheckout(ctx context.Context, provider providers.Provider, payment *models.Payment, tier *tierModels.BoostTier, listingTitle string) (*models.CheckoutResponse, error) {
	result, err := provider.CreateCheckout(ctx, providers.CheckoutSession{
		PaymentID:   payment.ID,
		AmountOre:   models.SEKToOre(payment.AmountSEK),
		Description: fmt.Sprintf("%s boost: %s", tier.Name, listingTitle),
		SuccessURL:  s.boostPageURL("success", payment.ID),
		CancelURL:   s.boostPageURL("cancel", payment.ID),
	})
	if err != nil {
		s.failCheckout(ctx, provider.Code(), payment.ID, err)
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}

	if err := s.repo.SetProviderReference(ctx, payment.ID, result.Reference); err != nil {
		return nil, err
	}
	return &models.CheckoutResponse{PaymentID: payment.ID, CheckoutURL: result.RedirectURL}, nil
}

func (s *PaymentService) failCheckout(ctx context.Context, providerCode string, paymentID uuid.UUID, checkoutErr error) {
	s.logger.Log(ctx, config.ErrorLevel, "Payment provider failed to create checkout", map[string]any{
		"error": checkoutErr.Error(), "provider": providerCode, "paymentId": paymentID.String(),
	})
	if err := s.repo.MarkCheckoutFailed(ctx, paymentID); err != nil {
		s.logger.Log(ctx, config.ErrorLevel, "Failed to mark payment failed after checkout error", map[string]any{
			"error": err.Error(), "paymentId": paymentID.String(),
		})
	}
}

func (s *PaymentService) boostPageURL(page string, paymentID uuid.UUID) string {
	return fmt.Sprintf("%s/boost/%s?payment_id=%s", s.frontendURL, page, paymentID)
}

func parseUUID(field, value string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %s must be a UUID", ErrInvalidPaymentRequest, field)
	}
	return parsed, nil
}
