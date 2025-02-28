package log

import (
	"context"
	"log/slog"
	"os"
)

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	With(args ...any) Logger
	WithContext(context.Context) Logger
}

var _ Logger = &slogLogger{}

// Implement the interface for slog
type slogLogger struct {
	logger *slog.Logger
	ctx    context.Context
}

func NewSimpleSlogLogger(logLevel slog.Leveler) Logger {
	opts := &slog.HandlerOptions{}
	if logLevel != nil {
		opts.Level = logLevel
	}
	return NewSlogLogger(slog.New(slog.NewJSONHandler(os.Stdout, opts)))
}

func NewSlogLogger(logger *slog.Logger) Logger {
	return &slogLogger{logger: logger}
}

func (s *slogLogger) Debug(msg string, args ...any) {
	s.logger.DebugContext(s.ctx, msg, args...)
}

func (s *slogLogger) Info(msg string, args ...any) {
	s.logger.InfoContext(s.ctx, msg, args...)
}

func (s *slogLogger) Warn(msg string, args ...any) {
	s.logger.WarnContext(s.ctx, msg, args...)
}

func (s *slogLogger) Error(msg string, args ...any) {
	s.logger.ErrorContext(s.ctx, msg, args...)
}

// With returns a new *SLogLogger with the provided key/value pairs attached
func (s *slogLogger) With(args ...any) Logger {
	return &slogLogger{
		logger: s.logger.With(args...),
		ctx:    s.ctx,
	}
}

// WithContext returns an *SLogLogger which still points to the same underlying *slog.Logger,
// but has the provided context attached for Debug, Info, Warn, and Error calls.
func (s *slogLogger) WithContext(ctx context.Context) Logger {
	return &slogLogger{
		logger: s.logger,
		ctx:    ctx,
	}
}

// NoOpLogger. an implementation of Logger which does nothing
type NoOpLogger struct{}

var _ Logger = &NoOpLogger{}

func (*NoOpLogger) Debug(string, ...any) {}
func (*NoOpLogger) Info(string, ...any)  {}
func (*NoOpLogger) Warn(string, ...any)  {}
func (*NoOpLogger) Error(string, ...any) {}
func (n *NoOpLogger) With(...any) Logger {
	return n
}
func (n *NoOpLogger) WithContext(context.Context) Logger {
	return n
}
