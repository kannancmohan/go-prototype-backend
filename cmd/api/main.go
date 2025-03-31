package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/kannancmohan/go-prototype-backend/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend/cmd/internal/apprunner"
	log_impl "github.com/kannancmohan/go-prototype-backend/cmd/internal/common/log"
	app_trace "github.com/kannancmohan/go-prototype-backend/cmd/internal/common/trace"
)

var (
	version = "dev"     // Overwritten by ldflags.
	commit  = "unknown" // Overwritten by ldflags.
)

func main() {
	log.Printf("app v%s (commit: %s)\n", version, commit)
	appConf, err := app.NewAppConf("simple-app", simpleAppEnvVar{})
	if err != nil {
		panic(fmt.Errorf("error init AppConf: %w", err))
	}
	logger := log_impl.NewSimpleSlogLogger(log_impl.INFO, nil, log_impl.NewTraceIDHandler)

	tp, shutdown, err := app_trace.NewOTelTracerProvider(app_trace.OpenTelemetryConfig{
		Host:        "localhost",
		Port:        3200,
		ConnType:    app_trace.OTelConnTypeHTTP,
		ServiceName: appConf.Name,
	})
	if err != nil {
		panic(fmt.Errorf("error creating otel tracer provider: %w", err))
	}
	defer func() {
		shtErr := shutdown(context.Background())
		if shtErr != nil {
			logger.Error("failed to shutdown TracerProvider", "error", shtErr.Error())
		}
	}()

	runner, err := apprunner.NewAppRunner(
		NewSimpleApp(9933),
		appConf,
		apprunner.WithLogger(logger),
		apprunner.WithMetricsApp(app.NewMetricsServerApp()),
		apprunner.WithTracerProvider(tp),
	)
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
