package logger

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type logrusLogFunc interface {
	Info(args ...interface{})
	Debug(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
}

type ruslog struct {
	logger *logrus.Logger
}

func NewRusLog(logger *logrus.Logger) Logger {
	return &ruslog{logger}
}

func (rl *ruslog) Info(msg string, keyvals ...interface{}) {
	logger := rl.makeFieldLogger(keyvals...)
	logger.Info(msg)
}

func (rl *ruslog) Debug(msg string, keyvals ...interface{}) {
	logger := rl.makeFieldLogger(keyvals...)
	logger.Debug(msg)
}

func (rl *ruslog) Warn(msg string, keyvals ...interface{}) {
	logger := rl.makeFieldLogger(keyvals...)
	logger.Warn(msg)
}

func (rl *ruslog) Error(msg string, keyvals ...interface{}) {
	logger := rl.makeFieldLogger(keyvals...)
	logger.Error(msg)
}

func (rl *ruslog) SetLevel(level Level) {
	switch level {
	case InfoLevel:
		rl.logger.Level = logrus.InfoLevel
	case DebugLevel:
		rl.logger.Level = logrus.DebugLevel
	case WarnLevel:
		rl.logger.Level = logrus.WarnLevel
	case ErrorLevel:
		rl.logger.Level = logrus.ErrorLevel
	default:
		rl.logger.Level = logrus.InfoLevel
	}
}

func (rl *ruslog) makeFieldLogger(keyvals ...interface{}) logrusLogFunc {
	var logger *logrus.Entry
	num := len(keyvals)
	for i := 0; i < num; i += 2 {
		key := fmt.Sprintf("%v", keyvals[i])
		var val interface{} = nil
		if num > i+1 {
			val = keyvals[i+1]
		}

		if logger == nil {
			logger = rl.logger.WithField(key, val)
		} else {
			logger = logger.WithField(key, val)
		}
	}

	if logger == nil {
		return rl.logger
	}

	return logger
}
