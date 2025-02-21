package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewMetricsAddonAppRunner(cfg MetricsExporterConfig) *MetricsAddonAppRunner {
	exporter := newMetricsExporter(cfg)
	return &MetricsAddonAppRunner{
		server: exporter,
		runner: NewSingletonRunner(&addonRunnable{
			runner: exporter,
		}, false),
	}
}

var _ Runnable = &MetricsAddonAppRunner{}

type MetricsAddonAppRunner struct {
	runner *SingletonRunner
	server *MetricsExporter
}

func (m *MetricsAddonAppRunner) Run(ctx context.Context) error {
	return m.runner.Run(ctx)
}

func (m *MetricsAddonAppRunner) RegisterCollectors(collectors ...prometheus.Collector) error {
	return m.server.RegisterCollectors(collectors...)
}

type MetricsProvider interface {
	PrometheusCollectors() []prometheus.Collector
}

type MetricsExporterConfig struct {
	Registerer prometheus.Registerer
	Gatherer   prometheus.Gatherer
	Port       int
}

func newMetricsExporter(cfg MetricsExporterConfig) *MetricsExporter {
	if cfg.Registerer == nil {
		cfg.Registerer = prometheus.DefaultRegisterer
	}
	if cfg.Gatherer == nil {
		cfg.Gatherer = prometheus.DefaultGatherer
	}
	if cfg.Port <= 0 {
		cfg.Port = 9090
	}
	return &MetricsExporter{
		Registerer: cfg.Registerer,
		Gatherer:   cfg.Gatherer,
		Port:       cfg.Port,
	}
}

var _ addonApp = &MetricsExporter{}

// MetricsExporter exports prometheus metrics
type MetricsExporter struct {
	Registerer prometheus.Registerer
	Gatherer   prometheus.Gatherer
	Port       int
}

// RegisterCollectors registers the provided collectors with the Exporter's Registerer.
// If there is an error registering any of the provided collectors, registration is halted and an error is returned.
func (e *MetricsExporter) RegisterCollectors(metrics ...prometheus.Collector) error {
	for _, m := range metrics {
		err := e.Registerer.Register(m)
		if err != nil {
			return err
		}
	}
	return nil
}

// Run creates an HTTP server which exposes a /metrics endpoint on the configured port (if <=0, uses the default 9090)
func (e *MetricsExporter) Run(stopCh <-chan struct{}) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.InstrumentMetricHandler(
		e.Registerer, promhttp.HandlerFor(e.Gatherer, promhttp.HandlerOpts{}),
	))
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", e.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()
	go func() {
		for range stopCh {
			// block until receives a message
			break
		}
		ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second)
		defer cancelFunc()
		errCh <- server.Shutdown(ctx)
	}()
	err := <-errCh
	return err
}
