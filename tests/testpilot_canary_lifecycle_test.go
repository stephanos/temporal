//go:build test_dep && integration

package tests

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/temporal/provision"
	"go.temporal.io/server/tools/canary/assessment"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/controller"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
	"go.temporal.io/server/tools/umpire/evaluation"
	"go.temporal.io/server/tools/umpire/recordedrun"
	"google.golang.org/grpc/credentials/insecure"
)

// The canary's early proof: the pinned canary Case, prepared by preflight under the test cluster's
// names, runs twice, serially, in-process through the real Driver, each Run fenced on one lease.
// The test passes a plaintext transport directly, as only a test or the harness build may, and a
// decide that admits each Run through fn-26 in memory and assesses it under the canary's
// Evaluation Profile. Both Runs close satisfied with distinct Run IDs that
// the lease's signals name, cleanup verifies them closed and releases the lease, and progress
// carries no raw coordinate.
func TestTestpilotCanaryLifecycle(t *testing.T) {
	env := newTestpilotTestEnvironment(t)
	ctx, cancel := context.WithTimeout(env.Context(), 5*time.Minute)
	defer cancel()
	coordinates := authority.Coordinates{
		GRPC: env.FrontendGRPCAddress(), Namespace: "umpire-canary-lifecycle", TaskQueue: "umpire-canary-lifecycle-queue",
		HandlerQueue: "umpire-canary-lifecycle-handler-queue", NexusEndpoint: "umpire-canary-lifecycle-endpoint",
	}
	release, err := provision.Create(ctx, provision.Clients{Workflow: env.FrontendClient(), Operator: env.OperatorClient()}, provision.Resources{
		Namespace: coordinates.Namespace, TaskQueue: coordinates.TaskQueue,
		NexusEndpoint: coordinates.NexusEndpoint, NexusTaskQueue: coordinates.HandlerQueue, RetainNamespace: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), testpilotCleanupTimeout)
		defer cancel()
		require.NoError(t, release(cleanupCtx))
	})

	canary, err := policy.Embedded()
	require.NoError(t, err)
	canary.Coordinates = coordinates.Digests()
	dispatch := map[string]string{
		preflight.VariableEventName: "workflow_dispatch", preflight.VariableRepository: canary.Repository,
		preflight.VariableRef:         canary.TrustedRef,
		preflight.VariableWorkflowRef: canary.Repository + "/" + canary.WorkflowPath + "@" + canary.TrustedRef,
		preflight.VariableRunID:       "1", preflight.VariableRunAttempt: "1",
	}
	redactor := authority.NewRedactor(coordinates.GRPC, coordinates.Namespace, coordinates.TaskQueue, coordinates.HandlerQueue, coordinates.NexusEndpoint)
	scope, err := preflight.Check(ctx, preflight.Input{
		Policy: canary, Coordinates: coordinates, Redactor: redactor, Namespaces: env.FrontendClient(),
		Lookup: func(key string) (string, bool) { value, ok := dispatch[key]; return value, ok },
	})
	require.NoError(t, err)
	store, err := recovery.Create(filepath.Join(t.TempDir(), "recovery.json"), scope.InvocationID)
	require.NoError(t, err)

	// The controller starts an iteration only while one iteration's whole worst case fits in what
	// is left, which is longer than a test context's ceiling; it gets the invocation limit instead.
	runCtx, cancelRun := context.WithTimeout(context.Background(), canary.Limits.Invocation())
	defer cancelRun()
	profile, err := assessment.LoadProfile(canary.EvaluationProfile)
	require.NoError(t, err)
	recordedOnce := false
	var progress bytes.Buffer
	result, err := controller.Run(runCtx, controller.Config{
		Policy: canary, Scope: scope, Namespace: coordinates.Namespace,
		Transport: authority.Transport{Target: coordinates.GRPC, Credentials: insecure.NewCredentials()},
		Redactor:  redactor, Service: env.FrontendClient(), Identity: "umpire-canary-lifecycle",
		Dial: client.Dial,
		Decide: func(run *testpilotpb.Run, _ *testpilotpb.Verdict) controller.Outcome {
			// The first Run's record is the admission tests' fixture when UMPIRE_CANARY_RECORD names
			// the file to write; a test cluster's Run is the only one ever written.
			if path := os.Getenv("UMPIRE_CANARY_RECORD"); path != "" && !recordedOnce {
				recordedOnce = true
				encoded, err := recordedrun.Encode(canary.CaseIdentity, scope.Prepared.Identity(), run)
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(path, encoded, 0o644))
			}
			subject, err := assessment.Admit(canary, scope.Prepared.Identity(), run)
			if err != nil {
				return controller.Outcome{Status: controller.StatusUnconstructible, Err: err}
			}
			decision := evaluation.Assess(subject, *profile)
			return controller.Outcome{Status: decision.Outcome}
		},
		Recovery: store, Progress: &progress, Started: time.Now(),
	})
	require.NoError(t, err)
	require.False(t, result.Unreconciled)
	require.Len(t, result.Iterations, canary.Limits.Iterations, "progress: %s", progress.String())
	var runIDs []string
	for _, iteration := range result.Iterations {
		require.Equal(t, controller.StatusAccepted, iteration.Outcome.Status, "iteration %s: %v; progress: %s", iteration.RunID, iteration.Outcome.Err, progress.String())
		require.NotContains(t, runIDs, iteration.RunID, "each Run is its own")
		runIDs = append(runIDs, iteration.RunID)
	}
	require.Equal(t, runIDs, result.Cleanup.Fenced, "the lease's signals name exactly the Runs made")
	require.Equal(t, runIDs, result.Cleanup.Closed)
	require.Empty(t, result.Cleanup.Unverified)
	require.True(t, result.Cleanup.Released, "cleanup: %v", result.Cleanup.Err)

	lease, err := env.FrontendClient().GetWorkflowExecutionHistory(ctx, &workflowservice.GetWorkflowExecutionHistoryRequest{
		Namespace:              coordinates.Namespace,
		Execution:              &commonpb.WorkflowExecution{WorkflowId: canary.Lease.WorkflowID, RunId: result.Lease.RunID},
		HistoryEventFilterType: enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
	})
	require.NoError(t, err)
	events := lease.GetHistory().GetEvents()
	require.NotEmpty(t, events)
	require.Equal(t, controller.ReasonReleased, events[len(events)-1].GetWorkflowExecutionTerminatedEventAttributes().GetReason())
	for _, runID := range runIDs {
		described, err := env.FrontendClient().DescribeWorkflowExecution(ctx, &workflowservice.DescribeWorkflowExecutionRequest{
			Namespace: coordinates.Namespace, Execution: &commonpb.WorkflowExecution{WorkflowId: runID},
		})
		require.NoError(t, err)
		require.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, described.GetWorkflowExecutionInfo().GetStatus(),
			"the canary Case's workflow ID is its Run ID, and it completed on its own")
	}

	record := store.Snapshot()
	require.Equal(t, recovery.PhaseReleased, record.Phase)
	require.Equal(t, recovery.HeldTook, record.Lease.Held)
	require.Len(t, record.Iterations, len(runIDs))
	for _, value := range []string{coordinates.GRPC, coordinates.Namespace, coordinates.TaskQueue, coordinates.HandlerQueue, coordinates.NexusEndpoint} {
		require.NotContains(t, progress.String(), value, "progress carries no raw coordinate")
	}
}
