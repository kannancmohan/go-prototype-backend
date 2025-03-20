package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/kannancmohan/go-prototype-backend/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend/internal/common/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

type simpleAppEnvVar struct {
	EnvName  string
	LogLevel string
}

func NewSimpleApp(port int) *simpleApp {
	return &simpleApp{
		port:            port,
		shutdownTimeout: 5 * time.Second,
	}
}

type simpleApp struct {
	port            int
	shutdownTimeout time.Duration
	server          *http.Server
	log             log.Logger
	appConf         app.AppConf[simpleAppEnvVar]
	tp              trace.TracerProvider
	mu              sync.Mutex
}

func (t *simpleApp) Run(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	mux := http.NewServeMux()
	simpleHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.log.WithContext(r.Context()).Debug("Incoming request")
		fmt.Fprintf(w, "main-handler: %s\n", r.URL.Query().Get("name"))
	})

	// otelhttp.NewHandler will automatically extract the tracing context from an incoming request if present and creates a new span
	// as a child of the existing trace. It automatically creates a new trace ID and span if the request does not have tracing context
	mux.Handle("/", otelhttp.NewHandler(simpleHandler, "handle-request"))

	t.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", t.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		t.log.Info(fmt.Sprintf("simple server started on port %d", t.port))
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("simple server failed: %w", err)
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// TODO check whether we need to close the server . check how metrics server is stopped
	}
	return nil
}

func (t *simpleApp) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.server == nil {
		return nil // Server was never started
	}

	t.log.Debug("stopping simple server gracefully")
	shutdownCtx, cancel := context.WithTimeout(ctx, t.shutdownTimeout)
	defer cancel()

	if err := t.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to stop simple server: %w", err)
	}
	t.log.Info("simple server stopped")
	return nil
}

func (t *simpleApp) SetLogger(logger log.Logger) {
	t.log = logger
}

func (t *simpleApp) SetAppConf(conf app.AppConf[simpleAppEnvVar]) {
	t.appConf = conf
}

func (t *simpleApp) SetTracerProvider(tp trace.TracerProvider) {
	t.tp = tp
}

var _ app.Loggable = &simpleApp{}

var _ app.AppConfigSetter[simpleAppEnvVar] = &simpleApp{}

var _ app.Traceable = &simpleApp{}
