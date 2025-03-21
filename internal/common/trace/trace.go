package trace

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// SpanAttribute holds the configuration for the Span attribute.
type SpanAttribute struct {
	key   string
	value string
}

// WithAttribute optional attribute for Span.
func WithAttribute(key, value string) SpanAttribute {
	return SpanAttribute{key: key, value: value}
}

// NewOTELSpan can be used in func for tracing
// It should be place as first line of function. eg usage "defer trace.NewOTELSpan(ctx,tracer,"user.create").End()".
func NewOTELSpan(ctx context.Context, tracer trace.Tracer, spanName string, spanAttributes ...SpanAttribute) trace.Span {
	if tracer == nil {
		_, span := noop.NewTracerProvider().Tracer("").Start(ctx, spanName)
		return span // Return a no-op span
	}
	_, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
	for _, v := range spanAttributes {
		span.SetAttributes(attribute.String(v.key, v.value))
	}
	return span
}
