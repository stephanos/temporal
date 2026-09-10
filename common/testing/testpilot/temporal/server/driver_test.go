package server

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func fixture(t *testing.T, address string) (*Driver, *testpilotspb.Case, []protoreflect.MethodDescriptor) {
	t.Helper()
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{Name: proto.String("echo.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), Dependency: []string{"google/protobuf/wrappers.proto"}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Echo"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Length"), InputType: proto.String(".google.protobuf.StringValue"), OutputType: proto.String(".google.protobuf.Int64Value")}}}}}, protoregistry.GlobalFiles)
	require.NoError(t, err)
	catalog, err := testpilot.NewCatalog(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{protodesc.ToFileDescriptorProto(file), protodesc.ToFileDescriptorProto(wrapperspb.File_google_protobuf_wrappers_proto), protodesc.ToFileDescriptorProto(healthpb.File_grpc_health_v1_health_proto)}})
	require.NoError(t, err)
	limits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	contractLimits := &testpilotspb.ContractLimits{MaxRules: 8, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	profile := testpilot.ProfileSpec{Identity: "test-host", Catalog: catalog, ProgramLimits: limits, ContractLimits: contractLimits, Opcodes: []testpilot.Opcode{testpilot.InvokeRPC, testpilot.AwaitSlot}, Roles: []testpilot.RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{"/grpc.health.v1.Health/Check", "/example.Echo/Length"}}}}
	host, err := New(Options{Profile: profile, Endpoints: map[string]Endpoint{"endpoint": {Target: address, Credentials: insecure.NewCredentials(), Metadata: metadata.Pairs("authorization", "host-secret")}}})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, host.Close(context.Background())) })
	nodes := []*testpilotspb.InstructionDefinition{rpcNode("check", "/grpc.health.v1.Health/Check"), rpcNode("length", "/example.Echo/Length")}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "case", Program: &testpilotspb.Program{ProgramId: "program", Roles: []*testpilotspb.RoleDefinition{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}}, Limits: proto.CloneOf(limits), Entrypoints: []*testpilotspb.EntrypointDefinition{{EntrypointId: "controller", Activation: &testpilotspb.EntrypointDefinition_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: nodes}}, Cleanup: &testpilotspb.CleanupDefinition{EntrypointId: "cleanup"}}, Contract: &testpilotspb.Contract{ContractId: "contract", Limits: contractLimits, Rules: []*testpilotspb.ContractRuleDefinition{{RuleId: "safety", Kind: testpilotspb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "start", States: []*testpilotspb.ContractStateDefinition{{StateId: "start", Status: testpilotspb.CONTRACT_STATE_STATUS_NONTERMINAL}, {StateId: "good", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED}}, Transitions: []*testpilotspb.ContractTransitionDefinition{{TransitionId: "complete", SourceStateId: "start", TargetStateId: "good", Predicate: &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}}, EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}}, SupportKind: testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT}}}}}}
	_, err = testpilot.Prepare(source, host)
	require.NoError(t, err)
	return host, source, []protoreflect.MethodDescriptor{healthpb.File_grpc_health_v1_health_proto.Services().ByName("Health").Methods().ByName("Check"), file.Services().Get(0).Methods().Get(0)}
}
func rpcNode(id, method string) *testpilotspb.InstructionDefinition {
	return &testpilotspb.InstructionDefinition{InstructionId: id, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRPC{EndpointRoleId: "endpoint", Method: method}}}, Outcome: &testpilotspb.InstructionOutcomeDefinition{Fields: []*testpilotspb.OutcomeFieldDefinition{{Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: "temporal.server.api.testpilot.v1.InstructionOutcomeStatus"}}}}}}}}, Limits: &testpilotspb.InstructionLimits{TimeoutMilliseconds: 2000, MaxAttempts: 1, MaxEmittedEvents: 4, MaxResponseBytes: 4096}}
}
func coordinate(run, node string) testpilot.Coordinate {
	return testpilot.Coordinate{RunID: run, EntrypointID: "controller", ActivationID: "controller", InstructionID: node, Attempt: 1}
}
func request(method protoreflect.MethodDescriptor, value string) proto.Message {
	m := dynamicpb.NewMessage(method.Input())
	m.Set(method.Input().Fields().Get(0), protoreflect.ValueOfString(value))
	return m
}
func startGRPC(t *testing.T, interceptor grpc.UnaryServerInterceptor) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	healthpb.RegisterHealthServer(server, health.NewServer())
	server.RegisterService(&grpc.ServiceDesc{ServiceName: "example.Echo", HandlerType: (*interface{})(nil), Methods: []grpc.MethodDesc{{MethodName: "Length", Handler: func(srv any, ctx context.Context, decode func(any) error, intercept grpc.UnaryServerInterceptor) (any, error) {
		input := &wrapperspb.StringValue{}
		if err := decode(input); err != nil {
			return nil, err
		}
		handler := func(context.Context, any) (any, error) { return wrapperspb.Int64(int64(len(input.Value))), nil }
		if intercept == nil {
			return handler(ctx, input)
		}
		return intercept(ctx, input, &grpc.UnaryServerInfo{Server: srv, FullMethod: "/example.Echo/Length"}, handler)
	}}}}, struct{}{})
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); require.NoError(t, <-done) })
	return listener.Addr().String()
}
func TestUnaryTransportAndResponseOwnership(t *testing.T) {
	received := make(chan metadata.MD, 2)
	address := startGRPC(t, func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		received <- md
		return handler(ctx, req)
	})
	h, source, methods := fixture(t, address)
	s, err := h.open(t.Context(), "run", source.Program)
	require.NoError(t, err)
	expected := []proto.Message{&healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, wrapperspb.Int64(5)}
	for i, node := range []string{"check", "length"} {
		value := ""
		if i == 1 {
			value = "hello"
		}
		req := request(methods[i], value)
		handle, err := s.InvokeRPC(t.Context(), coordinate("run", node), "endpoint", methods[i], req)
		require.NoError(t, err)
		req.ProtoReflect().Set(methods[i].Input().Fields().Get(0), protoreflect.ValueOfString("changed"))
		result, err := handle.Wait(t.Context())
		require.NoError(t, err)
		require.True(t, proto.Equal(expected[i], result.Response))
		require.True(t, proto.Equal(&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}, result.Outcome))
		result.Response.ProtoReflect().Clear(methods[i].Output().Fields().Get(0))
		result.Outcome.Detail = "changed"
		again, err := handle.Wait(t.Context())
		require.NoError(t, err)
		require.True(t, proto.Equal(expected[i], again.Response))
		require.Empty(t, again.Outcome.Detail)
		require.Equal(t, []string{"host-secret"}, (<-received).Get("authorization"))
		require.NotContains(t, fmt.Sprint(again), "host-secret")
	}
	snapshot := h.Snapshot()
	snapshot.Roles[0].Methods[0] = "changed"
	snapshot.ProgramLimits.MaxAttempts = 1
	require.True(t, proto.Equal(source.Program.Limits, h.Snapshot().ProgramLimits))
	require.NoError(t, s.Close(t.Context()))
}
func TestPreparationAndRuntimeRejection(t *testing.T) {
	h, source, methods := fixture(t, "127.0.0.1:1")
	for _, method := range []string{"/missing.Service/Call", "/grpc.health.v1.Health/Watch"} {
		candidate := proto.CloneOf(source)
		candidate.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().Method = method
		_, err := testpilot.Prepare(candidate, h)
		require.Error(t, err)
	}
	profile := h.Snapshot()
	profile.Roles[0].Methods = profile.Roles[0].Methods[1:]
	_, err := testpilot.Prepare(source, profile)
	require.Error(t, err)
	s, err := h.open(t.Context(), "run", source.Program)
	require.NoError(t, err)
	for _, test := range []struct {
		name       string
		coordinate testpilot.Coordinate
		role       string
		method     protoreflect.MethodDescriptor
		message    proto.Message
	}{
		{"wrong run", coordinate("foreign", "check"), "endpoint", methods[0], request(methods[0], "")},
		{"wrong role", coordinate("run", "check"), "missing", methods[0], request(methods[0], "")},
		{"wrong descriptor", coordinate("run", "check"), "endpoint", methods[0], wrapperspb.String("")},
		{"oversized", coordinate("run", "check"), "endpoint", methods[0], request(methods[0], strings.Repeat("x", 4096))},
		{"stream", coordinate("run", "check"), "endpoint", methods[0].Parent().(protoreflect.ServiceDescriptor).Methods().ByName("Watch"), request(methods[0], "")},
	} {
		t.Run(test.name, func(t *testing.T) {
			handle, err := s.InvokeRPC(t.Context(), test.coordinate, test.role, test.method, test.message)
			require.Error(t, err)
			require.Nil(t, handle)
			require.NotContains(t, err.Error(), "host-secret")
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	handle, err := s.InvokeRPC(ctx, coordinate("run", "check"), "endpoint", methods[0], request(methods[0], ""))
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, handle)
	handle, err = s.InvokeRPC(t.Context(), coordinate("run", "check"), "endpoint", methods[0], request(methods[0], ""))
	require.NoError(t, err)
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS, result.Outcome.Status)
	require.Equal(t, "unavailable", result.Outcome.ProtocolCode)
	require.Empty(t, result.Outcome.Detail)
}
func TestProtocolFailureTimeoutCancellationAndResponseLimit(t *testing.T) {
	for _, kind := range []string{"non-ok", "timeout", "cancel", "oversized"} {
		t.Run(kind, func(t *testing.T) {
			entered := make(chan struct{})
			address := startGRPC(t, func(ctx context.Context, _ any, _ *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
				close(entered)
				switch kind {
				case "non-ok":
					return nil, status.Error(codes.PermissionDenied, "host-secret")
				case "oversized":
					return wrapperspb.String(strings.Repeat("x", 5000)), nil
				default:
					<-ctx.Done()
					return nil, ctx.Err()
				}
			})
			h, source, methods := fixture(t, address)
			if kind == "timeout" {
				source.Program.Entrypoints[0].Instructions[0].Limits.TimeoutMilliseconds = 20
			}
			s, err := h.open(t.Context(), "run", source.Program)
			require.NoError(t, err)
			handle, err := s.InvokeRPC(t.Context(), coordinate("run", "check"), "endpoint", methods[0], request(methods[0], ""))
			require.NoError(t, err)
			if kind == "cancel" {
				<-entered
				require.NoError(t, handle.Cancel(t.Context()))
			}
			result, err := handle.Wait(t.Context())
			require.NoError(t, err)
			want := testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS
			if kind == "cancel" {
				want = testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED
			}
			if kind == "timeout" {
				want = testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT
			}
			require.Equal(t, want, result.Outcome.Status)
			require.Nil(t, result.Response)
			require.NotContains(t, fmt.Sprint(result), "host-secret")
			require.NoError(t, handle.Drain(t.Context()))
		})
	}
}
func TestParallelSessionsAndQuarantineCapacity(t *testing.T) {
	address := startGRPC(t, nil)
	h, source, methods := fixture(t, address)
	t.Run("parallel", func(t *testing.T) {
		for i := 0; i < 8; i++ {
			t.Run(fmt.Sprintf("run%d", i), func(t *testing.T) {
				t.Parallel()
				run := fmt.Sprintf("run%d", i)
				s, err := h.open(t.Context(), run, source.Program)
				require.NoError(t, err)
				handle, err := s.InvokeRPC(t.Context(), coordinate(run, "length"), "endpoint", methods[1], request(methods[1], run))
				require.NoError(t, err)
				result, err := handle.Wait(t.Context())
				require.NoError(t, err)
				require.True(t, proto.Equal(wrapperspb.Int64(4), result.Response))
				require.NoError(t, s.Close(t.Context()))
			})
		}
	})
	h.profile.ProgramLimits.MaxAttempts = 1
	s, err := h.open(t.Context(), "stuck", source.Program)
	require.NoError(t, err)
	released := make(chan struct{})
	e, err := s.start(t.Context(), coordinate("stuck", "check"), source.Program.Entrypoints[0].Instructions[0].Limits, func(context.Context) testpilot.EffectResult {
		<-released
		return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	require.ErrorIs(t, e.Drain(ctx), context.DeadlineExceeded)
	require.NoError(t, s.Quarantine(t.Context(), e))
	require.NoError(t, s.Close(t.Context()))
	other, err := h.open(t.Context(), "other", source.Program)
	require.NoError(t, err)
	denied, err := other.InvokeRPC(t.Context(), coordinate("other", "check"), "endpoint", methods[0], request(methods[0], ""))
	require.ErrorIs(t, err, errCapacity)
	require.Nil(t, denied)
	require.Error(t, other.Quarantine(t.Context(), e))
	close(released)
	require.NoError(t, e.Drain(t.Context()))
	handle, err := other.InvokeRPC(t.Context(), coordinate("other", "check"), "endpoint", methods[0], request(methods[0], ""))
	require.NoError(t, err)
	require.NoError(t, handle.Drain(t.Context()))
}

func TestDriverValidateRejectsEmptyPreparedProgram(t *testing.T) {
	err := (&Driver{}).Validate(t.Context(), testpilot.PreparedProgram{})
	require.ErrorIs(t, err, errInvalid)
}

var _ testpilot.Driver = (*Driver)(nil)
var _ testpilot.Profile = (*Driver)(nil)
var _ testpilot.Session = (*Session)(nil)

func TestSessionAndEffectIdentityCollisions(t *testing.T) {
	address := startGRPC(t, nil)
	h, source, methods := fixture(t, address)
	s, err := h.open(t.Context(), "run", source.Program)
	require.NoError(t, err)
	_, err = h.open(t.Context(), "run", source.Program)
	require.Error(t, err)
	handle, err := s.InvokeRPC(t.Context(), coordinate("run", "check"), "endpoint", methods[0], request(methods[0], ""))
	require.NoError(t, err)
	require.NoError(t, handle.Drain(t.Context()))
	duplicate, err := s.InvokeRPC(t.Context(), coordinate("run", "check"), "endpoint", methods[0], request(methods[0], ""))
	require.Error(t, err)
	require.Nil(t, duplicate)
	_, err = s.Reserve(t.Context(), testpilot.ReservationRequest{})
	require.Error(t, err)
	// A fault is a worker-lifecycle outage; the server Driver owns no worker and refuses it the
	// way it refuses reservations.
	_, err = s.InjectFault(t.Context(), coordinate("run", "check"), "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.Error(t, err)
	require.NoError(t, s.Close(t.Context()))
	denied, err := s.InvokeRPC(t.Context(), coordinate("run", "length"), "endpoint", methods[1], request(methods[1], "x"))
	require.ErrorIs(t, err, errClosed)
	require.Nil(t, denied)
	require.NoError(t, h.Close(t.Context()))
	_, err = h.open(t.Context(), "later", source.Program)
	require.ErrorIs(t, err, errClosed)
}
