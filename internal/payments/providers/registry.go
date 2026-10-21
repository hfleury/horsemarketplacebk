package providers

import (
	"fmt"
	"sort"
)

// Registry looks providers up by code. Adding a provider means passing one
// more implementation to NewRegistry; nothing else changes.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry panics on a duplicate code: that is a wiring mistake, and it
// must stop the app at startup rather than route payments to the wrong provider.
func NewRegistry(providers ...Provider) *Registry {
	byCode := make(map[string]Provider, len(providers))
	for _, provider := range providers {
		code := provider.Code()
		if _, taken := byCode[code]; taken {
			panic(fmt.Sprintf("payment provider %q registered twice", code))
		}
		byCode[code] = provider
	}
	return &Registry{providers: byCode}
}

func (r *Registry) Get(code string) (Provider, error) {
	provider, ok := r.providers[code]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, code)
	}
	return provider, nil
}

// Codes returns the registered codes sorted, so API output is stable.
func (r *Registry) Codes() []string {
	codes := make([]string, 0, len(r.providers))
	for code := range r.providers {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}
