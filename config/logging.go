//go:generate mockgen -source=config/logging.go -destination=config/mock_zerolog_service.go -package=config
package config

import "context"

type Logging interface {
	Log(ctx context.Context, level LogLevel, msg string, fields map[string]any)
	WithTrace(ctx context.Context, traceID string) context.Context
	GetLoggerFromContext(c context.Context) Logging
}
