// Package providers holds the payment provider abstraction. Each provider
// (sandbox, and later Stripe, Swish and so on) implements Provider, so the
// payment service never knows any provider's API or event names.
package providers

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

var (
	ErrUnknownProvider  = errors.New("unknown payment provider")
	ErrInvalidSignature = errors.New("invalid provider event signature")
	ErrMalformedEvent   = errors.New("malformed provider event")
)

type Provider interface {
	Code() string
	CreateCheckout(ctx context.Context, session CheckoutSession) (CheckoutResult, error)
	// ParseEvent authenticates and decodes a provider callback in one step, so
	// an Event is never returned for a request that failed verification.
	ParseEvent(ctx context.Context, header http.Header, body []byte) (Event, error)
}

type CheckoutSession struct {
	PaymentID   uuid.UUID
	AmountOre   int64
	Description string
	SuccessURL  string
	CancelURL   string
}

type CheckoutResult struct {
	// Reference is the provider's own ID for this payment; events carry it.
	Reference   string
	RedirectURL string
}

type Outcome string

const (
	OutcomePaid     Outcome = "paid"
	OutcomeFailed   Outcome = "failed"
	OutcomeRefunded Outcome = "refunded"
	// OutcomeIgnored acknowledges an event type we don't act on, so the
	// provider stops retrying it.
	OutcomeIgnored Outcome = "ignored"
)

type Event struct {
	Reference string
	Outcome   Outcome
}
