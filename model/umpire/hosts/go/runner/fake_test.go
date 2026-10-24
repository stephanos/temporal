package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/anypb"
	runv1 "umpire.local/model/gen/umpire/run/v1"
	model "umpire.local/model/host"
)

const controlType = "type.googleapis.com/umpire.control.Value"

func control(value byte) *anypb.Any {
	return &anypb.Any{TypeUrl: controlType, Value: []byte{value}}
}

func guest(t *testing.T, variable, name string) []byte {
	t.Helper()
	path := os.Getenv(variable)
	if path == "" {
		target := os.Getenv("CARGO_TARGET_DIR")
		if target == "" {
			target = "../../../../target"
		}
		path = filepath.Join(target, "wasm32-wasip1", "guest", name)
	}
	wasm, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s required; build with model/scripts/tests/build-guests.sh: %v", name, err)
	}
	return wasm
}

func statefulGuest(t *testing.T) []byte {
	return guest(t, "UMPIRE_STATEFUL_WASM", "stateful_control.wasm")
}

// step is one event a fake exchange emits. A zero id or request takes the correct one; async
// emits the next step before this one's decision arrives.
type step struct {
	kind    EventKind
	value   byte
	call    uint64
	fault   string
	id      uint64
	request uint64
	async   bool
}

func (s step) decided() bool {
	return s.kind == EventObservation || s.kind == EventCallback || s.kind == EventAnnotate
}

// good is the control system: Start admits, Deliver polls once, takes a worker callback between
// polls and completes, Finish completes.
var good = map[byte][]step{
	0: {{kind: EventObservation, value: 0}, {kind: EventDone}},
	1: {{kind: EventObservation, value: 20}, {kind: EventCallback, value: 4, call: 7}, {kind: EventObservation, value: 2}, {kind: EventDone}},
	3: {{kind: EventObservation, value: 3}, {kind: EventDone}},
}

// buggy completes Deliver without ever calling the worker.
var buggy = map[byte][]step{
	0: good[0],
	1: {{kind: EventObservation, value: 20}, {kind: EventObservation, value: 2}, {kind: EventDone}},
	3: good[3],
}

type decided struct {
	id       uint64
	decision Decision
}

// fakeDriver acts on a scripted system. Its knobs inject each transport failure.
type fakeDriver struct {
	script       map[byte][]step
	startErr     error
	closeErr     error
	blockRespond bool
	respondErr   error
	hangClose    chan struct{}
	delay        time.Duration
	redact       func(*runv1.Event) bool
	// onStart runs inside Start with Start's ctx, before the exchange exists.
	onStart func(ctx context.Context, clock *testClock, made int)
	// whenWaiting runs once the Runner waits for the exchange's first event; the exchange then
	// stays silent until it is closed.
	whenWaiting func(clock *testClock)
	clock       *testClock

	mu        sync.Mutex
	exchanges []*fakeExchange
}

func (d *fakeDriver) Start(ctx context.Context, request uint64, payload *anypb.Any) (Exchange, error) {
	made := 0
	if d.clock != nil {
		made = d.clock.count(readDeadline)
	}
	if d.onStart != nil {
		d.onStart(ctx, d.clock, made)
	}
	if d.startErr != nil {
		return nil, d.startErr
	}
	if payload.GetTypeUrl() != controlType || len(payload.GetValue()) != 1 {
		return nil, fmt.Errorf("unexpected request %v", payload)
	}
	steps, ok := d.script[payload.Value[0]]
	if !ok {
		return nil, fmt.Errorf("no script for request %d", payload.Value[0])
	}
	exchangeCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	x := &fakeExchange{
		driver: d, request: request, events: make(chan Event), decisions: make(chan decided, len(steps)),
		ctx: exchangeCtx, cancel: cancel, done: make(chan struct{}), outstanding: map[uint64]bool{},
	}
	d.mu.Lock()
	d.exchanges = append(d.exchanges, x)
	d.mu.Unlock()
	if d.whenWaiting != nil {
		go x.silent(made)
	} else {
		go x.run(steps)
	}
	return x, nil
}

func (d *fakeDriver) Redact(event *runv1.Event) bool {
	return d.redact != nil && d.redact(event)
}

func (d *fakeDriver) all() []*fakeExchange {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]*fakeExchange(nil), d.exchanges...)
}

type fakeExchange struct {
	driver    *fakeDriver
	request   uint64
	events    chan Event
	decisions chan decided
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}

	mu          sync.Mutex
	closed      bool
	outstanding map[uint64]bool
	received    []decided
}

func (x *fakeExchange) run(steps []step) {
	defer close(x.done)
	defer close(x.events)
	var next uint64
	pending := 0
	record := func(got decided) {
		x.mu.Lock()
		x.received = append(x.received, got)
		x.mu.Unlock()
	}
	await := func() bool {
		for ; pending > 0; pending-- {
			select {
			case got := <-x.decisions:
				record(got)
			case <-x.ctx.Done():
				// Respond accepted a decision before the Runner closed the exchange: the Driver
				// received it, whichever of the two the select happened to see first.
				for {
					select {
					case got := <-x.decisions:
						record(got)
					default:
						return false
					}
				}
			}
		}
		return true
	}
	for _, s := range steps {
		next++
		event := Event{ID: next, Request: x.request, Kind: s.kind, Call: s.call, Fault: s.fault}
		if s.id != 0 {
			event.ID = s.id
		}
		if s.request != 0 {
			event.Request = s.request
		}
		if s.kind == EventObservation || s.kind == EventCallback || s.kind == EventAnnotate {
			event.Payload = control(s.value)
		}
		if s.kind == EventDone && !await() {
			return
		}
		if s.decided() {
			x.mu.Lock()
			x.outstanding[event.ID] = true
			x.mu.Unlock()
		}
		if x.driver.delay > 0 {
			select {
			case <-time.After(x.driver.delay):
			case <-x.ctx.Done():
				return
			}
		}
		select {
		case x.events <- event:
		case <-x.ctx.Done():
			return
		}
		if s.decided() {
			pending++
			if !s.async && !await() {
				return
			}
		}
	}
	// A script that ends without completing closes its stream prematurely.
	if last := steps[len(steps)-1].kind; last == EventDone || last == EventFault {
		<-x.ctx.Done()
	}
}

// silent waits until the Runner's read deadline after Start exists, acts, and sends nothing.
func (x *fakeExchange) silent(made int) {
	defer close(x.done)
	defer close(x.events)
	x.driver.clock.await(readDeadline, made+1)
	x.driver.whenWaiting(x.driver.clock)
	<-x.ctx.Done()
}

func (x *fakeExchange) Events() <-chan Event { return x.events }

func (x *fakeExchange) Respond(ctx context.Context, event uint64, decision Decision) error {
	x.mu.Lock()
	switch {
	case x.closed:
		x.mu.Unlock()
		return errors.New("exchange closed")
	case !x.outstanding[event]:
		x.mu.Unlock()
		return fmt.Errorf("unknown event %d", event)
	}
	delete(x.outstanding, event)
	x.mu.Unlock()
	if x.driver.blockRespond {
		x.driver.clock.expire(readDeadline)
		<-ctx.Done()
		return ctx.Err()
	}
	if x.driver.respondErr != nil {
		return x.driver.respondErr
	}
	select {
	case x.decisions <- decided{id: event, decision: decision}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (x *fakeExchange) Close() error {
	x.mu.Lock()
	x.closed = true
	x.mu.Unlock()
	x.cancel()
	if x.driver.hangClose != nil {
		x.driver.clock.expire(readDeadline)
		<-x.driver.hangClose
	}
	<-x.done
	return x.driver.closeErr
}

func (x *fakeExchange) joined() bool {
	select {
	case <-x.done:
		return true
	default:
		return false
	}
}

func (x *fakeExchange) decisionsReceived() []decided {
	x.mu.Lock()
	defer x.mu.Unlock()
	return append([]decided(nil), x.received...)
}

// harness makes Runners over one guest whose Drivers come from a factory, counting both.
type harness struct {
	t       *testing.T
	wasm    []byte
	hosts   atomic.Int32
	clock   *testClock
	drivers []*fakeDriver
	mu      sync.Mutex
	make    func(run int) *fakeDriver
}

func newHarness(t *testing.T, wasm []byte, makeDriver func(run int) *fakeDriver) *harness {
	return &harness{t: t, wasm: wasm, make: makeDriver, clock: newTestClock()}
}

func (h *harness) runner() *Runner {
	r := New(h.wasm, func(context.Context) (Driver, error) {
		h.mu.Lock()
		defer h.mu.Unlock()
		driver := h.make(len(h.drivers))
		driver.clock = h.clock
		h.drivers = append(h.drivers, driver)
		return driver, nil
	})
	if h.make == nil {
		r = New(h.wasm, nil)
	}
	r.deadline = h.clock.deadline
	r.newHost = func(ctx context.Context, wasm []byte) (*model.Host, error) {
		h.hosts.Add(1)
		return model.New(ctx, wasm)
	}
	return r
}

func (h *harness) run(mode Mode, opts Options) *Run {
	h.t.Helper()
	run, err := h.runner().Run(context.Background(), mode, opts)
	if err != nil {
		h.t.Fatalf("%s: %v", mode, err)
	}
	return run
}

func driverFor(script map[byte][]step) func(int) *fakeDriver {
	return func(int) *fakeDriver { return &fakeDriver{script: script} }
}

func configured(seed uint64) Options {
	return Options{World: control(0), Seed: seed}
}
