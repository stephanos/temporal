package verification

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

// declaredView prepares the correlated fixture's Program with one Run Event declaration per
// projection kind it names, so the Contract's rules resolve against declarations rather than
// spelling them.
func declaredView(t *testing.T, catalog *ir.Catalog, view execution.ProgramView, kinds []string, fields ...string) execution.ProgramView {
	t.Helper()
	program := &testpilotspb.Program{ProgramId: "declared.program", Observations: []*testpilotspb.Observation{{ObservationId: "evidence", Type: messageType("temporal.server.api.testpilot.v1.CorrelatedEvidence")}}, Entrypoints: []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}}}, Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"}}
	for index, kind := range kinds {
		// The Contract counts ordinals in one source; the other kinds count in sources of their
		// own so that no two declarations share a source and key path.
		source := "source"
		if index > 0 {
			source += "-" + kind
		}
		declaration := &testpilotspb.EvidenceDeclaration{EvidenceId: kind, EvidenceSource: source, Operation: "role_id", Source: &testpilotspb.EvidenceDeclaration_RunEvent{RunEvent: &testpilotspb.RunEventSource{Kind: testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED}}}
		for _, field := range fields {
			declaration.Fields = append(declaration.Fields, &testpilotspb.EvidenceFieldDeclaration{FieldId: field, Path: "role_id"})
		}
		program.Evidence = append(program.Evidence, declaration)
	}
	prepared, err := execution.Prepare(&testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "declared.case", Program: program, Contract: &testpilotspb.Contract{ContractId: "declared"}}, catalog, execution.Profile{Identity: "host", CatalogIdentity: catalog.Identity(), Limits: view.Limits()})
	require.NoError(t, err)
	return prepared.View()
}

// A Program that declares its evidence is the one place the kinds, sources and fields are written,
// so a correlated Contract resolves each projection rule against it and rejects what it does not
// declare. A Program that declares nothing leaves the Contract's rules as they are.
func TestCorrelatedPrepareResolvesProjectionRulesAgainstDeclarations(t *testing.T) {
	contract, catalog, view, ceiling, correlated := correlatedFixture(t, 4)
	require.Empty(t, view.Evidence())
	_, err := Prepare(contract, catalog, view, ceiling, correlated)
	require.NoError(t, err)
	kinds := []string{"request", "both", "tick", "reply", "poll"}
	_, err = Prepare(contract, catalog, declaredView(t, catalog, view, kinds), ceiling, correlated)
	require.NoError(t, err)
	for name, tc := range map[string]struct {
		kinds  []string
		fields []string
		mutate func(*testpilotspb.Contract)
		detail string
	}{
		"undeclared kind": {kinds: kinds[:4], detail: "correlated evidence kind poll is not declared by the Program"},
		"undeclared source": {kinds: kinds, mutate: func(c *testpilotspb.Contract) { c.Correlated.Sources = append(c.Correlated.Sources, "other") },
			detail: "correlated evidence source other is not declared by the Program"},
		"undeclared field": {kinds: kinds, mutate: func(c *testpilotspb.Contract) {
			c.Correlated.ProjectionRules[0].Fields = []*testpilotspb.CorrelatedFieldPolicy{{FieldId: "count", Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN}}
		}, detail: "correlated evidence field count is not declared by evidence request"},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := proto.CloneOf(contract)
			if tc.mutate != nil {
				tc.mutate(candidate)
			}
			_, err := Prepare(candidate, catalog, declaredView(t, catalog, view, tc.kinds, tc.fields...), ceiling, correlated)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, ir.Unknown, diagnostic.Category)
			require.Equal(t, tc.detail, diagnostic.Detail)
		})
	}
}
