package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/model/clock"
)

// The clock conformance corpus pins the shared simulation clock, which stays
// authoritative for the native backend and the runner, against the Rust
// duplicate in the cluster helper (fn-178.5). Steps drive the exported
// BrokerClock API; blocking Quiesce calls run on goroutines and the driver waits
// until each one either answered or is parked, so every step is deterministic.
const (
	clockConformanceDirectory = "../../../../rust/wasmhost/tests/testdata/clock_conformance"
	clockConformancePin       = clockConformanceDirectory + "/PINNED_GO_CLOCK_SHA256"
	clockConformanceSchema    = "gomad.clock-conformance/v1"
)

type clockConformanceCorpus struct {
	Schema               string                 `json:"schema"`
	Name                 string                 `json:"name"`
	Source               string                 `json:"source"`
	Origin               string                 `json:"origin"`
	ImplementationSHA256 string                 `json:"clock_implementation_sha256"`
	Clock                clockConformanceClock  `json:"clock"`
	Initial              int64                  `json:"initial"`
	Steps                []clockConformanceStep `json:"steps"`
}

// Kind "broker" is NewBrokerClock (strict, current+1, Register activates);
// "process" is NewProcessClock(forward).
type clockConformanceClock struct {
	Kind    string `json:"kind"`
	Forward bool   `json:"forward"`
}

type clockConformanceStep struct {
	Op         string                 `json:"op"`
	Handle     string                 `json:"handle,omitempty"`
	Other      string                 `json:"other,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Request    uint64                 `json:"request,omitempty"`
	Arrivals   uint32                 `json:"arrivals,omitempty"`
	Generation uint64                 `json:"generation,omitempty"`
	Current    int64                  `json:"current,omitempty"`
	Deadline   int64                  `json:"deadline,omitempty"`
	Expected   clockConformanceOutput `json:"expected"`
}

type clockConformanceResponse struct {
	Handle     string `json:"handle"`
	Generation uint64 `json:"generation"`
	Kind       uint8  `json:"kind"`
	Time       int64  `json:"time"`
	Error      string `json:"error"`
}

type clockConformanceOutput struct {
	Error     string                     `json:"error"`
	Value     int64                      `json:"value,omitempty"`
	Responses []clockConformanceResponse `json:"responses"`
	Current   int64                      `json:"current"`
	Waiting   []string                   `json:"waiting"`
}

type clockConformanceCall struct {
	handle string
	done   chan clockConformanceResponse
	cancel context.CancelFunc
}

type clockConformanceBuilder struct {
	t       *testing.T
	clock   *clock.BrokerClock
	handles map[string]*clock.BrokerParticipant
	order   []string
	pending []*clockConformanceCall
	issued  *clockConformanceCall
	corpus  clockConformanceCorpus
}

func newClockConformance(t *testing.T, name, source, origin string, kind clockConformanceClock) *clockConformanceBuilder {
	t.Helper()
	b := &clockConformanceBuilder{t: t, handles: map[string]*clock.BrokerParticipant{}, corpus: clockConformanceCorpus{
		Schema: clockConformanceSchema, Name: name, Source: source, Origin: origin,
		ImplementationSHA256: clock.BrokerImplementationSHA256(), Clock: kind, Steps: []clockConformanceStep{},
	}}
	switch kind.Kind {
	case "broker":
		b.clock = clock.NewBrokerClock()
	case "process":
		b.clock = clock.NewProcessClock(kind.Forward)
	default:
		t.Fatalf("clock kind %q", kind.Kind)
	}
	b.corpus.Initial = b.clock.Current()
	return b
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// settle waits until every outstanding Quiesce has answered or is parked in the
// arbiter. A parked call cannot answer without another step.
func (b *clockConformanceBuilder) settle(output *clockConformanceOutput, wait map[*clockConformanceCall]bool) {
	b.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	// The step's own calls go first: a new Quiesce may settle calls that were
	// parked before it reached the arbiter, so they are judged only after it.
	ordered := make([]*clockConformanceCall, 0, len(b.pending))
	for _, call := range b.pending {
		if wait[call] || call == b.issued {
			ordered = append(ordered, call)
		}
	}
	for _, call := range b.pending {
		if !wait[call] && call != b.issued {
			ordered = append(ordered, call)
		}
	}
	b.issued = nil
	remaining := make([]*clockConformanceCall, 0, len(ordered))
	for _, call := range ordered {
		for {
			select {
			case response := <-call.done:
				output.Responses = append(output.Responses, response)
				call.cancel()
				call = nil
			default:
			}
			if call == nil {
				break
			}
			if !wait[call] && b.clock.IsQuiescent(b.handles[call.handle]) {
				remaining = append(remaining, call)
				break
			}
			if time.Now().After(deadline) {
				b.t.Fatalf("clock conformance %s: quiesce for %s neither answered nor parked", b.corpus.Name, call.handle)
			}
			runtime.Gosched()
		}
	}
	b.pending = remaining
	sort.Slice(output.Responses, func(i, j int) bool { return output.Responses[i].Handle < output.Responses[j].Handle })
	output.Current = b.clock.Current()
	for _, call := range b.pending {
		output.Waiting = append(output.Waiting, call.handle)
	}
	sort.Strings(output.Waiting)
}

func (b *clockConformanceBuilder) record(step clockConformanceStep, err error, value int64, wait map[*clockConformanceCall]bool) clockConformanceOutput {
	b.t.Helper()
	output := clockConformanceOutput{Error: errorText(err), Value: value, Responses: []clockConformanceResponse{}, Waiting: []string{}}
	b.settle(&output, wait)
	step.Expected = output
	b.corpus.Steps = append(b.corpus.Steps, step)
	return output
}

func (b *clockConformanceBuilder) waiting(handle string) bool {
	for _, call := range b.pending {
		if call.handle == handle {
			return true
		}
	}
	return false
}

func (b *clockConformanceBuilder) register(name string, inactive bool) string {
	b.t.Helper()
	var participant *clock.BrokerParticipant
	var err error
	op := "register"
	if inactive {
		op = "register_inactive"
		participant, err = b.clock.RegisterInactive(name)
	} else {
		participant, err = b.clock.Register(name)
	}
	handle := ""
	if err == nil {
		handle = fmt.Sprintf("h%03d", len(b.order)+1)
		b.handles[handle] = participant
		b.order = append(b.order, handle)
	}
	b.record(clockConformanceStep{Op: op, Name: name, Handle: handle}, err, 0, nil)
	return handle
}

func (b *clockConformanceBuilder) activate(handle string) {
	b.t.Helper()
	value := b.clock.Activate(b.handles[handle])
	b.record(clockConformanceStep{Op: "activate", Handle: handle}, nil, value, nil)
}

func (b *clockConformanceBuilder) activateAt(handle string, current int64) {
	b.t.Helper()
	value, err := b.clock.ActivateAt(b.handles[handle], current)
	b.record(clockConformanceStep{Op: "activate_at", Handle: handle, Current: current}, err, value, nil)
}

func (b *clockConformanceBuilder) quiesce(handle string, generation uint64, current, deadline int64, arrivals uint32) clockConformanceOutput {
	b.t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	call := &clockConformanceCall{handle: handle, done: make(chan clockConformanceResponse, 1), cancel: cancel}
	participant := b.handles[handle]
	// A second Quiesce while this handle is parked answers at once with an
	// error, so the driver must wait for it rather than treat it as parked.
	wait := map[*clockConformanceCall]bool{call: b.waiting(handle) || participant == nil}
	go func() {
		response, err := b.clock.Quiesce(ctx, participant, clock.BrokerTimeRequest{Generation: generation, Current: current, Deadline: deadline, Arrivals: arrivals})
		call.done <- clockConformanceResponse{Handle: handle, Generation: response.Generation, Kind: response.Kind, Time: response.Time, Error: errorText(err)}
	}()
	b.pending = append(b.pending, call)
	b.issued = call
	return b.record(clockConformanceStep{Op: "quiesce", Handle: handle, Generation: generation, Current: current, Deadline: deadline, Arrivals: arrivals}, nil, 0, wait)
}

func (b *clockConformanceBuilder) cancel(handle string) {
	b.t.Helper()
	wait := map[*clockConformanceCall]bool{}
	for _, call := range b.pending {
		if call.handle == handle {
			call.cancel()
			wait[call] = true
		}
	}
	if len(wait) == 0 {
		b.t.Fatalf("clock conformance %s: cancel of %s without a parked quiesce", b.corpus.Name, handle)
	}
	b.record(clockConformanceStep{Op: "cancel", Handle: handle}, nil, 0, wait)
}

// event applies one progress event by its corpus op name.
func (b *clockConformanceBuilder) event(op, handle, other string, request uint64, arrivals uint32, current int64) clockConformanceOutput {
	b.t.Helper()
	p, q := b.handles[handle], b.handles[other]
	var err error
	switch op {
	case "accepted":
		err = b.clock.Accepted(p, request, arrivals)
	case "delivered":
		err = b.clock.Delivered(p, request)
	case "consume":
		err = b.clock.Consume(p, arrivals)
	case "remove":
		err = b.clock.Remove(p)
	case "control_forwarded":
		err = b.clock.ControlForwarded(p, q, request, arrivals)
	case "wait_accepted":
		err = b.clock.WaitAccepted(p, request, arrivals)
	case "wait_suspended":
		err = b.clock.WaitSuspended(p, request)
	case "wait_resumed":
		err = b.clock.WaitResumed(p, request)
	case "completion_reserved":
		err = b.clock.CompletionReserved(p, request)
	case "request_accepted":
		err = b.clock.RequestAccepted(p, arrivals)
	case "response_delivered":
		err = b.clock.ResponseDelivered(p)
	case "forwarded":
		err = b.clock.Forwarded(p, q, arrivals, current)
	case "dispatched":
		err = b.clock.Dispatched(p)
	case "dispatch_unavailable":
		err = b.clock.DispatchUnavailable(p)
	case "response_arrived":
		err = b.clock.ResponseArrived(p, q, arrivals)
	case "abandoned_discarded":
		err = b.clock.AbandonedResponseDiscarded(p, arrivals)
	case "runnable":
		err = b.clock.Runnable(p)
	default:
		b.t.Fatalf("unknown clock event %q", op)
	}
	return b.record(clockConformanceStep{Op: op, Handle: handle, Other: other, Request: request, Arrivals: arrivals, Current: current}, err, 0, nil)
}

// finish removes every live participant so parked goroutines end.
func (b *clockConformanceBuilder) finish() clockConformanceCorpus {
	b.t.Helper()
	for _, call := range b.pending {
		call.cancel()
		<-call.done
	}
	b.pending = nil
	return b.corpus
}

var clockConformanceEventOps = []string{"accepted", "delivered", "consume", "remove", "control_forwarded", "wait_accepted", "wait_suspended", "wait_resumed", "completion_reserved", "request_accepted", "response_delivered", "forwarded", "dispatched", "dispatch_unavailable", "response_arrived", "abandoned_discarded", "runnable"}

func clockConformanceScripted(t *testing.T) []clockConformanceCorpus {
	var corpora []clockConformanceCorpus
	add := func(name, origin string, kind clockConformanceClock, run func(*clockConformanceBuilder, int64)) {
		b := newClockConformance(t, name, "scripted", origin, kind)
		run(b, b.corpus.Initial)
		corpora = append(corpora, b.finish())
	}
	strict := clockConformanceClock{Kind: "process"}
	forward := clockConformanceClock{Kind: "process", Forward: true}
	broker := clockConformanceClock{Kind: "broker"}
	add("quoted-participant-names", "src/clock.rs clock_quotes_names_like_go", broker, func(b *clockConformanceBuilder, _ int64) {
		for _, name := range []string{"node/1", "a\"b\\c", "line\n\t\r\a\b\f\v\x00\x1f\u0085ü"} {
			b.register(name, true)
			b.register(name, true)
		}
	})
	// pairs registers coordinator and node/1 inactive, then activates both.
	pair := func(b *clockConformanceBuilder) (string, string) {
		c, n := b.register("coordinator", true), b.register("node/1", true)
		b.activate(c)
		b.activate(n)
		return c, n
	}
	add("earliest-deadline", "simulation_time_test.go TestSimulationTimeArbiterAdvancesEveryParticipantToEarliestDeadline", strict, func(b *clockConformanceBuilder, at int64) {
		c, n := pair(b)
		b.quiesce(c, 1, at, at+20, 0)
		b.quiesce(n, 1, at, at+10, 0)
	})
	add("forward-tick-epoch", "simulation_time_test.go TestSimulationTimeArbiterAcceptsForwardTickEpoch", forward, func(b *clockConformanceBuilder, at int64) {
		c := b.register("coordinator", true)
		b.activate(c)
		b.quiesce(c, 1, at+100, at+110, 0)
	})
	add("forward-activation-adopts-current", "simulation_time_test.go TestSimulationTimeArbiterForwardActivationAdoptsCurrent", forward, func(b *clockConformanceBuilder, at int64) {
		n := b.register("node/1", true)
		b.activateAt(n, at+100)
	})
	add("strict-activation-rejects-future", "simulation_time_test.go TestSimulationTimeArbiterStrictActivationRejectsFutureCurrent", strict, func(b *clockConformanceBuilder, at int64) {
		n := b.register("node/1", true)
		b.activateAt(n, at+100)
		b.activateAt(n, at-1)
		b.activateAt("", at)
	})
	for _, reverse := range []bool{false, true} {
		add(fmt.Sprintf("forward-tick-order-independent-%t", reverse), "simulation_time_test.go TestSimulationTimeArbiterForwardTickEpochIsOrderIndependent", forward, func(b *clockConformanceBuilder, at int64) {
			lower, higher := b.register("node/lower", true), b.register("node/higher", true)
			b.activate(lower)
			b.activate(higher)
			first, second := lower, higher
			firstAt, firstDeadline, secondAt, secondDeadline := at+11, at+50, at+17, at+40
			if reverse {
				first, second = higher, lower
				firstAt, firstDeadline, secondAt, secondDeadline = at+17, at+40, at+11, at+50
			}
			b.quiesce(first, 1, firstAt, firstDeadline, 0)
			b.quiesce(second, 1, secondAt, secondDeadline, 0)
		})
	}
	add("forward-tick-past-deadline", "simulation_time_test.go TestSimulationTimeArbiterRejectsForwardTickEpochPastItsDeadline", forward, func(b *clockConformanceBuilder, at int64) {
		c := b.register("coordinator", true)
		b.activate(c)
		b.quiesce(c, 1, at+100, at+99, 0)
		b.quiesce(c, 2, at+100, at+101, 0)
	})
	add("external-work-cancels-epoch", "simulation_time_test.go TestSimulationTimeArbiterCancelsAnEpochWhenExternalWorkArrives", strict, func(b *clockConformanceBuilder, at int64) {
		_, n := pair(b)
		b.quiesce(n, 1, at, at+10, 0)
		b.event("runnable", n, "", 0, 0, 0)
	})
	add("external-and-inactive-block-advance", "simulation_time_test.go TestSimulationTimeArbiterDoesNotAdvancePastExternalWorkOrInactiveParticipants", strict, func(b *clockConformanceBuilder, at int64) {
		c, n := b.register("coordinator", true), b.register("node/1", true)
		b.activate(c)
		b.event("accepted", c, "", 1, 0, 0)
		b.quiesce(c, 1, at, at+20, 0)
		b.event("delivered", c, "", 1, 0, 0)
		b.event("consume", c, "", 0, 1, 0)
		b.quiesce(c, 2, at, at+20, 0)
		b.activate(n)
		b.quiesce(n, 1, at, at+10, 0)
	})
	add("externally-blocked-excluded", "simulation_time_test.go TestSimulationTimeArbiterExcludesAnExternallyBlockedParticipant", strict, func(b *clockConformanceBuilder, at int64) {
		c, n := pair(b)
		b.event("accepted", c, "", 1, 0, 0)
		b.quiesce(n, 1, at, at+10, 0)
		b.event("delivered", c, "", 1, 0, 0)
		b.quiesce(c, 1, at, at+20, 1)
	})
	add("last-runnable-blocks-externally", "simulation_time_test.go TestSimulationTimeArbiterSettlesWhenLastRunnableParticipantBlocksExternally", strict, func(b *clockConformanceBuilder, at int64) {
		c, n := pair(b)
		b.quiesce(c, 1, at, at+10, 0)
		b.event("accepted", n, "", 1, 0, 0)
	})
	add("handled-request-blocks-settle", "simulation_time_test.go TestSimulationTimeArbiterDoesNotSettleWhileExternalRequestIsHandled", strict, func(b *clockConformanceBuilder, at int64) {
		c, n := pair(b)
		b.event("request_accepted", n, "", 0, 0, 0)
		b.quiesce(c, 1, at, at+10, 0)
		b.cancel(c)
	})
	add("forwarded-request-delivered", "simulation_time_test.go TestSimulationTimeArbiterAdvancesAfterForwardedRequestIsDelivered", strict, func(b *clockConformanceBuilder, at int64) {
		c, n := pair(b)
		b.event("forwarded", n, c, 0, 0, 0)
		b.event("response_delivered", c, "", 0, 0, 0)
		b.quiesce(c, 1, at, at+10, 1)
	})
	add("forward-external-atomic", "simulation_time_test.go TestSimulationTimeArbiterForwardsExternalRequestAtomically", strict, func(b *clockConformanceBuilder, at int64) {
		c, n := pair(b)
		b.quiesce(c, 1, at, at+10, 0)
		b.event("forwarded", n, c, 0, 0, 0)
		b.quiesce(n, 1, at, at+10, 0)
		b.event("response_delivered", c, "", 0, 0, 0)
		b.event("dispatched", c, "", 0, 0, 0)
		b.quiesce(c, 2, at, at+10, 2)
		b.event("response_arrived", c, n, 0, 0, 0)
		b.quiesce(n, 2, at, at+15, 1)
	})
	add("restart-hold-and-activation", "simulation_time_test.go TestSimulationTimeArbiterActivatesRestartAtCurrentTime", strict, func(b *clockConformanceBuilder, at int64) {
		c, n := pair(b)
		b.quiesce(c, 1, at, at+10, 0)
		b.quiesce(n, 1, at, at+10, 0)
		b.event("remove", n, "", 0, 0, 0)
		restarted := b.register("node/2", true)
		b.quiesce(c, 2, at+10, at+30, 0)
		b.activate(restarted)
		b.quiesce(restarted, 1, at+10, at+20, 0)
	})
	add("restart-runnable-hold-removal", "src/clock.rs clock_restart_hold_blocks_advance_until_removed", broker, func(b *clockConformanceBuilder, at int64) {
		a, old := b.register("a/1", false), b.register("b/1", false)
		hold := b.register("restart/b/1", false)
		b.event("remove", old, "", 0, 0, 0)
		parked := b.quiesce(a, 1, at, at+5, 0)
		if len(parked.Responses) != 0 || parked.Current != at {
			t.Fatal("restart hold did not preserve the parked deadline")
		}
		restarted := b.register("b/2", false)
		removed := b.event("remove", hold, "", 0, 0, 0)
		if len(removed.Responses) != 0 || removed.Current != at {
			t.Fatal("restart became runnable before quiescing")
		}
		advanced := b.quiesce(restarted, 1, at, at+5, 0)
		if len(advanced.Responses) != 2 || advanced.Current != at+5 {
			t.Fatal("restart failed to release both participants at the deadline")
		}
		for _, response := range advanced.Responses {
			if response.Generation != 1 || response.Kind != 1 || response.Time != at+5 || response.Error != "" {
				t.Fatalf("restart response differs: %+v", response)
			}
		}
		inactive := b.event("runnable", old, "", 0, 0, 0)
		if inactive.Error != "simulation time participant is inactive" {
			t.Fatalf("old incarnation remained active: %+v", inactive)
		}
	})
	add("strict-rejects-forward-tick", "simulation_time_test.go TestSimulationTimeArbiterRejectsForwardTickEpochInStrictMode", strict, func(b *clockConformanceBuilder, at int64) {
		c := b.register("coordinator", true)
		b.activate(c)
		b.quiesce(c, 1, at+100, at+110, 0)
		b.quiesce(c, 1, at, at+5, 0)
	})
	add("forward-early-responses-do-not-rewind", "simulation_time_test.go TestSimulationTimeArbiterForwardTickEarlyResponsesDoNotRewind", forward, func(b *clockConformanceBuilder, at int64) {
		c, n := b.register("coordinator", true), b.register("node/1", true)
		b.activate(c)
		b.activate(n)
		b.quiesce(c, 1, at+50, at+60, 0)
		b.quiesce(n, 1, at+10, at+20, 0)
		b.quiesce(n, 2, at+5, at+70, 0)
	})
	add("forward-malformed-request", "simulation_time_test.go TestSimulationTimeArbiterMalformedForwardTickRequestDoesNotAdvance", forward, func(b *clockConformanceBuilder, at int64) {
		c := b.register("coordinator", true)
		b.activate(c)
		b.quiesce(c, 2, at+100, at+110, 0)
		b.quiesce(c, 1, at+100, at+110, 3)
	})
	add("forwarded-model-current", "simulation_time_test.go TestSimulationTimeArbiterAdoptsForwardedModelCurrent", forward, func(b *clockConformanceBuilder, at int64) {
		c, n := b.register("coordinator", true), b.register("node/1", true)
		b.activate(c)
		b.activate(n)
		b.event("forwarded", n, c, 0, 0, at+25)
		b.event("forwarded", n, c, 0, 0, at-1)
		b.event("forwarded", n, n, 0, 0, at+30)
		b.event("forwarded", n, "", 0, 0, at+30)
	})
	add("strict-forwarded-model-current", "simulation_time_test.go TestSimulationTimeArbiterRejectsInvalidForwardedModelCurrentWithoutMutation", strict, func(b *clockConformanceBuilder, at int64) {
		c, n := pair(b)
		b.event("forwarded", n, c, 0, 0, at+25)
		b.event("forwarded", n, c, 0, 0, 1)
		b.event("forwarded", n, c, 0, 1, at)
	})
	add("broker-inactive-participant", "simulation_broker_test.go TestBrokerClockRejectsInactiveParticipantWithoutAdvancing", broker, func(b *clockConformanceBuilder, at int64) {
		a := b.register("a/1", false)
		b.event("remove", a, "", 0, 0, 0)
		b.quiesce(a, 1, at, at+10, 0)
		b.quiesce("", 1, at, at+10, 0)
		b.register("a/1", false)
		b.register("", false)
		b.register(strings.Repeat("n", 513), false)
	})
	add("wait-barrier-lifecycle", "simulation_progress.go coordinatorWait* transitions", strict, func(b *clockConformanceBuilder, at int64) {
		c, n := pair(b)
		b.event("wait_accepted", c, "", 7, 0, 0)
		b.event("wait_accepted", c, "", 7, 0, 0)
		b.quiesce(n, 1, at, at+10, 0)
		b.event("wait_suspended", c, "", 7, 0, 0)
		b.quiesce(c, 1, at, at+20, 0)
		b.event("wait_resumed", c, "", 7, 0, 0)
		b.event("wait_resumed", c, "", 7, 0, 0)
		b.event("delivered", c, "", 7, 0, 0)
		b.event("delivered", c, "", 7, 0, 0)
		b.event("wait_suspended", c, "", 8, 0, 0)
		b.event("completion_reserved", c, "", 9, 0, 0)
		b.event("completion_reserved", c, "", 9, 0, 0)
		b.event("delivered", c, "", 9, 0, 0)
		b.event("control_forwarded", c, n, 10, 0, 0)
		b.event("control_forwarded", c, n, 10, 0, 0)
		b.event("delivered", c, "", 10, 0, 0)
		b.event("dispatch_unavailable", n, "", 0, 0, 0)
		b.event("abandoned_discarded", c, "", 0, 9, 0)
		b.event("abandoned_discarded", c, "", 0, 1, 0)
		b.quiesce(c, 2, at+10, at+20, 2)
	})
	return corpora
}

// clockConformanceBrokerShaped replays the Go broker's clock calls for a
// two-node cluster: every non-idle callback is Accepted, its reply Delivered
// with the arrival credit consumed by the next request, idles quiesce, and a
// crash registers restart/<node>/<inc> before removing the incarnation.
func clockConformanceBrokerShaped(t *testing.T) []clockConformanceCorpus {
	var corpora []clockConformanceCorpus
	for seed := uint64(1); seed <= 8; seed++ {
		random := rand.New(rand.NewPCG(seed, 0x178_0005))
		b := newClockConformance(t, fmt.Sprintf("recorded-broker-shaped-%02d", seed), "recorded", fmt.Sprintf("wasi/broker.go clock call sequence, PCG(%d, 0x1780005)", seed), clockConformanceClock{Kind: "broker"})
		type session struct {
			name, handle, hold string
			incarnation        int
			request, gen       uint64
			arrivals           uint32
		}
		sessions := []*session{{name: "a", incarnation: 1}, {name: "b", incarnation: 1}}
		for _, s := range sessions {
			s.handle = b.register(fmt.Sprintf("%s/%d", s.name, s.incarnation), false)
		}
		for range 60 + random.IntN(40) {
			index := random.IntN(len(sessions))
			s := sessions[index]
			if b.waiting(s.handle) {
				// A parked incarnation acts only after the arbiter answers it.
				s = sessions[1-index]
				if b.waiting(s.handle) {
					break
				}
			}
			switch weight := random.IntN(100); {
			case s.hold != "":
				// Restart: a fresh incarnation joins, then the hold leaves.
				s.incarnation++
				s.handle = b.register(fmt.Sprintf("%s/%d", s.name, s.incarnation), false)
				b.event("remove", s.hold, "", 0, 0, 0)
				s.hold, s.request, s.gen, s.arrivals = "", 0, 0, 0
			case weight < 50:
				s.request++
				b.event("accepted", s.handle, "", s.request, s.arrivals, 0)
				s.arrivals = 0
				b.event("delivered", s.handle, "", s.request, 0, 0)
				s.arrivals = 1
			case weight < 85:
				s.gen++
				deadline := []int64{1, 10, 1000, math.MaxInt64 - b.clock.Current()}[random.IntN(4)]
				b.quiesce(s.handle, s.gen, b.clock.Current(), b.clock.Current()+deadline, s.arrivals)
				s.arrivals = 0
			case weight < 95:
				s.hold = b.register(fmt.Sprintf("restart/%s/%d", s.name, s.incarnation), false)
				b.event("remove", s.handle, "", 0, 0, 0)
			default:
				b.event("consume", s.handle, "", 0, s.arrivals, 0)
				s.arrivals = 0
			}
		}
		corpora = append(corpora, b.finish())
	}
	return corpora
}

func clockConformanceGenerated(t *testing.T) []clockConformanceCorpus {
	var corpora []clockConformanceCorpus
	for seed := uint64(1); seed <= 24; seed++ {
		random := rand.New(rand.NewPCG(seed, 0x178_0025))
		kind := []clockConformanceClock{{Kind: "broker"}, {Kind: "process"}, {Kind: "process", Forward: true}}[seed%3]
		b := newClockConformance(t, fmt.Sprintf("generated-%02d", seed), "generated", fmt.Sprintf("PCG(%d, 0x1780025)", seed), kind)
		generations := map[string]uint64{}
		var handles []string
		names := 0
		pick := func() string {
			if len(handles) == 0 || random.IntN(25) == 0 {
				return ""
			}
			return handles[random.IntN(len(handles))]
		}
		for range 2 + random.IntN(3) {
			names++
			handle := b.register(fmt.Sprintf("node/%d", names), kind.Kind == "process" && random.IntN(2) == 0)
			handles = append(handles, handle)
		}
		for range 140 + random.IntN(100) {
			switch weight := random.IntN(100); {
			case weight < 6:
				names++
				name := fmt.Sprintf("node/%d", names)
				if random.IntN(8) == 0 {
					name = "node/1"
				}
				if handle := b.register(name, kind.Kind == "process" && random.IntN(2) == 0); handle != "" {
					handles = append(handles, handle)
				}
			case weight < 12:
				handle := pick()
				if random.IntN(2) == 0 {
					b.activate(handle)
				} else {
					b.activateAt(handle, b.clock.Current()+int64(random.IntN(3))*5-1)
				}
			case weight < 45:
				handle := pick()
				generation := generations[handle] + 1
				if random.IntN(10) == 0 {
					generation += uint64(random.IntN(3)) + 1
				}
				current := b.clock.Current()
				if kind.Forward && random.IntN(3) == 0 {
					current += int64(random.IntN(20))
				}
				if random.IntN(15) == 0 {
					current += 5
				}
				deadline := current + []int64{0, 3, 10, 40, math.MaxInt64 - current}[random.IntN(5)]
				if random.IntN(20) == 0 {
					deadline = current - 1
				}
				out := b.quiesce(handle, generation, current, deadline, uint32(random.IntN(2)))
				for _, response := range out.Responses {
					if response.Handle == handle && response.Error == "" {
						generations[handle] = response.Generation
					}
				}
				if slices.Contains(out.Waiting, handle) {
					generations[handle] = generation
				}
			case weight < 48:
				var parked []string
				for _, call := range b.pending {
					parked = append(parked, call.handle)
				}
				if len(parked) != 0 {
					b.cancel(parked[random.IntN(len(parked))])
				}
			default:
				op := clockConformanceEventOps[random.IntN(len(clockConformanceEventOps))]
				if op == "remove" && random.IntN(3) != 0 {
					op = "runnable"
				}
				var current int64
				if op == "forwarded" && random.IntN(2) == 0 {
					current = b.clock.Current() + int64(random.IntN(4))*7
				}
				b.event(op, pick(), pick(), uint64(1+random.IntN(4)), uint32(random.IntN(3)), current)
			}
		}
		corpora = append(corpora, b.finish())
	}
	return corpora
}

func clockConformanceCorpora(t *testing.T) []clockConformanceCorpus {
	corpora := append(clockConformanceScripted(t), clockConformanceGenerated(t)...)
	return append(corpora, clockConformanceBrokerShaped(t)...)
}

func readClockConformance(t *testing.T, path string) clockConformanceCorpus {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var corpus clockConformanceCorpus
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&corpus); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return corpus
}

// TestClockConformanceCorpusIsCurrent fails on any shared-clock behavior change
// until the corpus is regenerated, which forces the Rust clock replay
// (`cargo test --lib clock_conformance`) to be rerun against it.
func TestClockConformanceCorpusIsCurrent(t *testing.T) {
	regenerate := os.Getenv("GOMAD_CLOCK_CONFORMANCE_REGENERATE") == "1"
	if regenerate {
		if err := os.MkdirAll(clockConformanceDirectory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	want := map[string]bool{}
	sources := map[string]int{}
	for _, corpus := range clockConformanceCorpora(t) {
		name := corpus.Name + ".json"
		if want[name] {
			t.Fatalf("duplicate clock conformance corpus %s", name)
		}
		want[name] = true
		sources[corpus.Source]++
		data, err := json.Marshal(corpus)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		path := filepath.Join(clockConformanceDirectory, name)
		if regenerate {
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, data) {
			t.Errorf("clock conformance corpus %s diverges from the Go clock at %s; review the Go change, then run GOMAD_CLOCK_CONFORMANCE_REGENERATE=1 go test -tags test_dep -run '^TestClockConformance' ./internal/backends/wasm/cluster and rerun the Rust replay", name, firstClockConformanceDivergence(got, corpus))
		}
	}
	if regenerate {
		if err := os.WriteFile(clockConformancePin, []byte(clock.BrokerImplementationSHA256()+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, source := range []string{"scripted", "generated", "recorded"} {
		if sources[source] == 0 {
			t.Errorf("clock conformance corpus lacks %s cases", source)
		}
	}
	entries, err := os.ReadDir(clockConformanceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".json" && !want[entry.Name()] {
			t.Errorf("unexpected clock conformance corpus %s", entry.Name())
		}
	}
}

func firstClockConformanceDivergence(checkedIn []byte, current clockConformanceCorpus) string {
	var old clockConformanceCorpus
	if err := json.Unmarshal(checkedIn, &old); err != nil {
		return "undecodable checked-in corpus: " + err.Error()
	}
	hash := ""
	if old.ImplementationSHA256 != current.ImplementationSHA256 {
		hash = fmt.Sprintf("; implementation hash %s -> %s", old.ImplementationSHA256, current.ImplementationSHA256)
	}
	for index := range max(len(old.Steps), len(current.Steps)) {
		if index >= len(old.Steps) || index >= len(current.Steps) {
			return fmt.Sprintf("step count %d -> %d (origin %s)%s", len(old.Steps), len(current.Steps), current.Origin, hash)
		}
		left, _ := json.Marshal(old.Steps[index])
		right, _ := json.Marshal(current.Steps[index])
		if !bytes.Equal(left, right) {
			return fmt.Sprintf("step %d %s (origin %s)%s", index, current.Steps[index].Op, current.Origin, hash)
		}
	}
	if hash != "" {
		return "steps identical (origin " + current.Origin + ")" + hash
	}
	return "header (origin " + current.Origin + ")"
}

func TestClockConformanceDriftPinIsCurrent(t *testing.T) {
	pin, err := os.ReadFile(clockConformancePin)
	if err != nil {
		t.Fatal(err)
	}
	if want := clock.BrokerImplementationSHA256() + "\n"; !bytes.Equal(pin, []byte(want)) {
		t.Fatalf("Go simulation clock implementation changed (%s, pinned %s): regenerate the clock conformance corpus and rerun the Rust clock replay before updating the pin", strings.TrimSpace(want), strings.TrimSpace(string(pin)))
	}
	entries, err := os.ReadDir(clockConformanceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if corpus := readClockConformance(t, filepath.Join(clockConformanceDirectory, entry.Name())); corpus.ImplementationSHA256+"\n" != string(pin) || corpus.Schema != clockConformanceSchema {
			t.Errorf("%s is pinned to %s, not %s", entry.Name(), corpus.ImplementationSHA256, strings.TrimSpace(string(pin)))
		}
	}
}

// TestClockConformanceGenerationIsDeterministic regenerates the corpus twice in
// one process; goroutine scheduling must never reach the recorded bytes.
func TestClockConformanceGenerationIsDeterministic(t *testing.T) {
	first, err := json.Marshal(clockConformanceGenerated(t))
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(clockConformanceGenerated(t))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("clock conformance generation depends on goroutine scheduling")
	}
}
