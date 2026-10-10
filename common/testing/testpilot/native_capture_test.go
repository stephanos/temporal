package testpilot_test

import (
	"context"
	"encoding/json"
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/casefile"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	pbduration "go.temporal.io/server/common/testing/testpilot/duration"
	"go.temporal.io/server/common/testing/testpilot/evaluation"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestNativeCELCapturedDescriptorRoundTrip(t *testing.T) {
	source, profile := proofFixture(t)
	profile.ContractLimits.MaxWorkPerEvent = 4000000
	descriptors := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("native-capture.proto"), Package: proto.String("nativecapture"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("text"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}}}}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Source"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Read"), InputType: proto.String(".temporal.server.api.testpilot.v1.InstructionOutcome"), OutputType: proto.String(".temporal.server.api.testpilot.v1.InstructionOutcome")}}}}}}}
	var err error
	descriptors.File[0].Dependency = []string{"temporal/server/api/testpilot/v1/run.proto"}
	descriptors.File = append(descriptors.File, testsupport.DescriptorClosure(testpilotspb.File_temporal_server_api_testpilot_v1_run_proto).File...)
	profile.Catalog, err = testpilot.NewCatalog(descriptors)
	require.NoError(t, err)
	profile.Roles = []testpilot.RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{"/nativecapture.Source/Read"}}}
	profile.Opcodes = []testpilot.Opcode{testpilot.InvokeRPC}
	source.Program.Roles = []*testpilotspb.Role{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}}
	message := &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcome"}}}
	source.Program.Observations = []*testpilotspb.Observation{{ObservationId: "record", Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: message}}}}
	for _, id := range []string{"first", "second"} {
		source.Program.Entrypoints[0].Instructions = append(source.Program.Entrypoints[0].Instructions, &testpilotspb.InstructionNode{InstructionId: id, Limits: &testpilotspb.InstructionLimits{Timeout: pbduration.FromMilliseconds(1000), MaxAttempts: proto.Int64(1)}, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: "/nativecapture.Source/Read", ResponseReads: []*testpilotspb.ResponseRead{{Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_ObservationId{ObservationId: "record"}}}}}}}}})
	}
	current := cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_ObservationId{ObservationId: "record"}})
	saved := cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_CaptureId{CaptureId: "saved"}})
	left, right := cel.Path(current, "detail"), cel.Path(saved, "detail")
	rule := source.Contract.Rules[0]
	rule.Captures = []*testpilotspb.ContractCapture{{CaptureId: "saved", Type: message}}
	rule.States = []*testpilotspb.ContractState{{StateId: "start", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING}, {StateId: "saved", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING}, {StateId: "good", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED}}
	rule.Transitions = []*testpilotspb.ContractTransition{
		{TransitionId: "capture", SourceStateId: "start", TargetStateId: "saved", EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}}, Predicate: cel.Present(current), CaptureAssignments: []*testpilotspb.ContractCaptureAssignment{{CaptureId: "saved", ObservationId: "record"}}, SupportsEvent: proto.Bool(true)},
		{TransitionId: "compare", SourceStateId: "saved", TargetStateId: "good", EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}}, Predicate: cel.All(cel.Present(left), cel.Present(right), cel.Compare("_==_", left, right)), SupportsEvent: proto.Bool(true)},
	}
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	driver := &facadetest.Driver{DriverIdentity: prepared.Identity(), OnOpen: func(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
		return &testsupport.Session{OnInvokeRPC: func(_ context.Context, _ testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message) (testpilot.EffectHandle, error) {
			response := dynamicpb.NewMessage(method.Output())
			response.Set(method.Output().Fields().ByName("detail"), protoreflect.ValueOfString("owned"))
			return testsupport.Completed(testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}), nil
		}}, nil
	}}
	run, online, err := prepared.Run(t.Context(), driver)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, online.Status, run.String())
	var values []*celpb.Value
	for _, event := range run.Events {
		for _, observation := range event.Observations {
			values = append(values, observation.Value)
		}
	}
	require.Len(t, values, 2)
	require.Equal(t, "type.googleapis.com/temporal.server.api.testpilot.v1.InstructionOutcome", values[0].GetObjectValue().TypeUrl)
	caseJSON, err := protojson.Marshal(source)
	require.NoError(t, err)
	canonical, err := casefile.Compact(caseJSON)
	require.NoError(t, err)
	caseIdentity, err := recordedrun.CaseIdentity(canonical)
	require.NoError(t, err)
	encoded, err := recordedrun.Encode(caseIdentity, prepared.Identity(), run)
	require.NoError(t, err)
	record, err := recordedrun.Decode(encoded)
	require.NoError(t, err)
	offline, _, err := prepared.Evaluate(t.Context(), record.Run)
	require.NoError(t, err)
	require.True(t, proto.Equal(online, offline))
	subject, err := evaluation.Admit(canonical, encoded, prepared.Identity().Catalog)
	require.NoError(t, err)
	require.Equal(t, caseIdentity, subject.CaseIdentity)
	require.Equal(t, source.CaseId, subject.CaseID)
	require.True(t, proto.Equal(online, subject.Verdict))
	for _, control := range []struct {
		name, companion, reason string
	}{
		{"missing-companion", "", evaluation.ReasonIncompatible},
		{"crossed-companion-before-payload", recordedrun.Digest([]byte("another current-format Case")), evaluation.ReasonCrossed},
	} {
		t.Run(control.name, func(t *testing.T) {
			fields := map[string]any{"run": json.RawMessage(`{"retiredPayload":true}`)}
			if control.companion != "" {
				fields["case"] = control.companion
			}
			invalid, err := json.Marshal(fields)
			require.NoError(t, err)
			_, err = evaluation.Admit(canonical, invalid, prepared.Identity().Catalog)
			rejection, ok := evaluation.IsRejection(err)
			require.True(t, ok, "%v", err)
			require.Equal(t, control.reason, rejection.Reason)
		})
	}
}
