package worker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/hfleury/horsemarketplacebk/internal/tasks"
	"github.com/hfleury/horsemarketplacebk/internal/worker"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
)

type fakeAlertSender struct {
	receivedProductIDs []string
	err                error
}

func (f *fakeAlertSender) SendMatchAlerts(ctx context.Context, productID string) error {
	f.receivedProductIDs = append(f.receivedProductIDs, productID)
	return f.err
}

func TestHandleMatchSavedSearchesTask_InvalidPayload_SkipsRetry(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "invalid JSON", payload: "{not json"},
		{name: "empty product ID", payload: `{"product_id":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alerts := &fakeAlertSender{}
			task := asynq.NewTask(tasks.TypeMatchSavedSearches, []byte(tt.payload))

			err := worker.NewSavedSearchAlertHandler(alerts).HandleMatchSavedSearchesTask(context.Background(), task)

			assert.ErrorIs(t, err, asynq.SkipRetry)
			assert.Empty(t, alerts.receivedProductIDs)
		})
	}
}

func TestHandleMatchSavedSearchesTask_ServiceError_IsRetryable(t *testing.T) {
	smtpDown := errors.New("smtp down")
	alerts := &fakeAlertSender{err: smtpDown}
	task, _ := tasks.NewMatchSavedSearchesTask("product-1")

	err := worker.NewSavedSearchAlertHandler(alerts).HandleMatchSavedSearchesTask(context.Background(), task)

	assert.ErrorIs(t, err, smtpDown)
	assert.NotErrorIs(t, err, asynq.SkipRetry)
}

func TestHandleMatchSavedSearchesTask_Success_PassesProductID(t *testing.T) {
	alerts := &fakeAlertSender{}
	task, _ := tasks.NewMatchSavedSearchesTask("product-1")

	err := worker.NewSavedSearchAlertHandler(alerts).HandleMatchSavedSearchesTask(context.Background(), task)

	assert.NoError(t, err)
	assert.Equal(t, []string{"product-1"}, alerts.receivedProductIDs)
}
