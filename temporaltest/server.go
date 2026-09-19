// Package temporaltest provides utilities for end to end Temporal server testing.
package temporaltest

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/temporal"
	temporalite "go.temporal.io/server/temporaltest/internal"
)

// A TestServer is a Temporal server listening on a system-chosen port on the
// local loopback interface, for use in end-to-end tests.
//
// Methods on TestServer are not safe for concurrent use.
type TestServer struct {
	server                     liteServer
	serverResource             *ownedLifecycleResource
	defaultTestNamespace       string
	defaultClient              client.Client
	clients                    []ownedClient
	workers                    []ownedWorker
	t                          *testing.T
	defaultClientOptions       client.Options
	defaultWorkerOptions       worker.Options
	serverOptions              []temporal.ServerOption
	frontendHTTPPort           int
	lifecycleBackend           lifecycleBackend
	clientSequence             int
	cleanupStarted             bool
	startupFailureCleanupLimit time.Duration
}

func (ts *TestServer) fatal(err error) {
	if ts.t == nil {
		panic(err)
	}
	ts.t.Fatal(err)
}

// NewWorker registers and starts a Temporal worker on the specified task queue.
func (ts *TestServer) NewWorker(taskQueue string, registerFunc func(registry worker.Registry)) worker.Worker {
	ctx, cancel := context.WithTimeout(context.Background(), defaultClientStartupTimeout)
	defer cancel()
	worker, err := ts.NewWorkerWithContext(ctx, taskQueue, registerFunc)
	if err != nil {
		ts.fatal(err)
	}
	return worker
}

// NewWorkerWithContext registers and starts a Temporal worker on the specified
// task queue, returning lifecycle errors instead of terminating the caller.
func (ts *TestServer) NewWorkerWithContext(
	ctx context.Context,
	taskQueue string,
	registerFunc func(registry worker.Registry),
) (worker.Worker, error) {
	return ts.NewWorkerWithOptionsContext(ctx, taskQueue, registerFunc, ts.defaultWorkerOptions)
}

// NewWorkerWithOptions returns a Temporal worker on the specified task queue.
//
// WorkflowPanicPolicy is always set to worker.FailWorkflow so that workflow executions
// fail fast when workflow code panics or detects non-determinism.
// WorkerStopTimeout defaults to 10 seconds when it is not positive.
func (ts *TestServer) NewWorkerWithOptions(taskQueue string, registerFunc func(registry worker.Registry), opts worker.Options) worker.Worker {
	ctx, cancel := context.WithTimeout(context.Background(), defaultClientStartupTimeout)
	defer cancel()
	worker, err := ts.NewWorkerWithOptionsContext(ctx, taskQueue, registerFunc, opts)
	if err != nil {
		ts.fatal(err)
	}
	return worker
}

// NewWorkerWithOptionsContext returns a Temporal worker on the specified task
// queue, returning lifecycle errors instead of terminating the caller.
//
// WorkflowPanicPolicy is always set to worker.FailWorkflow so that workflow executions
// fail fast when workflow code panics or detects non-determinism.
// WorkerStopTimeout defaults to 10 seconds when it is not positive.
func (ts *TestServer) NewWorkerWithOptionsContext(
	ctx context.Context,
	taskQueue string,
	registerFunc func(registry worker.Registry),
	opts worker.Options,
) (worker.Worker, error) {
	if ctx == nil {
		return nil, &LifecycleError{Operation: LifecycleOperationCreateWorker, Err: errors.New("nil context")}
	}
	if err := ts.acquisitionAllowed(LifecycleOperationCreateWorker); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, ts.startupError(LifecycleOperationCreateWorker, err)
	}
	opts.WorkflowPanicPolicy = worker.FailWorkflow
	if opts.WorkerStopTimeout <= 0 {
		opts.WorkerStopTimeout = defaultWorkerStopTimeout
	}

	temporalClient, err := ts.GetDefaultClientWithContext(ctx)
	if err != nil {
		return nil, err
	}
	temporalWorker, err := callLifecycle(func() (worker.Worker, error) {
		return ts.lifecycleBackend.createWorker(temporalClient, taskQueue, opts)
	})
	var resource *ownedLifecycleResource
	if temporalWorker != nil {
		resource = ts.ownWorker(taskQueue, temporalWorker)
	}
	if err != nil {
		return temporalWorker, ts.startupError(LifecycleOperationCreateWorker, err)
	}
	if temporalWorker == nil {
		return nil, ts.startupError(LifecycleOperationCreateWorker, errors.New("worker construction returned nil"))
	}

	if err := resource.acquisitionOperation(ctx, func(context.Context) error {
		return callLifecycleError(func() error {
			return ts.lifecycleBackend.registerWorker(temporalWorker, registerFunc)
		})
	}); err != nil {
		return temporalWorker, ts.startupError(LifecycleOperationRegisterWorker, err)
	}

	if err := resource.acquisitionOperation(ctx, func(startCtx context.Context) error {
		return ts.lifecycleBackend.startWorker(startCtx, temporalWorker)
	}); err != nil {
		return temporalWorker, ts.startupError(LifecycleOperationStartWorker, err)
	}

	return temporalWorker, nil
}

// GetDefaultClient returns the default Temporal client configured for making requests to the server.
//
// It is configured to use a pre-registered test namespace and will be closed on TestServer.Stop.
func (ts *TestServer) GetDefaultClient() client.Client {
	ctx, cancel := context.WithTimeout(context.Background(), defaultClientStartupTimeout)
	defer cancel()
	client, err := ts.GetDefaultClientWithContext(ctx)
	if err != nil {
		ts.fatal(err)
	}
	return client
}

// GetDefaultClientWithContext returns the default Temporal client configured
// for making requests to the server, returning lifecycle errors instead of
// terminating the caller.
//
// It is configured to use a pre-registered test namespace and will be closed on TestServer.Stop.
func (ts *TestServer) GetDefaultClientWithContext(ctx context.Context) (client.Client, error) {
	if ctx == nil {
		return nil, &LifecycleError{Operation: LifecycleOperationCreateClient, Err: errors.New("nil context")}
	}
	if err := ts.acquisitionAllowed(LifecycleOperationCreateClient); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, ts.startupError(LifecycleOperationCreateClient, err)
	}
	if ts.defaultClient == nil {
		defaultClient, err := ts.NewClientWithOptionsContext(ctx, ts.defaultClientOptions)
		if err != nil {
			return defaultClient, err
		}
		ts.defaultClient = defaultClient
	}
	return ts.defaultClient, nil
}

// GetDefaultNamespace returns the randomly generated namespace which has been pre-registered with the test server.
func (ts *TestServer) GetDefaultNamespace() string {
	return ts.defaultTestNamespace
}

// GetFrontendHostPort returns the host:port for this server.
//
// When constructing a Temporal client from within the same process,
// GetDefaultClient or NewClientWithOptions should be used instead.
func (ts *TestServer) GetFrontendHostPort() string {
	return ts.server.FrontendHostPort()
}

// NewClientWithOptions returns a new Temporal client configured for making requests to the server.
//
// If no namespace option is set it will use a pre-registered test namespace.
// The returned client will be closed on TestServer.Stop.
func (ts *TestServer) NewClientWithOptions(opts client.Options) client.Client {
	ctx, cancel := context.WithTimeout(context.Background(), defaultClientStartupTimeout)
	defer cancel()
	client, err := ts.NewClientWithOptionsContext(ctx, opts)
	if err != nil {
		ts.fatal(err)
	}
	return client
}

// NewClientWithOptionsContext returns a new Temporal client configured for
// making requests to the server, returning lifecycle errors instead of
// terminating the caller.
//
// If no namespace option is set it will use a pre-registered test namespace.
// The returned client will be closed on TestServer.Stop.
func (ts *TestServer) NewClientWithOptionsContext(ctx context.Context, opts client.Options) (client.Client, error) {
	if ctx == nil {
		return nil, &LifecycleError{Operation: LifecycleOperationCreateClient, Err: errors.New("nil context")}
	}
	if err := ts.acquisitionAllowed(LifecycleOperationCreateClient); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, ts.startupError(LifecycleOperationCreateClient, err)
	}
	if opts.Namespace == "" {
		opts.Namespace = ts.defaultTestNamespace
	}
	if opts.Logger == nil {
		opts.Logger = &testLogger{ts.t}
	}

	temporalClient, err := callLifecycle(func() (client.Client, error) {
		return ts.lifecycleBackend.createClient(ctx, ts.server, opts)
	})
	if temporalClient != nil {
		ts.ownClient(temporalClient)
	}
	if err != nil {
		return temporalClient, ts.startupError(LifecycleOperationCreateClient, fmt.Errorf("error creating client: %w", err))
	}
	if temporalClient == nil {
		return nil, ts.startupError(LifecycleOperationCreateClient, errors.New("client construction returned nil"))
	}
	if err := ctx.Err(); err != nil {
		return temporalClient, ts.startupError(LifecycleOperationCreateClient, err)
	}

	return temporalClient, nil
}

// Stop closes test clients and shuts down the server.
func (ts *TestServer) Stop() {
	if err := ts.StopContext(context.Background()); err != nil {
		// Log instead of throwing error because there's no need to fail the test
		// if it already succeeded.
		if ts.t != nil {
			ts.t.Logf("error shutting down Temporal server: %s", err)
			return
		}
		ts.fatal(err)
	}
}

// NewServer starts and returns a new TestServer.
//
// If not specifying the WithT option, the caller should execute Stop when finished to close
// the server and release resources.
func NewServer(opts ...TestServerOption) *TestServer {
	server, err := NewServerWithContext(context.Background(), opts...)
	if err != nil {
		server.fatal(err)
	}
	return server
}

// NewServerWithContext starts and returns a new TestServer, returning a partial
// owned handle with any startup error so callers can inspect or retry cleanup.
//
// If not specifying the WithT option, the caller should execute StopContext
// when finished to close the server and release resources.
func NewServerWithContext(ctx context.Context, opts ...TestServerOption) (*TestServer, error) {
	testNamespace := fmt.Sprintf("temporaltest-%d", rand.Intn(1e6))

	ts := TestServer{
		defaultTestNamespace:       testNamespace,
		lifecycleBackend:           defaultLifecycleBackend(),
		startupFailureCleanupLimit: defaultStartupFailureCleanupLimit,
	}
	if ctx == nil {
		return &ts, &LifecycleError{Operation: LifecycleOperationCreateServer, Err: errors.New("nil context")}
	}

	// Apply options
	for _, opt := range opts {
		if err := callLifecycleError(func() error {
			opt.apply(&ts)
			return nil
		}); err != nil {
			return &ts, ts.startupError(LifecycleOperationApplyOptions, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return &ts, &LifecycleError{Operation: LifecycleOperationCreateServer, Err: err}
	}

	if ts.t != nil {
		ts.t.Cleanup(ts.Stop)
	}

	server, err := callLifecycle(func() (liteServer, error) {
		return ts.lifecycleBackend.createServer(&ts)
	})
	if server != nil {
		ts.ownServer(server)
	}
	if err != nil {
		return &ts, ts.startupError(LifecycleOperationCreateServer, fmt.Errorf("error creating server: %w", err))
	}
	if server == nil {
		return &ts, ts.startupError(LifecycleOperationCreateServer, errors.New("server construction returned nil"))
	}

	// Start does not block as long as InterruptOn is unset.
	if err := ts.serverResource.acquisitionOperation(ctx, func(startCtx context.Context) error {
		return ts.lifecycleBackend.startServer(startCtx, server)
	}); err != nil {
		return &ts, ts.startupError(LifecycleOperationStartServer, err)
	}

	// This sleep helps avoid a panic in github.com/temporalio/ringpop-go@v0.0.0-20230606200434-b5c079f412d3/swim/labels.go:175
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return &ts, nil
	case <-ctx.Done():
		return &ts, ts.startupError(LifecycleOperationServerReady, ctx.Err())
	}
}

func (ts *TestServer) newLiteServer() (*temporalite.LiteServer, error) {
	return temporalite.NewLiteServer(&temporalite.LiteServerConfig{
		Namespaces: []string{ts.defaultTestNamespace},
		Ephemeral:  true,
		Logger:     log.NewNoopLogger(),
		DynamicConfig: dynamicconfig.StaticClient{
			dynamicconfig.ForceSearchAttributesCacheRefreshOnRead.Key(): []dynamicconfig.ConstrainedValue{{Value: true}},
		},
		// Disable "accept incoming network connections?" prompt on macOS
		FrontendIP:       "127.0.0.1",
		FrontendHTTPPort: ts.frontendHTTPPort,
	}, ts.serverOptions...)
}
