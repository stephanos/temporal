package workflow

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	persistencespb "go.temporal.io/server/api/persistence/v1"
	"go.temporal.io/server/chasm"
	"go.temporal.io/server/common/definition"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/metrics/metricstest"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/telemetry"
	"go.temporal.io/server/service/history/consts"
	historyi "go.temporal.io/server/service/history/interfaces"
	"go.uber.org/mock/gomock"
)

func TestForceTerminateWorkflowRecordsMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	mutableState := historyi.NewMockMutableState(ctrl)
	mutableState.EXPECT().GetStartedWorkflowTask().Return(nil)
	mutableState.EXPECT().AddWorkflowExecutionTerminatedEvent(
		"force terminate",
		nil,
		consts.IdentityHistoryService,
		false,
		nil,
	).Return(&historypb.HistoryEvent{
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionTerminatedEventAttributes{
			WorkflowExecutionTerminatedEventAttributes: &historypb.WorkflowExecutionTerminatedEventAttributes{},
		},
	}, nil)
	mutableState.EXPECT().GetWorkflowKey().Return(definition.WorkflowKey{
		NamespaceID: "namespace-id", WorkflowID: "workflow-id", RunID: "run-id",
	})
	mutableState.EXPECT().GetNamespaceEntry().Return(namespace.NewLocalNamespaceForTest(
		&persistencespb.NamespaceInfo{Name: "test-namespace"},
		nil,
		"active",
	))

	metricsHandler := metricstest.NewCaptureHandler()
	capture := metricsHandler.StartCapture()
	defer metricsHandler.StopCapture(capture)

	err := ForceTerminateWorkflow(
		mutableState,
		"force terminate",
		nil,
		consts.IdentityHistoryService,
		false,
		nil,
		metricsHandler,
		chasm.ExecutionForceTerminationReasonEventBatchSizeExceedsLimit,
	)
	require.NoError(t, err)

	recordings := capture.Snapshot()[metrics.ExecutionForceTerminations.Name()]
	require.Len(t, recordings, 1)
	require.Equal(t, int64(1), recordings[0].Value)
	require.Equal(t, "test-namespace", recordings[0].Tags["namespace"])
	require.Equal(t, string(chasm.WorkflowArchetype), recordings[0].Tags["archetype"])
	require.Equal(t, string(chasm.ExecutionForceTerminationReasonEventBatchSizeExceedsLimit), recordings[0].Tags["reason"])
}

func TestForceTerminateWorkflowWithContextEmitsCloseObservation(t *testing.T) {
	ctrl := gomock.NewController(t)
	mutableState := historyi.NewMockMutableState(ctrl)
	mutableState.EXPECT().GetStartedWorkflowTask().Return(nil)
	mutableState.EXPECT().AddWorkflowExecutionTerminatedEvent(
		"force terminate", nil, consts.IdentityHistoryService, false, nil,
	).Return(&historypb.HistoryEvent{
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionTerminatedEventAttributes{
			WorkflowExecutionTerminatedEventAttributes: &historypb.WorkflowExecutionTerminatedEventAttributes{},
		},
	}, nil)
	mutableState.EXPECT().GetWorkflowKey().Return(definition.WorkflowKey{
		NamespaceID: "namespace-id", WorkflowID: "workflow-id", RunID: "run-id",
	})
	mutableState.EXPECT().GetNamespaceEntry().Return(nil)
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { require.NoError(t, provider.Shutdown(t.Context())) }()
	ctx, span := provider.Tracer("test").Start(t.Context(), "force terminate")
	metricsHandler := metricstest.NewCaptureHandler()
	capture := metricsHandler.StartCapture()
	defer metricsHandler.StopCapture(capture)

	require.NoError(t, ForceTerminateWorkflowWithContext(
		ctx, mutableState, "force terminate", nil, consts.IdentityHistoryService, false, nil,
		metricsHandler, chasm.ExecutionForceTerminationReasonEventBatchSizeExceedsLimit,
	))
	span.End()
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Len(t, spans[0].Events, 1)
	require.Equal(t, telemetry.EventWorkflowExecutionClosed, spans[0].Events[0].Name)
	require.Equal(t, []attribute.KeyValue{
		telemetry.AttrWorkflowID.String("workflow-id"),
		telemetry.AttrRunID.String("run-id"),
		telemetry.AttrNamespaceID.String("namespace-id"),
		telemetry.AttrWorkflowCloseOutcome.String(telemetry.WorkflowCloseOutcomeTerminated),
		telemetry.AttrWorkflowSuccessorRunID.String(""),
	}, spans[0].Events[0].Attributes)
	require.Len(t, capture.Snapshot()[metrics.ExecutionForceTerminations.Name()], 1)
}

func TestForceTerminateWorkflowWithContextRejectsFailedWrite(t *testing.T) {
	ctrl := gomock.NewController(t)
	mutableState := historyi.NewMockMutableState(ctrl)
	mutableState.EXPECT().GetStartedWorkflowTask().Return(nil)
	writeErr := errors.New("write failed")
	mutableState.EXPECT().AddWorkflowExecutionTerminatedEvent(
		"force terminate", nil, consts.IdentityHistoryService, false, nil,
	).Return(nil, writeErr)

	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { require.NoError(t, provider.Shutdown(t.Context())) }()
	ctx, span := provider.Tracer("test").Start(t.Context(), "force terminate")
	metricsHandler := metricstest.NewCaptureHandler()
	capture := metricsHandler.StartCapture()
	defer metricsHandler.StopCapture(capture)

	require.ErrorIs(t, ForceTerminateWorkflowWithContext(
		ctx, mutableState, "force terminate", nil, consts.IdentityHistoryService, false, nil,
		metricsHandler, chasm.ExecutionForceTerminationReasonEventBatchSizeExceedsLimit,
	), writeErr)
	span.End()
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Empty(t, spans[0].Events)
	require.Empty(t, capture.Snapshot()[metrics.ExecutionForceTerminations.Name()])
}
