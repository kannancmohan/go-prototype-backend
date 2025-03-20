package apprunner_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kannancmohan/go-prototype-backend/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend/cmd/internal/apprunner"
)

// mockApp is a mock implementation of the App interface for testing.

var _ app.ConfigSetter[any] = &mockApp{}

type mockApp struct {
	appConf  app.Conf[any]
	RunFunc  func(ctx context.Context) error
	StopFunc func(ctx context.Context) error
}

func (m *mockApp) Run(ctx context.Context) error {
	return m.RunFunc(ctx)
}

func (m *mockApp) Stop(ctx context.Context) error {
	return m.StopFunc(ctx)
}

func (m *mockApp) SetAppConf(conf app.Conf[any]) {
	m.appConf = conf
}

func TestAppRunner_Run(t *testing.T) {
	tests := []struct {
		name        string
		mainApp     *mockApp
		config      []apprunner.AppRunnerOption
		ctxTimeout  time.Duration
		expectError bool
		errorMsg    string
	}{
		{
			name: "Successful Run",
			mainApp: &mockApp{
				RunFunc: func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
				StopFunc: func(_ context.Context) error {
					return nil
				},
			},
			config:      []apprunner.AppRunnerOption{},
			ctxTimeout:  1 * time.Second,
			expectError: false,
		},
		{
			name: "App Failure",
			mainApp: &mockApp{
				RunFunc: func(ctx context.Context) error {
					return errors.New("mock app failed")
				},
				StopFunc: func(_ context.Context) error {
					return nil
				},
			},
			config:      []apprunner.AppRunnerOption{},
			ctxTimeout:  1 * time.Second,
			expectError: true,
			errorMsg:    "mock app failed",
		},
		{
			name: "Metrics Server Enabled",
			mainApp: &mockApp{
				RunFunc: func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
				StopFunc: func(_ context.Context) error {
					return nil
				},
			},
			config:      []apprunner.AppRunnerOption{apprunner.WithMetricsApp(app.NewMetricsServerApp())},
			ctxTimeout:  1 * time.Second,
			expectError: false,
		},
		{
			name: "Additional Apps",
			mainApp: &mockApp{
				RunFunc: func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
				StopFunc: func(_ context.Context) error {
					return nil
				},
			},
			config: []apprunner.AppRunnerOption{apprunner.WithAdditionalApps(
				[]app.App{
					&mockApp{
						RunFunc: func(ctx context.Context) error {
							<-ctx.Done()
							return nil
						},
						StopFunc: func(ctx context.Context) error {
							return nil
						},
					},
				},
			)},
			ctxTimeout:  1 * time.Second,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, _ := apprunner.NewAppRunner(tt.mainApp, app.EmptyAppConf, tt.config...)
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

func TestAppRunner_WithAppConf(t *testing.T) {
	tests := []struct {
		name            string
		mainApp         *mockApp
		config          []apprunner.AppRunnerOption
		inputAppConf    app.Conf[any]
		ctxTimeout      time.Duration
		expectedAppConf app.Conf[any]
	}{
		{
			name: "Successful Run",
			mainApp: &mockApp{
				RunFunc: func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
				StopFunc: func(_ context.Context) error {
					return nil
				},
			},
			config:          []apprunner.AppRunnerOption{},
			inputAppConf:    app.Conf[any]{Name: "test", EnvVar: struct{ EnvVarName1 string }{EnvVarName1: "EnvVarName1"}},
			ctxTimeout:      1 * time.Second,
			expectedAppConf: app.Conf[any]{Name: "test", EnvVar: struct{ EnvVarName1 string }{EnvVarName1: "EnvVarName1"}},
		},
		{
			name: "Successful Run - With empty AppConf",
			mainApp: &mockApp{
				RunFunc: func(ctx context.Context) error {
					<-ctx.Done()
					return nil
				},
				StopFunc: func(_ context.Context) error {
					return nil
				},
			},
			config:          []apprunner.AppRunnerOption{},
			inputAppConf:    app.Conf[any]{},
			ctxTimeout:      1 * time.Second,
			expectedAppConf: app.Conf[any]{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, _ := apprunner.NewAppRunner(tt.mainApp, tt.inputAppConf, tt.config...)
			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			defer cancel()

			err := runner.Run(ctx)
			if !reflect.DeepEqual(tt.expectedAppConf, tt.mainApp.appConf) {
				t.Errorf("expected AppConf %q, got %q", tt.expectedAppConf, tt.mainApp.appConf)
			} else if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}
