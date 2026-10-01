package delivery

import (
	"context"
	"errors"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// StartActivityPath is the full gRPC method path of WorkflowService.StartActivityExecution, the
// carrier of a standalone activity's reservation.
const StartActivityPath = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"

// ActivityDelivery is one activity task as the worker receives it: the header its start request
// carried, the physical activity it names, and its three identities. ActivityRunID is the run the
// server started, the logical operation every attempt belongs to. Attempt is the server's number
// for this attempt. DeliveryID is a bounded opaque name of this delivery of the attempt.
type ActivityDelivery struct {
	Header                              *commonpb.Header
	Namespace, ActivityID, ActivityType string
	TaskQueue, ActivityRunID            string
	Attempt                             int32
	DeliveryID                          string
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
	attempts := of.state.bundle.attempts
	last := 0
	for index, attempt := range attempts {
		if attempt.activation.attempt != 0 {
			last = index
		}
	}
	var released []testpilot.ReservationIdentity
	for _, attempt := range attempts[last+1:] {
		if attempt.authority == reserved {
			attempt.authority = canceled
			released = append(released, attempt.identity)
		}
	}
	return released, nil
}

func (l *Ledger) activityRoute(state *routeState) route {
	return route{Version: routeVersion, Kind: activityRoute, SessionID: l.config.SessionID, RunID: l.config.RunID, Origin: state.bundle.origin, Reservation: state.identity, Activity: state.bundle.activityBinding}
}
