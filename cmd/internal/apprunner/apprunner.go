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
	AdditionalApps      []App                  // Additional apps to run
	ExitWait            *time.Duration         // Maximum duration to wait for apps to stop
	MetricsServerConfig MetricsServerAppConfig // Configuration for the metrics server
}

type appRunner struct {
	apps     []App
	exitWait *time.Duration
	mu       sync.Mutex // Mutex to protect the apps slice
}

func NewAppRunner(mainApp App, config AppRunnerConfig) *appRunner {
	apps := []App{mainApp}

	if config.MetricsServerConfig.Enabled {
		apps = append(apps, newMetricsServerApp(config.MetricsServerConfig))
	}

	if len(config.AdditionalApps) > 0 {
		apps = append(apps, config.AdditionalApps...)
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

func (ar *appRunner) stopApps(ctx context.Context) error {
	// Create a context with the ExitWait timeout
	var cancel context.CancelFunc
	if ar.exitWait != nil {
		ctx, cancel = context.WithTimeout(ctx, *ar.exitWait)
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
