package delivery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestPrepareRPCClonesAndPreservesApplicationRequest(t *testing.T) {
	f := newFixture(t, "run", "session")
	request := workflowRequest(f)
	request.Header = &commonpb.Header{Fields: map[string]*commonpb.Payload{"application": {Data: []byte("kept")}}}
	request.Input = &commonpb.Payloads{Payloads: []*commonpb.Payload{{Data: []byte("input")}}}
	snapshot := proto.CloneOf(request)

	preparedMessage, err := f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startMethod(t), request, 1<<20)
	require.NoError(t, err)
	prepared := preparedMessage.(*workflowservice.StartWorkflowExecutionRequest)
	require.NotSame(t, request, prepared)
	require.True(t, proto.Equal(snapshot, request))
	require.Equal(t, []byte("kept"), prepared.Header.Fields["application"].Data)
	require.Equal(t, []byte("input"), prepared.Input.Payloads[0].Data)
	require.Contains(t, prepared.Header.Fields, reservedWorkflowHeader)

	delete(prepared.Header.Fields, reservedWorkflowHeader)
	require.True(t, proto.Equal(request, prepared))
}

func TestPrepareRPCRejectsCollisionBindingAndByteErrors(t *testing.T) {
	for name, mutate := range map[string]func(*fixture, *workflowservice.StartWorkflowExecutionRequest){
		"reserved collision": func(_ *fixture, request *workflowservice.StartWorkflowExecutionRequest) {
			request.Header = &commonpb.Header{Fields: map[string]*commonpb.Payload{reservedWorkflowHeader: {Data: []byte("anything")}}}
		},
		"namespace":   func(_ *fixture, request *workflowservice.StartWorkflowExecutionRequest) { request.Namespace = "other" },
		"workflow id": func(_ *fixture, request *workflowservice.StartWorkflowExecutionRequest) { request.WorkflowId = "other" },
		"workflow type": func(_ *fixture, request *workflowservice.StartWorkflowExecutionRequest) {
			request.WorkflowType.Name = "other"
		},
		"task queue": func(_ *fixture, request *workflowservice.StartWorkflowExecutionRequest) {
			request.TaskQueue.Name = "other"
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, "run", "session")
			request := workflowRequest(f)
			mutate(f, request)
			_, err := f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startMethod(t), request, 1<<20)
			if name == "reserved collision" {
				require.ErrorIs(t, err, ErrReservedHeader)
			} else {
				require.ErrorIs(t, err, ErrBindingMismatch)
			}
		})
	}

	f := newFixture(t, "run", "session")
	prepared, err := f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startMethod(t), workflowRequest(f), 1<<20)
	require.NoError(t, err)
	_, err = f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", startMethod(t), workflowRequest(f), int64(proto.Size(prepared)-1))
	require.ErrorIs(t, err, ErrCapacity)
}

// StartBinding serves the composite Driver and PrepareRPC alike, so it reads a dynamic request and
// rejects one that is not a StartWorkflow request or lacks a binding field.
func TestStartBindingReadsDynamicStartRequestsOnly(t *testing.T) {
	f := newFixture(t, "run", "session")
	request := workflowRequest(f)
	request.Header = &commonpb.Header{Fields: map[string]*commonpb.Payload{"application": {Data: []byte("kept")}}}
	encoded, err := proto.Marshal(request)
	require.NoError(t, err)
	dynamicRequest := dynamicpb.NewMessage(startMethod(t).Input())
	require.NoError(t, proto.Unmarshal(encoded, dynamicRequest))
	binding, header, err := StartBinding(dynamicRequest)
	require.NoError(t, err)
	require.Equal(t, f.binding, binding)
	require.True(t, header.IsValid())

	for name, message := range map[string]proto.Message{
		"not a start request": &workflowservice.SignalWorkflowExecutionRequest{Namespace: "namespace"},
		"workflow type":       &workflowservice.StartWorkflowExecutionRequest{Namespace: "namespace", WorkflowId: "workflow-id", TaskQueue: request.TaskQueue},
		"task queue":          &workflowservice.StartWorkflowExecutionRequest{Namespace: "namespace", WorkflowId: "workflow-id", WorkflowType: request.WorkflowType},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := StartBinding(message.ProtoReflect())
			require.ErrorIs(t, err, ErrInvalid)
		})
	}
}

func TestPrepareRPCPassesUnrelatedCallsThrough(t *testing.T) {
	f := newFixture(t, "run", "session")
	request := workflowRequest(f)
	prepared, err := f.ledger.PrepareRPC(context.Background(), nil, "temporal", startMethod(t), request, 1)
	require.NoError(t, err)
	require.Same(t, request, prepared)
}

func TestPrepareRPCSupportsConstructedDynamicMessage(t *testing.T) {
	f := newFixture(t, "run", "session")
	method := startMethod(t)
	request := workflowRequest(f)
	encoded, err := proto.Marshal(request)
	require.NoError(t, err)
	dynamicRequest := dynamicpb.NewMessage(method.Input())
	require.NoError(t, proto.Unmarshal(encoded, dynamicRequest))
	snapshot := proto.Clone(dynamicRequest)

	prepared, err := f.ledger.PrepareRPC(context.Background(), &f.bundle, "temporal", method, dynamicRequest, 1<<20)
	require.NoError(t, err)
	require.IsType(t, &dynamicpb.Message{}, prepared)
	require.True(t, proto.Equal(snapshot, dynamicRequest))
	require.Equal(t, method.Input(), prepared.ProtoReflect().Descriptor())
	decoded := &workflowservice.StartWorkflowExecutionRequest{}
	preparedBytes, err := proto.Marshal(prepared)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(preparedBytes, decoded))
	require.Contains(t, decoded.Header.Fields, reservedWorkflowHeader)
}

func TestInvalidDeliveriesRejectBeforeReservationConsumption(t *testing.T) {
	f := newFixture(t, "run", "session")
	validHeader := workflowHeader(t, f)
	for name, test := range map[string]struct {
		header *commonpb.Header
		err    error
	}{
		"missing":   {err: ErrRouteMissing},
		"malformed": {header: &commonpb.Header{Fields: map[string]*commonpb.Payload{reservedWorkflowHeader: {Metadata: map[string][]byte{"encoding": []byte("wrong")}, Data: []byte("route")}}}, err: ErrRouteMalformed},
		"oversized": {header: &commonpb.Header{Fields: map[string]*commonpb.Payload{reservedWorkflowHeader: {Metadata: map[string][]byte{"encoding": []byte(workflowRouteEncoding)}, Data: make([]byte, f.ledger.config.Limits.MaxHeaderBytes+1)}}}, err: ErrRouteOversized},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := f.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: test.header, Namespace: f.binding.Namespace, WorkflowID: f.binding.WorkflowID, WorkflowType: f.binding.WorkflowType, TaskQueue: f.binding.TaskQueue, TemporalRunID: "temporal-run"})
			require.ErrorIs(t, err, test.err)
			require.Zero(t, f.workflow.consumeCount.Load())
		})
	}

	validPayload := validHeader.Fields[reservedWorkflowHeader]
	validPayload.Data[0] = '['
	_, err := f.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: validHeader, Namespace: f.binding.Namespace, WorkflowID: f.binding.WorkflowID, WorkflowType: f.binding.WorkflowType, TaskQueue: f.binding.TaskQueue, TemporalRunID: "temporal-run"})
	require.Error(t, err)
	require.Zero(t, f.workflow.consumeCount.Load())
}

func TestStartResponseMustAgreeWithFirstWorkflowDelivery(t *testing.T) {
	f := newFixture(t, "run", "session")
	admitWorkflow(t, f, "temporal-run")
	require.NoError(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartWorkflowExecutionResponse{RunId: "temporal-run"}))
	require.ErrorIs(t, f.ledger.PinStartResponse(context.Background(), f.bundle, &workflowservice.StartWorkflowExecutionResponse{RunId: "crossed"}), ErrRouteConflict)

	other := newFixture(t, "run-two", "session-two")
	require.NoError(t, other.ledger.PinStartResponse(context.Background(), other.bundle, &workflowservice.StartWorkflowExecutionResponse{RunId: "response-first"}))
	_, err := other.ledger.AdmitWorkflow(context.Background(), WorkflowDelivery{Header: workflowHeader(t, other), Namespace: other.binding.Namespace, WorkflowID: other.binding.WorkflowID, WorkflowType: other.binding.WorkflowType, TaskQueue: other.binding.TaskQueue, TemporalRunID: "crossed"})
	require.ErrorIs(t, err, ErrRouteConflict)
	require.Zero(t, other.workflow.consumeCount.Load())
}

// A prepared dispatch carries the route alone; the worker merges it into the Case's own header.
func TestPrepareNexusReturnsAnIndependentRouteHeader(t *testing.T) {
	f := newFixture(t, "run", "session")
	workflow := admitWorkflow(t, f, "temporal-run")
	dispatch, err := f.ledger.PrepareNexus(context.Background(), workflow, "start-nexus")
	require.NoError(t, err)
	header := dispatch.Header()
	require.Len(t, header, 1)
	route := header.Get(reservedNexusHeader)
	require.NotEmpty(t, route)

	header.Set(reservedNexusHeader, "changed")
	require.Equal(t, route, dispatch.Header().Get(reservedNexusHeader))
}
