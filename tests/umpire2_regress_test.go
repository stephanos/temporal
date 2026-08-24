package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/payloads"
	umpirefw "go.temporal.io/server/common/testing/umpire"
	coreregress "go.temporal.io/server/common/testing/umpire/regress"
	regressactivity "go.temporal.io/server/tools/umpire2/regress/activity"
	"go.temporal.io/server/tools/umpire2/regress/capability"
	"go.temporal.io/server/tools/umpire2/regress/nexus"
	"go.temporal.io/server/tools/umpire2/regress/workflow"
	"go.temporal.io/server/tools/umpire2/umpiretest"
)

func TestUmpire2SparseRegressionOrdinaryNexusCompletion(t *testing.T) {
	resultDigest, err := umpirefw.CanonicalProtoDigest("result", payloads.MustEncodeSingle("ok"))
	require.NoError(t, err)
	plan := coreregress.OnePath(
		nexus.ScheduleEmbedded("op", "caller"),
		nexus.RespondStart("op", nexus.Sync),
		nexus.State("op", nexus.Completed),
		nexus.ResultDigest("op", resultDigest),
		nexus.LinkEndpoint("op", "workflow-event:umpire-regression/handler/handler-run/handler-start"),
		workflow.NexusStorageAbsent("caller", "op"),
	)
	runUmpire2SparseRegression(t, plan)
}

func TestUmpire2SparseRegressionCompletionBeforeStartResponse(t *testing.T) {
	plan := coreregress.AllPaths(
		nexus.Complete("op", nexus.Succeeded),
		nexus.RespondStart("op", nexus.Async),
		nexus.State("op", nexus.Completed),
		nexus.LateStartResponseAccepted("op"),
	)
	runUmpire2SparseRegression(t, plan)
}

func TestUmpire2SparseRegressionCancellationRetry(t *testing.T) {
	plan := coreregress.OnePath(
		nexus.State("op", nexus.Started),
		coreregress.During(
			nexus.FailNext(nexus.CancelNexusOperation),
			nexus.CancelWithRetry("op"),
		),
		nexus.CancelRequestFailed("op"),
		nexus.State("op", nexus.Canceled),
	)
	runUmpire2SparseRegression(t, plan)
}

func runUmpire2SparseRegressionStartToCloseTimeout(t *testing.T, chasmEnabled bool) {
	t.Helper()
	plan := coreregress.OnePath(
		nexus.Schedule("op", nexus.StartToClose(2*time.Second)),
		nexus.RespondStart("op", nexus.Async),
		nexus.State("op", nexus.TimedOut),
	)
	runUmpire2SparseRegressionWithCHASM(t, plan, chasmEnabled)
}

func TestUmpire2SparseRegressionSharedHandlerWorkflow(t *testing.T) {
	plan := coreregress.AllPaths(
		coreregress.AnyOrder(
			nexus.Start("left", nexus.HandlerWorkflow("handler")),
			nexus.Start("right", nexus.HandlerWorkflow("handler")),
		),
		workflow.State("handler", workflow.Completed),
		nexus.State("left", nexus.Completed),
		nexus.State("right", nexus.Completed),
		nexus.CallbackReferenceConsistent("left", "handler"),
		nexus.CallbackReferenceConsistent("right", "handler"),
	)
	runUmpire2SparseRegressionWithCHASM(t, plan, false)
}

func runUmpire2SparseRegressionCallbackAfterCallerCompletion(t *testing.T, chasmEnabled bool) {
	t.Helper()
	plan := coreregress.OnePath(
		nexus.State("op", nexus.Started),
		workflow.State("caller", workflow.Completed),
		nexus.Complete("op", nexus.Succeeded),
		nexus.State("op", nexus.CallbackFailed),
	)
	runUmpire2SparseRegressionWithCHASM(t, plan, chasmEnabled)
}

func runUmpire2SparseRegressionBidirectionalNexusActivityLinks(t *testing.T, chasmEnabled bool) {
	t.Helper()
	plan := coreregress.OnePath(
		coreregress.Require(capability.ActivityCallbacks),
		nexus.StartActivity("op", "activity"),
		regressactivity.State("activity", regressactivity.Completed),
		nexus.LinkedToActivity("op", "activity"),
		regressactivity.LinkedToNexusOperation("activity", "op"),
	)
	runUmpire2SparseRegressionWithCHASM(t, plan, chasmEnabled)
}

func runUmpire2SparseRegression(t *testing.T, plan coreregress.Plan) {
	runUmpire2SparseRegressionWithCHASM(t, plan, true)
}

func runUmpire2SparseRegressionWithCHASM(t *testing.T, plan coreregress.Plan, chasmEnabled bool) {
	t.Helper()
	umpiretest.RequireRegression(t, plan, umpiretest.WithCHASM(chasmEnabled))
}
