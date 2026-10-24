package runner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	annotationsv1 "umpire.local/model/gen/umpire/annotations/v1"
	modelv1 "umpire.local/model/gen/umpire/model/v1"
	runv1 "umpire.local/model/gen/umpire/run/v1"
)

func TestWaitsEndInDriverFaults(t *testing.T) {
	wasm := statefulGuest(t)
	silent := func(act func(*testClock)) func(int) *fakeDriver {
		return func(int) *fakeDriver { return &fakeDriver{script: good, whenWaiting: act} }
	}
	t.Run("read timeout", func(t *testing.T) {
		h := newHarness(t, wasm, silent(func(clock *testClock) { clock.expire(readDeadline) }))
		run := h.run(ModeExplore, configured(1))
		requireShape(t, run, "reset", "propose:", "fault")
		requireRefused(t, run, 2, "read timeout without Model acceptance")
		requireReplays(t, wasm, run)
	})
	t.Run("run time budget", func(t *testing.T) {
		h := newHarness(t, wasm, silent(func(clock *testClock) { clock.expire(runDeadline) }))
		run := h.run(ModeExplore, configured(1))
		requireRefused(t, run, 2, "run time budget")
	})
	t.Run("driver unavailable", func(t *testing.T) {
		r := New(wasm, func(context.Context) (Driver, error) { return nil, errors.New("no deployment") })
		run, err := r.Run(context.Background(), ModeExplore, configured(1))
		if err != nil {
			t.Fatal(err)
		}
		requireShape(t, run, "reset", "fault")
		requireRefused(t, run, 1, "driver: no deployment")
		requireReplays(t, wasm, run)
	})
	interrupted := func(t *testing.T, cancel context.CancelFunc, h *harness, ctx context.Context) {
		t.Helper()
		_, err := h.runner().Run(ctx, ModeExplore, configured(1))
		cancel()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("interrupted Run: %v", err)
		}
		exchanges := h.drivers[0].all()
		if len(exchanges) != 1 || !exchanges[0].joined() {
			t.Fatalf("the interrupted exchange was not closed and joined when Run returned (%d exchanges)", len(exchanges))
		}
	}
	t.Run("interrupted while starting", func(t *testing.T) {
		for range 50 {
			ctx, cancel := context.WithCancel(context.Background())
			h := newHarness(t, wasm, func(int) *fakeDriver {
				return &fakeDriver{script: good, onStart: func(context.Context, *testClock, int) { cancel() }}
			})
			interrupted(t, cancel, h, ctx)
		}
	})
	t.Run("interrupted while waiting", func(t *testing.T) {
		for range 50 {
			ctx, cancel := context.WithCancel(context.Background())
			interrupted(t, cancel, newHarness(t, wasm, silent(func(*testClock) { cancel() })), ctx)
		}
	})
}

func TestModelRefusalsRefuseTheRun(t *testing.T) {
	wasm := statefulGuest(t)
	t.Run("pre-admission callback", func(t *testing.T) {
		script := map[byte][]step{0: {{kind: EventCallback, value: 4, call: 1}}}
		h := newHarness(t, wasm, driverFor(script))
		run := h.run(ModeExplore, configured(1))
		requireShape(t, run, "reset", "propose:", "refusal:STATE")
		requireRefused(t, run, 2, "model refused METHOD_ANSWER: REFUSAL_CODE_STATE")
		if got := h.drivers[0].all()[0].decisionsReceived(); len(got) != 1 || got[0].decision.Refusal.GetCode() != modelv1.RefusalCode_REFUSAL_CODE_STATE {
			t.Fatalf("the Driver did not receive the refusal: %v", got)
		}
		requireReplays(t, wasm, run)
	})
	t.Run("annotate", func(t *testing.T) {
		script := map[byte][]step{0: {{kind: EventAnnotate, value: 1}}}
		h := newHarness(t, wasm, driverFor(script))
		run := h.run(ModeExplore, configured(1))
		if refusal := run.GetEvents()[2].GetRefusal(); refusal.GetMethod() != modelv1.Method_METHOD_ANNOTATE {
			t.Fatalf("annotate event: %v", run.GetEvents()[2])
		}
		requireRefused(t, run, 2, "METHOD_ANNOTATE")
		requireReplays(t, wasm, run)
	})
	t.Run("stock Kettle has no stateful operation", func(t *testing.T) {
		kettle := guest(t, "UMPIRE_KETTLE_WASM", "umpire_kettle_guest.wasm")
		h := newHarness(t, kettle, driverFor(good))
		run := h.run(ModeExplore, Options{Seed: 1})
		requireShape(t, run, "reset", "refusal:UNSUPPORTED")
		requireRefused(t, run, 1, "model refused METHOD_PROPOSE: REFUSAL_CODE_UNSUPPORTED")
		if len(h.drivers[0].all()) != 0 {
			t.Fatal("a refused proposal reached the Driver")
		}
		requireReplays(t, kettle, run)
	})
}

func TestReplayIncludesAnnotate(t *testing.T) {
	kettle := guest(t, "UMPIRE_KETTLE_WASM", "umpire_kettle_guest.wasm")
	r := New(kettle, nil)
	ctx := context.Background()
	s, err := r.newSession(ctx, ModeReplay, 1, Options{}.withDefaults(), nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	input, err := anypb.New(&annotationsv1.AnnotationRequest{Method: "/umpire.reference.kettle.v1.KettleService/Describe", Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.call(ctx, 0, 0, &modelv1.ResetRequest{}); err != nil {
		t.Fatal(err)
	}
	if event, err := s.call(ctx, 0, 0, &modelv1.AnnotateRequest{Input: input}); err != nil || event.GetAnnotation() == nil {
		t.Fatalf("annotate: %v %v", event, err)
	}
	s.request++
	if _, err := s.call(ctx, 0, 0, &modelv1.ProposeRequest{Seed: ProposalSeed(1, 1)}); err != nil {
		t.Fatal(err)
	}
	run := s.finish()
	if err := s.close(ctx, nil); err != nil {
		t.Fatal(err)
	}
	requireShape(t, run, "reset", "annotate", "refusal:UNSUPPORTED")
	requireReplays(t, kettle, run)
	tampered := proto.CloneOf(run)
	tampered.GetEvents()[1].GetAnnotation().GetResponse().GetOutput().Value = []byte{1}
	replayed, err := r.Run(ctx, ModeReplay, Options{Recorded: tampered})
	if err != nil {
		t.Fatal(err)
	}
	requireRefused(t, replayed, 1, "replay: event 1 differs")
}

func recordGood(t *testing.T, wasm []byte) *Run {
	t.Helper()
	return newHarness(t, wasm, driverFor(good)).run(ModeExplore, configured(1))
}

func TestReplayRejectsAlteredRuns(t *testing.T) {
	wasm := statefulGuest(t)
	recorded := recordGood(t, wasm)
	for _, tc := range []struct {
		name   string
		alter  func(*Run)
		event  uint64
		reason string
	}{
		{"other model", func(run *Run) { run.ModelHash = strings.Repeat("0", 64) }, 0, "another model"},
		{"other protocol", func(run *Run) { run.ProtocolVersion = 2 }, 0, "protocol version 2"},
		{"other world", func(run *Run) { run.World = control(1) }, 0, "recorded world"},
		{"other seed", func(run *Run) { run.Seed = 9 }, 1, "another seed"},
		{"missing event", func(run *Run) { run.Events = append(run.Events[:4:4], run.Events[5:]...) }, 4, "out of sequence"},
		{"duplicate event", func(run *Run) {
			run.Events = append(run.Events[:5:5], run.Events[4:]...)
		}, 5, "out of sequence"},
		{"reordered events", func(run *Run) {
			run.Events[4], run.Events[5] = run.Events[5], run.Events[4]
			run.Events[4].Sequence, run.Events[5].Sequence = 4, 5
		}, 5, "repeats or precedes Driver event 2"},
		{"crossed correlation", func(run *Run) { run.Events[5].Request = 1 }, 5, "crossed with request 1"},
		{"second reset", func(run *Run) { run.Events[3] = &runv1.Event{Sequence: 3, Request: 1, Kind: run.Events[0].Kind} }, 3, "one Reset"},
		{"tampered judgment", func(run *Run) { run.Events[4].GetJudgment().GetResponse().ContinuePolling = false }, 4, "event 4 differs"},
		{"tampered reply", func(run *Run) { run.Events[5].GetAnswer().GetResponse().Reply = control(41) }, 5, "event 5 differs"},
		{"tampered observation", func(run *Run) { run.Events[6].GetJudgment().GetRequest().Observation = control(3) }, 6, "event 6 differs"},
		{"tampered refusal", func(run *Run) { run.Events[9].GetRefusal().GetRefusal().Message = "other" }, 9, "event 9 differs"},
		{"redacted", func(run *Run) { run.Events[2].Redacted = true }, 2, "redacted"},
		{"other verdict", func(run *Run) {
			run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Unexplained{Unexplained: &runv1.Unexplained{Event: 6}}}
		}, 10, "recorded verdict differs"},
		{"truncated before a final observation, relabeled proposal budget", func(run *Run) {
			run.Events = run.Events[:5]
			run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{
				Termination: runv1.Termination_TERMINATION_PROPOSAL_BUDGET, Limit: 2,
			}}}
		}, 5, "termination does not fit"},
		{"truncated, relabeled with another event budget", func(run *Run) {
			run.Events = run.Events[:5]
			run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{
				Termination: runv1.Termination_TERMINATION_EVENT_BUDGET, Limit: 4,
			}}}
		}, 5, "termination does not fit"},
		{"truncated, relabeled question", func(run *Run) {
			run.Events = run.Events[:9]
			run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{Termination: runv1.Termination_TERMINATION_QUESTION}}}
		}, 9, "termination does not fit"},
		{"no events, labeled question", func(run *Run) {
			run.Events = nil
			run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{Termination: runv1.Termination_TERMINATION_QUESTION}}}
		}, 0, "begins with its Reset"},
		{"no events, labeled an empty event budget", func(run *Run) {
			run.Events = nil
			run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{Termination: runv1.Termination_TERMINATION_EVENT_BUDGET}}}
		}, 0, "begins with its Reset"},
		{"budget claimed for a quiescent Run", func(run *Run) {
			run.Events = run.Events[:9]
			run.Verdict = &runv1.Verdict{Outcome: &runv1.Verdict_Explained{Explained: &runv1.Explained{Termination: runv1.Termination_TERMINATION_QUIESCENT}}}
		}, 9, "recorded verdict differs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			altered := proto.CloneOf(recorded)
			tc.alter(altered)
			replayed, err := New(wasm, nil).Run(context.Background(), ModeReplay, Options{Recorded: altered})
			if err != nil {
				t.Fatal(err)
			}
			requireRefused(t, replayed, tc.event, tc.reason)
			if !strings.HasPrefix(replayed.GetVerdict().GetRefused().GetReason(), "replay: ") {
				t.Fatalf("reason %q", replayed.GetVerdict().GetRefused().GetReason())
			}
		})
	}
}

func TestReplayRejectsCallsAfterTheEndingEvent(t *testing.T) {
	wasm := statefulGuest(t)
	unexplained := newHarness(t, wasm, driverFor(buggy)).run(ModeExplore, configured(1))
	if unexplained.GetVerdict().GetUnexplained().GetEvent() != 5 {
		t.Fatalf("buggy: %v", unexplained.GetVerdict())
	}
	// A fault inserted before the Unexplained judgment would relabel the Run as Refused.
	altered := proto.CloneOf(unexplained)
	fault := &runv1.Event{Request: 2, Kind: &runv1.Event_DriverFault{DriverFault: &runv1.DriverFault{Reason: "inserted"}}}
	altered.Events = append(altered.Events[:5:5], append([]*runv1.Event{fault}, altered.Events[5:]...)...)
	for i, event := range altered.Events {
		event.Sequence = uint64(i)
	}
	altered.Verdict = refused(5, "driver fault: inserted")
	replayed, err := New(wasm, nil).Run(context.Background(), ModeReplay, Options{Recorded: altered})
	if err != nil {
		t.Fatal(err)
	}
	requireRefused(t, replayed, 6, "event 6 follows the event that ended the Run")
}

func TestRedactedRunsAreRefusedForReplay(t *testing.T) {
	wasm := statefulGuest(t)
	opts := configured(1)
	opts.Redact = func(event *runv1.Event) bool { return event.GetAnswer() != nil }
	run := newHarness(t, wasm, driverFor(good)).run(ModeExplore, opts)
	requireExplained(t, run, runv1.Termination_TERMINATION_QUIESCENT)
	if !run.GetEvents()[5].GetRedacted() {
		t.Fatal("the Answer is not marked redacted")
	}
	replayed, err := New(wasm, nil).Run(context.Background(), ModeReplay, Options{Recorded: run})
	if err != nil {
		t.Fatal(err)
	}
	requireRefused(t, replayed, 5, "replay: event 5 is redacted")
}

func TestTheDriverRedactsWhenTheOptionsDoNot(t *testing.T) {
	wasm := statefulGuest(t)
	answers := func(int) *fakeDriver {
		return &fakeDriver{script: good, redact: func(event *runv1.Event) bool { return event.GetAnswer() != nil }}
	}
	run := newHarness(t, wasm, answers).run(ModeExplore, configured(1))
	requireExplained(t, run, runv1.Termination_TERMINATION_QUIESCENT)
	for _, event := range run.GetEvents() {
		if event.GetRedacted() != (event.GetAnswer() != nil) {
			t.Fatalf("event %d: redacted %t", event.GetSequence(), event.GetRedacted())
		}
	}
	opts := configured(1)
	opts.Redact = func(*runv1.Event) bool { return false }
	run = newHarness(t, wasm, answers).run(ModeExplore, opts)
	for _, event := range run.GetEvents() {
		if event.GetRedacted() {
			t.Fatalf("event %d: the Options' redaction takes precedence", event.GetSequence())
		}
	}
}

func TestRedactionReachesRefusedCallsAndTheWorld(t *testing.T) {
	wasm := statefulGuest(t)
	script := map[byte][]step{0: {{kind: EventCallback, value: 4, call: 1}}}
	strip := func(event *runv1.Event) bool {
		if callback := event.GetAnswer().GetRequest().GetCallback(); len(callback.GetValue()) > 0 {
			callback.Value = nil
			return true
		}
		if world := event.GetReset_().GetRequest().GetConfiguration(); len(world.GetValue()) > 0 {
			world.Value = nil
			return true
		}
		return false
	}
	opts := configured(1)
	run := newHarness(t, wasm, func(int) *fakeDriver { return &fakeDriver{script: script, redact: strip} }).run(ModeExplore, opts)
	requireRefused(t, run, 2, "model refused METHOD_ANSWER")
	refused := run.GetEvents()[2]
	request := new(modelv1.AnswerRequest)
	if err := proto.Unmarshal(refused.GetRefusal().GetRequest(), request); err != nil {
		t.Fatal(err)
	}
	if !refused.GetRedacted() || len(request.GetCallback().GetValue()) != 0 || request.GetCallback().GetTypeUrl() != controlType {
		t.Fatalf("the refused Answer keeps its evidence: %v", refused)
	}
	if len(run.GetWorld().GetValue()) != 0 || !run.GetEvents()[0].GetRedacted() {
		t.Fatalf("the world keeps its evidence: %v", run.GetWorld())
	}
	if len(opts.World.GetValue()) == 0 {
		t.Fatal("redaction reached the caller's world")
	}
}

func TestQuestionAsksOneProposeOrJudge(t *testing.T) {
	wasm := statefulGuest(t)
	r := New(wasm, nil)
	ctx := context.Background()
	ask := func(opts Options) *Run {
		t.Helper()
		run, err := r.Run(ctx, ModeQuestion, opts)
		if err != nil {
			t.Fatal(err)
		}
		requireReplays(t, wasm, run)
		return run
	}
	run := ask(Options{World: control(0), Seed: 4})
	requireShape(t, run, "reset", "propose:")
	requireExplained(t, run, runv1.Termination_TERMINATION_QUESTION)
	if run.GetEvents()[1].GetProposal().GetResponse().GetKind() != modelv1.ProposalKind_PROPOSAL_KIND_BOOTSTRAP {
		t.Fatal("a fresh configured Model does not bootstrap")
	}

	recorded := recordGood(t, wasm)
	requireShape(t, ask(Options{Recorded: recorded, Prefix: 3}), "reset", "propose:", "judge:0", "propose:Deliver")
	requireShape(t, ask(Options{Recorded: recorded, Prefix: 4, Observation: control(20)}), "reset", "propose:", "judge:0", "propose:Deliver", "judge:20:poll")
	unexplained := ask(Options{Recorded: recorded, Prefix: 4, Observation: control(2)})
	if unexplained.GetVerdict().GetUnexplained().GetEvent() != 4 {
		t.Fatalf("question verdict: %v", unexplained.GetVerdict())
	}
	pending := ask(Options{Recorded: recorded, Prefix: 4})
	requireRefused(t, pending, 4, "METHOD_PROPOSE: REFUSAL_CODE_STATE")
	quiescent := ask(Options{Recorded: recorded, Prefix: 9})
	requireExplained(t, quiescent, runv1.Termination_TERMINATION_QUIESCENT)

	for _, opts := range []Options{{Recorded: recorded}, {Recorded: recorded, Prefix: 11}} {
		if _, err := r.Run(ctx, ModeQuestion, opts); err == nil {
			t.Fatalf("question accepted prefix %d", opts.Prefix)
		}
	}
	altered := proto.CloneOf(recorded)
	altered.GetEvents()[2].GetJudgment().GetResponse().GetExplained().Steps = []string{"Deliver"}
	run, err := r.Run(ctx, ModeQuestion, Options{Recorded: altered, Prefix: 3})
	if err != nil {
		t.Fatal(err)
	}
	requireRefused(t, run, 2, "replay: event 2 differs")
}

// flaky completes Deliver without the worker on the Runs the predicate selects.
func flaky(bad func(run int) bool) func(int) *fakeDriver {
	return func(run int) *fakeDriver {
		if bad(run) {
			return &fakeDriver{script: buggy}
		}
		return &fakeDriver{script: good}
	}
}

func TestReduceReportsTheShortestReproductionFound(t *testing.T) {
	wasm := statefulGuest(t)
	original := newHarness(t, wasm, driverFor(buggy)).run(ModeExplore, configured(1))
	if original.GetVerdict().GetUnexplained() == nil {
		t.Fatalf("original: %v", original.GetVerdict())
	}
	h := newHarness(t, wasm, driverFor(buggy))
	var recorded []*Run
	var progress strings.Builder
	reduced := h.run(ModeReduce, Options{Recorded: original, Progress: &progress, Record: func(_ context.Context, run *Run) error {
		recorded = append(recorded, run)
		return nil
	}})
	if reduced.GetVerdict().GetUnexplained().GetEvent() != 5 || len(recorded) != 1 || recorded[0] != reduced || len(h.drivers) != 2 || h.hosts.Load() != 3 {
		t.Fatalf("reduced %v; recorded %d; candidates %d", reduced.GetVerdict(), len(recorded), len(h.drivers))
	}
	if !strings.Contains(progress.String(), "mode=reduce iteration=1 seed=1 events=3 verdict=explained termination=proposal_budget") {
		t.Fatalf("progress:\n%s", progress.String())
	}
	requireReplays(t, wasm, reduced)

	gone := newHarness(t, wasm, flaky(func(int) bool { return false }))
	recorded = nil
	notReproduced := gone.run(ModeReduce, Options{Recorded: original, Record: func(_ context.Context, run *Run) error {
		recorded = append(recorded, run)
		return nil
	}})
	requireRefused(t, notReproduced, uint64(len(notReproduced.GetEvents())), "reduce: not reproduced within 2 candidates")
	if len(gone.drivers) != 2 || len(recorded) != 0 {
		t.Fatalf("not reproduced after %d candidates published %d Runs", len(gone.drivers), len(recorded))
	}
	for name, alter := range map[string]func(*Run){
		"protocol": func(run *Run) { run.ProtocolVersion = 2 },
		"seed":     func(run *Run) { run.Events[3].GetProposal().GetRequest().Seed++ },
	} {
		altered := proto.CloneOf(original)
		alter(altered)
		if _, err := gone.runner().Run(context.Background(), ModeReduce, Options{Recorded: altered}); err == nil || !strings.Contains(err.Error(), "does not replay") {
			t.Fatalf("reduced a Run with another %s: %v", name, err)
		}
	}
	if _, err := gone.runner().Run(context.Background(), ModeReduce, Options{Recorded: recordGood(t, wasm)}); err == nil {
		t.Fatal("reduced an Explained Run")
	}
}

func TestRepeatRecordsIndependentRunsOfTheSameSeeds(t *testing.T) {
	wasm := statefulGuest(t)
	h := newHarness(t, wasm, flaky(func(run int) bool { return run == 1 }))
	store := t.TempDir()
	var publications []Publication
	opts := configured(1)
	opts.Count = 3
	opts.Record = func(ctx context.Context, run *Run) error {
		publication, err := Publish(ctx, store, run)
		publications = append(publications, publication)
		return err
	}
	decisive := h.run(ModeRepeat, opts)
	if decisive.GetVerdict().GetUnexplained() == nil || len(publications) != 3 {
		t.Fatalf("decisive %v of %d", decisive.GetVerdict(), len(publications))
	}
	first, second, third := publications[0], publications[1], publications[2]
	if first.Existing || second.Existing || !third.Existing || first.Key != third.Key || first.Key == second.Key {
		t.Fatalf("publications: %+v", publications)
	}
	if !strings.HasPrefix(second.Key, h.runner().ModelHash()+"/1/") || !strings.HasPrefix(first.Key, h.runner().ModelHash()+"/1/") {
		t.Fatalf("keys %s %s are not under model/seed", first.Key, second.Key)
	}
	for _, publication := range publications {
		loaded, err := Load(publication.Path)
		if err != nil {
			t.Fatal(err)
		}
		requireReplays(t, wasm, loaded)
	}
	if _, err := h.runner().Run(context.Background(), ModeRepeat, configured(1)); err == nil {
		t.Fatal("repeat accepted a zero count")
	}
}

func TestProgressNamesPositionsOnly(t *testing.T) {
	wasm := statefulGuest(t)
	const secret = "secret-credential"
	store := t.TempDir()
	var progress strings.Builder
	slow := func(run int) *fakeDriver {
		script := good
		if run == 1 {
			script = map[byte][]step{0: good[0], 1: {{kind: EventFault, fault: "dial " + secret}}}
		}
		return &fakeDriver{script: script, delay: 2 * time.Millisecond}
	}
	opts := configured(2)
	opts.Count = 2
	opts.Progress = &progress
	opts.Record = func(ctx context.Context, run *Run) error {
		_, err := Publish(ctx, store, run)
		return err
	}
	run := newHarness(t, wasm, slow).run(ModeRepeat, opts)
	requireRefused(t, run, 4, secret)
	text := progress.String()
	for _, want := range []string{
		"umpire: mode=repeat iteration=1 seed=1 event=1 request=1\n",
		"umpire: mode=repeat iteration=1 seed=1 event=3 request=2\n",
		"umpire: mode=repeat iteration=1 seed=1 events=10 verdict=explained termination=quiescent\n",
		"umpire: mode=repeat iteration=1 seed=2 events=5 verdict=refused at=4\n",
		"umpire: mode=repeat iteration=2 seed=2 events=10 verdict=explained termination=quiescent\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("progress lacks %q:\n%s", want, text)
		}
	}
	for _, leak := range []string{secret, store, controlType, "Deliver"} {
		if strings.Contains(text, leak) {
			t.Fatalf("progress leaks %q:\n%s", leak, text)
		}
	}
}
