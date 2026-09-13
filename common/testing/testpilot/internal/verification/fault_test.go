package verification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

func textLiteral(text string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: text}}}}
}

func enumLiteral(name string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: name}}}}}
}

func faultEvent(sequence, elapsed int64, role string, kind testpilotspb.FaultKind) *testpilotspb.RunEvent {
	e := event(sequence, elapsed, testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED)
	e.Payload = &testpilotspb.RunEvent_FaultInjected{FaultInjected: &testpilotspb.FaultInjected{RoleId: role, Kind: kind}}
	return e
}

// payloadField reads one field of a Run Event payload arm through a path from the payload reference.
func payloadField(arm, field string) *testpilotspb.Expression {
	payload := &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_RunEvent{RunEvent: &testpilotspb.RunEventReference{Selection: &testpilotspb.RunEventReference_Payload{Payload: &testpilotspb.RunEventPayloadReference{}}}}}}}
	path := arm + "." + field
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: payload, Path: path}}}
}

// The whole evidence path in one pass: a Contract transition filtered on the fault event kind and
// predicated on both fields of its payload is admitted, matches the recorded fault online, and the
// offline replay produces the identical Verdict. A rule that only compiled would prove nothing. An
// event of a kind that may lack the arm reads its fields as absent, which no comparison matches.
func TestEvaluatorMatchesRecordedFaultPayload(t *testing.T) {
	faulted := []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED}
	for _, tc := range []struct {
		name     string
		kinds    []testpilotspb.RunEventKind
		recorded *testpilotspb.RunEvent
		want     testpilotspb.VerdictStatus
	}{
		{"matching fault", faulted, faultEvent(2, 10, "queue", testpilotspb.FAULT_KIND_WORKER_STOP), testpilotspb.VERDICT_STATUS_SATISFIED},
		{"other kind", faulted, faultEvent(2, 10, "queue", testpilotspb.FAULT_KIND_WORKER_RESUME), testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
		{"other role", faulted, faultEvent(2, 10, "other", testpilotspb.FAULT_KIND_WORKER_STOP), testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
		{"absent arm", append([]testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}, faulted...), event(2, 10, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED), testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, cat, view, limits := fixture(t)
			r := c.Rules[0]
			r.Transitions[0].EventFilter.Kinds = tc.kinds
			r.Transitions[0].Predicate = all(
				equal(payloadField("fault_injected", "role_id"), textLiteral("queue")),
				equal(payloadField("fault_injected", "kind"), enumLiteral("FAULT_KIND_WORKER_STOP")),
			)
			p, err := Prepare(c, cat, view, limits, nil)
			require.NoError(t, err)

			run := &testpilotspb.Run{RunId: "run", CaseId: "case", ProgramId: "program", Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Events: []*testpilotspb.RunEvent{
				event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED),
				tc.recorded,
				event(3, 20, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED),
			}}
			monitor, err := p.New(context.Background(), view)
			require.NoError(t, err)
			online := monitor.(*Evaluator)
			for _, e := range run.Events {
				_, err := online.Observe(context.Background(), e)
				require.NoError(t, err)
			}
			live, err := online.Close(context.Background(), run)
			require.NoError(t, err)
			require.Equal(t, tc.want, live.Status)

			_, verdict, err := p.evaluate(context.Background(), run)
			require.NoError(t, err)
			liveBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(live)
			require.NoError(t, err)
			offlineBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(verdict)
			require.NoError(t, err)
			require.Equal(t, liveBytes, offlineBytes)
		})
	}
}

// A payload path is typed by the arm its first segment names. It rejects, located at that segment,
// when no kind the transition considers can carry the arm; it is absent when some considered kind may
// lack the arm, which a comparison reads without a presence guard; and it is present when every
// considered kind requires the arm.
func TestPrepareLocatesPayloadPathsTheFilterCannotCarry(t *testing.T) {
	located := "contract.rules[rule].transitions[first].predicate.compare.left.path.path"
	completed := testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED
	faulted := testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED
	faultKind := equal(payloadField("fault_injected", "kind"), enumLiteral("FAULT_KIND_WORKER_STOP"))
	for _, tc := range []struct {
		name      string
		kinds     []testpilotspb.RunEventKind
		predicate *testpilotspb.Expression
		category  ir.ErrorCategory
		path      string
	}{
		{name: "an arm every kind requires", kinds: []testpilotspb.RunEventKind{faulted}, predicate: faultKind},
		{name: "an arm no kind carries", kinds: []testpilotspb.RunEventKind{completed}, predicate: faultKind, category: ir.Unknown, path: located},
		{name: "an arm some kind may lack", kinds: []testpilotspb.RunEventKind{completed, faulted}, predicate: faultKind},
		{name: "a guarded arm some kind may lack", kinds: []testpilotspb.RunEventKind{completed, faulted}, predicate: all(present(payloadField("fault_injected", "kind")), faultKind)},
		{name: "an optional arm", kinds: []testpilotspb.RunEventKind{completed}, predicate: equal(payloadField("outcome", "detail"), textLiteral(""))},
		{name: "an unknown arm", kinds: []testpilotspb.RunEventKind{faulted}, predicate: equal(payloadField("observations", "observation_id"), textLiteral("")), category: ir.Unknown, path: located},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, cat, view, limits := fixture(t)
			c.Rules[0].Transitions[0].EventFilter.Kinds = tc.kinds
			c.Rules[0].Transitions[0].Predicate = tc.predicate
			_, err := Prepare(c, cat, view, limits, nil)
			if tc.category == "" {
				require.NoError(t, err)
				return
			}
			var admissionErr *ir.Error
			require.ErrorAs(t, err, &admissionErr)
			require.Equal(t, tc.category, admissionErr.Category)
			if tc.path != "" {
				require.Equal(t, tc.path, admissionErr.Path)
			}
		})
	}
}

// A payload the evaluated event does not carry is absent, so a guard over it is false rather than a
// read of a zero value.
func TestEvaluatorReadsAnUncarriedPayloadAsAbsent(t *testing.T) {
	c, cat, view, limits := fixture(t)
	r := c.Rules[0]
	r.Transitions[0].EventFilter.Kinds = []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}
	r.Transitions[0].Predicate = present(payloadField("outcome", "detail"))
	p, err := Prepare(c, cat, view, limits, nil)
	require.NoError(t, err)
	carrying := event(3, 20, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED)
	carrying.Payload = &testpilotspb.RunEvent_Outcome{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}
	run := &testpilotspb.Run{RunId: "run", CaseId: "case", ProgramId: "program", Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Events: []*testpilotspb.RunEvent{
		event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED),
		event(2, 10, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED),
		carrying,
		event(4, 30, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED),
	}}
	_, verdict, err := p.evaluate(context.Background(), run)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.Status)
	require.Equal(t, []int64{3}, verdict.Rules[0].SupportingEventSequences)
}
