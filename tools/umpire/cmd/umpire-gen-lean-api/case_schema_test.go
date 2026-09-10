package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestCaseSchemaRoundTripsRefinedValues(t *testing.T) {
	t.Parallel()

	input := &testpilotspb.Case{
		Version:    &testpilotspb.FormatVersion{Major: 1},
		CaseId:     "nexus.async-success",
		Provenance: &testpilotspb.CaseProvenance{ProducerId: "lean.temporal.nexus", ProducerVersion: "1", ProducerData: []byte("definitions")},
		Program: &testpilotspb.Program{
			ProgramId: "nexus.async-success.program",
			Roles:     []*testpilotspb.RoleDefinition{{RoleId: "frontend", Kind: testpilotspb.ROLE_KIND_ENDPOINT}},
			Slots: []*testpilotspb.SlotDefinition{
				{SlotId: "workflow-id", Content: &testpilotspb.SlotDefinition_Value{Value: testpilotSingularScalarType(testpilotspb.SCALAR_KIND_TEXT)}},
				{SlotId: "completion-authority", Content: &testpilotspb.SlotDefinition_OpaqueCapability{OpaqueCapability: &testpilotspb.OpaqueCapabilityType{}}},
			},
			Entrypoints: []*testpilotspb.EntrypointDefinition{{
				EntrypointId: "controller",
				Activation:   &testpilotspb.EntrypointDefinition_Controller{Controller: &testpilotspb.ControllerActivation{}},
			}},
		},
		Contract: &testpilotspb.Contract{ContractId: "nexus.async-success.contract"},
	}

	wire, err := proto.Marshal(input)
	require.NoError(t, err)
	var wireOutput testpilotspb.Case
	require.NoError(t, proto.Unmarshal(wire, &wireOutput))
	protorequire.ProtoEqual(t, input, &wireOutput)

	jsonValue, err := protojson.Marshal(input)
	require.NoError(t, err)
	var jsonOutput testpilotspb.Case
	require.NoError(t, protojson.Unmarshal(jsonValue, &jsonOutput))
	protorequire.ProtoEqual(t, input, &jsonOutput)
}

func TestRunSchemaRoundTripsDiagnosticSupportPresence(t *testing.T) {
	t.Parallel()

	input := &testpilotspb.Run{
		RunId:             "run-1",
		Events:            []*testpilotspb.RunEvent{{Sequence: 7, Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, ExecutionIncomplete: true}},
		EvaluationFailure: &testpilotspb.Run_EvaluationFailureSequence{EvaluationFailureSequence: 7},
		Diagnostics: []*testpilotspb.RunDiagnostic{
			{DiagnosticId: "without-support", Kind: testpilotspb.RUN_DIAGNOSTIC_KIND_EXECUTION},
			{
				DiagnosticId: "with-support",
				Kind:         testpilotspb.RUN_DIAGNOSTIC_KIND_MONITOR,
				Support:      &testpilotspb.RunDiagnostic_SupportingEventSequence{SupportingEventSequence: 7},
			},
		},
	}

	wire, err := proto.Marshal(input)
	require.NoError(t, err)
	var wireOutput testpilotspb.Run
	require.NoError(t, proto.Unmarshal(wire, &wireOutput))
	protorequire.ProtoEqual(t, input, &wireOutput)

	jsonValue, err := protojson.Marshal(input)
	require.NoError(t, err)
	var jsonOutput testpilotspb.Run
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
		{name: "value kind", input: `{"text":"value","boolValue":true}`, target: new(testpilotspb.Value)},
		{
			name: "cardinality",
			input: `{"singular":{"scalar":{"kind":"SCALAR_KIND_TEXT"}},` +
				`"repeated":{"element":{"scalar":{"kind":"SCALAR_KIND_TEXT"}}}}`,
			target: new(testpilotspb.ValueType),
		},
		{name: "instruction", input: `{"invokeRpc":{},"awaitSlot":{}}`, target: new(testpilotspb.Instruction)},
		{
			name: "capture type",
			input: `{"scalar":{"kind":"SCALAR_KIND_NATURAL"},` +
				`"enumeration":{"protobufType":"temporal.api.enums.v1.EventType"}}`,
			target: new(testpilotspb.ContractCaptureType),
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

func testpilotSingularScalarType(kind testpilotspb.ScalarKind) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{
		Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: kind}},
	}}}
}

func TestActivationReservationSchemaRoundTrip(t *testing.T) {
	input := `{"instructionId":"start","activationReservations":[{"entrypointId":"workflow","count":"3"},{"entrypointId":"handler","count":"2"}]}`
	var node testpilotspb.InstructionDefinition
	require.NoError(t, protojson.Unmarshal([]byte(input), &node))
	wire, err := proto.Marshal(&node)
	require.NoError(t, err)
	var decoded testpilotspb.InstructionDefinition
	require.NoError(t, proto.Unmarshal(wire, &decoded))
	output, err := protojson.Marshal(&decoded)
	require.NoError(t, err)
	require.JSONEq(t, input, string(output))
}
