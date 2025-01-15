package config

import (
	"bytes"
	"context"
	"regexp"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestZerologService_WithTrace(t *testing.T) {
	loggerService := NewZerologService()

	traceId := "test-trace-id"
	ctx := loggerService.WithTrace(context.Background(), traceId)

	assert.Equal(t, traceId, ctx.Value("traceID"))
}

func TestZerologService_Log(t *testing.T) {
	var logBuffer bytes.Buffer

	output := zerolog.ConsoleWriter{Out: &logBuffer}
	logger := zerolog.New(output).With().Timestamp().Logger()
	loggerService := &ZerologService{Logger: &logger}

	traceID := "test-trace-id"
	ctx := context.WithValue(context.Background(), "traceID", traceID)
	msg := "Test log message"
	fields := map[string]any{"key1": "value1", "key2": "value2", "key3": 123}

	loggerService.Log(ctx, InfoLevel, msg, fields)

	//loggerService.Logger.Sync()
	cleanOutput := stripANSI(logBuffer.String())

	assert.Contains(t, cleanOutput, traceID, "Trace ID should be present in the log")
	assert.Contains(t, cleanOutput, "INF", "Log level should be present in the log")
	assert.Contains(t, cleanOutput, msg, "Message should be present in the log")
	assert.Contains(t, cleanOutput, "key1=value1", "Custom field key1 should be present in the log")
	assert.Contains(t, cleanOutput, "key2=value2", "Custom field key2 should be present in the log")
	assert.Contains(t, cleanOutput, "key3=123", "Custom field key2 should be present in the log")
}

// Helper function to remove ANSI escape codes from strings
func stripANSI(input string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(input, "")
}
