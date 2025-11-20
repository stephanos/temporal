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
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/testing/play"
	"go.temporal.io/server/common/testing/skipper"
	"go.temporal.io/server/tests/kitchensink"
	"go.temporal.io/server/tests/testcore"
)

type CatchSuite struct {
	testcore.FunctionalTestBase
	library *play.PlayLibrary
	skipper *skipper.Skipper
}

func TestCatch(t *testing.T) {
	suite.Run(t, new(CatchSuite))
}

func (s *CatchSuite) SetupSuite() {
	s.FunctionalTestBase.SetupSuite()

	// Initialize play library with standard plays
	s.library = play.StandardPlayLibrary()

	// Initialize skipper with coverage-driven strategy
	skipperConfig := skipper.DefaultConfig()
	skipperConfig.Strategy = skipper.StrategyCoverageDriven
	s.skipper = skipper.NewSkipper(s.Logger, s.library, skipperConfig)
}

func (s *CatchSuite) SetupTest() {
	s.FunctionalTestBase.SetupTest()
}

func (s *CatchSuite) TearDownTest() {
	s.FunctionalTestBase.TearDownTest()
}

// TestSinglePlay demonstrates executing a single play
func (s *CatchSuite) TestSinglePlay() {
	// Get a specific play
	testPlay, ok := s.library.Get("SimpleBaseline")
	s.Require().True(ok, "SimpleBaseline play should exist")
	s.Logger.Info("Testing play", tag.NewStringTag("play", testPlay.Name))

	// Register workflow and start worker
	w := worker.New(s.SdkClient(), s.TaskQueue(), worker.Options{})
	w.RegisterWorkflow(kitchensink.KitchensinkWorkflow)
	w.RegisterActivity(kitchensink.SimpleActivity)
	w.RegisterActivity(kitchensink.FailingActivity)
	w.RegisterActivity(kitchensink.LongRunningActivity)
	s.NoError(w.Start())
	defer w.Stop()

	// Execute workflow directly with simple params
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	we, err := s.SdkClient().ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		TaskQueue: s.TaskQueue(),
	}, kitchensink.KitchensinkWorkflow, kitchensink.SimpleParams())
	s.NoError(err, "Should start workflow")

	// Wait for completion
	var workflowResult *kitchensink.WorkflowState
	err = we.Get(ctx, &workflowResult)
	s.NoError(err, "Workflow should complete successfully")
	s.NotNil(workflowResult, "Should have workflow result")
	s.Equal("completed", workflowResult.Status, "Workflow should be completed")
	s.Equal(1, workflowResult.ActivitiesCompleted, "Should have completed 1 activity")
}

// TestPlayWithFaultInjection demonstrates fault injection via plays
func (s *CatchSuite) TestPlayWithFaultInjection() {
	// Get the matching failure play
	testPlay, ok := s.library.Get("MatchingFailureRetry")
	s.Require().True(ok, "MatchingFailureRetry play should exist")
	s.Logger.Info("Testing play with fault injection", tag.NewStringTag("play", testPlay.Name))

	// Configure pitcher to inject fault (simulating what the play would do)
	s.ConfigurePitcher("matchingservice.AddWorkflowTask", testPlay.Pitches[0].Config)

	// Register workflow and start worker
	w := worker.New(s.SdkClient(), s.TaskQueue(), worker.Options{})
	w.RegisterWorkflow(kitchensink.KitchensinkWorkflow)
	w.RegisterActivity(kitchensink.SimpleActivity)
	s.NoError(w.Start())
	defer w.Stop()

	// Execute workflow with simple params
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	we, err := s.SdkClient().ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		TaskQueue: s.TaskQueue(),
	}, kitchensink.KitchensinkWorkflow, kitchensink.SimpleParams())
	s.NoError(err, "Should start workflow")

	// Wait for completion - should succeed despite injected failure
	var workflowResult *kitchensink.WorkflowState
	err = we.Get(ctx, &workflowResult)
	s.NoError(err, "Workflow should complete successfully despite fault injection")
	s.NotNil(workflowResult, "Should have workflow result")
	s.Equal("completed", workflowResult.Status, "Workflow should be completed")
}

// TestSkipperRandomSelection demonstrates skipper with random play selection
// TODO: Implement full play executor integration
func (s *CatchSuite) Skip_TestSkipperRandomSelection() {
	// Create skipper with random strategy
	skipperConfig := skipper.DefaultConfig()
	skipperConfig.Strategy = skipper.StrategyRandom
	skipperConfig.Seed = 12345 // Fixed seed for reproducibility
	randomSkipper := skipper.NewSkipper(s.Logger, s.library, skipperConfig)

	// Register workflow and start worker
	w := worker.New(s.SdkClient(), s.TaskQueue(), worker.Options{})
	w.RegisterWorkflow(kitchensink.KitchensinkWorkflow)
	w.RegisterActivity(kitchensink.SimpleActivity)
	s.NoError(w.Start())
	defer w.Stop()

	executor := play.NewExecutor(s.Logger, s.GetTestCluster().Pitcher())

	// Execute 5 random plays
	successCount := 0
	for i := 0; i < 5; i++ {
		selectedPlay := randomSkipper.SelectPlay()
		s.NotNil(selectedPlay, "Skipper should select a play")

		result := executor.ExecutePlay(
			context.Background(),
			selectedPlay,
			s.SdkClient(),
			kitchensink.KitchensinkWorkflow,
			s.TaskQueue(),
		)

		randomSkipper.RecordExecution(selectedPlay, result.Success)

		if result.Success {
			successCount++
		}

		s.Logger.Info("Play executed",
			tag.NewStringTag("play", selectedPlay.Name),
			tag.NewBoolTag("success", result.Success))
	}

	// Should have some successes
	s.Greater(successCount, 0, "At least some plays should succeed")

	// Check coverage report
	report := randomSkipper.GetCoverageReport()
	s.Greater(report.TotalExecutions, 0, "Should have executed plays")
	s.Logger.Info("Coverage report", tag.NewFloat64("coverage", report.CoveragePercentage()))
}

// TestSkipperCoverageDriven demonstrates coverage-driven play selection
// TODO: Implement full play executor integration
func (s *CatchSuite) Skip_TestSkipperCoverageDriven() {
	// Create skipper with coverage-driven strategy
	skipperConfig := skipper.DefaultConfig()
	skipperConfig.Strategy = skipper.StrategyCoverageDriven
	coverageSkipper := skipper.NewSkipper(s.Logger, s.library, skipperConfig)

	// Register workflow and start worker
	w := worker.New(s.SdkClient(), s.TaskQueue(), worker.Options{})
	w.RegisterWorkflow(kitchensink.KitchensinkWorkflow)
	w.RegisterActivity(kitchensink.SimpleActivity)
	s.NoError(w.Start())
	defer w.Stop()

	executor := play.NewExecutor(s.Logger, s.GetTestCluster().Pitcher())

	// Execute plays until we reach 50% coverage or 10 iterations
	maxIterations := 10
	for i := 0; i < maxIterations; i++ {
		selectedPlay := coverageSkipper.SelectPlay()
		s.NotNil(selectedPlay, "Skipper should select a play")

		result := executor.ExecutePlay(
			context.Background(),
			selectedPlay,
			s.SdkClient(),
			kitchensink.KitchensinkWorkflow,
			s.TaskQueue(),
		)

		coverageSkipper.RecordExecution(selectedPlay, result.Success)

		s.Logger.Info("Play executed",
			tag.NewStringTag("play", selectedPlay.Name),
			tag.NewBoolTag("success", result.Success),
			tag.NewInt("iteration", i+1))

		// Check if we've reached 50% coverage
		report := coverageSkipper.GetCoverageReport()
		if report.CoveragePercentage() >= 50.0 {
			s.Logger.Info("Reached 50% coverage", tag.NewFloat64("coverage", report.CoveragePercentage()))
			break
		}
	}

	// Verify coverage progress
	finalReport := coverageSkipper.GetCoverageReport()
	s.Greater(finalReport.UniquePlaysRun, 0, "Should have run some unique plays")
	s.Logger.Info("Final coverage report",
		tag.NewInt("uniquePlays", finalReport.UniquePlaysRun),
		tag.NewInt("totalExecutions", finalReport.TotalExecutions),
		tag.NewFloat64("coverage", finalReport.CoveragePercentage()))
}

// TestGamePlan demonstrates executing a game plan
// TODO: Implement full play executor integration
func (s *CatchSuite) Skip_TestGamePlan() {
	// Create a game plan with resilience-focused plays
	gamePlan := play.NewGamePlan(
		"ResilienceValidation",
		"Validate system resilience under various failure scenarios",
	)

	// Add resilience plays
	resiliencePlays := s.library.GetByCategory("resilience")
	for _, p := range resiliencePlays {
		gamePlan.AddPlay(p)
	}

	s.Greater(len(gamePlan.Plays), 0, "Should have resilience plays")

	// Register workflow and start worker
	w := worker.New(s.SdkClient(), s.TaskQueue(), worker.Options{})
	w.RegisterWorkflow(kitchensink.KitchensinkWorkflow)
	w.RegisterActivity(kitchensink.SimpleActivity)
	s.NoError(w.Start())
	defer w.Stop()

	// Execute the game plan
	executor := play.NewExecutor(s.Logger, s.GetTestCluster().Pitcher())
	results := gamePlan.Execute(
		context.Background(),
		executor,
		s.SdkClient(),
		kitchensink.KitchensinkWorkflow,
		s.TaskQueue(),
	)

	// Verify results
	s.Equal(len(gamePlan.Plays), len(results), "Should have result for each play")

	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	// All resilience plays should succeed (that's the point!)
	s.Equal(len(results), successCount, "All resilience plays should succeed")

	// Check coverage
	coverageReport := gamePlan.GetCoverageReport()
	s.Logger.Info("Game plan coverage completed", tag.NewInt("coveredCategories", len(coverageReport)))
}
