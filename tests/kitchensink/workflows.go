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

package kitchensink

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WorkflowParams configures which features the kitchensink workflow will exercise
type WorkflowParams struct {
	// Activities configuration
	ActivityCount      int
	ActivityDuration   time.Duration
	ActivityRetryLimit int

	// Timer configuration
	UseTimer      bool
	TimerDuration time.Duration

	// Child workflow configuration
	UseChildWorkflow bool
	ChildCount       int

	// Signal configuration
	WaitForSignal bool
	SignalName    string

	// Query configuration
	EnableQuery bool
	QueryName   string

	// Cancellation
	HandleCancellation bool
}

// DefaultParams returns a kitchensink workflow with all features enabled
func DefaultParams() WorkflowParams {
	return WorkflowParams{
		ActivityCount:      3,
		ActivityDuration:   100 * time.Millisecond,
		ActivityRetryLimit: 3,
		UseTimer:           true,
		TimerDuration:      100 * time.Millisecond,
		UseChildWorkflow:   true,
		ChildCount:         2,
		WaitForSignal:      false,
		SignalName:         "test-signal",
		EnableQuery:        true,
		QueryName:          "state",
		HandleCancellation: true,
	}
}

// SimpleParams returns minimal workflow configuration
func SimpleParams() WorkflowParams {
	return WorkflowParams{
		ActivityCount:    1,
		ActivityDuration: 50 * time.Millisecond,
		UseTimer:         false,
	}
}

// WorkflowState tracks workflow execution state
type WorkflowState struct {
	ActivitiesCompleted int
	TimersCompleted     int
	ChildrenCompleted   int
	SignalsReceived     int
	Status              string
}

// KitchensinkWorkflow demonstrates comprehensive Temporal features
func KitchensinkWorkflow(ctx workflow.Context, params WorkflowParams) (*WorkflowState, error) {
	state := &WorkflowState{Status: "running"}
	logger := workflow.GetLogger(ctx)

	// Set up query handler
	if params.EnableQuery {
		err := workflow.SetQueryHandler(ctx, params.QueryName, func() (*WorkflowState, error) {
			return state, nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to set query handler: %w", err)
		}
	}

	// Set up signal handler
	signalChan := workflow.GetSignalChannel(ctx, params.SignalName)
	signalReceived := false

	// Set up cancellation handling
	if params.HandleCancellation {
		defer func() {
			if ctx.Err() != nil {
				state.Status = "cancelled"
			}
		}()
	}

	// Execute activities
	logger.Info("Starting activities", "count", params.ActivityCount)
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: int32(params.ActivityRetryLimit),
		},
	}
	activityCtx := workflow.WithActivityOptions(ctx, activityOptions)

	for i := 0; i < params.ActivityCount; i++ {
		var result string
		err := workflow.ExecuteActivity(activityCtx, SimpleActivity, i, params.ActivityDuration).Get(ctx, &result)
		if err != nil {
			return state, fmt.Errorf("activity %d failed: %w", i, err)
		}
		state.ActivitiesCompleted++
		logger.Info("Activity completed", "index", i, "result", result)
	}

	// Execute timer
	if params.UseTimer {
		logger.Info("Starting timer", "duration", params.TimerDuration)
		err := workflow.Sleep(ctx, params.TimerDuration)
		if err != nil {
			return state, fmt.Errorf("timer failed: %w", err)
		}
		state.TimersCompleted++
		logger.Info("Timer completed")
	}

	// Execute child workflows
	if params.UseChildWorkflow {
		logger.Info("Starting child workflows", "count", params.ChildCount)
		childOptions := workflow.ChildWorkflowOptions{
			WorkflowExecutionTimeout: 30 * time.Second,
		}
		childCtx := workflow.WithChildOptions(ctx, childOptions)

		for i := 0; i < params.ChildCount; i++ {
			childParams := SimpleParams()
			var childState *WorkflowState
			childFuture := workflow.ExecuteChildWorkflow(childCtx, KitchensinkWorkflow, childParams)
			err := childFuture.Get(ctx, &childState)
			if err != nil {
				return state, fmt.Errorf("child workflow %d failed: %w", i, err)
			}
			state.ChildrenCompleted++
			logger.Info("Child workflow completed", "index", i)
		}
	}

	// Wait for signal if configured
	if params.WaitForSignal {
		logger.Info("Waiting for signal", "name", params.SignalName)
		var signalData string
		signalChan.Receive(ctx, &signalData)
		signalReceived = true
		state.SignalsReceived++
		logger.Info("Signal received", "data", signalData)
	} else {
		// Check for signal non-blocking
		signalChan.ReceiveAsync(&signalReceived)
		if signalReceived {
			state.SignalsReceived++
		}
	}

	state.Status = "completed"
	logger.Info("Kitchensink workflow completed", "state", state)
	return state, nil
}

// SimpleActivity performs a simple task with configurable duration
func SimpleActivity(ctx context.Context, index int, duration time.Duration) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("SimpleActivity started", "index", index, "duration", duration)

	// Simulate work
	time.Sleep(duration)

	result := fmt.Sprintf("activity-%d-completed", index)
	logger.Info("SimpleActivity completed", "result", result)
	return result, nil
}

// FailingActivity simulates an activity that fails a certain number of times before succeeding
func FailingActivity(ctx context.Context, failCount int) (string, error) {
	logger := activity.GetLogger(ctx)
	info := activity.GetInfo(ctx)

	attempt := info.Attempt
	logger.Info("FailingActivity attempt", "attempt", attempt, "failCount", failCount)

	if int(attempt) <= failCount {
		return "", fmt.Errorf("simulated failure (attempt %d/%d)", attempt, failCount)
	}

	return fmt.Sprintf("succeeded-after-%d-attempts", failCount), nil
}

// LongRunningActivity simulates a long-running activity with heartbeats
func LongRunningActivity(ctx context.Context, duration time.Duration) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("LongRunningActivity started", "duration", duration)

	// Send heartbeats every 100ms
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.Now().Add(duration)
	progress := 0

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			progress += 10
			activity.RecordHeartbeat(ctx, progress)
			logger.Info("LongRunningActivity heartbeat", "progress", progress)
		}
	}

	logger.Info("LongRunningActivity completed")
	return "long-running-completed", nil
}
