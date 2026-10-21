// Package analytics sends product-analytics events. Domain services depend only
// on Tracker, so the analytics vendor stays behind this package.
package analytics

import (
	"context"
	"io"
)

const (
	EventUserSignedUp        = "user_signed_up"
	EventEmailVerified       = "email_verified"
	EventUserLoggedIn        = "user_logged_in"
	EventListingCreated      = "listing_created"
	EventListingPublished    = "listing_published"
	EventConversationStarted = "conversation_started"
	EventMessageSent         = "message_sent"
	EventCallbackRequested   = "callback_requested"
	EventFavoriteAdded       = "favorite_added"
	EventSavedSearchCreated  = "saved_search_created"
	EventListingReported     = "listing_reported"
	EventBoostPaid           = "boost_paid"
)

// Event is one business action performed by the user identified by DistinctID.
// Properties must never hold message text, contact details or other free text.
type Event struct {
	Name       string
	DistinctID string
	Properties map[string]any
}

// Tracker records events on a best-effort basis: it never returns an error,
// because the action being tracked has already succeeded.
type Tracker interface {
	Track(ctx context.Context, event Event)
}

// Client is a Tracker whose buffered events are flushed on Close.
type Client interface {
	Tracker
	io.Closer
}

// NoopClient is used when analytics is not configured.
type NoopClient struct{}

func (NoopClient) Track(context.Context, Event) {}

func (NoopClient) Close() error { return nil }
