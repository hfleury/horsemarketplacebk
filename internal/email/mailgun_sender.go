package email

import (
	"context"
	"log"
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

	log.Printf("[MailgunSender] sending email to=%s domain=%s from=%s", to, m.Domain, m.From)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, m.Timeout)
	defer cancel()

	_, id, err := mg.Send(ctxWithTimeout, msg)
	if err != nil {
		log.Printf("[MailgunSender] send error: %v", err)
		return err
	}
	log.Printf("[MailgunSender] sent email id=%s to=%s", id, to)
	return nil
}
