package act

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/umpire/model/v1"
	realizationv1 "umpire.local/model/umpire/realization/v1"
	"umpire.local/model/umpire/runner"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// exchange is one proposed request in flight. One goroutine runs it: it numbers and sends the
// events, and waits for each decision before its next dependent read or worker reply.
type exchange struct {
	driver  *Driver
	request uint64
	ctx     context.Context
	cancel  context.CancelFunc
	events  chan runner.Event
	done    chan struct{}
	// calls receives the worker's callback for a task this exchange let the worker take.
	calls chan *workerCall

	mu      sync.Mutex
	next    uint64
	closed  bool
	waiting map[uint64]*waiter
}

type waiter struct {
	kind     runner.EventKind
	decision chan decision
}

// decision is a Runner decision, with a callback's reply already converted for the SDK.
type decision struct {
	runner.Decision
	delivery delivery
}

func newExchange(d *Driver, request uint64) *exchange {
	ctx, cancel := context.WithCancel(context.Background())
	return &exchange{
		driver: d, request: request, ctx: ctx, cancel: cancel,
		events: make(chan runner.Event), done: make(chan struct{}),
		calls:   make(chan *workerCall, 1),
		waiting: map[uint64]*waiter{},
	}
}

func (x *exchange) Events() <-chan runner.Event { return x.events }

func (x *exchange) Respond(_ context.Context, event uint64, d runner.Decision) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.closed {
		return fmt.Errorf("event %d: the exchange is closed", event)
	}
	w := x.waiting[event]
	if w == nil {
		return fmt.Errorf("event %d awaits no decision", event)
	}
	resolved := decision{Decision: d}
	switch {
	case d.Refusal != nil:
	case w.kind == runner.EventObservation && d.Judgment != nil:
	case w.kind == runner.EventCallback && d.Reply != nil:
		var err error
		if resolved.delivery, err = deliverable(d.Reply); err != nil {
			return fmt.Errorf("event %d: %w", event, err)
		}
	default:
		return fmt.Errorf("event %d: the decision does not answer a %s", event, kindName(w.kind))
	}
	delete(x.waiting, event)
	w.decision <- resolved
	return nil
}

func (x *exchange) Close() error {
	x.mu.Lock()
	x.closed = true
	x.mu.Unlock()
	x.cancel()
	<-x.done
	x.driver.forget(x)
	return nil
}

func kindName(kind runner.EventKind) string {
	if kind == runner.EventCallback {
		return "callback"
	}
	return "observation"
}

// emit sends one event and, for an Observation or Callback, returns where its decision arrives.
// It returns nil when the exchange was closed first.
func (x *exchange) emit(kind runner.EventKind, payload *anypb.Any, workerCall uint64, fault string) (chan decision, bool) {
	x.mu.Lock()
	x.next++
	event := runner.Event{ID: x.next, Request: x.request, Kind: kind, Payload: payload, Call: workerCall, Fault: fault}
	var ch chan decision
	if kind == runner.EventObservation || kind == runner.EventCallback {
		ch = make(chan decision, 1)
		x.waiting[event.ID] = &waiter{kind: kind, decision: ch}
	}
	x.mu.Unlock()
	select {
	case x.events <- event:
		return ch, true
	case <-x.ctx.Done():
		return nil, false
	}
}

func (x *exchange) fault(format string, values ...any) {
	x.emit(runner.EventFault, nil, 0, fmt.Sprintf(format, values...))
}

// await waits for a decision. ok is false when the exchange ended or the wait's bound passed.
func (x *exchange) await(ctx context.Context, ch chan decision) (decision, bool) {
	select {
	case d := <-ch:
		return d, true
	case <-ctx.Done():
		return decision{}, false
	}
}

// observe reports an Observation and waits for the Model's judgment. It returns the decision,
// or ok false once the exchange has ended.
func (x *exchange) observe(ctx context.Context, observation *realizationv1.Observation) (*modelv1.JudgeResponse, bool) {
	payload, err := pack(observation)
	if err != nil {
		x.fault("%v: %v", errMalformed, err)
		return nil, false
	}
	ch, ok := x.emit(runner.EventObservation, payload, 0, "")
	if !ok {
		return nil, false
	}
	d, ok := x.await(ctx, ch)
	if !ok {
		if x.ctx.Err() == nil {
			x.fault("read timeout without Model acceptance")
		}
		return nil, false
	}
	// A refusal or an Unexplained judgment ends the Run; the Runner closes the exchange.
	if d.Refusal != nil || d.Judgment.GetUnexplained() != nil {
		return nil, false
	}
	return d.Judgment, true
}

// unary sends one request and reports its single Observation.
func (x *exchange) unary(request *anypb.Any) {
	ctx, cancel := context.WithTimeout(x.ctx, x.driver.opts.RPCTimeout)
	defer cancel()
	observation, reached, err := call(ctx, x.driver.conn, request)
	answered := reached.answered.Load()
	switch {
	case observation != nil:
		x.driver.started(request, observation)
	case reached.sent.Load():
		// The server may have acted on a call it never answered or answered unreadably.
		x.driver.started(request, nil)
	default:
	}
	switch {
	case x.ctx.Err() != nil:
		return
	case errors.Is(err, errMalformed):
		x.fault("%v", err)
		return
	case !answered:
		x.fault("the server did not answer %s: %v", request.GetTypeUrl(), err)
		return
	}
	judgment, ok := x.observe(x.ctx, observation)
	switch {
	case !ok:
	case judgment.GetContinuePolling():
		x.fault("the Model asked to read %s again, which is not a read", observation.GetMethod())
	default:
		x.emit(runner.EventDone, nil, 0, "")
	}
}

// read repeats one read until the Model accepts a final observation. The instruction's fields
// are its only mechanics: the poll interval spaces continued reads, the backoff spaces retries
// of a read the server did not answer, and the read timeout bounds the whole exchange.
func (x *exchange) read(instruction *temporalv1.ObserveActivity) {
	timeout := time.Duration(instruction.GetReadTimeoutMs()) * time.Millisecond
	if limit := x.driver.opts.MaxReadTimeout; limit > 0 && (timeout <= 0 || timeout > limit) {
		timeout = limit
	}
	ctx, cancel := context.WithTimeout(x.ctx, timeout)
	defer cancel()
	tasks := instruction.GetWorkerTasks()
	if tasks > 0 {
		if err := x.driver.ensureWorker(ctx); err != nil {
			x.fault("worker: %v", err)
			return
		}
	}
	interval := time.Duration(instruction.GetPollIntervalMs()) * time.Millisecond
	backoff := time.Duration(instruction.GetBackoffMs()) * time.Millisecond
	for {
		observation, reached, err := call(ctx, x.driver.conn, instruction.GetRequest())
		answered := reached.answered.Load()
		switch {
		case x.ctx.Err() != nil:
			return
		case errors.Is(err, errMalformed):
			x.fault("%v", err)
			return
		case !answered && ctx.Err() != nil:
			x.fault("read timeout without Model acceptance")
			return
		case !answered:
			if !x.sleep(ctx, backoff) {
				return
			}
			continue
		default:
		}
		judgment, ok := x.observe(ctx, observation)
		if !ok {
			return
		}
		if !judgment.GetContinuePolling() {
			x.emit(runner.EventDone, nil, 0, "")
			return
		}
		if tasks > 0 {
			took, ok := x.work(ctx, interval)
			if !ok {
				return
			}
			if took {
				tasks--
			}
		}
		if !x.sleep(ctx, interval) {
			return
		}
	}
}

// sleep waits d, faulting when the read timeout ends first.
func (x *exchange) sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		if x.ctx.Err() == nil {
			x.fault("read timeout without Model acceptance")
		}
		return false
	}
}

// run dispatches the proposal to its exchange kind and ends the event stream.
func (x *exchange) run(request *anypb.Any) {
	defer close(x.done)
	defer close(x.events)
	if request.MessageIs(&temporalv1.ObserveActivity{}) {
		instruction := new(temporalv1.ObserveActivity)
		if err := request.UnmarshalTo(instruction); err != nil {
			x.fault("the read instruction does not decode: %v", err)
			return
		}
		x.read(instruction)
		return
	}
	x.unary(request)
}

// validStart reports why a proposal cannot be sent at all, or nil.
func validStart(request *anypb.Any) error {
	if request.MessageIs(&temporalv1.ObserveActivity{}) {
		instruction := new(temporalv1.ObserveActivity)
		if err := request.UnmarshalTo(instruction); err != nil {
			return err
		}
		if instruction.GetRequest() == nil {
			return errors.New("the read instruction names no read")
		}
		// The poll interval is also how long a worker window waits for a poll.
		if instruction.GetPollIntervalMs() == 0 || instruction.GetBackoffMs() == 0 || instruction.GetReadTimeoutMs() == 0 {
			return errors.New("the read instruction needs a poll interval, a backoff and a read timeout")
		}
		request = instruction.GetRequest()
	}
	_, err := method(request)
	return err
}
