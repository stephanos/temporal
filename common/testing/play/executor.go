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

package play

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/testing/pitcher"
)

// Executor runs plays and validates outcomes
type Executor struct {
	logger  log.Logger
	pitcher pitcher.Pitcher
}

// NewExecutor creates a new play executor
func NewExecutor(logger log.Logger, pitcher pitcher.Pitcher) *Executor {
	return &Executor{
		logger:  logger,
		pitcher: pitcher,
	}
}

// PlayResult contains the outcome of executing a play
type PlayResult struct {
	// Play that was executed
	Play *Play

	// Success indicates if the play passed all criteria
	Success bool

	// Duration is how long the play took to execute
	Duration time.Duration

	// Violations contains any property violations detected
	Violations []string

	// Error contains execution error if any
	Error error

	// WorkflowResult contains the workflow execution result
	WorkflowResult any
}

// ExecutePlay runs a single play and validates the outcome
func (e *Executor) ExecutePlay(
	ctx context.Context,
	play *Play,
	sdkClient client.Client,
	workflowFunc interface{},
	taskQueue string,
) *PlayResult {
	result := &PlayResult{
		Play:       play,
		Violations: []string{},
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	e.logger.Info("Executing play", tag.NewStringTag("name", play.Name), tag.NewStringTag("description", play.Description))

	// Configure pitcher with all pitches
	for _, pitch := range play.Pitches {
		e.logger.Info("Configuring pitch", tag.NewStringTag("target", pitch.Target), tag.NewStringTag("description", pitch.Description))
		e.pitcher.Configure(pitch.Target, pitch.Config)
	}

	// Execute the workflow
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue:           taskQueue,
		WorkflowRunTimeout:  play.ExpectedOutcome.MaxDuration,
		WorkflowTaskTimeout: 10 * time.Second,
	}

	workflowRun, err := sdkClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		workflowFunc,
		e.buildWorkflowParams(play.Workload),
	)
	if err != nil {
		result.Error = fmt.Errorf("failed to start workflow: %w", err)
		result.Success = false
		return result
	}

	// Wait for workflow to complete
	err = workflowRun.Get(ctx, &result.WorkflowResult)
	if err != nil {
		result.Error = fmt.Errorf("workflow execution failed: %w", err)
		result.Success = false
		return result
	}

	// Validate outcome
	result.Success = e.validateOutcome(play.ExpectedOutcome, result)

	e.logger.Info("Play execution completed",
		tag.NewStringTag("name", play.Name),
		tag.NewBoolTag("success", result.Success),
		tag.NewDurationTag("duration", result.Duration))
	return result
}

// buildWorkflowParams constructs workflow parameters from workload config
func (e *Executor) buildWorkflowParams(workload WorkloadConfig) any {
	// For now, return nil to use workflow's defaults
	// The workflow function signature will determine what params are needed
	// Tests can pass specific params via workload.Params if needed
	return nil
}

// validateOutcome checks if the play result matches expected outcome
func (e *Executor) validateOutcome(expected OutcomeConfig, result *PlayResult) bool {
	// Check duration
	if result.Duration > expected.MaxDuration {
		e.logger.Warn("Play exceeded max duration",
			tag.NewDurationTag("duration", result.Duration),
			tag.NewDurationTag("max", expected.MaxDuration))
		return false
	}

	// Check violations
	if len(result.Violations) > 0 {
		// Check if violations are allowed
		allowedMap := make(map[string]bool)
		for _, allowed := range expected.AllowedViolations {
			allowedMap[allowed] = true
		}

		for _, violation := range result.Violations {
			if !allowedMap[violation] {
				e.logger.Warn("Unexpected violation", tag.NewStringTag("violation", violation))
				return false
			}
		}
	}

	// Additional validation could check workflow state, activities completed, etc.
	// This would require inspecting result.WorkflowResult

	return true
}

// ExecutePlaySequence runs multiple plays in sequence
func (e *Executor) ExecutePlaySequence(
	ctx context.Context,
	plays []*Play,
	sdkClient client.Client,
	workflowFunc interface{},
	taskQueue string,
) []*PlayResult {
	results := make([]*PlayResult, 0, len(plays))

	for _, play := range plays {
		// Reset pitcher between plays
		e.pitcher.Reset()

		result := e.ExecutePlay(ctx, play, sdkClient, workflowFunc, taskQueue)
		results = append(results, result)

		// Stop on first failure if configured
		if !result.Success {
			e.logger.Warn("Play failed, stopping sequence", tag.NewStringTag("play", play.Name))
			// Could make this configurable
		}
	}

	return results
}

// GamePlan represents a collection of plays that should be executed together
type GamePlan struct {
	// Name of the game plan
	Name string

	// Description explains the testing strategy
	Description string

	// Plays to execute
	Plays []*Play

	// Coverage tracks which scenarios have been tested
	Coverage map[string]bool

	// StopOnFailure indicates whether to stop on first failure
	StopOnFailure bool
}

// NewGamePlan creates a new game plan
func NewGamePlan(name, description string) *GamePlan {
	return &GamePlan{
		Name:          name,
		Description:   description,
		Plays:         []*Play{},
		Coverage:      make(map[string]bool),
		StopOnFailure: false,
	}
}

// AddPlay adds a play to the game plan
func (gp *GamePlan) AddPlay(play *Play) {
	gp.Plays = append(gp.Plays, play)
	for _, tag := range play.Tags {
		gp.Coverage[tag] = false
	}
}

// Execute runs all plays in the game plan
func (gp *GamePlan) Execute(
	ctx context.Context,
	executor *Executor,
	sdkClient client.Client,
	workflowFunc interface{},
	taskQueue string,
) []*PlayResult {
	results := make([]*PlayResult, 0, len(gp.Plays))

	for _, play := range gp.Plays {
		executor.pitcher.Reset()

		result := executor.ExecutePlay(ctx, play, sdkClient, workflowFunc, taskQueue)
		results = append(results, result)

		// Update coverage
		if result.Success {
			for _, tag := range play.Tags {
				gp.Coverage[tag] = true
			}
		}

		if !result.Success && gp.StopOnFailure {
			executor.logger.Warn("Play failed, stopping game plan",
				tag.NewStringTag("play", play.Name),
				tag.NewStringTag("gameplan", gp.Name))
			break
		}
	}

	return results
}

// GetCoverageReport returns a summary of covered scenarios
func (gp *GamePlan) GetCoverageReport() map[string]bool {
	return gp.Coverage
}
