package logger

type noop struct{}

func NewNoopLogger() Logger {
	return noop{}
}

func (n noop) Info(msg string, keyvals ...interface{}) {
	return
}

func (n noop) Debug(msg string, keyvals ...interface{}) {
	return
}

func (n noop) Warn(msg string, keyvals ...interface{}) {
	return
}

func (n noop) Error(msg string, keyvals ...interface{}) {
	return
}

func (n noop) SetLevel(level Level) {
	return
}
