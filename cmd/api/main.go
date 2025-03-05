package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/apprunner"
	log_impl "github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/common/log"
	app_trace "github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/common/trace"
	"github.com/kannancmohan/go-prototype-backend-apps-temp/internal/common/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

func main() {

	appConf := &app.AppConf[testAppEnvVar]{Name: "test-app"} //TODO ensure mandatory fields(eg Name) are available
	logger := log_impl.NewSimpleSlogLogger(log_impl.INFO)

	tracer, shutdown, err := app_trace.NewOTelTracerProvider(app_trace.OpenTelemetryConfig{Host: "localhost",
		Port:        3200,
		ConnType:    app_trace.OTelConnTypeHTTP,
		ServiceName: appConf.Name,
	})
	if err != nil {
		panic(fmt.Errorf("error creating otel tracer provider: %w", err))
	}
	defer shutdown(context.Background())

	runner, err := apprunner.NewAppRunner(NewTestApp(9933),
		apprunner.AppRunnerConfig{
			ExitWait: 5 * time.Second,
			MetricsServerConfig: app.MetricsServerAppConfig{
				Enabled: true,
			},
			Logger: logger,
			TracingConfig: apprunner.TracingConfig{
				Enabled:        true,
				TracerProvider: tracer,
			},
		},
		appConf)
	if err != nil {
		panic(fmt.Errorf("error creating apprunner: %w", err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	logger.Info("Starting App...")

	err = runner.Run(ctx)
	if err != nil {
		panic(fmt.Errorf("error running apprunner: %w", err))
	}
}

type testAppEnvVar struct {
	EnvName  string
	LogLevel string
}

func NewTestApp(port int) *testApp {
	return &testApp{
		port:            port,
		shutdownTimeout: 5 * time.Second,
	}
}

type testApp struct {
	port            int
	shutdownTimeout time.Duration
	server          *http.Server
	log             log.Logger
	appConf         *app.AppConf[testAppEnvVar]
	tracer          trace.Tracer
	mu              sync.Mutex
}

func (t *testApp) Run(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	mux := http.NewServeMux()
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "main-handler: %s\n", r.URL.Query().Get("name"))
	})

	// using otelhttp.NewHandler to automatically extract trace context(if any) from incoming request
	// and to add this extracted trace context to the request’s context.Context
	mux.Handle("/", otelhttp.NewHandler(testHandler, "handle-request"))

	t.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", t.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		t.log.Info(fmt.Sprintf("test server started on port %d", t.port))
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("test server failed: %w", err)
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		//TODO check whether we need to close the server . check how metrics server is stopped
	}
	return nil
}

func (t *testApp) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.server == nil {
		return nil // Server was never started
	}

	t.log.Debug("stopping test server gracefully")
	shutdownCtx, cancel := context.WithTimeout(ctx, t.shutdownTimeout)
	defer cancel()

	if err := t.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to stop test server: %w", err)
	}
	t.log.Info("test server stopped")
	return nil
}

func (t *testApp) SetLogger(logger log.Logger) {
	t.log = logger
}

func (t *testApp) SetAppConf(conf *app.AppConf[testAppEnvVar]) {
	t.appConf = conf
}

func (t *testApp) SetTracer(tracer trace.Tracer) {
	t.tracer = tracer
}

var _ app.Loggable = &testApp{}
var _ app.AppConfigSetter[testAppEnvVar] = &testApp{}
