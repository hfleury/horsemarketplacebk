package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/payments/models"
	"github.com/hfleury/horsemarketplacebk/internal/payments/providers"
)

// HandleProviderEvent applies an authenticated provider callback. Duplicate and
// out-of-order events return nil, so the provider stops retrying them.
func (s *PaymentService) HandleProviderEvent(ctx context.Context, providerCode string, header http.Header, body []byte) error {
	provider, err := s.registry.Get(providerCode)
	if err != nil {
		return err
	}
	event, err := provider.ParseEvent(ctx, header, body)
	if err != nil {
		return err
	}
	if event.Outcome == providers.OutcomeIgnored {
		return nil
	}

	providerID, err := s.findProviderID(ctx, providerCode)
	if err != nil {
		return err
	}
	switch event.Outcome {
	case providers.OutcomePaid:
		return s.applyPaid(ctx, providerCode, providerID, event.Reference)
	case providers.OutcomeFailed:
		return s.applyTransition(ctx, providerID, event.Reference, models.StatusPending, models.StatusFailed)
	case providers.OutcomeRefunded:
		return s.applyTransition(ctx, providerID, event.Reference, models.StatusPaid, models.StatusRefunded)
	default:
		return fmt.Errorf("%w: unsupported outcome %q", providers.ErrMalformedEvent, event.Outcome)
	}
}

func (s *PaymentService) applyPaid(ctx context.Context, providerCode string, providerID uuid.UUID, reference string) error {
	confirmed, err := s.repo.ConfirmPaid(ctx, providerID, reference)
	if err != nil {
		return err
	}
	if confirmed == nil {
		return s.requireKnownPayment(ctx, providerID, reference)
	}

	s.trackBoostPaid(ctx, providerCode, confirmed)
	s.notifyBoostConfirmed(ctx, confirmed)
	return nil
}

func (s *PaymentService) applyTransition(ctx context.Context, providerID uuid.UUID, reference string, from, to models.PaymentStatus) error {
	transitioned, err := s.repo.TransitionStatus(ctx, providerID, reference, from, to)
	if err != nil {
		return err
	}
	if transitioned {
		return nil
	}
	return s.requireKnownPayment(ctx, providerID, reference)
}

// requireKnownPayment tells an event that changed nothing because it was
// already applied (nil) from one whose reference we don't know yet, which may
// arrive before the reference is saved and must be retried.
func (s *PaymentService) requireKnownPayment(ctx context.Context, providerID uuid.UUID, reference string) error {
	exists, err := s.repo.PaymentExists(ctx, providerID, reference)
	if err != nil {
		return err
	}
	if !exists {
		return ErrPaymentNotFound
	}
	return nil
}
