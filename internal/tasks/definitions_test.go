package tasks_test

import (
	"encoding/json"
	"testing"

	"github.com/hfleury/horsemarketplacebk/internal/tasks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMatchSavedSearchesTask_CarriesProductID(t *testing.T) {
	task, err := tasks.NewMatchSavedSearchesTask("product-1")
	require.NoError(t, err)

	assert.Equal(t, "savedsearch:match", task.Type())
	var payload tasks.MatchSavedSearchesPayload
	require.NoError(t, json.Unmarshal(task.Payload(), &payload))
	assert.Equal(t, "product-1", payload.ProductID)
}
