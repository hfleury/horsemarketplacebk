package config

import "context"

type Logging interface {
	Log(ctx context.Context, level LogLevel, msg string, fields map[string]any)
	WithTrace(ctx context.Context, traceID string) context.Context
}
