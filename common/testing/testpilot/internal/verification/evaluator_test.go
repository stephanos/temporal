package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func event(sequence, elapsed int64, kind testpilotpb.RunEventKind) *testpilotpb.RunEvent {
	return &testpilotpb.RunEvent{Sequence: sequence, ElapsedMilliseconds: elapsed, Kind: kind, SourceId: fmt.Sprint(sequence)}
}
func TestEvaluatorHorizonsAndReplay(t *testing.T) {
	for _, tc := range []struct {
		name       string
		witness    int64
		incomplete bool
		kind       testpilotpb.RunEventKind
		want       testpilotpb.VerdictStatus
		stop       int64
	}{
		{"before", 4999, false, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, testpilotpb.VERDICT_STATUS_SATISFIED, 0},
		{"at", 5000, false, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, testpilotpb.VERDICT_STATUS_VIOLATED, 2},
		{"late", 6000, false, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, testpilotpb.VERDICT_STATUS_VIOLATED, 2},
		{"timeout", 5000, false, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT, testpilotpb.VERDICT_STATUS_VIOLATED, 2},
		{"closure", 5000, false, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED, testpilotpb.VERDICT_STATUS_VIOLATED, 2},
		{"early", 4000, false, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, 0},
		{"incomplete", 6000, true, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, cat, view, limits := fixture(t)
			c.Rules[0].Kind = testpilotpb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS
			c.Rules[0].Horizon = &testpilotpb.ContractHorizonDefinition{ElapsedMilliseconds: 5000, ViolationStateId: "bad"}
			p, err := Prepare(c, cat, view, limits)
			require.NoError(t, err)
			run := &testpilotpb.Run{RunId: "run", CaseId: "case", ProgramId: "program", Status: testpilotpb.RUN_STATUS_COMPLETED, Events: []*testpilotpb.RunEvent{event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED), event(2, tc.witness, tc.kind)}}
			run.Events[1].ExecutionIncomplete = tc.incomplete
			if tc.incomplete {
				run.Status = testpilotpb.RUN_STATUS_INCOMPLETE
			}
			if tc.stop > 0 {
				run.Status = testpilotpb.RUN_STATUS_STOPPED_BY_MONITOR
			}
			if tc.kind != testpilotpb.RUN_EVENT_KIND_RUN_CLOSED {
				run.Events = append(run.Events, event(3, 7000, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED))
			}
			monitor, err := execution.NewMonitor(context.Background(), p, view)
			require.NoError(t, err)
			var firstStop int64
			for _, e := range run.Events {
				d, err := monitor.Observe(context.Background(), e)
				require.NoError(t, err)
				if d == execution.Stop && firstStop == 0 {
					firstStop = e.Sequence
				}
			}
			require.Equal(t, tc.stop, firstStop)
			live, err := monitor.Close(context.Background(), run)
			require.NoError(t, err)
			require.Equal(t, tc.want, live.Status)
			offline, verdict, err := p.evaluate(context.Background(), run)
			require.NoError(t, err)
			liveBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(live)
			require.NoError(t, err)
			offlineBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(verdict)
			require.NoError(t, err)
			require.Equal(t, liveBytes, offlineBytes)
			trace, err := json.Marshal(monitor.(*Evaluator).trace)
			require.NoError(t, err)
			replay, err := json.Marshal(offline.trace)
			require.NoError(t, err)
			require.Equal(t, trace, replay)
		})
	}
}

func observed(sequence, elapsed, id int64) *testpilotpb.RunEvent {
	e := event(sequence, elapsed, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED)
	e.Observations = []*testpilotpb.ObservationResult{{ObservationId: "id", Value: &testpilotpb.Value{Value: &testpilotpb.Value_SignedInteger{SignedInteger: fmt.Sprint(id)}}}}
	return e
}
func TestEvaluatorCaptureCorrelationAndStop(t *testing.T) {
	c, cat, view, limits := fixture(t, 16<<20)
	r := c.Rules[0]
	addCapture(r)
	r.States = append(r.States, &testpilotpb.ContractStateDefinition{StateId: "saved", Status: testpilotpb.CONTRACT_STATE_STATUS_NONTERMINAL})
	r.Transitions = []*testpilotpb.ContractTransitionDefinition{transition("save", "start", "saved", present(observation("id"))), transition("match", "saved", "bad", all(present(observation("id")), equal(observation("id"), capture("saved")))), transition("shadowed", "saved", "good", all(present(observation("id")), equal(observation("id"), capture("saved"))))}
	assign(r.Transitions[0])
	p, err := Prepare(c, cat, view, limits)
	require.NoError(t, err)
	c.Limits.MaxWorkPerEvent = p.workPerEvent
	p, err = Prepare(c, cat, view, limits)
	require.NoError(t, err)
	for _, id := range []int64{7, 19} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			t.Parallel()
			monitor, err := p.New(context.Background(), view)
			require.NoError(t, err)
			e := monitor.(*Evaluator)
			run := &testpilotpb.Run{RunId: "run", ProgramId: "program", Status: testpilotpb.RUN_STATUS_STOPPED_BY_MONITOR, Events: []*testpilotpb.RunEvent{event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED), observed(2, 1000, id), observed(3, 2000, id+1), observed(4, 3000, id), event(5, 4000, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED)}}
			for i, v := range run.Events {
				d, err := e.Observe(context.Background(), v)
				require.NoError(t, err)
				if i < 3 {
					require.Equal(t, execution.Continue, d)
				} else {
					require.Equal(t, execution.Stop, d)
				}
			}
			require.Equal(t, []transitionTrace{{2, "rule", "save", "start", "saved"}, {4, "rule", "match", "saved", "bad"}}, e.trace)
			require.Equal(t, int64(2), e.rules[0].captures["saved"].sequence)
			require.Equal(t, fmt.Sprint(id), e.rules[0].captures["saved"].value.GetSignedInteger())
			run.Events[1].Observations[0].Value.GetValue().(*testpilotpb.Value_SignedInteger).SignedInteger = "999"
			require.Equal(t, fmt.Sprint(id), e.rules[0].captures["saved"].value.GetSignedInteger())
			run.Events[1].Observations[0].Value.GetValue().(*testpilotpb.Value_SignedInteger).SignedInteger = fmt.Sprint(id)
			live, err := e.Close(context.Background(), run)
			require.NoError(t, err)
			require.Equal(t, []int64{2, 4}, live.SupportingEventSequences)
			offline, err := p.Evaluate(context.Background(), run)
			require.NoError(t, err)
			require.True(t, proto.Equal(live, offline))
		})
	}
}

func TestEvaluatorMessageCaptureDescriptorBoundsAndOwnership(t *testing.T) {
	c, catalog, view, ceiling := fixture(t, 64)
	c.Limits.MaxCaptureBytes = 72
	rule := c.Rules[0]
	rule.Captures = []*testpilotpb.ContractCaptureDefinition{{CaptureId: "saved-message", Type: &testpilotpb.ContractCaptureType{Type: &testpilotpb.ContractCaptureType_Message{Message: &testpilotpb.NamedType{ProtobufType: "example.Empty"}}}}}
	rule.Transitions[0] = transition("save", "start", "good", present(observation("message")))
	rule.Transitions[0].CaptureAssignments = []*testpilotpb.ContractCaptureAssignment{{CaptureId: "saved-message", Observation: &testpilotpb.ObservationRef{ObservationId: "message"}}}

	tooSmall := proto.CloneOf(c)
	tooSmall.Limits.MaxCaptureBytes--
	_, err := Prepare(tooSmall, catalog, view, ceiling)
	require.Error(t, err)
	wrongDescriptor := proto.CloneOf(c)
	wrongDescriptor.Rules[0].Captures[0].Type.GetMessage().ProtobufType = "example.Missing"
	_, err = Prepare(wrongDescriptor, catalog, view, ceiling)
	require.Error(t, err)

	prepared, err := Prepare(c, catalog, view, ceiling)
	require.NoError(t, err)
	monitor, err := prepared.New(t.Context(), view)
	require.NoError(t, err)
	evaluator := monitor.(*Evaluator)
	_, err = evaluator.Observe(t.Context(), event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	value := &testpilotpb.Value{Value: &testpilotpb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/example.Empty"}}}
	observed := event(2, 1, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED)
	observed.Observations = []*testpilotpb.ObservationResult{{ObservationId: "message", Value: value}}
	_, err = evaluator.Observe(t.Context(), observed)
	require.NoError(t, err)
	require.Equal(t, int64(proto.Size(value))+8, evaluator.captureBytes)
	value.GetMessageValue().TypeUrl = "type.googleapis.com/example.Missing"
	require.Equal(t, "type.googleapis.com/example.Empty", evaluator.rules[0].captures["saved-message"].value.GetMessageValue().GetTypeUrl())

	for _, test := range []struct {
		name   string
		mutate func(*Evaluator, *testpilotpb.Value)
	}{
		{name: "descriptor", mutate: func(_ *Evaluator, value *testpilotpb.Value) {
			value.GetMessageValue().TypeUrl = "type.googleapis.com/example.Missing"
		}},
		{name: "retained byte ceiling", mutate: func(evaluator *Evaluator, value *testpilotpb.Value) {
			evaluator.captureBytes = c.Limits.MaxCaptureBytes - int64(proto.Size(value)) - 7
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			monitor, err := prepared.New(t.Context(), view)
			require.NoError(t, err)
			evaluator := monitor.(*Evaluator)
			_, err = evaluator.Observe(t.Context(), event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED))
			require.NoError(t, err)
			candidate := &testpilotpb.Value{Value: &testpilotpb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/example.Empty"}}}
			test.mutate(evaluator, candidate)
			observed := event(2, 1, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED)
			observed.Observations = []*testpilotpb.ObservationResult{{ObservationId: "message", Value: candidate}}
			_, err = evaluator.Observe(t.Context(), observed)
			require.Error(t, err)
			require.Empty(t, evaluator.rules[0].captures)
			require.Empty(t, evaluator.trace)
		})
	}
}

func TestEvaluatorFailurePrefixAndAtomicity(t *testing.T) {
	for _, priorViolation := range []bool{false, true} {
		t.Run(fmt.Sprint(priorViolation), func(t *testing.T) {
			c, cat, view, limits := fixture(t)
			r := c.Rules[0]
			r.Kind = testpilotpb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS
			r.Horizon = &testpilotpb.ContractHorizonDefinition{ElapsedMilliseconds: 5000, ViolationStateId: "bad"}
			r.Transitions[0].TargetStateId = "bad"
			p, err := Prepare(c, cat, view, limits)
			require.NoError(t, err)
			monitor, err := p.New(context.Background(), view)
			require.NoError(t, err)
			e := monitor.(*Evaluator)
			run := &testpilotpb.Run{RunId: "run", ProgramId: "program", Status: testpilotpb.RUN_STATUS_INCOMPLETE, Events: []*testpilotpb.RunEvent{event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED)}}
			_, err = e.Observe(context.Background(), run.Events[0])
			require.NoError(t, err)
			if priorViolation {
				v := observed(2, 1000, 7)
				run.Events = append(run.Events, v)
				d, err := e.Observe(context.Background(), v)
				require.NoError(t, err)
				require.Equal(t, execution.Stop, d)
			}
			sequence := int64(len(run.Events) + 1)
			failed := event(sequence, 5000, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, err = e.Observe(ctx, failed)
			require.ErrorIs(t, err, context.Canceled)
			run.Events = append(run.Events, failed, event(sequence+1, 7000, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED))
			run.EvaluationFailure = &testpilotpb.Run_EvaluationFailureSequence{EvaluationFailureSequence: sequence}
			live, err := e.Close(context.Background(), run)
			require.NoError(t, err)
			want := testpilotpb.VERDICT_STATUS_INCONCLUSIVE
			if priorViolation {
				want = testpilotpb.VERDICT_STATUS_VIOLATED
			}
			require.Equal(t, want, live.Status)
			replay, offline, err := p.evaluate(context.Background(), run)
			require.NoError(t, err)
			require.True(t, proto.Equal(live, offline))
			require.Equal(t, e.trace, replay.trace)
			for _, bad := range []int64{0, sequence + 2} {
				invalidRun := proto.CloneOf(run)
				invalidRun.EvaluationFailure = &testpilotpb.Run_EvaluationFailureSequence{EvaluationFailureSequence: bad}
				_, err := p.Evaluate(context.Background(), invalidRun)
				require.Error(t, err)
			}
		})
	}
}

func TestEvaluatorRuntimeBoundsAndMalformedEvents(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*testpilotpb.RunEvent)
	}{
		{"sequence", func(e *testpilotpb.RunEvent) { e.Sequence = 3 }},
		{"elapsed", func(e *testpilotpb.RunEvent) { e.ElapsedMilliseconds = -1 }},
		{"unknown observation", func(e *testpilotpb.RunEvent) { e.Observations[0].ObservationId = "private-slot" }},
		{"wrong type", func(e *testpilotpb.RunEvent) {
			e.Observations[0].Value = &testpilotpb.Value{Value: &testpilotpb.Value_Text{Text: "not an int"}}
		}},
		{"duplicate observation", func(e *testpilotpb.RunEvent) {
			e.Observations = append(e.Observations, proto.CloneOf(e.Observations[0]))
		}},
		{"bytes", func(e *testpilotpb.RunEvent) {
			e.Observations[0] = &testpilotpb.ObservationResult{ObservationId: "text", Value: &testpilotpb.Value{Value: &testpilotpb.Value_Text{Text: string(make([]byte, 4097))}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, cat, view, limits := fixture(t)
			p, err := Prepare(c, cat, view, limits)
			require.NoError(t, err)
			m, err := p.New(context.Background(), view)
			require.NoError(t, err)
			e := m.(*Evaluator)
			_, err = e.Observe(context.Background(), event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED))
			require.NoError(t, err)
			bad := observed(2, 1000, 7)
			tc.mutate(bad)
			_, err = e.Observe(context.Background(), bad)
			require.Error(t, err)
			require.Empty(t, e.trace)
			require.Equal(t, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, e.verdict(testpilotpb.RUN_STATUS_INCOMPLETE).Status)
		})
	}
}

type cancelOnCheck struct {
	context.Context
	cancel    context.CancelFunc
	remaining int
}

func (c *cancelOnCheck) Err() error {
	c.remaining--
	if c.remaining <= 0 {
		c.cancel()
	}
	return c.Context.Err()
}
func TestEvaluatorEventCommitIsAtomicAcrossRules(t *testing.T) {
	c, cat, view, limits := fixture(t)
	other := proto.CloneOf(c.Rules[0])
	other.RuleId = "other"
	c.Rules = append(c.Rules, other)
	p, err := Prepare(c, cat, view, limits)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &cancelOnCheck{Context: base, cancel: cancel, remaining: 5}
	_, err = e.Observe(ctx, event(2, 1000, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED))
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, e.trace)
	for _, state := range e.rules {
		require.Equal(t, 0, state.state)
		require.Empty(t, state.captures)
		require.Empty(t, state.support)
	}
}
func TestEvaluatorIncompleteCannotAcceptLateWitness(t *testing.T) {
	c, cat, view, limits := fixture(t)
	r := c.Rules[0]
	r.Kind = testpilotpb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS
	r.Horizon = &testpilotpb.ContractHorizonDefinition{ElapsedMilliseconds: 5000, ViolationStateId: "bad"}
	p, err := Prepare(c, cat, view, limits)
	require.NoError(t, err)
	incomplete := event(2, 4000, testpilotpb.RUN_EVENT_KIND_DIAGNOSTIC)
	incomplete.ExecutionIncomplete = true
	run := &testpilotpb.Run{RunId: "run", ProgramId: "program", Status: testpilotpb.RUN_STATUS_INCOMPLETE, Events: []*testpilotpb.RunEvent{event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED), incomplete, observed(3, 6000, 7), event(4, 7000, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED)}}
	e, verdict, err := p.evaluate(context.Background(), run)
	require.NoError(t, err)
	require.Equal(t, testpilotpb.RULE_VERDICT_STATUS_INCONCLUSIVE, verdict.Rules[0].Status)
	require.Empty(t, e.trace)
}
func TestEvaluatorCaptureNamesAreRuleLocal(t *testing.T) {
	c, cat, view, limits := fixture(t)
	first := c.Rules[0]
	addCapture(first)
	first.Transitions[0].Predicate = present(observation("id"))
	assign(first.Transitions[0])
	second := proto.CloneOf(first)
	second.RuleId = "other"
	second.Captures[0].Type.GetScalar().Kind = testpilotpb.SCALAR_KIND_TEXT
	second.Transitions[0].Predicate = present(observation("text"))
	second.Transitions[0].CaptureAssignments[0].Observation.ObservationId = "text"
	c.Rules = append(c.Rules, second)
	p, err := Prepare(c, cat, view, limits)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	values := observed(2, 1000, 7)
	values.Observations = append(values.Observations, &testpilotpb.ObservationResult{ObservationId: "text", Value: &testpilotpb.Value{Value: &testpilotpb.Value_Text{Text: "distinct"}}})
	_, err = e.Observe(context.Background(), values)
	require.NoError(t, err)
	require.Equal(t, "7", e.rules[0].captures["saved"].value.GetSignedInteger())
	require.Equal(t, "distinct", e.rules[1].captures["saved"].value.GetText())
}
func TestEvaluatorEventCountBound(t *testing.T) {
	c, cat, view, limits := fixture(t)
	c.Rules[0].Transitions[0].Predicate = boolean(false)
	p, err := Prepare(c, cat, view, limits)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	_, err = e.Observe(context.Background(), event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED))
	require.NoError(t, err)
	for i := int64(2); i <= view.Limits().MaxRunEvents; i++ {
		_, err = e.Observe(context.Background(), event(i, i, testpilotpb.RUN_EVENT_KIND_DIAGNOSTIC))
		require.NoError(t, err)
	}
	_, err = e.Observe(context.Background(), event(view.Limits().MaxRunEvents+1, 1000, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED))
	require.Error(t, err)
	require.Empty(t, e.trace)
}

func TestEvaluatorCancellationAfterCommitPreservesProof(t *testing.T) {
	c, cat, view, limits := fixture(t)
	c.Rules[0].Transitions[0].TargetStateId = "bad"
	p, err := Prepare(c, cat, view, limits)
	require.NoError(t, err)
	e, err := p.newEvaluator(context.Background(), view)
	require.NoError(t, err)
	run := &testpilotpb.Run{RunId: "run", ProgramId: "program", Status: testpilotpb.RUN_STATUS_INCOMPLETE, Events: []*testpilotpb.RunEvent{event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED), observed(2, 1000, 7), event(3, 2000, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED)}}
	_, err = e.Observe(context.Background(), run.Events[0])
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	decision, err := e.Observe(ctx, run.Events[1])
	require.NoError(t, err)
	require.Equal(t, execution.Stop, decision)
	cancel()
	_, err = e.Observe(context.Background(), run.Events[2])
	require.NoError(t, err)
	live, err := e.Close(ctx, run)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, testpilotpb.VERDICT_STATUS_VIOLATED, live.Status)
	replay, offline, err := p.evaluate(context.Background(), run)
	require.NoError(t, err)
	require.True(t, proto.Equal(live, offline))
	require.Equal(t, e.trace, replay.trace)
}

func TestEvaluatorCloseCancellationAndTerminalState(t *testing.T) {
	for _, violated := range []bool{false, true} {
		for _, check := range []int{1, 3, 130} {
			t.Run(fmt.Sprintf("violated=%v/check=%d", violated, check), func(t *testing.T) {
				c, cat, view, limits := fixture(t)
				if violated {
					c.Rules[0].Transitions[0].TargetStateId = "bad"
				}
				p, err := Prepare(c, cat, view, limits)
				require.NoError(t, err)
				e, err := p.newEvaluator(context.Background(), view)
				require.NoError(t, err)
				run := &testpilotpb.Run{RunId: "run", ProgramId: "program", Status: testpilotpb.RUN_STATUS_INCOMPLETE, Events: []*testpilotpb.RunEvent{event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED), observed(2, 1000, 7)}}
				for i := int64(3); i < 128; i++ {
					run.Events = append(run.Events, event(i, 2000, testpilotpb.RUN_EVENT_KIND_DIAGNOSTIC))
				}
				run.Events = append(run.Events, event(128, 3000, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED))
				for _, v := range run.Events {
					_, err = e.Observe(context.Background(), v)
					require.NoError(t, err)
				}
				base, cancel := context.WithCancel(context.Background())
				defer cancel()
				ctx := &cancelOnCheck{Context: base, cancel: cancel, remaining: check}
				verdict, err := e.Close(ctx, run)
				require.ErrorIs(t, err, context.Canceled)
				want := testpilotpb.VERDICT_STATUS_INCONCLUSIVE
				if violated {
					want = testpilotpb.VERDICT_STATUS_VIOLATED
				}
				require.Equal(t, want, verdict.Status)
				require.Equal(t, []int64{2}, verdict.SupportingEventSequences)
				replay, offline, err := p.evaluate(context.Background(), run)
				require.NoError(t, err)
				require.True(t, proto.Equal(verdict, offline))
				require.Equal(t, e.trace, replay.trace)
				_, err = e.Close(context.Background(), run)
				require.Error(t, err)
				_, err = e.Observe(context.Background(), event(129, 4000, testpilotpb.RUN_EVENT_KIND_DIAGNOSTIC))
				require.Error(t, err)
			})
		}
	}
}
