package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/operator"
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

	app, err := NewTestApp()
	if err != nil {
		panic(fmt.Errorf("unable to create app: %w", err))
	}

	err = runner.Run(ctx, app)
	if err != nil {
		panic(fmt.Errorf("error running operator: %w", err))
	}
}

type testApp struct {
	port       int
	KubeConfig app.RestConfig
}

func (t *testApp) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "main-handler: %s\n", r.URL.Query().Get("name"))
	}))
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", t.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	return nil
}

func NewTestApp() (app.App, error) {

	testApp := &testApp{
		port:       9991,
		KubeConfig: app.RestConfig{},
	}

	app, err := app.NewSimpleApp(app.SimpleAppConfig{
		Name:       "simple-reconciler-app",
		KubeConfig: testApp.KubeConfig,
	})
	if err != nil {
		return app, err
	}
	app.AddRunnable(testApp)
	return app, err
}
