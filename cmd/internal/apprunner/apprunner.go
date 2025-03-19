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

type AppRunnerOption func(*appRunnerConfig)

// appRunnerConfig holds the configuration for the appRunner.
type appRunnerConfig struct {
	metricsApp     *app.MetricsServerApp
	log            log.Logger
	tracerProvider trace.TracerProvider
	additionalApps []app.App
	exitWait       time.Duration // Maximum duration to wait for apps to stop
}

func WithMetricsApp(metricsApp *app.MetricsServerApp) AppRunnerOption {
	return func(c *appRunnerConfig) {
		c.metricsApp = metricsApp
	}
}

func WithLogger(log log.Logger) AppRunnerOption {
	return func(c *appRunnerConfig) {
		c.log = log
	}
}

func WithTracerProvider(tp trace.TracerProvider) AppRunnerOption {
	return func(c *appRunnerConfig) {
		c.tracerProvider = tp
	}
}

func WithAdditionalApps(additionalApps []app.App) AppRunnerOption {
	return func(c *appRunnerConfig) {
		c.additionalApps = additionalApps
	}
}

func WithExitWait(exitWait time.Duration) AppRunnerOption {
	return func(c *appRunnerConfig) {
		c.exitWait = exitWait
	}
}

type appRunner struct {
	apps     []app.App
	log      log.Logger
	tracer   trace.Tracer  // appRunner could trace its own operations(eg app start/stop)
	exitWait time.Duration // Maximum duration to wait for apps to stop
	mu       sync.Mutex    // Mutex to protect the apps slice
}

func NewAppRunner[T any](mainApp app.App, appsCommonCfg app.AppConf[T], opts ...AppRunnerOption) (*appRunner, error) {
	if mainApp == nil {
		return nil, fmt.Errorf("mainApp cannot be nil")
	}

	config := appRunnerConfig{
		log:            &log.NoOpLogger{},        // Default logger
		exitWait:       5 * time.Second,          // Default exit wait time
		tracerProvider: otel.GetTracerProvider(), // Default TracerProvider
	}

	// Apply functional options
	for _, opt := range opts {
		opt(&config)
	}

	apps := []app.App{mainApp}

	if len(config.additionalApps) > 0 {
		apps = append(apps, config.additionalApps...)
	}

	if config.metricsApp != nil {
		// register the metrics collectors from apps that supports it
		for _, ap := range apps {
			if provider, ok := ap.(app.MetricsSetter); ok {
				collectors := provider.PrometheusCollectors()
				config.metricsApp.RegisterCollectors(collectors...)
			}
		}
		apps = append(apps, config.metricsApp)
	}

	// set the logger and appConf to apps that supports it
	for _, ap := range apps {
		if loggableApp, ok := ap.(app.Loggable); ok {
			loggableApp.SetLogger(config.log)
		}
		if configurableApp, ok := ap.(app.AppConfigSetter[T]); ok {
			configurableApp.SetAppConf(appsCommonCfg)
		}
	}

	// set tracing to apps that supports it
	var appRunnerTracer trace.Tracer
	if config.tracerProvider != nil {
		traceProvide := config.tracerProvider
		appRunnerTracer = traceProvide.Tracer("apprunner") //creating a tracer for appRunner in case it needs tracing
		for _, ap := range apps {
			if traceable, ok := ap.(app.Traceable); ok {
				traceable.SetTracerProvider(traceProvide)
			}
		}
	}

	return &appRunner{
		apps:     apps,
		log:      config.log,
		exitWait: config.exitWait,
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
