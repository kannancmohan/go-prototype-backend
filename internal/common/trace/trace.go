package trace

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type SpanAttKeyValue map[string]string

func (s SpanAttKeyValue) getKeyValue() (string, string) {
	for k, v := range s {
		return k, v
	}
	return "", ""
}

func NewOTELSpan(ctx context.Context, tracer trace.Tracer, spanName string, spanAttributes ...SpanAttKeyValue) trace.Span {
	if tracer == nil {
		_, span := noop.NewTracerProvider().Tracer("").Start(ctx, spanName)
		return span // Return a no-op span
	}
	_, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
	for _, attributes := range spanAttributes {
		k, v := attributes.getKeyValue()
		span.SetAttributes(attribute.String(k, v))
	}
	return span
}
