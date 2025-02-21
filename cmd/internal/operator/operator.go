package operator

import (
	"context"
	"errors"
	"io/fs"
	"sync"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/app"
)

type OperatorConfig struct {
	MetricsConfig OperatorMetricsConfig
	KubeConfig    app.RestConfig
	// Filesystem is an fs.FS that can be used in lieu of the OS filesystem.
	// if empty, it defaults to os.DirFS(".")
	Filesystem fs.FS
}

type OperatorMetricsConfig struct {
	app.MetricsExporterConfig
	Enabled   bool
	Namespace string
}

type Operator struct {
	config        OperatorConfig
	metricsServer *app.MetricsAddonAppRunner
	startMux      sync.Mutex
	running       bool
	runningWG     sync.WaitGroup
}

// NewRunner creates a new, properly-initialized instance of a Runner
func NewOperator(cfg OperatorConfig) (*Operator, error) {
	op := Operator{
		config: cfg,
	}
	if cfg.MetricsConfig.Enabled {
		op.metricsServer = app.NewMetricsAddonAppRunner(cfg.MetricsConfig.MetricsExporterConfig)
	}
	return &op, nil
}

func (s *Operator) Run(ctx context.Context, a app.App) error {
	if a == nil {
		return errors.New("app cannot be nil")
	}

	s.runningWG.Add(1)
	defer s.runningWG.Done()

	err := func() error {
		s.startMux.Lock()
		defer s.startMux.Unlock()
		if !s.running {
			s.running = true
			go func() {
				s.runningWG.Wait()
				s.running = false
			}()
		}
		return nil
	}()
	if err != nil {
		return err
	}

	// Build the operator
	runner := app.NewMultiRunner()

	// Main loop
	r := a.Runner()
	if r != nil {
		runner.AddRunnable(r)
	}

	// Metrics
	if s.metricsServer != nil {
		err = s.metricsServer.RegisterCollectors(runner.PrometheusCollectors()...)
		if err != nil {
			return err
		}
		runner.AddRunnable(s.metricsServer)
	}

	return runner.Run(ctx)
}
