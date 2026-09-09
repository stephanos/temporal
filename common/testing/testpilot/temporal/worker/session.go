package worker

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/nexus-rpc/sdk-go/nexus"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type programDefinition struct {
	snapshot       *testpilotspb.Program
	entries        map[string]entryDefinition
	registrations  []queueRegistration
	endpoints      map[string]string
	queueWorkflows map[string]map[string]struct{}
	// faultQueues maps each task-queue role a fault instruction names to the queue that role
	// resolves to, so realizing a fault never re-reads the Program.
	faultQueues map[string]string
	hasAsync    bool
	hasFault    bool
}

type entryDefinition struct {
	plan                testpilot.EntrypointPlan
	namespace           string
	queue, workflowType string
	service, operation  string
}

type Session struct {
	host               *Driver
	mu                 contextMutex
	closeMu            contextMutex
	runID              string
	id                 string
	definition         programDefinition
	ledger             *delivery.Ledger
	options            SessionOptions
	reservations       map[string]*reservation
	carriers           map[testpilot.Coordinate]*Carrier
	nexusResults       map[string]*nexusResult
	workflowKeys       map[workflowRouteIndex]struct{}
	nexusKeys          map[nexusRouteIndex]struct{}
	nexusDispatch      map[nexusDispatchKey]nexus.Header
	workflowAdmissions map[workflowAdmissionKey]*workflowAdmission
	nexusAdmissions    map[nexusRouteIndex]nexusAdmission
	next               atomic.Uint64
	diagnostics        int
	closed             bool
	failure            error
	workers            *workerLease
	stopComplete       bool
	released           bool
	removed            bool
}

func newSession(host *Driver, runID, sessionID string, definition programDefinition, options SessionOptions) (*Session, error) {
	if host == nil || definition.snapshot == nil || definition.snapshot.GetLimits() == nil {
		return nil, ErrInvalid
	}
	limits := definition.snapshot.GetLimits()
	ledger, err := delivery.New(delivery.Config{RunID: runID, SessionID: sessionID, Limits: delivery.Limits{
		MaxRoutes: boundedInt(limits.GetMaxActivations()), MaxHandles: boundedInt(limits.GetMaxActivations()),
		MaxHeaderBytes: boundedInt(limits.GetMaxRequestBytes()), MaxDiagnostics: boundedInt(limits.GetMaxRunEvents()),
	}})
	if err != nil {
		return nil, err
	}
	return &Session{host: host, mu: newContextMutex(), closeMu: newContextMutex(), runID: runID, id: sessionID, definition: definition, ledger: ledger, options: options,
		reservations: make(map[string]*reservation), carriers: make(map[testpilot.Coordinate]*Carrier), nexusResults: make(map[string]*nexusResult),
		workflowKeys: make(map[workflowRouteIndex]struct{}), nexusKeys: make(map[nexusRouteIndex]struct{}), nexusDispatch: make(map[nexusDispatchKey]nexus.Header),
		workflowAdmissions: make(map[workflowAdmissionKey]*workflowAdmission), nexusAdmissions: make(map[nexusRouteIndex]nexusAdmission)}, nil
}

func (s *Session) Reserve(ctx context.Context, request testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	if s == nil || ctx == nil || request.Origin.RunID != s.runID || request.EntrypointID == "" || request.Count <= 0 {
		return nil, ErrInvalid
	}
	entry, exists := s.definition.entries[request.EntrypointID]
	if !exists || entry.plan.Context() != testpilotspb.ENTRYPOINT_KIND_WORKFLOW && entry.plan.Context() != testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER {
		return nil, ErrInvalid
	}
	if err := s.mu.lock(ctx); err != nil {
		return nil, err
	}
	defer s.mu.unlock()
	if s.closed || s.failure != nil {
		return nil, errors.Join(ErrClosed, s.failure)
	}
	maximum := boundedInt(s.definition.snapshot.GetLimits().GetMaxActivations())
	if int(request.Count) > maximum-len(s.reservations) {
		return nil, ErrCapacity
	}
	result := make([]testpilot.ReservationHandle, 0, request.Count)
	for ordinal := int64(0); ordinal < request.Count; ordinal++ {
		identity := testpilot.ReservationIdentity{Origin: request.Origin, EntrypointID: request.EntrypointID, Ordinal: ordinal, ID: fmt.Sprintf("reservation-%d", s.next.Add(1))}
		raw := newReservation(identity)
		retained, err := s.ledger.RetainReservation(ctx, raw)
		if err != nil {
			raw.finish(testpilot.EffectResult{}, err)
			cleanupCtx, cancel := s.host.cleanupContext()
			for _, handle := range result {
				_ = handle.Cancel(cleanupCtx)
				delete(s.reservations, handle.Identity().ID)
			}
			cancel()
			return nil, err
		}
		s.reservations[identity.ID] = raw
		result = append(result, retained)
	}
	return result, nil
}

func (*Session) InvokeRPC(context.Context, testpilot.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (testpilot.EffectHandle, error) {
	return nil, ErrUnsupportedOperation
}

func (*Session) InvokeCapability(context.Context, testpilot.Coordinate, testpilot.OpaqueCapability, proto.Message) (testpilot.EffectHandle, error) {
	return nil, ErrUnsupportedOperation
}

// faultEffect carries the settled outcome of one fault instruction. The transition is performed
// before the handle is returned, so Wait reports what actually happened rather than a promise.
type faultEffect struct{ result testpilot.EffectResult }

func (e faultEffect) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	if ctx == nil {
		return testpilot.EffectResult{}, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return testpilot.EffectResult{}, err
	}
	return e.result, nil
}
func (faultEffect) Cancel(context.Context) error { return nil }
func (faultEffect) Drain(context.Context) error  { return nil }

// InjectFault realizes one deliberate worker outage on the dedicated group this Run holds. A
// transition the Driver cannot make is reported as a failed instruction outcome plus a Driver
// invariant diagnostic: the Run records that the fault was requested and not realized, and the
// Verdict is left to the Contract rather than being decided here.
func (s *Session) InjectFault(ctx context.Context, at testpilot.Coordinate, roleID string, kind testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	if s == nil || ctx == nil || at.RunID != s.runID || roleID == "" {
		return nil, ErrInvalid
	}
	queue, declared := s.definition.faultQueues[roleID]
	if !declared {
		return nil, ErrInvalid
	}
	if err := s.mu.lock(ctx); err != nil {
		return nil, err
	}
	closed, failure, workers := s.closed, s.failure, s.workers
	s.mu.unlock()
	if closed || failure != nil {
		return nil, errors.Join(ErrClosed, failure)
	}
	if workers == nil {
		return nil, ErrInvalid
	}
	var err error
	switch kind {
	case testpilotspb.FAULT_KIND_WORKER_STOP:
		err = workers.stopWorker(ctx, queue)
	case testpilotspb.FAULT_KIND_WORKER_RESUME:
		err = workers.resumeWorker(ctx, queue)
	default:
		return nil, ErrInvalid
	}
	// A queue this lease does not hold is a rejected dispatch, not a recorded outage.
	if errors.Is(err, ErrInvalid) || errors.Is(err, ErrUnsupportedOperation) {
		return nil, err
	}
	if err == nil {
		return faultEffect{result: testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}}, nil
	}
	if diagnoseErr := s.Diagnose(ctx, s.runID, &testpilotspb.RunDiagnostic{
		DiagnosticId: fmt.Sprintf("fault-%d", s.next.Add(1)),
		Kind:         testpilotspb.RUN_DIAGNOSTIC_KIND_INVARIANT,
		Code:         "fault_not_realized",
		Detail:       kind.String() + " on " + queue + ": " + err.Error(),
	}); diagnoseErr != nil {
		return nil, errors.Join(err, diagnoseErr)
	}
	return faultEffect{result: testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS}}}, nil
}

func (s *Session) Bridge(ctx context.Context) (testpilot.CapabilityBridge, error) {
	if s == nil || ctx == nil {
		return nil, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if nilValue(s.options.Bridge) {
		return nil, ErrInvalid
	}
	return s.options.Bridge, nil
}

func (s *Session) Quarantine(ctx context.Context, handle testpilot.EffectHandle) error {
	if s == nil || s.options.Quarantine == nil {
		return ErrInvalid
	}
	return s.ledger.Quarantine(ctx, handle, func(ctx context.Context, handle testpilot.EffectHandle, complete delivery.CompletionFunc) error {
		return s.options.Quarantine(ctx, handle, func() { complete() })
	})
}

func (s *Session) Close(ctx context.Context) error {
	if s == nil || ctx == nil {
		return ErrInvalid
	}
	if err := s.closeMu.lock(ctx); err != nil {
		return err
	}
	defer s.closeMu.unlock()
	if err := s.mu.lock(ctx); err != nil {
		return err
	}
	s.closed = true
	workers := s.workers
	s.mu.unlock()
	if !s.stopComplete {
		if _, err := s.ledger.Stop(ctx); err != nil {
			return err
		}
		s.stopComplete = true
	}
	// A release that could not resume a stopped worker is still a completed release: the session
	// is removed either way and the failure is returned, so cleanup is reported failed rather than
	// leaving the session registered behind an error.
	// release always reaches the registry, so the hold is gone even when the resume it attempted
	// first could not finish; the failure is returned and cleanup records it.
	var releaseErr error
	if !s.released && workers != nil {
		releaseErr = workers.release(ctx)
		s.released = true
	}
	if !s.removed {
		if err := s.host.removeSession(ctx, s, true); err != nil {
			return errors.Join(releaseErr, err)
		}
		s.removed = true
	}
	return releaseErr
}

func (s *Session) Diagnose(ctx context.Context, runID string, diagnostic *testpilotspb.RunDiagnostic) error {
	if s == nil || ctx == nil || runID != s.runID || diagnostic == nil {
		return ErrInvalid
	}
	if err := s.mu.lock(ctx); err != nil {
		return err
	}
	if s.diagnostics >= boundedInt(s.definition.snapshot.GetLimits().GetMaxRunEvents()) {
		s.mu.unlock()
		return ErrCapacity
	}
	s.diagnostics++
	sink := s.options.Diagnose
	s.mu.unlock()
	if sink == nil {
		return nil
	}
	return sink(ctx, runID, proto.CloneOf(diagnostic))
}

func (s *Session) workerFailed(queue string, failure error) {
	if failure == nil {
		return
	}
	ctx, cancel := s.host.cleanupContext()
	defer cancel()
	if s.mu.lock(ctx) != nil {
		return
	}
	if s.failure == nil && s.dependsOnQueue(queue) {
		s.failure = failure
	}
	s.mu.unlock()
}

func (s *Session) dependsOnQueue(queue string) bool {
	for _, registration := range s.definition.registrations {
		if registration.queue == queue {
			return true
		}
	}
	return false
}

func (s *Session) rawReservation(id string) (*reservation, error) {
	if s.mu.lock(context.Background()) != nil {
		return nil, ErrClosed
	}
	defer s.mu.unlock()
	raw := s.reservations[id]
	if raw == nil {
		return nil, ErrClosed
	}
	return raw, nil
}

func (s *Session) finishActivation(activation delivery.Activation, outcome *testpilotspb.InstructionOutcome, err error) {
	raw, lookupErr := s.rawReservation(activation.Reservation().ID)
	if lookupErr == nil {
		raw.finish(testpilot.EffectResult{Outcome: outcome}, err)
	}
}

func (h *Driver) removeSession(ctx context.Context, session *Session, tombstone bool) error {
	if h == nil || session == nil || ctx == nil {
		return ErrInvalid
	}
	if err := h.mu.lock(ctx); err != nil {
		return err
	}
	if h.sessions[session.runID] != session {
		h.mu.unlock()
		return nil
	}
	delete(h.sessions, session.runID)
	if !tombstone || h.options.diagnostics == 0 {
		h.removeRouteIndexesLocked(session)
	} else {
		if len(h.tombstones) == h.options.diagnostics {
			h.evictOldestTombstoneLocked()
		}
		h.tombstones = append(h.tombstones, session)
	}
	h.mu.unlock()
	return nil
}
