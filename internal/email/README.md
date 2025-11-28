Email service
=============

This package provides a small pluggable email `Sender` interface and two
implementations useful for development and production.

Implementations included:
- `SMTPSender` — uses the standard library `net/smtp` to send via any SMTP server (Mailgun, SendGrid SMTP, Gmail SMTP, etc.).
- `MockSender` — a no-op/in-memory sender useful in tests or local runs.

Which provider to use?
- SendGrid: generous free tier for developers, excellent docs and Go SDK.
- Mailgun: developer-friendly, easy to use via SMTP or HTTP API.
- Mailtrap: great for development/testing (captures emails, doesn't actually send to users).
- Amazon SES: very cheap and scalable but requires AWS setup.
- Gmail SMTP: can be used for quick testing, but not recommended for production (rate limits, OAuth/password requirements).

Recommendation
- For production-like sending with a free tier and good deliverability, use SendGrid or Mailgun.
- For local development and CI, use Mailtrap or `MockSender`.

Usage examples

Instantiate the SMTP sender (example env vars shown):

  host=mail.smtp-provider.com
  port=587
  username=your-smtp-user
  password=your-smtp-password
  from=no-reply@example.com

  sender := email.NewSMTPSender(host, 587, username, password, from)
  _ = sender.Send(ctx, "user@example.com", "Welcome", "Hello!")

Or use the mock sender in tests:

  mock := email.NewMockSender()
  _ = mock.Send(ctx, "a@b.com", "subj", "body")
