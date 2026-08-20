package participant

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestSDKProgramAsynchronousAndDeferredResponsesDoNotWaitForCompletion(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(SDKActivity, activity.RegisterOptions{Name: "umpire3-test-activity"})
	env.OnActivity("umpire3-test-activity", mock.Anything, mock.Anything).After(time.Hour).Return(nil)
	operations := []Operation{
		{CommandID: "async", SDKOperation: SDKExecuteActivity, Response: ResponseAsynchronous},
		{CommandID: "deferred", SDKOperation: SDKExecuteActivity, Response: ResponseDeferred},
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
		Plan:          Plan{FormatVersion: FormatVersion, ProgramID: "sdk-response-modes", Operations: operations},
		ActivityType:  "umpire3-test-activity", ChildType: "umpire3-test-child",
	})
	require.NoError(t, env.GetWorkflowError())
	var results []Result
	require.NoError(t, env.GetWorkflowResult(&results))
	require.Equal(t, []Result{
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
	operations := []Operation{
		{CommandID: "workflow", SDKOperation: SDKExecuteWorkflow, Response: ResponseSynchronous},
		{CommandID: "activity", SDKOperation: SDKExecuteActivity, Response: ResponseSynchronous},
		{CommandID: "retry", SDKOperation: SDKRetry, Response: ResponseSynchronous},
		{CommandID: "child", SDKOperation: SDKExecuteChild, Response: ResponseAsynchronous},
		{CommandID: "cancel", SDKOperation: SDKCancel, Response: ResponseSynchronous},
		{CommandID: "timer", SDKOperation: SDKStartTimer, Response: ResponseBlocking, MaxBlockNanos: int64(time.Second)},
		{CommandID: "failure", SDKOperation: SDKReturnFailure, Response: ResponseFailure},
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
		Plan: Plan{
			FormatVersion: FormatVersion, ProgramID: "sdk-program", Operations: operations,
		},
		ActivityType: "umpire3-test-activity", ChildType: "umpire3-test-child",
	})
	require.NoError(t, env.GetWorkflowError())
	var results []Result
	require.NoError(t, env.GetWorkflowResult(&results))
	require.Equal(t, []Result{
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
	operation := Operation{CommandID: "timer", SDKOperation: SDKStartTimer, Response: ResponseSynchronous}
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(SDKCommandSignalName, operation)
	}, time.Second)

	env.ExecuteWorkflow(SDKProgramWorkflow, SDKWorkflowInput{
		FormatVersion: SDKProgramFormatVersion,
		Plan:          Plan{FormatVersion: FormatVersion, ProgramID: "sdk-auto-close", Operations: []Operation{operation}},
		ActivityType:  "unused-activity", ChildType: "unused-child",
	})
	require.NoError(t, env.GetWorkflowError())
	var results []Result
	require.NoError(t, env.GetWorkflowResult(&results))
	require.Equal(t, []Result{{CommandID: "timer", Status: "completed"}}, results)
}

func TestNewSDKRunnerRequiresBoundedCompleteConfiguration(t *testing.T) {
	_, err := NewSDKRunner(SDKOptions{})
	require.EqualError(t, err, "SDK participant requires client, registry, task queue, and workflow ID")
}

func TestSDKProgramSuppressesStaleSuccessAfterCommittedCancellation(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflowWithOptions(SDKChildWorkflow, workflow.RegisterOptions{Name: "umpire3-test-child"})
	operations := []Operation{
		{
			CommandID: "commit", SemanticAction: "commit-cancellation",
			SDKOperation: SDKCancel, Response: ResponseSynchronous,
		},
		{
			CommandID: "worker", SemanticAction: "worker-returns-success",
			SDKOperation: SDKHandleNexus, Response: ResponseSynchronous,
		},
		{
			CommandID: "persist", SemanticAction: "persist-success",
			SDKOperation: SDKHandleNexus, Response: ResponseSynchronous,
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
		Plan: Plan{
			FormatVersion: FormatVersion, ProgramID: "sdk-cancellation", Operations: operations,
		},
		ActivityType: "unused-activity", ChildType: "umpire3-test-child",
	})
	require.NoError(t, env.GetWorkflowError())
	var results []Result
	require.NoError(t, env.GetWorkflowResult(&results))
	require.Equal(t, []Result{
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

	plan := Plan{Capabilities: []string{"activity", "callback", "nexus", "workflow"}}
	require.Equal(t, []string{"callback", "nexus"}, missingSDKCapabilities(plan, SDKOptions{}))
	require.Empty(t, missingSDKCapabilities(plan, SDKOptions{
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
