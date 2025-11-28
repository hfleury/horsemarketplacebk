package email

import (
	"context"
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
	return nil
}
