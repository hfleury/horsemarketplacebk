package tasks

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

const (
	TypeProcessImage       = "image:process"
	TypeMatchSavedSearches = "savedsearch:match"
)

type ProcessImagePayload struct {
	MediaID string `json:"media_id"`
}

func NewProcessImageTask(mediaID string) (*asynq.Task, error) {
	payload, err := json.Marshal(ProcessImagePayload{MediaID: mediaID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeProcessImage, payload), nil
}

type MatchSavedSearchesPayload struct {
	ProductID string `json:"product_id"`
}

// NewMatchSavedSearchesTask needs no Unique option: alerts are claimed per
// (search, listing) pair, so retries and duplicate enqueues are harmless.
func NewMatchSavedSearchesTask(productID string) (*asynq.Task, error) {
	payload, err := json.Marshal(MatchSavedSearchesPayload{ProductID: productID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeMatchSavedSearches, payload,
		asynq.Queue("default"), asynq.MaxRetry(3), asynq.Timeout(2*time.Minute),
	), nil
}
