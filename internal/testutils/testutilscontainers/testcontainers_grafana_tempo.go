package testutilscontainers

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	tempoImage               = "grafana/tempo:2.7.1"
	tempoConfigPath          = "/etc/tempo.yaml"
	tempoAPIPort             = "3200"
	tempoExposedAPIPort      = tempoAPIPort + "/tcp"
	tempoGrpcPort            = "4317"
	tempoExposedOTLPGrpcPort = tempoGrpcPort + "/tcp"
	tempoHTTPPort            = "4318"
	tempoExposedOTLPHttpPort = tempoHTTPPort + "/tcp"
)

var _ tContainer[testcontainers.Container] = &testTempoContainer{}

type testTempoContainer struct {
	container testcontainers.Container
}

// NewTempoContainer to create new tempo container.
func NewTempoContainer() *testTempoContainer {
	return &testTempoContainer{}
}

func (p *testTempoContainer) Start(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        tempoImage,
		Cmd:          []string{"-config.file=" + tempoConfigPath},
		ExposedPorts: []string{tempoExposedAPIPort, tempoExposedOTLPGrpcPort, tempoExposedOTLPHttpPort},
		WaitingFor:   wait.ForLog("Tempo started").WithStartupTimeout(30 * time.Second),
		LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
			{
				PreStarts: []testcontainers.ContainerHook{
					func(ctx context.Context, c testcontainers.Container) error {
						err := c.CopyToContainer(ctx, getTempoConfig(tempoAPIPort, tempoGrpcPort, tempoHTTPPort), tempoConfigPath, 0o644)
						if err != nil {
							return fmt.Errorf("failed to copy tempo config to container: %w", err)
						}
						return nil
					},
				},
			},
		},
	}
	tempo, err := testcontainers.GenericContainer(timeoutCtx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return fmt.Errorf("failed to get grafana container: %w", err)
	}
	p.container = tempo
	return nil
}

func (p *testTempoContainer) Stop(ctx context.Context) error {
	if p.container != nil {
		if err := p.container.Terminate(ctx); err != nil {
			return fmt.Errorf("failed to stop grafana container: %w", err)
		}
	}
	return nil
}

func (p *testTempoContainer) GetContainer() (testcontainers.Container, error) {
	if p.container == nil {
		return nil, fmt.Errorf("GetContainer : grafana-tempo container not initialized/started")
	}
	return p.container, nil
}

func (p *testTempoContainer) GetContainerAPIAddress(ctx context.Context) (testContainerAddr, error) {
	if p.container == nil {
		return testContainerAddr{}, fmt.Errorf("GetContainerApiAddress : grafana-tempo container not initialized/started")
	}
	host, err := p.container.Host(ctx)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get grafana-tempo container host: %w", err)
	}
	port, err := p.container.MappedPort(ctx, tempoExposedAPIPort)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get grafana-tempo container api port: %w", err)
	}
	return newTestContainerAddr(host, port.Int()), nil
}

func (p *testTempoContainer) GetContainerOTLPHttpAddress(ctx context.Context) (testContainerAddr, error) {
	if p.container == nil {
		return testContainerAddr{}, fmt.Errorf("GetContainerOTLPHttpAddress : grafana-tempo container not initialized/started")
	}
	host, err := p.container.Host(ctx)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get grafana-tempo container host: %w", err)
	}
	port, err := p.container.MappedPort(ctx, tempoExposedOTLPHttpPort)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get grafana-tempo container otlp-http port: %w", err)
	}
	return newTestContainerAddr(host, port.Int()), nil
}

func (p *testTempoContainer) GetContainerOTLPGrpcAddress(ctx context.Context) (testContainerAddr, error) {
	if p.container == nil {
		return testContainerAddr{}, fmt.Errorf("GetContainerOTLPGrpcAddress : grafana-tempo container not initialized/started")
	}
	host, err := p.container.Host(ctx)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get grafana-tempo container host: %w", err)
	}
	port, err := p.container.MappedPort(ctx, tempoExposedOTLPGrpcPort)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get grafana-tempo container otlp-grpc port: %w", err)
	}
	return newTestContainerAddr(host, port.Int()), nil
}

func getTempoConfig(tempoAPIPort, tempoOtlpGrpcPort, tempoOtlpHTTPPort string) []byte {
	config := `
server:
  http_listen_port: %s

distributor:
  receivers:
    otlp:
      protocols:
        http:
          endpoint: "0.0.0.0:%s"
        grpc:
          endpoint: "0.0.0.0:%s"

ingester:
  trace_idle_period: 10s
  max_block_bytes: 1000000
  max_block_duration: 5m
  complete_block_timeout: 30m

compactor:
  compaction:
    block_retention: 24h

storage:
  trace:
    backend: local
    local:
      path: /tmp/tempo/traces  # Local storage for traces
`
	return []byte(fmt.Sprintf(config, tempoAPIPort, tempoOtlpHTTPPort, tempoOtlpGrpcPort))
}
