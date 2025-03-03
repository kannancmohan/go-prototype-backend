package apprunner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend-apps-temp/internal/common/log"
)

type AppRunnerConfig struct {
	AdditionalApps      []app.App                  // Additional apps to run
	ExitWait            time.Duration              // Maximum duration to wait for apps to stop. (Need to set a value greater than 0 to take effect)
	MetricsServerConfig app.MetricsServerAppConfig // Configuration for the metrics server
	Logger              log.Logger
}

type appRunner struct {
	apps     []app.App
	log      log.Logger
	exitWait time.Duration // Maximum duration to wait for apps to stop
	mu       sync.Mutex    // Mutex to protect the apps slice
}

func NewAppRunner[T any](mainApp app.App, config AppRunnerConfig, appConfig *app.AppConf[T]) (*appRunner, error) {
	if mainApp == nil {
		return nil, fmt.Errorf("mainApp cannot be nil")
	}

	// if logger is not available, setting it to NoOpLogger which does nothing when its method are called
	if config.Logger == nil {
		config.Logger = &log.NoOpLogger{}
	}

	apps := []app.App{mainApp}

	if len(config.AdditionalApps) > 0 {
		apps = append(apps, config.AdditionalApps...)
	}

	// register the metrics collectors from apps that supports it
	if config.MetricsServerConfig.Enabled {
		metricsApp := app.NewMetricsServerApp(config.MetricsServerConfig)
		for _, ap := range apps {
			if provider, ok := ap.(app.MetricsSetter); ok {
				collectors := provider.PrometheusCollectors()
				metricsApp.RegisterCollectors(collectors...)
			}
		}
		apps = append(apps, metricsApp)
	}

	// set the logger and appConf into each app that supports it
	for _, ap := range apps {
		if loggableApp, ok := ap.(app.Loggable); ok {
			loggableApp.SetLogger(config.Logger)
		}
		if configurableApp, ok := ap.(app.AppConfigSetter[T]); ok {
			configurableApp.SetAppConf(appConfig)
		}
	}

	return &appRunner{
		apps:     apps,
		log:      config.Logger,
		exitWait: config.ExitWait,
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

	// Wait for all apps to stop or for the ExitWait timeout
	select {
	case <-done: // All apps stopped
		return err
	case <-stopCtx.Done(): // ExitWait timeout exceeded
		return fmt.Errorf("stopping apps exceeded ExitWait duration: %w", stopCtx.Err())
	}
}
