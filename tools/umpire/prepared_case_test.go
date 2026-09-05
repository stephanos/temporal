package umpire

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/await"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type runHost struct {
	identity    HostIdentity
	mu          sync.Mutex
	runIDs      []string
	sessions    []*runSession
	quarantine  []*runEffect
	active      int
	diagnostics []*umpirespb.RunDiagnostic
}

func (h *runHost) Identity(context.Context) (HostIdentity, error) { return h.identity, nil }
func (h *runHost) Open(_ context.Context, runID string, _ PreparedProgram) (Session, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	session := &runSession{host: h, value: runID}
	h.runIDs = append(h.runIDs, runID)
	h.sessions = append(h.sessions, session)
	return session, nil
}

type runSession struct {
	host   *runHost
	value  string
	mu     sync.Mutex
	closed int
}

func (*runSession) Reserve(context.Context, ReservationRequest) ([]ReservationHandle, error) {
	panic("empty Program must not reserve activations")
}
func (s *runSession) InvokeRPC(_ context.Context, coordinate Coordinate, _ string, method protoreflect.MethodDescriptor, request proto.Message) (EffectHandle, error) {
	if coordinate.InstructionID == "late" {
		return newRunEffect(EffectResult{}, false), nil
	}
	if coordinate.InstructionID == "consume" {
		field := request.ProtoReflect().Descriptor().Fields().ByName("text")
		if request.ProtoReflect().Get(field).String() != s.value {
			return nil, errors.New("run read another Run's Slot value")
		}
	}
	response := dynamicpb.NewMessage(method.Output())
	response.Set(response.Descriptor().Fields().ByName("text"), protoreflect.ValueOfString(s.value))
	return newRunEffect(EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}, true), nil
}
func (*runSession) CompleteNexusOperation(context.Context, Coordinate, OpaqueCapability, *umpirespb.Value) (EffectHandle, error) {
	panic("empty Program must not complete Nexus operations")
}
func (*runSession) Bridge(context.Context) (CapabilityBridge, error) {
	panic("empty Program must not open a capability bridge")
}
func (s *runSession) Quarantine(_ context.Context, handle EffectHandle) error {
	effect := handle.(*runEffect)
	s.host.mu.Lock()
	s.host.quarantine = append(s.host.quarantine, effect)
	s.host.active++
	s.host.mu.Unlock()
	go func() {
		<-effect.done
		s.host.mu.Lock()
		defer s.host.mu.Unlock()
		s.host.active--
		s.host.diagnostics = append(s.host.diagnostics, &umpirespb.RunDiagnostic{Kind: umpirespb.RUN_DIAGNOSTIC_KIND_POST_CLOSE_EVENT, Code: "quarantine_completed"})
	}()
	return nil
}
func (s *runSession) Close(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed++
	return nil
}
func (s *runSession) Diagnose(_ context.Context, _ string, diagnostic *umpirespb.RunDiagnostic) error {
	s.host.mu.Lock()
	defer s.host.mu.Unlock()
	s.host.diagnostics = append(s.host.diagnostics, proto.CloneOf(diagnostic))
	return nil
}

type runEffect struct {
	done   chan struct{}
	result EffectResult
	once   sync.Once
}

func newRunEffect(result EffectResult, complete bool) *runEffect {
	effect := &runEffect{done: make(chan struct{}), result: result}
	if complete {
		effect.complete()
	}
	return effect
}

func (e *runEffect) Wait(ctx context.Context) (EffectResult, error) {
	select {
	case <-e.done:
		return e.result, nil
	case <-ctx.Done():
		return EffectResult{}, ctx.Err()
	}
}
func (*runEffect) Cancel(context.Context) error { return nil }
func (e *runEffect) Drain(context.Context) error {
	select {
	case <-e.done:
		return nil
	default:
		return context.DeadlineExceeded
	}
}
func (e *runEffect) complete() { e.once.Do(func() { close(e.done) }) }

func statefulRunFixture(t *testing.T) (*umpirespb.Case, ProfileSpec) {
	t.Helper()
	source, profile := preparationFixture(t)
	catalog, err := NewCatalog(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: proto.String("run.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("text"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}}}}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Service"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Call"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload")}}}}}}})
	require.NoError(t, err)
	profile.Catalog = catalog
	profile.Roles = []RolePolicy{{ID: "endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT, Methods: []string{"/example.Service/Call"}}}
	profile.Capabilities = []Capability{InvokeRPC}
	source.Program.Roles = []*umpirespb.ProgramRole{{RoleId: "endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT}}
	source.Program.Slots = []*umpirespb.SlotSchema{{SlotId: "result", Kind: umpirespb.SLOT_KIND_VALUE, Type: runScalar(umpirespb.SCALAR_KIND_TEXT)}}
	source.Program.Observations = []*umpirespb.ObservationSchema{{ObservationId: "result", Type: runScalar(umpirespb.SCALAR_KIND_TEXT)}}
	call := runRPCNode("call")
	call.Instruction.GetInvokeRpc().ResponseProjections = []*umpirespb.ResponseProjection{{Source: runField("text"), Cardinality: umpirespb.PROJECTION_CARDINALITY_ONE, Sinks: []*umpirespb.ProjectionSink{{Sink: &umpirespb.ProjectionSink_SlotId{SlotId: "result"}}, {Sink: &umpirespb.ProjectionSink_ObservationId{ObservationId: "result"}}}}}
	consume := runRPCNode("consume")
	consume.Dependencies = []*umpirespb.InstructionReference{{EntrypointId: "controller", InstructionId: "call"}}
	consume.Guard = &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Present{Present: &umpirespb.PresentExpression{Operand: runSlot("result")}}}
	consume.Instruction.GetInvokeRpc().RequestAssignments = []*umpirespb.RequestAssignment{{Target: runField("text"), Value: runSlot("result")}}
	source.Program.Entrypoints[0].Nodes = []*umpirespb.InstructionNode{call, consume, runRPCNode("late")}
	rule := source.Contract.Rules[0]
	rule.States[1].Terminal = umpirespb.CONTRACT_TERMINAL_STATE_VIOLATED
	rule.Transitions[0].Predicate = &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Equals{Equals: &umpirespb.EqualsExpression{Left: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_RunEvent{RunEvent: &umpirespb.RunEventFieldReference{Field: umpirespb.RUN_EVENT_FIELD_INSTRUCTION_ID}}}, Right: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "consume"}}}}}}}
	return source, profile
}

func runScalar(kind umpirespb.ScalarKind) *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Scalar{Scalar: &umpirespb.ScalarType{Kind: kind}}}}}
}
func runRPCNode(id string) *umpirespb.InstructionNode {
	return &umpirespb.InstructionNode{InstructionId: id, Instruction: &umpirespb.Instruction{Instruction: &umpirespb.Instruction_InvokeRpc{InvokeRpc: &umpirespb.InvokeRPC{EndpointRoleId: "endpoint", Method: "/example.Service/Call"}}}, Outcome: &umpirespb.InstructionOutcomeSchema{Fields: []*umpirespb.OutcomeFieldSchema{{Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_STATUS, Type: &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Enumeration{Enumeration: &umpirespb.NamedType{ProtobufType: "temporal.server.api.umpire.v1.InstructionOutcomeStatus"}}}}}}}}, Bounds: &umpirespb.InstructionBounds{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 8, MaxResponseBytes: 4096}}
}
func runField(name string) *umpirespb.FieldPath {
	return &umpirespb.FieldPath{Segments: []*umpirespb.FieldPathSegment{{Field: name}}}
}
func runSlot(id string) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Slot{Slot: &umpirespb.SlotReference{SlotId: id}}}
}

func TestPreparedCaseRunUsesFreshSequentialAndConcurrentState(t *testing.T) {
	source, profile := statefulRunFixture(t)
	prepared, err := PrepareCase(source, profile)
	require.NoError(t, err)
	host := &runHost{identity: prepared.Identity()}

	const concurrent = 8
	const sequential = 2
	const count = concurrent + sequential
	type result struct {
		run     *umpirespb.Run
		verdict *umpirespb.Verdict
		err     error
	}
	results := make(chan result, count)
	execute := func() {
		run, verdict, err := prepared.Run(t.Context(), host)
		results <- result{run: run, verdict: verdict, err: err}
	}
	for range sequential {
		execute()
	}
	var wait sync.WaitGroup
	for range concurrent {
		wait.Go(execute)
	}
	wait.Wait()
	close(results)

	identities := make(map[string]struct{}, count)
	observations := make(map[string]struct{}, count)
	serialized := make(map[string][]byte, count)
	serializedVerdicts := make(map[string][]byte, count)
	returned := make([]result, 0, count)
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, umpirespb.VERDICT_KIND_VIOLATED, result.verdict.GetKind())
		run := result.run
		require.Equal(t, source.GetCaseId(), run.GetCaseId())
		require.Equal(t, source.GetProgram().GetProgramId(), run.GetProgramId())
		require.Equal(t, umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR, run.GetDisposition())
		require.NotEmpty(t, run.GetRunId())
		require.NotContains(t, identities, run.GetRunId())
		identities[run.GetRunId()] = struct{}{}
		observation := runObservation(run, "result")
		require.Equal(t, run.GetRunId(), observation)
		require.NotContains(t, observations, observation)
		observations[observation] = struct{}{}
		serialized[run.GetRunId()], err = proto.Marshal(run)
		require.NoError(t, err)
		serializedVerdicts[run.GetRunId()], err = proto.Marshal(result.verdict)
		require.NoError(t, err)
		returned = append(returned, result)
	}
	host.mu.Lock()
	require.Len(t, host.runIDs, count)
	require.Len(t, host.sessions, count)
	for _, session := range host.sessions {
		require.Equal(t, 1, session.closed)
	}
	require.Equal(t, count, host.active)
	quarantined := append([]*runEffect(nil), host.quarantine...)
	host.mu.Unlock()
	require.Len(t, quarantined, count)
	for _, effect := range quarantined {
		effect.complete()
	}
	await.RequireTrue(t, func() bool {
		host.mu.Lock()
		defer host.mu.Unlock()
		return host.active == 0 && len(host.diagnostics) == count
	}, time.Second, time.Millisecond)
	for _, result := range returned {
		after, err := proto.Marshal(result.run)
		require.NoError(t, err)
		require.Equal(t, serialized[result.run.GetRunId()], after)
		afterVerdict, err := proto.Marshal(result.verdict)
		require.NoError(t, err)
		require.Equal(t, serializedVerdicts[result.run.GetRunId()], afterVerdict)
	}
	firstRunVerdict := proto.CloneOf(returned[0].run.GetVerdict())
	returned[0].run.Events[0].SourceId = "mutated"
	returned[0].verdict.Kind = umpirespb.VERDICT_KIND_INCONCLUSIVE
	require.True(t, proto.Equal(firstRunVerdict, returned[0].run.GetVerdict()))
	for _, result := range returned[1:] {
		after, err := proto.Marshal(result.run)
		require.NoError(t, err)
		require.Equal(t, serialized[result.run.GetRunId()], after)
		require.Equal(t, umpirespb.VERDICT_KIND_VIOLATED, result.verdict.GetKind())
	}
}

func runObservation(run *umpirespb.Run, id string) string {
	for _, event := range run.GetEvents() {
		for _, observation := range event.GetObservations() {
			if observation.GetObservationId() == id {
				return observation.GetValue().GetText()
			}
		}
	}
	return ""
}

func TestPreparedCaseRunRejectsPreflightBeforeOpen(t *testing.T) {
	source, profile := preparationFixture(t)
	prepared, err := PrepareCase(source, profile)
	require.NoError(t, err)
	validHost := &testHost{identity: prepared.Identity()}

	for _, test := range []struct {
		name    string
		prepare func() (*PreparedCase, context.Context, Host)
	}{
		{name: "nil context", prepare: func() (*PreparedCase, context.Context, Host) { return prepared, nil, validHost }},
		{name: "typed nil Host", prepare: func() (*PreparedCase, context.Context, Host) { return prepared, t.Context(), nilHostMap(nil) }},
		{name: "identity mismatch", prepare: func() (*PreparedCase, context.Context, Host) { return prepared, t.Context(), &testHost{} }},
		{name: "factory failure", prepare: func() (*PreparedCase, context.Context, Host) {
			candidate := *prepared
			candidate.factory = factoryFunc(func(context.Context, execution.ProgramView) (execution.Monitor, error) {
				return nil, errors.New("factory failed")
			})
			return &candidate, t.Context(), validHost
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate, ctx, host := test.prepare()
			run, verdict, err := candidate.Run(ctx, host)
			require.Error(t, err)
			require.Nil(t, run)
			require.Nil(t, verdict)
		})
	}
	require.Zero(t, validHost.opens)
}
