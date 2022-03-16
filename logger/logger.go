package logger

import (
	"os"

	"github.com/rs/zerolog"
)

type Level int

const (
	InvalidLevel Level = iota - 1
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Logger interface {
	Info(msg string, keyvals ...interface{})
	Debug(msg string, keyvals ...interface{})
	Warn(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
	SetLevel(level Level)
}

func NewStandardLogger() Logger {
	return NewZerolog(zerolog.New(os.Stderr).With().Timestamp().Logger())
}
