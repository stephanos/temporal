package execution

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
)

type scheduler struct {
	values           *valueStore
	recorder         *recorder
	session          Session
	mu               sync.Mutex
	started          bool
	owned            []EffectHandle
	cancellations    []context.CancelFunc
	reservations     map[string]bool
	attempts         int64
	completions      chan schedulerCompletion
	waits            sync.WaitGroup
	pending          int
	ordinaryCanceled bool
	cleanupCanceled  bool
	closing          bool
	closed           chan struct{}
	lateTimeout      time.Duration
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
	handle   ReservationHandle
	identity ReservationIdentity
	source   string
	cause    string
}
type schedulerCompletion struct {
	node        *scheduledNode
	reservation *scheduledReservation
	result      EffectResult
	err         error
	cleanup     bool
}

func newScheduler(p *PreparedProgram, runID, caseID string, session Session, monitor Monitor, now func() time.Time) (*scheduler, error) {
	if isNil(session) {
		return nil, invalid(ir.Malformed, "scheduler", "Session required")
	}
	values, err := newValueStore(p, runID)
	if err != nil {
		return nil, err
	}
	recorder, err := newRecorder(p.view, runID, caseID, monitor, now, values.seal, session.Diagnose)
	if err != nil {
		return nil, err
	}
	return &scheduler{values: values, recorder: recorder, session: session, reservations: map[string]bool{}, completions: make(chan schedulerCompletion, int(p.source.Limits.MaxNodes+p.source.Limits.MaxActivations)), closed: make(chan struct{}), lateTimeout: time.Duration(p.source.Limits.MaxCleanupDurationMilliseconds) * time.Millisecond}, nil
}
func (s *scheduler) outstanding() []EffectHandle {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]EffectHandle(nil), s.owned...)
}
func (s *scheduler) retain(handles []EffectHandle) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, h := range handles {
		if !isNil(h) {
			s.owned = append(s.owned, h)
		}
	}
}
func (s *scheduler) fail(code string, err error) error {
	return s.recorder.fail(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, code, err)
}

// execute leaves accepted handles and buffered completions owned when Stop transfers control to cleanup.
func (s *scheduler) execute(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return invalid(ir.Unavailable, "scheduler", "controller schedule already started")
	}
	s.started = true
	s.mu.Unlock()
	decision, err := s.recorder.publish(ctx, []*umpirespb.RunEvent{{Kind: umpirespb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "scheduler.open"}}, nil)
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
		select {
		case <-s.recorder.halted:
			return s.recorder.schedulingFailure()
		case <-ctx.Done():
			return s.fail("schedule_cancelled", ctx.Err())
		case completion := <-s.completions:
			if !completion.cleanup {
				active--
			}
			ready, decision, err = s.acceptCompletion(ctx, completion, false)
			if err != nil || decision == Stop {
				return err
			}
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
	coordinates := &umpirespb.RunCoordinates{EntrypointId: g.id, ActivationId: values.id}
	_, err = s.recorder.publishCleanup(ctx, []*umpirespb.RunEvent{{Kind: umpirespb.RUN_EVENT_KIND_CLEANUP_STARTED, SourceId: "scheduler.cleanup.open", Coordinates: coordinates}}, nil)
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
		_, err := s.recorder.publishCleanup(ctx, []*umpirespb.RunEvent{{Kind: umpirespb.RUN_EVENT_KIND_CLEANUP_COMPLETED, SourceId: "scheduler.cleanup.close", CausalSourceIds: causes, Coordinates: coordinates}}, nil)
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
		select {
		case <-ctx.Done():
			return ctx.Err()
		case completion := <-s.completions:
			if !completion.cleanup {
				s.mu.Lock()
				s.pending--
				s.mu.Unlock()
				_ = s.publishSettledCompletion(ctx, completion)
				continue
			}
			active--
			ready, _, err = s.acceptCompletion(ctx, completion, true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *scheduler) ownedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.owned)
}

func (s *scheduler) outstandingSince(index int) []EffectHandle {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index > len(s.owned) {
		return nil
	}
	return append([]EffectHandle(nil), s.owned[index:]...)
}

func (s *scheduler) cancelWaits() {
	s.mu.Lock()
	cancellations := append([]context.CancelFunc(nil), s.cancellations...)
	s.mu.Unlock()
	for _, cancel := range cancellations {
		cancel()
	}
}

func (s *scheduler) settle(ctx context.Context, handles []EffectHandle, cleanup, cancelHandles bool) error {
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

func (s *scheduler) cancelOwned(ctx context.Context, handles []EffectHandle) error {
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
			s.recorder.report(umpirespb.RUN_DIAGNOSTIC_KIND_HOST_CONTRACT, code, err)
		}
	}
	return result
}

func (s *scheduler) drainOwned(ctx context.Context, handles []EffectHandle) error {
	var result error
	for _, handle := range handles {
		drainErr := handle.Drain(ctx)
		if drainErr == nil && ctx.Err() != nil {
			drainErr = ctx.Err()
			result = errors.Join(result, drainErr)
			s.recorder.report(umpirespb.RUN_DIAGNOSTIC_KIND_HOST_CONTRACT, "effect_drain_context_violated", drainErr)
		}
		if drainErr != nil {
			result = errors.Join(result, s.quarantine(handle, drainErr))
		}
	}
	return result
}

func (s *scheduler) quarantine(handle EffectHandle, drainErr error) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.lateTimeout)
	defer cancel()
	err := s.session.Quarantine(ctx, handle)
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		s.recorder.report(umpirespb.RUN_DIAGNOSTIC_KIND_HOST_CONTRACT, "quarantine_failed", err)
		return err
	}
	s.recorder.report(umpirespb.RUN_DIAGNOSTIC_KIND_LIMIT, "effect_quarantined", drainErr)
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
		select {
		case completion := <-s.completions:
			s.mu.Lock()
			s.pending--
			s.mu.Unlock()
			err := s.publishSettledCompletion(ctx, completion)
			if completion.cleanup == cleanup {
				result = errors.Join(result, err)
			}
		case <-ctx.Done():
			return errors.Join(result, s.settleBufferedCompletions(cleanup))
		}
	}
	return result
}

func (s *scheduler) publishSettledCompletion(ctx context.Context, completion schedulerCompletion) error {
	if s.expectedCancellation(completion) {
		return nil
	}
	parent := ctx
	if isNil(parent) || parent.Err() != nil {
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

func (s *scheduler) settleBufferedCompletions(cleanup bool) error {
	var result error
	for {
		select {
		case completion := <-s.completions:
			s.mu.Lock()
			s.pending--
			s.mu.Unlock()
			err := s.publishSettledCompletion(context.Background(), completion)
			if completion.cleanup == cleanup {
				result = errors.Join(result, err)
			}
		default:
			return result
		}
	}
}

func (s *scheduler) beginClose() {
	s.mu.Lock()
	s.closing = true
	s.mu.Unlock()
	for {
		select {
		case completion := <-s.completions:
			s.mu.Lock()
			s.pending--
			s.mu.Unlock()
			_ = s.publishSettledCompletion(context.Background(), completion)
		default:
			return
		}
	}
}

func (s *scheduler) finishClose() {
	close(s.closed)
}
func (s *scheduler) openControllers(ctx context.Context) ([]scheduledNode, Decision, error) {
	var decision Decision
	var ready []scheduledNode
	for ordinal, g := range s.values.program.graphs {
		if g.cleanup || g.context != umpirespb.ENTRYPOINT_CONTEXT_CONTROLLER {
			continue
		}
		values, err := s.values.activate(g.id, fmt.Sprintf("controller.%d", ordinal))
		if err != nil {
			return nil, Stop, s.fail("activation_failed", err)
		}
		a := &scheduledActivation{values: values, ordinal: ordinal, remaining: make([]int, len(g.nodes)), completed: make([]string, len(g.nodes)), pending: len(g.nodes)}
		decision, err = s.recorder.publish(ctx, []*umpirespb.RunEvent{s.activationEvent(a, false)}, nil)
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
			decision, err = s.recorder.publish(ctx, []*umpirespb.RunEvent{s.activationEvent(a, true)}, nil)
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
	s.mu.Lock()
	s.pending--
	s.mu.Unlock()
	decision, err := s.publishCompletion(ctx, completion)
	if err != nil || decision == Stop && !cleanup || completion.node == nil {
		return nil, decision, err
	}
	return s.completeNode(ctx, *completion.node, s.nodeSource(*completion.node)+".completed")
}
func (s *scheduler) activationEvent(a *scheduledActivation, closed bool) *umpirespb.RunEvent {
	kind := umpirespb.RUN_EVENT_KIND_ACTIVATION_OPENED
	source := fmt.Sprintf("scheduler.g%d.open", a.ordinal)
	causes := []string{"scheduler.open"}
	if closed {
		kind = umpirespb.RUN_EVENT_KIND_ACTIVATION_CLOSED
		source = fmt.Sprintf("scheduler.g%d.close", a.ordinal)
		causes = []string{fmt.Sprintf("scheduler.g%d.open", a.ordinal)}
		for _, id := range a.completed {
			if id != "" {
				causes = append(causes, id)
			}
		}
	}
	return &umpirespb.RunEvent{Kind: kind, SourceId: source, CausalSourceIds: causes, Coordinates: &umpirespb.RunCoordinates{EntrypointId: a.values.graph.id, ActivationId: a.values.id}}
}
func (s *scheduler) nodeSource(task scheduledNode) string {
	if task.activation.cleanup {
		return fmt.Sprintf("scheduler.cleanup.n%d.a1", task.index)
	}
	return fmt.Sprintf("scheduler.g%d.n%d.a1", task.activation.ordinal, task.index)
}
func (s *scheduler) coordinate(task scheduledNode) Coordinate {
	return Coordinate{RunID: s.values.runID, EntrypointID: task.activation.values.graph.id, ActivationID: task.activation.values.id, InstructionID: task.activation.values.graph.nodes[task.index].source.InstructionId, Attempt: 1}
}
func eventCoordinates(c Coordinate) *umpirespb.RunCoordinates {
	return &umpirespb.RunCoordinates{EntrypointId: c.EntrypointID, ActivationId: c.ActivationID, InstructionId: c.InstructionID, Attempt: c.Attempt}
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
		d, err := s.recorder.publish(ctx, []*umpirespb.RunEvent{s.activationEvent(a, true)}, nil)
		return ready, d, err
	}
	return ready, Continue, nil
}
func (s *scheduler) dispatch(ctx context.Context, task scheduledNode, cleanup bool) (int, bool, Decision, error) {
	a := task.activation.values
	n := a.graph.nodes[task.index]
	c := s.coordinate(task)
	request, input, enabled, err := s.prepareInput(ctx, task)
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
	operationCtx, cancel := context.WithTimeout(ctx, time.Duration(n.source.Bounds.TimeoutMilliseconds)*time.Millisecond)
	effect, reservations, bridge, err := s.admitDispatch(operationCtx, task, request, input, cleanup)
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

func (s *scheduler) publishInstructionStart(ctx context.Context, task scheduledNode, coordinate Coordinate, cleanup bool) (Decision, error) {
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
	return publish(ctx, []*umpirespb.RunEvent{{Kind: umpirespb.RUN_EVENT_KIND_INSTRUCTION_STARTED, SourceId: s.nodeSource(task) + ".started", Coordinates: eventCoordinates(coordinate), CausalSourceIds: causes}}, nil)
}

func (s *scheduler) admitDispatch(ctx context.Context, task scheduledNode, request proto.Message, input *umpirespb.Value, cleanup bool) (EffectHandle, []scheduledReservation, SlotBridge, error) {
	n := task.activation.values.graph.nodes[task.index]
	var effect EffectHandle
	var reservations []scheduledReservation
	var bridge SlotBridge
	admit := s.recorder.admit
	if cleanup {
		admit = s.recorder.admitCleanup
	}
	err := admit(ctx, func(ctx context.Context) ([]EffectHandle, error) {
		if s.attempts >= s.values.program.source.Limits.MaxAttempts {
			return nil, invalid(ir.LimitExceeded, "scheduler", "attempt ceiling exceeded")
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
		effect, bridge, err = s.acceptEffect(ctx, task, request, input)
		if !isNil(effect) {
			accepted = append(accepted, effect)
		} else if err == nil && n.opcode != AwaitSlot {
			err = invalid(ir.Malformed, "effect", "nil effect handle")
		}
		return accepted, err
	}, s.retain)
	return effect, reservations, bridge, err
}

func (s *scheduler) startWaits(ctx, operationCtx context.Context, cancel context.CancelFunc, task scheduledNode, effect EffectHandle, bridge SlotBridge, reservations []scheduledReservation, cleanup bool) {
	for _, reservation := range reservations {
		s.waits.Add(1)
		go func() {
			defer s.waits.Done()
			result, err := reservation.handle.Wait(ctx)
			s.deliverCompletion(schedulerCompletion{reservation: &reservation, result: result, err: err, cleanup: cleanup})
		}()
	}
	s.waits.Add(1)
	go func() {
		defer s.waits.Done()
		defer cancel()
		result, err := s.waitNode(operationCtx, task, effect, bridge)
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			result = EffectResult{Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT}}
			err = nil
		}
		s.deliverCompletion(schedulerCompletion{node: &task, result: result, err: err, cleanup: cleanup})
	}()
}
func (s *scheduler) prepareInput(ctx context.Context, task scheduledNode) (proto.Message, *umpirespb.Value, bool, error) {
	a := task.activation.values
	n := a.graph.nodes[task.index]
	c := s.coordinate(task)
	var request proto.Message
	var input *umpirespb.Value
	if n.opcode == InvokeRPC {
		var enabled bool
		var err error
		request, enabled, _, err = a.request(ctx, c, a.workLimit())
		if err != nil {
			return nil, nil, false, err
		}
		if !enabled {
			return nil, nil, false, nil
		}
	} else {
		work, err := a.newWork(ctx, a.workLimit())
		if err != nil {
			return nil, nil, false, err
		}
		if n.guard != nil {
			guard, err := a.evaluate(work, n.guard)
			if err != nil {
				return nil, nil, false, err
			}
			if !guard.GetBoolValue() {
				return nil, nil, false, nil
			}
		}
		if n.input != nil {
			input, err = a.evaluate(work, n.input)
			if err != nil {
				return nil, nil, false, err
			}
		}
	}

	return request, input, true, nil
}
func (s *scheduler) reserve(ctx context.Context, task scheduledNode) ([]EffectHandle, []scheduledReservation, error) {
	var accepted []EffectHandle
	var reservations []scheduledReservation
	n := task.activation.values.graph.nodes[task.index]
	c := s.coordinate(task)
	for declarationIndex, declaration := range n.source.ActivationReservations {
		request := ReservationRequest{Origin: c, EntrypointID: declaration.EntrypointId, Count: declaration.Count}
		acquired, err := s.session.Reserve(ctx, request)
		for _, h := range acquired {
			if !isNil(h) {
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
func (s *scheduler) validateReservations(task scheduledNode, declarationIndex int, request ReservationRequest, acquired []ReservationHandle) ([]scheduledReservation, error) {
	var reservations []scheduledReservation
	if int64(len(acquired)) != request.Count {
		return nil, invalid(ir.Malformed, "reservation", "wrong reservation count")
	}
	ordinals := map[int64]bool{}
	for _, h := range acquired {
		if isNil(h) {
			return nil, invalid(ir.Malformed, "reservation", "nil reservation")
		}
		id := h.Identity()
		if !validID(id.ID) || id.Origin != request.Origin || id.EntrypointID != request.EntrypointID || id.Ordinal < 0 || id.Ordinal >= request.Count || ordinals[id.Ordinal] || s.reservations[id.ID] {
			return nil, invalid(ir.Malformed, "reservation", "crossed or duplicate reservation identity")
		}
		ordinals[id.Ordinal] = true
		s.reservations[id.ID] = true
		reservations = append(reservations, scheduledReservation{handle: h, identity: id, source: fmt.Sprintf("%s.r%d.i%d", s.nodeSource(task), declarationIndex, id.Ordinal), cause: s.nodeSource(task) + ".started"})
	}
	return reservations, nil
}
func (s *scheduler) acceptEffect(ctx context.Context, task scheduledNode, request proto.Message, input *umpirespb.Value) (EffectHandle, SlotBridge, error) {
	n := task.activation.values.graph.nodes[task.index]
	c := s.coordinate(task)
	var effect EffectHandle
	var bridge SlotBridge
	var err error
	switch n.opcode {
	case InvokeRPC:
		effect, err = s.session.InvokeRPC(ctx, c, n.source.Instruction.GetInvokeRpc().EndpointRoleId, n.method, request)
	case AwaitSlot, CompleteNexusOperation:
		slot := n.source.Instruction.GetAwaitSlot().GetSlotId()
		if n.opcode == CompleteNexusOperation {
			slot = n.source.Instruction.GetCompleteNexusOperation().CapabilitySlotId
		}
		if s.values.program.slots[slot].Opaque() {
			bridge, err = s.session.Bridge(ctx)
			if err == nil && isNil(bridge) {
				err = invalid(ir.Malformed, "bridge", "nil bridge")
			}
		}
		if err == nil && n.opcode == CompleteNexusOperation {
			var capability OpaqueCapability
			capability, err = bridge.Consume(ctx, slot)
			if err == nil && isNil(capability) {
				err = invalid(ir.Malformed, "bridge", "nil capability")
			}
			if err == nil {
				effect, err = s.session.CompleteNexusOperation(ctx, c, capability, input)
			}
		}
	default:
		err = invalid(ir.Unsupported, "scheduler", "controller opcode required")
	}

	return effect, bridge, err
}
func (s *scheduler) waitNode(ctx context.Context, task scheduledNode, effect EffectHandle, bridge SlotBridge) (EffectResult, error) {
	a := task.activation.values
	n := a.graph.nodes[task.index]
	var result EffectResult
	var err error
	if n.opcode == AwaitSlot {
		slot := n.source.Instruction.GetAwaitSlot().SlotId
		if bridge != nil {
			err = bridge.Await(ctx, slot)
		} else {
			err = a.awaitSlot(ctx, slot)
		}
		if err == nil {
			result.Outcome = &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}
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
		if completion.result.Outcome == nil || completion.result.Outcome.Status != umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED || !isNil(completion.result.Response) || completion.result.Outcome.Value != nil {
			return Stop, s.recorder.completionFailure(ctx, "activation_failed", invalid(ir.Malformed, "reservation", "required activation failed or returned unexpected payload"))
		}
		publish := s.recorder.publish
		if completion.cleanup {
			publish = s.recorder.publishCleanup
		}
		return publish(ctx, []*umpirespb.RunEvent{{Kind: umpirespb.RUN_EVENT_KIND_DIAGNOSTIC, SourceId: reservation.source, CausalSourceIds: []string{reservation.cause}, Coordinates: eventCoordinates(id.Origin), Outcome: completion.result.Outcome}}, nil)
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
	kind := umpirespb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED
	if batch.outcome.Status == umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT {
		kind = umpirespb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT
	}
	facts := []*umpirespb.RunEvent{{Kind: kind, SourceId: source + ".completed", Coordinates: eventCoordinates(batch.coordinate), CausalSourceIds: []string{source + ".started"}, Outcome: batch.outcome}}
	for _, projection := range batch.facts {
		coordinate := eventCoordinates(batch.coordinate)
		coordinate.EmittedIndex = projection.index
		facts = append(facts, &umpirespb.RunEvent{Kind: umpirespb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, SourceId: fmt.Sprintf("%s.p%d.i%d", source, projection.projection, projection.index), Coordinates: coordinate, CausalSourceIds: []string{source + ".completed"}, Observations: projection.observations})
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
