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
			Sources: []*umpirespb.SourceLocation{{
				Path:       "Temporal/Feature/Nexus/Operations.lean",
				Line:       42,
				Column:     7,
				Provenance: "checked-model",
			}},
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
			Observations: []*umpirespb.ObservationSchema{{
				ObservationId: "history-event-type",
				Type:          singularEnumType("temporal.api.enums.v1.EventType"),
			}},
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
			Cleanup: &umpirespb.CleanupGraph{Context: umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER},
			Limits:  &umpirespb.ProgramLimits{MaxEntrypoints: 4, MaxNodes: 32, MaxRunEvents: 256},
		},
		Contract: &umpirespb.Contract{
			ContractId: "nexus.async-success.contract",
			Rules: []*umpirespb.ContractRule{{
				RuleId:       "workflow-completes",
				Kind:         umpirespb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS,
				InitialState: "pending",
				States: []*umpirespb.ContractState{
					{StateId: "pending"},
					{StateId: "satisfied", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_SATISFIED},
					{StateId: "violated", Terminal: umpirespb.CONTRACT_TERMINAL_STATE_VIOLATED},
				},
				Transitions: []*umpirespb.ContractTransition{{
					TransitionId: "observe-completion",
					SourceState:  "pending",
					TargetState:  "satisfied",
					Predicate: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Present{Present: &umpirespb.PresentExpression{
						Operand: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Observation{Observation: &umpirespb.ObservationReference{
							ObservationId: "history-event-type",
						}}},
					}}},
					Support: umpirespb.CONTRACT_SUPPORT_MATCHING_EVENT,
				}},
				Horizon: &umpirespb.ContractHorizon{ElapsedMilliseconds: 30_000, ViolationStateId: "violated"},
			}},
			Limits: &umpirespb.ContractLimits{MaxRules: 8, MaxStates: 32, MaxTransitions: 64},
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
