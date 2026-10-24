package runner

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	runv1 "umpire.local/model/gen/umpire/run/v1"
)

// kinds names each event of a Run, for comparing shapes.
func kinds(run *Run) []string {
	var names []string
	for _, event := range run.GetEvents() {
		switch {
		case event.GetReset_() != nil:
			names = append(names, "reset")
		case event.GetProposal() != nil:
			names = append(names, "propose:"+event.GetProposal().GetResponse().GetAction())
		case event.GetJudgment() != nil:
			judgment := event.GetJudgment().GetResponse()
			name := fmt.Sprintf("judge:%d", event.GetJudgment().GetRequest().GetObservation().GetValue()[0])
			switch {
			case judgment.GetUnexplained() != nil:
				name += ":unexplained"
			case judgment.GetContinuePolling():
				name += ":poll"
			default:
			}
			names = append(names, name)
		case event.GetAnswer() != nil:
			names = append(names, fmt.Sprintf("answer:%d", event.GetAnswer().GetRequest().GetCallback().GetValue()[0]))
		case event.GetAnnotation() != nil:
			names = append(names, "annotate")
		case event.GetRefusal() != nil:
			names = append(names, "refusal:"+strings.TrimPrefix(event.GetRefusal().GetRefusal().GetCode().String(), "REFUSAL_CODE_"))
		default:
			names = append(names, "fault")
		}
	}
	return names
}

var goodShape = []string{"reset", "propose:", "judge:0", "propose:Deliver", "judge:20:poll", "answer:4", "judge:2", "propose:Finish", "judge:3", "refusal:QUIESCENT"}

func requireShape(t *testing.T, run *Run, want ...string) {
	t.Helper()
	if got := kinds(run); !slices.Equal(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func requireRefused(t *testing.T, run *Run, event uint64, reason string) {
	t.Helper()
	refusal := run.GetVerdict().GetRefused()
	if refusal == nil || refusal.GetEvent() != event || !strings.Contains(refusal.GetReason(), reason) {
		t.Fatalf("verdict = %v, want Refused at %d containing %q", run.GetVerdict(), event, reason)
	}
}

func requireExplained(t *testing.T, run *Run, termination runv1.Termination) {
	t.Helper()
	if run.GetVerdict().GetExplained().GetTermination() != termination || run.GetVerdict().GetExplained() == nil {
		t.Fatalf("verdict = %v, want Explained %v", run.GetVerdict(), termination)
	}
}

// requireReplays replays a Run without any Driver and requires the same verdict and events.
func requireReplays(t *testing.T, wasm []byte, run *Run) {
	t.Helper()
	replayed, err := New(wasm, nil).Run(context.Background(), ModeReplay, Options{Recorded: run})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(replayed, run) {
		t.Fatalf("replay differs: %v\nrecorded %v", replayed.GetVerdict(), run.GetVerdict())
	}
}

func TestExploreRunsTheLoopOverTheControlSystem(t *testing.T) {
	wasm := statefulGuest(t)
	h := newHarness(t, wasm, driverFor(good))
	var recorded []*Run
	opts := configured(2)
	opts.Record = func(_ context.Context, run *Run) error {
		recorded = append(recorded, run)
		return nil
	}
	run := h.run(ModeExplore, opts)
	if len(recorded) != 2 || run != recorded[1] {
		t.Fatalf("recorded %d Runs; decisive is not the last", len(recorded))
	}
	if h.hosts.Load() != 2 || len(h.drivers) != 2 {
		t.Fatalf("hosts=%d drivers=%d, want one fresh each per Run", h.hosts.Load(), len(h.drivers))
	}
	for i, run := range recorded {
		requireShape(t, run, goodShape...)
		requireExplained(t, run, runv1.Termination_TERMINATION_QUIESCENT)
		if run.GetSeed() != uint64(i+1) || run.GetModelHash() != h.runner().ModelHash() || run.GetProtocolVersion() != ProtocolVersion || !proto.Equal(run.GetWorld(), control(0)) {
			t.Fatalf("run %d identity: seed=%d hash=%s", i, run.GetSeed(), run.GetModelHash())
		}
		for j, event := range run.GetEvents() {
			if event.GetSequence() != uint64(j) {
				t.Fatalf("event %d has sequence %d", j, event.GetSequence())
			}
		}
		if answer := run.GetEvents()[5]; answer.GetAnswer().GetCall() != 7 || answer.GetRequest() != 2 || answer.GetDriverEvent() != 2 {
			t.Fatalf("answer correlation: %v", answer)
		}
		requireReplays(t, wasm, run)
	}
	exchanges := h.drivers[0].all()
	if len(exchanges) != 3 {
		t.Fatalf("exchanges = %d", len(exchanges))
	}
	deliver := exchanges[1].decisionsReceived()
	if len(deliver) != 3 || !deliver[0].decision.Judgment.GetContinuePolling() || !proto.Equal(deliver[1].decision.Reply, control(42)) ||
		deliver[2].decision.Judgment.GetExplained() == nil || deliver[2].decision.Judgment.GetContinuePolling() {
		t.Fatalf("Deliver decisions: %v", deliver)
	}
	for _, exchange := range exchanges {
		if !exchange.joined() {
			t.Fatal("an exchange was not closed and joined")
		}
	}
}

func TestUnexplainedEndsTheRunAtItsEvent(t *testing.T) {
	wasm := statefulGuest(t)
	script := map[byte][]step{
		0: buggy[0],
		// The system keeps reporting after the Unexplained; Close cancels its blocked sends.
		1: {{kind: EventObservation, value: 20}, {kind: EventObservation, value: 2}, {kind: EventObservation, value: 2, async: true}, {kind: EventObservation, value: 2, async: true}, {kind: EventDone}},
	}
	h := newHarness(t, wasm, driverFor(script))
	run := h.run(ModeExplore, configured(1))
	requireShape(t, run, "reset", "propose:", "judge:0", "propose:Deliver", "judge:20:poll", "judge:2:unexplained")
	if run.GetVerdict().GetUnexplained().GetEvent() != 5 {
		t.Fatalf("verdict = %v", run.GetVerdict())
	}
	for _, exchange := range h.drivers[0].all() {
		if !exchange.joined() {
			t.Fatal("the blocked exchange was not joined")
		}
	}
	requireReplays(t, wasm, run)
}

func TestQueuedEventsAreServedInReceiveOrder(t *testing.T) {
	wasm := statefulGuest(t)
	script := map[byte][]step{
		0: good[0],
		1: {{kind: EventObservation, value: 20, async: true}, {kind: EventCallback, value: 4, call: 9, async: true}, {kind: EventObservation, value: 2}, {kind: EventDone}},
		3: good[3],
	}
	h := newHarness(t, wasm, driverFor(script))
	run := h.run(ModeExplore, configured(1))
	requireShape(t, run, goodShape...)
	got := h.drivers[0].all()[1].decisionsReceived()
	if len(got) != 3 || got[0].id != 1 || got[0].decision.Judgment == nil || got[1].id != 2 || got[1].decision.Reply == nil || got[2].id != 3 {
		t.Fatalf("decisions = %v", got)
	}
	requireReplays(t, wasm, run)
}

func TestBudgetsEndExplainedRuns(t *testing.T) {
	wasm := statefulGuest(t)
	h := newHarness(t, wasm, driverFor(good))
	opts := configured(1)
	opts.Budget = 1
	run := h.run(ModeExplore, opts)
	requireShape(t, run, "reset", "propose:", "judge:0")
	requireExplained(t, run, runv1.Termination_TERMINATION_PROPOSAL_BUDGET)
	requireReplays(t, wasm, run)

	if run.GetVerdict().GetExplained().GetLimit() != 1 {
		t.Fatalf("proposal budget limit: %v", run.GetVerdict())
	}

	for budget, shape := range map[int][]string{
		1: {"reset"},
		2: {"reset", "propose:"},
		4: {"reset", "propose:", "judge:0", "propose:Deliver"},
		5: {"reset", "propose:", "judge:0", "propose:Deliver", "judge:20:poll"},
	} {
		opts = configured(1)
		opts.EventBudget = budget
		run = h.run(ModeExplore, opts)
		requireShape(t, run, shape...)
		requireExplained(t, run, runv1.Termination_TERMINATION_EVENT_BUDGET)
		if run.GetVerdict().GetExplained().GetLimit() != uint64(budget) {
			t.Fatalf("event budget %d limit: %v", budget, run.GetVerdict())
		}
		requireReplays(t, wasm, run)
	}
}

func TestStartMustReturnOnceSent(t *testing.T) {
	wasm := statefulGuest(t)
	start := func(t *testing.T, onStart func(ctx context.Context, clock *testClock, made int)) (*Run, *harness) {
		t.Helper()
		h := newHarness(t, wasm, func(int) *fakeDriver { return &fakeDriver{script: good, onStart: onStart} })
		run := h.run(ModeExplore, configured(1))
		requireRefused(t, run, 2, "start did not return within the read timeout")
		requireReplays(t, wasm, run)
		return run, h
	}
	t.Run("late within the grace", func(t *testing.T) {
		run, h := start(t, func(ctx context.Context, clock *testClock, made int) {
			clock.expire(readDeadline)
			<-ctx.Done()
			clock.await(readDeadline, made+1)
		})
		requireShape(t, run, "reset", "propose:", "fault")
		if exchanges := h.drivers[0].all(); len(exchanges) != 1 || !exchanges[0].joined() {
			t.Fatal("the late exchange was not closed and joined")
		}
	})
	t.Run("past the grace", func(t *testing.T) {
		release := make(chan struct{})
		run, _ := start(t, func(ctx context.Context, clock *testClock, made int) {
			clock.expire(readDeadline)
			<-ctx.Done()
			clock.await(readDeadline, made+1)
			clock.expire(readDeadline)
			<-release
		})
		close(release)
		requireShape(t, run, "reset", "propose:", "fault", "fault")
		if reason := run.GetEvents()[3].GetDriverFault().GetReason(); reason != "start did not return after its context ended" {
			t.Fatalf("second fault: %q", reason)
		}
	})
	t.Run("ignoring its context", func(t *testing.T) {
		release := make(chan struct{})
		run, _ := start(t, func(_ context.Context, clock *testClock, made int) {
			clock.expire(readDeadline)
			clock.await(readDeadline, made+1)
			clock.expire(readDeadline)
			<-release
		})
		close(release)
		requireShape(t, run, "reset", "propose:", "fault", "fault")
	})
}

func TestRefusedResetReplays(t *testing.T) {
	wasm := statefulGuest(t)
	run := newHarness(t, wasm, driverFor(good)).run(ModeExplore, Options{World: control(9), Seed: 1})
	requireShape(t, run, "refusal:UNSUPPORTED")
	requireRefused(t, run, 0, "METHOD_RESET")
	requireReplays(t, wasm, run)
	altered := proto.CloneOf(run)
	altered.World = control(8)
	replayed, err := New(wasm, nil).Run(context.Background(), ModeReplay, Options{Recorded: altered})
	if err != nil {
		t.Fatal(err)
	}
	requireRefused(t, replayed, 0, "recorded world")
}

func TestDriverFaultsRefuseTheRun(t *testing.T) {
	wasm := statefulGuest(t)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	deliver := func(steps ...step) map[byte][]step {
		return map[byte][]step{0: good[0], 1: steps, 3: good[3]}
	}
	for _, tc := range []struct {
		name   string
		driver *fakeDriver
		shape  []string
		reason string
	}{
		{"start", &fakeDriver{script: good, startErr: errors.New("unreachable endpoint")}, []string{"reset", "propose:", "fault"}, "start: unreachable endpoint"},
		{"fault event", &fakeDriver{script: deliver(step{kind: EventFault, fault: "callback timeout"})}, []string{"reset", "propose:", "judge:0", "propose:Deliver", "fault"}, "callback timeout"},
		{"premature close", &fakeDriver{script: deliver(step{kind: EventObservation, value: 20})}, []string{"reset", "propose:", "judge:0", "propose:Deliver", "judge:20:poll", "fault"}, "closed without completion"},
		{"done before acceptance", &fakeDriver{script: deliver(step{kind: EventObservation, value: 20}, step{kind: EventDone})}, []string{"reset", "propose:", "judge:0", "propose:Deliver", "judge:20:poll", "fault"}, "before the Model accepted"},
		{"crossed request", &fakeDriver{script: deliver(step{kind: EventObservation, value: 20, request: 1})}, []string{"reset", "propose:", "judge:0", "propose:Deliver", "fault"}, "correlates with request 1, not 2"},
		{"duplicate event", &fakeDriver{script: deliver(step{kind: EventObservation, value: 20}, step{kind: EventCallback, value: 4, id: 1})}, []string{"reset", "propose:", "judge:0", "propose:Deliver", "judge:20:poll", "fault"}, "repeats or precedes"},
		{"unknown kind", &fakeDriver{script: deliver(step{kind: EventKind(99)})}, []string{"reset", "propose:", "judge:0", "propose:Deliver", "fault"}, "unknown kind"},
		{"blocked respond", &fakeDriver{script: good, blockRespond: true}, []string{"reset", "propose:", "judge:0", "fault"}, "respond: context deadline exceeded"},
		{"canceled respond", &fakeDriver{script: good, respondErr: context.Canceled}, []string{"reset", "propose:", "judge:0", "fault"}, "respond: context canceled"},
		{"close", &fakeDriver{script: good, closeErr: errors.New("worker did not stop")}, []string{"reset", "propose:", "judge:0", "fault"}, "close: worker did not stop"},
		{"hung close", &fakeDriver{script: good, hangClose: release}, []string{"reset", "propose:", "judge:0", "fault"}, "close did not return"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, wasm, func(int) *fakeDriver { return tc.driver })
			run := h.run(ModeExplore, configured(1))
			requireShape(t, run, tc.shape...)
			requireRefused(t, run, uint64(len(tc.shape)-1), tc.reason)
			requireReplays(t, wasm, run)
		})
	}
}
