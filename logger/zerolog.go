package logger

import (
	"github.com/rs/zerolog"
)

type zlog struct {
	logger zerolog.Logger
}

func NewZerolog(logger zerolog.Logger) Logger {
	return &zlog{logger}
}

func (z *zlog) Info(msg string, keyvals ...interface{}) {
	z.logger.Info().Fields(keyvals).Msg(msg)
}

func (z *zlog) Debug(msg string, keyvals ...interface{}) {
	z.logger.Debug().Fields(keyvals).Msg(msg)
}

func (z *zlog) Warn(msg string, keyvals ...interface{}) {
	z.logger.Warn().Fields(keyvals).Msg(msg)
}

func (z *zlog) Error(msg string, keyvals ...interface{}) {
	z.logger.Error().Fields(keyvals).Msg(msg)
}

func (z *zlog) SetLevel(level Level) {
	switch level {
	case InfoLevel:
		z.logger = z.logger.Level(zerolog.InfoLevel)
	case DebugLevel:
		z.logger = z.logger.Level(zerolog.DebugLevel)
	case WarnLevel:
		z.logger = z.logger.Level(zerolog.WarnLevel)
	case ErrorLevel:
		z.logger = z.logger.Level(zerolog.ErrorLevel)
	default:
		z.logger = z.logger.Level(zerolog.InfoLevel)
	}
}
