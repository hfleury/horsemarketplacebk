package email

import (
	"context"
	"time"

	mailgun "github.com/mailgun/mailgun-go/v4"
)

// MailgunSender sends email using the Mailgun HTTP API.
type MailgunSender struct {
	Domain  string
	APIKey  string
	From    string
	Timeout time.Duration
}

// NewMailgunSender constructs a MailgunSender. Pass a sensible timeout (e.g., 10*time.Second) or 0 for default.
func NewMailgunSender(domain, apiKey, from string, timeout time.Duration) *MailgunSender {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &MailgunSender{Domain: domain, APIKey: apiKey, From: from, Timeout: timeout}
}

// Send implements the Sender interface using Mailgun's API.
func (m *MailgunSender) Send(ctx context.Context, to, subject, body string) error {
	mg := mailgun.NewMailgun(m.Domain, m.APIKey)
	msg := mg.NewMessage(m.From, subject, body, to)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, m.Timeout)
	defer cancel()

	_, _, err := mg.Send(ctxWithTimeout, msg)
	return err
}
