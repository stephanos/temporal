// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package catch

import (
	"context"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/tools/catch/pitcher"
)

// SimpleWorkflow is a basic workflow for testing
func SimpleWorkflow(ctx workflow.Context) error {
	// Execute a simple activity
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
	})
	err := workflow.ExecuteActivity(ctx, SimpleActivity).Get(ctx, nil)
	return err
}

// SimpleActivity is a basic activity for testing
func SimpleActivity(ctx context.Context) error {
	return nil
}

func TestWorkflow(t *testing.T) {
	ts := NewTestSuite(t)
	ts.Setup()

	ts.Run(t, "SimpleWorkflow", func(ctx context.Context, t *testing.T, s *TestSuite) {
		// Register workflow and start worker
		taskQueue := s.TaskQueue()
		w := worker.New(s.SdkClient(), taskQueue, worker.Options{})
		w.RegisterWorkflow(SimpleWorkflow)
		w.RegisterActivity(SimpleActivity)
		require.NoError(t, w.Start())
		defer w.Stop()

		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Build the play scenario
		play := pitcher.NewPlayBuilder().
			WithStartWorkflow(
				s.SdkClient(),
				client.StartWorkflowOptions{
					TaskQueue: taskQueue,
				},
				SimpleWorkflow,
				nil,
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
		w.RegisterWorkflow(SimpleWorkflow)
		w.RegisterActivity(SimpleActivity)
		require.NoError(t, w.Start())
		defer w.Stop()

		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		// Generate unique workflow ID for matching
		workflowID := "test-workflow-" + uuid.New()

		// Build the play scenario with fault injection
		play := pitcher.NewPlayBuilder().
			WithFault(
				"*matchingservice.AddWorkflowTaskRequest",
				pitcher.FailPlay(pitcher.ErrorResourceExhausted),
				&pitcher.MatchCriteria{
					WorkflowID: workflowID,
				},
			).
			WithStartWorkflow(
				s.SdkClient(),
				client.StartWorkflowOptions{
					ID:        workflowID,
					TaskQueue: taskQueue,
				},
				SimpleWorkflow,
				nil,
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
		moveHistory := s.Umpire().MoveHistory()
		addWorkflowTaskMoves := moveHistory.QueryByWorkflowIDAndType(workflowID, "AddWorkflowTask")
		require.GreaterOrEqual(t, len(addWorkflowTaskMoves), 2,
			"Expected at least 2 AddWorkflowTask calls (1 failed, 1+ successful)")
		t.Logf("Total AddWorkflowTask moves captured: %d", len(addWorkflowTaskMoves))
	})
}
