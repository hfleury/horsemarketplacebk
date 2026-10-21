package analytics

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hfleury/horsemarketplacebk/config"
	mockconfig "github.com/hfleury/horsemarketplacebk/internal/mocks/config"
	"github.com/posthog/posthog-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeEnqueuer struct {
	messages   []posthog.Message
	enqueueErr error
	closed     bool
}

func (f *fakeEnqueuer) Enqueue(message posthog.Message) error {
	f.messages = append(f.messages, message)
	return f.enqueueErr
}

func (f *fakeEnqueuer) Close() error {
	f.closed = true
	return nil
}

func TestPostHogClient_TrackMapsEventToCapture(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	client := &PostHogClient{client: enqueuer}
	properties := map[string]any{"listing_id": "listing-1"}

	client.Track(context.Background(), Event{Name: EventFavoriteAdded, DistinctID: "user-1", Properties: properties})

	require.Len(t, enqueuer.messages, 1)
	capture, ok := enqueuer.messages[0].(posthog.Capture)
	require.True(t, ok)
	assert.Equal(t, "user-1", capture.DistinctId)
	assert.Equal(t, EventFavoriteAdded, capture.Event)
	assert.Equal(t, posthog.Properties{"listing_id": "listing-1", "$geoip_disable": true}, capture.Properties)
	assert.Equal(t, map[string]any{"listing_id": "listing-1"}, properties, "caller's properties must not be mutated")
}

func TestPostHogClient_TrackSkipsEventWithoutDistinctID(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	client := &PostHogClient{client: enqueuer}

	client.Track(context.Background(), Event{Name: EventUserSignedUp})

	assert.Empty(t, enqueuer.messages)
}

func TestPostHogClient_TrackLogsEnqueueFailureAsWarning(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := mockconfig.NewMockLogging(ctrl)
	logger.EXPECT().Log(gomock.Any(), config.WarnLevel, "Failed to enqueue analytics event", map[string]any{
		"event": EventUserLoggedIn,
		"error": "queue full",
	})
	client := &PostHogClient{client: &fakeEnqueuer{enqueueErr: errors.New("queue full")}, logger: logger}

	client.Track(context.Background(), Event{Name: EventUserLoggedIn, DistinctID: "user-1"})
}

func TestPostHogClient_CloseClosesSDKClient(t *testing.T) {
	enqueuer := &fakeEnqueuer{}
	client := &PostHogClient{client: enqueuer}

	require.NoError(t, client.Close())

	assert.True(t, enqueuer.closed)
}

func TestNew_ReturnsNoopClientWithoutAPIKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := mockconfig.NewMockLogging(ctrl)
	logger.EXPECT().Log(gomock.Any(), config.InfoLevel, gomock.Any(), gomock.Any())

	client, err := New(config.PostHogConfig{Host: "https://eu.i.posthog.com"}, logger)

	require.NoError(t, err)
	assert.Equal(t, NoopClient{}, client)
}

func TestNew_ReturnsPostHogClientWithAPIKey(t *testing.T) {
	client, err := New(config.PostHogConfig{APIKey: "phc_test", Host: "https://eu.i.posthog.com"}, nil)

	require.NoError(t, err)
	postHogClient, ok := client.(*PostHogClient)
	require.True(t, ok)
	assert.NoError(t, postHogClient.Close())
}
