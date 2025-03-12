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

type MetricsServerAppConfig struct {
	Registerer      prometheus.Registerer
	Gatherer        prometheus.Gatherer
	Port            int
	Path            string
	ShutdownTimeout time.Duration // Timeout for graceful shutdown
}

func NewMetricsServerApp(cfg MetricsServerAppConfig) *MetricsServerApp {
	if cfg.Registerer == nil {
		cfg.Registerer = prometheus.DefaultRegisterer
	}
	if cfg.Gatherer == nil {
		cfg.Gatherer = prometheus.DefaultGatherer
	}
	if cfg.Port < 1 {
		cfg.Port = 9090
	}
	if cfg.Path == "" {
		cfg.Path = "/metrics"
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 5 * time.Second
	}
	return &MetricsServerApp{
		Registerer:      cfg.Registerer,
		Gatherer:        cfg.Gatherer,
		Port:            cfg.Port,
		Path:            cfg.Path,
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

var _ App = &MetricsServerApp{}
var _ Loggable = &MetricsServerApp{}

type MetricsServerApp struct {
	Registerer      prometheus.Registerer
	Gatherer        prometheus.Gatherer
	Port            int
	Path            string
	shutdownTimeout time.Duration
	server          *http.Server
	log             log.Logger
	mu              sync.Mutex
}

// RegisterCollectors. custom function to register app specific metrics collector
func (e *MetricsServerApp) RegisterCollectors(metrics ...prometheus.Collector) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, m := range metrics {
		if err := e.Registerer.Register(m); err != nil {
			return fmt.Errorf("failed to register collector: %w", err)
		}
	}
	return nil
}

func (e *MetricsServerApp) Run(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	mux := http.NewServeMux()
	mux.Handle(e.Path, promhttp.InstrumentMetricHandler(
		e.Registerer, promhttp.HandlerFor(e.Gatherer, promhttp.HandlerOpts{}),
	))

	e.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", e.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		e.log.Info(fmt.Sprintf("metrics server started on port %d at path %s", e.Port, e.Path))
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("metrics server failed: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		stopCtx, cancel := context.WithTimeout(context.Background(), e.shutdownTimeout)
		defer cancel()
		if err := e.server.Shutdown(stopCtx); err != nil {
			return fmt.Errorf("metrics server shutdown failed: %w", err)
		}
		return fmt.Errorf("metrics server context canceled: %w", ctx.Err())
	}
}

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

func (e *MetricsServerApp) SetLogger(logger log.Logger) {
	e.log = logger
}
