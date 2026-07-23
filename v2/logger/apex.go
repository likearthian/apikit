package logger

import (
	"fmt"

	"github.com/apex/log"
)

type apexLogFunc interface {
	Info(msg string)
	Debug(msg string)
	Warn(msg string)
	Error(msg string)
}

type apexLogger struct {
	logger *log.Logger
}

func NewApexLogger(logger *log.Logger) Logger {
	return &apexLogger{logger: logger}
}

func (a *apexLogger) Info(msg string, keyvals ...interface{}) {
	logger := a.makeFieldLogger(keyvals...)
	logger.Info(msg)
}

func (a *apexLogger) Debug(msg string, keyvals ...interface{}) {
	logger := a.makeFieldLogger(keyvals...)
	logger.Debug(msg)
}

func (a *apexLogger) Warn(msg string, keyvals ...interface{}) {
	logger := a.makeFieldLogger(keyvals...)
	logger.Warn(msg)
}

func (a *apexLogger) Error(msg string, keyvals ...interface{}) {
	logger := a.makeFieldLogger(keyvals...)
	logger.Error(msg)
}

func (a *apexLogger) SetLevel(level Level) {
	switch level {
	case DebugLevel:
		a.logger.Level = log.DebugLevel
	case InfoLevel:
		a.logger.Level = log.InfoLevel
	case WarnLevel:
		a.logger.Level = log.WarnLevel
	case ErrorLevel:
		a.logger.Level = log.ErrorLevel
	default:
		a.logger.Level = log.InfoLevel
	}
}

func (a *apexLogger) makeFieldLogger(keyvals ...interface{}) apexLogFunc {
	var logger *log.Entry
	num := len(keyvals)
	for i := 0; i < num; i += 2 {
		key := fmt.Sprintf("%v", keyvals[i])
		var val interface{} = nil
		if num > i+1 {
			val = keyvals[i+1]
		}

		if logger == nil {
			logger = a.logger.WithField(key, val)
		} else {
			logger = logger.WithField(key, val)
		}
	}

	if logger == nil {
		return a.logger
	}

	return logger
}
