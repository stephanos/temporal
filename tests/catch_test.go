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
	omeskitchensink "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/server/tests/kitchensink"
	"go.temporal.io/server/tests/testcore"
	"go.temporal.io/server/tools/catch/pitcher"
)

type CatchSuite struct {
	testcore.FunctionalTestBase
}

func TestCatch(t *testing.T) {
	suite.Run(t, new(CatchSuite))
}

func (s *CatchSuite) SetupSuite() {
	s.FunctionalTestBase.SetupSuite()
}

func (s *CatchSuite) SetupTest() {
	s.FunctionalTestBase.SetupTest()

	// Reset pitcher state before each test
	pitcher.Set(pitcher.New())
}

func (s *CatchSuite) TearDownTest() {
	// Clear pitcher after each test
	pitcher.Set(nil)
	s.FunctionalTestBase.TearDownTest()
}

// TestSimpleWorkflow demonstrates running a workflow without fault injection
func (s *CatchSuite) TestSimpleWorkflow() {
	// Register workflow and start worker
	w := worker.New(s.SdkClient(), s.TaskQueue(), worker.Options{})
	kitchensink.RegisterWorkflows(w)
	s.NoError(w.Start())
	defer w.Stop()

	// Execute workflow with simple Omes actions - just one activity
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	workflowInput := &omeskitchensink.WorkflowInput{
		InitialActions: []*omeskitchensink.ActionSet{
			omeskitchensink.NoOpSingleActivityActionSet(),
		},
	}

	we, err := s.SdkClient().ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		TaskQueue: s.TaskQueue(),
	}, kitchensink.KitchenSinkWorkflow, workflowInput)
	s.NoError(err, "Should start workflow")

	// Wait for completion
	err = we.Get(ctx, nil)
	s.NoError(err, "Workflow should complete successfully")
}
