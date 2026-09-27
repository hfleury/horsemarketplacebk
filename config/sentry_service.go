package config

import (
	"github.com/getsentry/sentry-go"
)

// InitSentry configures the Sentry SDK. Called once at startup; a no-op when
// dsn is empty (e.g. local development), matching how MAILGUN_* config works.
func InitSentry(dsn, environment string) error {
	if dsn == "" {
		return nil
	}

	return sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      environment,
		AttachStacktrace: true,
	})
}

// reportToSentry sends a log event's message and fields to Sentry. Only
// called for Panic/Fatal/Error levels — see ZerologService.Log.
func reportToSentry(traceID, function string, line int, msg string, fields map[string]any) {
	if sentry.CurrentHub().Client() == nil {
		return
	}

	context := make(sentry.Context, len(fields)+1)
	context["line"] = line
	for k, v := range fields {
		context[k] = v
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("trace_id", traceID)
		scope.SetTag("function", function)
		scope.SetContext("log_fields", context)
		sentry.CaptureMessage(msg)
	})
}
