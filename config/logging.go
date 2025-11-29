//go:generate mockgen -source=logging.go -destination=internal/mocks/config/mock_zerolog_service.go -package=mockconfig
package config

import "context"

type Logging interface {
	Log(ctx context.Context, level LogLevel, msg string, fields map[string]any)
	WithTrace(ctx context.Context, traceID string) context.Context
	GetLoggerFromContext(c context.Context) Logging
}
