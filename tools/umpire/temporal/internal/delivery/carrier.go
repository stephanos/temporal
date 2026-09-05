package delivery

import (
	"bytes"
	"context"
	"encoding/base64"
	"maps"

	"github.com/nexus-rpc/sdk-go/nexus"
	commonpb "go.temporal.io/api/common/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	startWorkflowPath      = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	reservedWorkflowHeader = "temporal-umpire-reserved-workflow-v1"
	reservedNexusHeader    = "temporal-umpire-reserved-nexus-v1"
	workflowRouteEncoding  = "binary/temporal-umpire-reservation-route"
)

type WorkflowDelivery struct {
	Header                              *commonpb.Header
	Namespace, WorkflowID, WorkflowType string
	TaskQueue, TemporalRunID            string
}

type NexusDelivery struct {
	Header    nexus.Header
	RequestID string
}

type NexusDispatch struct {
	header nexus.Header
	value  *umpirespb.Value
}

type startRequestFields struct {
	namespace, workflowID, workflowType, taskQueue string
	header                                         protoreflect.Message
}

func (d NexusDispatch) Header() nexus.Header    { return maps.Clone(d.header) }
func (d NexusDispatch) Value() *umpirespb.Value { return proto.CloneOf(d.value) }

func (l *Ledger) PrepareRPC(ctx context.Context, carrier *Bundle, role string, method protoreflect.MethodDescriptor, request proto.Message, maximumBytes int64) (proto.Message, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if carrier == nil {
		return request, nil
	}
	if nilValue(method) || nilValue(request) || maximumBytes <= 0 || method.IsStreamingClient() || method.IsStreamingServer() || methodPath(method) != startWorkflowPath || request.ProtoReflect().Descriptor() != method.Input() {
		return nil, ErrInvalid
	}
	fields, err := startFields(request.ProtoReflect())
	if err != nil {
		return nil, err
	}
	if hasReservedWorkflowHeader(fields.header) {
		return nil, ErrReservedHeader
	}
	route, err := l.prepareWorkflowRoute(ctx, *carrier, role, method, fields)
	if err != nil {
		return nil, err
	}
	encoded, err := (routeCodec{maximumBytes: l.config.Limits.MaxHeaderBytes}).encode(route)
	if err != nil {
		return nil, err
	}
	prepared := proto.Clone(request)
	if err := injectWorkflowHeader(prepared.ProtoReflect(), encoded); err != nil {
		return nil, err
	}
	if int64(proto.Size(prepared)) > maximumBytes {
		return nil, ErrCapacity
	}
	return prepared, nil
}

func (l *Ledger) prepareWorkflowRoute(ctx context.Context, carrier Bundle, role string, method protoreflect.MethodDescriptor, fields startRequestFields) (route, error) {
	if err := l.mu.LockContext(ctx); err != nil {
		return route{}, err
	}
	state, err := l.bundleLocked(carrier)
	if err != nil {
		l.mu.Unlock()
		return route{}, err
	}
	if l.stopped || state.workflow.authority == canceled || state.workflow.authority == terminal {
		l.diagnoseLocked()
		l.mu.Unlock()
		return route{}, ErrRouteStale
	}
	if role != state.plan.EndpointRoleID || methodPath(method) != state.plan.Method {
		l.mu.Unlock()
		return route{}, ErrRouteCrossed
	}
	if fields.namespace != state.binding.Namespace || fields.workflowID != state.binding.WorkflowID || fields.workflowType != state.binding.WorkflowType || fields.taskQueue != state.binding.TaskQueue {
		l.mu.Unlock()
		return route{}, ErrBindingMismatch
	}
	route := l.workflowRoute(state.workflow)
	l.mu.Unlock()
	return route, nil
}

func (l *Ledger) AdmitWorkflow(ctx context.Context, delivery WorkflowDelivery) (Activation, error) {
	encoded, err := decodeWorkflowHeader(delivery.Header, l.config.Limits.MaxHeaderBytes)
	if err != nil {
		return Activation{}, err
	}
	wire, err := (routeCodec{maximumBytes: l.config.Limits.MaxHeaderBytes}).decode(encoded, workflowRoute)
	if err != nil {
		return Activation{}, err
	}
	providedBinding := binding{Namespace: delivery.Namespace, WorkflowID: delivery.WorkflowID, WorkflowType: delivery.WorkflowType, TaskQueue: delivery.TaskQueue}
	if !validBinding(providedBinding) || !validRouteText(delivery.TemporalRunID) {
		return Activation{}, ErrInvalid
	}
	if err := l.mu.LockContext(ctx); err != nil {
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
	state.authority = admitted
	state.activation = activationData{coordinate: coordinate, temporalRunID: delivery.TemporalRunID}
	activation := Activation{ledger: l, state: state, data: state.activation}
	l.mu.Unlock()
	return activation, nil
}

func (l *Ledger) PrepareNexus(ctx context.Context, workflow Activation, sourceInstructionID string, header nexus.Header, value *umpirespb.Value) (NexusDispatch, error) {
	if err := contextError(ctx); err != nil {
		return NexusDispatch{}, err
	}
	if !validRouteText(sourceInstructionID) {
		return NexusDispatch{}, ErrInvalid
	}
	if _, collision := header[reservedNexusHeader]; collision {
		return NexusDispatch{}, ErrReservedHeader
	}
	if err := l.mu.LockContext(ctx); err != nil {
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
	routeValue := base64.RawURLEncoding.EncodeToString(encoded)
	preparedHeader := maps.Clone(header)
	if preparedHeader == nil {
		preparedHeader = make(nexus.Header)
	}
	preparedHeader[reservedNexusHeader] = routeValue
	if nexusHeaderBytes(preparedHeader) > l.config.Limits.MaxHeaderBytes {
		return NexusDispatch{}, ErrCapacity
	}
	return NexusDispatch{header: preparedHeader, value: proto.CloneOf(value)}, nil
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
	if err := l.mu.LockContext(ctx); err != nil {
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
	state.authority = admitted
	state.activation = activationData{coordinate: coordinate, temporalRunID: state.bundle.workflow.activation.temporalRunID, requestID: delivery.RequestID}
	activation := Activation{ledger: l, state: state, data: state.activation}
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

func startFields(message protoreflect.Message) (startRequestFields, error) {
	fields := message.Descriptor().Fields()
	namespace := fields.ByName("namespace")
	workflowID := fields.ByName("workflow_id")
	workflowType := fields.ByName("workflow_type")
	taskQueue := fields.ByName("task_queue")
	header := fields.ByName("header")
	if namespace == nil || workflowID == nil || workflowType == nil || taskQueue == nil || header == nil || !message.Has(workflowType) || !message.Has(taskQueue) {
		return startRequestFields{}, ErrInvalid
	}
	typeName := workflowType.Message().Fields().ByName("name")
	queueName := taskQueue.Message().Fields().ByName("name")
	if typeName == nil || queueName == nil {
		return startRequestFields{}, ErrInvalid
	}
	result := startRequestFields{
		namespace:    message.Get(namespace).String(),
		workflowID:   message.Get(workflowID).String(),
		workflowType: message.Get(workflowType).Message().Get(typeName).String(),
		taskQueue:    message.Get(taskQueue).Message().Get(queueName).String(),
	}
	if message.Has(header) {
		result.header = message.Get(header).Message()
	}
	return result, nil
}

func hasReservedWorkflowHeader(header protoreflect.Message) bool {
	if header == nil || !header.IsValid() {
		return false
	}
	fields := header.Descriptor().Fields().ByName("fields")
	return fields != nil && header.Get(fields).Map().Has(protoreflect.ValueOfString(reservedWorkflowHeader).MapKey())
}

func injectWorkflowHeader(message protoreflect.Message, encoded []byte) error {
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
	header.Mutable(fieldsField).Map().Set(protoreflect.ValueOfString(reservedWorkflowHeader).MapKey(), protoreflect.ValueOfMessage(payload))
	return nil
}

func decodeWorkflowHeader(header *commonpb.Header, maximumBytes int) ([]byte, error) {
	if header == nil {
		return nil, ErrRouteMissing
	}
	payload, exists := header.Fields[reservedWorkflowHeader]
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

func nexusHeaderBytes(header nexus.Header) int {
	size := 0
	for key, value := range header {
		size += len(key) + len(value)
	}
	return size
}

func methodPath(method protoreflect.MethodDescriptor) string {
	return "/" + string(method.Parent().FullName()) + "/" + string(method.Name())
}

func validActivationCoordinate(coordinate umpire.Coordinate, runID, entrypointID string) bool {
	return coordinate.RunID == runID && coordinate.EntrypointID == entrypointID && validRouteText(coordinate.ActivationID)
}
