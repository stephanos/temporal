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

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.temporal.io/server/common/testing/pitcher"
	"go.temporal.io/server/tests/testcore"
)

type PitcherMVPSuite struct {
	testcore.FunctionalTestBase
}

func TestPitcherMVP(t *testing.T) {
	suite.Run(t, new(PitcherMVPSuite))
}

// TestPitcherIntegration verifies that pitcher fault injection works end-to-end
func (s *PitcherMVPSuite) TestPitcherIntegration() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Configure pitcher: fail first AddWorkflowTask call
	s.ConfigurePitcher("matchingservice.AddWorkflowTask", pitcher.PitchConfig{
		Action: "fail",
		Count:  1, // Only inject once
		Params: map[string]any{
			"error": pitcher.ErrorResourceExhausted,
		},
	})

	// Register and start worker
	w := worker.New(s.SdkClient(), s.TaskQueue(), worker.Options{})
	w.RegisterWorkflow(SimpleWorkflow)
	s.NoError(w.Start())
	defer w.Stop()

	// Start a simple workflow
	workflowID := "pitcher-mvp-test"
	we, err := s.SdkClient().ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: s.TaskQueue(),
	}, SimpleWorkflow)
	s.NoError(err)

	// Despite the injected failure, the workflow should complete successfully
	// due to retry logic
	err = we.Get(ctx, nil)
	s.NoError(err, "Workflow should complete despite transient matching failure")
}

// TestPitcherReset verifies that pitcher state is reset between tests
func (s *PitcherMVPSuite) TestPitcherReset() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This test should have no pitcher configuration from the previous test
	// The workflow should complete without any injected failures

	w := worker.New(s.SdkClient(), s.TaskQueue(), worker.Options{})
	w.RegisterWorkflow(SimpleWorkflow)
	s.NoError(w.Start())
	defer w.Stop()

	workflowID := "pitcher-reset-test"
	we, err := s.SdkClient().ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: s.TaskQueue(),
	}, SimpleWorkflow)
	s.NoError(err)

	err = we.Get(ctx, nil)
	s.NoError(err, "Workflow should complete without any failures")
}

// SimpleWorkflow is a minimal workflow for testing
func SimpleWorkflow(ctx workflow.Context) error {
	return nil
}
