package control

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"go.temporal.io/server/api/historyservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/chasm"
	"go.temporal.io/server/common/rpc/grpcfaults"
	serviceerrors "go.temporal.io/server/common/serviceerror"
	"go.temporal.io/server/common/testing/testhooks"
)

var ErrConflict = errors.New("activity delivery is already held")

// Injector applies one test hook to the server under test and returns its removal.
type Injector func(testhooks.Hook) func()

// Deliveries holds the validated dispatches of named activities inside one in-process server and
// observes what the server's authoritative admission decided for each released one. Admission is
// read from the response of history's RecordActivityTaskStarted, after its handler returned: a
// success is an attempt the durable update committed, an obsolete-task refusal is a decision that
// committed nothing, and any other answer decides nothing that can be told, so it is no decision.
type Deliveries struct {
	mu      sync.Mutex
	holds   map[string]*Held
	closed  bool
	removes []func()
}

func NewDeliveries(inject Injector) (*Deliveries, error) {
	if inject == nil {
		return nil, ErrInvalid
	}
	d := &Deliveries{holds: make(map[string]*Held)}
	d.removes = append(d.removes,
		inject(testhooks.NewHook(testhooks.ActivityDispatch, d.dispatch)),
		inject(testhooks.NewHook(testhooks.GRPCResponseFaultGeneratorByNamespaceID, grpcfaults.ResponseCallback(d.answered))),
	)
	return d, nil
}

// Hold arms a hold on every dispatch of the activity until the returned Held releases it or closes.
func (d *Deliveries) Hold(activityID string) (*Held, error) {
	if d == nil || activityID == "" {
		return nil, ErrInvalid
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil, ErrClosed
	}
	if _, held := d.holds[activityID]; held {
		return nil, ErrConflict
	}
	h := &Held{owner: d, activityID: activityID, gate: NewDeliveryGate(activityID), decided: make(chan *testpilotspb.DeliveryAdmission, 1)}
	d.holds[activityID] = h
	return h, nil
}

// Close cancels every delivery still held and removes the hooks.
func (d *Deliveries) Close() {
	if d == nil {
		return
	}
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	holds := d.holds
	d.holds = map[string]*Held{}
	removes := d.removes
	d.mu.Unlock()
	for _, h := range holds {
		h.gate.Close()
	}
	for _, remove := range removes {
		if remove != nil {
			remove()
		}
	}
}

func (d *Deliveries) lookup(activityID string) *Held {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.holds[activityID]
}

func (d *Deliveries) dispatch(ctx context.Context, delivery testhooks.ActivityDelivery) error {
	h := d.lookup(delivery.Execution.BusinessID)
	if h == nil {
		return nil
	}
	h.arrived(delivery)
	return h.gate.Arrive(ctx, delivery.Execution.BusinessID)
}

func (d *Deliveries) answered(_ context.Context, _ string, request, response any, err error) *grpcfaults.Outcome {
	started, ok := request.(*historyservice.RecordActivityTaskStartedRequest)
	if !ok {
		return nil
	}
	ref, refErr := chasm.DeserializeComponentRef(started.GetComponentRef())
	if refErr != nil {
		return nil
	}
	if h := d.lookup(ref.BusinessID); h != nil {
		h.answered(ref.ExecutionKey, started.GetStamp(), response, err)
	}
	return nil
}

// Held is the hold on one activity's dispatches.
type Held struct {
	owner      *Deliveries
	activityID string
	gate       *DeliveryGate[string]
	decided    chan *testpilotspb.DeliveryAdmission

	mu       sync.Mutex
	first    *testhooks.ActivityDelivery
	released bool
}

func (h *Held) arrived(delivery testhooks.ActivityDelivery) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.first == nil {
		h.first = &delivery
	}
}

// answered keeps the first decision of admission for the held delivery once it is released. An
// answer for another run or another stamp is another delivery's, and decides nothing of this one.
func (h *Held) answered(execution chasm.ExecutionKey, stamp int32, response any, err error) {
	h.mu.Lock()
	first, released := h.first, h.released
	h.mu.Unlock()
	if !released || first == nil || execution != first.Execution || stamp != first.Stamp {
		return
	}
	admission := &testpilotspb.DeliveryAdmission{ActivityId: execution.BusinessID, ActivityRunId: execution.RunID, DeliveryId: strconv.FormatInt(int64(stamp), 10)}
	var obsolete *serviceerrors.ObsoleteMatchingTask
	switch started, ok := response.(*historyservice.RecordActivityTaskStartedResponse); {
	case err == nil && ok && started != nil:
		admission.Decision, admission.Attempt = testpilotspb.DELIVERY_ADMISSION_DECISION_ADMITTED, started.GetAttempt()
	case errors.As(err, &obsolete):
		admission.Decision = testpilotspb.DELIVERY_ADMISSION_DECISION_REJECTED
	default:
		return
	}
	select {
	case h.decided <- admission:
	default:
	}
}

// Await waits until the server holds a dispatch of the activity, and returns it.
func (h *Held) Await(ctx context.Context) (testhooks.ActivityDelivery, error) {
	if h == nil {
		return testhooks.ActivityDelivery{}, ErrInvalid
	}
	if err := h.gate.WaitHeld(ctx); err != nil {
		return testhooks.ActivityDelivery{}, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return *h.first, nil
}

// Release lets the held dispatch reach matching, runs deliver, which polls as a worker would so
// that admission is asked, and returns admission's decision once observed. deliver is canceled
// when the decision is in, and its own failure before then is the release's.
func (h *Held) Release(ctx context.Context, deliver func(context.Context) error) (*testpilotspb.DeliveryAdmission, error) {
	if h == nil || ctx == nil || deliver == nil {
		return nil, ErrInvalid
	}
	h.mu.Lock()
	if h.first == nil {
		h.mu.Unlock()
		return nil, ErrNotHeld
	}
	h.released = true
	h.mu.Unlock()
	if err := h.gate.Release(); err != nil {
		return nil, err
	}
	polling, stop := context.WithCancel(ctx)
	defer stop()
	delivered := make(chan error, 1)
	go func() { delivered <- deliver(polling) }()
	select {
	case admission := <-h.decided:
		stop()
		<-delivered
		return admission, nil
	case err := <-delivered:
		if err == nil {
			err = errors.New("delivery ended before admission decided")
		}
		return nil, err
	case <-ctx.Done():
		<-delivered
		return nil, ctx.Err()
	}
}

// Close cancels a delivery still held and stops observing the activity.
func (h *Held) Close() {
	if h == nil {
		return
	}
	h.gate.Close()
	h.owner.mu.Lock()
	if h.owner.holds[h.activityID] == h {
		delete(h.owner.holds, h.activityID)
	}
	h.owner.mu.Unlock()
}
