package testutilscontainers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	grafana_lgtm "github.com/testcontainers/testcontainers-go/modules/grafana-lgtm"
)

const (
	grafanaLgtmImage = "grafana/otel-lgtm:0.8.6"
)

var _ tContainer[*grafana_lgtm.GrafanaLGTMContainer] = &testLgtmContainer{}

type testLgtmContainer struct {
	metricsAppAddr string
	container      *grafana_lgtm.GrafanaLGTMContainer
}

// metricsAppAddr is the endpoint from where prometheus will pulls metrics from.
func NewTestLgtmContainer(metricsAppAddr string) *testLgtmContainer {
	if metricsAppAddr == "" {
		metricsAppAddr = "localhost:9090" // default address of metrics app
	}
	return &testLgtmContainer{metricsAppAddr: metricsAppAddr}
}

func (p *testLgtmContainer) Start(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	container, err := grafana_lgtm.Run(timeoutCtx, grafanaLgtmImage)
	if err != nil {
		return err
	}
	p.container = container

	return nil
}

func (p *testLgtmContainer) Stop(ctx context.Context) error {
	if p.container != nil {
		if err := p.container.Terminate(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *testLgtmContainer) GetContainer() (*grafana_lgtm.GrafanaLGTMContainer, error) {
	if p.container == nil {
		return nil, fmt.Errorf("GetContainer : lgtm container not initialized/started")
	}
	return p.container, nil
}

func (p *testLgtmContainer) GetPrometheusAddress(ctx context.Context) (testContainerAddr, error) {
	if p.container == nil {
		return testContainerAddr{}, fmt.Errorf("GetPrometheusAddress : lgtm container not initialized/started")
	}

	addr, err := p.container.PrometheusHttpEndpoint(ctx)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get prometheus address: %w", err)
	}
	return generateTContainerAddr(addr)
}

func (p *testLgtmContainer) GetGrafanaAddress(ctx context.Context) (testContainerAddr, error) {
	if p.container == nil {
		return testContainerAddr{}, fmt.Errorf("GetGrafanaAddress : lgtm container not initialized/started")
	}

	addr, err := p.container.HttpEndpoint(ctx)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get grafana address: %w", err)
	}
	return generateTContainerAddr(addr)
}

func (p *testLgtmContainer) GetTempoAddress(ctx context.Context) (testContainerAddr, error) {
	if p.container == nil {
		return testContainerAddr{}, fmt.Errorf("GetTempoAddress : lgtm container not initialized/started")
	}

	addr, err := p.container.TempoEndpoint(ctx)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get tempo address: %w", err)
	}
	return generateTContainerAddr(addr)
}

func (p *testLgtmContainer) GetOtelHTTPAddress(ctx context.Context) (testContainerAddr, error) {
	if p.container == nil {
		return testContainerAddr{}, fmt.Errorf("GetOtelHTTPAddress : lgtm container not initialized/started")
	}

	addr, err := p.container.OtlpHttpEndpoint(ctx)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get otel http address: %w", err)
	}
	return generateTContainerAddr(addr)
}

func (p *testLgtmContainer) GetOtelGRPCAddress(ctx context.Context) (testContainerAddr, error) {
	if p.container == nil {
		return testContainerAddr{}, fmt.Errorf("GetOtelGRPCAddress : lgtm container not initialized/started")
	}

	addr, err := p.container.OtlpGrpcEndpoint(ctx)
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to get otel grpc address: %w", err)
	}
	return generateTContainerAddr(addr)
}

func generateTContainerAddr(address string) (testContainerAddr, error) {
	addrSplit := strings.Split(address, ":")
	if splitLen := len(addrSplit); splitLen != 2 {
		return testContainerAddr{}, fmt.Errorf("failed to split address. expected 2 splits, but splits length was %d", splitLen)
	}
	port, err := strconv.Atoi(addrSplit[1])
	if err != nil {
		return testContainerAddr{}, fmt.Errorf("failed to convert address. %w", err)
	}
	return newTestContainerAddr(addrSplit[0], port), nil
}
