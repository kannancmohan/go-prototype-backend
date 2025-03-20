package app

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/kannancmohan/go-prototype-backend/internal/common/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsServerApp creates a new MetricsServerAppConfig with the given options.
func NewMetricsServerApp(opts ...MetricsServerAppOption) *MetricsServerApp {
	config := &MetricsServerApp{
		registerer:      prometheus.DefaultRegisterer, // Default Prometheus registerer
		gatherer:        prometheus.DefaultGatherer,   // Default Prometheus gatherer
		port:            9090,                         // Default port
		path:            "/metrics",                   // Default path
		shutdownTimeout: 5 * time.Second,              // Default shutdown timeout
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

var _ App = &MetricsServerApp{}

var _ Loggable = &MetricsServerApp{}

// MetricsServerApp app to expose metrics endpoint.
type MetricsServerApp struct {
	registerer      prometheus.Registerer
	gatherer        prometheus.Gatherer
	port            int
	path            string
	shutdownTimeout time.Duration
	server          *http.Server
	log             log.Logger
	mu              sync.Mutex
}

// MetricsServerAppOption optional parameter for NewMetricsServerApp.
type MetricsServerAppOption func(*MetricsServerApp)

// WithRegisterer generates an optional MetricsServerAppOption with the given prometheus.Registerer.
func WithRegisterer(registerer prometheus.Registerer) MetricsServerAppOption {
	return func(c *MetricsServerApp) {
		c.registerer = registerer
	}
}

// WithGatherer generates an optional MetricsServerAppOption with the given prometheus.Gatherer.
func WithGatherer(gatherer prometheus.Gatherer) MetricsServerAppOption {
	return func(c *MetricsServerApp) {
		c.gatherer = gatherer
	}
}

// WithPort generates an optional MetricsServerAppOption with the given port.
func WithPort(port int) MetricsServerAppOption {
	return func(c *MetricsServerApp) {
		c.port = port
	}
}

// WithPath generates an optional MetricsServerAppOption with the given path.
func WithPath(path string) MetricsServerAppOption {
	return func(c *MetricsServerApp) {
		c.path = path
	}
}

// WithShutdownTimeout generates an optional MetricsServerAppOption with the given timeout.
func WithShutdownTimeout(timeout time.Duration) MetricsServerAppOption {
	return func(c *MetricsServerApp) {
		c.shutdownTimeout = timeout
	}
}

// RegisterCollectors custom function to register app specific metrics collector.
func (e *MetricsServerApp) RegisterCollectors(metrics ...prometheus.Collector) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, m := range metrics {
		if err := e.registerer.Register(m); err != nil {
			return fmt.Errorf("failed to register collector: %w", err)
		}
	}
	return nil
}

// Run method to start the MetricsServerApp.
func (e *MetricsServerApp) Run(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	mux := http.NewServeMux()
	mux.Handle(e.path, promhttp.InstrumentMetricHandler(
		e.registerer, promhttp.HandlerFor(e.gatherer, promhttp.HandlerOpts{}),
	))

	e.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", e.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		e.log.Info(fmt.Sprintf("metrics server started on port %d at path %s", e.port, e.path))
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("metrics server failed: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), e.shutdownTimeout)
		defer cancel()
		if err := e.server.Shutdown(stopCtx); err != nil {
			return fmt.Errorf("metrics server shutdown failed: %w", err)
		}
		return fmt.Errorf("metrics server context canceled: %w", ctx.Err())
	}
}

// Stop method to stop the MetricsServerApp.
func (e *MetricsServerApp) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.server == nil {
		return nil // Server was never started
	}

	e.log.Debug("stopping metrics server gracefully")
	shutdownCtx, cancel := context.WithTimeout(ctx, e.shutdownTimeout)
	defer cancel()

	if err := e.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to stop metrics server: %w", err)
	}
	e.log.Info("metrics server stopped")
	return nil
}

// SetLogger implementation of Loggable interface to automatically inject logger.
func (e *MetricsServerApp) SetLogger(logger log.Logger) {
	e.log = logger
}
