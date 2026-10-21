package providers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/payments/providers"
	"github.com/stretchr/testify/assert"
)

const testSecret = "sandbox-secret"

func sandboxHeader(secret string) http.Header {
	header := http.Header{}
	if secret != "" {
		header.Set(providers.SandboxSecretHeader, secret)
	}
	return header
}

func TestSandboxProvider_Code(t *testing.T) {
	assert.Equal(t, "sandbox", providers.NewSandboxProvider(testSecret).Code())
}

func TestSandboxProvider_CreateCheckout(t *testing.T) {
	paymentID := uuid.New()
	session := providers.CheckoutSession{
		PaymentID:  paymentID,
		AmountOre:  9900,
		SuccessURL: "https://example.se/boost/success?payment_id=" + paymentID.String(),
		CancelURL:  "https://example.se/boost/cancel?payment_id=" + paymentID.String(),
	}

	result, err := providers.NewSandboxProvider(testSecret).CreateCheckout(context.Background(), session)

	assert.NoError(t, err)
	assert.Equal(t, "sbx_"+paymentID.String(), result.Reference)
	assert.Equal(t, session.SuccessURL, result.RedirectURL)
}

func TestSandboxProvider_ParseEvent(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		body      string
		wantEvent providers.Event
		wantErr   error
	}{
		{name: "paid", secret: testSecret, body: `{"reference":"sbx_1","outcome":"paid"}`, wantEvent: providers.Event{Reference: "sbx_1", Outcome: providers.OutcomePaid}},
		{name: "failed", secret: testSecret, body: `{"reference":"sbx_1","outcome":"failed"}`, wantEvent: providers.Event{Reference: "sbx_1", Outcome: providers.OutcomeFailed}},
		{name: "refunded", secret: testSecret, body: `{"reference":"sbx_1","outcome":"refunded"}`, wantEvent: providers.Event{Reference: "sbx_1", Outcome: providers.OutcomeRefunded}},
		{name: "missing secret", secret: "", body: `{"reference":"sbx_1","outcome":"paid"}`, wantErr: providers.ErrInvalidSignature},
		{name: "wrong secret", secret: "nope", body: `{"reference":"sbx_1","outcome":"paid"}`, wantErr: providers.ErrInvalidSignature},
		{name: "same length, wrong secret", secret: "sandbox-secreT", body: `{"reference":"sbx_1","outcome":"paid"}`, wantErr: providers.ErrInvalidSignature},
		{name: "secret checked before body", secret: "nope", body: `not json`, wantErr: providers.ErrInvalidSignature},
		{name: "invalid JSON", secret: testSecret, body: `not json`, wantErr: providers.ErrMalformedEvent},
		{name: "empty reference", secret: testSecret, body: `{"reference":"","outcome":"paid"}`, wantErr: providers.ErrMalformedEvent},
		{name: "unknown outcome", secret: testSecret, body: `{"reference":"sbx_1","outcome":"chargeback"}`, wantErr: providers.ErrMalformedEvent},
		{name: "ignored is not a sandbox outcome", secret: testSecret, body: `{"reference":"sbx_1","outcome":"ignored"}`, wantErr: providers.ErrMalformedEvent},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := providers.NewSandboxProvider(testSecret).ParseEvent(context.Background(), sandboxHeader(tt.secret), []byte(tt.body))
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantEvent, event)
		})
	}
}

func TestSandboxProvider_EmptySecretRejectsEverything(t *testing.T) {
	_, err := providers.NewSandboxProvider("").ParseEvent(context.Background(), http.Header{}, []byte(`{"reference":"sbx_1","outcome":"paid"}`))
	assert.ErrorIs(t, err, providers.ErrInvalidSignature)
}
