package providers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	SandboxCode         = "sandbox"
	SandboxSecretHeader = "X-Sandbox-Secret"
	sandboxRefPrefix    = "sbx_"
)

// SandboxProvider is a test-only provider. Its checkout approves instantly by
// sending the buyer straight to the success URL; the payment is confirmed only
// when a developer posts an event to the webhook, as a real provider would.
type SandboxProvider struct {
	secret []byte
}

func NewSandboxProvider(secret string) *SandboxProvider {
	return &SandboxProvider{secret: []byte(secret)}
}

func (p *SandboxProvider) Code() string {
	return SandboxCode
}

// CreateCheckout derives the reference from the payment ID, so a developer can
// build it from the payment_id in the success URL.
func (p *SandboxProvider) CreateCheckout(_ context.Context, session CheckoutSession) (CheckoutResult, error) {
	return CheckoutResult{
		Reference:   sandboxRefPrefix + session.PaymentID.String(),
		RedirectURL: session.SuccessURL,
	}, nil
}

type sandboxEventBody struct {
	Reference string `json:"reference"`
	Outcome   string `json:"outcome"`
}

func (p *SandboxProvider) ParseEvent(_ context.Context, header http.Header, body []byte) (Event, error) {
	if !p.hasValidSecret(header) {
		return Event{}, ErrInvalidSignature
	}

	var decoded sandboxEventBody
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Event{}, fmt.Errorf("%w: %v", ErrMalformedEvent, err)
	}
	if decoded.Reference == "" {
		return Event{}, fmt.Errorf("%w: reference is required", ErrMalformedEvent)
	}
	outcome, err := parseSandboxOutcome(decoded.Outcome)
	if err != nil {
		return Event{}, err
	}
	return Event{Reference: decoded.Reference, Outcome: outcome}, nil
}

func (p *SandboxProvider) hasValidSecret(header http.Header) bool {
	received := []byte(header.Get(SandboxSecretHeader))
	return len(received) > 0 && subtle.ConstantTimeCompare(received, p.secret) == 1
}

func parseSandboxOutcome(value string) (Outcome, error) {
	switch outcome := Outcome(value); outcome {
	case OutcomePaid, OutcomeFailed, OutcomeRefunded:
		return outcome, nil
	default:
		return "", fmt.Errorf("%w: unknown outcome %q", ErrMalformedEvent, value)
	}
}
