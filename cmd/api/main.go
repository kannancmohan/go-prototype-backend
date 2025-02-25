package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/app"
	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/apprunner"
)

func main() {

	testApp := NewTestApp(9933)
	runner := apprunner.NewAppRunner(testApp, apprunner.AppRunnerConfig{
		ExitWait: 5 * time.Second,
		MetricsServerConfig: app.MetricsServerAppConfig{
			Enabled: true,
		},
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	log.Print("\u001B[1;32mStarting App\u001B[0m")

	err := runner.Run(ctx)
	if err != nil {
		panic(fmt.Errorf("error running apprunner: %w", err))
	}
}

func NewTestApp(port int) *testApp {
	return &testApp{
		port:            port,
		shutdownTimeout: 5 * time.Second,
	}
}

type testApp struct {
	port            int
	shutdownTimeout time.Duration
	server          *http.Server
	mu              sync.Mutex
}

func (t *testApp) Stop(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.server == nil {
		return nil // Server was never started
	}

	slog.Debug("stopping test server gracefully")
	shutdownCtx, cancel := context.WithTimeout(ctx, t.shutdownTimeout)
	defer cancel()

	if err := t.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to stop test server: %w", err)
	}
	slog.Info("test server stopped")
	return nil
}

func (t *testApp) Run(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "main-handler: %s\n", r.URL.Query().Get("name"))
	}))
	t.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", t.port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		slog.Info(fmt.Sprintf("test server started on port %d", t.port))
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("test server failed: %w", err)
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		//TODO check whether we need to close the server . check how metrics server is stopped
	}
	return nil
}
