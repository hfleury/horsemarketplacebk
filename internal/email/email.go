package email

import (
	"context"
)

// Sender is the interface that wraps the basic Send method.
// Implementations should be lightweight so swapping providers is trivial.
type Sender interface {
	// Send sends an email to a single recipient. Body may contain HTML.
	Send(ctx context.Context, to string, subject string, body string) error
}
