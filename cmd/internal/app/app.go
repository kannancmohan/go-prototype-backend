package app

import (
	"context"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/internal/common/log"
	"github.com/prometheus/client_golang/prometheus"
)

var EmptyAppConf *AppConf[struct{}]

type AppConf[T any] struct {
	Name   string
	EnvVar T
}

type App interface {
	Run(ctx context.Context) error
	Stop(ctx context.Context) error
}

// MetricsSetter interface. apps that needs to expose additional metrics to prometheus should implement this interface
type MetricsSetter interface {
	PrometheusCollectors() []prometheus.Collector
}

// Loggable . apps that need to use the logger should implement this interface
// appRunner will automatically set logger to apps that implement this interface
type Loggable interface {
	SetLogger(log.Logger)
}

// AppConfigSetter . apps that need AppConf should implement this interface
// appRunner will automatically set AppConf to apps that implement this interface
type AppConfigSetter[T any] interface {
	SetAppConf(conf *AppConf[T])
}
