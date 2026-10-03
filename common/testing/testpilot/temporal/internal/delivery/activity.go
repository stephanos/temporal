package delivery

import (
	"context"
	"errors"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// StartActivityPath is the full gRPC method path of WorkflowService.StartActivityExecution, the
// carrier of a standalone activity's reservation.
const StartActivityPath = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"

// ActivityDelivery is one activity task as the worker receives it: the header its start request or
// schedule command carried, the physical activity it names, and its identities. ActivityRunID is
// the run the server started for a standalone activity, the logical operation every attempt
// belongs to; WorkflowRunID is the run of the workflow that scheduled any other activity. Attempt
// is the server's number for this attempt. DeliveryID is a bounded opaque name of this delivery of
// the attempt.
type ActivityDelivery struct {
	Header                              *commonpb.Header
	Namespace, ActivityID, ActivityType string
	TaskQueue, ActivityRunID            string
	WorkflowRunID                       string
	Attempt                             int32
	DeliveryID                          string
}

// Scheduled reports whether the delivery is of an activity a workflow scheduled, which carries the
// scheduled-activity route rather than a start's.
func (d ActivityDelivery) Scheduled() bool {
	_, scheduled := d.Header.GetFields()[ScheduledActivityHeader]
	return scheduled
}

func (d ActivityDelivery) Binding() ActivityBinding {
	return ActivityBinding{Namespace: d.Namespace, ActivityID: d.ActivityID, ActivityType: d.ActivityType, TaskQueue: d.TaskQueue}
}

// ActivityStartBinding reads the activity binding a StartActivityExecution request names, and the
// header it carries beside the binding. It reads by descriptor, because requests are dynamic
// messages.
func ActivityStartBinding(message protoreflect.Message) (ActivityBinding, protoreflect.Message, error) {
	fields := message.Descriptor().Fields()
	namespace := fields.ByName("namespace")
	activityID := fields.ByName("activity_id")
	activityType := fields.ByName("activity_type")
	taskQueue := fields.ByName("task_queue")
	header := fields.ByName("header")
	if namespace == nil || activityID == nil || activityType == nil || taskQueue == nil || header == nil || !message.Has(activityType) || !message.Has(taskQueue) {
		return ActivityBinding{}, nil, ErrInvalid
	}
	typeName := activityType.Message().Fields().ByName("name")
	queueName := taskQueue.Message().Fields().ByName("name")
	if typeName == nil || queueName == nil {
		return ActivityBinding{}, nil, ErrInvalid
	}
	binding := ActivityBinding{
		Namespace:    message.Get(namespace).String(),
		ActivityID:   message.Get(activityID).String(),
		ActivityType: message.Get(activityType).Message().Get(typeName).String(),
		TaskQueue:    message.Get(taskQueue).Message().Get(queueName).String(),
	}
	var headerMessage protoreflect.Message
	if message.Has(header) {
		headerMessage = message.Get(header).Message()
	}
	return binding, headerMessage, nil
}

// ErrAttemptUndeclared rejects an activity attempt whose number is past the last attempt the
// activity's script declares.
var ErrAttemptUndeclared = errors.New("activity attempt is past the attempts its script declares")

// AdmitActivity admits one delivery of the activity a carrier started. The attempt Temporal numbers
// N is the Nth reservation of the operation, whatever order the attempts arrive in, so an attempt
// is one activation and always the one its number names. A delivery of an attempt already
// admitted, under whatever delivery identity, is a replay of that admission and consumes nothing.
// An attempt past the last reservation is undeclared, one whose reservation was released is stale,
// and an attempt of another run conflicts.
func (l *Ledger) AdmitActivity(ctx context.Context, delivery ActivityDelivery) (Activation, error) {
	encoded, err := decodeReservedHeader(delivery.Header, reservedActivityHeader, l.config.Limits.MaxHeaderBytes)
	if err != nil {
		return Activation{}, err
	}
	wire, err := (routeCodec{maximumBytes: l.config.Limits.MaxHeaderBytes}).decode(encoded, activityRoute)
	if err != nil {
		return Activation{}, err
	}
	providedBinding := delivery.Binding()
	if !validActivityBinding(providedBinding) || !validRouteText(delivery.ActivityRunID) || delivery.Attempt <= 0 || !validRouteText(delivery.DeliveryID) {
		return Activation{}, ErrInvalid
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return Activation{}, err
	}
	if wire.SessionID != l.config.SessionID || wire.RunID != l.config.RunID {
		l.mu.Unlock()
		return Activation{}, ErrRouteCrossed
	}
	operation := l.operations[wire.Reservation.ID]
	if operation == nil {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Activation{}, ErrRouteStale
	}
	if wire != l.activityRoute(operation.attempts[0]) {
		l.mu.Unlock()
		return Activation{}, ErrRouteCrossed
	}
	if providedBinding != operation.activityBinding {
		l.mu.Unlock()
		return Activation{}, ErrBindingMismatch
	}
	if l.stopped || operation.triggerFinal && operation.triggerStatus != TriggerSucceeded {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Activation{}, ErrRouteStale
	}
	if operation.responseRunID != "" && operation.responseRunID != delivery.ActivityRunID || operation.startedRunID() != "" && operation.startedRunID() != delivery.ActivityRunID {
		l.mu.Unlock()
		return Activation{}, ErrRouteConflict
	}
	if int(delivery.Attempt) > len(operation.attempts) {
		l.mu.Unlock()
		return Activation{}, ErrAttemptUndeclared
	}
	attempt := operation.attempts[delivery.Attempt-1]
	if attempt.activation.attempt != 0 {
		l.mu.Unlock()
		return Activation{ledger: l, state: attempt, data: attempt.activation, replay: true}, nil
	}
	if attempt.authority != reserved {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Activation{}, ErrRouteStale
	}
	return l.consumeLocked(ctx, attempt, activationData{temporalRunID: delivery.ActivityRunID, attempt: delivery.Attempt, deliveryID: delivery.DeliveryID})
}

// ReleaseActivityAttempts releases the declared attempts of a closed activity that no attempt was
// delivered for: every attempt still reserved after the last one admitted. Each stops being
// admissible and is named, once, for the caller to settle as not needed; the ledger cancels nothing,
// because not needed is an outcome and a cancellation is not. An earlier attempt still reserved is
// one the worker never saw, which the activity closing does not explain, so it stays reserved.
func (l *Ledger) ReleaseActivityAttempts(ctx context.Context, of Activation) ([]testpilot.ReservationIdentity, error) {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	defer l.mu.Unlock()
	if of.ledger != l || of.state == nil || of.state.kind != activityRoute || of.data != of.state.activation {
		return nil, ErrRouteCrossed
	}
	return releaseUndeliveredAttempts(of.state.bundle.attempts), nil
}

// releaseUndeliveredAttempts releases the attempts still reserved after the last one delivered,
// and names them. With none delivered it releases nothing.
func releaseUndeliveredAttempts(attempts []*routeState) []testpilot.ReservationIdentity {
	last := -1
	for index, attempt := range attempts {
		if attempt.activation.attempt != 0 {
			last = index
		}
	}
	var released []testpilot.ReservationIdentity
	if last < 0 {
		return nil
	}
	for _, attempt := range attempts[last+1:] {
		if attempt.authority == reserved {
			attempt.authority = canceled
			released = append(released, attempt.identity)
		}
	}
	return released
}

func (l *Ledger) activityRoute(state *routeState) route {
	return route{Version: routeVersion, Kind: activityRoute, SessionID: l.config.SessionID, RunID: l.config.RunID, Origin: state.bundle.origin, Reservation: state.identity, Activity: state.bundle.activityBinding}
}

// ActivityDispatch is the header entry an activity schedule command carries so its attempts reach
// the reservations its workflow's start reserved.
type ActivityDispatch struct {
	payload *commonpb.Payload
}

// Header is the dispatch's header entry: its name and its payload.
func (d ActivityDispatch) Header() (string, *commonpb.Payload) {
	return ScheduledActivityHeader, proto.CloneOf(d.payload)
}

// PrepareActivity prepares the route of the activity the admitted workflow's schedule command
// sourceInstructionID reaches. A schedule command that reaches no reserved activity crosses.
func (l *Ledger) PrepareActivity(ctx context.Context, workflow Activation, sourceInstructionID string) (ActivityDispatch, error) {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return ActivityDispatch{}, err
	}
	if !validRouteText(sourceInstructionID) {
		return ActivityDispatch{}, ErrInvalid
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return ActivityDispatch{}, err
	}
	workflowState, err := l.activationLocked(workflow, workflowRoute)
	if err != nil {
		l.mu.Unlock()
		return ActivityDispatch{}, err
	}
	if l.stopped || workflowState.authority != admitted || workflowState.bundle.parentReleased {
		l.diagnoseLocked()
		l.mu.Unlock()
		return ActivityDispatch{}, ErrRouteStale
	}
	key := sourceKey{workflowEntrypoint: workflowState.identity.EntrypointID, workflowOrdinal: workflowState.identity.Ordinal, sourceInstruction: sourceInstructionID}
	attempts := workflowState.bundle.scheduled[key]
	if len(attempts) == 0 {
		l.mu.Unlock()
		return ActivityDispatch{}, ErrRouteCrossed
	}
	wire := l.scheduledActivityRoute(attempts[0])
	l.mu.Unlock()
	encoded, err := (routeCodec{maximumBytes: l.config.Limits.MaxHeaderBytes}).encode(wire)
	if err != nil {
		return ActivityDispatch{}, err
	}
	return ActivityDispatch{payload: &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte(workflowRouteEncoding)}, Data: encoded}}, nil
}

// ScheduledEntrypoint is the activity entrypoint a scheduled activity's route names, read before
// admission so a task of another activity is refused without consuming a reservation.
func (l *Ledger) ScheduledEntrypoint(delivery ActivityDelivery) (string, error) {
	encoded, err := decodeReservedHeader(delivery.Header, ScheduledActivityHeader, l.config.Limits.MaxHeaderBytes)
	if err != nil {
		return "", err
	}
	wire, err := (routeCodec{maximumBytes: l.config.Limits.MaxHeaderBytes}).decode(encoded, scheduledActivityRoute)
	if err != nil {
		return "", err
	}
	return wire.Reservation.EntrypointID, nil
}

// AdmitScheduledActivity admits one delivery of an activity a workflow scheduled, as AdmitActivity
// does one of a standalone activity: the attempt Temporal numbers N is the Nth reservation of the
// activity whatever order the attempts arrive in, a delivery of an attempt already admitted is a
// replay of that admission, and an attempt past the last reservation is undeclared. The attempts
// belong to the workflow run that scheduled them and share one activity ID, so a delivery naming
// another of either conflicts. An attempt delivered once its workflow closed is stale.
func (l *Ledger) AdmitScheduledActivity(ctx context.Context, delivery ActivityDelivery) (Activation, error) {
	encoded, err := decodeReservedHeader(delivery.Header, ScheduledActivityHeader, l.config.Limits.MaxHeaderBytes)
	if err != nil {
		return Activation{}, err
	}
	wire, err := (routeCodec{maximumBytes: l.config.Limits.MaxHeaderBytes}).decode(encoded, scheduledActivityRoute)
	if err != nil {
		return Activation{}, err
	}
	if !validActivityBinding(delivery.Binding()) || !validRouteText(delivery.WorkflowRunID) || delivery.Attempt <= 0 || !validRouteText(delivery.DeliveryID) {
		return Activation{}, ErrInvalid
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return Activation{}, err
	}
	if wire.SessionID != l.config.SessionID || wire.RunID != l.config.RunID {
		l.mu.Unlock()
		return Activation{}, ErrRouteCrossed
	}
	attempts := l.scheduled[wire.Reservation.ID]
	if len(attempts) == 0 {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Activation{}, ErrRouteStale
	}
	if wire != l.scheduledActivityRoute(attempts[0]) {
		l.mu.Unlock()
		return Activation{}, ErrRouteCrossed
	}
	bundle := attempts[0].bundle
	if delivery.Namespace != bundle.binding.Namespace {
		l.mu.Unlock()
		return Activation{}, ErrBindingMismatch
	}
	if delivery.WorkflowRunID != wire.WorkflowRunID {
		l.mu.Unlock()
		return Activation{}, ErrRouteConflict
	}
	for _, attempt := range attempts {
		if attempt.activation.attempt != 0 && attempt.activation.activityID != delivery.ActivityID {
			l.mu.Unlock()
			return Activation{}, ErrRouteConflict
		}
	}
	if int(delivery.Attempt) > len(attempts) {
		l.mu.Unlock()
		return Activation{}, ErrAttemptUndeclared
	}
	attempt := attempts[delivery.Attempt-1]
	if attempt.activation.attempt != 0 {
		l.mu.Unlock()
		return Activation{ledger: l, state: attempt, data: attempt.activation, replay: true}, nil
	}
	if l.stopped || bundle.parentReleased || attempt.authority != reserved || bundle.workflow.authority != admitted {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Activation{}, ErrRouteStale
	}
	return l.consumeLocked(ctx, attempt, activationData{temporalRunID: delivery.WorkflowRunID, attempt: delivery.Attempt, deliveryID: delivery.DeliveryID, activityID: delivery.ActivityID})
}

// scheduledActivityRoute is the route of the scheduled activity whose first attempt state is: the
// workflow activation that scheduled it and the schedule command.
func (l *Ledger) scheduledActivityRoute(state *routeState) route {
	workflow := state.bundle.workflow
	return route{Version: routeVersion, Kind: scheduledActivityRoute, SessionID: l.config.SessionID, RunID: l.config.RunID, Origin: state.bundle.origin, Reservation: state.identity, Binding: state.bundle.binding, WorkflowReservation: workflow.identity.ID, WorkflowEntrypoint: state.source.workflowEntrypoint, WorkflowOrdinal: state.source.workflowOrdinal, WorkflowRunID: workflow.activation.temporalRunID, SourceInstructionID: state.source.sourceInstruction}
}
