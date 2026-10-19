package mockemail

import (
	"context"
	"log"
	"sync"
)

type SentEmail struct {
	To      string
	Subject string
	Body    string
}

// MockSender is a simple in-memory sender useful for tests and local development.
type MockSender struct {
	LastTo      string
	LastSubject string
	LastBody    string
	Sent        []SentEmail
	// ErrFor makes Send fail, without recording, for the listed recipients.
	ErrFor map[string]error

	mutex sync.Mutex
}

func NewMockSender() *MockSender {
	return &MockSender{}
}

func (m *MockSender) Send(ctx context.Context, to, subject, body string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if err, failing := m.ErrFor[to]; failing {
		return err
	}
	m.LastTo = to
	m.LastSubject = subject
	m.LastBody = body
	m.Sent = append(m.Sent, SentEmail{To: to, Subject: subject, Body: body})
	log.Printf("[MockSender] captured email to=%s subject=%s", to, subject)
	return nil
}
