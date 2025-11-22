package catch

import (
	"context"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	omeskitchensink "github.com/temporalio/omes/loadgen/kitchensink"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/api/matchingservice/v1"
	"go.temporal.io/server/tools/umpire/pitcher"
	rostertypes "go.temporal.io/server/tools/umpire/roster/types"
)

func TestWorkflow(t *testing.T) {
	ts := NewTestSuite(t)
	ts.Setup()

	ts.Run(t, "KitchenSinkWorkflow", func(ctx context.Context, t *testing.T, s *TestSuite) {
		// Register workflow and start worker
		taskQueue := s.TaskQueue()
		w := worker.New(s.SdkClient(), taskQueue, worker.Options{})
		RegisterWorkflows(w)
		require.NoError(t, w.Start())
		defer w.Stop()

		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Build the play scenario with KitchenSinkWorkflow
		workflowInput := &omeskitchensink.WorkflowInput{
			InitialActions: []*omeskitchensink.ActionSet{
				{
					Actions: []*omeskitchensink.Action{
						{
							Variant: &omeskitchensink.Action_ReturnResult{
								ReturnResult: &omeskitchensink.ReturnResultAction{},
							},
						},
					},
				},
			},
		}
		play := pitcher.NewPlayBuilder().
			WithStartWorkflow(
				s.SdkClient(),
				client.StartWorkflowOptions{
					TaskQueue: taskQueue,
				},
				KitchenSinkWorkflow,
				workflowInput,
			).
			Build()

		// Execute the play
		workflowRun, err := s.Pitcher().Execute(ctx, play)
		require.NoError(t, err, "Should execute play")

		// Wait for completion
		err = workflowRun.Get(ctx, nil)
		require.NoError(t, err, "Workflow should complete successfully")
	})

	ts.Run(t, "WorkflowWithFaultInjection", func(ctx context.Context, t *testing.T, s *TestSuite) {
		// Register workflow and start worker
		taskQueue := s.TaskQueue()
		w := worker.New(s.SdkClient(), taskQueue, worker.Options{})
		RegisterWorkflows(w)
		require.NoError(t, w.Start())
		defer w.Stop()

		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		// Generate unique workflow ID for matching
		workflowID := "test-workflow-" + uuid.New()

		// Build the play scenario with fault injection using KitchenSinkWorkflow
		workflowInput := &omeskitchensink.WorkflowInput{
			InitialActions: []*omeskitchensink.ActionSet{
				{
					Actions: []*omeskitchensink.Action{
						{
							Variant: &omeskitchensink.Action_ReturnResult{
								ReturnResult: &omeskitchensink.ReturnResultAction{},
							},
						},
					},
				},
			},
		}
		play := pitcher.NewPlayBuilder().
			WithFault(
				&matchingservice.AddWorkflowTaskRequest{},
				pitcher.FailPlay(serviceerror.NewResourceExhausted(enumspb.RESOURCE_EXHAUSTED_CAUSE_SYSTEM_OVERLOADED, "pitcher fault injection")),
				&pitcher.MatchCriteria{
					Entities: []rostertypes.EntityID{
						rostertypes.NewEntityIDFromType(rostertypes.WorkflowType, workflowID),
					},
				},
			).
			WithStartWorkflow(
				s.SdkClient(),
				client.StartWorkflowOptions{
					ID:        workflowID,
					TaskQueue: taskQueue,
				},
				KitchenSinkWorkflow,
				workflowInput,
			).
			Build()

		// Execute the play - the pitcher will set up faults and start the workflow
		workflowRun, err := s.Pitcher().Execute(ctx, play)
		require.NoError(t, err, "Should execute play despite fault injection")

		// Wait for completion - workflow should complete despite the initial fault
		// The system will retry the AddWorkflowTask call automatically
		err = workflowRun.Get(ctx, nil)
		require.NoError(t, err, "Workflow should complete successfully after automatic retry")

		// Verify moves were recorded via scorebook
		// Moves are now immediately available since they're recorded via gRPC interceptor
		scorebook := s.Umpire().Scorebook()
		addWorkflowTaskMoves := scorebook.QueryByWorkflowIDAndType(workflowID, "AddWorkflowTask")
		require.GreaterOrEqual(t, len(addWorkflowTaskMoves), 2,
			"Expected at least 2 AddWorkflowTask calls (1 failed, 1+ successful)")
		t.Logf("Total AddWorkflowTask moves captured: %d", len(addWorkflowTaskMoves))
	})
}
