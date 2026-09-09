package verification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

func runEventField(field testpilotspb.RunEventField) *testpilotspb.ContractExpression {
	return &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_RunEvent{RunEvent: &testpilotspb.RunEventFieldRef{Field: field}}}
}

func textLiteral(text string) *testpilotspb.ContractExpression {
	return &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: text}}}}
}

func enumLiteral(number int32) *testpilotspb.ContractExpression {
	return &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: number}}}}}
}

func faultEvent(sequence, elapsed int64, role string, kind testpilotspb.FaultKind) *testpilotspb.RunEvent {
	e := event(sequence, elapsed, testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED)
	e.FaultInjected = &testpilotspb.FaultInjected{RoleId: role, Kind: kind}
	return e
}

// The whole evidence path in one pass: a Contract transition filtered on the new event kind and
// predicated on both new event fields is admitted, matches the recorded fault online, and the
// offline replay produces the identical Verdict. A rule that only compiled would prove nothing.
func TestEvaluatorMatchesRecordedFaultFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind testpilotspb.FaultKind
		role string
		want testpilotspb.VerdictStatus
	}{
		{"matching fault", testpilotspb.FAULT_KIND_WORKER_STOP, "queue", testpilotspb.VERDICT_STATUS_SATISFIED},
		{"other kind", testpilotspb.FAULT_KIND_WORKER_RESUME, "queue", testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
		{"other role", testpilotspb.FAULT_KIND_WORKER_STOP, "other", testpilotspb.VERDICT_STATUS_INCONCLUSIVE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, cat, view, limits := fixture(t)
			r := c.Rules[0]
			r.Transitions[0].EventFilter.Kinds = []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED}
			r.Transitions[0].Predicate = all(
				equal(runEventField(testpilotspb.RUN_EVENT_FIELD_FAULT_ROLE_ID), textLiteral("queue")),
				equal(runEventField(testpilotspb.RUN_EVENT_FIELD_FAULT_KIND), enumLiteral(int32(testpilotspb.FAULT_KIND_WORKER_STOP))),
			)
			p, err := Prepare(c, cat, view, limits)
			require.NoError(t, err)

			run := &testpilotspb.Run{RunId: "run", CaseId: "case", ProgramId: "program", Status: testpilotspb.RUN_STATUS_COMPLETED, Events: []*testpilotspb.RunEvent{
				event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED),
				faultEvent(2, 10, tc.role, tc.kind),
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

func TestPrepareAdmitsFaultEventReferences(t *testing.T) {
	for _, tc := range []struct {
		name    string
		field   testpilotspb.RunEventField
		admit   bool
		operand *testpilotspb.ContractExpression
	}{
		{"fault role id", testpilotspb.RUN_EVENT_FIELD_FAULT_ROLE_ID, true, textLiteral("queue")},
		{"fault kind", testpilotspb.RUN_EVENT_FIELD_FAULT_KIND, true, enumLiteral(int32(testpilotspb.FAULT_KIND_WORKER_STOP))},
		{"beyond the declared range", testpilotspb.RUN_EVENT_FIELD_FAULT_KIND + 1, false, textLiteral("queue")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, cat, view, limits := fixture(t)
			c.Rules[0].Transitions[0].Predicate = equal(runEventField(tc.field), tc.operand)
			_, err := Prepare(c, cat, view, limits)
			if tc.admit {
				require.NoError(t, err)
				return
			}
			// FAULT_KIND is the last declared field, so a number past it is not an enum value
			// at all and the surface check rejects it before the reference range is consulted.
			var admissionErr *ir.Error
			require.ErrorAs(t, err, &admissionErr)
			require.Equal(t, ir.Unknown, admissionErr.Category)
		})
	}
}
