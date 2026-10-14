package delivery

import (
	"bytes"
	"context"
	"encoding/base64"
	"maps"

	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/primitive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	reservedWorkflowHeader = "temporal-testpilot-reserved-workflow-v1"
	reservedNexusHeader    = "temporal-testpilot-reserved-nexus-v1"
	reservedActivityHeader = "temporal-testpilot-reserved-activity-v1"
	// ScheduledActivityHeader routes the attempts of an activity a workflow scheduled.
	ScheduledActivityHeader = "temporal-testpilot-reserved-scheduled-activity-v1"
	workflowRouteEncoding   = "binary/temporal-testpilot-reservation-route"
)

type WorkflowDelivery struct {
	Header                              *commonpb.Header
	Namespace, WorkflowID, WorkflowType string
	TaskQueue, TemporalRunID            string
}

func (d WorkflowDelivery) Binding() WorkflowBinding {
	return WorkflowBinding{Namespace: d.Namespace, WorkflowID: d.WorkflowID, WorkflowType: d.WorkflowType, TaskQueue: d.TaskQueue}
}

type NexusDelivery struct {
	Header    nexus.Header
	RequestID string
}

type NexusDispatch struct {
	header nexus.Header
}

func (d NexusDispatch) Header() nexus.Header { return maps.Clone(d.header) }

func (l *Ledger) PrepareRPC(ctx context.Context, carrier *Bundle, role string, method protoreflect.MethodDescriptor, request proto.Message, maximumBytes int64) (proto.Message, error) {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return nil, err
	}
	if carrier == nil {
		return request, nil
	}
	if primitive.NilValue(method) || primitive.NilValue(request) || maximumBytes <= 0 || method.IsStreamingClient() || method.IsStreamingServer() || request.ProtoReflect().Descriptor() != method.Input() {
		return nil, ErrInvalid
	}
	var start startBinding
	var header protoreflect.Message
	var reserved string
	var err error
	switch primitive.MethodPath(method) {
	case primitive.StartWorkflowPath:
		start.kind, reserved = workflowRoute, reservedWorkflowHeader
		start.workflow, header, err = StartBinding(request.ProtoReflect())
	case StartActivityPath:
		start.kind, reserved = activityRoute, reservedActivityHeader
		start.activity, header, err = ActivityStartBinding(request.ProtoReflect())
	default:
		return nil, ErrInvalid
	}
	if err != nil {
		return nil, err
	}
	if hasReservedHeader(header, reserved) {
		return nil, ErrReservedHeader
	}
	route, err := l.prepareStartedRoute(ctx, *carrier, role, method, start)
	if err != nil {
		return nil, err
	}
	encoded, err := (routeCodec{maximumBytes: l.config.Limits.MaxHeaderBytes}).encode(route)
	if err != nil {
		return nil, err
	}
	prepared := proto.Clone(request)
	if err := injectReservedHeader(prepared.ProtoReflect(), reserved, encoded); err != nil {
		return nil, err
	}
	if int64(proto.Size(prepared)) > maximumBytes {
		return nil, ErrCapacity
	}
	return prepared, nil
}

// prepareStartedRoute is the route of the entity the carrier's start request names. A carrier
// prepares only the start its plan names, on the role its plan names, for the binding it reserved.
func (l *Ledger) prepareStartedRoute(ctx context.Context, carrier Bundle, role string, method protoreflect.MethodDescriptor, start startBinding) (route, error) {
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return route{}, err
	}
	defer l.mu.Unlock()
	state, err := l.bundleLocked(carrier)
	if err != nil {
		return route{}, err
	}
	started := state.started()
	if l.stopped || started.authority == canceled || started.authority == terminal {
		l.diagnoseLocked()
		return route{}, ErrRouteStale
	}
	if role != state.plan.EndpointRoleID || primitive.MethodPath(method) != state.plan.Method {
		return route{}, ErrRouteCrossed
	}
	if start.workflow != state.binding || start.activity != state.activityBinding {
		return route{}, ErrBindingMismatch
	}
	if started.kind == activityRoute {
		return l.activityRoute(started), nil
	}
	return l.workflowRoute(started), nil
}

func (l *Ledger) AdmitWorkflow(ctx context.Context, delivery WorkflowDelivery) (Activation, error) {
	encoded, err := decodeReservedHeader(delivery.Header, reservedWorkflowHeader, l.config.Limits.MaxHeaderBytes)
	if err != nil {
		return Activation{}, err
	}
	wire, err := (routeCodec{maximumBytes: l.config.Limits.MaxHeaderBytes}).decode(encoded, workflowRoute)
	if err != nil {
		return Activation{}, err
	}
	providedBinding := delivery.Binding()
	if !validBinding(providedBinding) || !validRouteText(delivery.TemporalRunID) {
		return Activation{}, ErrInvalid
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return Activation{}, err
	}
	if wire.SessionID != l.config.SessionID || wire.RunID != l.config.RunID {
		l.mu.Unlock()
		return Activation{}, ErrRouteCrossed
	}
	state := l.routes[wire.Reservation.ID]
	if state == nil {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Activation{}, ErrRouteStale
	}
	if wire != l.workflowRoute(state) {
		l.mu.Unlock()
		return Activation{}, ErrRouteCrossed
	}
	if providedBinding != state.bundle.binding {
		l.mu.Unlock()
		return Activation{}, ErrBindingMismatch
	}
	if l.stopped || state.authority == canceled || state.authority == terminal {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Activation{}, ErrRouteStale
	}
	if state.bundle.responseRunID != "" && state.bundle.responseRunID != delivery.TemporalRunID {
		l.mu.Unlock()
		return Activation{}, ErrRouteConflict
	}
	if state.authority == admitted {
		if state.activation.temporalRunID != delivery.TemporalRunID {
			l.mu.Unlock()
			return Activation{}, ErrRouteConflict
		}
		activation := Activation{ledger: l, state: state, data: state.activation, replay: true}
		l.mu.Unlock()
		return activation, nil
	}
	return l.consumeLocked(ctx, state, activationData{temporalRunID: delivery.TemporalRunID})
}

func (l *Ledger) PrepareNexus(ctx context.Context, workflow Activation, sourceInstructionID string) (NexusDispatch, error) {
	if err := primitive.ContextError(ctx, ErrInvalid); err != nil {
		return NexusDispatch{}, err
	}
	if !validRouteText(sourceInstructionID) {
		return NexusDispatch{}, ErrInvalid
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return NexusDispatch{}, err
	}
	workflowState, err := l.activationLocked(workflow, workflowRoute)
	if err != nil {
		l.mu.Unlock()
		return NexusDispatch{}, err
	}
	if l.stopped || workflowState.authority != admitted || workflowState.bundle.parentReleased {
		l.diagnoseLocked()
		l.mu.Unlock()
		return NexusDispatch{}, ErrRouteStale
	}
	key := sourceKey{workflowEntrypoint: workflowState.identity.EntrypointID, workflowOrdinal: workflowState.identity.Ordinal, sourceInstruction: sourceInstructionID}
	handler := workflowState.bundle.nexus[key]
	if handler == nil {
		l.mu.Unlock()
		return NexusDispatch{}, ErrRouteCrossed
	}
	if handler.authority == canceled || handler.authority == terminal {
		l.diagnoseLocked()
		l.mu.Unlock()
		return NexusDispatch{}, ErrRouteStale
	}
	wire := l.nexusRoute(handler)
	l.mu.Unlock()
	encoded, err := (routeCodec{maximumBytes: l.config.Limits.MaxHeaderBytes}).encode(wire)
	if err != nil {
		return NexusDispatch{}, err
	}
	preparedHeader := nexus.Header{reservedNexusHeader: base64.RawURLEncoding.EncodeToString(encoded)}
	if primitive.NexusHeaderBytes(preparedHeader) > l.config.Limits.MaxHeaderBytes {
		return NexusDispatch{}, ErrCapacity
	}
	return NexusDispatch{header: preparedHeader}, nil
}

func (l *Ledger) AdmitNexus(ctx context.Context, delivery NexusDelivery) (Activation, error) {
	encoded, err := decodeNexusHeader(delivery.Header, l.config.Limits.MaxHeaderBytes)
	if err != nil {
		return Activation{}, err
	}
	wire, err := (routeCodec{maximumBytes: l.config.Limits.MaxHeaderBytes}).decode(encoded, nexusRoute)
	if err != nil {
		return Activation{}, err
	}
	if !validRouteText(delivery.RequestID) {
		return Activation{}, ErrInvalid
	}
	if err := l.mu.LockContext(ctx, ErrInvalid); err != nil {
		return Activation{}, err
	}
	if wire.SessionID != l.config.SessionID || wire.RunID != l.config.RunID {
		l.mu.Unlock()
		return Activation{}, ErrRouteCrossed
	}
	state := l.routes[wire.Reservation.ID]
	if state == nil {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Activation{}, ErrRouteStale
	}
	if state.kind != nexusRoute || wire != l.nexusRoute(state) {
		l.mu.Unlock()
		return Activation{}, ErrRouteCrossed
	}
	if l.stopped {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Activation{}, ErrRouteStale
	}
	if state.authority == admitted {
		if state.activation.requestID != delivery.RequestID {
			l.mu.Unlock()
			return Activation{}, ErrRouteConflict
		}
		activation := Activation{ledger: l, state: state, data: state.activation, replay: true}
		l.mu.Unlock()
		return activation, nil
	}
	if state.bundle.parentReleased || state.authority == canceled || state.authority == terminal || state.bundle.workflow.authority == canceled || state.bundle.workflow.authority == terminal {
		l.diagnoseLocked()
		l.mu.Unlock()
		return Activation{}, ErrRouteStale
	}
	if state.bundle.workflow.authority != admitted {
		l.mu.Unlock()
		return Activation{}, ErrRouteCrossed
	}
	return l.consumeLocked(ctx, state, activationData{temporalRunID: state.bundle.workflow.activation.temporalRunID, requestID: delivery.RequestID})
}

// consumeLocked consumes the route's reservation and admits the route with the given activation
// data. It releases l.mu.
func (l *Ledger) consumeLocked(ctx context.Context, state *routeState, data activationData) (Activation, error) {
	coordinate, consumeErr := state.retained.handle.Consume(ctx)
	if consumeErr != nil || ctx.Err() != nil {
		state.authority = canceled
		if ctx.Err() != nil {
			l.mu.Unlock()
			return Activation{}, ctx.Err()
		}
		pending := l.pendingCancellationLocked([]*routeState{state})
		l.mu.Unlock()
		_ = l.cancel(ctx, pending)
		return Activation{}, ErrLifecycle
	}
	if !validActivationCoordinate(coordinate, l.config.RunID, state.identity.EntrypointID) {
		state.authority = canceled
		pending := l.pendingCancellationLocked([]*routeState{state})
		l.mu.Unlock()
		_ = l.cancel(ctx, pending)
		return Activation{}, ErrRouteConflict
	}
	data.coordinate = coordinate
	state.authority = admitted
	state.activation = data
	activation := Activation{ledger: l, state: state, data: data}
	l.mu.Unlock()
	return activation, nil
}

func (l *Ledger) workflowRoute(state *routeState) route {
	return route{Version: routeVersion, Kind: workflowRoute, SessionID: l.config.SessionID, RunID: l.config.RunID, Origin: state.bundle.origin, Reservation: state.identity, Binding: state.bundle.binding}
}

func (l *Ledger) nexusRoute(state *routeState) route {
	workflow := state.bundle.workflow
	return route{Version: routeVersion, Kind: nexusRoute, SessionID: l.config.SessionID, RunID: l.config.RunID, Origin: state.bundle.origin, Reservation: state.identity, Binding: state.bundle.binding, WorkflowReservation: workflow.identity.ID, WorkflowEntrypoint: state.source.workflowEntrypoint, WorkflowOrdinal: state.source.workflowOrdinal, WorkflowRunID: workflow.activation.temporalRunID, SourceInstructionID: state.source.sourceInstruction}
}

// StartBinding reads the workflow binding a StartWorkflow request names, and the header it carries
// beside the binding. It reads by descriptor, because requests are dynamic messages.
func StartBinding(message protoreflect.Message) (WorkflowBinding, protoreflect.Message, error) {
	fields := message.Descriptor().Fields()
	namespace := fields.ByName("namespace")
	workflowID := fields.ByName("workflow_id")
	workflowType := fields.ByName("workflow_type")
	taskQueue := fields.ByName("task_queue")
	header := fields.ByName("header")
	if namespace == nil || workflowID == nil || workflowType == nil || taskQueue == nil || header == nil || !message.Has(workflowType) || !message.Has(taskQueue) {
		return WorkflowBinding{}, nil, ErrInvalid
	}
	typeName := workflowType.Message().Fields().ByName("name")
	queueName := taskQueue.Message().Fields().ByName("name")
	if typeName == nil || queueName == nil {
		return WorkflowBinding{}, nil, ErrInvalid
	}
	binding := WorkflowBinding{
		Namespace:    message.Get(namespace).String(),
		WorkflowID:   message.Get(workflowID).String(),
		WorkflowType: message.Get(workflowType).Message().Get(typeName).String(),
		TaskQueue:    message.Get(taskQueue).Message().Get(queueName).String(),
	}
	var headerMessage protoreflect.Message
	if message.Has(header) {
		headerMessage = message.Get(header).Message()
	}
	return binding, headerMessage, nil
}

func hasReservedHeader(header protoreflect.Message, name string) bool {
	if header == nil || !header.IsValid() {
		return false
	}
	fields := header.Descriptor().Fields().ByName("fields")
	return fields != nil && header.Get(fields).Map().Has(protoreflect.ValueOfString(name).MapKey())
}

func injectReservedHeader(message protoreflect.Message, name string, encoded []byte) error {
	headerField := message.Descriptor().Fields().ByName("header")
	if headerField == nil {
		return ErrInvalid
	}
	header := message.Mutable(headerField).Message()
	fieldsField := header.Descriptor().Fields().ByName("fields")
	if fieldsField == nil || !fieldsField.IsMap() {
		return ErrInvalid
	}
	var payload protoreflect.Message
	if fieldsField.MapValue().Message() == (&commonpb.Payload{}).ProtoReflect().Descriptor() {
		payload = (&commonpb.Payload{}).ProtoReflect()
	} else {
		payload = dynamicpb.NewMessage(fieldsField.MapValue().Message())
	}
	metadataField := payload.Descriptor().Fields().ByName("metadata")
	dataField := payload.Descriptor().Fields().ByName("data")
	if metadataField == nil || dataField == nil || !metadataField.IsMap() {
		return ErrInvalid
	}
	payload.Mutable(metadataField).Map().Set(protoreflect.ValueOfString("encoding").MapKey(), protoreflect.ValueOfBytes([]byte(workflowRouteEncoding)))
	payload.Set(dataField, protoreflect.ValueOfBytes(bytes.Clone(encoded)))
	header.Mutable(fieldsField).Map().Set(protoreflect.ValueOfString(name).MapKey(), protoreflect.ValueOfMessage(payload))
	return nil
}

func decodeReservedHeader(header *commonpb.Header, name string, maximumBytes int) ([]byte, error) {
	if header == nil {
		return nil, ErrRouteMissing
	}
	payload, exists := header.Fields[name]
	if !exists {
		return nil, ErrRouteMissing
	}
	if payload == nil || len(payload.Metadata) != 1 || !bytes.Equal(payload.Metadata["encoding"], []byte(workflowRouteEncoding)) || len(payload.ExternalPayloads) != 0 {
		return nil, ErrRouteMalformed
	}
	if maximumBytes <= 0 || len(payload.Data) > maximumBytes {
		return nil, ErrRouteOversized
	}
	return bytes.Clone(payload.Data), nil
}

func decodeNexusHeader(header nexus.Header, maximumBytes int) ([]byte, error) {
	value, exists := header[reservedNexusHeader]
	if !exists || value == "" {
		return nil, ErrRouteMissing
	}
	if len(value) > maximumBytes {
		return nil, ErrRouteOversized
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != value {
		return nil, ErrRouteMalformed
	}
	return decoded, nil
}

func validActivationCoordinate(coordinate testpilot.Coordinate, runID, entrypointID string) bool {
	return coordinate.RunID == runID && coordinate.EntrypointID == entrypointID && validRouteText(coordinate.ActivationID)
}
