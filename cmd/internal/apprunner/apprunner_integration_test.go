//go:build !skip_integration_tests

package apprunner_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/kannancmohan/go-prototype-backend/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend/cmd/internal/apprunner"
	log_impl "github.com/kannancmohan/go-prototype-backend/cmd/internal/common/log"
	app_trace "github.com/kannancmohan/go-prototype-backend/cmd/internal/common/trace"
	"github.com/kannancmohan/go-prototype-backend/internal/common/log"
	"github.com/kannancmohan/go-prototype-backend/internal/testutils"
	tc_testutils "github.com/kannancmohan/go-prototype-backend/internal/testutils/testcontainers"
	"github.com/prometheus/client_golang/prometheus"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestAppRunnerMetricsIntegration(t *testing.T) {
	appHost, err := testutils.GetLocalIP()
	if err != nil {
		t.Fatalf("failed to get local ip address: %v", err)
	}

	ports, err := testutils.GetFreePorts(2)
	if err != nil {
		t.Fatalf("failed to get free ports: %v", err)
	}

	appPort := ports[0]
	metricsAppPort := ports[1]
	ctx := context.Background()

	promContainer := tc_testutils.NewPrometheusContainer(fmt.Sprintf("%s:%d", appHost, metricsAppPort))
	err = promContainer.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start prometheus container : %v", err)
	}
	defer promContainer.Stop(context.Background())

	containerAddr, err := promContainer.GetContainerAddress(ctx)
	if err != nil {
		t.Fatalf("failed to get prometheus container address: %v", err)
	}

	config := apprunner.AppRunnerConfig{
		MetricsServerConfig: app.MetricsServerAppConfig{
			Enabled:         true,
			Port:            metricsAppPort,
			Path:            "/metrics",
			ShutdownTimeout: 5 * time.Second,
		},
		Logger:   log_impl.NewSimpleSlogLogger(log_impl.INFO),
		ExitWait: 5 * time.Second,
	}

	runner, _ := apprunner.NewAppRunner(newTestApp("app1", appPort, nil), config, app.EmptyAppConf)
	defer runner.StopApps()

	go func() {
		runCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		if err := runner.Run(runCtx); err != nil {
			t.Logf("AppRunner failed: %v", err)
		}
	}()

	if err := testutils.WaitForPort(appPort, 15*time.Second); err != nil {
		t.Fatalf("failed waiting for app port: %v", err)
	}

	retryDelay := 2 * time.Second
	appUrl := fmt.Sprintf("http://%s:%d/", appHost, appPort)
	_, err = testutils.RetryHTTPGetRequest(appUrl, "", http.StatusOK, 2, retryDelay)
	if err != nil {
		t.Error("expected app response, received error instead.", err.Error())
	}

	metricsAppUrl := fmt.Sprintf("http://%s:%d/metrics", appHost, metricsAppPort)
	_, err = testutils.RetryHTTPGetRequest(metricsAppUrl, "", http.StatusOK, 2, retryDelay)
	if err != nil {
		t.Error("expected metrics response, received error instead.", err.Error())
	}

	promUrl := fmt.Sprintf("http://%s/api/v1/query?query=test_app_requests_total", containerAddr.Address)
	_, err = testutils.RetryHTTPGetRequest(promUrl, "test_app_requests_total", http.StatusOK, 5, retryDelay)
	if err != nil {
		t.Error("expected 'test_app_requests_total' in prometheus response, received error instead.", err.Error())
	}

}

func TestAppRunnerTracerIntegration(t *testing.T) {

	port, err := testutils.GetFreePorts(1)
	if err != nil {
		t.Fatalf("failed to get free ports: %v", err)
	}

	appPort := port[0]
	ctx := context.Background()

	tempo := tc_testutils.NewTempoContainer()
	err = tempo.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start grafana-tempo container : %v", err)
	}
	defer tempo.Stop(context.Background())

	tempoOtlpHttpAddr, err := tempo.GetContainerOTLPHttpAddress(ctx)
	if err != nil {
		t.Fatalf("failed to get grafana-tempo container otlp-http address: %v", err)
	}

	tempoApiAddr, err := tempo.GetContainerApiAddress(ctx)
	if err != nil {
		t.Fatalf("failed to get grafana-tempo container api address: %v", err)
	}

	tp, shutdown, err := app_trace.NewOTelTracerProvider(app_trace.OpenTelemetryConfig{Host: tempoOtlpHttpAddr.Host,
		Port:        tempoOtlpHttpAddr.Port,
		ConnType:    app_trace.OTelConnTypeHTTP,
		ServiceName: "tracer-int-test-service-dev"},
	)
	if err != nil {
		panic(fmt.Errorf("error creating otel tracer provider: %w", err))
	}
	defer shutdown(context.Background())

	config := apprunner.AppRunnerConfig{
		MetricsServerConfig: app.MetricsServerAppConfig{
			Enabled: false,
		},
		Logger: log_impl.NewSimpleSlogLogger(log_impl.INFO),
		TracingConfig: apprunner.TracingConfig{
			Enabled:        true,
			TracerProvider: tp,
		},
		ExitWait: 5 * time.Second,
	}

	runner, _ := apprunner.NewAppRunner(newTestApp("app1", appPort, nil), config, app.EmptyAppConf)
	defer runner.StopApps()

	go func() {
		runCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		if err := runner.Run(runCtx); err != nil {
			t.Logf("AppRunner failed: %v", err)
		}
	}()

	if err := testutils.WaitForPort(appPort, 15*time.Second); err != nil {
		t.Fatalf("failed waiting for app port: %v", err)
	}

	retryDelay := 2 * time.Second
	appUrl := fmt.Sprintf("http://%s:%d/", "localhost", appPort)
	_, err = testutils.RetryHTTPGetRequest(appUrl, "", http.StatusOK, 2, retryDelay)
	if err != nil {
		t.Error("expected app response, received error instead.", err.Error())
	}

	tempoSearchUrl := fmt.Sprintf("http://%s/api/v2/search/tag/.%s/values", tempoApiAddr.Address, tracerTestTagName)
	_, err = testutils.RetryHTTPGetRequest(tempoSearchUrl, "app1"+tracerTestTagValuePostfix, http.StatusOK, 10, retryDelay)
	if err != nil {
		t.Error("expected given value in tempo response, received error instead.", err.Error())
	}
}

func TestAppRunnerDistributedTracingWithMultipleApps(t *testing.T) {
	port, err := testutils.GetFreePorts(2)
	if err != nil {
		t.Fatalf("failed to get free ports: %v", err)
	}

	app1Port := port[0]
	app2Port := port[1]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempo := tc_testutils.NewTempoContainer()
	err = tempo.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start grafana-tempo container : %v", err)
	}
	defer tempo.Stop(context.Background())

	tempoOtlpHttpAddr, err := tempo.GetContainerOTLPHttpAddress(ctx)
	if err != nil {
		t.Fatalf("failed to get grafana-tempo container otlp-http address: %v", err)
	}

	tempoApiAddr, err := tempo.GetContainerApiAddress(ctx)
	if err != nil {
		t.Fatalf("failed to get grafana-tempo container api address: %v", err)
	}

	//APP1
	app1Config, app1ConfigCleanup, err := createAppRunnerConfig("app1", tempoOtlpHttpAddr.Host, tempoOtlpHttpAddr.Port, 0)
	if err != nil {
		panic(fmt.Errorf("error creating AppRunnerConfig for app1: %w", err))
	}
	defer app1ConfigCleanup(context.Background())

	app1 := newTestApp("app1", app1Port, newSimpleTestService(app2Port))
	app1Runner, _ := apprunner.NewAppRunner(app1, app1Config, app.EmptyAppConf)

	//APP2
	app2Config, app2ConfigCleanup, err := createAppRunnerConfig("app2", tempoOtlpHttpAddr.Host, tempoOtlpHttpAddr.Port, 0)
	if err != nil {
		panic(fmt.Errorf("error creating AppRunnerConfig for app2: %w", err))
	}
	defer app2ConfigCleanup(context.Background())

	app2 := newTestApp("app2", app2Port, nil)
	app2Runner, _ := apprunner.NewAppRunner(app2, app2Config, app.EmptyAppConf)

	//execute both apps
	var wg sync.WaitGroup //to wait for the apps to shutdown properly
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := app1Runner.Run(ctx); err != nil {
			t.Logf("AppRunner failed to start app1: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := app2Runner.Run(ctx); err != nil {
			t.Logf("AppRunner failed to start app2: %v", err)
		}
	}()

	if err := testutils.WaitForPort(app1Port, 15*time.Second); err != nil {
		t.Fatalf("failed waiting for app1 port: %v", err)
	}

	if err := testutils.WaitForPort(app2Port, 15*time.Second); err != nil {
		t.Fatalf("failed waiting for app2 port: %v", err)
	}

	retryDelay := 2 * time.Second
	appUrl := fmt.Sprintf("http://%s:%d/", "localhost", app1Port)
	_, err = testutils.RetryHTTPGetRequest(appUrl, "", http.StatusOK, 2, retryDelay)
	if err != nil {
		t.Error("expected app response, received error instead.", err.Error())
	}

	tempoSearchUrl := fmt.Sprintf("http://%s/api/v2/search/tag/.%s/values", tempoApiAddr.Address, tracerTestTagName)
	_, err = testutils.RetryHTTPGetRequest(tempoSearchUrl, "app1"+tracerTestTagValuePostfix, http.StatusOK, 10, retryDelay)
	if err != nil {
		t.Error("expected given value in tempo response, received error instead.", err.Error())
	}

	cancel()  // Cancel the context to signal the apps to shut down
	wg.Wait() // Wait for the apps to shut down

}

type testApp struct {
	name           string
	port           int
	server         *http.Server
	requestCounter prometheus.Counter
	log            log.Logger
	tracer         trace.Tracer
	service        testService
}

func newTestApp(appName string, appPort int, service testService) *testApp {
	return &testApp{
		port: appPort,
		name: appName,
		requestCounter: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "test_app_requests_total",
				Help: "Total number of requests processed by the test app.",
			},
		),
		service: service,
	}
}

func (e *testApp) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//adding some additional attributes to tracing
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(attribute.String(tracerTestTagName, e.name+tracerTestTagValuePostfix))

		e.log.Info("Incoming request", "app", e.name, "TraceID", span.SpanContext().TraceID(), "SpanID", span.SpanContext().SpanID())

		if e.requestCounter != nil {
			e.requestCounter.Inc() //increment prometheus counter metrics
		}

		if e.service != nil {
			resp, _ := e.service.invokeExternalService(r.Context())
			e.log.Debug("invoked test-handler with external service", "app", e.name, "externalResp", resp)
			fmt.Fprintf(w, "invoked test-handler in %s with external service resp: %s", e.name, resp)
		} else {
			e.log.Debug("invoked test-handler", "app", e.name)
			fmt.Fprintf(w, "invoked test-handler in %s", e.name)
		}
	})

	mux.Handle("/", otelhttp.NewHandler(testHandler, "handle-request"))
	e.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", e.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		e.log.Info("starting app", "appName", e.name, "port", e.port)
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("test server failed: %w", err)
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	return nil
}

func (e *testApp) Stop(ctx context.Context) error {
	if e.server == nil {
		return nil // Server was never started
	}

	if err := e.server.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("failed to stop test server: %w", err)
	}
	e.log.Info("stopped the app", "appName", e.name)
	return nil
}

func (e *testApp) PrometheusCollectors() []prometheus.Collector {
	return []prometheus.Collector{e.requestCounter}
}

func (e *testApp) SetLogger(logger log.Logger) {
	e.log = logger
}

func (e *testApp) SetTracer(tracer trace.Tracer) {
	e.tracer = tracer
}

func (e *testApp) newOTELSpan(ctx context.Context, spanName, spanAttName, spanAttValue string) trace.Span {
	if e.tracer == nil {
		_, span := noop.NewTracerProvider().Tracer("").Start(ctx, spanName)
		return span // Return a no-op span
	}
	_, span := e.tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))
	span.SetAttributes(attribute.String(spanAttName, spanAttValue))

	return span
}

type testService interface {
	invokeExternalService(context.Context) (string, error)
}

func newSimpleTestService(externalServicePort int) simpleTestService {
	return simpleTestService{
		externalPort: externalServicePort,
		httpClient: &http.Client{
			//the otelhttp.NewTransport will automatically injected trace context into outgoing requests
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

type simpleTestService struct {
	externalPort int
	httpClient   *http.Client
}

func (t simpleTestService) invokeExternalService(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:%d/", t.externalPort), nil)
	if err != nil {
		return "", fmt.Errorf("Failed to create HTTP request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

const (
	tracerTestTagName         = "handler.id"
	tracerTestTagValuePostfix = "_simple-service__test-handler"
)

func createAppRunnerConfig(appName, tempoHost string, tempoPort, metricsAppPort int, additionalApps ...app.App) (apprunner.AppRunnerConfig, func(context.Context) error, error) {
	config := apprunner.AppRunnerConfig{
		Logger:         log_impl.NewSimpleSlogLogger(log_impl.INFO),
		ExitWait:       5 * time.Second,
		AdditionalApps: additionalApps,
	}
	if metricsAppPort > 0 {
		config.MetricsServerConfig = app.MetricsServerAppConfig{
			Enabled: true,
			Port:    metricsAppPort,
		}
	}
	if tempoHost != "" && tempoPort > 0 {
		tp, shutdown, err := app_trace.NewOTelTracerProvider(app_trace.OpenTelemetryConfig{Host: tempoHost,
			Port:        tempoPort,
			ConnType:    app_trace.OTelConnTypeHTTP,
			ServiceName: appName},
		)
		if err != nil {
			return config, nil, err
		}

		config.TracingConfig = apprunner.TracingConfig{
			Enabled:        true,
			TracerProvider: tp,
		}
		return config, shutdown, nil
	}

	return config, func(ctx context.Context) error { return nil }, nil
}
