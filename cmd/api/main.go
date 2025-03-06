package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/kannancmohan/go-prototype-backend/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend/cmd/internal/apprunner"
	log_impl "github.com/kannancmohan/go-prototype-backend/cmd/internal/common/log"
	app_trace "github.com/kannancmohan/go-prototype-backend/cmd/internal/common/trace"
)

func main() {

	appConf, err := app.NewAppConf("simple-app", simpleAppEnvVar{})
	if err != nil {
		panic(fmt.Errorf("error init AppConf: %w", err))
	}
	logger := log_impl.NewSimpleSlogLogger(log_impl.INFO)

	tp, shutdown, err := app_trace.NewOTelTracerProvider(app_trace.OpenTelemetryConfig{Host: "localhost",
		Port:        3200,
		ConnType:    app_trace.OTelConnTypeHTTP,
		ServiceName: appConf.Name,
	})
	if err != nil {
		panic(fmt.Errorf("error creating otel tracer provider: %w", err))
	}
	defer shutdown(context.Background())

	runner, err := apprunner.NewAppRunner(NewSimpleApp(9933),
		apprunner.AppRunnerConfig{
			ExitWait: 5 * time.Second,
			MetricsServerConfig: app.MetricsServerAppConfig{
				Enabled: true,
			},
			Logger: logger,
			TracingConfig: apprunner.TracingConfig{
				Enabled:        true,
				TracerProvider: tp,
			},
		},
		appConf)
	if err != nil {
		panic(fmt.Errorf("error creating apprunner: %w", err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	logger.Info("Starting App...")

	err = runner.Run(ctx)
	if err != nil {
		panic(fmt.Errorf("error running apprunner: %w", err))
	}
}
