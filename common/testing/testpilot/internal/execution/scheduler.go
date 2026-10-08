package execution

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type scheduler struct {
	values        *valueStore
	recorder      *recorder
	session       contract.Session
	mu            sync.Mutex
	started       bool
	owned         []contract.EffectHandle
	cancellations []context.CancelFunc
	reservations  map[string]bool
	// attemptFacts holds, per activity a carrier started, its latest recorded attempt.
	attemptFacts     map[string]priorAttempt
	attempts         int64
	completions      chan schedulerCompletion
	waits            sync.WaitGroup
	pending          int
	ordinaryCanceled bool
	cleanupCanceled  bool
	closing          bool
	closed           chan struct{}
	lateTimeout      time.Duration
	// runEventOrdinals count the evidence lifted per source out of recorded Run Events.
	evidenceMu       sync.Mutex
	runEventOrdinals map[string]int64
}
type scheduledActivation struct {
	values    *activationValues
	ordinal   int
	cleanup   bool
	remaining []int
	completed []string
	pending   int
}
type scheduledNode struct {
	activation *scheduledActivation
	index      int
}
type scheduledReservation struct {
	handle   contract.ReservationHandle
	identity contract.ReservationIdentity
	source   string
	cause    string
	// values is the activation of the instruction that carried the reservation, under whose work
	// ceiling the reservation's record is lifted into evidence.
	values *activationValues
}
type schedulerCompletion struct {
	node        *scheduledNode
	reservation *scheduledReservation
	result      contract.EffectResult
	err         error
	cleanup     bool
}

func newScheduler(p *PreparedProgram, runID, caseID string, session contract.Session, monitor Monitor, now func() time.Time) (*scheduler, error) {
	if ir.IsNil(session) {
		return nil, ir.Invalid(ir.Malformed, "scheduler", "Session required")
	}
	values, err := newValueStore(p, runID)
	if err != nil {
		return nil, err
	}
	recorder, err := newRecorder(p.view, runID, caseID, monitor, now, values.seal, session.Diagnose)
	if err != nil {
		return nil, err
	}
	return &scheduler{values: values, recorder: recorder, session: session, reservations: map[string]bool{}, attemptFacts: map[string]priorAttempt{}, completions: make(chan schedulerCompletion, int(p.limits.MaxNodes+p.limits.MaxActivations)), closed: make(chan struct{}), lateTimeout: time.Duration(p.limits.MaxCleanupDurationMilliseconds) * time.Millisecond, runEventOrdinals: map[string]int64{}}, nil
}

// performsNothing reports whether the named entrypoint of the source Program carries no instruction
// at all. The source is read rather than a prepared graph because a worker entrypoint's graph is
// the worker's own, and it is worker entrypoints a carrier reserves.
func (s *scheduler) performsNothing(entrypointID string) bool {
	for _, entrypoint := range s.values.program.source.GetEntrypoints() {
		if entrypoint.GetEntrypointId() == entrypointID {
			return len(entrypoint.GetInstructions()) == 0
		}
	}
	return false
}

// entrypointKind is the kind of the named entrypoint of the source Program, read from the source for
// the reason performsNothing gives.
func (s *scheduler) entrypointKind(entrypointID string) contract.EntrypointKind {
	for _, entrypoint := range s.values.program.source.GetEntrypoints() {
		if entrypoint.GetEntrypointId() == entrypointID {
			return contract.EntrypointKindOf(entrypoint)
		}
	}
	return 0
}

// attemptNumbering is how the named activity entrypoint of the source Program numbers its attempts,
// read from the source for the reason performsNothing gives; nil for any other entrypoint.
func (s *scheduler) attemptNumbering(entrypointID string) *testpilotspb.AttemptNumbering {
	for _, entrypoint := range s.values.program.source.GetEntrypoints() {
		if entrypoint.GetEntrypointId() == entrypointID {
			return entrypoint.GetActivity().GetAttemptNumbering()
		}
	}
	return nil
}

// reservationVerdict is what the Run does with the outcome a reservation settles with.
type reservationVerdict uint8

const (
	// reservationRejected fails the Run and records nothing: no Driver may report the outcome.
	reservationRejected reservationVerdict = iota
	reservationRecorded
	// reservationRecordedThenFailed records the outcome and then makes the Run incomplete, so the
	// record is never replaced by the failure it caused.
	reservationRecordedThenFailed
)

// attemptFact says what an outcome must say about an activity attempt.
type attemptFact uint8

const (
	// noAttempt outcomes carry no activity attempt.
	noAttempt attemptFact = iota
	// deliveredAttempt outcomes name the activity run, the SDK attempt of the reservation's position
	// and the delivery the worker ran.
	deliveredAttempt
	// undeliveredAttempt outcomes name the activity run alone, and follow a recorded attempt of the
	// same activity in the same run: Temporal may never have created an attempt for the position.
	undeliveredAttempt
)

// priorAttempt is the latest recorded attempt of one activity: the source of its Run Event, and the
// activity run it named, which every later outcome of the activity must name too.
type priorAttempt struct {
	source        string
	activityRunID string
}

type reservationOutcome struct {
	kind     contract.EntrypointKind
	status   testpilotspb.InstructionOutcomeStatus
	response testpilotspb.ActivityAttemptResponse
}

type reservationRule struct {
	verdict reservationVerdict
	attempt attemptFact
	// unusedOnly admits the outcome only of an entrypoint that performs nothing: the Program
	// reserved it because its carrier can activate the entrypoint, not because the Case needs it to
	// run, so the reservation released unconsumed when its parent finished is recorded as canceled
	// rather than failing the Run.
	unusedOnly bool
}

// reservationOutcomes is every outcome a reservation may settle with, by the kind of entrypoint it
// activates. Anything absent, an unset or unknown enum value included, is rejected.
var reservationOutcomes = map[reservationOutcome]reservationRule{
	{kind: contract.WorkflowEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}:     {verdict: reservationRecorded},
	{kind: contract.WorkflowEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED}:      {verdict: reservationRecorded, unusedOnly: true},
	{kind: contract.NexusHandlerEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}: {verdict: reservationRecorded},
	{kind: contract.NexusHandlerEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED}:  {verdict: reservationRecorded, unusedOnly: true},
	{kind: contract.ActivityEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED}: {
		verdict: reservationRecorded, attempt: deliveredAttempt,
	},
	{kind: contract.ActivityEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE}: {
		verdict: reservationRecorded, attempt: deliveredAttempt,
	},
	{kind: contract.ActivityEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_NON_RETRYABLE}: {
		verdict: reservationRecorded, attempt: deliveredAttempt,
	},
	{kind: contract.ActivityEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_CANCELED}: {
		verdict: reservationRecorded, attempt: deliveredAttempt,
	},
	{kind: contract.ActivityEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_WITHHELD}: {
		verdict: reservationRecorded, attempt: deliveredAttempt,
	},
	{kind: contract.ActivityEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING}: {
		verdict: reservationRecorded, attempt: deliveredAttempt,
	},
	{kind: contract.ActivityEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED}: {
		verdict: reservationRecordedThenFailed, attempt: deliveredAttempt,
	},
	{kind: contract.ActivityEntrypoint, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED, response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED}: {
		verdict: reservationRecorded, attempt: undeliveredAttempt,
	},
}

// judgeReservation reads reservationOutcomes for the outcome of the reservation at ordinal of an
// entrypoint of the kind. unused says the entrypoint performs nothing, and priorRun is the activity
// run the attempts of the same activity recorded so far named, empty when none is recorded. An
// activity's attempts are numbered as its entrypoint declares: the reservation at ordinal is the
// attempt numbered first + ordinal, and where every attempt is of one run, an outcome that names
// another is crossed and rejected.
func judgeReservation(kind contract.EntrypointKind, numbering *testpilotspb.AttemptNumbering, unused bool, ordinal int64, priorRun string, outcome *testpilotspb.InstructionOutcome) reservationVerdict {
	attempt := outcome.GetActivityAttempt()
	// An outcome the table does not list reads as the zero rule, whose verdict is rejection.
	rule := reservationOutcomes[reservationOutcome{kind: kind, status: outcome.GetStatus(), response: attempt.GetResponse()}]
	if rule.unusedOnly && !unused {
		return reservationRejected
	}
	sameRun := !numbering.GetOneRun() || priorRun == "" || attempt.GetActivityRunId() == priorRun
	var valid bool
	switch rule.attempt {
	case deliveredAttempt:
		valid = attempt.GetActivityRunId() != "" && sameRun && numbering.GetFirst() > 0 && int64(attempt.GetSdkAttempt()) == numbering.GetFirst()+ordinal && attempt.GetDeliveryId() != ""
	case undeliveredAttempt:
		valid = priorRun != "" && attempt.GetActivityRunId() != "" && sameRun && attempt.GetSdkAttempt() == 0 && attempt.GetDeliveryId() == "" && !attempt.GetHeartbeatInvoked()
	default:
		valid = attempt == nil
	}
	if !valid {
		return reservationRejected
	}
	return rule.verdict
}

func (s *scheduler) outstanding() []contract.EffectHandle {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]contract.EffectHandle(nil), s.owned...)
}
func (s *scheduler) retain(handles []contract.EffectHandle) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, h := range handles {
		if !ir.IsNil(h) {
			s.owned = append(s.owned, h)
		}
	}
}
func (s *scheduler) fail(code string, err error) error {
	return s.recorder.fail(testpilotspb.RUN_DIAGNOSTIC_KIND_EXECUTION, code, err)
}

// execute leaves accepted handles and buffered completions owned when Stop transfers control to cleanup.
func (s *scheduler) execute(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return ir.Invalid(ir.Unavailable, "scheduler", "controller schedule already started")
	}
	s.started = true
	s.mu.Unlock()
	decision, err := s.recorder.publish(ctx, []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "scheduler.open"}}, nil)
	if err != nil || decision == Stop {
		return err
	}
	ready, decision, err := s.openControllers(ctx)
	if err != nil || decision == Stop {
		return err
	}
	active := 0
	for len(ready) > 0 || active > 0 {
		count, decision, err := s.dispatchReady(ctx, ready, false)
		active += count
		ready = nil
		if err != nil || decision == Stop {
			return err
		}
		if active == 0 {
			break
		}
		completion, ok := s.takeCompletion(ctx, s.recorder.halted)
		if !ok {
			select {
			case <-s.recorder.halted:
				return s.recorder.schedulingFailure()
			default:
				return s.fail("schedule_cancelled", ctx.Err())
			}
		}
		if !completion.cleanup {
			active--
		}
		ready, decision, err = s.acceptCompletion(ctx, completion, false)
		if err != nil || decision == Stop {
			return err
		}
	}
	return nil
}

func (s *scheduler) executeCleanup(ctx context.Context) (result error) {
	g := s.values.program.cleanupGraph()
	values, err := s.values.activate(g.id, "cleanup")
	if err != nil {
		return err
	}
	activation := &scheduledActivation{values: values, cleanup: true, remaining: make([]int, len(g.nodes)), completed: make([]string, len(g.nodes)), pending: len(g.nodes)}
	coordinates := &testpilotspb.RunEventCoordinates{EntrypointId: g.id, ActivationId: values.id}
	_, err = s.recorder.publishCleanup(ctx, []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_CLEANUP_STARTED, SourceId: "scheduler.cleanup.open", Coordinates: coordinates}}, nil)
	if err != nil {
		return err
	}
	defer func() {
		causes := []string{"scheduler.cleanup.open"}
		for _, source := range activation.completed {
			if source != "" {
				causes = append(causes, source)
			}
		}
		_, err := s.recorder.publishCleanup(ctx, []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_CLEANUP_COMPLETED, SourceId: "scheduler.cleanup.close", CausalSourceIds: causes, Coordinates: coordinates}}, nil)
		result = errors.Join(result, err)
	}()
	ready := make([]scheduledNode, 0, len(g.nodes))
	for _, index := range g.order {
		activation.remaining[index] = len(g.nodes[index].dependencies)
		if activation.remaining[index] == 0 {
			ready = append(ready, scheduledNode{activation: activation, index: index})
		}
	}
	active := 0
	for len(ready) > 0 || active > 0 {
		count, _, err := s.dispatchReady(ctx, ready, true)
		active += count
		ready = nil
		if err != nil {
			return err
		}
		if active == 0 {
			break
		}
		completion, ok := s.takeCompletion(ctx, nil)
		if !ok {
			return ctx.Err()
		}
		if !completion.cleanup {
			_ = s.publishSettledCompletion(ctx, completion)
			continue
		}
		active--
		ready, _, err = s.acceptCompletion(ctx, completion, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *scheduler) ownedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.owned)
}

func (s *scheduler) outstandingSince(index int) []contract.EffectHandle {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index > len(s.owned) {
		return nil
	}
	return append([]contract.EffectHandle(nil), s.owned[index:]...)
}

func (s *scheduler) cancelWaits() {
	s.mu.Lock()
	cancellations := append([]context.CancelFunc(nil), s.cancellations...)
	s.mu.Unlock()
	for _, cancel := range cancellations {
		cancel()
	}
}

func (s *scheduler) settle(ctx context.Context, handles []contract.EffectHandle, cleanup, cancelHandles bool) error {
	var result error
	if cancelHandles {
		s.markCanceled(cleanup)
		result = s.cancelOwned(ctx, handles)
	}
	result = errors.Join(result, s.drainOwned(ctx, handles))
	if cancelHandles {
		s.cancelWaits()
	}
	return errors.Join(result, s.settleCompletions(ctx, cleanup))
}

func (s *scheduler) markCanceled(cleanup bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cleanup {
		s.cleanupCanceled = true
	} else {
		s.ordinaryCanceled = true
	}
}

func (s *scheduler) cancelOwned(ctx context.Context, handles []contract.EffectHandle) error {
	var result error
	for _, handle := range handles {
		err := handle.Cancel(ctx)
		code := "effect_cancel_failed"
		if err == nil {
			err = ctx.Err()
			code = "effect_cancel_context_violated"
		}
		if err != nil {
			result = errors.Join(result, err)
			s.recorder.report(testpilotspb.RUN_DIAGNOSTIC_KIND_DRIVER_CONTRACT, code, err)
		}
	}
	return result
}

func (s *scheduler) drainOwned(ctx context.Context, handles []contract.EffectHandle) error {
	var result error
	for _, handle := range handles {
		drainErr := handle.Drain(ctx)
		if drainErr == nil && ctx.Err() != nil {
			drainErr = ctx.Err()
			result = errors.Join(result, drainErr)
			s.recorder.report(testpilotspb.RUN_DIAGNOSTIC_KIND_DRIVER_CONTRACT, "effect_drain_context_violated", drainErr)
		}
		if drainErr != nil {
			result = errors.Join(result, s.quarantine(handle, drainErr))
		}
	}
	return result
}

func (s *scheduler) quarantine(handle contract.EffectHandle, drainErr error) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.lateTimeout)
	defer cancel()
	err := s.session.Quarantine(ctx, handle)
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		s.recorder.report(testpilotspb.RUN_DIAGNOSTIC_KIND_DRIVER_CONTRACT, "quarantine_failed", err)
		return err
	}
	s.recorder.report(testpilotspb.RUN_DIAGNOSTIC_KIND_LIMIT, "effect_quarantined", drainErr)
	return nil
}

func (s *scheduler) settleCompletions(ctx context.Context, cleanup bool) error {
	var result error
	for {
		s.mu.Lock()
		pending := s.pending
		s.mu.Unlock()
		if pending == 0 {
			break
		}
		completion, ok := s.takeCompletion(ctx, nil)
		if !ok {
			return errors.Join(result, s.drainCompletions(cleanup))
		}
		result = errors.Join(result, s.settleCompletion(ctx, completion, cleanup))
	}
	return result
}

// takeCompletion waits for the next completion and retires it from pending. It reports false when
// ctx ends or halted closes first; a nil halted never closes.
func (s *scheduler) takeCompletion(ctx context.Context, halted <-chan struct{}) (schedulerCompletion, bool) {
	select {
	case <-halted:
		return schedulerCompletion{}, false
	case <-ctx.Done():
		return schedulerCompletion{}, false
	case completion := <-s.completions:
		s.mu.Lock()
		s.pending--
		s.mu.Unlock()
		return completion, true
	}
}

// drainCompletions settles every completion already buffered, without waiting for more.
func (s *scheduler) drainCompletions(cleanup bool) error {
	var result error
	for {
		select {
		case completion := <-s.completions:
			s.mu.Lock()
			s.pending--
			s.mu.Unlock()
			result = errors.Join(result, s.settleCompletion(context.Background(), completion, cleanup))
		default:
			return result
		}
	}
}

// settleCompletion publishes a completion the schedule no longer waits for; its failure counts only
// toward the phase that dispatched it.
func (s *scheduler) settleCompletion(ctx context.Context, completion schedulerCompletion, cleanup bool) error {
	err := s.publishSettledCompletion(ctx, completion)
	if completion.cleanup != cleanup {
		return nil
	}
	return err
}

func (s *scheduler) publishSettledCompletion(ctx context.Context, completion schedulerCompletion) error {
	if s.expectedCancellation(completion) {
		return nil
	}
	parent := ctx
	if ir.IsNil(parent) || parent.Err() != nil {
		parent = context.Background()
	}
	publishCtx, cancel := context.WithTimeout(parent, s.lateTimeout)
	defer cancel()
	_, err := s.publishCompletion(publishCtx, completion)
	return err
}

func (s *scheduler) expectedCancellation(completion schedulerCompletion) bool {
	if completion.err == nil || !errors.Is(completion.err, context.Canceled) && !errors.Is(completion.err, context.DeadlineExceeded) {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if completion.cleanup {
		return s.cleanupCanceled
	}
	return s.ordinaryCanceled
}

func (s *scheduler) beginClose() {
	s.mu.Lock()
	s.closing = true
	s.mu.Unlock()
	_ = s.drainCompletions(false)
}

func (s *scheduler) finishClose() {
	close(s.closed)
}
func (s *scheduler) openControllers(ctx context.Context) ([]scheduledNode, Decision, error) {
	var decision Decision
	var ready []scheduledNode
	for ordinal, g := range s.values.program.graphs {
		if g.cleanup || g.context != contract.ControllerEntrypoint {
			continue
		}
		values, err := s.values.activate(g.id, fmt.Sprintf("controller.%d", ordinal))
		if err != nil {
			return nil, Stop, s.fail("activation_failed", err)
		}
		a := &scheduledActivation{values: values, ordinal: ordinal, remaining: make([]int, len(g.nodes)), completed: make([]string, len(g.nodes)), pending: len(g.nodes)}
		decision, err = s.recorder.publish(ctx, []*testpilotspb.RunEvent{s.activationEvent(a, false)}, nil)
		if err != nil || decision == Stop {
			return nil, decision, err
		}
		for _, index := range g.order {
			a.remaining[index] = len(g.nodes[index].dependencies)
			if a.remaining[index] == 0 {
				ready = append(ready, scheduledNode{a, index})
			}
		}
		if a.pending == 0 {
			decision, err = s.recorder.publish(ctx, []*testpilotspb.RunEvent{s.activationEvent(a, true)}, nil)
			if err != nil || decision == Stop {
				return nil, decision, err
			}
		}
	}

	return ready, Continue, nil
}
func (s *scheduler) dispatchReady(ctx context.Context, ready []scheduledNode, cleanup bool) (int, Decision, error) {
	active := 0
	for len(ready) > 0 {
		task := ready[0]
		ready = ready[1:]
		count, skipped, decision, err := s.dispatch(ctx, task, cleanup)
		active += count
		if err != nil || decision == Stop && !cleanup {
			return active, decision, err
		}
		if skipped {
			more, decision, err := s.completeNode(ctx, task, "")
			ready = append(ready, more...)
			if err != nil || decision == Stop {
				return active, decision, err
			}
		}
	}

	return active, Continue, nil
}
func (s *scheduler) acceptCompletion(ctx context.Context, completion schedulerCompletion, cleanup bool) ([]scheduledNode, Decision, error) {
	decision, err := s.publishCompletion(ctx, completion)
	if err != nil || decision == Stop && !cleanup || completion.node == nil {
		return nil, decision, err
	}
	return s.completeNode(ctx, *completion.node, s.nodeSource(*completion.node)+".completed")
}
func (s *scheduler) activationEvent(a *scheduledActivation, closed bool) *testpilotspb.RunEvent {
	kind := testpilotspb.RUN_EVENT_KIND_ACTIVATION_OPENED
	source := fmt.Sprintf("scheduler.g%d.open", a.ordinal)
	causes := []string{"scheduler.open"}
	if closed {
		kind = testpilotspb.RUN_EVENT_KIND_ACTIVATION_CLOSED
		source = fmt.Sprintf("scheduler.g%d.close", a.ordinal)
		causes = []string{fmt.Sprintf("scheduler.g%d.open", a.ordinal)}
		for _, id := range a.completed {
			if id != "" {
				causes = append(causes, id)
			}
		}
	}
	return &testpilotspb.RunEvent{Kind: kind, SourceId: source, CausalSourceIds: causes, Coordinates: &testpilotspb.RunEventCoordinates{EntrypointId: a.values.graph.id, ActivationId: a.values.id}}
}
func (s *scheduler) nodeSource(task scheduledNode) string {
	if task.activation.cleanup {
		return fmt.Sprintf("scheduler.cleanup.n%d.a1", task.index)
	}
	return fmt.Sprintf("scheduler.g%d.n%d.a1", task.activation.ordinal, task.index)
}
func (s *scheduler) coordinate(task scheduledNode) contract.Coordinate {
	return contract.Coordinate{RunID: s.values.runID, EntrypointID: task.activation.values.graph.id, ActivationID: task.activation.values.id, InstructionID: task.activation.values.graph.nodes[task.index].source.InstructionId, Attempt: 1}
}
func eventCoordinates(c contract.Coordinate) *testpilotspb.RunEventCoordinates {
	return &testpilotspb.RunEventCoordinates{EntrypointId: c.EntrypointID, ActivationId: c.ActivationID, InstructionId: c.InstructionID, Attempt: c.Attempt}
}
func (s *scheduler) completeNode(ctx context.Context, task scheduledNode, source string) ([]scheduledNode, Decision, error) {
	a := task.activation
	a.completed[task.index] = source
	a.pending--
	var ready []scheduledNode
	for _, index := range a.values.graph.nodes[task.index].successors {
		a.remaining[index]--
		if a.remaining[index] == 0 {
			ready = append(ready, scheduledNode{a, index})
		}
	}
	if a.pending == 0 {
		if a.cleanup {
			return ready, Continue, nil
		}
		d, err := s.recorder.publish(ctx, []*testpilotspb.RunEvent{s.activationEvent(a, true)}, nil)
		return ready, d, err
	}
	return ready, Continue, nil
}
func (s *scheduler) dispatch(ctx context.Context, task scheduledNode, cleanup bool) (int, bool, Decision, error) {
	a := task.activation.values
	n := a.graph.nodes[task.index]
	c := s.coordinate(task)
	request, enabled, err := s.prepareInput(ctx, task)
	if err != nil {
		return 0, false, Stop, s.dispatchFailure(cleanup, "input_failed", err)
	}
	if !enabled {
		return 0, true, Continue, nil
	}
	decision, err := s.publishInstructionStart(ctx, task, c, cleanup)
	if err != nil || decision == Stop && !cleanup {
		return 0, false, decision, err
	}
	operationCtx, cancel := context.WithTimeout(ctx, time.Duration(n.timeoutMilliseconds)*time.Millisecond)
	effect, reservations, bridge, err := s.admitDispatch(operationCtx, task, request, cleanup)
	if err != nil {
		cancel()
		return 0, false, Stop, err
	}
	s.mu.Lock()
	s.cancellations = append(s.cancellations, cancel)
	s.pending += len(reservations) + 1
	s.mu.Unlock()
	s.startWaits(ctx, operationCtx, cancel, task, effect, bridge, reservations, cleanup)
	return 1 + len(reservations), false, Continue, nil
}

func (s *scheduler) dispatchFailure(cleanup bool, code string, err error) error {
	if cleanup {
		return err
	}
	return s.fail(code, err)
}

func (s *scheduler) publishInstructionStart(ctx context.Context, task scheduledNode, coordinate contract.Coordinate, cleanup bool) (Decision, error) {
	n := task.activation.values.graph.nodes[task.index]
	causes := []string{fmt.Sprintf("scheduler.g%d.open", task.activation.ordinal)}
	if cleanup {
		causes[0] = "scheduler.cleanup.open"
	}
	for _, index := range n.dependencies {
		if source := task.activation.completed[index]; source != "" {
			causes = append(causes, source)
		}
	}
	publish := s.recorder.publish
	if cleanup {
		publish = s.recorder.publishCleanup
	}
	return publish(ctx, []*testpilotspb.RunEvent{{Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_STARTED, SourceId: s.nodeSource(task) + ".started", Coordinates: eventCoordinates(coordinate), CausalSourceIds: causes}}, nil)
}

func (s *scheduler) admitDispatch(ctx context.Context, task scheduledNode, request proto.Message, cleanup bool) (contract.EffectHandle, []scheduledReservation, contract.HandleBridge, error) {
	n := task.activation.values.graph.nodes[task.index]
	var effect contract.EffectHandle
	var reservations []scheduledReservation
	var bridge contract.HandleBridge
	admit := s.recorder.admit
	if cleanup {
		admit = s.recorder.admitCleanup
	}
	err := admit(ctx, func(ctx context.Context) ([]contract.EffectHandle, error) {
		if s.attempts >= s.values.program.limits.MaxAttempts {
			return nil, ir.Invalid(ir.LimitExceeded, "scheduler", "attempt ceiling exceeded")
		}
		s.attempts++
		accepted, reserved, err := s.reserve(ctx, task)
		reservations = reserved
		if err != nil {
			return accepted, err
		}
		if err := ctx.Err(); err != nil {
			return accepted, err
		}
		effect, bridge, err = s.acceptEffect(ctx, task, request)
		if !ir.IsNil(effect) {
			accepted = append(accepted, effect)
		} else if err == nil && n.opcode != contract.AwaitSlot {
			err = ir.Invalid(ir.Malformed, "effect", "nil effect handle")
		}
		return accepted, err
	}, s.retain)
	return effect, reservations, bridge, err
}

func (s *scheduler) startWaits(ctx, operationCtx context.Context, cancel context.CancelFunc, task scheduledNode, effect contract.EffectHandle, bridge contract.HandleBridge, reservations []scheduledReservation, cleanup bool) {
	for _, group := range s.waitGroups(reservations) {
		s.waits.Add(1)
		go func() {
			defer s.waits.Done()
			for _, reservation := range group {
				result, err := reservation.handle.Wait(ctx)
				s.deliverCompletion(schedulerCompletion{reservation: &reservation, result: result, err: err, cleanup: cleanup})
			}
		}()
	}
	s.waits.Add(1)
	go func() {
		defer s.waits.Done()
		defer cancel()
		result, err := s.waitNode(operationCtx, task, effect, bridge)
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			result = contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT}}
			err = nil
		}
		if err == nil {
			result = s.reportExpiry(task, result)
		}
		s.deliverCompletion(schedulerCompletion{node: &task, result: result, err: err, cleanup: cleanup})
	}()
}

// waitGroups splits an instruction's reservations into the groups observed independently. The
// reservations of one activity entrypoint are its attempts in order, and what a later attempt was
// depends on the earlier ones, so one waiter observes them in that order and their Run Events keep
// it. Every other reservation is a group of its own.
func (s *scheduler) waitGroups(reservations []scheduledReservation) [][]scheduledReservation {
	var groups [][]scheduledReservation
	attempts := map[string]int{}
	for _, reservation := range reservations {
		entrypointID := reservation.identity.EntrypointID
		index, grouped := attempts[entrypointID]
		if s.entrypointKind(entrypointID) != contract.ActivityEntrypoint || !grouped {
			attempts[entrypointID] = len(groups)
			groups = append(groups, []scheduledReservation{reservation})
			continue
		}
		groups[index] = append(groups[index], reservation)
	}
	for _, group := range groups {
		slices.SortFunc(group, func(left, right scheduledReservation) int {
			return cmp.Compare(left.identity.Ordinal, right.identity.Ordinal)
		})
	}
	return groups
}

// prepareInput builds an RPC node's request, or evaluates any other node's guard: only a worker's
// Finish carries an input, and no controller runs one.
func (s *scheduler) prepareInput(ctx context.Context, task scheduledNode) (proto.Message, bool, error) {
	a := task.activation.values
	n := a.graph.nodes[task.index]
	if n.opcode == contract.InvokeRPC || n.opcode == contract.ReadEvidence {
		request, enabled, _, err := a.request(ctx, s.coordinate(task), a.workLimit())
		return request, enabled, err
	}
	work, err := a.newWork(ctx, a.workLimit())
	if err != nil {
		return nil, false, err
	}
	_, enabled, err := n.evaluateGuarded(func(e *ir.Expression) (*testpilotspb.Value, error) { return a.evaluate(work, e) })
	return nil, enabled, err
}
func (s *scheduler) reserve(ctx context.Context, task scheduledNode) ([]contract.EffectHandle, []scheduledReservation, error) {
	var accepted []contract.EffectHandle
	var reservations []scheduledReservation
	n := task.activation.values.graph.nodes[task.index]
	c := s.coordinate(task)
	for declarationIndex, declaration := range n.reservations {
		request := contract.ReservationRequest{Origin: c, EntrypointID: declaration.EntrypointID, Count: declaration.Count}
		acquired, err := s.session.Reserve(ctx, request)
		for _, h := range acquired {
			if !ir.IsNil(h) {
				accepted = append(accepted, h)
			}
		}
		if err != nil {
			return accepted, reservations, err
		}
		validated, err := s.validateReservations(task, declarationIndex, request, acquired)
		if err != nil {
			return accepted, reservations, err
		}
		reservations = append(reservations, validated...)
	}

	return accepted, reservations, nil
}
func (s *scheduler) validateReservations(task scheduledNode, declarationIndex int, request contract.ReservationRequest, acquired []contract.ReservationHandle) ([]scheduledReservation, error) {
	var reservations []scheduledReservation
	if int64(len(acquired)) != request.Count {
		return nil, ir.Invalid(ir.Malformed, "reservation", "wrong reservation count")
	}
	ordinals := map[int64]bool{}
	for _, h := range acquired {
		if ir.IsNil(h) {
			return nil, ir.Invalid(ir.Malformed, "reservation", "nil reservation")
		}
		id := h.Identity()
		if !ir.ValidID(id.ID) || id.Origin != request.Origin || id.EntrypointID != request.EntrypointID || id.Ordinal < 0 || id.Ordinal >= request.Count || ordinals[id.Ordinal] || s.reservations[id.ID] {
			return nil, ir.Invalid(ir.Malformed, "reservation", "crossed or duplicate reservation identity")
		}
		ordinals[id.Ordinal] = true
		s.reservations[id.ID] = true
		reservations = append(reservations, scheduledReservation{handle: h, identity: id, source: fmt.Sprintf("%s.r%d.i%d", s.nodeSource(task), declarationIndex, id.Ordinal), cause: s.nodeSource(task) + ".started", values: task.activation.values})
	}
	return reservations, nil
}
func (s *scheduler) acceptEffect(ctx context.Context, task scheduledNode, request proto.Message) (contract.EffectHandle, contract.HandleBridge, error) {
	n := task.activation.values.graph.nodes[task.index]
	accept := opcodes[n.opcode].accept
	if accept == nil {
		return nil, nil, ir.Invalid(ir.Unsupported, "scheduler", "controller capability required")
	}
	return accept(s, ctx, task, s.coordinate(task), n, request)
}
func (s *scheduler) acceptRPC(ctx context.Context, _ scheduledNode, c contract.Coordinate, n *node, request proto.Message) (contract.EffectHandle, contract.HandleBridge, error) {
	effect, err := s.session.InvokeRPC(ctx, c, n.source.Instruction.GetInvokeRpc().EndpointRoleId, n.method, request)
	return effect, nil, err
}
func (s *scheduler) acceptReadEvidence(ctx context.Context, task scheduledNode, c contract.Coordinate, n *node, request proto.Message) (contract.EffectHandle, contract.HandleBridge, error) {
	a := task.activation.values
	// A read once was admitted with interval 0, which the Driver contract reads once.
	effect, err := s.session.PollRPC(ctx, c, n.source.Instruction.GetReadEvidence().EndpointRoleId, n.method, request, time.Duration(n.pollIntervalMilliseconds)*time.Millisecond, func(ctx context.Context, response proto.Message) (bool, error) {
		satisfied, _, err := a.readSatisfied(ctx, c, response, a.workLimit())
		return satisfied, err
	})
	return effect, nil, err
}
func (s *scheduler) acceptFault(ctx context.Context, _ scheduledNode, c contract.Coordinate, n *node, _ proto.Message) (contract.EffectHandle, contract.HandleBridge, error) {
	fault := n.source.Instruction.GetInjectFault()
	effect, err := s.session.InjectFault(ctx, c, fault.GetRoleId(), fault.GetKind())
	return effect, nil, err
}

// An AwaitSlot consumes nothing: it only takes the bridge an opaque Slot is awaited through.
func (s *scheduler) acceptAwaitSlot(ctx context.Context, _ scheduledNode, _ contract.Coordinate, n *node, _ proto.Message) (contract.EffectHandle, contract.HandleBridge, error) {
	bridge, err := s.handleBridge(ctx, n.source.Instruction.GetAwaitSlot().GetSlotId())
	return nil, bridge, err
}

// A typed completion consumes its handle Slot and delivers the payload or failure it carries.
func (s *scheduler) acceptOperationCompletion(ctx context.Context, _ scheduledNode, c contract.Coordinate, n *node, _ proto.Message) (contract.EffectHandle, contract.HandleBridge, error) {
	completion := n.source.Instruction.GetNexusOperationCompletion()
	bridge, err := s.handleBridge(ctx, completion.GetHandleSlotId())
	if err != nil {
		return nil, bridge, err
	}
	handle, err := bridge.Consume(ctx, completion.GetHandleSlotId())
	if err == nil && ir.IsNil(handle) {
		err = ir.Invalid(ir.Malformed, "bridge", "nil opaque handle")
	}
	if err != nil {
		return nil, bridge, err
	}
	effect, err := s.session.InvokeHandle(ctx, c, handle, carriedCompletion(completion))
	return effect, bridge, err
}

// handleBridge is the Session's bridge for an opaque Slot; a value Slot needs none.
func (s *scheduler) handleBridge(ctx context.Context, slot string) (contract.HandleBridge, error) {
	if !s.values.program.slots[slot].Opaque() {
		return nil, nil
	}
	bridge, err := s.session.Bridge(ctx)
	if err == nil && ir.IsNil(bridge) {
		err = ir.Invalid(ir.Malformed, "bridge", "nil bridge")
	}
	return bridge, err
}
func (s *scheduler) waitNode(ctx context.Context, task scheduledNode, effect contract.EffectHandle, bridge contract.HandleBridge) (contract.EffectResult, error) {
	a := task.activation.values
	n := a.graph.nodes[task.index]
	var result contract.EffectResult
	var err error
	if n.opcode == contract.AwaitSlot {
		slot := n.source.Instruction.GetAwaitSlot().SlotId
		if bridge != nil {
			err = bridge.Await(ctx, slot)
		} else {
			err = a.awaitSlot(ctx, slot)
		}
		if err == nil {
			result.Outcome = &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
		}
	} else {
		result, err = effect.Wait(ctx)
	}

	return result, err
}
func (s *scheduler) publishCompletion(ctx context.Context, completion schedulerCompletion) (Decision, error) {
	if completion.err != nil {
		if completion.cleanup {
			return Stop, completion.err
		}
		return Stop, s.recorder.completionFailure(ctx, "effect_wait_failed", completion.err)
	}
	if completion.reservation != nil {
		reservation := completion.reservation
		id := reservation.identity
		outcome := completion.result.Outcome
		// The attempts of one activity share every part of their source but the ordinal.
		activity := reservation.source[:strings.LastIndex(reservation.source, ".")]
		prior := s.attemptFacts[activity]
		verdict := judgeReservation(s.entrypointKind(id.EntrypointID), s.attemptNumbering(id.EntrypointID), s.performsNothing(id.EntrypointID), id.Ordinal, prior.activityRunID, outcome)
		if verdict == reservationRejected || !ir.IsNil(completion.result.Response) || outcome.Value != nil {
			return Stop, s.recorder.completionFailure(ctx, "activation_failed", ir.Invalid(ir.Malformed, "reservation", "required activation failed or returned unexpected payload"))
		}
		causes := []string{reservation.cause}
		if attempt := outcome.GetActivityAttempt(); attempt.GetDeliveryId() != "" {
			s.attemptFacts[activity] = priorAttempt{source: reservation.source, activityRunID: attempt.GetActivityRunId()}
		} else if attempt != nil {
			// An attempt that was never delivered is what it is because of the attempt before it.
			causes = append(causes, prior.source)
		}
		publish := s.recorder.publish
		if completion.cleanup {
			publish = s.recorder.publishCleanup
		}
		record := &testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, SourceId: reservation.source, CausalSourceIds: causes, Coordinates: eventCoordinates(id.Origin), Payload: &testpilotspb.RunEvent_Outcome{Outcome: outcome}}
		// What the reservation settled with happened whatever the Program's declarations make of
		// it, so a record that cannot be lifted is still recorded: without evidence, and as the
		// event from which execution is incomplete, so no reader takes it for a record its
		// declaration rejected.
		liftErr := s.liftRunEvents(ctx, reservation.values, []*testpilotspb.RunEvent{record})
		record.ExecutionIncomplete = liftErr != nil
		decision, err := publish(ctx, []*testpilotspb.RunEvent{record}, nil)
		switch {
		case err != nil:
			return decision, err
		// The record's own failure is named before the failure to lift it.
		case verdict == reservationRecordedThenFailed && (decision != Stop || liftErr != nil):
			return Stop, s.recorder.completionFailure(ctx, "activation_failed", ir.Invalid(ir.Malformed, "reservation", "required activation failed"))
		case liftErr != nil:
			return Stop, s.recorder.completionFailure(ctx, "outcome_failed", liftErr)
		default:
			return decision, nil
		}
	}
	task := *completion.node
	a := task.activation.values
	batch, _, err := a.stage(ctx, s.coordinate(task), completion.result, a.workLimit())
	if err != nil {
		if completion.cleanup {
			return Stop, err
		}
		return Stop, s.recorder.completionFailure(ctx, "outcome_failed", err)
	}
	source := s.nodeSource(task)
	kind := testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED
	if batch.outcome.Status == testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT {
		kind = testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT
	}
	facts := []*testpilotspb.RunEvent{{Kind: kind, SourceId: source + ".completed", Coordinates: eventCoordinates(batch.coordinate), CausalSourceIds: []string{source + ".started"}, Payload: &testpilotspb.RunEvent_Outcome{Outcome: batch.outcome}}}
	// A realized fault is recorded as its own fact, so a Contract can reference the outage rather
	// than infer it from the instruction that requested it. A requested-but-unrealized fault
	// never reaches here, which is what keeps intent distinguishable from evidence.
	if n := task.activation.values.graph.nodes[task.index]; n.opcode == contract.InjectFault && batch.outcome.GetStatus() == testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		fault := n.source.Instruction.GetInjectFault()
		facts = append(facts, &testpilotspb.RunEvent{
			Kind:            testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED,
			SourceId:        source + ".fault",
			Coordinates:     eventCoordinates(batch.coordinate),
			CausalSourceIds: []string{source + ".completed"},
			Payload:         &testpilotspb.RunEvent_FaultInjected{FaultInjected: &testpilotspb.FaultInjected{RoleId: fault.GetRoleId(), Kind: fault.GetKind()}},
		})
	}
	for _, fact := range batch.facts {
		coordinate := eventCoordinates(batch.coordinate)
		coordinate.EmittedIndex = fact.index
		facts = append(facts, &testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, SourceId: fmt.Sprintf("%s.p%d.i%d", source, fact.read, fact.index), Coordinates: coordinate, CausalSourceIds: []string{source + ".completed"}, Observations: fact.observations})
	}
	if err := s.liftRunEvents(ctx, a, facts); err != nil {
		if completion.cleanup {
			return Stop, err
		}
		return Stop, s.recorder.completionFailure(ctx, "outcome_failed", err)
	}
	if completion.cleanup {
		return s.recorder.publishCleanup(ctx, facts, func() error { return a.commit(ctx, batch) })
	}
	return s.recorder.publish(ctx, facts, func() error { return a.commit(ctx, batch) })
}

func (s *scheduler) deliverCompletion(completion schedulerCompletion) {
	s.mu.Lock()
	if !s.closing {
		s.completions <- completion
		s.mu.Unlock()
		return
	}
	closed := s.closed
	s.mu.Unlock()
	<-closed
	ctx, cancel := context.WithTimeout(context.Background(), s.lateTimeout)
	defer cancel()
	_ = s.publishSettledCompletion(ctx, completion)
}

// reportExpiry names, on a hinted or once read that timed out, the condition it waited for, the
// bound it waited within and the declarations that bound comes from, whether the Driver timed it
// out or the scheduler did when the instruction's context expired. Every other outcome keeps its
// bytes, so a Case that declares neither records what it did before.
func (s *scheduler) reportExpiry(task scheduledNode, result contract.EffectResult) contract.EffectResult {
	n := task.activation.values.graph.nodes[task.index]
	if n.opcode != contract.ReadEvidence || (len(n.source.GetWaitHints()) == 0 && !n.once) || result.Outcome.GetStatus() != testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT {
		return result
	}
	outcome := proto.CloneOf(result.Outcome)
	limits := s.values.program.limits
	// The detail is an outcome field, so it must fit the outcome's byte bound with room to spare.
	outcome.Detail = expiryDetail(n, s.values.program.boundScale, int(min(1024, max(limits.MaxRequestBytes, limits.MaxResponseBytes)/2)))
	result.Outcome = outcome
	return result
}

func expiryDetail(n *node, scale contract.BoundScale, limit int) string {
	read := n.source.GetInstruction().GetReadEvidence()
	var b strings.Builder
	b.WriteString("evidence " + read.GetEvidenceId() + ": until " + renderCondition(read.GetUntil()))
	if n.once {
		// A read once waits for nothing; its timeout bounds only the one RPC.
		b.WriteString(" did not hold when read once, within " + strconv.FormatInt(n.timeoutMilliseconds, 10) + " ms")
	} else {
		b.WriteString(" did not hold within " + strconv.FormatInt(n.timeoutMilliseconds, 10) + " ms")
	}
	hints := n.source.GetWaitHints()
	if len(hints) > 0 && scale.Scaled() {
		fmt.Fprintf(&b, " (declared %d ms, scaled by %d%%)", n.source.GetLimits().GetTimeoutMilliseconds(), scale.Percent())
	}
	for i, hint := range hints {
		if b.Len() > limit {
			break
		}
		separator := ", "
		if i == 0 {
			separator = "; hints: "
		}
		fmt.Fprintf(&b, "%s%s (%s:%d) %d ms", separator, hint.GetHintId(), hint.GetSource().GetPath(), hint.GetSource().GetLine(), hint.GetAtMostMilliseconds())
	}
	return truncateDetail(b.String(), limit)
}

// truncateDetail cuts text to at most limit bytes on a rune boundary, marking the cut.
func truncateDetail(text string, limit int) string {
	const marker = "..."
	if len(text) <= limit {
		return text
	}
	if limit < len(marker) {
		return ""
	}
	cut := limit - len(marker)
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return strings.ToValidUTF8(text[:cut], "?") + marker
}

// A condition is rendered by hand rather than as protobuf text, whose output is deliberately
// unstable, and only as deep and as long as naming it in a detail needs.
const (
	conditionRenderDepth = 8
	conditionRenderBytes = 512
)

type conditionRenderer struct{ b strings.Builder }

func renderCondition(e *testpilotspb.Expression) string {
	var r conditionRenderer
	r.expression(e, 0)
	return truncateDetail(r.b.String(), conditionRenderBytes)
}

func (r *conditionRenderer) full() bool { return r.b.Len() > conditionRenderBytes }

func (r *conditionRenderer) write(parts ...string) {
	for _, part := range parts {
		// One byte past the bound is enough for renderCondition to mark the cut.
		room := conditionRenderBytes + 1 - r.b.Len()
		if room <= 0 {
			return
		}
		r.b.WriteString(part[:min(len(part), room)])
	}
}

func (r *conditionRenderer) expression(e *testpilotspb.Expression, depth int) {
	if r.full() {
		return
	}
	if depth >= conditionRenderDepth {
		r.write("...")
		return
	}
	switch arm := e.GetExpression().(type) {
	case *testpilotspb.Expression_Literal:
		r.value(arm.Literal, depth)
	case *testpilotspb.Expression_Reference:
		r.reference(arm.Reference)
	case *testpilotspb.Expression_Path:
		r.expression(arm.Path.GetOperand(), depth+1)
		if arm.Path.GetPath() != "" {
			r.write(".", arm.Path.GetPath())
		}
	case *testpilotspb.Expression_Present:
		r.call("present", depth, arm.Present.GetOperand())
	case *testpilotspb.Expression_Compare:
		r.operand(arm.Compare.GetLeft(), depth)
		r.write(" ", comparisonSymbol(arm.Compare.GetOperator()), " ")
		r.operand(arm.Compare.GetRight(), depth)
	case *testpilotspb.Expression_Not:
		r.call("not", depth, arm.Not.GetOperand())
	case *testpilotspb.Expression_All:
		r.call("all", depth, arm.All.GetOperands()...)
	case *testpilotspb.Expression_Any:
		r.call("any", depth, arm.Any.GetOperands()...)
	default:
		r.write("<absent>")
	}
}

// operand renders one side of a comparison, parenthesized when it is a comparison itself.
func (r *conditionRenderer) operand(e *testpilotspb.Expression, depth int) {
	if e.GetCompare() == nil {
		r.expression(e, depth+1)
		return
	}
	r.write("(")
	r.expression(e, depth+1)
	r.write(")")
}

func (r *conditionRenderer) call(name string, depth int, operands ...*testpilotspb.Expression) {
	r.write(name, "(")
	for i, operand := range operands {
		if r.full() {
			return
		}
		if i > 0 {
			r.write(", ")
		}
		r.expression(operand, depth+1)
	}
	r.write(")")
}

func (r *conditionRenderer) reference(reference *testpilotspb.Reference) {
	message := reference.ProtoReflect()
	field := message.WhichOneof(message.Descriptor().Oneofs().ByName("reference"))
	switch {
	case field == nil:
		r.write("<absent>")
	case reference.GetProjectedValue() != nil:
		r.write("value")
	case reference.GetOutcome() != nil:
		instruction := reference.GetOutcome().GetInstruction()
		r.write("outcome(", instruction.GetEntrypointId(), ".", instruction.GetInstructionId(), ").", reference.GetOutcome().GetField().String())
	case field.Kind() == protoreflect.StringKind:
		r.write(string(field.Name()), "(", message.Get(field).String(), ")")
	default:
		r.write(string(field.Name()))
	}
}

func (r *conditionRenderer) value(value *testpilotspb.Value, depth int) {
	switch arm := value.GetValue().(type) {
	case *testpilotspb.Value_TextValue:
		r.write(strconv.Quote(truncateDetail(arm.TextValue, conditionRenderBytes)))
	case *testpilotspb.Value_BoolValue:
		r.write(strconv.FormatBool(arm.BoolValue))
	case *testpilotspb.Value_BytesValue:
		r.write("bytes(", strconv.Itoa(len(arm.BytesValue)), ")")
	case *testpilotspb.Value_SignedIntegerValue:
		r.write(arm.SignedIntegerValue)
	case *testpilotspb.Value_UnsignedIntegerValue:
		r.write(arm.UnsignedIntegerValue)
	case *testpilotspb.Value_FloatingPointValue:
		r.write(strconv.FormatFloat(arm.FloatingPointValue, 'g', -1, 64))
	case *testpilotspb.Value_EnumValue:
		r.write(arm.EnumValue.GetName())
	case *testpilotspb.Value_MessageValue:
		r.write("message(", arm.MessageValue.GetTypeUrl(), ")")
	case *testpilotspb.Value_ListValue:
		if depth+1 >= conditionRenderDepth {
			r.write("[...]")
			return
		}
		r.write("[")
		for i, element := range arm.ListValue.GetValues() {
			if r.full() {
				return
			}
			if i > 0 {
				r.write(", ")
			}
			r.value(element, depth+1)
		}
		r.write("]")
	case *testpilotspb.Value_MapValue:
		r.write("map(", strconv.Itoa(len(arm.MapValue.GetEntries())), " entries)")
	default:
		r.write("<absent>")
	}
}

func comparisonSymbol(operator testpilotspb.ComparisonOperator) string {
	switch operator {
	case testpilotspb.COMPARISON_OPERATOR_EQUAL:
		return "=="
	case testpilotspb.COMPARISON_OPERATOR_NOT_EQUAL:
		return "!="
	case testpilotspb.COMPARISON_OPERATOR_LESS_THAN:
		return "<"
	case testpilotspb.COMPARISON_OPERATOR_LESS_THAN_OR_EQUAL:
		return "<="
	case testpilotspb.COMPARISON_OPERATOR_GREATER_THAN:
		return ">"
	case testpilotspb.COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL:
		return ">="
	default:
		return "?"
	}
}
