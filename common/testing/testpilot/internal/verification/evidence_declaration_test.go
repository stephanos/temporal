package verification

import (
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/casefile"
	"go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// declaredView supplies Program extraction declarations independently of Contract evidence policy.
func declaredView(t *testing.T, catalog *ir.Catalog, view execution.ProgramView, kinds []string, fields ...string) execution.ProgramView {
	t.Helper()
	program := &testpilotspb.Program{ProgramId: "declared.program", Observations: []*testpilotspb.Observation{{ObservationId: "evidence", Type: messageType("temporal.server.api.testpilot.v1.CorrelatedEvidence")}}, Entrypoints: []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &emptypb.Empty{}}}}, Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"}}
	for index, kind := range kinds {
		// The Contract counts ordinals in one source; the other kinds count in sources of their
		// own so that no two declarations share a source and key path.
		source := "source"
		if index > 0 {
			source += "-" + kind
		}
		value := cel.Path(cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &emptypb.Empty{}}}), "role_id")
		declaration := &testpilotspb.EvidenceDeclaration{EvidenceId: kind, Kind: kind, EvidenceSource: source, Operation: value, Source: &testpilotspb.EvidenceDeclaration_RunEvent{RunEvent: &testpilotspb.RunEventSource{Kind: testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED}}}
		for _, field := range fields {
			declaration.Fields = append(declaration.Fields, &testpilotspb.NamedExpression{FieldId: field, Value: proto.CloneOf(value)})
		}
		program.Evidence = append(program.Evidence, declaration)
	}
	prepared, err := execution.Prepare(&testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: casefile.CurrentMajor, Minor: casefile.CurrentMinor}, CaseId: "declared.case", Program: program, Contract: &testpilotspb.Contract{ContractId: "declared"}}, catalog, execution.Profile{Identity: "host", CatalogIdentity: catalog.Identity(), Limits: view.Limits()})
	require.NoError(t, err)
	return prepared.View()
}

func TestCorrelatedExternalPolicyIsIndependentOfProgramDeclarations(t *testing.T) {
	contract, catalog, view, ceiling, correlated := correlatedFixture(t, 0)
	view = declaredView(t, catalog, view, []string{"request"})
	require.Len(t, view.Evidence(), 1, "the generic path must survive a nonempty extraction catalog")
	contract.Correlated.Sources = append(contract.Correlated.Sources, "external-source")
	contract.Correlated.ProjectionRules = append(contract.Correlated.ProjectionRules, &testpilotspb.CorrelatedProjectionRule{
		Kind: "external-kind", Meaning: testpilotspb.CORRELATED_EVIDENCE_MEANING_CONFIRMED, ResultIds: []string{"both"},
		Fields: []*testpilotspb.CorrelatedFieldPolicy{{FieldId: "external-field", Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN}},
	})
	contract.Correlated.Rules[0].Correlation = equal(correlatedFieldOperand("external-field"), textLiteral("kept"))
	correlated.MaxCorrelationDepth = nativeCorrelationDepth(contract.Correlated.Rules[0].Correlation.Cel.Expr)
	prepared, err := Prepare(contract, catalog, view, ceiling, correlated)
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		mutate   func(*testpilotspb.CorrelatedEvidence)
		category ir.ErrorCategory
	}{
		"supported-external-evidence": {},
		"disallowed-kind":             {category: ir.Unknown, mutate: func(e *testpilotspb.CorrelatedEvidence) { e.Kind = "outside-policy" }},
		"disallowed-source":           {category: ir.Malformed, mutate: func(e *testpilotspb.CorrelatedEvidence) { e.Identity.EvidenceSource = "outside-policy" }},
		"disallowed-scope":            {category: ir.Malformed, mutate: func(e *testpilotspb.CorrelatedEvidence) { e.Identity.Scope[0].FieldId = "outside-policy" }},
		"retained-field-outside-policy": {category: ir.Malformed, mutate: func(e *testpilotspb.CorrelatedEvidence) {
			e.Fields = append(e.Fields, &testpilotspb.NamedValue{FieldId: "outside-policy", Value: &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: "kept"}}})
		}},
		"wrong-field-type": {category: ir.TypeMismatch, mutate: func(e *testpilotspb.CorrelatedEvidence) {
			e.Fields[0].Value = &celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: true}}
		}},
	} {
		t.Run(name, func(t *testing.T) {
			evidence := correlatedEvidence(0, "external-kind", "a")
			evidence.Identity.EvidenceSource = "external-source"
			evidence.Fields = []*testpilotspb.NamedValue{{FieldId: "external-field", Value: &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: "kept"}}}}
			if tc.mutate != nil {
				tc.mutate(evidence)
			}
			evaluator, err := prepared.newEvaluator(t.Context(), view)
			require.NoError(t, err)
			run := &testpilotspb.Run{RunId: "run", CaseId: "declared.case", ProgramId: view.ProgramID(), Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Events: []*testpilotspb.RunEvent{
				event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED), correlatedEvent(t, 2, evidence), event(3, 0, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED),
			}}
			_, err = evaluator.Observe(t.Context(), run.Events[0])
			require.NoError(t, err)
			_, err = evaluator.Observe(t.Context(), run.Events[1])
			if tc.mutate != nil {
				var diagnostic *ir.Error
				require.ErrorAs(t, err, &diagnostic)
				require.Equal(t, tc.category, diagnostic.Category)
				require.Empty(t, evaluator.correlated.accepted)
				require.Zero(t, evaluator.correlated.transitions)
				return
			}
			require.NoError(t, err)
			_, err = evaluator.Observe(t.Context(), run.Events[2])
			require.NoError(t, err)
			live, err := evaluator.Close(t.Context(), run)
			require.NoError(t, err)
			require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, live.Status)
			_, offline, err := prepared.evaluate(t.Context(), run)
			require.NoError(t, err)
			require.True(t, proto.Equal(live, offline), "generic external evidence replays through native CEL")
			require.Equal(t, "external-field", evaluator.correlated.accepted[0].Fields[0].FieldId)
			require.Equal(t, "kept", evaluator.correlated.accepted[0].Fields[0].GetValue().GetStringValue())
		})
	}
}
