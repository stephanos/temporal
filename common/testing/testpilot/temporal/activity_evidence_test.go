package temporal_test

import (
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/types/known/emptypb"
)

func projectedPath(path string) *testpilotspb.Expression {
	return cel.Path(cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &emptypb.Empty{}}}), path)
}

func compared(operator string, left *testpilotspb.Expression, right *celpb.Value) *testpilotspb.Expression {
	return cel.Compare(operator, left, cel.Literal(right))
}

func stepReference(field testpilotspb.CorrelatedStepField, definitionID string) *testpilotspb.Expression {
	return cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{Field: field, DefinitionId: definitionID}}})
}

func textOf(text string) *celpb.Value {
	return &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: text}}
}

// deliveredAttempts makes the Case read its evidence from the Run's own record of each attempt a
// worker was delivered for the activity its start instruction started, keyed by the Run, and hold
// every delivery to answer with the outcome: each delivered attempt is one step of the Run's
// activity whose outcome is "started".
func deliveredAttempts(source *testpilotspb.Case, answer string) {
	source.Program.Observations = []*testpilotspb.Observation{{ObservationId: "evidence", Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.CorrelatedEvidence"}}}}}}}
	source.Program.Evidence = []*testpilotspb.EvidenceDeclaration{{
		EvidenceId: "attemptDelivered", EvidenceSource: "attempts", Kind: "attemptDelivered",
		Source: &testpilotspb.EvidenceDeclaration_RunEvent{RunEvent: &testpilotspb.RunEventSource{
			Kind:        testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC,
			Instruction: &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "start-activity"}, RunKeyed: true,
		}},
		Guard: cel.All(
			cel.Present(projectedPath("activity_attempt.sdk_attempt")),
			compared("_>_", projectedPath("activity_attempt.sdk_attempt"), &celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: 0}}),
			cel.Present(projectedPath("activity_attempt.delivery_id")),
			compared("_!=_", projectedPath("activity_attempt.delivery_id"), textOf("")),
		),
		Scope: testsupport.LiteralExpressions([]*testpilotspb.NamedValue{{FieldId: "run", Value: textOf("one")}}),
		Fields: []*testpilotspb.NamedExpression{
			{FieldId: "attempt", Value: projectedPath("activity_attempt.sdk_attempt")},
			{FieldId: "delivery", Value: projectedPath("activity_attempt.delivery_id")},
		},
	}}
	model := func(definition, value string) *testpilotspb.ModelValue {
		return &testpilotspb.ModelValue{DefinitionId: definition, Value: value}
	}
	source.Contract = &testpilotspb.Contract{ContractId: "contract", Correlated: &testpilotspb.CorrelatedContract{
		ProjectionId: "projection", ProjectionFingerprint: "projection-v1", EvidenceObservationId: "evidence",
		ScopeFields: []string{"run"}, OperationField: "operation", Sources: []string{"attempts"},
		InitialStateId: "scheduled",
		States: []*testpilotspb.CorrelatedState{
			{StateId: "scheduled", Atom: model("state", "scheduled")},
			{StateId: "attempted", Atom: model("state", "attempted")},
		},
		Results: []*testpilotspb.CorrelatedResult{{ResultId: "delivered", Action: model("action", "deliver"), StateId: "attempted", Outcome: model("outcome", "started")}},
		Transitions: []*testpilotspb.CorrelatedTransition{
			{PriorStateId: "scheduled", ResultId: "delivered"}, {PriorStateId: "attempted", ResultId: "delivered"},
		},
		ProjectionRules: []*testpilotspb.CorrelatedProjectionRule{{
			Kind: "attemptDelivered", Meaning: testpilotspb.CORRELATED_EVIDENCE_MEANING_CONFIRMED, ResultIds: []string{"delivered"},
			Fields: []*testpilotspb.CorrelatedFieldPolicy{
				{FieldId: "attempt", Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_UINT64}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN},
				{FieldId: "delivery", Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN},
			},
		}},
		Rules: []*testpilotspb.CorrelatedRule{{
			RuleId:   "delivery-answers",
			Trigger:  compared("_>_", cel.Size(stepReference(testpilotspb.CORRELATED_STEP_FIELD_ACTION, "action")), &celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: 0}}),
			Response: cel.Compare("@in", cel.Literal(textOf(answer)), stepReference(testpilotspb.CORRELATED_STEP_FIELD_OUTCOME, "outcome")),
			Ending:   testpilotspb.TRACE_ENDING_PARTIAL,
		}},
	}}
}

// attemptEvidence is, for each attempt record of the Run in order, its source and the evidence the
// Run carries beside it, an empty value where it carries none.
func attemptEvidence(t *testing.T, run *testpilotspb.Run) ([]string, []*testpilotspb.CorrelatedEvidence) {
	t.Helper()
	var sources []string
	var lifted []*testpilotspb.CorrelatedEvidence
	for _, event := range run.GetEvents() {
		if event.GetOutcome().GetActivityAttempt() == nil {
			continue
		}
		sources = append(sources, event.GetSourceId())
		evidence := &testpilotspb.CorrelatedEvidence{}
		if len(event.GetObservations()) > 0 {
			require.Len(t, event.GetObservations(), 1)
			require.Equal(t, "evidence", event.GetObservations()[0].GetObservationId())
			require.NoError(t, event.GetObservations()[0].GetValue().GetObjectValue().UnmarshalTo(evidence))
		}
		lifted = append(lifted, evidence)
	}
	return sources, lifted
}

// Evidence a guard selects from the Run's record of an activity's attempts is lifted once, as the
// Run records each attempt, and the record carries it. The Contract therefore concludes the same
// from the live Run, through the composite Driver, a real SDK worker and a server that retries, as
// from the recorded Run replayed with no Driver: the same Verdict, and the same evidence behind a
// violation. A declared position no attempt was delivered for is recorded and is no evidence.
func TestActivityAttemptEvidenceIsTheSameLiveAndReplayed(t *testing.T) {
	const first, second, third = "scheduler.g0.n0.a1.r0.i0", "scheduler.g0.n0.a1.r0.i1", "scheduler.g0.n0.a1.r0.i2"
	delivered := func(ordinal int64, attempt uint64, token string) *testpilotspb.CorrelatedEvidence {
		return &testpilotspb.CorrelatedEvidence{
			Kind:     "attemptDelivered",
			Identity: &testpilotspb.CorrelatedIdentity{EvidenceSource: "attempts", Ordinal: ordinal, Scope: []*testpilotspb.NamedValue{{FieldId: "run", Value: textOf("one")}}},
			Fields: []*testpilotspb.NamedValue{
				{FieldId: "attempt", Value: &celpb.Value{Kind: &celpb.Value_Uint64Value{Uint64Value: attempt}}},
				{FieldId: "delivery", Value: textOf(deliveryOf(token))},
			},
		}
	}
	for name, test := range map[string]struct {
		script      []*testpilotspb.InstructionNode
		answer      string
		sources     []string
		evidence    []*testpilotspb.CorrelatedEvidence
		status      testpilotspb.VerdictStatus
		disposition testpilotspb.RunDisposition
		violations  func(*testpilotspb.Run) []testpilot.RuleViolation
	}{
		"a failed attempt, its retry and a position not needed": {
			script:      []*testpilotspb.InstructionNode{attemptFailure("first-attempt", "transient", false), attemptFinish("second-attempt", textLiteral("done")), attemptFinish("third-attempt", textLiteral("late"))},
			answer:      "started",
			sources:     []string{first, second, third},
			evidence:    []*testpilotspb.CorrelatedEvidence{delivered(0, 1, "token-1"), delivered(1, 2, "token-2"), {}},
			status:      testpilotspb.VERDICT_STATUS_SATISFIED,
			disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
			violations:  func(*testpilotspb.Run) []testpilot.RuleViolation { return nil },
		},
		"a delivery the Contract holds to another outcome": {
			script:      []*testpilotspb.InstructionNode{attemptFinish("first-attempt", textLiteral("done"))},
			answer:      "accepted",
			sources:     []string{first},
			evidence:    []*testpilotspb.CorrelatedEvidence{delivered(0, 1, "token-1")},
			status:      testpilotspb.VERDICT_STATUS_VIOLATED,
			disposition: testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR,
			violations: func(run *testpilotspb.Run) []testpilot.RuleViolation {
				for _, event := range run.GetEvents() {
					if event.GetSourceId() == first {
						return []testpilot.RuleViolation{{RuleID: "delivery-answers", Sequence: event.GetSequence(), CorrelatedKind: "attemptDelivered"}}
					}
				}
				return nil
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			source := activityScriptCase(test.script)
			deliveredAttempts(source, test.answer)
			server := &activityServer{tasks: make(chan *workflowservice.PollActivityTaskQueueResponse, 8), closed: make(chan struct{})}
			prepared, run, verdict := runActivityScript(t, server, source)

			sources, evidence := attemptEvidence(t, run)
			require.Equal(t, test.sources, sources)
			for _, lifted := range test.evidence {
				if lifted.GetKind() != "" {
					lifted.Operation = run.GetRunId()
				}
			}
			protorequire.ProtoSliceEqual(t, test.evidence, evidence)
			require.Equal(t, test.status, verdict.GetStatus(), run.GetDiagnostics())
			require.Equal(t, test.disposition, run.GetDisposition(), run.GetDiagnostics())
			require.Empty(t, run.GetDiagnostics())

			replayed, evaluation, err := prepared.Evaluate(t.Context(), run)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, verdict, replayed)
			protorequire.ProtoEqual(t, run.GetVerdict(), replayed)
			require.Equal(t, &testpilot.Evaluation{Violations: test.violations(run)}, evaluation)
		})
	}
}
