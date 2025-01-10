package config

import "github.com/rs/zerolog"

type LogLevel int

const (
	PanicLevel LogLevel = iota + 5
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel = -1
)

func (l LogLevel) ToZerologLevel() zerolog.Level {
	switch l {
	case PanicLevel:
		return zerolog.PanicLevel
	case FatalLevel:
		return zerolog.FatalLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case DebugLevel:
		return zerolog.DebugLevel
	case TraceLevel:
		return zerolog.TraceLevel
	default:
		return zerolog.InfoLevel
	}
}

type LoggingModel struct {
	TimeFormat *string
	Level      *string
}
