package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type programDefinition struct {
	snapshot *testpilotspb.Program
	// limits is the Profile's Program ceilings the Program was prepared under.
	limits        *testpilotspb.ProgramLimits
	entries       map[string]entryDefinition
	registrations []queueRegistration
	endpoints     map[string]string
	// queues are the task queues the task-queue roles an activity schedule names bind.
	queues          map[string]string
	queueWorkflows  map[string]map[string]struct{}
	queueActivities map[string]map[string]struct{}
	outages         OutagePlan
	hasAsync        bool
}

type entryDefinition struct {
	plan                testpilot.EntrypointPlan
	namespace           string
	queue, workflowType string
	activityType        string
	service, operation  string
}

type Session struct {
	host            *Driver
	mu              primitive.Mutex
	closeMu         primitive.Mutex
	runID           string
	id              string
	definition      programDefinition
	ledger          *delivery.Ledger
	options         SessionOptions
	reservations    map[string]*reservation
	carriers        map[testpilot.Coordinate]*Carrier
	nexusResults    map[string]*nexusResult
	activityAnswers map[string]*activityAnswer
	activityScripts map[activityScriptKey]*activityScript
	activityWatches map[activityScriptKey]struct{}
	// watching ends with the Session and bounds what the Session waits on the server for.
	watching      context.Context
	stopWatching  context.CancelFunc
	workflowKeys  map[delivery.WorkflowBinding]struct{}
	activityKeys  map[delivery.ActivityBinding]struct{}
	nexusKeys     map[nexusRouteIndex]struct{}
	nexusDispatch map[nexusDispatchKey]nexus.Header
	// activityDispatch is the header entry each schedule command of an admitted workflow carries to
	// the activity it reaches, and scheduledKeys the Driver index entries those entries hold.
	activityDispatch   map[nexusDispatchKey]delivery.ActivityDispatch
	scheduledKeys      map[scheduledRouteIndex]struct{}
	workflowAdmissions map[workflowAdmissionKey]*workflowAdmission
	nexusAdmissions    map[nexusRouteIndex]nexusAdmission
	next               atomic.Uint64
	diagnostics        int
	closed             bool
	failure            error
	outage             *Outage
	stopComplete       bool
	released           bool
	removed            bool
}

func newSession(host *Driver, runID, sessionID string, definition programDefinition, options SessionOptions) (*Session, error) {
	if host == nil || definition.snapshot == nil || definition.limits == nil {
		return nil, ErrInvalid
	}
	limits := definition.limits
	ledger, err := delivery.New(delivery.Config{RunID: runID, SessionID: sessionID, Limits: delivery.Limits{
		MaxRoutes: boundedInt(limits.GetMaxActivations()), MaxHandles: boundedInt(limits.GetMaxActivations()),
		MaxHeaderBytes: boundedInt(limits.GetMaxRequestBytes()), MaxDiagnostics: boundedInt(limits.GetMaxRunEvents()),
	}})
	if err != nil {
		return nil, err
	}
	watching, stopWatching := context.WithCancel(context.Background())
	return &Session{host: host, mu: primitive.NewMutex(), closeMu: primitive.NewMutex(), runID: runID, id: sessionID, definition: definition, ledger: ledger, options: options,
		reservations: make(map[string]*reservation), carriers: make(map[testpilot.Coordinate]*Carrier), nexusResults: make(map[string]*nexusResult),
		activityAnswers: make(map[string]*activityAnswer), activityScripts: make(map[activityScriptKey]*activityScript),
		activityWatches: make(map[activityScriptKey]struct{}), watching: watching, stopWatching: stopWatching,
		workflowKeys: make(map[delivery.WorkflowBinding]struct{}), activityKeys: make(map[delivery.ActivityBinding]struct{}), nexusKeys: make(map[nexusRouteIndex]struct{}), nexusDispatch: make(map[nexusDispatchKey]nexus.Header),
		activityDispatch: make(map[nexusDispatchKey]delivery.ActivityDispatch), scheduledKeys: make(map[scheduledRouteIndex]struct{}),
		workflowAdmissions: make(map[workflowAdmissionKey]*workflowAdmission), nexusAdmissions: make(map[nexusRouteIndex]nexusAdmission)}, nil
}

func (s *Session) Reserve(ctx context.Context, request testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	if s == nil || ctx == nil || request.Origin.RunID != s.runID || request.EntrypointID == "" || request.Count <= 0 {
		return nil, ErrInvalid
	}
	entry, exists := s.definition.entries[request.EntrypointID]
	if !exists || entry.plan.Kind() != testpilot.WorkflowEntrypoint && entry.plan.Kind() != testpilot.ActivityEntrypoint && entry.plan.Kind() != testpilot.NexusHandlerEntrypoint {
		return nil, ErrInvalid
	}
	if err := s.mu.LockContext(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	defer s.mu.Unlock()
	if s.closed || s.failure != nil {
		return nil, errors.Join(ErrClosed, s.failure)
	}
	maximum := boundedInt(s.definition.limits.GetMaxActivations())
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

func (*Session) PollRPC(context.Context, testpilot.Coordinate, string, protoreflect.MethodDescriptor, proto.Message, time.Duration, testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	return nil, ErrUnsupportedOperation
}

func (*Session) InvokeHandle(context.Context, testpilot.Coordinate, testpilot.OpaqueHandle, proto.Message) (testpilot.EffectHandle, error) {
	return nil, ErrUnsupportedOperation
}

// faultEffect settles one fault instruction. The state flip already happened on the dispatch
// path; the blocking stop or resume runs here, where the scheduler's own deadline handling turns
// an expired instruction bound into a timed-out instruction.
type faultEffect struct {
	work   Settle
	result testpilot.EffectResult
	once   sync.Once
	done   chan struct{}
	err    error
}

func newFaultEffect(work Settle, result testpilot.EffectResult) *faultEffect {
	return &faultEffect{work: work, result: result, done: make(chan struct{})}
}

func (e *faultEffect) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	if ctx == nil {
		return testpilot.EffectResult{}, ErrInvalid
	}
	if e.work == nil {
		if err := ctx.Err(); err != nil {
			return testpilot.EffectResult{}, err
		}
		return e.result, nil
	}
	e.once.Do(func() {
		e.err = e.work(ctx)
		close(e.done)
	})
	if e.err == nil {
		return e.result, nil
	}
	// A deadline is the scheduler's to classify; every other failure is a fault the Driver could
	// not realize, which the Run carries as a non-success outcome naming the reason.
	if ctx.Err() != nil {
		return testpilot.EffectResult{}, e.err
	}
	// CONSIDER(umpire): a transition that cannot complete records no Driver invariant diagnostic
	// here, unlike one refused at dispatch, although the package README states both do.
	return testpilot.EffectResult{Outcome: unrealizedFault(e.err)}, nil
}

// Cancel does not interrupt a transition that is already under way: the group's recorded state
// has already flipped, and release is what puts a stopped worker back.
func (e *faultEffect) Cancel(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	return ctx.Err()
}

// Drain waits for the transition to settle so the scheduler never treats an in-flight stop or
// resume as finished.
func (e *faultEffect) Drain(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if e.work == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done:
		return nil
	}
}

func unrealizedFault(cause error) *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{
		Status:       testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE,
		ProtocolCode: "fault_not_realized",
		Detail:       cause.Error(),
	}
}

// InjectFault realizes one deliberate worker outage on the dedicated group this Run holds. A
// transition the Driver refuses outright, or cannot complete, is reported as a failed instruction
// outcome naming the reason plus a Driver invariant diagnostic: the Run records that the fault was
// requested and not realized, and the Verdict is left to the Contract rather than decided here.
func (s *Session) InjectFault(ctx context.Context, at testpilot.Coordinate, roleID string, kind testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	if s == nil || ctx == nil || at.RunID != s.runID || roleID == "" {
		return nil, ErrInvalid
	}
	queue, _, err := s.definition.outages.resolve(roleID, kind)
	if err != nil {
		return nil, err
	}
	if err := s.mu.LockContext(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	closed, failure, outage := s.closed, s.failure, s.outage
	s.mu.Unlock()
	if closed || failure != nil {
		return nil, errors.Join(ErrClosed, failure)
	}
	if outage == nil {
		return nil, ErrInvalid
	}
	settle, err := outage.Begin(ctx, roleID, kind)
	if err == nil {
		return newFaultEffect(settle, testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}), nil
	}
	// A queue this lease does not hold is a rejected dispatch, never a recorded outage.
	if errors.Is(err, ErrInvalid) || errors.Is(err, ErrUnsupportedOperation) || ctx.Err() != nil {
		return nil, err
	}
	s.diagnoseFault(ctx, kind, queue, err)
	return newFaultEffect(nil, testpilot.EffectResult{Outcome: unrealizedFault(err)}), nil
}

// diagnoseFault notifies the Driver's own diagnostic sink. It is best effort and runs on a context
// the instruction's deadline cannot cancel: the Run already carries the reason on the outcome, so
// a sink that is unavailable must not turn an unrealized fault into a failed dispatch.
func (s *Session) diagnoseFault(ctx context.Context, kind testpilotspb.FaultKind, queue string, cause error) {
	notify, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultCleanupTimeout)
	defer cancel()
	_ = s.Diagnose(notify, s.runID, &testpilotspb.RunDiagnostic{
		DiagnosticId: fmt.Sprintf("fault-%d", s.next.Add(1)),
		Kind:         testpilotspb.RUN_DIAGNOSTIC_KIND_INVARIANT,
		Code:         "fault_not_realized",
		Detail:       kind.String() + " on " + queue + ": " + cause.Error(),
	})
}

func (s *Session) Bridge(ctx context.Context) (testpilot.HandleBridge, error) {
	if s == nil || ctx == nil {
		return nil, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if primitive.NilValue(s.options.Bridge) {
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
	if err := s.closeMu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	defer s.closeMu.Unlock()
	if err := s.mu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	s.closed = true
	outage := s.outage
	s.mu.Unlock()
	s.stopWatching()
	if !s.stopComplete {
		if _, err := s.ledger.Stop(ctx); err != nil {
			return err
		}
		s.stopComplete = true
	}
	// Restore always reaches the registry, so the hold is gone even when the resume it attempted
	// first could not finish; the session is removed either way and the failure is returned, so
	// cleanup is reported failed rather than leaving the session registered behind an error.
	var releaseErr error
	if !s.released && outage != nil {
		releaseErr = outage.Restore(ctx)
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
	if err := s.mu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	if s.diagnostics >= boundedInt(s.definition.limits.GetMaxRunEvents()) {
		s.mu.Unlock()
		return ErrCapacity
	}
	s.diagnostics++
	sink := s.options.Diagnose
	s.mu.Unlock()
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
	if s.mu.LockContext(ctx, ErrInvalid) != nil {
		return
	}
	if s.failure == nil && s.dependsOnQueue(queue) {
		s.failure = failure
	}
	s.mu.Unlock()
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
	if s.mu.LockContext(context.Background(), ErrInvalid) != nil {
		return nil, ErrClosed
	}
	defer s.mu.Unlock()
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

// watchActivityClosure asks the server, once per started activity, to say when the activity run
// closes, and then settles every declared attempt still reserved after the last one delivered as
// not needed. What the worker offered is not evidence of that: the SDK sends the answer afterwards,
// the send can fail, and the server can time the attempt out and issue the next, which must then
// find its reservation. Nothing is watched while no later attempt is reserved.
func (s *Session) watchActivityClosure(input delivery.ActivityDelivery, settled delivery.Activation) {
	closed := s.host.options.activityClosed
	if closed == nil {
		return
	}
	own := settled.Reservation()
	key := activityScriptKey{origin: own.Origin, entrypointID: own.EntrypointID}
	if err := s.mu.LockContext(s.watching, ErrInvalid); err != nil {
		return
	}
	_, watched := s.activityWatches[key]
	reserved := false
	for _, raw := range s.reservations {
		if raw.identity.Origin == own.Origin && raw.identity.EntrypointID == own.EntrypointID && raw.identity.Ordinal > own.Ordinal && !raw.settled() {
			reserved = true
		}
	}
	if s.closed || watched || !reserved {
		s.mu.Unlock()
		return
	}
	s.activityWatches[key] = struct{}{}
	s.mu.Unlock()
	go func() {
		activityRunID := settled.TemporalRunID()
		closedRunID, err := closed(s.watching, input.Namespace, input.ActivityID, activityRunID)
		switch {
		case s.watching.Err() != nil:
		case err != nil:
			s.diagnoseClosure("activity_closure_unobserved", fmt.Sprintf("activity run %q: %v", activityRunID, err))
		case closedRunID != activityRunID:
			// The outcome of another run says nothing about this one, whose later attempts may
			// still come.
			s.diagnoseClosure("activity_closure_crossed", fmt.Sprintf("activity run %q: the server reported the outcome of run %q", activityRunID, closedRunID))
		default:
			s.releaseActivityAttempts(settled)
		}
	}()
}

// diagnoseClosure tells the Run that the server did not say whether an activity closed, so its
// later declared attempts stay reserved. It is best effort.
func (s *Session) diagnoseClosure(code, detail string) {
	notify, cancel := context.WithTimeout(s.watching, defaultCleanupTimeout)
	defer cancel()
	_ = s.Diagnose(notify, s.runID, &testpilotspb.RunDiagnostic{
		DiagnosticId: fmt.Sprintf("activity-closure-%d", s.next.Add(1)),
		Kind:         testpilotspb.RUN_DIAGNOSTIC_KIND_INVARIANT,
		Code:         code,
		Detail:       boundedText(detail),
	})
}

// releaseActivityAttempts settles, as not needed, the declared attempts of a closed activity that
// no attempt was delivered for. Each records the activity run and neither an SDK attempt nor a
// delivery: Temporal may never have created an attempt for the position.
func (s *Session) releaseActivityAttempts(of delivery.Activation) {
	released, err := s.ledger.ReleaseActivityAttempts(s.watching, of)
	if err != nil {
		return
	}
	for _, identity := range released {
		raw, err := s.rawReservation(identity.ID)
		if err != nil {
			continue
		}
		raw.finish(testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED, ActivityAttempt: &testpilotspb.ActivityAttempt{
			ActivityRunId: of.TemporalRunID(), Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED,
		}}}, nil)
	}
}

func (h *Driver) removeSession(ctx context.Context, session *Session, tombstone bool) error {
	if h == nil || session == nil || ctx == nil {
		return ErrInvalid
	}
	if err := h.mu.LockContext(ctx, ErrInvalid); err != nil {
		return err
	}
	if h.sessions[session.runID] != session {
		h.mu.Unlock()
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
	h.mu.Unlock()
	return nil
}
