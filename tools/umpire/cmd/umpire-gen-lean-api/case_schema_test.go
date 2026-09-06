package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestCaseSchemaRoundTripsRefinedValues(t *testing.T) {
	t.Parallel()

	input := &testpilotpb.Case{
		Version:    &testpilotpb.FormatVersion{Major: 1},
		CaseId:     "nexus.async-success",
		Provenance: &testpilotpb.CaseProvenance{ProducerId: "lean.temporal.nexus", ProducerVersion: "1", ProducerData: []byte("definitions")},
		Program: &testpilotpb.Program{
			ProgramId: "nexus.async-success.program",
			Roles:     []*testpilotpb.RoleDefinition{{RoleId: "frontend", Kind: testpilotpb.ROLE_KIND_ENDPOINT}},
			Slots: []*testpilotpb.SlotDefinition{
				{SlotId: "workflow-id", Content: &testpilotpb.SlotDefinition_Value{Value: testpilotSingularScalarType(testpilotpb.SCALAR_KIND_TEXT)}},
				{SlotId: "completion-authority", Content: &testpilotpb.SlotDefinition_OpaqueCapability{OpaqueCapability: &testpilotpb.OpaqueCapabilityType{}}},
			},
			Entrypoints: []*testpilotpb.EntrypointDefinition{{
				EntrypointId: "controller",
				Activation:   &testpilotpb.EntrypointDefinition_Controller{Controller: &testpilotpb.ControllerActivation{}},
			}},
		},
		Contract: &testpilotpb.Contract{ContractId: "nexus.async-success.contract"},
	}

	wire, err := proto.Marshal(input)
	require.NoError(t, err)
	var wireOutput testpilotpb.Case
	require.NoError(t, proto.Unmarshal(wire, &wireOutput))
	protorequire.ProtoEqual(t, input, &wireOutput)

	jsonValue, err := protojson.Marshal(input)
	require.NoError(t, err)
	var jsonOutput testpilotpb.Case
	require.NoError(t, protojson.Unmarshal(jsonValue, &jsonOutput))
	protorequire.ProtoEqual(t, input, &jsonOutput)
}

func TestRunSchemaRoundTripsDiagnosticSupportPresence(t *testing.T) {
	t.Parallel()

	input := &testpilotpb.Run{
		RunId:             "run-1",
		Events:            []*testpilotpb.RunEvent{{Sequence: 7, Kind: testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, ExecutionIncomplete: true}},
		EvaluationFailure: &testpilotpb.Run_EvaluationFailureSequence{EvaluationFailureSequence: 7},
		Diagnostics: []*testpilotpb.RunDiagnostic{
			{DiagnosticId: "without-support", Kind: testpilotpb.RUN_DIAGNOSTIC_KIND_EXECUTION},
			{
				DiagnosticId: "with-support",
				Kind:         testpilotpb.RUN_DIAGNOSTIC_KIND_MONITOR,
				Support:      &testpilotpb.RunDiagnostic_SupportingEventSequence{SupportingEventSequence: 7},
			},
		},
	}

	wire, err := proto.Marshal(input)
	require.NoError(t, err)
	var wireOutput testpilotpb.Run
	require.NoError(t, proto.Unmarshal(wire, &wireOutput))
	protorequire.ProtoEqual(t, input, &wireOutput)

	jsonValue, err := protojson.Marshal(input)
	require.NoError(t, err)
	var jsonOutput testpilotpb.Run
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
		{name: "value kind", input: `{"text":"value","boolValue":true}`, target: new(testpilotpb.Value)},
		{
			name: "cardinality",
			input: `{"singular":{"scalar":{"kind":"SCALAR_KIND_TEXT"}},` +
				`"repeated":{"element":{"scalar":{"kind":"SCALAR_KIND_TEXT"}}}}`,
			target: new(testpilotpb.ValueType),
		},
		{name: "instruction", input: `{"invokeRpc":{},"awaitSlot":{}}`, target: new(testpilotpb.Instruction)},
		{
			name: "capture type",
			input: `{"scalar":{"kind":"SCALAR_KIND_NATURAL"},` +
				`"enumeration":{"protobufType":"temporal.api.enums.v1.EventType"}}`,
			target: new(testpilotpb.ContractCaptureType),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := protojson.Unmarshal([]byte(test.input), test.target)
			require.ErrorContains(t, err, "oneof")
		})
	}
}

func TestCaseSchemaExcludesConcreteDriverAuthorityFields(t *testing.T) {
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
		"temporal/server/api/testpilot/v1/value.proto",
		"temporal/server/api/testpilot/v1/expression.proto",
		"temporal/server/api/testpilot/v1/instruction.proto",
		"temporal/server/api/testpilot/v1/program.proto",
		"temporal/server/api/testpilot/v1/outcome.proto",
		"temporal/server/api/testpilot/v1/contract.proto",
		"temporal/server/api/testpilot/v1/run.proto",
		"temporal/server/api/testpilot/v1/case.proto",
	} {
		file, err := protoregistry.GlobalFiles.FindFileByPath(path)
		require.NoError(t, err)
		messages := file.Messages()
		for messageIndex := range messages.Len() {
			fields := messages.Get(messageIndex).Fields()
			for fieldIndex := range fields.Len() {
				_, found := forbidden[fields.Get(fieldIndex).Name()]
				require.False(t, found, "%s contains concrete Driver authority field %q", path, fields.Get(fieldIndex).Name())
			}
		}
	}
}

func testpilotSingularScalarType(kind testpilotpb.ScalarKind) *testpilotpb.ValueType {
	return &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{
		Type: &testpilotpb.SingularType_Scalar{Scalar: &testpilotpb.ScalarType{Kind: kind}},
	}}}
}

func TestActivationReservationSchemaRoundTrip(t *testing.T) {
	input := `{"instructionId":"start","activationReservations":[{"entrypointId":"workflow","count":"3"},{"entrypointId":"handler","count":"2"}]}`
	var node testpilotpb.InstructionDefinition
	require.NoError(t, protojson.Unmarshal([]byte(input), &node))
	wire, err := proto.Marshal(&node)
	require.NoError(t, err)
	var decoded testpilotpb.InstructionDefinition
	require.NoError(t, proto.Unmarshal(wire, &decoded))
	output, err := protojson.Marshal(&decoded)
	require.NoError(t, err)
	require.JSONEq(t, input, string(output))
}
