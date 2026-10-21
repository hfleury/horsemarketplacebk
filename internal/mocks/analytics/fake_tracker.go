// Package mockanalytics provides a recording analytics.Tracker for tests.
package mockanalytics

import (
	"context"
	"sync"

	"github.com/hfleury/horsemarketplacebk/internal/analytics"
)

type FakeTracker struct {
	mu     sync.Mutex
	events []analytics.Event
}

func NewFakeTracker() *FakeTracker {
	return &FakeTracker{}
}

func (f *FakeTracker) Track(_ context.Context, event analytics.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

// Events returns a copy of every event tracked so far, in order.
func (f *FakeTracker) Events() []analytics.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]analytics.Event(nil), f.events...)
}
