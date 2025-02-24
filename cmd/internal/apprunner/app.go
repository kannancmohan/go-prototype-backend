package apprunner

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type App interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Implementation of metrics sidecar app
type MetricsServerAppConfig struct {
	Registerer prometheus.Registerer
	Gatherer   prometheus.Gatherer
	Port       int
	Path       string
}

func newMetricsServerApp(cfg MetricsServerAppConfig) *MetricsServerApp {
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
	return &MetricsServerApp{
		Registerer: cfg.Registerer,
		Gatherer:   cfg.Gatherer,
		Port:       cfg.Port,
		Path:       cfg.Path,
	}
}

var _ App = &MetricsServerApp{}

// MetricsServerApp exports prometheus metrics
type MetricsServerApp struct {
	Registerer prometheus.Registerer
	Gatherer   prometheus.Gatherer
	Port       int
	Path       string
	server     *http.Server
	mu         sync.Mutex // Mutex to protect the server field
}

// RegisterCollectors registers the provided collectors with the Exporter's Registerer.
// If there is an error registering any of the provided collectors, registration is halted and an error is returned.
func (e *MetricsServerApp) RegisterCollectors(metrics ...prometheus.Collector) error {
	for _, m := range metrics {
		err := e.Registerer.Register(m)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *MetricsServerApp) Run(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Create a new HTTP server
	mux := http.NewServeMux()
	mux.Handle(e.Path, promhttp.InstrumentMetricHandler(
		e.Registerer, promhttp.HandlerFor(e.Gatherer, promhttp.HandlerOpts{}),
	))

	e.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", e.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		log.Printf("Metrics server started on port %d at path %s", e.Port, e.Path)
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("metrics server failed: %w", err)
		}
	}()

	// Wait for the server to start or for the context to be canceled
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Context was canceled, stop the server
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	log.Println("Stopping metrics server gracefully")
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second) // Allow 5 seconds for graceful shutdown
	defer cancel()

	if err := e.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to stop metrics server: %w", err)
	}

	log.Println("Metrics server stopped")
	return nil
}
