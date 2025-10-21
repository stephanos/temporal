package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/persistence"
	"go.temporal.io/server/tests/testcore"
)

type LostTaskTestSuite struct {
	testcore.FunctionalTestBase
}

func TestUmpireLostTaskTestSuite(t *testing.T) {
	// Enable OTEL debug mode to capture proto payloads in spans
	t.Setenv("TEMPORAL_OTEL_DEBUG", "true")
	suite.Run(t, new(LostTaskTestSuite))
}

func (s *LostTaskTestSuite) SetupSuite() {
	// Ensure OTEL debug mode is enabled
	os.Setenv("TEMPORAL_OTEL_DEBUG", "true")
	s.FunctionalTestBase.SetupSuite()
}

// TestLostTaskDetection verifies that umpire detects when a task is lost from matching.
// This test simulates a scenario where:
// 1. A workflow task is stored to persistence
// 2. The task is deleted from persistence via CompleteTasksLessThan
// 3. A worker polls the task queue and gets an empty response
// 4. Umpire detects that a task was stored but never successfully polled
func (s *LostTaskTestSuite) TestLostTaskDetection() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start a workflow without any workers
	workflowID := "umpire-lost-task-test"
	workflowType := "TestWorkflow"
	taskQueue := s.TaskQueue()

	startResp, err := s.FrontendClient().StartWorkflowExecution(ctx, &workflowservice.StartWorkflowExecutionRequest{
		Namespace:    s.Namespace().String(),
		WorkflowId:   workflowID,
		WorkflowType: &commonpb.WorkflowType{Name: workflowType},
		TaskQueue:    &taskqueuepb.TaskQueue{Name: taskQueue},
	})
	s.NoError(err)
	s.T().Logf("Workflow started: %s/%s on task queue %s", workflowID, startResp.GetRunId(), taskQueue)

	// Wait for at least one task to be created in persistence
	// Tasks may be spilled to persistence if matching service has memory pressure
	// or if there are no pollers available
	taskMgr := s.GetTestCluster().TestBase().TaskMgr
	var deleted int
	s.Eventually(func() bool {
		// Try to delete tasks from persistence
		var err error
		deleted, err = taskMgr.CompleteTasksLessThan(ctx, &persistence.CompleteTasksLessThanRequest{
			NamespaceID:        s.NamespaceID().String(),
			TaskQueueName:      taskQueue,
			TaskType:           enumspb.TASK_QUEUE_TYPE_WORKFLOW,
			ExclusiveMaxTaskID: 999999999, // Delete all tasks
			Limit:              1000,      // Max number of tasks to delete
		})
		if err != nil {
			s.T().Logf("CompleteTasksLessThan error: %v", err)
			return false
		}
		s.T().Logf("Deleted %d tasks from persistence", deleted)
		return deleted > 0
	}, 10*time.Second, 100*time.Millisecond, "Expected at least one task to be stored in persistence")

	// Now poll the task queue - this should return empty since we deleted the task
	pollCtx, pollCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pollCancel()
	pollResp, err := s.FrontendClient().PollWorkflowTaskQueue(pollCtx, &workflowservice.PollWorkflowTaskQueueRequest{
		Namespace: s.Namespace().String(),
		TaskQueue: &taskqueuepb.TaskQueue{Name: taskQueue},
		Identity:  "test-worker",
	})
	s.NoError(err)
	s.T().Logf("Poll response: taskToken length = %d", len(pollResp.TaskToken))

	// Wait for umpire to detect the lost task
	// The model checks for tasks that were stored but never polled
	s.T().Logf("Waiting for umpire to detect lost task...")

	s.Eventually(func() bool {
		violations := s.GetUmpire().Check(ctx)
		if len(violations) > 0 {
			s.T().Logf("Umpire detected %d violation(s):", len(violations))
			for _, v := range violations {
				s.T().Logf("  [%s] %s - Tags: %v", v.Model, v.Message, v.Tags)
			}
			return true
		}
		s.T().Logf("Umpire Check returned 0 violations, waiting...")
		return false
	}, 30*time.Second, 1*time.Second, "Expected umpire to detect lost task")

	// Clean up
	_, err = s.FrontendClient().TerminateWorkflowExecution(ctx, &workflowservice.TerminateWorkflowExecutionRequest{
		Namespace: s.Namespace().String(),
		WorkflowExecution: &commonpb.WorkflowExecution{
			WorkflowId: workflowID,
			RunId:      startResp.GetRunId(),
		},
		Reason: "test cleanup",
	})
	s.NoError(err)
}
