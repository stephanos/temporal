package server

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

const describeMethod = "/example.Describe/Pending"

// pendingFile is a service whose one method answers with the pending entries of an operation,
// each carrying the operation's key and its attempt count: the shape a read declaration polls.
func pendingFile(t *testing.T) protoreflect.FileDescriptor {
	t.Helper()
	optional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name: proto.String("pending.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Request"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("workflow_id"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: optional}}},
			{Name: proto.String("Entry"), Field: []*descriptorpb.FieldDescriptorProto{
				{Name: proto.String("key"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(), Label: optional},
				{Name: proto.String("attempt"), Number: proto.Int32(2), Type: descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(), Label: optional},
			}},
			{Name: proto.String("Response"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("entries"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), TypeName: proto.String(".example.Entry")}}},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Describe"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Pending"), InputType: proto.String(".example.Request"), OutputType: proto.String(".example.Response")}}}},
	}, nil)
	require.NoError(t, err)
	return file
}

// pollFixture is a Program whose one controller instruction polls an operation's attempt count
// through the pending read, under a Profile that authorizes exactly that.
func pollFixture(t *testing.T, address string) (*Driver, *testpilotspb.Case, protoreflect.MethodDescriptor) {
	t.Helper()
	file := pendingFile(t)
	descriptors := &descriptorpb.FileDescriptorSet{}
	seen := map[string]bool{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if seen[file.Path()] {
			return
		}
		seen[file.Path()] = true
		for index := 0; index < file.Imports().Len(); index++ {
			add(file.Imports().Get(index))
		}
		descriptors.File = append(descriptors.File, protodesc.ToFileDescriptorProto(file))
	}
	add(testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	add(file)
	catalog, err := testpilot.NewCatalog(descriptors)
	require.NoError(t, err)
	limits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 4, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 8, MaxRequestBytes: 4096, MaxResponseBytes: 8192, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000, MaxInstructionEmittedEvents: 8, MaxInstructionResponseBytes: 8192}
	contractLimits := &testpilotspb.ContractLimits{MaxRules: 8, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	profile := testpilot.ProfileSpec{Identity: "poll-host", Catalog: catalog, ProgramLimits: limits, ContractLimits: contractLimits, Opcodes: []testpilot.Opcode{testpilot.ReadEvidence}, Roles: []testpilot.RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{describeMethod}}}}
	host, err := New(Options{Profile: profile, Endpoints: map[string]Endpoint{"endpoint": {Target: address, Credentials: insecure.NewCredentials(), Metadata: metadata.Pairs("authorization", "host-secret")}}})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, host.Close(context.Background())) })
	node := &testpilotspb.InstructionNode{InstructionId: "poll", Limits: &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 2000}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}}, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ReadEvidence{ReadEvidence: &testpilotspb.ReadEvidence{
		EvidenceId: "pendingAttempts", EndpointRoleId: "endpoint", PollIntervalMilliseconds: 5,
		Until: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{
			Operator: testpilotspb.COMPARISON_OPERATOR_GREATER_THAN,
			Left:     &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &testpilotspb.ProjectedValueReference{}}}}}, Path: "attempt"}}},
			Right:    &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_SignedIntegerValue{SignedIntegerValue: "1"}}}},
		}}},
	}}}}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "poll", Program: &testpilotspb.Program{
		ProgramId:    "program",
		Roles:        []*testpilotspb.Role{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}},
		Observations: []*testpilotspb.Observation{{ObservationId: "evidence", Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.CorrelatedEvidence"}}}}}}},
		Evidence: []*testpilotspb.EvidenceDeclaration{{
			EvidenceId: "pendingAttempts", EvidenceSource: "describe", Operation: "key",
			Source: &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: describeMethod, Path: "entries"}},
			Scope:  []*testpilotspb.NamedValue{{FieldId: "run", Value: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "one"}}}},
			Fields: []*testpilotspb.EvidenceFieldDeclaration{{FieldId: "attempts", Path: "attempt"}},
		}},
		Entrypoints: []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: []*testpilotspb.InstructionNode{node}}},
		Cleanup:     &testpilotspb.Cleanup{EntrypointId: "cleanup"},
	}, Contract: &testpilotspb.Contract{ContractId: "contract", Rules: []*testpilotspb.ContractRule{{RuleId: "safety", Kind: testpilotspb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "start", States: []*testpilotspb.ContractState{{StateId: "start", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING}, {StateId: "good", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED}}, Transitions: []*testpilotspb.ContractTransition{{TransitionId: "complete", SourceStateId: "start", TargetStateId: "good", Predicate: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}}, EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}}, SupportKind: testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT}}}}}}
	_, err = testpilot.Prepare(source, host)
	require.NoError(t, err)
	return host, source, file.Services().Get(0).Methods().Get(0)
}

// startDescribe serves the pending read with one entry whose attempt count is the number of calls
// so far, so the second poll is the first the fixture's condition accepts.
func startDescribe(t *testing.T, file protoreflect.FileDescriptor, calls *atomic.Int32) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	method := file.Services().Get(0).Methods().Get(0)
	server.RegisterService(&grpc.ServiceDesc{ServiceName: "example.Describe", HandlerType: (*interface{})(nil), Methods: []grpc.MethodDesc{{MethodName: "Pending", Handler: func(_ any, _ context.Context, decode func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
		input := dynamicpb.NewMessage(method.Input())
		if err := decode(input); err != nil {
			return nil, err
		}
		response := dynamicpb.NewMessage(method.Output())
		entries := response.Mutable(method.Output().Fields().ByName("entries")).List()
		entry := entries.AppendMutable().Message()
		entry.Set(entry.Descriptor().Fields().ByName("key"), protoreflect.ValueOfInt64(5))
		entry.Set(entry.Descriptor().Fields().ByName("attempt"), protoreflect.ValueOfInt32(calls.Add(1)))
		return response, nil
	}}}}, struct{}{})
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); require.NoError(t, <-done) })
	return listener.Addr().String()
}

func attemptOf(t *testing.T, response proto.Message) int32 {
	t.Helper()
	entries := response.ProtoReflect().Get(response.ProtoReflect().Descriptor().Fields().ByName("entries")).List()
	require.Equal(t, 1, entries.Len())
	entry := entries.Get(0).Message()
	return int32(entry.Get(entry.Descriptor().Fields().ByName("attempt")).Int())
}

// One poll is one effect: the Session repeats the declaration's RPC until the runtime's predicate
// accepts a response, and the instruction's attempt and identity are counted once.
func TestPollRPCRepeatsTheReadUntilSatisfied(t *testing.T) {
	var calls atomic.Int32
	h, source, method := pollFixture(t, startDescribe(t, pendingFile(t), &calls))
	s, err := h.open(t.Context(), "run", source.Program, h.profile.ProgramLimits)
	require.NoError(t, err)
	request := dynamicpb.NewMessage(method.Input())
	var seen []int32
	handle, err := s.PollRPC(t.Context(), coordinate("run", "poll"), "endpoint", method, request, 5*time.Millisecond, func(_ context.Context, response proto.Message) (bool, error) {
		attempt := attemptOf(t, response)
		seen = append(seen, attempt)
		return attempt > 1, nil
	})
	require.NoError(t, err)
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, []int32{1, 2}, seen)
	require.EqualValues(t, 2, calls.Load())
	require.True(t, proto.Equal(&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}, result.Outcome))
	require.EqualValues(t, 2, attemptOf(t, result.Response))
	// The instruction started once, so its coordinate cannot start again.
	_, err = s.PollRPC(t.Context(), coordinate("run", "poll"), "endpoint", method, request, 5*time.Millisecond, func(context.Context, proto.Message) (bool, error) { return true, nil })
	require.ErrorIs(t, err, errInvalid)
	require.NoError(t, s.Close(t.Context()))
}

// A poll the Case did not declare, or that names another role, method or interval than the
// declaration admits, is refused before any call.
func TestPollRPCRefusesWhatTheDeclarationDoesNotAdmit(t *testing.T) {
	var calls atomic.Int32
	h, source, method := pollFixture(t, startDescribe(t, pendingFile(t), &calls))
	s, err := h.open(t.Context(), "run", source.Program, h.profile.ProgramLimits)
	require.NoError(t, err)
	request := dynamicpb.NewMessage(method.Input())
	accept := func(context.Context, proto.Message) (bool, error) { return true, nil }
	for _, test := range []struct {
		name       string
		coordinate testpilot.Coordinate
		role       string
		interval   time.Duration
		predicate  testpilot.PollPredicate
	}{
		{"unknown instruction", coordinate("run", "missing"), "endpoint", time.Millisecond, accept},
		{"wrong role", coordinate("run", "poll"), "missing", time.Millisecond, accept},
		{"no interval", coordinate("run", "poll"), "endpoint", 0, accept},
		{"no predicate", coordinate("run", "poll"), "endpoint", time.Millisecond, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			handle, err := s.PollRPC(t.Context(), test.coordinate, test.role, method, request, test.interval, test.predicate)
			require.Error(t, err)
			require.Nil(t, handle)
		})
	}
	require.Zero(t, calls.Load())
	// A poll that times out before its condition holds reports the timeout as its outcome.
	never := func(context.Context, proto.Message) (bool, error) { return false, nil }
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Millisecond)
	defer cancel()
	handle, err := s.PollRPC(ctx, coordinate("run", "poll"), "endpoint", method, request, 5*time.Millisecond, never)
	require.NoError(t, err)
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT, result.Outcome.Status)
	require.Positive(t, calls.Load())
	require.NoError(t, s.Close(t.Context()))
}
