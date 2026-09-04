package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestCaseSchemaRoundTripsSourceShapedValues(t *testing.T) {
	t.Parallel()

	input := &umpirespb.Case{
		Version: &umpirespb.FormatVersion{Major: 1},
		CaseId:  "nexus.async-success",
		Metadata: &umpirespb.CaseMetadata{
			ProducerId: "lean.temporal.nexus",
			Definitions: []*umpirespb.CaseDefinitionBinding{
				{DefinitionId: "temporal.nexus.target", BehaviorFingerprint: "target/v1", Kind: umpirespb.CASE_DEFINITION_KIND_TARGET},
				{DefinitionId: "temporal.nexus.provider", BehaviorFingerprint: "provider/v1", Kind: umpirespb.CASE_DEFINITION_KIND_PROVIDER},
				{DefinitionId: "temporal.nexus.law", BehaviorFingerprint: "law/v1", Kind: umpirespb.CASE_DEFINITION_KIND_LAW},
				{DefinitionId: "temporal.nexus.connector", BehaviorFingerprint: "connector/v1", Kind: umpirespb.CASE_DEFINITION_KIND_CONNECTOR},
				{DefinitionId: "temporal.nexus.kernel", BehaviorFingerprint: "kernel/v1", Kind: umpirespb.CASE_DEFINITION_KIND_KERNEL},
			},
			Sources: []*umpirespb.SourceLocation{{
				Path:       "Temporal/Feature/Nexus/Operations.lean",
				Line:       42,
				Column:     7,
				Provenance: "checked-model",
			}},
			KnownGaps: []*umpirespb.CaseKnownGap{
				{Kind: umpirespb.CASE_KNOWN_GAP_KIND_INTERPRETATION, Code: "temporal.nexus.gap"},
				{
					Kind:    umpirespb.CASE_KNOWN_GAP_KIND_CLAIM,
					Code:    "temporal.nexus.gap",
					Subject: &umpirespb.OptionalString{Value: "temporal.nexus.target"},
					Detail:  &umpirespb.OptionalString{Value: "claim remains local"},
				},
			},
		},
		Program: &umpirespb.Program{
			ProgramId: "nexus.async-success.program",
			Roles: []*umpirespb.ProgramRole{
				{RoleId: "frontend", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT},
				{RoleId: "nexus-worker", Kind: umpirespb.SYMBOLIC_ROLE_KIND_WORKER},
			},
			Slots: []*umpirespb.SlotSchema{
				{SlotId: "workflow-id", Type: singularScalarType(umpirespb.SCALAR_KIND_TEXT)},
				{SlotId: "history-events", Type: repeatedMessageType("temporal.api.history.v1.HistoryEvent")},
				{SlotId: "completion-authority", Type: opaqueCapabilityType(), Kind: umpirespb.SLOT_KIND_OPAQUE_CAPABILITY},
			},
			Observations: []*umpirespb.ObservationSchema{
				{
					ObservationId: "history-event-type",
					Type:          singularEnumType("temporal.api.enums.v1.EventType"),
				},
				{ObservationId: "scheduled-event-id", Type: singularScalarType(umpirespb.SCALAR_KIND_NATURAL)},
			},
			Entrypoints: []*umpirespb.Entrypoint{{
				EntrypointId: "controller",
				Context:      umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER,
				Activation: &umpirespb.ActivationBinding{Binding: &umpirespb.ActivationBinding_Controller{
					Controller: &umpirespb.ControllerActivation{},
				}},
				Nodes: []*umpirespb.InstructionNode{{
					InstructionId: "start-workflow",
					Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_InvokeRpc{InvokeRpc: &umpirespb.InvokeRPC{
						EndpointRoleId: "frontend",
						Method:         "temporal.api.workflowservice.v1.WorkflowService.StartWorkflowExecution",
						RequestAssignments: []*umpirespb.RequestAssignment{{
							Target: fieldPath("workflow_id"),
							Value: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{
								Value: &umpirespb.Value_Text{Text: "umpire-run"},
							}}},
						}},
					}}},
				}},
			}},
			Cleanup: &umpirespb.CleanupGraph{
				EntrypointId: "cleanup",
				Context:      umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER,
				Nodes: []*umpirespb.InstructionNode{
					{
						InstructionId: "release",
						Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_AwaitSlot{AwaitSlot: &umpirespb.AwaitSlot{
							SlotId: "completion-authority",
						}}},
					},
					{
						InstructionId: "confirm-release",
						Dependencies: []*umpirespb.InstructionReference{{
							EntrypointId:  "cleanup",
							InstructionId: "release",
						}},
						Guard: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Outcome{Outcome: &umpirespb.InstructionOutcomeReference{
							Instruction: &umpirespb.InstructionReference{EntrypointId: "cleanup", InstructionId: "release"},
							Field:       umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS,
						}}},
						Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_AwaitSlot{AwaitSlot: &umpirespb.AwaitSlot{
							SlotId: "workflow-id",
						}}},
					},
				},
			},
			Limits: &umpirespb.ProgramLimits{MaxEntrypoints: 4, MaxNodes: 32, MaxRunEvents: 256},
		},
		Contract: &umpirespb.Contract{
			ContractId: "nexus.async-success.contract",
			Rules: []*umpirespb.ContractRule{{
				RuleId:       "workflow-completes",
				Kind:         umpirespb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS,
				InitialState: "pending",
				Captures: []*umpirespb.ContractCaptureSchema{{
					CaptureId: "scheduled-event-id",
					Type: &umpirespb.ContractCaptureType{Type: &umpirespb.ContractCaptureType_Scalar{Scalar: &umpirespb.ScalarType{
						Kind: umpirespb.SCALAR_KIND_NATURAL,
					}}},
				}},
				States: []*umpirespb.ContractState{
					{StateId: "pending", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL},
					{StateId: "scheduled", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL},
					{StateId: "satisfied", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_SATISFIED},
					{StateId: "violated", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_VIOLATED},
				},
				Transitions: []*umpirespb.ContractTransition{
					{
						TransitionId: "capture-scheduled-event",
						SourceState:  "pending",
						TargetState:  "scheduled",
						Predicate: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Present{Present: &umpirespb.PresentExpression{
							Operand: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Observation{Observation: &umpirespb.ObservationReference{
								ObservationId: "scheduled-event-id",
							}}},
						}}},
						CaptureAssignments: []*umpirespb.ContractCaptureAssignment{{
							CaptureId: "scheduled-event-id",
							Observation: &umpirespb.ObservationReference{
								ObservationId: "scheduled-event-id",
							},
						}},
					},
					{
						TransitionId: "observe-completion",
						SourceState:  "scheduled",
						TargetState:  "satisfied",
						Predicate: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Equals{Equals: &umpirespb.EqualsExpression{
							Left: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Capture{Capture: &umpirespb.CaptureReference{
								CaptureId: "scheduled-event-id",
							}}},
							Right: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Observation{Observation: &umpirespb.ObservationReference{
								ObservationId: "scheduled-event-id",
							}}},
						}}},
						Support: umpirespb.CONTRACT_SUPPORT_MATCHING_EVENT,
					},
				},
				Horizon: &umpirespb.ContractHorizon{ElapsedMilliseconds: 30_000, ViolationStateId: "violated"},
			}},
			Limits: &umpirespb.ContractLimits{
				MaxRules: 8, MaxStates: 32, MaxTransitions: 64, MaxCaptures: 8, MaxCaptureBytes: 4_096,
			},
		},
	}

	wire, err := proto.Marshal(input)
	require.NoError(t, err)
	var wireOutput umpirespb.Case
	require.NoError(t, proto.Unmarshal(wire, &wireOutput))
	protorequire.ProtoEqual(t, input, &wireOutput)

	jsonValue, err := protojson.Marshal(input)
	require.NoError(t, err)
	var jsonOutput umpirespb.Case
	require.NoError(t, protojson.Unmarshal(jsonValue, &jsonOutput))
	protorequire.ProtoEqual(t, input, &jsonOutput)
}

func TestRunSchemaRoundTripsDiagnosticSupportPresence(t *testing.T) {
	t.Parallel()

	input := &umpirespb.Run{
		RunId:                     "run-1",
		Events:                    []*umpirespb.RunEvent{{Sequence: 7, Kind: umpirespb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, ExecutionIncomplete: true}},
		EvaluationFailureSequence: &umpirespb.RunEventSequence{Value: 7},
		Diagnostics: []*umpirespb.RunDiagnostic{
			{DiagnosticId: "without-support", Kind: umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION},
			{
				DiagnosticId:            "with-support",
				Kind:                    umpirespb.RUN_DIAGNOSTIC_KIND_MONITOR,
				SupportingEventSequence: &umpirespb.RunEventSequence{Value: 7},
			},
		},
	}

	wire, err := proto.Marshal(input)
	require.NoError(t, err)
	var wireOutput umpirespb.Run
	require.NoError(t, proto.Unmarshal(wire, &wireOutput))
	protorequire.ProtoEqual(t, input, &wireOutput)

	jsonValue, err := protojson.Marshal(input)
	require.NoError(t, err)
	var jsonOutput umpirespb.Run
	require.NoError(t, protojson.Unmarshal(jsonValue, &jsonOutput))
	protorequire.ProtoEqual(t, input, &jsonOutput)
}

func TestCaseSchemaProtoJSONRejectsCrossedClosedUnions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		target proto.Message
	}{
		{name: "value kind", input: `{"text":"value","boolValue":true}`, target: new(umpirespb.Value)},
		{
			name: "cardinality",
			input: `{"singular":{"scalar":{"kind":"SCALAR_KIND_TEXT"}},` +
				`"repeated":{"element":{"scalar":{"kind":"SCALAR_KIND_TEXT"}}}}`,
			target: new(umpirespb.ValueType),
		},
		{name: "instruction", input: `{"invokeRpc":{},"awaitSlot":{}}`, target: new(umpirespb.Instruction)},
		{
			name: "capture type",
			input: `{"scalar":{"kind":"SCALAR_KIND_NATURAL"},` +
				`"enumeration":{"protobufType":"temporal.api.enums.v1.EventType"}}`,
			target: new(umpirespb.ContractCaptureType),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := protojson.Unmarshal([]byte(test.input), test.target)
			require.ErrorContains(t, err, "oneof")
		})
	}
}

func TestCaseSchemaExcludesConcreteHostAuthorityFields(t *testing.T) {
	t.Parallel()

	forbidden := map[protoreflect.Name]struct{}{
		"address":      {},
		"api_key":      {},
		"callback_url": {},
		"credential":   {},
		"credentials":  {},
		"endpoint_url": {},
		"headers":      {},
		"token":        {},
	}
	for _, path := range []string{
		"temporal/server/api/umpire/v1/value.proto",
		"temporal/server/api/umpire/v1/program.proto",
		"temporal/server/api/umpire/v1/contract.proto",
		"temporal/server/api/umpire/v1/run.proto",
		"temporal/server/api/umpire/v1/case.proto",
	} {
		file, err := protoregistry.GlobalFiles.FindFileByPath(path)
		require.NoError(t, err)
		messages := file.Messages()
		for messageIndex := range messages.Len() {
			fields := messages.Get(messageIndex).Fields()
			for fieldIndex := range fields.Len() {
				_, found := forbidden[fields.Get(fieldIndex).Name()]
				require.False(t, found, "%s contains concrete Host authority field %q", path, fields.Get(fieldIndex).Name())
			}
		}
	}
}

func singularScalarType(kind umpirespb.ScalarKind) *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{
		Type: &umpirespb.SingularType_Scalar{Scalar: &umpirespb.ScalarType{Kind: kind}},
	}}}
}

func singularEnumType(protobufType string) *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{
		Type: &umpirespb.SingularType_Enumeration{Enumeration: &umpirespb.NamedType{ProtobufType: protobufType}},
	}}}
}

func repeatedMessageType(protobufType string) *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Repeated{Repeated: &umpirespb.RepeatedType{Element: &umpirespb.SingularType{
		Type: &umpirespb.SingularType_Message{Message: &umpirespb.NamedType{ProtobufType: protobufType}},
	}}}}
}

func opaqueCapabilityType() *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{
		Type: &umpirespb.SingularType_OpaqueCapability{OpaqueCapability: &umpirespb.OpaqueCapabilityType{}},
	}}}
}

func fieldPath(fields ...string) *umpirespb.FieldPath {
	path := &umpirespb.FieldPath{Segments: make([]*umpirespb.FieldPathSegment, 0, len(fields))}
	for _, field := range fields {
		path.Segments = append(path.Segments, &umpirespb.FieldPathSegment{Field: field})
	}
	return path
}

func TestActivationReservationSchemaRoundTrip(t *testing.T) {
	input := `{"instructionId":"start","activationReservations":[{"entrypointId":"workflow","count":"3"},{"entrypointId":"handler","count":"2"}]}`
	var node umpirespb.InstructionNode
	require.NoError(t, protojson.Unmarshal([]byte(input), &node))
	wire, err := proto.Marshal(&node)
	require.NoError(t, err)
	var decoded umpirespb.InstructionNode
	require.NoError(t, proto.Unmarshal(wire, &decoded))
	output, err := protojson.Marshal(&decoded)
	require.NoError(t, err)
	require.JSONEq(t, input, string(output))
}
