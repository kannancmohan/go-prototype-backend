package testcontainers_testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	prometheusImage            = "prom/prometheus:v3.1.0"
	prometheusExposedPort      = "9090/tcp"
	prometheusClientConfigPath = "/etc/prometheus/prometheus.yml"
)

var _ tContainer[testcontainers.Container] = &testPrometheusContainer{}

type testPrometheusContainer struct {
	metricsAppAddr string
	container      testcontainers.Container
}

// metricsAppAddr is the endpoint from where prometheus will pulls metrics from.
func NewPrometheusContainer(metricsAppAddr string) *testPrometheusContainer {
	if metricsAppAddr == "" {
		metricsAppAddr = "localhost:9090" //default address of metrics app
	}
	return &testPrometheusContainer{metricsAppAddr: metricsAppAddr}
}

func (p *testPrometheusContainer) Start(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        prometheusImage,
		ExposedPorts: []string{prometheusExposedPort},
		WaitingFor:   wait.ForHTTP("/").WithPort(prometheusExposedPort),
		LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
			{
				PreStarts: []testcontainers.ContainerHook{
					func(ctx context.Context, c testcontainers.Container) error {
						err := c.CopyToContainer(ctx, getPrometheusConfig(p.metricsAppAddr), prometheusClientConfigPath, 0o644)
						if err != nil {
							return fmt.Errorf("failed to copy prometheus config to container: %w", err)
						}
						return nil
					},
				},
			},
		},
	}

	container, err := testcontainers.GenericContainer(timeoutCtx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return err
	}

	p.container = container

	return nil
}

func (p *testPrometheusContainer) Stop(ctx context.Context) error {
	if p.container != nil {
		if err := p.container.Terminate(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *testPrometheusContainer) GetContainer() (testcontainers.Container, error) {
	if p.container == nil {
		return nil, fmt.Errorf("GetContainer : prometheus container not initialized/started")
	}
	return p.container, nil
}

func (p *testPrometheusContainer) GetContainerAddress(ctx context.Context) (testContainerAddr, error) {
	if p.container == nil {
		return testContainerAddr{}, fmt.Errorf("GetContainerAddress : prometheus container not initialized/started")
	}
	host, err := p.container.Host(ctx)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get prometheus container host: %w", err)
	}
	port, err := p.container.MappedPort(ctx, prometheusExposedPort)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get prometheus container port: %w", err)
	}
	return newTestContainerAddr(host, port.Int()), nil
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
