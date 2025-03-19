package log

import (
	"context"
)

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	With(args ...any) Logger
	WithContext(context.Context) Logger
}

// NoOpLogger. an implementation of Logger which does nothing.
type NoOpLogger struct{}

var _ Logger = &NoOpLogger{}

func (NoOpLogger) Debug(string, ...any) {}
func (NoOpLogger) Info(string, ...any)  {}
func (NoOpLogger) Warn(string, ...any)  {}
func (NoOpLogger) Error(string, ...any) {}

func (n NoOpLogger) With(...any) Logger {
	return n
}

func (n NoOpLogger) WithContext(context.Context) Logger {
	return n
}
