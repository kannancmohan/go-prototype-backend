package operator

import (
	"context"
	"errors"
	"io/fs"
	"sync"

	"github.com/kannancmohan/go-prototype-backend-apps-temp/cmd/internal/app"
)

type OperatorConfig struct {
	// WebhookConfig contains configuration information for exposing k8s webhooks.
	// This can be empty if your App does not implement ValidatorApp, MutatorApp, or ConversionApp
	WebhookConfig OperatorWebhookConfig
	// MetricsConfig contains the configuration for exposing prometheus metrics, if desired
	MetricsConfig OperatorMetricsConfig
	// KubeConfig is the kubernetes rest.Config to use when communicating with the API server
	KubeConfig app.RestConfig
	// Filesystem is an fs.FS that can be used in lieu of the OS filesystem.
	// if empty, it defaults to os.DirFS(".")
	Filesystem fs.FS
}

type OperatorMetricsConfig struct {
	app.MetricsExporterConfig
	Enabled   bool
	Namespace string
}

type OperatorWebhookConfig struct {
	// Port is the port to open the webhook server on
	Port int
	// TLSConfig is the TLS Cert and Key to use for the HTTPS endpoints exposed for webhooks
	//TLSConfig k8s.TLSConfig
}

type Operator struct {
	config OperatorConfig
	//webhookServer *webhookServerRunner
	metricsServer *app.MetricsAddonAppRunner
	startMux      sync.Mutex
	running       bool
	runningWG     sync.WaitGroup
}

// NewRunner creates a new, properly-initialized instance of a Runner
func NewOperator(cfg OperatorConfig) (*Operator, error) {
	// Validate the KubeConfig by constructing a rest.RESTClient with it
	// TODO: this requires a GroupVersion, which gets set up based on the kind
	// _, err := rest.RESTClientFor(&cfg.KubeConfig)
	// if err != nil {
	// 	return nil, fmt.Errorf("invalid KubeConfig: %w", err)
	// }

	op := Operator{
		config: cfg,
	}

	// if cfg.WebhookConfig.TLSConfig.CertPath != "" {
	// 	ws, err := k8s.NewWebhookServer(k8s.WebhookServerConfig{
	// 		Port: cfg.WebhookConfig.Port,
	// 		TLSConfig: k8s.TLSConfig{
	// 			CertPath: cfg.WebhookConfig.TLSConfig.CertPath,
	// 			KeyPath:  cfg.WebhookConfig.TLSConfig.KeyPath,
	// 		},
	// 	})
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	op.webhookServer = newWebhookServerRunner(ws)
	// }
	if cfg.MetricsConfig.Enabled {
		op.metricsServer = app.NewMetricsAddonAppRunner(cfg.MetricsConfig.MetricsExporterConfig)
	}
	return &op, nil
}

// Run runs the Operator for the app built from the provided app.AppProvider, until the provided context.Context is closed,
// or an unrecoverable error occurs. If an app.App cannot be instantiated from the app.AppProvider, an error will be returned.
// Webserver components of Run (such as webhooks and the prometheus exporter) will remain running so long as at least one Run() call is still active.
//
//nolint:funlen
func (s *Operator) Run(ctx context.Context, provider app.AppProvider) error {
	if provider == nil {
		return errors.New("provider cannot be nil")
	}

	// Get capabilities from manifest
	// manifestData, err := s.getManifestData(provider)
	// if err != nil {
	// 	return fmt.Errorf("unable to get app manifest capabilities: %w", err)
	// }
	appConfig := app.AppConfig{
		KubeConfig: s.config.KubeConfig,
		//ManifestData:   *manifestData,
		AppSpecificConfig: provider.AppSpecificConfig(),
	}

	// Create the app
	a, err := provider.NewApp(appConfig)
	if err != nil {
		return err
	}

	s.runningWG.Add(1)
	defer s.runningWG.Done()

	err = func() error {
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

	// Admission control
	//anyWebhooks := false
	// vkCapabilities := make(map[string]capabilities)
	// for _, kind := range manifestData.Kinds {
	// 	for _, version := range kind.Versions {
	// 		if version.Admission == nil {
	// 			if kind.Conversion {
	// 				anyWebhooks = true
	// 				vkCapabilities[fmt.Sprintf("%s/%s", kind.Kind, version.Name)] = capabilities{
	// 					conversion: kind.Conversion,
	// 				}
	// 			}
	// 			continue
	// 		}
	// 		vkCapabilities[fmt.Sprintf("%s/%s", kind.Kind, version.Name)] = capabilities{
	// 			conversion: kind.Conversion,
	// 			mutation:   version.Admission.SupportsAnyMutation(),
	// 			validation: version.Admission.SupportsAnyValidation(),
	// 		}
	// 		if kind.Conversion || version.Admission.SupportsAnyMutation() || version.Admission.SupportsAnyValidation() {
	// 			anyWebhooks = true
	// 		}
	// 	}
	// }

	// if anyWebhooks {
	// 	if s.webhookServer == nil {
	// 		return errors.New("app has capabilities that require webhooks, but webhook server was not provided TLS config")
	// 	}
	// 	for _, kind := range a.ManagedKinds() {
	// 		c, ok := vkCapabilities[fmt.Sprintf("%s/%s", kind.Kind(), kind.Version())]
	// 		if !ok {
	// 			continue
	// 		}
	// 		if c.validation {
	// 			s.webhookServer.AddValidatingAdmissionController(&resource.SimpleValidatingAdmissionController{
	// 				ValidateFunc: func(ctx context.Context, request *resource.AdmissionRequest) error {
	// 					return a.Validate(ctx, s.translateAdmissionRequest(request))
	// 				},
	// 			}, kind)
	// 		}
	// 		if c.mutation {
	// 			s.webhookServer.AddMutatingAdmissionController(&resource.SimpleMutatingAdmissionController{
	// 				MutateFunc: func(ctx context.Context, request *resource.AdmissionRequest) (*resource.MutatingResponse, error) {
	// 					resp, err := a.Mutate(ctx, s.translateAdmissionRequest(request))
	// 					return s.translateMutatingResponse(resp), err
	// 				},
	// 			}, kind)
	// 		}
	// 		if c.conversion {
	// 			s.webhookServer.AddConverter(toWebhookConverter(a), metav1.GroupKind{
	// 				Group: kind.Group(),
	// 				Kind:  kind.Kind(),
	// 			})
	// 		}
	// 	}
	// 	runner.AddRunnable(s.webhookServer)
	// }

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

// var _ app.Runnable = &metricsServerRunner{}

// func newMetricsServerRunner(exporter *app.MetricsExporter) *metricsServerRunner {
// 	return &metricsServerRunner{
// 		server: exporter,
// 		runner: app.NewSingletonRunner(&k8sRunnable{
// 			runner: exporter,
// 		}, false),
// 	}
// }

// type metricsServerRunner struct {
// 	runner *app.SingletonRunner
// 	server *app.MetricsExporter
// }

// func (m *metricsServerRunner) Run(ctx context.Context) error {
// 	return m.runner.Run(ctx)
// }

// func (m *metricsServerRunner) RegisterCollectors(collectors ...prometheus.Collector) error {
// 	return m.server.RegisterCollectors(collectors...)
// }

// type k8sRunner interface {
// 	Run(<-chan struct{}) error
// }

// type k8sRunnable struct {
// 	runner k8sRunner
// }

// func (k *k8sRunnable) Run(ctx context.Context) error {
// 	return k.runner.Run(ctx.Done())
// }
