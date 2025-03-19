package trace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type OpenTelemetryConfig struct {
	Host        string
	Port        int
	ConnType    OTelConnType
	ServiceName string
}

type OTelConnType string

const (
	OTelConnTypeGRPC = OTelConnType("grpc")
	OTelConnTypeHTTP = OTelConnType("http")
)

type OTelTracerProviderShutdown func(context.Context) error

func NewOTelTracerProvider(cfg OpenTelemetryConfig) (*trace.TracerProvider, OTelTracerProviderShutdown, error) {
	if cfg.Host == "" || cfg.Port == 0 || cfg.ServiceName == "" {
		return nil, nil, errors.New("invalid OpenTelemetry configuration: Host, Port, and ServiceName are required")
	}

	var exporter *otlptrace.Exporter
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	switch cfg.ConnType {
	case OTelConnTypeGRPC:
		exporter, err = otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
			otlptracegrpc.WithInsecure(), // TODO check this
		)
	case OTelConnTypeHTTP:
		exporter, err = otlptracehttp.New(
			ctx,
			otlptracehttp.WithEndpoint(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
			otlptracehttp.WithInsecure(), // TODO check this
		)
	default:
		return nil, nil, fmt.Errorf("unsupported connection type: %s", cfg.ConnType)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
		resource.WithProcessRuntimeDescription(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTEL resource: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	// Set TracerProvider globally. This allows creation of tracers and spans in any part of app without the need of passing trace.TracerProvider
	// Also otelhttp.NewHandler could automatically extract/create trace context from incoming request without explicitly configuring trace.TracerProvider in handler
	// otel.SetTracerProvider(tp) //commented since tp is set via otelhttp.WithTracerProvider(tp) when configuring otelhttp.NewHandler

	// Sets the global propagator to W3C TraceContext format
	// 'otel.SetTextMapPropagator' defines how tracing context (trace ID, span ID, etc.) is injected and extracted from HTTP headers
	// you need to use otelhttp.NewTransport to inject tracing context to outgoing http calls and otelhttp.NewHandler for extract context from incoming http request
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, func(ctx context.Context) error { return tp.Shutdown(ctx) }, nil
}
