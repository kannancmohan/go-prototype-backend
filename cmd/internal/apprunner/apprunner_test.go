package apprunner_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/apprunner"
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
	tests := []struct {
		name        string
		mainApp     *MockApp
		config      apprunner.AppRunnerConfig
		ctxTimeout  time.Duration
		expectError bool
		errorMsg    string
	}{
		{
			name: "Successful Run",
			mainApp: &MockApp{
				RunFunc: func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
				StopFunc: func(ctx context.Context) error {
					return nil
				},
			},
			config: apprunner.AppRunnerConfig{
				MetricsServerConfig: app.MetricsServerAppConfig{
					Enabled: false,
				},
				ExitWait: 5 * time.Second,
			},
			ctxTimeout:  1 * time.Second,
			expectError: false,
		},
		{
			name: "App Failure",
			mainApp: &MockApp{
				RunFunc: func(ctx context.Context) error {
					return errors.New("mock app failed")
				},
				StopFunc: func(ctx context.Context) error {
					return nil
				},
			},
			config: apprunner.AppRunnerConfig{
				MetricsServerConfig: app.MetricsServerAppConfig{
					Enabled: false,
				},
				ExitWait: 5 * time.Second,
			},
			ctxTimeout:  1 * time.Second,
			expectError: true,
			errorMsg:    "mock app failed",
		},
		{
			name: "Metrics Server Enabled",
			mainApp: &MockApp{
				RunFunc: func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
				StopFunc: func(ctx context.Context) error {
					return nil
				},
			},
			config: apprunner.AppRunnerConfig{
				MetricsServerConfig: app.MetricsServerAppConfig{
					Enabled:         true,
					Port:            9090,
					Path:            "/metrics",
					ShutdownTimeout: 1 * time.Second,
				},
				ExitWait: 5 * time.Second,
			},
			ctxTimeout:  1 * time.Second,
			expectError: false,
		},
		{
			name: "Additional Apps",
			mainApp: &MockApp{
				RunFunc: func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
				StopFunc: func(ctx context.Context) error {
					return nil
				},
			},
			config: apprunner.AppRunnerConfig{
				MetricsServerConfig: app.MetricsServerAppConfig{
					Enabled: false,
				},
				AdditionalApps: []app.App{
					&MockApp{
						RunFunc: func(ctx context.Context) error {
							<-ctx.Done()
							return nil
						},
						StopFunc: func(ctx context.Context) error {
							return nil
						},
					},
				},
				ExitWait: 5 * time.Second,
			},
			ctxTimeout:  1 * time.Second,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := apprunner.NewAppRunner(tt.mainApp, tt.config)
			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			defer cancel()

			err := runner.Run(ctx)
			if tt.expectError {
				if err == nil {
					t.Error("Expected an error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}

		})
	}
}
