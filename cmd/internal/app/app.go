package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type AppSpecificConfig any

type ContentConfig struct {
	// AcceptContentTypes specifies the types client will accept and is optional. If not set, ContentType will be used to define the Accept header
	AcceptContentTypes string
	// ContentType specifies the format used to communicate with the server. If not set,"application/json" is used.
	ContentType string
}

// common rest specific config
type RestConfig struct {
	// Host must be a host string, a host:port pair, or a URL to the base of the apiserver.
	// If a URL is given then the (optional) Path of that URL represents a prefix that must
	// be appended to all request URIs used to access the apiserver. This allows a frontend
	// proxy to easily relocate all of the apiserver endpoints.
	Host string
	// APIPath is a sub-path that points to an API root.
	APIPath string

	// ContentConfig contains settings that affect how objects are transformed when
	// sent to the server.
	ContentConfig
}

type AppConfig struct {
	// KubeConfig is a kubernetes rest.Config used to communicate with the API server where the App's Kinds are stored.
	KubeConfig RestConfig
	// // ManifestData is the fetched ManifestData the runner is using for determining app kinds and capabilities.
	// ManifestData ManifestData
	// SpecificConfig is app-specific config (as opposed to generic config)
	AppSpecificConfig AppSpecificConfig
}

type AppProvider interface {
	AppSpecificConfig() AppSpecificConfig
	NewApp(AppConfig) (App, error)
}

var _ AppProvider = &SimpleAppProvider{}

func NewSimpleAppProvider(cfg AppSpecificConfig, newAppFunc func(cfg AppConfig) (App, error)) *SimpleAppProvider {
	return &SimpleAppProvider{
		SpecificConfig: cfg,
		NewAppFunc:     newAppFunc,
	}
}

type SimpleAppProvider struct {
	SpecificConfig AppSpecificConfig
	NewAppFunc     func(config AppConfig) (App, error)
}

func (p *SimpleAppProvider) AppSpecificConfig() AppSpecificConfig {
	return p.AppSpecificConfig
}

func (p *SimpleAppProvider) NewApp(settings AppConfig) (App, error) {
	return p.NewAppFunc(settings)
}

type App interface {
	// // Validate validates the incoming request, and returns an error if validation fails
	// Validate(ctx context.Context, request *AdmissionRequest) error
	// // Mutate runs mutation on the incoming request, responding with a MutatingResponse on success, or an error on failure
	// Mutate(ctx context.Context, request *AdmissionRequest) (*MutatingResponse, error)
	// // Convert converts the object based on the ConversionRequest, returning a RawObject which MUST contain
	// // the converted bytes and encoding (Raw and Encoding respectively), and MAY contain the Object representation of those bytes.
	// // It returns an error if the conversion fails, or if the functionality is not supported by the app.
	// Convert(ctx context.Context, req ConversionRequest) (*RawObject, error)
	// // CallResourceCustomRoute handles the call to a resource custom route, and returns a response to the request or an error.
	// // If the route doesn't exist, the implementer MAY return ErrCustomRouteNotFound to signal to the runner,
	// // or may choose to return a response with a not found status code and custom body.
	// // It returns an error if the functionality is not supported by the app.
	// CallResourceCustomRoute(ctx context.Context, request *ResourceCustomRouteRequest) (*ResourceCustomRouteResponse, error)
	// // ManagedKinds returns a slice of Kinds which are managed by this App.
	// // If there are multiple versions of a Kind, each one SHOULD be returned by this method,
	// // as app runners may depend on having access to all kinds.
	// ManagedKinds() []resource.Kind
	// // Runner returns a Runnable with an app main loop. Any business logic that is not/can not be exposed
	// // via other App interfaces should be contained within this method.
	// // Runnable MAY be nil, in which case, the app has no main loop business logic.
	Runner() Runnable
}

type SimpleAppConfig struct {
	Name       string
	KubeConfig RestConfig
	// InformerConfig AppInformerConfig
	// ManagedKinds   []AppManagedKind
	// UnmanagedKinds []AppUnmanagedKind
	// Converters     map[schema.GroupKind]Converter
	// // DiscoveryRefreshInterval is the interval at which the API discovery cache should be refreshed.
	// // This is primarily used by the DynamicPatcher in the OpinionatedWatcher/OpinionatedReconciler
	// // for sending finalizer add/remove patches to the latest version of the kind.
	// // This defaults to 10 minutes.
	// DiscoveryRefreshInterval time.Duration
}

var _ App = &SimpleApp{}
var _ MetricsProvider = &SimpleApp{}

func NewSimpleApp(config SimpleAppConfig) (*SimpleApp, error) {
	a := &SimpleApp{
		//informerController: operator.NewInformerController(operator.DefaultInformerControllerConfig()),
		runner: NewMultiRunner(),
		// clientGenerator:    k8s.NewClientRegistry(config.KubeConfig, k8s.DefaultClientConfig()),
		// kinds:              make(map[string]AppManagedKind),
		// internalKinds:      make(map[string]resource.Kind),
		// converters:         make(map[string]Converter),
		// customRoutes:       make(map[string]AppCustomRouteHandler),
		cfg:        config,
		collectors: make([]prometheus.Collector, 0),
	}
	// if config.InformerConfig.ErrorHandler != nil {
	// 	a.informerController.ErrorHandler = config.InformerConfig.ErrorHandler
	// }
	// discoveryRefresh := config.DiscoveryRefreshInterval
	// if discoveryRefresh == 0 {
	// 	discoveryRefresh = time.Minute * 10
	// }
	// p, err := k8s.NewDynamicPatcher(&config.KubeConfig, discoveryRefresh)
	// if err != nil {
	// 	return nil, err
	// }
	// a.patcher = p
	// for _, kind := range config.ManagedKinds {
	// 	err := a.manageKind(kind)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }
	// for _, kind := range config.UnmanagedKinds {
	// 	err := a.watchKind(kind)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }
	// for gk, converter := range config.Converters {
	// 	a.RegisterKindConverter(gk, converter)
	// }
	// a.runner.AddRunnable(a.informerController)
	return a, nil
}

type SimpleApp struct {
	//informerController *operator.InformerController
	runner *MultiRunner
	//clientGenerator    resource.ClientGenerator
	//kinds              map[string]AppManagedKind
	//internalKinds      map[string]resource.Kind
	cfg SimpleAppConfig
	// converters         map[string]Converter
	// customRoutes       map[string]AppCustomRouteHandler
	// patcher            *k8s.DynamicPatcher
	collectors []prometheus.Collector
}

func (a *SimpleApp) Runner() Runnable {
	return a.runner
}

func (a *SimpleApp) PrometheusCollectors() []prometheus.Collector {
	collectors := make([]prometheus.Collector, 0)
	collectors = append(collectors, a.collectors...)
	collectors = append(collectors, a.runner.PrometheusCollectors()...)
	return collectors
}

// a process that the app runs
type Runnable interface {
	// Run runs the process and blocks until context completes/error/process completes
	Run(context.Context) error
}

var _ Runnable = &MultiRunner{}
var _ MetricsProvider = &MultiRunner{}

func NewMultiRunner() *MultiRunner {
	return &MultiRunner{
		Runners:      make([]Runnable, 0),
		ErrorHandler: RunnableCollectorDefaultErrorHandler,
	}
}

type MultiRunner struct {
	Runners []Runnable
	// ErrorHandler is called if one of the Runners returns an error. If the function call returns true,
	// the context will be canceled and all other Runners will also be prompted to exit.
	// If ErrorHandler is nil, RunnableCollectorDefaultErrorHandler is used.
	ErrorHandler func(context.Context, error) bool
	// ExitWait is how long to wait for Runners to exit after ErrorHandler returns true or the context is canceled
	// before stopping execution and returning a timeout error instead of exiting gracefully.
	// If ExitWait is nil, Run execution will always block until all Runners have exited.
	ExitWait *time.Duration
}

// Run runs all Runners in separate goroutines, and calls ErrorHandler if any of them exits early with an error.
// If ErrorHandler returns true (or if there is no ErrorHandler), the other Runners are canceled and the error is returned.
func (m *MultiRunner) Run(ctx context.Context) error {
	propagatedContext, cancel := context.WithCancel(ctx)
	defer cancel()
	errs := make(chan error, len(m.Runners))
	errsClosed := false
	defer func() {
		errsClosed = true
		close(errs)
	}()
	wg := &sync.WaitGroup{}
	timedOut := false
	for _, runner := range m.Runners {
		wg.Add(1)
		go func(r Runnable) {
			err := r.Run(propagatedContext)
			wg.Done()
			if err != nil && !timedOut {
				if errsClosed {
					//logging.DefaultLogger.Warn("MultiRunner runner error encountered, but MultiRunner already completed", "error", err)
					slog.Warn("MultiRunner runner error encountered, but MultiRunner already completed", "error", err)
					return
				}
				errs <- err
			}
		}(runner)
	}
	for {
		select {
		case err := <-errs:
			handler := m.ErrorHandler
			if handler == nil {
				handler = RunnableCollectorDefaultErrorHandler
			}
			if handler(propagatedContext, err) {
				cancel()
				if m.ExitWait != nil {
					if waitOrTimeout(wg, *m.ExitWait) {
						timedOut = true
						return errors.Join(ErrRunnerExitTimeout, err)
					}
				} else {
					wg.Wait() // Wait for all the runners to stop
				}
				return err
			}
		case <-ctx.Done():
			cancel()
			if m.ExitWait != nil {
				if waitOrTimeout(wg, *m.ExitWait) {
					timedOut = true
					return ErrRunnerExitTimeout
				}
			} else {
				wg.Wait() // Wait for all the runners to stop
			}
			return nil
		}
	}
}

func (m *MultiRunner) PrometheusCollectors() []prometheus.Collector {
	collectors := make([]prometheus.Collector, 0)
	for _, runner := range m.Runners {
		if cast, ok := runner.(MetricsProvider); ok {
			collectors = append(collectors, cast.PrometheusCollectors()...)
		}
	}
	return collectors
}

func (m *MultiRunner) AddRunnable(runnable Runnable) {
	if m.Runners == nil {
		m.Runners = make([]Runnable, 0)
	}
	m.Runners = append(m.Runners, runnable)
}

var _ Runnable = &SingletonRunner{}
var _ MetricsProvider = &SingletonRunner{}

func NewSingletonRunner(runnable Runnable, stopOnAny bool) *SingletonRunner {
	return &SingletonRunner{
		Wrapped:   runnable,
		StopOnAny: stopOnAny,
	}
}

// SingletonRunner runs a single Runnable but allows for multiple distinct calls to Run() which can have independent lifecycles
type SingletonRunner struct {
	Wrapped   Runnable
	StopOnAny bool
	mux       sync.Mutex
	running   bool
	wg        sync.WaitGroup
	cancel    context.CancelCauseFunc
	ctx       context.Context
}

// Run runs until the provided context.Context is closed, the underlying Runnable completes, or
// another call to Run is stopped and StopOnAny is set to true (in which case ErrOtherRunStopped is returned)
func (s *SingletonRunner) Run(ctx context.Context) error {
	s.wg.Add(1)
	defer s.wg.Done()
	go func(c context.Context) {
		<-c.Done()
		if s.StopOnAny && s.cancel != nil {
			s.cancel(ErrOtherRunStopped)
		}
	}(ctx)

	func() {
		s.mux.Lock()
		defer s.mux.Unlock()
		if !s.running {
			s.running = true
			// Stop cancel propagation and set up our own cancel function
			derived := context.WithoutCancel(ctx)
			s.ctx, s.cancel = context.WithCancelCause(derived)
			go func() {
				s.wg.Wait()
				s.mux.Lock()
				s.running = false
				s.mux.Unlock()
			}()

			go func() {
				err := s.Wrapped.Run(s.ctx)
				s.cancel(err)
			}()
		}
	}()

	select {
	case <-s.ctx.Done():
		return context.Cause(s.ctx)
	case <-ctx.Done():
	}
	return nil
}

// PrometheusCollectors implements metrics.Provider by returning prometheus collectors for the wrapped Runnable if it implements metrics.Provider.
func (s *SingletonRunner) PrometheusCollectors() []prometheus.Collector {
	if cast, ok := s.Wrapped.(MetricsProvider); ok {
		return cast.PrometheusCollectors()
	}
	return nil
}

var RunnableCollectorDefaultErrorHandler = func(ctx context.Context, err error) bool {
	//logging.FromContext(ctx).Error("runner exited with error", "error", err)
	slog.Error("runner exited with error", "error", err)
	return true
}

var (
	ErrOtherRunStopped   = errors.New("run stopped by another run call")
	ErrRunnerExitTimeout = fmt.Errorf("exit wait time exceeded waiting for Runners to complete")
)

func waitOrTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		wg.Wait()
	}()
	select {
	case <-ch:
		return false
	case <-time.After(timeout):
		return true
	}
}
