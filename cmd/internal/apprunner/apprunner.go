package apprunner

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

type AppRunnerConfig struct {
	EnableMetricsService bool // Whether to enable the metrics service
	EnableAdditionalApps bool // Whether to enable an additional sidecar
	AdditionalApps       []App
	ExitWait             *time.Duration // Maximum duration to wait for apps to stop
}

type appRunner struct {
	apps     []App
	ExitWait *time.Duration // Maximum duration to wait for apps to stop
}

func NewAppRunner(mainApp App, config AppRunnerConfig, sidecars ...App) *appRunner {
	apps := append([]App{mainApp}, config.AdditionalApps...)
	if config.EnableMetricsService {
		apps = append(apps, newMetricsServerApp(MetricsServerAppConfig{}))
	}
	return &appRunner{
		apps:     apps,
		ExitWait: config.ExitWait,
	}

}

// Run starts all apps concurrently and handles errors and rollback.
func (ar *appRunner) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(ar.apps)) // Buffered channel to collect errors

	// Start all apps
	for _, app := range ar.apps {
		wg.Add(1)
		go func(a App) {
			defer wg.Done()

			if err := a.Run(ctx); err != nil {
				errChan <- fmt.Errorf("app failed: %w", err)
			}
		}(app)
	}

	// Wait for an app to fail or context cancellation
	select {
	case <-ctx.Done():
		// Context was canceled or timed out
		log.Println("Context canceled, stopping apps")
		if err := ar.stopApps(ctx); err != nil {
			return fmt.Errorf("failed to stop apps: %w", err)
		}
		return ctx.Err()
	case err := <-errChan:
		// An app failed, stop all apps
		log.Println("App failed, stopping all apps")
		if stopErr := ar.stopApps(ctx); stopErr != nil {
			return errors.Join(err, fmt.Errorf("failed to stop apps: %w", stopErr))
		}
		return err
	}
}

// stopApps stops all apps gracefully within the ExitWait duration.
func (ar *appRunner) stopApps(ctx context.Context) error {
	// Create a context with the ExitWait timeout
	var cancel context.CancelFunc
	if ar.ExitWait != nil {
		ctx, cancel = context.WithTimeout(ctx, *ar.ExitWait)
		defer cancel()
	}

	// Stop all apps
	var err error
	var wg sync.WaitGroup
	for _, app := range ar.apps {
		wg.Add(1)
		go func(a App) {
			defer wg.Done()
			if stopErr := a.Stop(ctx); stopErr != nil {
				err = errors.Join(err, fmt.Errorf("app stop error: %w", stopErr))
			}
		}(app)
	}

	// Wait for all apps to stop or for the ExitWait timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All apps stopped
		return err
	case <-ctx.Done():
		// ExitWait timeout exceeded
		return fmt.Errorf("stopping apps exceeded ExitWait duration: %w", ctx.Err())
	}
}
