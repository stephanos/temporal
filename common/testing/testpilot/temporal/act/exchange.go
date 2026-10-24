package act

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	realizationv1 "umpire.local/model/gen/umpire/realization/v1"
	"umpire.local/model/runner"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

// exchange is one proposed request in flight. One goroutine runs it: it numbers and sends the
// events, and waits for each decision before its next dependent read.
type exchange struct {
	driver  *Driver
	request uint64
	ctx     context.Context
	cancel  context.CancelFunc
	events  chan runner.Event
	done    chan struct{}

	mu       sync.Mutex
	next     uint64
	closed   bool
	waiting  map[uint64]*waiter
	buffered *anypb.Any
}

type waiter struct {
	kind     runner.EventKind
	decision chan decision
}

// decision is the judgment returned for one observation.
type decision = runner.Decision

func newExchange(d *Driver, request uint64) *exchange {
	ctx, cancel := context.WithCancel(context.Background())
	return &exchange{
		driver: d, request: request, ctx: ctx, cancel: cancel,
		events: make(chan runner.Event), done: make(chan struct{}),
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
	resolved := d
	switch {
	case d.Refusal != nil:
	case w.kind == runner.EventObservation && d.Judgment != nil:
	default:
		return fmt.Errorf("event %d: the decision does not answer an observation", event)
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
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.buffered != nil {
		evidence := x.buffered
		x.buffered = nil
		return &acknowledgedError{evidence: evidence}
	}
	return nil
}

// emit sends one event and returns where an Observation decision arrives.
// It returns nil when the exchange was closed first.
func (x *exchange) emit(kind runner.EventKind, payload *anypb.Any, fault string) (chan decision, bool) {
	x.mu.Lock()
	x.next++
	event := runner.Event{ID: x.next, Request: x.request, Kind: kind, Payload: payload, Fault: fault}
	var ch chan decision
	if kind == runner.EventObservation {
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

type acknowledgedError struct{ evidence *anypb.Any }

func (*acknowledgedError) Error() string {
	return "exchange closed before its acknowledged request could be judged"
}
func (e *acknowledgedError) Evidence() *anypb.Any { return e.evidence }

func (x *exchange) fault(format string, values ...any) {
	x.mu.Lock()
	evidence := x.buffered
	x.mu.Unlock()
	if _, ok := x.emit(runner.EventFault, evidence, fmt.Sprintf(format, values...)); ok {
		x.mu.Lock()
		x.buffered = nil
		x.mu.Unlock()
	}
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
	return x.observePayload(ctx, payload, observation.GetMethod() == workflowservice.WorkflowService_RequestCancelActivityExecution_FullMethodName)
}

func (x *exchange) observePayload(ctx context.Context, payload *anypb.Any, acknowledged bool) (*modelv1.JudgeResponse, bool) {
	ch, ok := x.emit(runner.EventObservation, payload, "")
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
	if acknowledged {
		x.mu.Lock()
		x.buffered = nil
		x.mu.Unlock()
	}
	if d.Refusal != nil || d.Judgment.GetUnexplained() != nil {
		return nil, false
	}
	return d.Judgment, true
}

// unary sends one request and reports its single Observation.
func (x *exchange) unary(request *anypb.Any, held uint64, notFoundRead *temporalv1.ObserveActivity) {
	ctx, cancel := context.WithTimeout(x.ctx, x.driver.opts.RPCTimeout)
	defer cancel()
	observation, reached, err := call(ctx, x.driver.conn, request)
	answered := reached.answered.Load()
	if answered && held != 0 {
		x.driver.polls.acknowledged(held)
	}
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
	if notFoundRead != nil && observation.GetError().GetCode() == int32(codes.NotFound) {
		evidence, err := pack(observation)
		if err != nil {
			x.fault("pack acknowledged request: %v", err)
			return
		}
		x.mu.Lock()
		x.buffered = evidence
		x.mu.Unlock()
		if !x.reconcile(notFoundRead) {
			return
		}
	}
	judgment, ok := x.observe(x.ctx, observation)
	switch {
	case !ok:
	case judgment.GetContinuePolling():
		x.fault("the Model asked to read %s again, which is not a read", observation.GetMethod())
	default:
		x.emit(runner.EventDone, nil, "")
	}
}

// reconcile delivers a fresh read before the buffered acknowledgment. Network order is
// request, read; judgment order is read, acknowledgment, so worker progress is checked first.
func (x *exchange) reconcile(instruction *temporalv1.ObserveActivity) bool {
	timeout := time.Duration(instruction.GetReadTimeoutMs()) * time.Millisecond
	if limit := x.driver.opts.MaxReadTimeout; limit > 0 && timeout > limit {
		timeout = limit
	}
	ctx, cancel := context.WithTimeout(x.ctx, timeout)
	defer cancel()
	for {
		observation, reached, err := call(ctx, x.driver.conn, instruction.GetRequest())
		switch {
		case x.ctx.Err() != nil:
			return false
		case errors.Is(err, errMalformed):
			x.fault("%v", err)
			return false
		case !reached.answered.Load() && ctx.Err() != nil:
			x.fault("read timeout without Model acceptance")
			return false
		case !reached.answered.Load():
			if !x.sleep(ctx, time.Duration(instruction.GetBackoffMs())*time.Millisecond) {
				return false
			}
			continue
		default:
		}
		current, err := pack(observation)
		if err != nil {
			x.fault("pack reconciliation read: %v", err)
			return false
		}
		x.mu.Lock()
		ack := x.buffered
		x.mu.Unlock()
		payload, err := pack(&temporalv1.ReconciledObservation{Acknowledged: ack, Current: current})
		if err != nil {
			x.fault("pack reconciliation evidence: %v", err)
			return false
		}
		judgment, ok := x.observePayload(ctx, payload, true)
		if !ok {
			return false
		}
		if !judgment.GetContinuePolling() {
			x.fault("reconciliation read ended the pending request before its acknowledgment")
			return false
		}
		return true
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
	interval := time.Duration(instruction.GetPollIntervalMs()) * time.Millisecond
	backoff := time.Duration(instruction.GetBackoffMs()) * time.Millisecond
	for {
		if err := x.driver.executionError(); err != nil {
			x.fault("worker executor: %v", err)
			return
		}
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
			x.emit(runner.EventDone, nil, "")
			return
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
	if err := x.driver.executionError(); err != nil {
		x.fault("worker executor: %v", err)
		return
	}
	var held uint64
	hold := false
	var notFoundRead *temporalv1.ObserveActivity
	if request.MessageIs(&temporalv1.TemporalInstruction{}) {
		instruction := new(temporalv1.TemporalInstruction)
		if err := request.UnmarshalTo(instruction); err != nil {
			x.fault("instruction does not decode: %v", err)
			return
		}
		hold = instruction.GetHoldWorkerPolls()
		if instruction.GetNotFoundRead() != nil {
			notFoundRead = new(temporalv1.ObserveActivity)
			if err := instruction.GetNotFoundRead().UnmarshalTo(notFoundRead); err != nil {
				x.fault("reconciliation read does not decode: %v", err)
				return
			}
		}
		request = instruction.GetRequest()
	}
	if hold && x.driver.polls == nil {
		x.fault("worker poll hold is unsupported for an external worker")
		return
	}
	if x.driver.polls != nil && !request.MessageIs(&workflowservice.StartActivityExecutionRequest{}) {
		var err error
		held, err = x.driver.polls.prepare(hold)
		if err != nil {
			x.fault("worker poll hold is unsupported: %v", err)
			return
		}
		if !hold {
			held = 0
		}
	}
	if request.MessageIs(&temporalv1.ObserveActivity{}) {
		instruction := new(temporalv1.ObserveActivity)
		if err := request.UnmarshalTo(instruction); err != nil {
			x.fault("the read instruction does not decode: %v", err)
			return
		}
		x.read(instruction)
		return
	}
	x.unary(request, held, notFoundRead)
}

// validStart reports why a proposal cannot be sent at all, or nil.
func validStart(request *anypb.Any) error {
	if request.MessageIs(&temporalv1.TemporalInstruction{}) {
		instruction := new(temporalv1.TemporalInstruction)
		if err := request.UnmarshalTo(instruction); err != nil {
			return err
		}
		if instruction.GetRequest() == nil {
			return errors.New("the instruction names no request")
		}
		if instruction.GetNotFoundRead() != nil {
			if !instruction.GetNotFoundRead().MessageIs(&temporalv1.ObserveActivity{}) {
				return errors.New("reconciliation instruction names no read")
			}
			if err := validStart(instruction.GetNotFoundRead()); err != nil {
				return err
			}
		}
		request = instruction.GetRequest()
	}
	if request.MessageIs(&temporalv1.ObserveActivity{}) {
		instruction := new(temporalv1.ObserveActivity)
		if err := request.UnmarshalTo(instruction); err != nil {
			return err
		}
		if instruction.GetRequest() == nil {
			return errors.New("the read instruction names no read")
		}
		if instruction.GetPollIntervalMs() == 0 || instruction.GetBackoffMs() == 0 || instruction.GetReadTimeoutMs() == 0 {
			return errors.New("the read instruction needs a poll interval, a backoff and a read timeout")
		}
		request = instruction.GetRequest()
	}
	_, err := method(request)
	return err
}
