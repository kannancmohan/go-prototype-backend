package main

import (
	"context"
	"fmt"
	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/operator"
	"log"
	"os"
	"os/signal"
)

func main() {
	kubeConfig := app.RestConfig{}
	runner, err := operator.NewOperator(operator.OperatorConfig{
		KubeConfig: kubeConfig,
		MetricsConfig: operator.OperatorMetricsConfig{
			Enabled: true,
		},
	})
	if err != nil {
		panic(fmt.Errorf("unable to create runner: %w", err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	log.Print("\u001B[1;32mStarting App\u001B[0m")

	appCfg := struct {
		Name string
		Type string
	}{
		Name: "Sammy",
		Type: "Shark",
	}

	err = runner.Run(ctx, app.NewSimpleAppProvider(appCfg, NewApp))
	if err != nil {
		panic(fmt.Errorf("error running operator: %w", err))
	}
}

func NewApp(config app.AppConfig) (app.App, error) {

	return app.NewSimpleApp(app.SimpleAppConfig{
		Name:       "simple-reconciler-app",
		KubeConfig: config.KubeConfig,
	})
}
