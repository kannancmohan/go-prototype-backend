package app

import (
	"context"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/internal/common/log"
	"github.com/prometheus/client_golang/prometheus"
)

type App interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

// MetricsProvider interface. apps that needs to expose metrics endpoint for prometheus should implement this interface
type MetricsProvider interface {
	PrometheusCollectors() []prometheus.Collector
}

// Loggable . apps that need to use the logger should implement this interface
type Loggable interface {
	SetLogger(log.Logger)
}
