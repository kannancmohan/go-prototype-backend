package apprunner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kannancmohan/go-prototype-backend/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend/internal/common/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type AppRunnerConfig struct {
	metricsApp     *app.MetricsServerApp
	log            log.Logger
	tracerProvider trace.TracerProvider
	additionalApps []app.App
	exitWait       time.Duration // Maximum duration to wait for apps to stop
}

func NewAppRunnerConfig(opts ...AppRunnerConfigOption) AppRunnerConfig {
	config := AppRunnerConfig{
		log:            &log.NoOpLogger{},
		exitWait:       5 * time.Second,
		tracerProvider: otel.GetTracerProvider(),
	}
	for _, opt := range opts {
		opt(&config)
	}
	return config

}

type AppRunnerConfigOption func(*AppRunnerConfig)

func WithMetricsApp(cfg app.MetricsServerAppConfig) AppRunnerConfigOption {
	return func(a *AppRunnerConfig) {
		a.metricsApp = app.NewMetricsServerApp(cfg)
	}
}

func WithLogger(log log.Logger) AppRunnerConfigOption {
	return func(a *AppRunnerConfig) {
		a.log = log
	}
}

func WithTracerProvider(tp trace.TracerProvider) AppRunnerConfigOption {
	return func(a *AppRunnerConfig) {
		a.tracerProvider = tp
	}
}

func WithExitWait(exitWait time.Duration) AppRunnerConfigOption {
	return func(a *AppRunnerConfig) {
		a.exitWait = exitWait
	}
}

func WithAdditionalApps(apps []app.App) AppRunnerConfigOption {
	return func(a *AppRunnerConfig) {
		a.additionalApps = apps
	}
}

type appRunner struct {
	apps     []app.App
	log      log.Logger
	tracer   trace.Tracer  // appRunner could trace its own operations(eg app start/stop)
	exitWait time.Duration // Maximum duration to wait for apps to stop
	mu       sync.Mutex    // Mutex to protect the apps slice
}

func NewAppRunner[T any](mainApp app.App, arConfig AppRunnerConfig, appsCommonConfig *app.AppConf[T]) (*appRunner, error) {
	if mainApp == nil {
		return nil, fmt.Errorf("mainApp cannot be nil")
	}

	apps := []app.App{mainApp}

	if len(arConfig.additionalApps) > 0 {
		apps = append(apps, arConfig.additionalApps...)
	}

	if arConfig.metricsApp != nil {
		// register the metrics collectors from apps that supports it
		for _, ap := range apps {
			if provider, ok := ap.(app.MetricsSetter); ok {
				collectors := provider.PrometheusCollectors()
				arConfig.metricsApp.RegisterCollectors(collectors...)
			}
		}
		apps = append(apps, arConfig.metricsApp)
	}

	// set the logger and appConf to apps that supports it
	for _, ap := range apps {
		if loggableApp, ok := ap.(app.Loggable); ok {
			loggableApp.SetLogger(arConfig.log)
		}
		if appsCommonConfig != nil {
			if configurableApp, ok := ap.(app.AppConfigSetter[T]); ok {
				configurableApp.SetAppConf(appsCommonConfig)
			}
		}
	}

	// set tracing to apps that supports it
	var appRunnerTracer trace.Tracer
	if arConfig.tracerProvider != nil {
		traceProvide := arConfig.tracerProvider
		appRunnerTracer = traceProvide.Tracer("apprunner") //creating a tracer for appRunner in case it needs to add tracing
		for _, ap := range apps {
			if traceable, ok := ap.(app.Traceable); ok {
				traceable.SetTracerProvider(traceProvide)
			}
		}
	}

	return &appRunner{
		apps:     apps,
		log:      arConfig.log,
		exitWait: arConfig.exitWait,
		tracer:   appRunnerTracer,
	}, nil
}

func (ar *appRunner) Run(ctx context.Context) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(ar.apps))

	// Start all apps
	for _, ap := range ar.apps {
		wg.Add(1)
		go func(a app.App) {
			defer wg.Done()
			if err := a.Run(ctx); err != nil {
				errChan <- fmt.Errorf("error starting app: %w", err)
			}
		}(ap)
	}

	// Wait for an app to fail or context cancellation
	select {
	case <-ctx.Done():
		ar.log.Info("Context canceled, stopping apps")
		if err := ar.StopApps(); err != nil {
			return fmt.Errorf("failed to stop apps: %w", err)
		}
		return nil
	case err := <-errChan:
		ar.log.Info("App failed, stopping all apps")
		if stopErr := ar.StopApps(); stopErr != nil {
			return errors.Join(err, fmt.Errorf("failed to stop apps: %w", stopErr))
		}
		return err
	}
}

func (ar *appRunner) StopApps() error {
	var stopCtx context.Context
	var cancel context.CancelFunc
	if ar.exitWait > 0 {
		stopCtx, cancel = context.WithTimeout(context.Background(), ar.exitWait)
	} else {
		stopCtx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	var err error
	var wg sync.WaitGroup
	for _, ap := range ar.apps {
		wg.Add(1)
		go func(a app.App) {
			defer wg.Done()
			if stopErr := a.Stop(stopCtx); stopErr != nil {
				err = errors.Join(err, fmt.Errorf("app stop error: %w", stopErr))
			}
		}(ap)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for all apps to stop or for context with ExitWait to timeout
	select {
	case <-done: // All apps stopped
		return err
	case <-stopCtx.Done(): // ExitWait timeout exceeded
		return fmt.Errorf("stopping apps exceeded ExitWait duration: %w", stopCtx.Err())
	}
}
