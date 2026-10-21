package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/posthog/posthog-go"
)

// shutdownTimeout bounds how long Close waits to flush queued events, so it fits
// inside the container's stop grace period.
const shutdownTimeout = 5 * time.Second

type enqueuer interface {
	Enqueue(posthog.Message) error
	Close() error
}

// PostHogClient adapts the PostHog SDK to Client.
type PostHogClient struct {
	client enqueuer
	logger config.Logging
}

// New returns a PostHog-backed Client, or a NoopClient when no API key is configured.
func New(cfg config.PostHogConfig, logger config.Logging) (Client, error) {
	if cfg.APIKey == "" {
		logger.Log(context.Background(), config.InfoLevel, "Product analytics disabled: POSTHOG_API_KEY is empty", nil)
		return NoopClient{}, nil
	}

	client, err := posthog.NewWithConfig(cfg.APIKey, posthog.Config{
		Endpoint:        cfg.Host,
		ShutdownTimeout: shutdownTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("creating PostHog client: %w", err)
	}
	return &PostHogClient{client: client, logger: logger}, nil
}

func (c *PostHogClient) Track(ctx context.Context, event Event) {
	if event.DistinctID == "" {
		return
	}

	properties := posthog.NewProperties()
	for name, value := range event.Properties {
		properties.Set(name, value)
	}
	// Events are sent from our server, so its IP says nothing about the user.
	properties.Set("$geoip_disable", true)

	err := c.client.Enqueue(posthog.Capture{
		DistinctId: event.DistinctID,
		Event:      event.Name,
		Properties: properties,
	})
	if err != nil {
		c.logger.Log(ctx, config.WarnLevel, "Failed to enqueue analytics event", map[string]any{
			"event": event.Name,
			"error": err.Error(),
		})
	}
}

func (c *PostHogClient) Close() error {
	return c.client.Close()
}
