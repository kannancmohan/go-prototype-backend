package apprunner

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockApp is a mock implementation of the App interface for testing.
type MockApp struct {
	RunFunc  func(ctx context.Context) error
	StopFunc func(ctx context.Context) error
}

func (m *MockApp) Run(ctx context.Context) error {
	return m.RunFunc(ctx)
}

func (m *MockApp) Stop(ctx context.Context) error {
	return m.StopFunc(ctx)
}

func TestAppRunner_Run(t *testing.T) {
	t.Run("Successful Run", func(t *testing.T) {
		app := &MockApp{
			RunFunc: func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
			StopFunc: func(ctx context.Context) error {
				return nil
			},
		}

		config := AppRunnerConfig{
			MetricsServerConfig: MetricsServerAppConfig{
				Enabled: false,
			},
			ExitWait: ptrDuration(5 * time.Second),
		}

		runner := NewAppRunner(app, config)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := runner.Run(ctx)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("App Failure", func(t *testing.T) {
		app := &MockApp{
			RunFunc: func(ctx context.Context) error {
				return errors.New("mock app failed")
			},
			StopFunc: func(ctx context.Context) error {
				return nil
			},
		}

		config := AppRunnerConfig{
			MetricsServerConfig: MetricsServerAppConfig{
				Enabled: false,
			},
			ExitWait: ptrDuration(5 * time.Second),
		}

		runner := NewAppRunner(app, config)
		ctx := context.Background()

		err := runner.Run(ctx)
		if err == nil {
			t.Error("Expected an error, got nil")
		} else if !strings.Contains(err.Error(), "mock app failed") {
			t.Errorf("Expected error 'app failed', got: %v", err)
		}
	})

	t.Run("Metrics Server Enabled", func(t *testing.T) {
		app := &MockApp{
			RunFunc: func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
			StopFunc: func(ctx context.Context) error {
				return nil
			},
		}

		config := AppRunnerConfig{
			MetricsServerConfig: MetricsServerAppConfig{
				Enabled:         true,
				Port:            9090,
				Path:            "/metrics",
				ShutdownTimeout: 1 * time.Second,
			},
			ExitWait: ptrDuration(5 * time.Second),
		}

		runner := NewAppRunner(app, config)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := runner.Run(ctx)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Additional Apps", func(t *testing.T) {
		app := &MockApp{
			RunFunc: func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
			StopFunc: func(ctx context.Context) error {
				return nil
			},
		}

		additionalApp := &MockApp{
			RunFunc: func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
			StopFunc: func(ctx context.Context) error {
				return nil
			},
		}

		config := AppRunnerConfig{
			MetricsServerConfig: MetricsServerAppConfig{
				Enabled: false,
			},
			AdditionalApps: []App{additionalApp},
			ExitWait:       ptrDuration(5 * time.Second),
		}

		runner := NewAppRunner(app, config)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := runner.Run(ctx)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("Concurrent Run and Stop", func(t *testing.T) {
		app := &MockApp{
			RunFunc: func(ctx context.Context) error {
				<-ctx.Done()
				return nil
			},
			StopFunc: func(ctx context.Context) error {
				return nil
			},
		}

		config := AppRunnerConfig{
			MetricsServerConfig: MetricsServerAppConfig{
				Enabled: false,
			},
			ExitWait: ptrDuration(5 * time.Second),
		}

		runner := NewAppRunner(app, config)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := runner.Run(ctx)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		}()

		wg.Wait()
	})
}

func ptrDuration(d time.Duration) *time.Duration {
	return &d
}
