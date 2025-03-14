package app

import (
	"context"
	"fmt"

	"github.com/kannancmohan/go-prototype-backend/internal/common/log"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

var EmptyAppConf *AppConf[struct{}]

func NewAppConf[T any](appName string, envVar T) (*AppConf[T], error) {
	if appName == "" {
		return nil, fmt.Errorf("mandatory fled 'appName' missing ")
	}
	return &AppConf[T]{Name: appName, EnvVar: envVar}, nil
}

type AppConf[T any] struct {
	Name   string
	EnvVar T
}

type App interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

// MetricsSetter interface. apps that needs to expose additional metrics to prometheus should implement this interface
type MetricsSetter interface {
	PrometheusCollectors() []prometheus.Collector
}

// Loggable . apps that need to use the logger should implement this interface
// appRunner will automatically set logger to apps that implement this interface
type Loggable interface {
	SetLogger(log.Logger)
}

// AppConfigSetter . apps that need AppConf should implement this interface
// appRunner will automatically set AppConf to apps that implement this interface
type AppConfigSetter[T any] interface {
	SetAppConf(*AppConf[T])
}

// Traceable . apps that need tracing should implement this interface
// appRunner will automatically set TracerProvider to apps that implement this interface
type Traceable interface {
	SetTracerProvider(trace.TracerProvider)
}
