package temporal

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	sdkmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/tools/umpire3/execution/participant"
)

func TestSDKParticipantAsynchronousResponseAcknowledgesSignalDispatch(t *testing.T) {
	t.Parallel()

	operation := participant.Operation{
		CommandID: "cancel", SemanticAction: "request-cancellation",
		SDKOperation: participant.SDKCancel, Response: participant.ResponseAsynchronous,
	}
	client := &sdkmocks.Client{}
	run := &sdkmocks.WorkflowRun{}
	client.On("SignalWorkflow", mock.Anything, "workflow", "run", SDKCommandSignalName, operation).
		Return(nil).Once()
	run.On("GetID").Return("workflow")
	run.On("GetRunID").Return("run")
	adapter := &SDKParticipantAdapter{
		options: SDKParticipantOptions{Client: client, WorkflowType: "workflow-type"},
		plan: participant.Plan{
			FormatVersion: participant.FormatVersion, ProgramID: "program",
			Operations: []participant.Operation{operation},
		},
		run: run, started: true,
	}

	result, err := adapter.Execute(t.Context(), operation)
	require.NoError(t, err)
	require.Equal(t, "accepted", result.Status)
	require.Equal(t, "workflow", result.WorkflowID)
	require.Equal(t, "run", result.RunID)
	require.False(t, adapter.workflowClosed)
	require.Equal(t, 1, adapter.programExecutions)
	client.AssertExpectations(t)
}

func TestSDKProgramAsynchronousAndDeferredResponsesDoNotWaitForCompletion(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(SDKActivity, activity.RegisterOptions{Name: "umpire3-test-activity"})
	env.OnActivity("umpire3-test-activity", mock.Anything, mock.Anything).After(time.Hour).Return(nil)
	operations := []participant.Operation{
		{CommandID: "async", SDKOperation: participant.SDKExecuteActivity, Response: participant.ResponseAsynchronous},
		{CommandID: "deferred", SDKOperation: participant.SDKExecuteActivity, Response: participant.ResponseDeferred},
	}
	for index, operation := range operations {
		operation := operation
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(SDKCommandSignalName, operation)
		}, time.Duration(index)*time.Second)
	}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SDKFinishSignalName, nil)
	}, time.Duration(len(operations))*time.Second)
	started := env.Now()

	env.ExecuteWorkflow(SDKProgramWorkflow, SDKWorkflowInput{
		FormatVersion: SDKProgramFormatVersion,
		Plan:          participant.Plan{FormatVersion: participant.FormatVersion, ProgramID: "sdk-response-modes", Operations: operations},
		ActivityType:  "umpire3-test-activity", ChildType: "umpire3-test-child",
	})
	require.NoError(t, env.GetWorkflowError())
	var results []participant.Result
	require.NoError(t, env.GetWorkflowResult(&results))
	require.Equal(t, []participant.Result{
		{CommandID: "async", Status: "accepted"},
		{CommandID: "deferred", Status: "deferred"},
	}, results)
	require.Less(t, env.Now().Sub(started), time.Minute)
}

func TestSDKProgramWorkflowExecutesRealActivityChildAndBoundedCommands(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(SDKActivity, activity.RegisterOptions{Name: "umpire3-test-activity"})
	env.RegisterWorkflowWithOptions(SDKChildWorkflow, workflow.RegisterOptions{Name: "umpire3-test-child"})
	operations := []participant.Operation{
		{CommandID: "workflow", SDKOperation: participant.SDKExecuteWorkflow, Response: participant.ResponseSynchronous},
		{CommandID: "activity", SDKOperation: participant.SDKExecuteActivity, Response: participant.ResponseSynchronous},
		{CommandID: "retry", SDKOperation: participant.SDKRetry, Response: participant.ResponseSynchronous},
		{CommandID: "child", SDKOperation: participant.SDKExecuteChild, Response: participant.ResponseAsynchronous},
		{CommandID: "cancel", SDKOperation: participant.SDKCancel, Response: participant.ResponseSynchronous},
		{CommandID: "timer", SDKOperation: participant.SDKStartTimer, Response: participant.ResponseBlocking, MaxBlockNanos: int64(time.Second)},
		{CommandID: "failure", SDKOperation: participant.SDKReturnFailure, Response: participant.ResponseFailure},
	}
	for index, operation := range operations {
		operation := operation
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(SDKCommandSignalName, operation)
		}, time.Duration(index)*time.Second)
	}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SDKFinishSignalName, nil)
	}, time.Duration(len(operations))*time.Second)

	env.ExecuteWorkflow(SDKProgramWorkflow, SDKWorkflowInput{
		FormatVersion: SDKProgramFormatVersion,
		Plan: participant.Plan{
			FormatVersion: participant.FormatVersion, ProgramID: "sdk-program", Operations: operations,
		},
		ActivityType: "umpire3-test-activity", ChildType: "umpire3-test-child",
	})
	require.NoError(t, env.GetWorkflowError())
	var results []participant.Result
	require.NoError(t, env.GetWorkflowResult(&results))
	require.Equal(t, []participant.Result{
		{CommandID: "workflow", Status: "completed"},
		{CommandID: "activity", Status: "completed"},
		{CommandID: "retry", Status: "completed"},
		{CommandID: "child", Status: "accepted"},
		{CommandID: "cancel", Status: "completed"},
		{CommandID: "timer", Status: "completed"},
		{CommandID: "failure", Status: "failed"},
	}, results)
}

func TestSDKProgramClosesWhenEveryDeclaredCommandCompletes(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	operation := participant.Operation{CommandID: "timer", SDKOperation: participant.SDKStartTimer, Response: participant.ResponseSynchronous}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SDKCommandSignalName, operation)
	}, time.Second)

	env.ExecuteWorkflow(SDKProgramWorkflow, SDKWorkflowInput{
		FormatVersion: SDKProgramFormatVersion,
		Plan:          participant.Plan{FormatVersion: participant.FormatVersion, ProgramID: "sdk-auto-close", Operations: []participant.Operation{operation}},
		ActivityType:  "unused-activity", ChildType: "unused-child",
	})
	require.NoError(t, env.GetWorkflowError())
	var results []participant.Result
	require.NoError(t, env.GetWorkflowResult(&results))
	require.Equal(t, []participant.Result{{CommandID: "timer", Status: "completed"}}, results)
}

func TestNewSDKRunnerRequiresBoundedCompleteConfiguration(t *testing.T) {
	_, err := NewSDKParticipantAdapter(SDKParticipantOptions{})
	require.EqualError(t, err, "SDK participant requires client, registry, task queue, and workflow ID")
}

func TestSDKProgramSuppressesStaleSuccessAfterCommittedCancellation(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(SDKChildWorkflow, workflow.RegisterOptions{Name: "umpire3-test-child"})
	operations := []participant.Operation{
		{
			CommandID: "commit", SemanticAction: "commit-cancellation",
			SDKOperation: participant.SDKCancel, Response: participant.ResponseSynchronous,
		},
		{
			CommandID: "worker", SemanticAction: "worker-returns-success",
			SDKOperation: participant.SDKHandleNexus, Response: participant.ResponseSynchronous,
		},
		{
			CommandID: "persist", SemanticAction: "persist-success",
			SDKOperation: participant.SDKHandleNexus, Response: participant.ResponseSynchronous,
		},
	}
	for index, operation := range operations {
		operation := operation
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(SDKCommandSignalName, operation)
		}, time.Duration(index)*time.Second)
	}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SDKFinishSignalName, nil)
	}, time.Duration(len(operations))*time.Second)

	env.ExecuteWorkflow(SDKProgramWorkflow, SDKWorkflowInput{
		FormatVersion: SDKProgramFormatVersion,
		Plan: participant.Plan{
			FormatVersion: participant.FormatVersion, ProgramID: "sdk-cancellation", Operations: operations,
		},
		ActivityType: "unused-activity", ChildType: "umpire3-test-child",
	})
	require.NoError(t, env.GetWorkflowError())
	var results []participant.Result
	require.NoError(t, env.GetWorkflowResult(&results))
	require.Equal(t, []participant.Result{
		{CommandID: "commit", Status: "completed"},
		{CommandID: "worker", Status: "suppressed"},
		{CommandID: "persist", Status: "suppressed"},
	}, results)
}

func TestSDKNameIsStableAndBounded(t *testing.T) {
	require.Equal(t, "hello-world", safeSDKName("hello/world"))
	require.LessOrEqual(t, len(safeSDKName(strings.Repeat("x", 256))), 120)
}

func TestSDKCapabilityPreflightRejectsProgramDriftBeforeAllocation(t *testing.T) {
	t.Parallel()

	plan := participant.Plan{Capabilities: []string{"activity", "callback", "nexus", "workflow"}}
	require.Equal(t, []string{"callback", "nexus"}, missingSDKCapabilities(plan, SDKParticipantOptions{}))
	require.Empty(t, missingSDKCapabilities(plan, SDKParticipantOptions{
		NexusEndpoint: "endpoint", NexusService: "service", NexusOperation: "operation",
		CallbackDriver: fakeCallbackDriver{},
	}))
}

type fakeCallbackDriver struct{}

func (fakeCallbackDriver) RegisterCompletionCallback(
	context.Context,
	string,
	string,
) (MechanismReceipt, error) {
	return MechanismReceipt{}, nil
}

func (fakeCallbackDriver) CompleteCompletionCallback(
	context.Context,
	string,
	string,
) (MechanismReceipt, error) {
	return MechanismReceipt{}, nil
}

func (fakeCallbackDriver) CleanupCompletionCallbacks(context.Context) error {
	return nil
}
