package apprunner_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/apprunner"
	"github.com/kannancmohan/go-prototype-backend-apps-temp/internal/testutils"
	tc_testutils "github.com/kannancmohan/go-prototype-backend-apps-temp/internal/testutils/testcontainers"
	"github.com/prometheus/client_golang/prometheus"
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
		ExitWait: 5 * time.Second,
	}

	runner := apprunner.NewAppRunner(NewExampleApp(appPort), config)
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

	promUrl := fmt.Sprintf("http://%s/api/v1/query?query=example_app_requests_total", containerAddr.Address)
	_, err = testutils.RetryHTTPGetRequest(promUrl, "example_app_requests_total", http.StatusOK, 5, retryDelay)
	if err != nil {
		t.Error("expected prometheus response, received error instead.", err.Error())
	}

}

type exampleApp struct {
	port           int
	server         *http.Server
	requestCounter prometheus.Counter
}

func NewExampleApp(port int) *exampleApp {
	return &exampleApp{
		port: port,
		requestCounter: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "example_app_requests_total",
				Help: "Total number of requests processed by the example app.",
			},
		),
	}
}

func (e *exampleApp) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.requestCounter.Inc() //increment prometheus counter metrics
		fmt.Fprintf(w, "main-handler: %s\n", r.URL.Query().Get("name"))
	}))
	e.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", e.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("starting example app at port:%d", e.port)
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

func (e *exampleApp) Stop(ctx context.Context) error {
	if e.server == nil {
		return nil // Server was never started
	}

	if err := e.server.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("failed to stop test server: %w", err)
	}
	log.Print("stopped the app")
	return nil
}

func (e *exampleApp) PrometheusCollectors() []prometheus.Collector {
	return []prometheus.Collector{e.requestCounter}
}
