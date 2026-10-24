// Package act is the Runner's Temporal Driver. It sends what the Model proposed over the
// generated WorkflowService API, reports what the server answered as correlated events, and lets
// an in-process Go SDK worker take a task only when the Model's read instruction admits one,
// delivering the Model's reply to it. It never decides what a message means: poll interval,
// read timeout, backoff and worker tasks come from the instruction, the rest from Options.
package act

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	realizationv1 "umpire.local/model/umpire/realization/v1"
	runv1 "umpire.local/model/umpire/run/v1"
	"umpire.local/model/umpire/runner"
)

// The Options defaults.
const (
	DefaultRPCTimeout        = 10 * time.Second
	DefaultWorkerPollTimeout = 5 * time.Second
	DefaultCloseTimeout      = 10 * time.Second
	// MinWorkerPollTimeout is the frontend's shortest long poll; it refuses shorter ones at once.
	MinWorkerPollTimeout = 2 * time.Second
)

// Options configure the Driver. Namespace and TaskQueue must be the ones the Model's World names;
// the Driver never reads the World.
type Options struct {
	// Address is the frontend's host:port.
	Address string
	// Namespace is where the worker polls and what Create provisions.
	Namespace string
	// TaskQueue is the queue the worker polls.
	TaskQueue string
	// Create registers Namespace before the first Run and deletes it when the Factory closes.
	Create bool
	// RPCTimeout bounds each call that is not a read.
	RPCTimeout time.Duration
	// WorkerPollTimeout bounds each worker poll the Driver admits. The frontend refuses long
	// polls shorter than two seconds.
	WorkerPollTimeout time.Duration
	// MaxReadTimeout caps an instruction's read timeout; zero keeps the instruction's.
	MaxReadTimeout time.Duration
	// CloseTimeout bounds closing one Driver.
	CloseTimeout time.Duration
	// Redact names the fields removed from a Run's evidence once it is judged, as a message's
	// full name followed by a field path, e.g.
	// temporal.api.workflowservice.v1.PollActivityTaskQueueResponse.task_token.
	Redact []string
	// Transport is how the Driver reaches Address; the zero value is plaintext without credentials.
	Transport Transport
}

// Transport is a credentialed connection to the frontend: the dial options of the Driver's own
// calls and the SDK worker client's TLS and credentials. It is never read from the environment.
type Transport struct {
	DialOptions       []grpc.DialOption
	ClientTLS         *tls.Config
	ClientCredentials client.Credentials
}

// The environment Configure reads.
const (
	EnvAddress           = "UMPIRE_TEMPORAL_ADDRESS"
	EnvNamespace         = "UMPIRE_TEMPORAL_NAMESPACE"
	EnvTaskQueue         = "UMPIRE_TEMPORAL_TASK_QUEUE"
	EnvCreate            = "UMPIRE_TEMPORAL_CREATE"
	EnvRPCTimeout        = "UMPIRE_TEMPORAL_RPC_TIMEOUT"
	EnvWorkerPollTimeout = "UMPIRE_TEMPORAL_WORKER_POLL_TIMEOUT"
	EnvRedact            = "UMPIRE_TEMPORAL_REDACT"
)

// Configure reads Options from the environment.
func Configure(getenv func(string) string) (Options, error) {
	opts := Options{Address: getenv(EnvAddress), Namespace: getenv(EnvNamespace), TaskQueue: getenv(EnvTaskQueue)}
	var err error
	if value := getenv(EnvCreate); value != "" {
		if opts.Create, err = strconv.ParseBool(value); err != nil {
			return opts, fmt.Errorf("%s: %w", EnvCreate, err)
		}
	}
	for name, field := range map[string]*time.Duration{EnvRPCTimeout: &opts.RPCTimeout, EnvWorkerPollTimeout: &opts.WorkerPollTimeout} {
		if value := getenv(name); value != "" {
			if *field, err = time.ParseDuration(value); err != nil {
				return opts, fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	if value := getenv(EnvRedact); value != "" {
		for path := range strings.SplitSeq(value, ",") {
			opts.Redact = append(opts.Redact, strings.TrimSpace(path))
		}
	}
	return opts, opts.validate()
}

func (o Options) validate() error {
	switch {
	case o.Address == "":
		return fmt.Errorf("%s is required", EnvAddress)
	case o.Namespace == "":
		return fmt.Errorf("%s is required", EnvNamespace)
	case o.TaskQueue == "":
		return fmt.Errorf("%s is required", EnvTaskQueue)
	case o.RPCTimeout < 0 || o.WorkerPollTimeout < 0 || o.MaxReadTimeout < 0 || o.CloseTimeout < 0:
		return errors.New("timeouts cannot be negative")
	case o.WorkerPollTimeout != 0 && o.WorkerPollTimeout < MinWorkerPollTimeout:
		return fmt.Errorf("%s must be at least %s", EnvWorkerPollTimeout, MinWorkerPollTimeout)
	}
	for _, path := range o.Redact {
		if err := resolvable(path); err != nil {
			return fmt.Errorf("%s: %w", EnvRedact, err)
		}
	}
	return nil
}

func (o Options) withDefaults() Options {
	if o.RPCTimeout == 0 {
		o.RPCTimeout = DefaultRPCTimeout
	}
	if o.WorkerPollTimeout == 0 {
		o.WorkerPollTimeout = DefaultWorkerPollTimeout
	}
	if o.CloseTimeout == 0 {
		o.CloseTimeout = DefaultCloseTimeout
	}
	return o
}

func (t Transport) clientTLS() *tls.Config {
	if t.ClientTLS == nil {
		return nil
	}
	return t.ClientTLS.Clone()
}

// Factory makes one Driver per Run and owns what outlives a Run: the provisioned namespace.
type Factory struct {
	opts Options

	mu          sync.Mutex
	provisioned bool
	cleanup     provision.Cleanup
	conn        *grpc.ClientConn
}

// NewFactory checks the Options.
func NewFactory(opts Options) (*Factory, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	return &Factory{opts: opts.withDefaults()}, nil
}

// New is the Factory's runner.NewDriver. The first call provisions when Options.Create is set.
func (f *Factory) New(ctx context.Context) (runner.Driver, error) {
	if err := f.provision(ctx); err != nil {
		return nil, err
	}
	return newDriver(f.opts)
}

func (f *Factory) provision(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.opts.Create || f.provisioned {
		return nil
	}
	conn, err := dial(f.opts)
	if err != nil {
		return err
	}
	cleanup, err := provision.Create(ctx, provision.Clients{
		Workflow: workflowservice.NewWorkflowServiceClient(conn), Operator: operatorservice.NewOperatorServiceClient(conn),
	}, provision.Resources{Namespace: f.opts.Namespace})
	if err != nil {
		return errors.Join(fmt.Errorf("provision: %w", err), conn.Close())
	}
	f.provisioned, f.cleanup, f.conn = true, cleanup, conn
	return nil
}

// Close removes what the Factory provisioned.
func (f *Factory) Close(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cleanup == nil {
		return nil
	}
	err := f.cleanup(ctx)
	f.cleanup = nil
	return errors.Join(err, f.conn.Close())
}

func dial(opts Options) (*grpc.ClientConn, error) {
	dialOptions := opts.Transport.DialOptions
	if len(dialOptions) == 0 {
		dialOptions = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	return grpc.NewClient(opts.Address, append(slices.Clone(dialOptions), grpc.WithStatsHandler(completion{}))...)
}

// Driver acts on one Temporal deployment for one Run.
type Driver struct {
	opts  Options
	conn  *grpc.ClientConn
	gate  *gate
	calls atomic.Uint64

	// ensureWorker starts the SDK worker; tests replace it to drive the gate themselves.
	ensureWorker func(context.Context) error
	workerMu     sync.Mutex
	sdk          client.Client
	worker       worker.Worker
	workerErr    error

	// released is closed once Close has terminated what the Driver started; activities the Model
	// did not answer are held until then.
	released    chan struct{}
	releaseOnce sync.Once

	mu         sync.Mutex
	exchanges  map[*exchange]bool
	activities []*workflowservice.TerminateActivityExecutionRequest
	teardowns  []error
	// unanswered are the task tokens of the tasks held because the Model did not answer them.
	unanswered map[string]bool
}

func newDriver(opts Options) (*Driver, error) {
	conn, err := dial(opts)
	if err != nil {
		return nil, err
	}
	d := &Driver{opts: opts, conn: conn, gate: newGate(opts.WorkerPollTimeout), exchanges: map[*exchange]bool{}, released: make(chan struct{}), unanswered: map[string]bool{}}
	d.ensureWorker = d.startWorker
	return d, nil
}

// Start sends the proposal and returns at once; its events report what happened.
func (d *Driver) Start(_ context.Context, request uint64, payload *anypb.Any) (runner.Exchange, error) {
	if err := validStart(payload); err != nil {
		return nil, err
	}
	x := newExchange(d, request)
	d.mu.Lock()
	d.exchanges[x] = true
	d.mu.Unlock()
	go x.run(payload)
	return x, nil
}

func (d *Driver) forget(x *exchange) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.exchanges, x)
}

// started remembers an activity this Driver may have started, so Close can end it if it is still
// open: a later Run's Start of the same World reuses its activity ID. observation is nil when the
// server did not answer the Start, which may still have started it; Close then ends it by ID.
func (d *Driver) started(request *anypb.Any, observation *realizationv1.Observation) {
	start := new(workflowservice.StartActivityExecutionRequest)
	if !request.MessageIs(start) || request.UnmarshalTo(start) != nil {
		return
	}
	var runID string
	if observation != nil {
		response := new(workflowservice.StartActivityExecutionResponse)
		if observation.GetResponse() == nil || observation.GetResponse().UnmarshalTo(response) != nil {
			return
		}
		runID = response.GetRunId()
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.activities = append(d.activities, &workflowservice.TerminateActivityExecutionRequest{
		Namespace: start.GetNamespace(), ActivityId: start.GetActivityId(), RunId: runID,
		Identity: "umpire", Reason: "the Run ended",
	})
}

func (d *Driver) teardown(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.teardowns = append(d.teardowns, err)
}

// Close cancels and joins the Driver's work. It refuses every held worker poll and cancels the
// admitted one, closes the exchanges, terminates the activities it started that are still open,
// and only then releases activities the Model did not answer, so the SDK's late response meets a
// terminated activity. A response the SDK still completes is a teardown fault.
func (d *Driver) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), d.opts.CloseTimeout)
	defer cancel()
	d.gate.stop()
	d.mu.Lock()
	exchanges := make([]*exchange, 0, len(d.exchanges))
	for x := range d.exchanges {
		exchanges = append(exchanges, x)
	}
	activities := d.activities
	d.activities = nil
	d.mu.Unlock()
	var errs []error
	for _, x := range exchanges {
		errs = append(errs, x.Close())
	}
	service := workflowservice.NewWorkflowServiceClient(d.conn)
	for _, activity := range activities {
		_, err := service.TerminateActivityExecution(ctx, activity)
		if code := status.Code(err); code != codes.OK && code != codes.NotFound && code != codes.FailedPrecondition {
			errs = append(errs, fmt.Errorf("terminate activity %s: %w", activity.GetActivityId(), err))
		}
	}
	d.releaseOnce.Do(func() { close(d.released) })
	d.workerMu.Lock()
	if d.worker != nil {
		d.worker.Stop()
		d.sdk.Close()
		d.worker, d.sdk = nil, nil
	}
	d.workerMu.Unlock()
	d.mu.Lock()
	errs = append(errs, d.teardowns...)
	d.teardowns = nil
	d.mu.Unlock()
	return errors.Join(append(errs, d.conn.Close())...)
}

// Redact removes the configured fields from an event's evidence.
func (d *Driver) Redact(event *runv1.Event) bool {
	return redact(event.ProtoReflect(), d.opts.Redact)
}

var _ runner.Redactor = (*Driver)(nil)
