package tasks

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const (
	TypeProcessImage = "image:process"
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
