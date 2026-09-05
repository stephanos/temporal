package delivery

import (
	"context"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
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

func TestPrepareNexusPreservesHeaderAndFullValue(t *testing.T) {
	f := newFixture(t, "run", "session")
	workflow := admitWorkflow(t, f, "temporal-run")
	value := &umpirespb.Value{Value: &umpirespb.Value_Text{Text: "unchanged"}}
	header := nexus.Header{"application": "kept"}
	dispatch, err := f.ledger.PrepareNexus(context.Background(), workflow, "start-nexus", header, value)
	require.NoError(t, err)
	require.Equal(t, nexus.Header{"application": "kept"}, header)
	require.Equal(t, "kept", dispatch.Header().Get("application"))
	require.NotEmpty(t, dispatch.Header().Get(reservedNexusHeader))
	require.True(t, proto.Equal(value, dispatch.Value()))

	returnedHeader := dispatch.Header()
	returnedHeader.Set("application", "changed")
	returnedValue := dispatch.Value()
	returnedValue.Value = &umpirespb.Value_Text{Text: "changed"}
	require.Equal(t, "kept", dispatch.Header().Get("application"))
	require.Equal(t, "unchanged", dispatch.Value().GetText())

	header.Set(reservedNexusHeader, "anything")
	_, err = f.ledger.PrepareNexus(context.Background(), workflow, "start-nexus", header, value)
	require.ErrorIs(t, err, ErrReservedHeader)
}
