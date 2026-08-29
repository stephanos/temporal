package nexus

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	nexussdk "github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/local"
)

func TestLiveParticipantRealizesOneForceCloseAndClosesOperationalSources(t *testing.T) {
	request := checkedCallerClosureRequest(t, "live-force-close")
	runtimeParticipant, err := NewParticipant(request)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	output, err := umpireruntime.Run(ctx, request, local.NewFactory(), runtimeParticipant)
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
}

func TestParticipantAdmitsOnlyTheExactCheckedRequest(t *testing.T) {
	_, err := NewParticipant(umpireruntime.CheckedRunRequest{})
	require.Error(t, err)

	participant, err := NewParticipant(checkedCallerClosureRequest(t, "exact-request"))
	require.NoError(t, err)
	require.NotNil(t, participant)
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

func TestHandlerBindsIdentityAndRejectsDuplicateDelivery(t *testing.T) {
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
	err = operation.Cancel(context.Background(), "operation.expected", nexussdk.CancelOperationOptions{})
	requireHandlerError(t, err, nexussdk.HandlerErrorTypeConflict)
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
}

func (c *recordingCleanupClient) TerminateWorkflow(
	context.Context,
	string,
	string,
	string,
	...interface{},
) error {
	c.terminations++
	return nil
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
