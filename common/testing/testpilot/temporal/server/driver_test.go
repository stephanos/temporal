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
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
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
	catalog, err := testpilot.NewCatalog(testsupport.DescriptorClosure(file, healthpb.File_grpc_health_v1_health_proto))
	require.NoError(t, err)
	contractLimits := &testpilotspb.ContractLimits{MaxRules: 8, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 16, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 8, MaxCaptureBytes: 65536}
	profile := testpilot.ProfileSpec{Identity: "test-host", Catalog: catalog, ProgramLimits: testsupport.ProgramLimits(), ContractLimits: contractLimits, Opcodes: []testpilot.Opcode{testpilot.InvokeRPC, testpilot.AwaitSlot}, Roles: []testpilot.RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{"/grpc.health.v1.Health/Check", "/example.Echo/Length"}}, {ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, {ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE}}, EnvironmentBindings: []testpilot.EnvironmentBinding{{ID: "namespace", Value: "namespace"}, {ID: "task-queue", Value: "task-queue"}}}
	host, err := New(Options{Profile: profile, Endpoints: map[string]Endpoint{"endpoint": {Target: address, Credentials: insecure.NewCredentials(), Metadata: metadata.Pairs("authorization", "host-secret")}}})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, host.Close(context.Background())) })
	nodes := []*testpilotspb.InstructionNode{rpcNode("check", "/grpc.health.v1.Health/Check"), rpcNode("length", "/example.Echo/Length")}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "case", Program: &testpilotspb.Program{ProgramId: "program", Roles: []*testpilotspb.Role{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}}, Entrypoints: []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: nodes}}, Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"}}, Contract: &testpilotspb.Contract{ContractId: "contract", Rules: []*testpilotspb.ContractRule{{RuleId: "safety", Kind: testpilotspb.CONTRACT_RULE_KIND_SAFETY, InitialStateId: "start", States: []*testpilotspb.ContractState{{StateId: "start", Status: testpilotspb.CONTRACT_STATE_STATUS_PENDING}, {StateId: "good", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED}}, Transitions: []*testpilotspb.ContractTransition{{TransitionId: "complete", SourceStateId: "start", TargetStateId: "good", Predicate: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}}, EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}}, SupportKind: testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT}}}}}}
	_, err = testpilot.Prepare(source, host)
	require.NoError(t, err)
	return host, source, []protoreflect.MethodDescriptor{healthpb.File_grpc_health_v1_health_proto.Services().ByName("Health").Methods().ByName("Check"), file.Services().Get(0).Methods().Get(0)}
}

// prepared is source's Program as preparation hands it to h.
func prepared(t *testing.T, h *Driver, source *testpilotspb.Case) testpilot.PreparedProgram {
	t.Helper()
	preparedCase, err := testpilot.Prepare(source, h)
	require.NoError(t, err)
	return facadetest.Capture(t, preparedCase)
}

func rpcNode(id, method string) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{InstructionId: id, Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: method}}}, Limits: &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 2000}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}}}
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
	s, err := h.OpenSession(t.Context(), "run", prepared(t, h, source))
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
	require.EqualValues(t, 32, h.Snapshot().ProgramLimits.MaxAttempts)
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
	s, err := h.OpenSession(t.Context(), "run", prepared(t, h, source))
	require.NoError(t, err)
	retried := coordinate("run", "check")
	retried.Attempt = 2
	for _, test := range []struct {
		name       string
		coordinate testpilot.Coordinate
		role       string
		method     protoreflect.MethodDescriptor
		message    proto.Message
		want       error
	}{
		{"wrong run", coordinate("foreign", "check"), "endpoint", methods[0], request(methods[0], ""), errUnauthorized},
		{"unknown instruction", coordinate("run", "missing"), "endpoint", methods[0], request(methods[0], ""), errUnauthorized},
		{"attempt beyond the instruction's", retried, "endpoint", methods[0], request(methods[0], ""), errUnauthorized},
		{"wrong role", coordinate("run", "check"), "missing", methods[0], request(methods[0], ""), errUnauthorized},
		{"method the instruction does not name", coordinate("run", "check"), "endpoint", methods[1], request(methods[1], "x"), errUnauthorized},
		{"wrong descriptor", coordinate("run", "check"), "endpoint", methods[0], wrapperspb.String(""), errUnauthorized},
		{"oversized", coordinate("run", "check"), "endpoint", methods[0], request(methods[0], strings.Repeat("x", 4096)), errCapacity},
		{"stream", coordinate("run", "check"), "endpoint", methods[0].Parent().(protoreflect.ServiceDescriptor).Methods().ByName("Watch"), request(methods[0], ""), errUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			handle, err := s.InvokeRPC(t.Context(), test.coordinate, test.role, test.method, test.message)
			require.ErrorIs(t, err, test.want)
			require.Nil(t, handle)
			require.NotContains(t, err.Error(), "host-secret")
		})
	}
	// A poll is authorized only for a ReadEvidence instruction, even on the method an RPC names.
	polled, err := s.PollRPC(t.Context(), coordinate("run", "check"), "endpoint", methods[0], request(methods[0], ""), time.Millisecond, func(context.Context, proto.Message) (bool, error) { return true, nil })
	require.ErrorIs(t, err, errUnauthorized)
	require.Nil(t, polled)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	handle, err := s.InvokeRPC(ctx, coordinate("run", "check"), "endpoint", methods[0], request(methods[0], ""))
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, handle)
	handle, err = s.InvokeRPC(t.Context(), coordinate("run", "check"), "endpoint", methods[0], request(methods[0], ""))
	require.NoError(t, err)
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, result.Outcome.Status)
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
				source.Program.Entrypoints[0].Instructions[0].Limits.Timeout = &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 20}
			}
			s, err := h.OpenSession(t.Context(), "run", prepared(t, h, source))
			require.NoError(t, err)
			handle, err := s.InvokeRPC(t.Context(), coordinate("run", "check"), "endpoint", methods[0], request(methods[0], ""))
			require.NoError(t, err)
			if kind == "cancel" {
				<-entered
				require.NoError(t, handle.Cancel(t.Context()))
			}
			result, err := handle.Wait(t.Context())
			require.NoError(t, err)
			want := testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE
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

// A session reads each instruction's bounds from its prepared plan: a node that writes no limits
// runs under the defaults preparation resolved, whatever the Profile's defaults are at call time.
func TestSessionReadsPreparedInstructionBounds(t *testing.T) {
	address := startGRPC(t, func(ctx context.Context, _ any, _ *grpc.UnaryServerInfo, _ grpc.UnaryHandler) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	h, source, methods := fixture(t, address)
	source.Program.Entrypoints[0].Instructions[0].Limits = nil
	h.profile.InstructionDefaults = testpilot.InstructionDefaults{TimeoutMilliseconds: 20, MaxAttempts: 2}
	s, err := h.OpenSession(t.Context(), "run", prepared(t, h, source))
	require.NoError(t, err)
	h.profile.InstructionDefaults = testpilot.InstructionDefaults{}
	beyond := coordinate("run", "check")
	beyond.Attempt = 3
	denied, err := s.InvokeRPC(t.Context(), beyond, "endpoint", methods[0], request(methods[0], ""))
	require.ErrorIs(t, err, errUnauthorized)
	require.Nil(t, denied)
	last := coordinate("run", "check")
	last.Attempt = 2
	handle, err := s.InvokeRPC(t.Context(), last, "endpoint", methods[0], request(methods[0], ""))
	require.NoError(t, err)
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT, result.Outcome.Status)
	require.NoError(t, s.Close(t.Context()))
}
func TestParallelSessionsAndQuarantineCapacity(t *testing.T) {
	address := startGRPC(t, nil)
	h, source, methods := fixture(t, address)
	t.Run("parallel", func(t *testing.T) {
		for i := 0; i < 8; i++ {
			t.Run(fmt.Sprintf("run%d", i), func(t *testing.T) {
				t.Parallel()
				run := fmt.Sprintf("run%d", i)
				s, err := h.OpenSession(t.Context(), run, prepared(t, h, source))
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
	s, err := h.OpenSession(t.Context(), "stuck", prepared(t, h, source))
	require.NoError(t, err)
	released := make(chan struct{})
	e, err := s.start(t.Context(), coordinate("stuck", "check"), 2000, func(context.Context) testpilot.EffectResult {
		<-released
		return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	require.ErrorIs(t, e.Drain(ctx), context.DeadlineExceeded)
	require.NoError(t, s.Quarantine(t.Context(), e))
	require.NoError(t, s.Close(t.Context()))
	other, err := h.OpenSession(t.Context(), "other", prepared(t, h, source))
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
	s, err := h.OpenSession(t.Context(), "run", prepared(t, h, source))
	require.NoError(t, err)
	_, err = h.OpenSession(t.Context(), "run", prepared(t, h, source))
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
	_, err = h.OpenSession(t.Context(), "later", prepared(t, h, source))
	require.ErrorIs(t, err, errClosed)
}
