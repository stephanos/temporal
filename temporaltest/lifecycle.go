package temporaltest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	temporalite "go.temporal.io/server/temporaltest/internal"
)

const (
	defaultWorkerStopTimeout   = 10 * time.Second
	startupFailureCleanupLimit = 15 * time.Second
)

// LifecycleResourceKind identifies a resource owned by a TestServer lifecycle.
type LifecycleResourceKind string

const (
	// LifecycleResourceWorker identifies an SDK worker.
	LifecycleResourceWorker LifecycleResourceKind = "worker"
	// LifecycleResourceClient identifies an SDK client.
	LifecycleResourceClient LifecycleResourceKind = "client"
	// LifecycleResourceServer identifies the embedded server.
	LifecycleResourceServer LifecycleResourceKind = "server"
)

// LifecycleResource identifies an owned resource that still requires cleanup.
type LifecycleResource struct {
	Kind LifecycleResourceKind
	Name string
}

// LifecycleOperation identifies a fallible lifecycle acquisition boundary.
type LifecycleOperation string

const (
	LifecycleOperationApplyOptions   LifecycleOperation = "apply-options"
	LifecycleOperationCreateServer   LifecycleOperation = "create-server"
	LifecycleOperationStartServer    LifecycleOperation = "start-server"
	LifecycleOperationServerReady    LifecycleOperation = "server-ready"
	LifecycleOperationCreateClient   LifecycleOperation = "create-client"
	LifecycleOperationCreateWorker   LifecycleOperation = "create-worker"
	LifecycleOperationRegisterWorker LifecycleOperation = "register-worker"
	LifecycleOperationStartWorker    LifecycleOperation = "start-worker"
)

// LifecycleError identifies the acquisition boundary that failed.
type LifecycleError struct {
	Operation LifecycleOperation
	Err       error
}

func (e *LifecycleError) Error() string {
	return fmt.Sprintf("temporaltest lifecycle %s: %v", e.Operation, e.Err)
}

// Unwrap returns the underlying acquisition error.
func (e *LifecycleError) Unwrap() error {
	return e.Err
}

// CleanupError reports why cleanup stopped and every resource still owned by
// the TestServer. A later call to StopContext waits for any release already in
// progress instead of starting it a second time.
type CleanupError struct {
	Remaining []LifecycleResource
	Err       error
}

func (e *CleanupError) Error() string {
	names := make([]string, 0, len(e.Remaining))
	for _, resource := range e.Remaining {
		names = append(names, fmt.Sprintf("%s %q", resource.Kind, resource.Name))
	}
	return fmt.Sprintf("temporaltest cleanup: %v; remaining: %s", e.Err, strings.Join(names, ", "))
}

// Unwrap returns the error that bounded cleanup.
func (e *CleanupError) Unwrap() error {
	return e.Err
}

type liteServer interface {
	StartContext(context.Context) error
	StopContext(context.Context) error
	NewClientWithOptions(context.Context, client.Options) (client.Client, error)
	FrontendHostPort() string
}

type lifecycleBackend struct {
	createServer   func(*TestServer) (liteServer, error)
	startServer    func(context.Context, liteServer) error
	stopServer     func(context.Context, liteServer) error
	createClient   func(context.Context, liteServer, client.Options) (client.Client, error)
	closeClient    func(context.Context, client.Client) error
	createWorker   func(client.Client, string, worker.Options) (worker.Worker, error)
	registerWorker func(worker.Registry, func(worker.Registry)) error
	startWorker    func(context.Context, worker.Worker) error
	stopWorker     func(context.Context, worker.Worker) error
}

func defaultLifecycleBackend() lifecycleBackend {
	return lifecycleBackend{
		createServer: func(server *TestServer) (liteServer, error) {
			return server.newLiteServer()
		},
		startServer: func(ctx context.Context, server liteServer) error {
			return server.StartContext(ctx)
		},
		stopServer: func(ctx context.Context, server liteServer) error {
			return server.StopContext(ctx)
		},
		createClient: func(ctx context.Context, server liteServer, opts client.Options) (client.Client, error) {
			return server.NewClientWithOptions(ctx, opts)
		},
		closeClient: func(_ context.Context, client client.Client) error {
			client.Close()
			return nil
		},
		createWorker: func(client client.Client, taskQueue string, opts worker.Options) (worker.Worker, error) {
			return worker.New(client, taskQueue, opts), nil
		},
		registerWorker: func(registry worker.Registry, registerFunc func(worker.Registry)) error {
			registerFunc(registry)
			return nil
		},
		startWorker: func(_ context.Context, worker worker.Worker) error {
			return worker.Start()
		},
		stopWorker: func(_ context.Context, worker worker.Worker) error {
			worker.Stop()
			return nil
		},
	}
}

func withLifecycleBackend(backend lifecycleBackend) TestServerOption {
	return applyFunc(func(server *TestServer) {
		server.lifecycleBackend = backend
	})
}

type lifecycleOperation struct {
	once sync.Once
	done chan struct{}
	err  error
}

func newLifecycleOperation() lifecycleOperation {
	return lifecycleOperation{done: make(chan struct{})}
}

func (o *lifecycleOperation) run(operation func() error) {
	o.once.Do(func() {
		go func() {
			defer close(o.done)
			defer func() {
				if recovered := recover(); recovered != nil {
					o.err = fmt.Errorf("panic: %v", recovered)
				}
			}()
			o.err = operation()
		}()
	})
}

func (o *lifecycleOperation) wait(ctx context.Context) error {
	select {
	case <-o.done:
		return o.err
	default:
	}
	select {
	case <-o.done:
		return o.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

type ownedLifecycleResource struct {
	resource LifecycleResource
	start    *lifecycleOperation
	release  lifecycleOperation
	releasef func(context.Context) error

	mu       sync.Mutex
	released bool
}

type ownedClient struct {
	client.Client
	resource *ownedLifecycleResource
}

type ownedWorker struct {
	worker.Worker
	resource *ownedLifecycleResource
}

func newOwnedLifecycleResource(resource LifecycleResource, release func(context.Context) error) *ownedLifecycleResource {
	return &ownedLifecycleResource{
		resource: resource,
		release:  newLifecycleOperation(),
		releasef: release,
	}
}

func (r *ownedLifecycleResource) startOperation(ctx context.Context, start func(context.Context) error) error {
	r.start = new(lifecycleOperation)
	*r.start = newLifecycleOperation()
	r.start.run(func() error { return start(context.Background()) })
	return r.start.wait(ctx)
}

func (r *ownedLifecycleResource) releaseResource(ctx context.Context) error {
	if r.isReleased() {
		return nil
	}
	if r.start != nil {
		if err := r.start.wait(ctx); err != nil && ctx.Err() != nil {
			return err
		}
	}
	r.release.run(func() error { return r.releasef(context.Background()) })
	if err := r.release.wait(ctx); err != nil {
		return err
	}
	r.mu.Lock()
	r.released = true
	r.mu.Unlock()
	return nil
}

func (r *ownedLifecycleResource) isReleased() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.released
}

func callLifecycle[T any](operation func() (T, error)) (value T, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return operation()
}

func callLifecycleError(operation func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return operation()
}

func (ts *TestServer) startupError(operation LifecycleOperation, err error) error {
	lifecycleErr := &LifecycleError{Operation: operation, Err: err}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), startupFailureCleanupLimit)
	defer cancel()
	return errors.Join(lifecycleErr, ts.StopContext(cleanupCtx))
}

func (ts *TestServer) acquisitionAllowed(operation LifecycleOperation) error {
	if ts.cleanupStarted {
		return &LifecycleError{Operation: operation, Err: errors.New("cleanup has started")}
	}
	return nil
}

func (ts *TestServer) ownServer(server liteServer) {
	ts.server = server
	ts.serverResource = newOwnedLifecycleResource(
		LifecycleResource{Kind: LifecycleResourceServer, Name: "server"},
		func(ctx context.Context) error {
			return ts.lifecycleBackend.stopServer(ctx, server)
		},
	)
}

func (ts *TestServer) ownClient(temporalClient client.Client) *ownedLifecycleResource {
	resource := newOwnedLifecycleResource(
		LifecycleResource{
			Kind: LifecycleResourceClient,
			Name: fmt.Sprintf("client-%d", ts.clientSequence+1),
		},
		func(ctx context.Context) error {
			return ts.lifecycleBackend.closeClient(ctx, temporalClient)
		},
	)
	ts.clientSequence++
	ts.clients = append(ts.clients, ownedClient{Client: temporalClient, resource: resource})
	return resource
}

func (ts *TestServer) ownWorker(taskQueue string, temporalWorker worker.Worker) *ownedLifecycleResource {
	resource := newOwnedLifecycleResource(
		LifecycleResource{Kind: LifecycleResourceWorker, Name: taskQueue},
		func(ctx context.Context) error {
			return ts.lifecycleBackend.stopWorker(ctx, temporalWorker)
		},
	)
	ts.workers = append(ts.workers, ownedWorker{Worker: temporalWorker, resource: resource})
	return resource
}

// OwnedResources returns every server, client, and worker that still requires
// cleanup, in deterministic workers-to-clients-to-server order.
func (ts *TestServer) OwnedResources() []LifecycleResource {
	resources := make([]LifecycleResource, 0, len(ts.workers)+len(ts.clients)+1)
	for _, worker := range ts.workers {
		if !worker.resource.isReleased() {
			resources = append(resources, worker.resource.resource)
		}
	}
	for _, client := range ts.clients {
		if !client.resource.isReleased() {
			resources = append(resources, client.resource.resource)
		}
	}
	if ts.serverResource != nil && !ts.serverResource.isReleased() {
		resources = append(resources, ts.serverResource.resource)
	}
	return resources
}

// StopContext closes workers, clients, and the server in that order. The
// context bounds waiting, and any in-progress release is reused by later calls.
func (ts *TestServer) StopContext(ctx context.Context) error {
	if ctx == nil {
		return &CleanupError{Remaining: ts.OwnedResources(), Err: errors.New("nil context")}
	}
	ts.cleanupStarted = true
	ts.defaultClient = nil

	var releaseErr error
	for index := len(ts.workers) - 1; index >= 0; index-- {
		if err := ts.workers[index].resource.releaseResource(ctx); err != nil {
			releaseErr = errors.Join(releaseErr, fmt.Errorf("stop worker %q: %w", ts.workers[index].resource.resource.Name, err))
		}
	}
	ts.compactWorkers()
	if len(ts.workers) != 0 {
		return ts.cleanupError(ctx, releaseErr)
	}

	for index := len(ts.clients) - 1; index >= 0; index-- {
		if err := ts.clients[index].resource.releaseResource(ctx); err != nil {
			releaseErr = errors.Join(releaseErr, fmt.Errorf("close client %q: %w", ts.clients[index].resource.resource.Name, err))
		}
	}
	ts.compactClients()
	if len(ts.clients) != 0 {
		return ts.cleanupError(ctx, releaseErr)
	}

	if ts.serverResource != nil {
		if err := ts.serverResource.releaseResource(ctx); err != nil {
			releaseErr = errors.Join(releaseErr, fmt.Errorf("stop server: %w", err))
		} else {
			ts.serverResource = nil
			ts.server = nil
		}
	}
	if releaseErr != nil {
		return ts.cleanupError(ctx, releaseErr)
	}
	return nil
}

func (ts *TestServer) cleanupError(ctx context.Context, releaseErr error) error {
	cause := releaseErr
	if ctx.Err() != nil {
		cause = errors.Join(ctx.Err(), releaseErr)
	}
	return &CleanupError{Remaining: ts.OwnedResources(), Err: cause}
}

func (ts *TestServer) compactWorkers() {
	remaining := ts.workers[:0]
	for _, worker := range ts.workers {
		if !worker.resource.isReleased() {
			remaining = append(remaining, worker)
		}
	}
	ts.workers = remaining
}

func (ts *TestServer) compactClients() {
	remaining := ts.clients[:0]
	for _, client := range ts.clients {
		if !client.resource.isReleased() {
			remaining = append(remaining, client)
		}
	}
	ts.clients = remaining
}

var _ liteServer = (*temporalite.LiteServer)(nil)
