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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	prometheusImage       = "prom/prometheus:v3.1.0"
	prometheusExposedPort = "9090"
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
	metricsAppAddr := fmt.Sprintf("%s:%d", appHost, metricsAppPort)
	ctx := context.Background()
	prometheus, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        prometheusImage,
			ExposedPorts: []string{fmt.Sprintf("%s/tcp", prometheusExposedPort)},
			WaitingFor:   wait.ForHTTP("/").WithPort(prometheusExposedPort + "/tcp"),
			LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
				{
					PreStarts: []testcontainers.ContainerHook{
						func(ctx context.Context, c testcontainers.Container) error {
							err := c.CopyToContainer(ctx, getPrometheusConfig(metricsAppAddr), "/etc/prometheus/prometheus.yml", 0o755)
							if err != nil {
								return fmt.Errorf("failed to copy script to container: %w", err)
							}
							return nil
						},
					},
				},
			},
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("failed to create prometheus container : %v", err)
	}
	defer prometheus.Terminate(ctx)

	prometheusHost, err := prometheus.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get prometheus container host: %v", err)
	}
	prometheusPort, err := prometheus.MappedPort(ctx, prometheusExposedPort)
	if err != nil {
		t.Fatalf("failed to get prometheus container port: %v", err)
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

	promUrl := fmt.Sprintf("http://%s:%s/api/v1/query?query=example_app_requests_total", prometheusHost, prometheusPort.Port())
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

func getPrometheusConfig(appHost string) []byte {
	config := `global:
  scrape_interval: 1s

scrape_configs:
  - job_name: "prometheus"
    static_configs:
      - targets: ["%s"]`

	return []byte(fmt.Sprintf(config, appHost))
}
