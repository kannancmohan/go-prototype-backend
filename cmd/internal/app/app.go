package app

import (
	"context"
	"fmt"

	"github.com/kannancmohan/go-prototype-backend/internal/common/log"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// EmptyAppConf an empty instance of AppConf.
func EmptyAppConf() Conf[struct{}] {
	return Conf[struct{}]{}
}

// NewAppConf creates a new AppConf.
func NewAppConf[T any](appName string, envVar T) (Conf[T], error) {
	if appName == "" {
		return Conf[T]{}, fmt.Errorf("mandatory fled 'appName' missing ")
	}
	return Conf[T]{Name: appName, EnvVar: envVar}, nil
}

// Conf creates a new Conf.
type Conf[T any] struct {
	Name   string
	EnvVar T
}

// App interface exposes methods for app.
type App interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

// MetricsSetter interface. apps that needs to expose additional metrics to prometheus should implement this interface.
type MetricsSetter interface {
	PrometheusCollectors() []prometheus.Collector
}

// Loggable . apps that need to use the logger should implement this interface
// appRunner will automatically set logger to apps that implement this interface.
type Loggable interface {
	SetLogger(log.Logger)
}

// ConfigSetter . apps that need AppConf should implement this interface
// appRunner will automatically set AppConf to apps that implement this interface.
type ConfigSetter[T any] interface {
	SetAppConf(Conf[T])
}

// Traceable . apps that need tracing should implement this interface
// appRunner will automatically set TracerProvider to apps that implement this interface.
type Traceable interface {
	SetTracerProvider(trace.TracerProvider)
}
