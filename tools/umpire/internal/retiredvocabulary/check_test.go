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
		{line: `"field": "RUN_EVENT_FIELD_` + `FAULT_KIND"`, want: []string{"RUN_EVENT_FIELD_" + "FAULT_*"}},
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
		{line: "[]*pb.Role{}, pb.Slot_OpaqueHandle, pb.Entrypoint_Workflow, pb.InstructionNode{}, pb.InstructionReference{}"},
		{line: "*pb.Program" + "Expression", want: []string{"Program" + "Expression"}},
		{line: "(predicate : Contract" + "Expression)", want: []string{"Contract" + "Expression"}},
		{line: "Program" + "Expr.slot id", want: []string{"Program" + "Expr"}},
		{line: "Testpilot.Authoring.Contract" + "Expr.capture id", want: []string{"Contract" + "Expr"}},
		{line: "&pb.ProgramEquals" + "Expression{}", want: []string{"ProgramEquals" + "Expression"}},
		{line: "pb.Contract" + "CompareExpression{}", want: []string{"Contract" + "CompareExpression"}},
		{line: "&pb.Slot" + "Ref{SlotId: id}", want: []string{"Slot" + "Ref"}},
		{line: "&pb.Observation" + "Ref{}", want: []string{"Observation" + "Ref"}},
		{line: "pb.InstructionOutcome" + "Ref{}", want: []string{"InstructionOutcome" + "Ref"}},
		{line: "pb.RunEventField" + "Ref{}", want: []string{"RunEventField" + "Ref"}},
		{line: "let reference : CorrelatedCapture" + "Ref := {}", want: []string{"CorrelatedCapture" + "Ref"}},
		// One Expression over one Reference replaces them; the Umpire and Go names that share a word stay.
		{line: "pb.Expression{}, pb.Reference_SlotId, pb.InstructionOutcomeReference{}, pb.RunReference{}, pb.RunEventReference{}, pb.CorrelatedCaptureReference{}"},
		{line: "Testpilot.Authoring.Expr.negate, pb.CompareExpression{}, pb.NotExpression{}, ir.ProgramContext, ir.ContractContext, ir.SlotReference"},
		{line: "&pb.Contract" + "Deadline{RuleEvents: 3}", want: []string{"Contract" + "Deadline"}},
		{line: "Type: &pb.Contract" + "CaptureType{}", want: []string{"Contract" + "CaptureType"}},
		{line: "[]*pb.Correlated" + "Binding{}", want: []string{"Correlated" + "Binding"}},
		{line: "(fields : Array CorrelatedEvidence" + "Field)", want: []string{"CorrelatedEvidence" + "Field"}},
		{line: "Program.correlatedEvidence" + "Binding fieldId path", want: []string{"CorrelatedEvidence" + "Binding"}},
		// The bound oneof, the singular type and the named values replace them; the correlated names
		// that share a prefix stay.
		{line: "pb.Deadline_RuleEvents{}, pb.SingularType_Message{}, pb.NamedValue{}, pb.NamedExpression{}, CorrelatedEvidenceRule, CorrelatedEvidenceProjection, CorrelatedRuleBinding"},
		{line: "&pb.Value_Natural" + "Value{Natural" + "Value: text}", want: []string{"Natural" + "Value"}},
		{line: `{"natural` + `Value": "1"}`, want: []string{"Natural" + "Value"}},
		{line: "{ value := some (.natural" + "_value text) }", want: []string{"natural" + "_value"}},
		{line: "pb.SCALAR_KIND_" + "NATURAL", want: []string{"SCALAR_KIND_" + "NATURAL"}},
		{line: "testpilotspb.ENTRYPOINT_" + "KIND_WORKFLOW", want: []string{"ENTRYPOINT_" + "KIND_*"}},
		// The unsigned integer arm and kind, the Go entrypoint classification and the model's natural
		// evidence scalar stay.
		{line: "pb.Value_UnsignedIntegerValue{}, pb.SCALAR_KIND_UINT64, contract.EntrypointKind, testpilot.WorkflowEntrypoint, EvidenceValue.natural"},
		{line: "(kind : NexusResponse" + "Kind)", want: []string{"NexusResponse" + "Kind"}},
		{line: "Program.respond" + "Nexus kind result", want: []string{"Respond" + "Nexus"}},
		{line: "&pb.Instruction_StartNexus" + "Operation{}", want: []string{"Instruction_StartNexus" + "Operation"}},
		{line: "case *pb.Instruction_CompleteNexus" + "Operation:", want: []string{"Instruction_CompleteNexus" + "Operation"}},
		{line: "source.GetStartNexus" + "Operation()", want: []string{"GetStartNexus" + "Operation"}},
		{line: "n.source.Instruction.GetCompleteNexus" + "Operation()", want: []string{"GetCompleteNexus" + "Operation"}},
		// Each rule requires a non-identifier boundary on both sides, so the typed successors and the
		// live WorkflowService method stay; the two HistoryService methods the generated Temporal.API
		// spells are why the bare start and completion names are not held at all.
		{line: "pb.NexusOperationCompletion{}, nexusOperationCompletion, pb.NexusHandlerReply{}, pb.WorkflowCommand{}"},
		{line: "workflowservice.RespondNexusTaskCompletedRequest{}, respondNexusTaskCompleted, historyservice.CompleteNexusOperationRequest{}"},
		{line: "def startNexusOperation : Method StartNexusOperationRequest StartNexusOperationResponse"},
		{line: "HistoryService.CompleteNexusOperation, completeNexusOperation, CompleteNexusOperationChasmRequest"},
		{line: "mo" + "del lifecycle", want: []string{"mo" + "del <name>"}},
		{line: "mo" + "del raceLifecycle", want: []string{"mo" + "del <name>"}},
		// The word stays live: `model/` is the tree, `model:` is a key of `property` and `scenario`,
		// `DeclaredModel` is what they resolve to, and a doc comment may wrap a line onto it.
		{line: "  model: lifecycle"},
		{line: "model: lifecycle"},
		{line: "model/Umpire/Command/Syntax.lean"},
		{line: "(candidate : Umpire.Command.DeclaredModel Setup State Action Outcome Fact)"},
		{line: "model honest: a witness used to refute `skip` must be reachable *under* `skip`."},
		{line: "model payload; the guard above proves every projection this module uses is admitted."},
		{line: "machine lifecycle"},
	} {
		require.Equal(t, tc.want, matched(tc.line), "line %q", tc.line)
	}
}
