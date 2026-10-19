package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeEnqueuer struct {
	enqueued []*asynq.Task
	err      error
}

func (f *fakeEnqueuer) EnqueueContext(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.enqueued = append(f.enqueued, task)
	return &asynq.TaskInfo{}, nil
}

func TestSavedSearchMatchProducer_NotifyListingPublished_EnqueuesMatchTask(t *testing.T) {
	enqueuer := &fakeEnqueuer{}

	err := NewSavedSearchMatchProducer(enqueuer).NotifyListingPublished(context.Background(), "product-1")

	require.NoError(t, err)
	require.Len(t, enqueuer.enqueued, 1)
	assert.Equal(t, TypeMatchSavedSearches, enqueuer.enqueued[0].Type())
	var payload MatchSavedSearchesPayload
	require.NoError(t, json.Unmarshal(enqueuer.enqueued[0].Payload(), &payload))
	assert.Equal(t, "product-1", payload.ProductID)
}

func TestSavedSearchMatchProducer_NotifyListingPublished_WrapsEnqueueError(t *testing.T) {
	redisDown := errors.New("redis down")
	enqueuer := &fakeEnqueuer{err: redisDown}

	err := NewSavedSearchMatchProducer(enqueuer).NotifyListingPublished(context.Background(), "product-1")

	assert.ErrorIs(t, err, redisDown)
}
