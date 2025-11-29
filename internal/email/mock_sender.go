package email

import (
	"context"
	"log"
)

// MockSender is a simple in-memory sender useful for tests and local development.
type MockSender struct {
	LastTo      string
	LastSubject string
	LastBody    string
}

func NewMockSender() *MockSender {
	return &MockSender{}
}

func (m *MockSender) Send(ctx context.Context, to, subject, body string) error {
	m.LastTo = to
	m.LastSubject = subject
	m.LastBody = body
	log.Printf("[MockSender] captured email to=%s subject=%s", to, subject)
	return nil
}
