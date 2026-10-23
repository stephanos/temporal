package act

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	"google.golang.org/protobuf/proto"
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

// gate admits the SDK worker's task polls only while a read exchange asks for a worker task, one
// poll at a time, and keeps the exchange's reads paused while that poll or its task is
// outstanding: a read between a taken task and its Answer would see a delivery the Model has not
// been told about.
type gate struct {
	timeout time.Duration

	mu      sync.Mutex
	changed chan struct{}
	open    *exchange
	owner   *exchange
	token   []byte
	task    *workflowservice.PollActivityTaskQueueResponse
}

func newGate(timeout time.Duration) *gate {
	return &gate{timeout: timeout, changed: make(chan struct{})}
}

func (g *gate) broadcast() {
	close(g.changed)
	g.changed = make(chan struct{})
}

// intercept holds every PollActivityTaskQueue until a read opens the gate, bounds the admitted
// poll, and captures the actual task it returns.
func (g *gate) intercept(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	if method != workflowservice.WorkflowService_PollActivityTaskQueue_FullMethodName {
		return invoke(ctx, method, req, reply, cc, opts...)
	}
	x, err := g.admit(ctx)
	if err != nil {
		return err
	}
	pollCtx, cancel := context.WithTimeout(ctx, g.timeout)
	err = invoke(pollCtx, method, req, reply, cc, opts...)
	cancel()
	task, _ := reply.(*workflowservice.PollActivityTaskQueueResponse)
	if err == nil && len(task.GetTaskToken()) > 0 {
		g.took(x, proto.Clone(task).(*workflowservice.PollActivityTaskQueueResponse))
		x.notify(pollTook)
	} else {
		g.finish(x)
		x.notify(pollEmpty)
	}
	return err
}

func (g *gate) admit(ctx context.Context) (*exchange, error) {
	for {
		g.mu.Lock()
		if x := g.open; x != nil && g.owner == nil {
			g.owner = x
			g.mu.Unlock()
			x.notify(pollAdmitted)
			return x, nil
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

func (g *gate) openFor(x *exchange) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.open = x
	g.broadcast()
}

// closeIdle closes the gate and reports whether no poll of x is outstanding.
func (g *gate) closeIdle(x *exchange) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open == x {
		g.open = nil
	}
	return g.owner != x
}

func (g *gate) took(x *exchange, task *workflowservice.PollActivityTaskQueueResponse) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open == x {
		g.open = nil
	}
	g.token, g.task = task.GetTaskToken(), task
}

// finish releases what x holds: its open gate and its poll or task.
func (g *gate) finish(x *exchange) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.open == x {
		g.open = nil
	}
	if g.owner == x {
		g.owner, g.token, g.task = nil, nil, nil
	}
	g.broadcast()
}

// claim returns the exchange a task belongs to, and the captured task when its token matches.
func (g *gate) claim(token []byte) (*exchange, *workflowservice.PollActivityTaskQueueResponse) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.owner == nil || g.task == nil {
		return nil, nil
	}
	if !bytes.Equal(token, g.token) {
		return g.owner, nil
	}
	return g.owner, g.task
}

func (x *exchange) notify(outcome pollOutcome) {
	select {
	case x.polls <- outcome:
	default:
	}
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
		if failed.GetFailure() == nil {
			return delivery{}, errors.New("a failure reply carries no failure")
		}
		return delivery{err: temporal.GetDefaultFailureConverter().FailureToError(failed.GetFailure())}, nil
	default:
		return delivery{}, fmt.Errorf("the SDK cannot send %q", request.GetTypeUrl())
	}
}

// work opens the gate for one worker task after a continued read and serves the callback if
// the worker takes one. took reports a task was taken; ok false means the exchange ended.
func (x *exchange) work(ctx context.Context, interval time.Duration) (took bool, ok bool) {
	g := x.driver.gate
	g.openFor(x)
	timer := time.NewTimer(interval)
	defer timer.Stop()
	admitted := false
	for {
		select {
		case outcome := <-x.polls:
			switch outcome {
			case pollAdmitted:
				admitted = true
			case pollEmpty:
				return false, true
			case pollTook:
				return true, x.callback(ctx)
			default:
			}
		case <-timer.C:
			if !admitted && g.closeIdle(x) {
				return false, true
			}
			admitted = true
		case <-ctx.Done():
			g.finish(x)
			if x.ctx.Err() == nil {
				x.fault("read timeout without Model acceptance")
			}
			return false, false
		}
	}
}

// callback reports the worker's call for the task this exchange let it take and hands the
// Model's decision back to the activity.
func (x *exchange) callback(ctx context.Context) bool {
	g := x.driver.gate
	defer g.finish(x)
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

func (d *Driver) deliver(ctx context.Context, token []byte) (converter.RawValue, error) {
	x, task := d.gate.claim(token)
	if x == nil {
		return converter.RawValue{}, temporal.NewNonRetryableApplicationError("no read let the worker take this task", "UmpireDriver", nil)
	}
	defer d.gate.finish(x)
	c := &workerCall{task: task, reply: make(chan decision, 1)}
	select {
	case x.calls <- c:
	case <-x.ctx.Done():
		return converter.RawValue{}, x.ctx.Err()
	case <-ctx.Done():
		return converter.RawValue{}, ctx.Err()
	}
	select {
	case got := <-c.reply:
		if got.Refusal != nil {
			return converter.RawValue{}, temporal.NewNonRetryableApplicationError("the Model refused the callback: "+got.Refusal.GetMessage(), "UmpireRefusal", nil)
		}
		return got.delivery.value, got.delivery.err
	case <-x.ctx.Done():
		return converter.RawValue{}, x.ctx.Err()
	case <-ctx.Done():
		return converter.RawValue{}, ctx.Err()
	}
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
			DialOptions: []grpc.DialOption{grpc.WithChainUnaryInterceptor(d.gate.intercept)},
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
