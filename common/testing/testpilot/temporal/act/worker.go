package act

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	sdklog "go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	realizationv1 "umpire.local/model/umpire/realization/v1"
	"umpire.local/model/umpire/runner"
)

type pollOutcome int

const (
	pollAdmitted pollOutcome = iota
	pollEmpty
	pollTook
)

// gate admits the SDK worker's task polls only through a window a read exchange opens after the
// Model asked to continue reading. A window admits at most one poll, and only while no other poll
// or task is outstanding; reads stay paused while it is, because a read between a taken task and
// its Answer would see a delivery the Model has not been told about. Each window has its own
// outcome channel, so an outcome is only ever counted in the window that admitted its poll.
type gate struct {
	timeout time.Duration
	ctx     context.Context
	cancel  context.CancelFunc

	mu      sync.Mutex
	changed chan struct{}
	stopped bool
	open    *window
	held    *window
	token   []byte
	task    *workflowservice.PollActivityTaskQueueResponse
}

// window is one opening of the gate by one read exchange.
type window struct {
	x        *exchange
	admitted bool
	// outcomes holds the admitted poll's two outcomes: admitted, then empty or took.
	outcomes chan pollOutcome
}

func newGate(timeout time.Duration) *gate {
	ctx, cancel := context.WithCancel(context.Background())
	return &gate{timeout: timeout, ctx: ctx, cancel: cancel, changed: make(chan struct{})}
}

func (g *gate) broadcast() {
	close(g.changed)
	g.changed = make(chan struct{})
}

// errStopped answers a poll the gate holds when the Driver closes.
var errStopped = status.Error(codes.Canceled, "the Driver is closing")

// intercept holds every PollActivityTaskQueue until a window admits it, bounds the admitted poll,
// and captures the actual task it returns.
func (g *gate) intercept(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if method != workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName {
		return invoke(ctx, method, req, reply, cc, opts...)
	}
	w, err := g.admit(ctx)
	if err != nil {
		return err
	}
	pollCtx, cancel := context.WithTimeout(ctx, g.timeout)
	stop := context.AfterFunc(g.ctx, cancel)
	err = invoke(pollCtx, method, req, reply, cc, opts...)
	stop()
	cancel()
	task, _ := reply.(*workflowservice.PollActivityTaskQueueResponse)
	if err == nil && len(task.GetTaskToken()) > 0 {
		g.took(w, proto.Clone(task).(*workflowservice.PollActivityTaskQueueResponse))
		w.outcomes <- pollTook
	} else {
		g.finish(w)
		w.outcomes <- pollEmpty
	}
	return err
}

func (g *gate) admit(ctx context.Context) (*window, error) {
	for {
		g.mu.Lock()
		if g.stopped {
			g.mu.Unlock()
			return nil, errStopped
		}
		if w := g.open; w != nil && g.held == nil {
			w.admitted, g.open, g.held = true, nil, w
			g.mu.Unlock()
			w.outcomes <- pollAdmitted
			return w, nil
		}
		changed := g.changed
		g.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (g *gate) openFor(x *exchange) *window {
	g.mu.Lock()
	defer g.mu.Unlock()
	w := &window{x: x, outcomes: make(chan pollOutcome, 2)}
	if !g.stopped {
		g.open = w
		g.broadcast()
	}
	return w
}

// closeIdle closes w and reports whether it admitted no poll.
func (g *gate) closeIdle(w *window) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open == w {
		g.open = nil
	}
	return !w.admitted
}

func (g *gate) took(w *window, task *workflowservice.PollActivityTaskQueueResponse) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.held == w {
		g.token, g.task = task.GetTaskToken(), task
	}
}

// finish releases what w holds: its opening and its poll or task.
func (g *gate) finish(w *window) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open == w {
		g.open = nil
	}
	if g.held == w {
		g.held, g.token, g.task = nil, nil, nil
	}
	g.broadcast()
}

// stop refuses every held and later poll and cancels the admitted one.
func (g *gate) stop() {
	g.mu.Lock()
	g.stopped, g.open = true, nil
	g.broadcast()
	g.mu.Unlock()
	g.cancel()
}

// claim returns the window a task was taken in, and the captured task when its token matches.
func (g *gate) claim(token []byte) (*window, *workflowservice.PollActivityTaskQueueResponse) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.held == nil || g.task == nil {
		return nil, nil
	}
	if !bytes.Equal(token, g.token) {
		return g.held, nil
	}
	return g.held, g.task
}

// workerCall is one activity execution waiting for the Model's reply.
type workerCall struct {
	task  *workflowservice.PollActivityTaskQueueResponse
	reply chan decision
}

// delivery is a reply as the SDK returns it from the activity.
type delivery struct {
	value converter.RawValue
	err   error
}

// deliverable converts the Model's reply into what the SDK sends for it: a completion's one
// payload, or a failure. Anything else cannot be sent through the SDK.
func deliverable(reply *anypb.Any) (delivery, error) {
	envelope := new(realizationv1.Reply)
	if err := reply.UnmarshalTo(envelope); err != nil {
		return delivery{}, fmt.Errorf("the reply does not decode: %w", err)
	}
	request := envelope.GetRequest()
	switch {
	case request.MessageIs(&workflowservice.RespondActivityTaskCompletedRequest{}):
		completed := new(workflowservice.RespondActivityTaskCompletedRequest)
		if err := request.UnmarshalTo(completed); err != nil {
			return delivery{}, err
		}
		if err := sendable(completed, "result"); err != nil {
			return delivery{}, err
		}
		payloads := completed.GetResult().GetPayloads()
		if len(payloads) != 1 {
			return delivery{}, fmt.Errorf("the SDK returns one result payload, not %d", len(payloads))
		}
		return delivery{value: converter.NewRawValue(payloads[0])}, nil
	case request.MessageIs(&workflowservice.RespondActivityTaskFailedRequest{}):
		failed := new(workflowservice.RespondActivityTaskFailedRequest)
		if err := request.UnmarshalTo(failed); err != nil {
			return delivery{}, err
		}
		if err := sendable(failed, "failure"); err != nil {
			return delivery{}, err
		}
		if failed.GetFailure() == nil {
			return delivery{}, errors.New("a failure reply carries no failure")
		}
		return delivery{err: temporal.GetDefaultFailureConverter().FailureToError(failed.GetFailure())}, nil
	default:
		return delivery{}, fmt.Errorf("the SDK cannot send %q", request.GetTypeUrl())
	}
}

// sdkOwned are the fields of a worker response the SDK fills itself.
var sdkOwned = map[protoreflect.Name]bool{"task_token": true, "namespace": true, "identity": true}

// sendable refuses a reply that sets a field the SDK cannot send: only the one field the activity
// returns, and the fields the SDK owns, may be set.
func sendable(request proto.Message, returned protoreflect.Name) error {
	var unsent []string
	request.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		if fd.Name() != returned && !sdkOwned[fd.Name()] {
			unsent = append(unsent, string(fd.Name()))
		}
		return true
	})
	if len(unsent) > 0 {
		return fmt.Errorf("the SDK cannot send the reply's %s", strings.Join(unsent, ", "))
	}
	return nil
}

// work opens a window after a continued read and serves the callback if the worker takes a
// task through it. The window waits the poll interval for a poll; once one is admitted, it waits
// for that poll's outcome. took reports a task was taken; ok false means the exchange ended.
func (x *exchange) work(ctx context.Context, interval time.Duration) (took bool, ok bool) {
	g := x.driver.gate
	w := g.openFor(x)
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		select {
		case outcome := <-w.outcomes:
			switch outcome {
			case pollEmpty:
				return false, true
			case pollTook:
				return true, x.callback(ctx, w)
			default:
			}
		case <-timer.C:
			if g.closeIdle(w) {
				return false, true
			}
		case <-ctx.Done():
			g.closeIdle(w)
			if x.ctx.Err() == nil {
				x.fault("read timeout without Model acceptance")
			}
			return false, false
		}
	}
}

// callback reports the worker's call for the task taken in w and hands the Model's decision back
// to the activity.
func (x *exchange) callback(ctx context.Context, w *window) bool {
	g := x.driver.gate
	defer g.finish(w)
	var c *workerCall
	select {
	case c = <-x.calls:
	case <-ctx.Done():
		if x.ctx.Err() == nil {
			x.fault("the worker took a task and never called back")
		}
		return false
	}
	if c.task == nil {
		x.fault("%v: the worker's task was not the one captured", errMalformed)
		return false
	}
	task, err := pack(c.task)
	if err != nil {
		x.fault("%v: %v", errMalformed, err)
		return false
	}
	payload, err := pack(&realizationv1.Callback{Task: task})
	if err != nil {
		x.fault("%v: %v", errMalformed, err)
		return false
	}
	ch, ok := x.emit(runner.EventCallback, payload, x.driver.calls.Add(1), "")
	if !ok {
		return false
	}
	d, ok := x.await(ctx, ch)
	if !ok {
		if x.ctx.Err() == nil {
			x.fault("read timeout without Model acceptance")
		}
		return false
	}
	c.reply <- d
	return d.Refusal == nil
}

// execute is the activity every task runs: it asks the Model, through the exchange that let the
// worker take the task, and returns the Model's reply.
func (d *Driver) execute(ctx context.Context, _ converter.EncodedValues) (converter.RawValue, error) {
	return d.deliver(ctx, activity.GetInfo(ctx).TaskToken)
}

// deliver hands a task to its exchange and returns the Model's reply. The Driver never answers a
// task the Model did not answer: a refused, unclaimed or abandoned task is held until Close has
// terminated what the Driver started, and only then released.
func (d *Driver) deliver(ctx context.Context, token []byte) (converter.RawValue, error) {
	w, task := d.gate.claim(token)
	if w == nil {
		return d.hold(ctx)
	}
	defer d.gate.finish(w)
	c := &workerCall{task: task, reply: make(chan decision, 1)}
	select {
	case w.x.calls <- c:
	case <-w.x.ctx.Done():
		return d.hold(ctx)
	case <-ctx.Done():
		return converter.RawValue{}, ctx.Err()
	}
	select {
	case got := <-c.reply:
		if got.Refusal != nil {
			return d.hold(ctx)
		}
		return got.delivery.value, got.delivery.err
	case <-w.x.ctx.Done():
		return d.hold(ctx)
	case <-ctx.Done():
		return converter.RawValue{}, ctx.Err()
	}
}

// hold keeps an activity the Model did not answer until the Driver releases it.
func (d *Driver) hold(ctx context.Context) (converter.RawValue, error) {
	select {
	case <-d.released:
		return converter.RawValue{}, errReleased
	case <-ctx.Done():
		return converter.RawValue{}, ctx.Err()
	}
}

// errReleased ends an activity the Model did not answer, once Close has terminated it.
var errReleased = errors.New("the Run ended without the Model answering this task")

// observeResponses records a worker response the SDK completed for a task the Model did not
// answer: a teardown fault, never evidence.
func (d *Driver) observeResponses(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	err := invoke(ctx, method, req, reply, cc, opts...)
	switch method {
	case workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName,
		workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName,
		workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName:
		if err == nil && d.isReleased() {
			d.teardown(fmt.Errorf("the SDK completed a task after the Run with %s", method[strings.LastIndexByte(method, '/')+1:]))
		}
	default:
	}
	return err
}

// startWorker starts the Driver's SDK worker once, when a read first lets it take a task.
func (d *Driver) startWorker(context.Context) error {
	d.workerMu.Lock()
	defer d.workerMu.Unlock()
	if d.worker != nil || d.workerErr != nil {
		return d.workerErr
	}
	sdk, err := client.NewLazyClient(client.Options{
		HostPort:  d.opts.Address,
		Namespace: d.opts.Namespace,
		Logger:    sdklog.NewStructuredLogger(slog.New(slog.DiscardHandler)),
		ConnectionOptions: client.ConnectionOptions{
			DialOptions: []grpc.DialOption{grpc.WithChainUnaryInterceptor(d.gate.intercept, d.observeResponses)},
		},
	})
	if err != nil {
		d.workerErr = err
		return err
	}
	w := worker.New(sdk, d.opts.TaskQueue, worker.Options{DisableWorkflowWorker: true, WorkerStopTimeout: time.Second})
	w.RegisterDynamicActivity(d.execute, activity.DynamicRegisterOptions{})
	if err := w.Start(); err != nil {
		sdk.Close()
		d.workerErr = err
		return err
	}
	d.sdk, d.worker = sdk, w
	return nil
}
