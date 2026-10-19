package tasks

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
)

// taskEnqueuer is the slice of *asynq.Client the producers need, so they
// can be tested without Redis.
type taskEnqueuer interface {
	EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// SavedSearchMatchProducer adapts asynq to the product service's
// "listing published" notification, so products code never imports asynq.
type SavedSearchMatchProducer struct {
	enqueuer taskEnqueuer
}

func NewSavedSearchMatchProducer(enqueuer taskEnqueuer) *SavedSearchMatchProducer {
	return &SavedSearchMatchProducer{enqueuer: enqueuer}
}

func (p *SavedSearchMatchProducer) NotifyListingPublished(ctx context.Context, productID string) error {
	task, err := NewMatchSavedSearchesTask(productID)
	if err != nil {
		return fmt.Errorf("build saved search match task: %w", err)
	}
	if _, err := p.enqueuer.EnqueueContext(ctx, task); err != nil {
		return fmt.Errorf("enqueue saved search match task: %w", err)
	}
	return nil
}
