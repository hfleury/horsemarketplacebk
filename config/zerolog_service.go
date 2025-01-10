package config

import (
	"context"
	"os"
	"runtime"

	"github.com/rs/zerolog"
)

type ZerologService struct {
	Logger *zerolog.Logger
}

func NewZerologService() *ZerologService {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	return &ZerologService{Logger: &logger}
}

func (zs *ZerologService) Log(ctx context.Context, level LogLevel, msg string, fields map[string]any) {
	traceID, _ := ctx.Value("traceID").(string)
	function, line := zs.getCallerInfo()

	event := zs.Logger.WithLevel(level.ToZerologLevel()).Str("trace_id", traceID).Str("function", function).Int("line", line)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(msg)
}

func (zs *ZerologService) WithTrace(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, "traceID", traceID)
}

func (zs *ZerologService) getCallerInfo() (function string, line int) {
	pc, _, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown", 0
	}
	fn := runtime.FuncForPC(pc)
	return fn.Name(), line
}
