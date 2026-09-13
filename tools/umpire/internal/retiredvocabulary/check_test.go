package retiredvocabulary

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Token literals are split so this file, which the scan reads like any other,
// does not trip the rules it exercises.
func TestRetiredRulesAreConfigured(t *testing.T) {
	t.Parallel()

	require.NoError(t, retiredRulesError)
	require.NotEmpty(t, retiredRules)
}

func TestAllowedNegativeFixtureHoldsRetiredTokensOnlyUnderTheMigrationBaseline(t *testing.T) {
	t.Parallel()

	token := "Transition" + "Kernel"
	require.True(t, allowedNegativeFixture(
		"common/testing/testpilot/internal/protocolmigration/testdata/baseline/fixtures/tests/testcore/testpilot/testdata/typed-nexus-case.json",
		token,
	))
	for _, path := range []string{
		"common/testing/testpilot/internal/protocolmigration/mapping.go",
		"common/testing/testpilot/internal/protocolmigration/README.md",
		"common/testing/testpilot/internal/protocolmigration/testdata/baseline-copy/case.json",
		"tests/testcore/testpilot/testdata/typed-nexus-case.json",
	} {
		require.False(t, allowedNegativeFixture(path, token), "path %q", path)
	}
}

func TestValidateRetiredTokenRejectsBareWords(t *testing.T) {
	t.Parallel()

	for _, token := range []string{"Target", "Behavior", "Case", "Step", "bou" + "nds", "T"} {
		require.ErrorContains(t, validateRetiredToken(token), "bare word", "token %q", token)
	}
	require.ErrorContains(t, validateRetiredToken(""), "must not be empty")

	for _, token := range []string{
		"Transition" + "Kernel",
		"semantic" + "Identity",
		"umpire-gen-regression-" + "projections",
		"Umpire." + "Refinement",
		"umpire-experiment/" + "v1",
		"await_" + "outcome",
		"behavior" + "%",
	} {
		require.NoError(t, validateRetiredToken(token), "token %q", token)
	}
}

func TestRetiredRulesHoldTheGlossaryRenamedProtocolNames(t *testing.T) {
	t.Parallel()

	matched := func(line string) []string {
		var names []string
		for _, rule := range retiredRules {
			if rule.pattern.MatchString(line) {
				names = append(names, rule.name)
			}
		}
		return names
	}
	for _, tc := range []struct {
		line string
		want []string
	}{
		{line: "status : " + "Run" + "Status", want: []string{"Run" + "Status"}},
		{line: "run" + "Status := completed", want: []string{"Run" + "Status"}},
		{line: "{ clause" + "_id := id }", want: []string{"clause" + "_id"}},
		{line: "rule.GetClause" + "Id()", want: []string{"GetClause" + "Id"}},
		{line: "(value : Correlated" + "Value)", want: []string{"Correlated" + "Value"}},
		{line: "[]*pb.ContractRule" + "Definition{}", want: []string{"ContractRule" + "Definition"}},
		{line: "[]*pb.ContractState" + "Definition{}", want: []string{"ContractState" + "Definition"}},
		{line: "[]*pb.ContractTransition" + "Definition{}", want: []string{"ContractTransition" + "Definition"}},
		{line: "[]*pb.ContractCapture" + "Definition{}", want: []string{"ContractCapture" + "Definition"}},
		{line: "pb.RUN_" + "STATUS_COMPLETED", want: []string{"RUN_" + "STATUS_*"}},
		{line: `"status": "CONTRACT_STATE_STATUS_` + `NONTERMINAL"`, want: []string{"CONTRACT_STATE_STATUS_" + "NONTERMINAL"}},
		{line: "pb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_" + "NON_SUCCESS", want: []string{"PROTOCOL_" + "NON_SUCCESS"}},
		{line: `return "PROTOCOL_` + `NON_SUCCESS"`, want: []string{"PROTOCOL_" + "NON_SUCCESS"}},
		// Umpire's Property clauses keep their identifier, and the renamed names are live.
		{line: "result.clauseId == clause.id"},
		{line: "pb.RUN_DISPOSITION_COMPLETED, pb.CONTRACT_STATE_STATUS_PENDING, pb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE"},
		{line: "[]*pb.ContractRule{}, pb.ContractRuleKind, ModelValue, RunDisposition, rule_id"},
		{line: "delivery.TriggerNonSuccess"},
		{line: "[]*pb.Response" + "Projection{}", want: []string{"Response" + "Projection"}},
		{line: `"response` + `Projections": [`, want: []string{"Response" + "Projections"}},
		{line: "Program.response" + "Projection path", want: []string{"Response" + "Projection"}},
		{line: "(targets : Array Projection" + "Target)", want: []string{"Projection" + "Target"}},
		{line: "(kind : Projection" + "Kind)", want: []string{"Projection" + "Kind"}},
		{line: `"kind": "PROJECTION_` + `KIND_ONE"`, want: []string{"PROJECTION_" + "KIND_*"}},
		{line: "&pb.OpaqueCapability" + "Type{}", want: []string{"OpaqueCapability" + "Type"}},
		{line: "string capability" + "_slot_id = 1;", want: []string{"capability" + "_slot_id"}},
		{line: `"capability` + `SlotId": "authority"`, want: []string{"Capability" + "SlotId"}},
		{line: "rpc.Capability" + "SlotId", want: []string{"Capability" + "SlotId"}},
		{line: "Program.invoke" + "RPC role method", want: []string{"invoke" + "RPC"}},
		{line: "[]*pb.Role" + "Definition{}", want: []string{"Role" + "Definition"}},
		{line: "[]*pb.Slot" + "Definition{}", want: []string{"Slot" + "Definition"}},
		{line: "[]*pb.Observation" + "Definition{}", want: []string{"Observation" + "Definition"}},
		{line: "*pb.Entrypoint" + "Definition", want: []string{"Entrypoint" + "Definition"}},
		{line: "*pb.Cleanup" + "Definition", want: []string{"Cleanup" + "Definition"}},
		{line: "[]*pb.Instruction" + "Definition{}", want: []string{"Instruction" + "Definition"}},
		{line: "(dependencies : Array Instruction" + "Ref)", want: []string{"Instruction" + "Ref"}},
		// The Opcode and Driver method keep the Go initialism; the model Projection, the Umpire role
		// Definition ID and the renamed names are live.
		{line: "contract.InvokeRPC, session.InvokeRPC(ctx), pb.InvokeRpc{}, Program.invokeRpc"},
		{line: "Umpire.Case.Projection.lower, CorrelatedEvidenceProjection, RoleDefinitionID, roleDefinitionId"},
		{line: "pb.ResponseRead{}, pb.ReadTarget_SlotId, pb.READ_CARDINALITY_ONE, pb.OpaqueHandleType{}, handle_slot_id"},
		{line: "[]*pb.Role{}, pb.Slot_OpaqueHandle, pb.Entrypoint_Workflow, pb.InstructionNode{}, pb.InstructionReference{}, InstructionOutcomeRef"},
	} {
		require.Equal(t, tc.want, matched(tc.line), "line %q", tc.line)
	}
}
