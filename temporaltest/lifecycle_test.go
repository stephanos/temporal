package temporaltest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func TestFrontendHTTPOptionStartsLoopbackListener(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	server, err := NewServerWithContext(ctx, WithT(t), WithFrontendHTTP())
	require.NoError(t, err)
	require.Positive(t, server.frontendHTTPPort)

	connection, err := net.DialTimeout(
		"tcp",
		net.JoinHostPort("127.0.0.1", strconv.Itoa(server.frontendHTTPPort)),
		time.Second,
	)
	require.NoError(t, err)
	require.NoError(t, connection.Close())
}

func TestLifecycleStartupFailuresUnwindOwnedResources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		failure    string
		start      func(context.Context, *lifecycleFixture) (*TestServer, error)
		wantEvents []string
	}{
		{
			name:    "server construction",
			failure: string(LifecycleOperationCreateServer),
			start: func(ctx context.Context, fixture *lifecycleFixture) (*TestServer, error) {
				return NewServerWithContext(ctx, withLifecycleBackend(fixture.backend()))
			},
			wantEvents: []string{"create-server"},
		},
		{
			name:    "server start",
			failure: string(LifecycleOperationStartServer),
			start: func(ctx context.Context, fixture *lifecycleFixture) (*TestServer, error) {
				return NewServerWithContext(ctx, withLifecycleBackend(fixture.backend()))
			},
			wantEvents: []string{"create-server", "start-server", "stop-server"},
		},
		{
			name:    "client creation",
			failure: string(LifecycleOperationCreateClient),
			start: func(ctx context.Context, fixture *lifecycleFixture) (*TestServer, error) {
				server, err := NewServerWithContext(ctx, withLifecycleBackend(fixture.backend()))
				require.NoError(t, err)
				_, err = server.NewClientWithOptionsContext(ctx, client.Options{})
				return server, err
			},
			wantEvents: []string{
				"create-server", "start-server", "create-client",
				"close-client", "stop-server",
			},
		},
		{
			name:    "worker construction",
			failure: string(LifecycleOperationCreateWorker),
			start: func(ctx context.Context, fixture *lifecycleFixture) (*TestServer, error) {
				server, err := NewServerWithContext(ctx, withLifecycleBackend(fixture.backend()))
				require.NoError(t, err)
				_, err = server.NewWorkerWithContext(ctx, "queue", func(worker.Registry) {})
				return server, err
			},
			wantEvents: []string{
				"create-server", "start-server", "create-client", "create-worker",
				"stop-worker", "close-client", "stop-server",
			},
		},
		{
			name:    "worker registration",
			failure: string(LifecycleOperationRegisterWorker),
			start: func(ctx context.Context, fixture *lifecycleFixture) (*TestServer, error) {
				server, err := NewServerWithContext(ctx, withLifecycleBackend(fixture.backend()))
				require.NoError(t, err)
				_, err = server.NewWorkerWithContext(ctx, "queue", func(worker.Registry) {})
				return server, err
			},
			wantEvents: []string{
				"create-server", "start-server", "create-client", "create-worker", "register-worker",
				"stop-worker", "close-client", "stop-server",
			},
		},
		{
			name:    "worker start",
			failure: string(LifecycleOperationStartWorker),
			start: func(ctx context.Context, fixture *lifecycleFixture) (*TestServer, error) {
				server, err := NewServerWithContext(ctx, withLifecycleBackend(fixture.backend()))
				require.NoError(t, err)
				_, err = server.NewWorkerWithContext(ctx, "queue", func(worker.Registry) {})
				return server, err
			},
			wantEvents: []string{
				"create-server", "start-server", "create-client", "create-worker", "register-worker", "start-worker",
				"stop-worker", "close-client", "stop-server",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fixture := newLifecycleFixture()
			fixture.acquireFailure = test.failure

			server, err := test.start(context.Background(), fixture)

			require.Error(t, err)
			require.NotNil(t, server)
			var lifecycleErr *LifecycleError
			require.ErrorAs(t, err, &lifecycleErr)
			require.Equal(t, LifecycleOperation(test.failure), lifecycleErr.Operation)
			require.Equal(t, test.wantEvents, fixture.snapshotEvents())
			require.Empty(t, server.OwnedResources())
			require.NoError(t, server.StopContext(context.Background()))
			require.Equal(t, test.wantEvents, fixture.snapshotEvents())
		})
	}
}

func TestLifecycleReleaseFailuresReportExactResidualOwnership(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		failure       string
		wantEvents    []string
		wantRemaining []LifecycleResource
	}{
		{
			name:       "worker stop",
			failure:    "stop-worker",
			wantEvents: []string{"stop-worker"},
			wantRemaining: []LifecycleResource{
				{Kind: LifecycleResourceWorker, Name: "queue"},
				{Kind: LifecycleResourceClient, Name: "client-1"},
				{Kind: LifecycleResourceServer, Name: "server"},
			},
		},
		{
			name:       "client close",
			failure:    "close-client",
			wantEvents: []string{"stop-worker", "close-client"},
			wantRemaining: []LifecycleResource{
				{Kind: LifecycleResourceClient, Name: "client-1"},
				{Kind: LifecycleResourceServer, Name: "server"},
			},
		},
		{
			name:       "server stop",
			failure:    "stop-server",
			wantEvents: []string{"stop-worker", "close-client", "stop-server"},
			wantRemaining: []LifecycleResource{
				{Kind: LifecycleResourceServer, Name: "server"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fixture := newLifecycleFixture()
			server := startCompleteLifecycle(t, fixture)
			fixture.clearEvents()
			fixture.releaseFailure = test.failure

			err := server.StopContext(context.Background())

			require.Error(t, err)
			var cleanupErr *CleanupError
			require.ErrorAs(t, err, &cleanupErr)
			require.Equal(t, test.wantRemaining, cleanupErr.Remaining)
			require.Equal(t, test.wantRemaining, server.OwnedResources())
			require.Equal(t, test.wantEvents, fixture.snapshotEvents())
		})
	}
}

func TestLifecycleDeadlineCanBeCleanedUpLaterWithoutDuplicateRelease(t *testing.T) {
	t.Parallel()
	fixture := newLifecycleFixture()
	fixture.blockWorkerStop = make(chan struct{})
	server := startCompleteLifecycle(t, fixture)
	fixture.clearEvents()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := server.StopContext(ctx)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	var cleanupErr *CleanupError
	require.ErrorAs(t, err, &cleanupErr)
	require.Equal(t, []LifecycleResource{
		{Kind: LifecycleResourceWorker, Name: "queue"},
		{Kind: LifecycleResourceClient, Name: "client-1"},
		{Kind: LifecycleResourceServer, Name: "server"},
	}, cleanupErr.Remaining)
	require.Equal(t, []string{"stop-worker"}, fixture.snapshotEvents())

	close(fixture.blockWorkerStop)
	require.NoError(t, server.StopContext(context.Background()))
	wantEvents := []string{"stop-worker", "close-client", "stop-server"}
	require.Equal(t, wantEvents, fixture.snapshotEvents())
	require.Empty(t, server.OwnedResources())

	require.NoError(t, server.StopContext(context.Background()))
	require.Equal(t, wantEvents, fixture.snapshotEvents())
}

func TestLifecycleRegistrationCancellationUnwindsWithoutStartingWorker(t *testing.T) {
	t.Parallel()
	fixture := newLifecycleFixture()
	fixture.registrationEntered = make(chan struct{})
	fixture.blockRegistration = make(chan struct{})
	defer func() {
		select {
		case <-fixture.blockRegistration:
		default:
			close(fixture.blockRegistration)
		}
	}()
	server, err := NewServerWithContext(
		context.Background(),
		withLifecycleBackend(fixture.backend()),
		withStartupFailureCleanupLimit(20*time.Millisecond),
	)
	require.NoError(t, err)
	fixture.clearEvents()
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, workerErr := server.NewWorkerWithContext(ctx, "queue", func(worker.Registry) {})
		result <- workerErr
	}()
	requireSignal(t, fixture.registrationEntered)

	cancel()
	select {
	case err = <-result:
	case <-time.After(time.Second):
		t.Fatal("worker registration ignored the lifecycle deadline")
	}

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	var cleanupErr *CleanupError
	require.ErrorAs(t, err, &cleanupErr)
	require.Equal(t, []LifecycleResource{
		{Kind: LifecycleResourceWorker, Name: "queue"},
		{Kind: LifecycleResourceClient, Name: "client-1"},
		{Kind: LifecycleResourceServer, Name: "server"},
	}, cleanupErr.Remaining)
	require.Equal(t, []string{
		"create-client", "create-worker", "register-worker",
	}, fixture.snapshotEvents())
	close(fixture.blockRegistration)
	require.NoError(t, server.StopContext(context.Background()))
	require.Empty(t, server.OwnedResources())
}

func TestLifecycleCancellationBetweenAcquisitionsDoesNotStartNextResource(t *testing.T) {
	t.Parallel()

	t.Run("server construction and start", func(t *testing.T) {
		t.Parallel()
		fixture := newLifecycleFixture()
		ctx, cancel := context.WithCancel(context.Background())
		fixture.afterCreateServer = cancel

		server, err := NewServerWithContext(ctx, withLifecycleBackend(fixture.backend()))

		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, []string{"create-server", "stop-server"}, fixture.snapshotEvents())
		require.Empty(t, server.OwnedResources())
	})

	t.Run("worker registration and start", func(t *testing.T) {
		t.Parallel()
		fixture := newLifecycleFixture()
		server, err := NewServerWithContext(context.Background(), withLifecycleBackend(fixture.backend()))
		require.NoError(t, err)
		fixture.clearEvents()
		ctx, cancel := context.WithCancel(context.Background())

		_, err = server.NewWorkerWithContext(ctx, "queue", func(worker.Registry) { cancel() })

		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, []string{
			"create-client", "create-worker", "register-worker",
			"stop-worker", "close-client", "stop-server",
		}, fixture.snapshotEvents())
		require.Empty(t, server.OwnedResources())
	})
}

func TestLegacyDefaultClientRetainsDialDeadline(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("deadline observed")
	fixture := newLifecycleFixture()
	fixture.checkClientContext = func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			return errors.New("client context has no deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 9*time.Second || remaining > 10*time.Second {
			return fmt.Errorf("unexpected client deadline: %s", remaining)
		}
		return wantErr
	}
	server, err := NewServerWithContext(context.Background(), withLifecycleBackend(fixture.backend()))
	require.NoError(t, err)

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		server.GetDefaultClient()
	}()

	require.Error(t, recovered.(error))
	require.ErrorIs(t, recovered.(error), wantErr)
	require.Empty(t, server.OwnedResources())
}

func TestLifecycleWithNilTestingIntegrationReturnsErrorsWithoutPanicking(t *testing.T) {
	t.Parallel()
	fixture := newLifecycleFixture()
	fixture.acquireFailure = "start-server"

	var server *TestServer
	var err error
	require.NotPanics(t, func() {
		server, err = NewServerWithContext(
			context.Background(),
			WithT(nil),
			withLifecycleBackend(fixture.backend()),
		)
	})
	require.Error(t, err)
	require.NotNil(t, server)
	require.Empty(t, server.OwnedResources())
}

func TestLifecycleCanceledContextPerformsNoAcquisition(t *testing.T) {
	t.Parallel()
	fixture := newLifecycleFixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server, err := NewServerWithContext(ctx, withLifecycleBackend(fixture.backend()))

	require.ErrorIs(t, err, context.Canceled)
	require.NotNil(t, server)
	require.Empty(t, server.OwnedResources())
	require.Empty(t, fixture.snapshotEvents())
}

func TestLifecycleCanceledAcquisitionUnwindsWithoutCreatingAnotherResource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation LifecycleOperation
		acquire   func(context.Context, *TestServer) error
	}{
		{
			name:      "client",
			operation: LifecycleOperationCreateClient,
			acquire: func(ctx context.Context, server *TestServer) error {
				_, err := server.NewClientWithOptionsContext(ctx, client.Options{})
				return err
			},
		},
		{
			name:      "worker",
			operation: LifecycleOperationCreateWorker,
			acquire: func(ctx context.Context, server *TestServer) error {
				_, err := server.NewWorkerWithContext(ctx, "queue", func(worker.Registry) {})
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fixture := newLifecycleFixture()
			server, err := NewServerWithContext(context.Background(), withLifecycleBackend(fixture.backend()))
			require.NoError(t, err)
			fixture.clearEvents()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err = test.acquire(ctx, server)

			require.ErrorIs(t, err, context.Canceled)
			var lifecycleErr *LifecycleError
			require.ErrorAs(t, err, &lifecycleErr)
			require.Equal(t, test.operation, lifecycleErr.Operation)
			require.Equal(t, []string{"stop-server"}, fixture.snapshotEvents())
			require.Empty(t, server.OwnedResources())
		})
	}
}

func TestLifecycleRejectsAcquisitionAfterCleanupStarts(t *testing.T) {
	t.Parallel()
	fixture := newLifecycleFixture()
	server, err := NewServerWithContext(context.Background(), withLifecycleBackend(fixture.backend()))
	require.NoError(t, err)
	require.NoError(t, server.StopContext(context.Background()))
	fixture.clearEvents()

	_, err = server.NewClientWithOptionsContext(context.Background(), client.Options{})

	require.Error(t, err)
	var lifecycleErr *LifecycleError
	require.ErrorAs(t, err, &lifecycleErr)
	require.Equal(t, LifecycleOperationCreateClient, lifecycleErr.Operation)
	require.Empty(t, fixture.snapshotEvents())
	require.Empty(t, server.OwnedResources())
}

func TestLifecycleAppliesExplicitWorkerStopTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts worker.Options
		want time.Duration
	}{
		{name: "default", want: defaultWorkerStopTimeout},
		{name: "caller configured", opts: worker.Options{WorkerStopTimeout: 2 * time.Second}, want: 2 * time.Second},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fixture := newLifecycleFixture()
			server, err := NewServerWithContext(context.Background(), withLifecycleBackend(fixture.backend()))
			require.NoError(t, err)
			_, err = server.NewWorkerWithOptionsContext(
				context.Background(),
				"queue",
				func(worker.Registry) {},
				test.opts,
			)
			require.NoError(t, err)
			require.Equal(t, test.want, fixture.workerOptions.WorkerStopTimeout)
			require.NoError(t, server.StopContext(context.Background()))
		})
	}
}

func startCompleteLifecycle(t *testing.T, fixture *lifecycleFixture) *TestServer {
	t.Helper()
	server, err := NewServerWithContext(context.Background(), withLifecycleBackend(fixture.backend()))
	require.NoError(t, err)
	_, err = server.NewWorkerWithContext(context.Background(), "queue", func(worker.Registry) {})
	require.NoError(t, err)
	return server
}

type lifecycleFixture struct {
	mu                  sync.Mutex
	events              []string
	acquireFailure      string
	releaseFailure      string
	blockWorkerStop     chan struct{}
	blockRegistration   chan struct{}
	registrationEntered chan struct{}
	afterCreateServer   func()
	checkClientContext  func(context.Context) error
	workerOptions       worker.Options
	clientSequence      int
}

func newLifecycleFixture() *lifecycleFixture {
	return &lifecycleFixture{}
}

func (f *lifecycleFixture) backend() lifecycleBackend {
	return lifecycleBackend{
		createServer: func(*TestServer) (liteServer, error) {
			f.record("create-server")
			if f.afterCreateServer != nil {
				f.afterCreateServer()
			}
			if f.acquireFailure == "create-server" {
				return nil, errors.New("injected server construction failure")
			}
			return &fixtureServer{}, nil
		},
		startServer: func(context.Context, liteServer) error {
			f.record("start-server")
			return f.acquireError("start-server")
		},
		stopServer: func(context.Context, liteServer) error {
			f.record("stop-server")
			return f.releaseError("stop-server")
		},
		createClient: func(ctx context.Context, _ liteServer, _ client.Options) (client.Client, error) {
			f.record("create-client")
			if f.checkClientContext != nil {
				return nil, f.checkClientContext(ctx)
			}
			f.mu.Lock()
			f.clientSequence++
			name := fmt.Sprintf("client-%d", f.clientSequence)
			f.mu.Unlock()
			created := &fixtureClient{name: name}
			return created, f.acquireError("create-client")
		},
		closeClient: func(context.Context, client.Client) error {
			f.record("close-client")
			return f.releaseError("close-client")
		},
		createWorker: func(_ client.Client, _ string, opts worker.Options) (worker.Worker, error) {
			f.record("create-worker")
			f.mu.Lock()
			f.workerOptions = opts
			f.mu.Unlock()
			created := &fixtureWorker{}
			return created, f.acquireError("create-worker")
		},
		registerWorker: func(registry worker.Registry, registerFunc func(worker.Registry)) error {
			f.record("register-worker")
			if f.registrationEntered != nil {
				close(f.registrationEntered)
			}
			if f.blockRegistration != nil {
				<-f.blockRegistration
			}
			if err := f.acquireError("register-worker"); err != nil {
				return err
			}
			registerFunc(registry)
			return nil
		},
		startWorker: func(context.Context, worker.Worker) error {
			f.record("start-worker")
			return f.acquireError("start-worker")
		},
		stopWorker: func(context.Context, worker.Worker) error {
			f.mu.Lock()
			f.events = append(f.events, "stop-worker")
			block := f.blockWorkerStop
			f.mu.Unlock()
			if block != nil {
				<-block
			}
			return f.releaseError("stop-worker")
		},
	}
}

func (f *lifecycleFixture) acquireError(boundary string) error {
	if f.acquireFailure == boundary {
		return fmt.Errorf("injected %s failure", boundary)
	}
	return nil
}

func (f *lifecycleFixture) releaseError(boundary string) error {
	if f.releaseFailure == boundary {
		return fmt.Errorf("injected %s failure", boundary)
	}
	return nil
}

func (f *lifecycleFixture) record(event string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func (f *lifecycleFixture) snapshotEvents() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.events...)
}

func (f *lifecycleFixture) clearEvents() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = nil
}

func requireSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("operation did not reach the expected boundary")
	}
}

type fixtureServer struct{}

func (*fixtureServer) StartContext(context.Context) error { return nil }
func (*fixtureServer) StopContext(context.Context) error  { return nil }
func (*fixtureServer) NewClientWithOptions(context.Context, client.Options) (client.Client, error) {
	return &fixtureClient{name: "unused"}, nil
}
func (*fixtureServer) FrontendHostPort() string { return "127.0.0.1:1" }

type fixtureClient struct {
	client.Client
	name string
}

func (*fixtureClient) Close() {}

type fixtureWorker struct {
	worker.Worker
}

func (*fixtureWorker) Start() error { return nil }
func (*fixtureWorker) Stop()        {}
