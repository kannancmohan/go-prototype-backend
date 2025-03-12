package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

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
	logger := log_impl.NewSimpleSlogLogger(log_impl.INFO, nil, log_impl.NewTraceIDHandler)

	tp, shutdown, err := app_trace.NewOTelTracerProvider(app_trace.OpenTelemetryConfig{Host: "localhost",
		Port:        3200,
		ConnType:    app_trace.OTelConnTypeHTTP,
		ServiceName: appConf.Name,
	})
	if err != nil {
		panic(fmt.Errorf("error creating otel tracer provider: %w", err))
	}
	defer shutdown(context.Background())

	appRunnerConf := apprunner.NewAppRunnerConfig(
		apprunner.WithLogger(logger),
		apprunner.WithMetricsApp(app.NewMetricsServerApp()),
		apprunner.WithTracerProvider(tp),
	)
	runner, err := apprunner.NewAppRunner(NewSimpleApp(9933), appRunnerConf, appConf)
	if err != nil {
		panic(fmt.Errorf("error creating apprunner: %w", err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	err = runner.Run(ctx)
	if err != nil {
		panic(fmt.Errorf("error running apprunner: %w", err))
	}
}
