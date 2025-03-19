package log

import (
	"context"
)

// Logger is a interface that exposes methods for writing logs.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	With(args ...any) Logger
	WithContext(context.Context) Logger
}

// NoOpLogger an implementation of Logger which does nothing.
type NoOpLogger struct{}

var _ Logger = &NoOpLogger{}

// Debug does nothing.
func (NoOpLogger) Debug(string, ...any) {}

// Info does nothing.
func (NoOpLogger) Info(string, ...any) {}

// Warn does nothing.
func (NoOpLogger) Warn(string, ...any) {}

// Error does nothing.
func (NoOpLogger) Error(string, ...any) {}

// With returns original NoOpLogger.
func (n NoOpLogger) With(...any) Logger {
	return n
}

// WithContext returns original NoOpLogger.
func (n NoOpLogger) WithContext(context.Context) Logger {
	return n
}
