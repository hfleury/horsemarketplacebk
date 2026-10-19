package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hfleury/horsemarketplacebk/internal/tasks"
	"github.com/hibiken/asynq"
)

var errMissingProductID = errors.New("missing product_id")

type savedSearchAlertSender interface {
	SendMatchAlerts(ctx context.Context, productID string) error
}

// SavedSearchAlertHandler consumes savedsearch:match tasks.
type SavedSearchAlertHandler struct {
	alerts savedSearchAlertSender
}

func NewSavedSearchAlertHandler(alerts savedSearchAlertSender) *SavedSearchAlertHandler {
	return &SavedSearchAlertHandler{alerts: alerts}
}

func (h *SavedSearchAlertHandler) HandleMatchSavedSearchesTask(ctx context.Context, t *asynq.Task) error {
	var payload tasks.MatchSavedSearchesPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("decode saved search match payload: %v: %w", err, asynq.SkipRetry)
	}
	if payload.ProductID == "" {
		return fmt.Errorf("decode saved search match payload: %w: %w", errMissingProductID, asynq.SkipRetry)
	}

	if err := h.alerts.SendMatchAlerts(ctx, payload.ProductID); err != nil {
		return fmt.Errorf("send saved search alerts for listing %s: %w", payload.ProductID, err)
	}
	return nil
}
