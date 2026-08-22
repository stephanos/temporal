package tests

import (
	"testing"

	"go.temporal.io/server/tests/umpire3/scenario"
	umpire3nexus "go.temporal.io/server/tests/umpire3/scenario/nexus"
)

func TestUmpire3SparseRegressionOrdinaryNexusCompletion(t *testing.T) {
	runUmpire3Regression(t, umpire3NexusClosureRegression("SparseRegressionOrdinaryNexusCompletion",
		scenario.ScheduleOperation("schedule",
			scenario.WithNexusCompletion(scenario.NexusCompletionOrdinary)),
		scenario.WorkerReturnsSuccess("success"),
		scenario.PersistSuccess("persist"),
	))
}

func TestUmpire3SparseRegressionCompletionBeforeStartResponse(t *testing.T) {
	runUmpire3Regression(t, umpire3NexusClosureRegression("SparseRegressionCompletionBeforeStartResponse",
		scenario.WorkerReturnsSuccess("success",
			scenario.WithNexusCompletion(scenario.NexusCompletionBeforeStart)),
		scenario.PersistSuccess("persist"),
	))
}

func TestUmpire3SparseRegressionCancellationRetry(t *testing.T) {
	operation := umpire3nexus.Operation("operation")
	runUmpire3Regression(t, umpire3nexus.Scenario("SparseRegressionCancellationRetry", operation,
		scenario.OnePath(
			operation.Schedule(),
			operation.Dispatch(),
			scenario.During(umpire3NexusRPCFault("drop-cancellation", scenario.Drop,
				scenario.OnRoutes("/service/operation/cancel")),
				operation.RequestCancellation(scenario.Asynchronously())),
			operation.CommitCancellation(),
			operation.AcquireOwnership(),
			operation.Retry(),
			operation.WorkerReturnsSuccess(),
			operation.PersistSuccess(),
			operation.CancellationSafety(),
		)))
}

func TestUmpire3SparseRegressionSharedHandlerWorkflow(t *testing.T) {
	runUmpire3Regression(t, scenario.IntegrationCallbackWorkflowScenario("SparseRegressionSharedHandlerWorkflow",
		[]scenario.Resource{
			scenario.NexusOperation("operation"),
			scenario.Callback("callback"),
			scenario.Workflow("handler"),
		},
		scenario.AllPaths(
			scenario.RegisterCallback("register-shared-handler"),
			scenario.RecordCallbackResponse("respond"),
			scenario.RequireCallbackResponseConsistency(),
		)), "hsm")
}

func TestUmpire3SparseRegressionStartToCloseTimeout(t *testing.T) {
	for _, chasmEnabled := range []bool{false, true} {
		t.Run(umpire3CHASMName(chasmEnabled), func(t *testing.T) {
			runUmpire3Regression(t, scenario.IntegrationNexusTimeoutScenario("SparseRegressionStartToCloseTimeout",
				[]scenario.Resource{scenario.NexusOperation("operation")},
				scenario.OnePath(
					scenario.ScheduleOperation("schedule"),
					scenario.TimeoutNexusOperation("timeout"),
					scenario.RequireNexusOperationTimeoutSemantics(),
				)), umpire3CHASMNameLower(chasmEnabled))
		})
	}
}

func TestUmpire3SparseRegressionCallbackAfterCallerCompletion(t *testing.T) {
	for _, chasmEnabled := range []bool{false, true} {
		t.Run(umpire3CHASMName(chasmEnabled), func(t *testing.T) {
			if chasmEnabled {
				t.Skip("Blocked on CHASM Nexus callback failure handling after caller completion")
			}
			runUmpire3Regression(t, scenario.IntegrationCallbackWorkflowScenario("SparseRegressionCallbackAfterCallerCompletion",
				[]scenario.Resource{
					scenario.NexusOperation("operation"),
					scenario.Callback("callback"),
					scenario.Workflow("caller"),
				},
				scenario.OnePath(
					scenario.RegisterCallback("register"),
					scenario.RecordCallbackResponse("respond"),
					scenario.RequireCallbackResponseConsistency(),
				)), umpire3CHASMNameLower(chasmEnabled))
		})
	}
}

func TestUmpire3SparseRegressionBidirectionalNexusActivityLinks(t *testing.T) {
	for _, chasmEnabled := range []bool{false, true} {
		t.Run(umpire3CHASMName(chasmEnabled), func(t *testing.T) {
			runUmpire3Regression(t, scenario.IntegrationNexusActivityScenario("SparseRegressionBidirectionalNexusActivityLinks",
				[]scenario.Resource{
					scenario.NexusOperation("operation"),
					scenario.Activity("activity"),
				},
				scenario.OnePath(
					scenario.LinkNexusActivity("link"),
					scenario.RequireNexusActivityLinkConsistency(),
				)), umpire3CHASMNameLower(chasmEnabled))
		})
	}
}

func umpire3CHASMName(enabled bool) string {
	if enabled {
		return "CHASM"
	}
	return "HSM"
}

func umpire3CHASMNameLower(enabled bool) string {
	if enabled {
		return "chasm"
	}
	return "hsm"
}
