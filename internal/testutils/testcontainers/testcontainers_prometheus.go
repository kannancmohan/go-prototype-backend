package testcontainers_testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	prometheusImage       = "prom/prometheus:v3.1.0"
	prometheusExposedPort = "9090"
)

type testPrometheusContainer struct {
}

func NewPrometheusContainer() *testPrometheusContainer {
	return &testPrometheusContainer{}
}

func (p *testPrometheusContainer) CreatePrometheusTestContainer(metricsAppAddr string) (testcontainers.Container, func(ctx context.Context) error, error) {
	if metricsAppAddr == "" {
		metricsAppAddr = "localhost:9090"
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ctr, err := testcontainers.GenericContainer(timeoutCtx, testcontainers.GenericContainerRequest{
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
		return ctr, func(ctx context.Context) error { return nil }, err
	}

	cleanupFunc := func(ctx context.Context) error {
		err := ctr.Terminate(ctx)
		if err != nil {
			return err
		}
		return nil
	}
	return ctr, cleanupFunc, nil

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
