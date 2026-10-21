package providers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hfleury/horsemarketplacebk/internal/payments/providers"
	"github.com/stretchr/testify/assert"
)

type namedProvider struct{ code string }

func (p namedProvider) Code() string { return p.code }

func (p namedProvider) CreateCheckout(context.Context, providers.CheckoutSession) (providers.CheckoutResult, error) {
	return providers.CheckoutResult{}, nil
}

func (p namedProvider) ParseEvent(context.Context, http.Header, []byte) (providers.Event, error) {
	return providers.Event{}, nil
}

func TestRegistry_Get(t *testing.T) {
	stripe := namedProvider{code: "stripe"}
	registry := providers.NewRegistry(stripe, namedProvider{code: "swish"})

	tests := []struct {
		name    string
		code    string
		want    providers.Provider
		wantErr error
	}{
		{name: "known code", code: "stripe", want: stripe},
		{name: "unknown code", code: "klarna", wantErr: providers.ErrUnknownProvider},
		{name: "empty code", code: "", wantErr: providers.ErrUnknownProvider},
		{name: "codes are case-sensitive", code: "Stripe", wantErr: providers.ErrUnknownProvider},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := registry.Get(tt.code)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.want, provider)
		})
	}
}

func TestRegistry_Codes(t *testing.T) {
	tests := []struct {
		name      string
		providers []providers.Provider
		want      []string
	}{
		{name: "empty registry", want: []string{}},
		{name: "one provider", providers: []providers.Provider{namedProvider{code: "sandbox"}}, want: []string{"sandbox"}},
		{
			name:      "sorted regardless of registration order",
			providers: []providers.Provider{namedProvider{code: "swish"}, namedProvider{code: "sandbox"}, namedProvider{code: "klarna"}},
			want:      []string{"klarna", "sandbox", "swish"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, providers.NewRegistry(tt.providers...).Codes())
		})
	}
}

func TestNewRegistry_DuplicateCodePanics(t *testing.T) {
	assert.PanicsWithValue(t, `payment provider "sandbox" registered twice`, func() {
		providers.NewRegistry(namedProvider{code: "sandbox"}, namedProvider{code: "sandbox"})
	})
}
