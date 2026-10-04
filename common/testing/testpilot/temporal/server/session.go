package server

import (
	"context"
	"errors"
	"strings"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type nodeKey struct{ entry, node string }

type Session struct {
	started                       map[testpilot.Coordinate]struct{}
	host                          *Driver
	runID                         string
	entries                       map[string]struct{}
	controllers                   map[string]struct{}
	instructions                  map[nodeKey]testpilot.InstructionPlan
	effects                       map[*effect]struct{}
	handles                       map[*opaqueHandle]struct{}
	slots                         map[string]*handleSlot
	minted, attempts, diagnostics int64
	closed                        bool
	closedSignal                  chan struct{}
}

func (s *Session) Reserve(ctx context.Context, _ testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return nil, err
	}
	return nil, errUnauthorized
}

// InjectFault refuses every fault: outages are worker-lifecycle events and the server Driver owns
// no worker, so it declines them the way it declines reservations rather than pretending to
// realize one.
func (s *Session) InjectFault(ctx context.Context, _ testpilot.Coordinate, _ string, _ testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return nil, err
	}
	return nil, errUnauthorized
}

func (s *Session) controllerPlan(c testpilot.Coordinate) (testpilot.InstructionPlan, error) {
	if c.RunID != s.runID || c.ActivationID == "" || len(c.ActivationID) > 256 || c.Attempt <= 0 {
		return testpilot.InstructionPlan{}, errUnauthorized
	}
	if _, ok := s.controllers[c.EntrypointID]; !ok {
		return testpilot.InstructionPlan{}, errUnauthorized
	}
	plan, ok := s.instructions[nodeKey{c.EntrypointID, c.InstructionID}]
	if !ok || c.Attempt > plan.MaxAttempts() {
		return testpilot.InstructionPlan{}, errUnauthorized
	}
	return plan, nil
}

// authorizeUnary is the authority boundary both unary effects share: the coordinate names a
// prepared controller instruction, the call names the role that instruction's opcode arm declares
// and the method it was admitted with, and the request is a bounded message of the method's input.
func (s *Session) authorizeUnary(ctx context.Context, c testpilot.Coordinate, opcode testpilot.Opcode, role string, method protoreflect.MethodDescriptor, request proto.Message) (testpilot.InstructionPlan, endpoint, error) {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return testpilot.InstructionPlan{}, endpoint{}, err
	}
	plan, err := s.controllerPlan(c)
	if err != nil {
		return testpilot.InstructionPlan{}, endpoint{}, err
	}
	if primitive.NilValue(method) || method.IsStreamingClient() || method.IsStreamingServer() || primitive.NilValue(request) {
		return testpilot.InstructionPlan{}, endpoint{}, errUnauthorized
	}
	instruction := plan.Source().GetInstruction()
	declared := instruction.GetInvokeRpc().GetEndpointRoleId()
	if opcode == testpilot.ReadEvidence {
		declared = instruction.GetReadEvidence().GetEndpointRoleId()
	}
	path := primitive.MethodPath(method)
	target, ok := s.host.endpoints[role]
	if !ok || !target.methods[path] || declared != role || primitive.NilValue(plan.Method()) || primitive.MethodPath(plan.Method()) != path || request.ProtoReflect().Descriptor() != method.Input() {
		return testpilot.InstructionPlan{}, endpoint{}, errUnauthorized
	}
	if int64(proto.Size(request)) > s.host.profile.ProgramLimits.MaxRequestBytes {
		return testpilot.InstructionPlan{}, endpoint{}, errCapacity
	}
	return plan, target, nil
}

func (s *Session) InvokeRPC(ctx context.Context, c testpilot.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message) (testpilot.EffectHandle, error) {
	plan, endpoint, err := s.authorizeUnary(ctx, c, testpilot.InvokeRPC, role, method, request)
	if err != nil {
		return nil, err
	}
	path := primitive.MethodPath(method)
	request = proto.Clone(request)
	handle, err := s.start(ctx, c, plan.TimeoutMilliseconds(), func(ctx context.Context) testpilot.EffectResult {
		response := dynamicpb.NewMessage(method.Output())
		ctx = metadata.NewOutgoingContext(ctx, endpoint.metadata.Copy())
		err := endpoint.connection.Invoke(ctx, path, request, response, grpc.MaxCallRecvMsgSize(int(s.host.profile.ProgramLimits.MaxInstructionResponseBytes)))
		if err != nil {
			return rpcFailure(ctx, err)
		}
		return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}, Response: response}
	})
	if err != nil {
		return nil, err
	}
	return handle, nil
}

// PollRPC repeats the read declaration's RPC on the endpoint role until the runtime's predicate
// accepts a response or the instruction's timeout ends it. One effect, one attempt: the polls are
// the effect's own calls, so the Session's attempt and identity accounting sees the instruction once.
// A zero interval reads once: a response the predicate does not accept ends the effect TIMED_OUT,
// with no response and no protocol code, as a poll whose timeout ran out.
func (s *Session) PollRPC(ctx context.Context, c testpilot.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message, interval time.Duration, satisfied testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	plan, endpoint, err := s.authorizeUnary(ctx, c, testpilot.ReadEvidence, role, method, request)
	if err != nil {
		return nil, err
	}
	if interval < 0 || satisfied == nil {
		return nil, errUnauthorized
	}
	path := primitive.MethodPath(method)
	request = proto.Clone(request)
	handle, err := s.start(ctx, c, plan.TimeoutMilliseconds(), func(ctx context.Context) testpilot.EffectResult {
		for {
			response := dynamicpb.NewMessage(method.Output())
			callCtx := metadata.NewOutgoingContext(ctx, endpoint.metadata.Copy())
			if err := endpoint.connection.Invoke(callCtx, path, request, response, grpc.MaxCallRecvMsgSize(int(s.host.profile.ProgramLimits.MaxInstructionResponseBytes))); err != nil {
				return rpcFailure(ctx, err)
			}
			accepted, err := satisfied(ctx, response)
			if err != nil {
				return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, ProtocolCode: "poll_predicate_failed", Detail: err.Error()}}
			}
			if accepted {
				return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}, Response: response}
			}
			if interval == 0 {
				return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT}}
			}
			timer := time.NewTimer(interval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return rpcFailure(ctx, ctx.Err())
			case <-timer.C:
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return handle, nil
}

func rpcFailure(ctx context.Context, err error) testpilot.EffectResult {
	code := status.Code(err)
	kind := testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || code == codes.DeadlineExceeded {
		kind = testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT
		code = codes.DeadlineExceeded
	}
	if errors.Is(ctx.Err(), context.Canceled) || code == codes.Canceled {
		kind = testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED
		code = codes.Canceled
	}
	return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: kind, ProtocolCode: strings.ToLower(code.String())}}
}

type effect struct {
	session     *Session
	cancel      context.CancelFunc
	done        chan struct{}
	result      testpilot.EffectResult
	quarantined bool
}

func (s *Session) start(ctx context.Context, c testpilot.Coordinate, timeoutMilliseconds int64, call func(context.Context) testpilot.EffectResult) (*effect, error) {
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	return s.startLocked(ctx, c, timeoutMilliseconds, call)
}

func (s *Session) startLocked(ctx context.Context, c testpilot.Coordinate, timeoutMilliseconds int64, call func(context.Context) testpilot.EffectResult) (*effect, error) {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return nil, err
	}
	if s.closed || s.host.closed {
		return nil, errClosed
	}
	if _, duplicate := s.started[c]; duplicate {
		return nil, errInvalid
	}
	// The duration ceilings scale as a hinted timeout does, so the cap never cuts a scaled bound.
	limits := s.host.profile.BoundScale.Ceilings(s.host.profile.ProgramLimits)
	if s.host.effects >= limits.MaxAttempts || s.attempts >= limits.MaxAttempts {
		return nil, errCapacity
	}
	timeout := min(timeoutMilliseconds, max(limits.MaxTotalDurationMilliseconds, limits.MaxCleanupDurationMilliseconds))
	if timeout <= 0 {
		return nil, errInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	e := &effect{session: s, cancel: cancel, done: make(chan struct{})}
	s.effects[e] = struct{}{}
	s.started[c] = struct{}{}
	s.attempts++
	s.host.effects++
	go func() {
		result := call(ctx)
		cancel()
		s.host.mu.Lock()
		e.result = result
		delete(s.effects, e)
		s.host.effects--
		if s.closed && len(s.effects) == 0 {
			delete(s.host.sessions, s.runID)
		}
		close(e.done)
		s.host.mu.Unlock()
	}()
	return e, nil
}

func (e *effect) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return testpilot.EffectResult{}, err
	}
	select {
	case <-ctx.Done():
		return testpilot.EffectResult{}, ctx.Err()
	case <-e.done:
		return primitive.CloneEffectResult(e.result), nil
	}
}
func (e *effect) Cancel(ctx context.Context) error {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return err
	}
	e.cancel()
	return nil
}
func (e *effect) Drain(ctx context.Context) error {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done:
		return nil
	}
}
func (s *Session) Quarantine(ctx context.Context, handle testpilot.EffectHandle) error {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return err
	}
	e, ok := handle.(*effect)
	if !ok || e == nil || e.session != s {
		return errUnauthorized
	}
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
		return err
	}
	defer s.host.mu.Unlock()
	e.quarantined = true
	return nil
}
func (s *Session) Close(ctx context.Context) error {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return err
	}
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
		return err
	}
	defer s.host.mu.Unlock()
	s.closeLocked()
	return nil
}
func (s *Session) closeLocked() {
	if s.closed {
		return
	}
	s.closed = true
	close(s.closedSignal)
	for e := range s.effects {
		e.cancel()
	}
	for handle := range s.handles {
		handle.invoke = nil
	}
	clear(s.handles)
	clear(s.slots)
	if len(s.effects) == 0 {
		delete(s.host.sessions, s.runID)
	}
}
func (s *Session) Diagnose(ctx context.Context, runID string, _ *testpilotspb.RunDiagnostic) error {
	if err := primitive.ContextError(ctx, errInvalid); err != nil {
		return err
	}
	if err := s.host.mu.LockContext(ctx, errInvalid); err != nil {
		return err
	}
	defer s.host.mu.Unlock()
	if runID != s.runID {
		return errUnauthorized
	}
	if s.diagnostics >= min(s.host.profile.ProgramLimits.MaxRunEvents, 64) {
		return errCapacity
	}
	s.diagnostics++
	return nil
}
