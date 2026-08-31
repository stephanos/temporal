package nexus

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	nexussdk "github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	"go.temporal.io/server/tools/umpire/runner"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/local"
)

func TestLiveParticipantRealizesOneForceCloseAndClosesOperationalSources(t *testing.T) {
	input := admitCallerClosureSet(t)
	ctx, cancel := context.WithTimeout(context.Background(), 135*time.Second)
	defer cancel()

	output, err := runner.Run(
		ctx, input, callerClosureInputBinding(),
		"umpire.local.caller-closure.live-force-close", Binding{},
	)
	require.NoError(t, err)
	run := output.ExperimentRun()
	require.Len(t, run.ControlAttempts, 1)
	require.Equal(t, "accepted", run.ControlAttempts[0].Status)
	require.EqualValues(t, "0", run.Cleanup.OpenHandleCount)

	eventCounts := map[string]int{}
	cancellationCallbacks := []string{}
	for _, fact := range output.RawEvidence().Facts {
		for _, field := range fact.Fields {
			switch field.FieldDefinitionID {
			case umpireruntime.EvidenceFieldEventType:
				value, ok := field.Value.(string)
				require.True(t, ok)
				eventCounts[value]++
			case umpireruntime.EvidenceFieldCancellationCallbackCount:
				cancellationCallbacks = append(cancellationCallbacks, fmt.Sprint(field.Value))
			}
		}
	}
	require.Equal(t, "succeeded", run.OperationalStatus)
	require.Equal(t, 1, eventCounts["temporal.history.NexusOperationScheduled"])
	require.Equal(t, 1, eventCounts["temporal.history.NexusOperationStarted"])
	require.Equal(t, 1, eventCounts["temporal.history.NexusOperationCancelRequested"])
	require.Equal(t, 1, eventCounts["temporal.history.NexusOperationCancelRequestCompleted"])
	require.Equal(t, 1, eventCounts["temporal.history.WorkflowExecutionCanceled"])
	require.Equal(t, []string{"1"}, cancellationCallbacks)
	require.Empty(t, rawFactsByKind(output.RawEvidence().Facts, duplicateObservationFactKind))
}

func TestLiveFaultedParticipantCompletesOneCancellationBeforeOneDuplicateObservation(t *testing.T) {
	input := admitCallerClosureDuplicateDeliverySet(t)
	ctx, cancel := context.WithTimeout(context.Background(), 135*time.Second)
	defer cancel()

	output, err := runner.Run(
		ctx, input, callerClosureDuplicateDeliveryInputBinding(),
		"umpire.local.caller-closure.live-duplicate-delivery", Binding{},
	)
	require.NoError(t, err)
	run := output.ExperimentRun()
	require.Equal(t, "succeeded", run.OperationalStatus)
	require.Len(t, run.ControlAttempts, 1)
	require.Equal(t, "accepted", run.ControlAttempts[0].Status)
	require.EqualValues(t, "0", run.Cleanup.OpenHandleCount)

	eventCounts := map[string]int{}
	for _, fact := range output.RawEvidence().Facts {
		for _, field := range fact.Fields {
			if field.FieldDefinitionID != umpireruntime.EvidenceFieldEventType {
				continue
			}
			value, ok := field.Value.(string)
			require.True(t, ok)
			eventCounts[value]++
		}
	}
	require.Equal(t, 1, eventCounts["temporal.history.NexusOperationCancelRequested"])
	require.Equal(t, 1, eventCounts["temporal.history.NexusOperationCancelRequestCompleted"])
	require.Len(t, rawFactsByKind(output.RawEvidence().Facts, duplicateObservationFactKind), 1)
}

func TestParticipantAdmitsOnlyTheExactCheckedRequest(t *testing.T) {
	_, err := NewParticipant(umpireruntime.CheckedRunRequest{})
	require.Error(t, err)

	participant, err := NewParticipant(checkedCallerClosureRequest(t, "exact-request"))
	require.NoError(t, err)
	require.NotNil(t, participant)

	faulted, err := NewParticipant(checkedDuplicateDeliveryRequest(t, "exact-faulted-request"))
	require.NoError(t, err)
	require.NotNil(t, faulted)
}

func TestParticipantRejectsWrongCorrelationAndDuplicateCommandsBeforeAdapterIO(t *testing.T) {
	request := checkedCallerClosureRequest(t, "command-identity")
	adapter := &recordingCommandAdapter{}
	participant, err := newParticipant(request, adapter)
	require.NoError(t, err)
	otherRequest := checkedCallerClosureRequest(t, "other-command-identity")
	wrongCommand, ok := otherRequest.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)

	wrong := participant.Prepare(context.Background(), inertEnvironment{}, wrongCommand)
	require.Equal(t, umpireruntime.ReceiptUnsupported, wrong.Status())
	require.Empty(t, adapter.calls)

	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	first := participant.Prepare(context.Background(), inertEnvironment{}, prepare)
	require.Equal(t, umpireruntime.ReceiptAccepted, first.Status())
	require.Equal(t, []umpireruntime.CommandKind{umpireruntime.CommandPrepare}, adapter.calls)

	duplicate := participant.Prepare(context.Background(), inertEnvironment{}, prepare)
	require.Equal(t, umpireruntime.ReceiptUnsupported, duplicate.Status())
	require.Equal(t, []umpireruntime.CommandKind{umpireruntime.CommandPrepare}, adapter.calls)

	realize, ok := request.Command(umpireruntime.CommandRealize)
	require.True(t, ok)
	require.Equal(t, umpireruntime.ReceiptAccepted,
		participant.Realize(context.Background(), inertEnvironment{}, realize).Status())
	observe, ok := request.Command(umpireruntime.CommandObserve)
	require.True(t, ok)
	require.Equal(t, umpireruntime.ReceiptAccepted,
		participant.Observe(context.Background(), inertEnvironment{}, observe).Status())
	cleanup, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	require.Equal(t, umpireruntime.ReceiptAccepted,
		participant.Cleanup(context.Background(), inertEnvironment{}, cleanup).Status())
	require.Equal(t, []umpireruntime.CommandKind{
		umpireruntime.CommandPrepare,
		umpireruntime.CommandRealize,
		umpireruntime.CommandObserve,
		umpireruntime.CommandCleanup,
	}, adapter.calls)

	secondRealize := participant.Realize(context.Background(), inertEnvironment{}, realize)
	require.Equal(t, umpireruntime.ReceiptUnsupported, secondRealize.Status())
	require.Len(t, adapter.calls, 4)
}

func TestParticipantCancellationIsOperationalAndPerformsNoAdapterIO(t *testing.T) {
	request := checkedCallerClosureRequest(t, "canceled-command")
	adapter := &recordingCommandAdapter{}
	participant, err := newParticipant(request, adapter)
	require.NoError(t, err)
	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	receipt := participant.Prepare(ctx, inertEnvironment{}, prepare)
	require.Equal(t, umpireruntime.ReceiptCanceled, receipt.Status())
	require.Empty(t, adapter.calls)
	require.Empty(t, receipt.Facts())
	require.Empty(t, receipt.AcquiredResources())
	require.Empty(t, receipt.ReleasedResources())
}

func TestParticipantCancellationBeforeRealizationIssuesNoControlRequest(t *testing.T) {
	request := checkedCallerClosureRequest(t, "canceled-realization")
	adapter := &recordingCommandAdapter{}
	participant, err := newParticipant(request, adapter)
	require.NoError(t, err)
	prepare, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	require.Equal(t, umpireruntime.ReceiptAccepted,
		participant.Prepare(context.Background(), inertEnvironment{}, prepare).Status())
	realize, ok := request.Command(umpireruntime.CommandRealize)
	require.True(t, ok)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	receipt := participant.Realize(ctx, inertEnvironment{}, realize)
	require.Equal(t, umpireruntime.ReceiptCanceled, receipt.Status())
	require.False(t, receipt.ControlAttempted())
	require.Equal(t, []umpireruntime.CommandKind{umpireruntime.CommandPrepare}, adapter.calls)
}

func TestRealizationContributesOneDuplicateObservationOnlyForTheFaultedProgram(t *testing.T) {
	for _, test := range []struct {
		name          string
		request       func(*testing.T) umpireruntime.CheckedRunRequest
		wantSynthetic int
	}{
		{
			name: "normal",
			request: func(t *testing.T) umpireruntime.CheckedRunRequest {
				return checkedCallerClosureRequest(t, "normal-realization")
			},
		},
		{
			name: "faulted",
			request: func(t *testing.T) umpireruntime.CheckedRunRequest {
				return checkedDuplicateDeliveryRequest(t, "faulted-realization")
			},
			wantSynthetic: 1,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := test.request(t)
			adapter, err := newSDKCommandAdapter(request)
			require.NoError(t, err)
			operation := newCallerClosureOperation(adapter.operationCorrelation)
			_, err = operation.Start(
				context.Background(), adapter.operationCorrelation, nexussdk.StartOperationOptions{},
			)
			require.NoError(t, err)
			completed := make(chan error, 1)
			sdkClient := &recordingRealizationClient{}
			sdkClient.onCancel = func() error {
				if err := operation.Cancel(
					context.Background(), adapter.operationCorrelation, nexussdk.CancelOperationOptions{},
				); err != nil {
					return err
				}
				completed <- temporal.NewCanceledError()
				return nil
			}
			environment := &recordingRealizationEnvironment{sdkClient: sdkClient}
			adapter.environment = environment
			adapter.operation = operation
			adapter.run = fixedWorkflowRun{workflowID: "workflow", runID: "run"}
			adapter.runDone = completed
			command, ok := request.Command(umpireruntime.CommandRealize)
			require.True(t, ok)
			premature, prematureErr := adapter.contributeDuplicateObservation(
				command, adapter.correlations(),
			)
			require.Empty(t, premature)
			if test.wantSynthetic == 1 {
				require.Error(t, prematureErr)
			} else {
				require.NoError(t, prematureErr)
			}

			receipt := adapter.Realize(context.Background(), inertEnvironment{}, command)
			require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
			require.True(t, receipt.ControlAttempted())
			synthetic := receiptFactsByKind(receipt, duplicateObservationFactKind)
			require.Len(t, synthetic, test.wantSynthetic)
			if test.wantSynthetic == 1 {
				require.Equal(t, umpireruntime.EvidenceSourceParticipantOutput,
					synthetic[0].SourceDefinitionID())
				require.Equal(t, adapter.operationCorrelation,
					factField(t, synthetic[0], umpireruntime.EvidenceFieldOperationCorrelationID))
				_, err := adapter.contributeDuplicateObservation(command, adapter.correlations())
				require.Error(t, err)
			}
			require.Equal(t, 1, sdkClient.cancellations)
			require.Equal(t, 0, sdkClient.historyReads)
			require.Equal(t, []uint64{1}, environment.controlCounts)

			duplicate := adapter.Realize(context.Background(), inertEnvironment{}, command)
			require.Equal(t, umpireruntime.ReceiptUnsupported, duplicate.Status())
			require.Empty(t, receiptFactsByKind(duplicate, duplicateObservationFactKind))
			require.Equal(t, 1, sdkClient.cancellations)
			require.Equal(t, 0, sdkClient.historyReads)
		})
	}
}

func TestFaultedRealizationEmitsNoSyntheticObservationWithoutCompletedCancellation(t *testing.T) {
	for _, test := range []struct {
		name       string
		suffix     string
		completion error
		cancelErr  error
		cancelWait bool
		wantStatus umpireruntime.ReceiptStatus
	}{
		{
			name: "cancellation request failed", suffix: "request-failed", cancelErr: errors.New("cancel failed"),
			wantStatus: umpireruntime.ReceiptFailed,
		},
		{
			name: "completion receipt failed", suffix: "receipt-failed", completion: errors.New("completion failed"),
			wantStatus: umpireruntime.ReceiptFailed,
		},
		{
			name: "completion receipt missing", suffix: "receipt-missing", cancelWait: true,
			wantStatus: umpireruntime.ReceiptCanceled,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := checkedDuplicateDeliveryRequest(t, "incomplete-realization-"+test.suffix)
			adapter, err := newSDKCommandAdapter(request)
			require.NoError(t, err)
			operation := newCallerClosureOperation(adapter.operationCorrelation)
			_, err = operation.Start(
				context.Background(), adapter.operationCorrelation, nexussdk.StartOperationOptions{},
			)
			require.NoError(t, err)
			adapter.operation = operation
			adapter.run = fixedWorkflowRun{workflowID: "workflow", runID: "run"}
			adapter.runDone = make(chan error, 1)
			sdkClient := &recordingRealizationClient{}
			environment := &recordingRealizationEnvironment{sdkClient: sdkClient}
			adapter.environment = environment
			sdkClient.onCancel = func() error {
				if test.cancelErr != nil {
					return test.cancelErr
				}
				if err := operation.Cancel(
					context.Background(), adapter.operationCorrelation, nexussdk.CancelOperationOptions{},
				); err != nil {
					return err
				}
				if !test.cancelWait {
					adapter.runDone <- test.completion
				}
				return nil
			}
			ctx := context.Background()
			cancel := func() {}
			if test.cancelWait {
				ctx, cancel = context.WithTimeout(ctx, time.Millisecond)
			}
			defer cancel()
			command, ok := request.Command(umpireruntime.CommandRealize)
			require.True(t, ok)

			receipt := adapter.Realize(ctx, inertEnvironment{}, command)
			require.Equal(t, test.wantStatus, receipt.Status())
			require.Empty(t, receiptFactsByKind(receipt, duplicateObservationFactKind))
			require.False(t, adapter.forceCloseAcknowledged)
			require.NotEqual(t, duplicateObservationContributed, adapter.duplicateObservation)
			require.Equal(t, 1, sdkClient.cancellations)
			require.Equal(t, 0, sdkClient.historyReads)
		})
	}
}

func TestFaultedRealizationRejectsAnUnstartedCancellationWithoutSyntheticObservation(t *testing.T) {
	request := checkedDuplicateDeliveryRequest(t, "rejected-unstarted-cancellation")
	adapter, err := newSDKCommandAdapter(request)
	require.NoError(t, err)
	operation := newCallerClosureOperation(adapter.operationCorrelation)
	adapter.operation = operation
	adapter.run = fixedWorkflowRun{workflowID: "workflow", runID: "run"}
	adapter.runDone = make(chan error, 1)
	sdkClient := &recordingRealizationClient{}
	adapter.environment = &recordingRealizationEnvironment{sdkClient: sdkClient}
	sdkClient.onCancel = func() error {
		return operation.Cancel(
			context.Background(), adapter.operationCorrelation, nexussdk.CancelOperationOptions{},
		)
	}
	command, ok := request.Command(umpireruntime.CommandRealize)
	require.True(t, ok)

	receipt := adapter.Realize(context.Background(), inertEnvironment{}, command)
	require.Equal(t, umpireruntime.ReceiptRejected, receipt.Status())
	require.Empty(t, receiptFactsByKind(receipt, duplicateObservationFactKind))
	require.False(t, adapter.forceCloseAcknowledged)
	require.NotEqual(t, duplicateObservationContributed, adapter.duplicateObservation)
	require.Equal(t, 1, sdkClient.cancellations)
}

func TestWorkerReadinessCancellationEmitsNoReadinessClaim(t *testing.T) {
	sdkClient := &unreadyClient{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForWorkerReadiness(ctx, sdkClient, "task-queue")
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 2, sdkClient.descriptions)
}

func TestSDKAndContextFailuresRemainOperationalReceipts(t *testing.T) {
	request := checkedCallerClosureRequest(t, "operational-failures")
	command, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	timedOut, timeoutCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer timeoutCancel()

	for _, test := range []struct {
		name   string
		ctx    context.Context
		err    error
		status umpireruntime.ReceiptStatus
		code   string
	}{
		{name: "SDK failure", ctx: context.Background(), err: errors.New("sdk failed"), status: umpireruntime.ReceiptFailed, code: runtimeCodeFailed},
		{name: "handler crash", ctx: context.Background(), err: errors.New("handler panicked"), status: umpireruntime.ReceiptFailed, code: runtimeCodeFailed},
		{name: "canceled", ctx: canceled, err: context.Canceled, status: umpireruntime.ReceiptCanceled, code: runtimeCodeCanceled},
		{name: "timed out", ctx: timedOut, err: context.DeadlineExceeded, status: umpireruntime.ReceiptCanceled, code: runtimeCodeTimedOut},
	} {
		t.Run(test.name, func(t *testing.T) {
			receipt := adapterFailureReceipt(test.ctx, command, test.err, nil, nil, adapterCorrelations{})
			require.Equal(t, test.status, receipt.Status())
			require.Equal(t, test.code, receiptField(t, receipt, umpireruntime.EvidenceFieldErrorCode))
			require.Empty(t, receipt.AcquiredResources())
			require.Empty(t, receipt.ReleasedResources())
		})
	}
}

func TestHandlerPanicsAreNonRetryable(t *testing.T) {
	operation := &panickingCallerClosureOperation{}
	guarded := &guardedCallerClosureOperation{delegate: operation}

	result, err := guarded.Start(context.Background(), "input", nexussdk.StartOperationOptions{})
	require.Nil(t, result)
	requireHandlerError(t, err, nexussdk.HandlerErrorTypeInternal)
	require.EqualValues(t, 1, operation.startCount)

	err = guarded.Cancel(context.Background(), "token", nexussdk.CancelOperationOptions{})
	requireHandlerError(t, err, nexussdk.HandlerErrorTypeInternal)
	require.EqualValues(t, 1, operation.cancelCount)
}

func TestHandlerBindsIdentityAndHandlesDuplicateCancellationIdempotently(t *testing.T) {
	operation := newCallerClosureOperation("operation.expected")

	result, err := operation.Start(context.Background(), "operation.other", nexussdk.StartOperationOptions{})
	require.Nil(t, result)
	requireHandlerError(t, err, nexussdk.HandlerErrorTypeBadRequest)
	require.EqualValues(t, 0, operation.startCount)

	result, err = operation.Start(context.Background(), "operation.expected", nexussdk.StartOperationOptions{})
	require.NoError(t, err)
	require.Equal(t,
		&nexussdk.HandlerStartOperationResultAsync{OperationToken: "operation.expected"},
		result,
	)
	result, err = operation.Start(context.Background(), "operation.expected", nexussdk.StartOperationOptions{})
	require.Nil(t, result)
	requireHandlerError(t, err, nexussdk.HandlerErrorTypeConflict)

	err = operation.Cancel(context.Background(), "operation.other", nexussdk.CancelOperationOptions{})
	requireHandlerError(t, err, nexussdk.HandlerErrorTypeBadRequest)
	require.EqualValues(t, 0, operation.cancelCount)
	require.NoError(t, operation.Cancel(
		context.Background(), "operation.expected", nexussdk.CancelOperationOptions{},
	))
	require.NoError(t, operation.Cancel(
		context.Background(), "operation.expected", nexussdk.CancelOperationOptions{},
	))
	require.EqualValues(t, 1, operation.startCount)
	require.EqualValues(t, 1, operation.cancelCount)
}

func TestCleanupReleasesEveryPartialPreparationExactlyOnce(t *testing.T) {
	request := checkedCallerClosureRequest(t, "partial-cleanup")
	command, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	resource, err := umpireruntime.NewResource(
		umpireruntime.ResourceParticipant,
		"runtime.correlation.participant.partial-cleanup",
	)
	require.NoError(t, err)

	for _, test := range []struct {
		name              string
		hasEnvironment    bool
		hasEndpoint       bool
		hasCaller         bool
		forceCloseAttempt bool
		forceCloseAck     bool
		wantTerminate     int
		wantDelete        int
		wantReleased      int
	}{
		{name: "nothing acquired"},
		{name: "environment acquired", hasEnvironment: true},
		{name: "endpoint acquired", hasEnvironment: true, hasEndpoint: true, wantDelete: 1, wantReleased: 1},
		{name: "caller started", hasEnvironment: true, hasEndpoint: true, hasCaller: true, wantTerminate: 1, wantDelete: 1, wantReleased: 1},
		{name: "force close attempted", hasEnvironment: true, hasEndpoint: true, hasCaller: true, forceCloseAttempt: true, wantTerminate: 1, wantDelete: 1, wantReleased: 1},
		{name: "force close acknowledged", hasEnvironment: true, hasEndpoint: true, hasCaller: true, forceCloseAttempt: true, forceCloseAck: true, wantDelete: 1, wantReleased: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			sdkClient := &recordingCleanupClient{}
			environment := &recordingCleanupEnvironment{sdkClient: sdkClient}
			adapter := &sdkCommandAdapter{participantResource: resource}
			if test.hasEnvironment {
				adapter.environment = environment
			}
			adapter.endpointAcquired = test.hasEndpoint
			if test.hasCaller {
				adapter.run = fixedWorkflowRun{workflowID: "workflow", runID: "run"}
			}
			adapter.forceCloseAttempted = test.forceCloseAttempt
			adapter.forceCloseAcknowledged = test.forceCloseAck

			first := adapter.Cleanup(context.Background(), inertEnvironment{}, command)
			require.Equal(t, umpireruntime.ReceiptAccepted, first.Status())
			require.Len(t, first.ReleasedResources(), test.wantReleased)
			second := adapter.Cleanup(context.Background(), inertEnvironment{}, command)
			require.Equal(t, umpireruntime.ReceiptAccepted, second.Status())
			require.Empty(t, second.ReleasedResources())
			require.Equal(t, test.wantTerminate, sdkClient.terminations)
			require.Equal(t, test.wantDelete, environment.deletions)
		})
	}
}

func TestCleanupFailureRetainsReleasedResourcesWithoutReacquiringThem(t *testing.T) {
	request := checkedCallerClosureRequest(t, "partial-cleanup-failure")
	command, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	resource, err := umpireruntime.NewResource(
		umpireruntime.ResourceParticipant,
		"runtime.correlation.participant.partial-cleanup-failure",
	)
	require.NoError(t, err)
	sdkClient := &recordingCleanupClient{terminateErr: errors.New("terminate failed")}
	environment := &recordingCleanupEnvironment{sdkClient: sdkClient}
	adapter := &sdkCommandAdapter{
		participantResource: resource,
		environment:         environment,
		endpointAcquired:    true,
		run:                 fixedWorkflowRun{workflowID: "workflow", runID: "run"},
	}

	first := adapter.Cleanup(context.Background(), inertEnvironment{}, command)
	require.Equal(t, umpireruntime.ReceiptFailed, first.Status())
	require.Empty(t, first.AcquiredResources())
	require.Equal(t, []umpireruntime.Resource{resource}, first.ReleasedResources())
	second := adapter.Cleanup(context.Background(), inertEnvironment{}, command)
	require.Equal(t, umpireruntime.ReceiptFailed, second.Status())
	require.Empty(t, second.AcquiredResources())
	require.Empty(t, second.ReleasedResources())
	require.Equal(t, 2, sdkClient.terminations)
	require.Equal(t, 1, environment.deletions)
}

func receiptField(t *testing.T, receipt umpireruntime.Receipt, definitionID string) string {
	t.Helper()
	values := []string{}
	for _, fact := range receipt.Facts() {
		for _, field := range fact.Fields() {
			if field.DefinitionID() == definitionID {
				values = append(values, field.Value())
			}
		}
	}
	require.Len(t, values, 1)
	return values[0]
}

func receiptFactsByKind(receipt umpireruntime.Receipt, kind string) []umpireruntime.Fact {
	facts := []umpireruntime.Fact{}
	for _, fact := range receipt.Facts() {
		if fact.KindDefinitionID() == kind {
			facts = append(facts, fact)
		}
	}
	return facts
}

func rawFactsByKind(facts []artifactv2.RawEvidenceFact, kind string) []artifactv2.RawEvidenceFact {
	matched := []artifactv2.RawEvidenceFact{}
	for _, fact := range facts {
		if fact.KindDefinitionID == kind {
			matched = append(matched, fact)
		}
	}
	return matched
}

func factField(t *testing.T, fact umpireruntime.Fact, definitionID string) string {
	t.Helper()
	for _, field := range fact.Fields() {
		if field.DefinitionID() == definitionID {
			return field.Value()
		}
	}
	require.FailNow(t, "missing fact field", definitionID)
	return ""
}

func requireHandlerError(t *testing.T, err error, kind nexussdk.HandlerErrorType) {
	t.Helper()
	var handlerError *nexussdk.HandlerError
	require.ErrorAs(t, err, &handlerError)
	require.Equal(t, kind, handlerError.Type)
	require.Equal(t, nexussdk.HandlerErrorRetryBehaviorNonRetryable, handlerError.RetryBehavior)
}

type panickingCallerClosureOperation struct {
	nexussdk.UnimplementedOperation[string, string]
	startCount  uint64
	cancelCount uint64
}

func (*panickingCallerClosureOperation) Name() string { return nexusOperationName }

func (o *panickingCallerClosureOperation) Start(
	context.Context,
	string,
	nexussdk.StartOperationOptions,
) (nexussdk.HandlerStartOperationResult[string], error) {
	o.startCount++
	panic("start")
}

func (o *panickingCallerClosureOperation) Cancel(
	context.Context,
	string,
	nexussdk.CancelOperationOptions,
) error {
	o.cancelCount++
	panic("cancel")
}

type recordingCleanupClient struct {
	client.Client
	terminations int
	terminateErr error
}

type recordingRealizationClient struct {
	client.Client
	cancellations int
	historyReads  int
	onCancel      func() error
}

type unreadyClient struct {
	client.Client
	descriptions int
}

func (c *unreadyClient) DescribeTaskQueue(
	context.Context,
	string,
	enumspb.TaskQueueType,
) (*workflowservice.DescribeTaskQueueResponse, error) {
	c.descriptions++
	return nil, errors.New("not ready")
}

func (c *recordingRealizationClient) CancelWorkflow(
	context.Context,
	string,
	string,
) error {
	c.cancellations++
	if c.onCancel == nil {
		return nil
	}
	return c.onCancel()
}

func (c *recordingRealizationClient) GetWorkflowHistory(
	context.Context,
	string,
	string,
	bool,
	enumspb.HistoryEventFilterType,
) client.HistoryEventIterator {
	c.historyReads++
	return nil
}

type recordingRealizationEnvironment struct {
	local.Environment
	sdkClient     client.Client
	controlCounts []uint64
}

func (e *recordingRealizationEnvironment) Client() client.Client { return e.sdkClient }

func (e *recordingRealizationEnvironment) Identities() local.Identities {
	return local.Identities{}
}

func (e *recordingRealizationEnvironment) RecordControlCount(
	_ umpireruntime.Command,
	_ string,
	count uint64,
) error {
	e.controlCounts = append(e.controlCounts, count)
	return nil
}

func (c *recordingCleanupClient) TerminateWorkflow(
	context.Context,
	string,
	string,
	string,
	...interface{},
) error {
	c.terminations++
	return c.terminateErr
}

type recordingCleanupEnvironment struct {
	local.Environment
	sdkClient *recordingCleanupClient
	deletions int
}

func (e *recordingCleanupEnvironment) Client() client.Client { return e.sdkClient }

func (e *recordingCleanupEnvironment) Identities() local.Identities { return local.Identities{} }

func (e *recordingCleanupEnvironment) DeleteWorkerEndpoint(
	context.Context,
	umpireruntime.Command,
	local.WorkerEndpoint,
) error {
	e.deletions++
	return nil
}

type fixedWorkflowRun struct {
	client.WorkflowRun
	workflowID string
	runID      string
}

func (r fixedWorkflowRun) GetID() string    { return r.workflowID }
func (r fixedWorkflowRun) GetRunID() string { return r.runID }

func checkedCallerClosureRequest(t *testing.T, suffix string) umpireruntime.CheckedRunRequest {
	t.Helper()
	request, err := CheckRequest(admitCallerClosureSet(t), "umpire.local.caller-closure."+suffix)
	require.NoError(t, err)
	return request
}

func checkedDuplicateDeliveryRequest(t *testing.T, suffix string) umpireruntime.CheckedRunRequest {
	t.Helper()
	request, err := CheckRequest(
		admitCallerClosureDuplicateDeliverySet(t),
		"umpire.local.caller-closure."+suffix,
	)
	require.NoError(t, err)
	return request
}

type recordingCommandAdapter struct {
	calls []umpireruntime.CommandKind
}

func (a *recordingCommandAdapter) Prepare(_ context.Context, _ umpireruntime.Environment, command umpireruntime.Command) umpireruntime.Receipt {
	return a.accept(command)
}

func (a *recordingCommandAdapter) Realize(_ context.Context, _ umpireruntime.Environment, command umpireruntime.Command) umpireruntime.Receipt {
	return a.accept(command)
}

func (a *recordingCommandAdapter) Observe(_ context.Context, _ umpireruntime.Environment, command umpireruntime.Command) umpireruntime.Receipt {
	return a.accept(command)
}

func (a *recordingCommandAdapter) Cleanup(_ context.Context, _ umpireruntime.Environment, command umpireruntime.Command) umpireruntime.Receipt {
	return a.accept(command)
}

func (a *recordingCommandAdapter) accept(command umpireruntime.Command) umpireruntime.Receipt {
	a.calls = append(a.calls, command.Kind())
	receipt, err := umpireruntime.NewReceipt(
		command,
		umpireruntime.ReceiptAccepted,
		[]umpireruntime.Fact{},
		[]umpireruntime.Resource{},
		[]umpireruntime.Resource{},
	)
	if err != nil {
		panic(err)
	}
	return receipt
}

type inertEnvironment struct{}

func (inertEnvironment) Isolate(context.Context, umpireruntime.Command) umpireruntime.Receipt {
	return umpireruntime.Receipt{}
}

func (inertEnvironment) Cleanup(context.Context, umpireruntime.Command) umpireruntime.Receipt {
	return umpireruntime.Receipt{}
}
