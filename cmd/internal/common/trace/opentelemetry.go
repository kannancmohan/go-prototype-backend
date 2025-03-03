package trace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
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

func NewOTelTracerProvider(cfg OpenTelemetryConfig) (*trace.TracerProvider, error) {
	if cfg.Host == "" || cfg.Port == 0 || cfg.ServiceName == "" {
		return nil, errors.New("invalid OpenTelemetry configuration: Host, Port, and ServiceName are required")
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
			otlptracegrpc.WithInsecure(),
		)
	case OTelConnTypeHTTP:
		exporter, err = otlptracehttp.New(
			ctx,
			otlptracehttp.WithEndpoint(fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
			otlptracehttp.WithInsecure(),
		)
	default:
		return nil, fmt.Errorf("unsupported connection type: %s", cfg.ConnType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
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
		return nil, fmt.Errorf("failed to create OTEL resource: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	//otel.SetTracerProvider(tp)// Set the global TracerProvider

	return tp, nil
}
