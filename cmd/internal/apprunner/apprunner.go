package apprunner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/app"
)

type AppRunnerConfig struct {
	AdditionalApps      []app.App                  // Additional apps to run
	ExitWait            time.Duration              // Maximum duration to wait for apps to stop. (Need to set a value greater than 0 to take effect)
	MetricsServerConfig app.MetricsServerAppConfig // Configuration for the metrics server
}

type appRunner struct {
	apps     []app.App
	exitWait time.Duration // Maximum duration to wait for apps to stop
	mu       sync.Mutex    // Mutex to protect the apps slice
}

func NewAppRunner(mainApp app.App, config AppRunnerConfig) *appRunner {
	apps := []app.App{mainApp}

	if len(config.AdditionalApps) > 0 {
		apps = append(apps, config.AdditionalApps...)
	}

	if config.MetricsServerConfig.Enabled {
		metricsApp := app.NewMetricsServerApp(config.MetricsServerConfig)
		for _, ap := range apps {
			if provider, ok := ap.(app.MetricsProvider); ok {
				collectors := provider.PrometheusCollectors()
				metricsApp.RegisterCollectors(collectors...)
			}
		}
		apps = append(apps, metricsApp)
	}

	return &appRunner{
		apps:     apps,
		exitWait: config.ExitWait,
	}
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
		slog.Info("Context canceled, stopping apps")
		if err := ar.StopApps(); err != nil {
			return fmt.Errorf("failed to stop apps: %w", err)
		}
		return nil
	case err := <-errChan:
		slog.Info("App failed, stopping all apps")
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
