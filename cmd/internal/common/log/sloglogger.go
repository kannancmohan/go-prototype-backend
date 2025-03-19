package log

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/kannancmohan/go-prototype-backend/internal/common/log"
	"go.opentelemetry.io/otel/trace"
)

type Level string

const (
	DEBUG Level = "debug"
	INFO  Level = "info"
	WARN  Level = "warn"
	ERROR Level = "error"
)

var _ log.Logger = slogLogger{}

type slogLogger struct {
	logger *slog.Logger
	ctx    context.Context
}

func NewSimpleSlogLogger(logLevel Level, writer io.Writer, customHandlers ...func(slog.Handler) slog.Handler) log.Logger {
	if writer == nil {
		writer = os.Stdout
	}
	var handler slog.Handler
	if logLevel == "" {
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{})
	} else {
		var level slog.Level
		_ = level.UnmarshalText([]byte(logLevel)) //TODO handler error
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
	}
	// Apply custom handlers in sequence
	for _, customHandler := range customHandlers {
		handler = customHandler(handler)
	}

	return &slogLogger{logger: slog.New(handler)}
}

func (s slogLogger) Debug(msg string, args ...any) {
	s.logger.DebugContext(s.ctx, msg, args...)
}

func (s slogLogger) Info(msg string, args ...any) {
	s.logger.InfoContext(s.ctx, msg, args...)
}

func (s slogLogger) Warn(msg string, args ...any) {
	s.logger.WarnContext(s.ctx, msg, args...)
}

func (s slogLogger) Error(msg string, args ...any) {
	s.logger.ErrorContext(s.ctx, msg, args...)
}

// With returns a new *SLogLogger with the provided key/value pairs attached
func (s slogLogger) With(args ...any) log.Logger {
	return &slogLogger{
		logger: s.logger.With(args...),
		ctx:    s.ctx,
	}
}

// WithContext returns an *SLogLogger which still points to the same underlying *slog.Logger,
// but has the provided context attached for Debug, Info, Warn, and Error calls.
func (s slogLogger) WithContext(ctx context.Context) log.Logger {
	return &slogLogger{
		logger: s.logger,
		ctx:    ctx,
	}
}

// custom slog handler to automatically add traceID to log
type traceIDHandler struct {
	traceIDKey  string
	nextHandler slog.Handler
}

func NewTraceIDHandler(nextHandler slog.Handler) slog.Handler {
	return &traceIDHandler{traceIDKey: "traceID", nextHandler: nextHandler}
}

func (h traceIDHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.nextHandler.Enabled(ctx, level)
}

func (h traceIDHandler) Handle(ctx context.Context, record slog.Record) error {
	// Extract traceId from the OpenTelemetry context
	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.HasTraceID() {
		record.Add(h.traceIDKey, slog.StringValue(spanContext.TraceID().String()))
	}
	return h.nextHandler.Handle(ctx, record)
}

func (h traceIDHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewTraceIDHandler(h.nextHandler.WithAttrs(attrs))
}

func (h traceIDHandler) WithGroup(name string) slog.Handler {
	return NewTraceIDHandler(h.nextHandler.WithGroup(name))
}

// custom handler to add custom attribute (e.g., requestID) to log.
type CustomAttrHandler struct {
	nextHandler slog.Handler
	attrKey     string
	ctxKey      any
}

func NewCustomAttrHandler(handler slog.Handler, attrKey string, ctxKey any) slog.Handler {
	return &CustomAttrHandler{
		nextHandler: handler,
		attrKey:     attrKey,
		ctxKey:      ctxKey,
	}
}

func (h CustomAttrHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.nextHandler.Enabled(ctx, level)
}

func (h CustomAttrHandler) Handle(ctx context.Context, record slog.Record) error {
	// Extract the custom value from the context
	if value, ok := ctx.Value(h.ctxKey).(string); ok {
		record.AddAttrs(slog.Any(h.attrKey, value))
	}
	return h.nextHandler.Handle(ctx, record)
}

func (h CustomAttrHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewCustomAttrHandler(h.nextHandler.WithAttrs(attrs), h.attrKey, h.ctxKey)
}

func (h CustomAttrHandler) WithGroup(name string) slog.Handler {
	return NewCustomAttrHandler(h.nextHandler.WithGroup(name), h.attrKey, h.ctxKey)
}
