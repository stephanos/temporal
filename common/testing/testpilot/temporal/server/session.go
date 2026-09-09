package server

import (
	"context"
	"errors"
	"strings"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
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
	nodes                         map[nodeKey]*testpilotspb.InstructionDefinition
	effects                       map[*effect]struct{}
	capabilities                  map[*opaqueCapability]struct{}
	slots                         map[string]*capabilitySlot
	minted, attempts, diagnostics int64
	closed                        bool
	closedSignal                  chan struct{}
}

func (s *Session) Reserve(ctx context.Context, _ testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return nil, errUnauthorized
}

// Faults are worker-lifecycle outages; the server Driver owns no worker, so it refuses them the
// way it refuses reservations rather than pretending to realize one.
func (s *Session) InjectFault(ctx context.Context, _ testpilot.Coordinate, _ string, _ testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	return nil, errUnauthorized
}

func (s *Session) controllerNode(c testpilot.Coordinate) (*testpilotspb.InstructionDefinition, error) {
	if c.RunID != s.runID || c.ActivationID == "" || len(c.ActivationID) > 256 || c.Attempt <= 0 {
		return nil, errUnauthorized
	}
	if _, ok := s.controllers[c.EntrypointID]; !ok {
		return nil, errUnauthorized
	}
	n := s.nodes[nodeKey{c.EntrypointID, c.InstructionID}]
	if n == nil || c.Attempt > n.GetLimits().GetMaxAttempts() {
		return nil, errUnauthorized
	}
	return n, nil
}

func (s *Session) InvokeRPC(ctx context.Context, c testpilot.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message) (testpilot.EffectHandle, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	n, err := s.controllerNode(c)
	if err != nil {
		return nil, err
	}
	rpc := n.GetInstruction().GetInvokeRpc()
	if rpc == nil || nilValue(method) || method.IsStreamingClient() || method.IsStreamingServer() || nilValue(request) {
		return nil, errUnauthorized
	}
	path := "/" + string(method.Parent().FullName()) + "/" + string(method.Name())
	endpoint, ok := s.host.endpoints[role]
	if !ok || !endpoint.methods[path] || rpc.EndpointRoleId != role || rpc.Method != path || request.ProtoReflect().Descriptor() != method.Input() {
		return nil, errUnauthorized
	}
	if int64(proto.Size(request)) > s.host.profile.ProgramLimits.MaxRequestBytes {
		return nil, errCapacity
	}
	request = proto.Clone(request)
	handle, err := s.start(ctx, c, n.Limits, func(ctx context.Context) testpilot.EffectResult {
		response := dynamicpb.NewMessage(method.Output())
		ctx = metadata.NewOutgoingContext(ctx, endpoint.metadata.Copy())
		err := endpoint.connection.Invoke(ctx, path, request, response, grpc.MaxCallRecvMsgSize(int(min(n.Limits.MaxResponseBytes, s.host.profile.ProgramLimits.MaxResponseBytes))))
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

func rpcFailure(ctx context.Context, err error) testpilot.EffectResult {
	code := status.Code(err)
	kind := testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS
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

func (s *Session) start(ctx context.Context, c testpilot.Coordinate, bounds *testpilotspb.InstructionLimits, call func(context.Context) testpilot.EffectResult) (*effect, error) {
	if err := s.host.mu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer s.host.mu.Unlock()
	return s.startLocked(ctx, c, bounds, call)
}

func (s *Session) startLocked(ctx context.Context, c testpilot.Coordinate, bounds *testpilotspb.InstructionLimits, call func(context.Context) testpilot.EffectResult) (*effect, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if s.closed || s.host.closed {
		return nil, errClosed
	}
	if _, duplicate := s.started[c]; duplicate {
		return nil, errInvalid
	}
	limits := s.host.profile.ProgramLimits
	if s.host.effects >= limits.MaxAttempts || s.attempts >= limits.MaxAttempts {
		return nil, errCapacity
	}
	timeout := min(bounds.GetTimeoutMilliseconds(), max(limits.MaxTotalDurationMilliseconds, limits.MaxCleanupDurationMilliseconds))
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
	if err := contextError(ctx); err != nil {
		return testpilot.EffectResult{}, err
	}
	select {
	case <-ctx.Done():
		return testpilot.EffectResult{}, ctx.Err()
	case <-e.done:
		return testpilot.EffectResult{Outcome: proto.CloneOf(e.result.Outcome), Response: proto.Clone(e.result.Response)}, nil
	}
}
func (e *effect) Cancel(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	e.cancel()
	return nil
}
func (e *effect) Drain(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
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
	if err := contextError(ctx); err != nil {
		return err
	}
	e, ok := handle.(*effect)
	if !ok || e == nil || e.session != s {
		return errUnauthorized
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
		return err
	}
	defer s.host.mu.Unlock()
	e.quarantined = true
	return nil
}
func (s *Session) Close(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
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
	for capability := range s.capabilities {
		capability.invoke = nil
	}
	clear(s.capabilities)
	clear(s.slots)
	if len(s.effects) == 0 {
		delete(s.host.sessions, s.runID)
	}
}
func (s *Session) Diagnose(ctx context.Context, runID string, _ *testpilotspb.RunDiagnostic) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := s.host.mu.LockContext(ctx); err != nil {
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
