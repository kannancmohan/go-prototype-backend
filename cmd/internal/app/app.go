package app

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

type App interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

// MetricsProvider apps that needs to export metrics to prometheus should implement this interface
type MetricsProvider interface {
	PrometheusCollectors() []prometheus.Collector
}
