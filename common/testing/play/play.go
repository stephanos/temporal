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
	"time"

	"go.temporal.io/server/common/testing/pitcher"
)

// Play represents a composed test scenario: a sequence of pitcher configurations
// and expected outcomes for a specific workload.
type Play struct {
	// Name is a unique identifier for this play
	Name string

	// Description explains what this play tests
	Description string

	// Pitches defines the fault injection configurations for this play
	Pitches []Pitch

	// Workload specifies what workflow pattern to execute
	Workload WorkloadConfig

	// ExpectedOutcome defines success criteria
	ExpectedOutcome OutcomeConfig

	// Tags for categorization and filtering
	Tags []string
}

// Pitch defines a single fault injection point in a play
type Pitch struct {
	// Target is the intercept point (e.g., "matchingservice.AddWorkflowTask")
	Target string

	// Config is the pitcher configuration for this intercept point
	Config pitcher.PitchConfig

	// Description of what this pitch tests
	Description string
}

// WorkloadConfig specifies which workflow pattern to execute
type WorkloadConfig struct {
	// Type identifies the workflow type (e.g., "kitchensink", "simple")
	Type string

	// Variant specifies the configuration variant
	Variant string

	// Params contains workload-specific parameters
	Params map[string]any
}

// OutcomeConfig defines success criteria for a play
type OutcomeConfig struct {
	// ExpectedState is the expected final workflow state
	ExpectedState string

	// MaxDuration is the maximum allowed execution time
	MaxDuration time.Duration

	// AllowedViolations lists property violations that are acceptable for this play
	// Empty slice means no violations are allowed
	AllowedViolations []string

	// MinActivitiesCompleted is the minimum number of activities that should complete
	MinActivitiesCompleted int

	// RequireSignals indicates if signals must be processed
	RequireSignals bool
}

// PlayLibrary is a collection of plays organized by category
type PlayLibrary struct {
	// Plays indexed by name
	Plays map[string]*Play

	// Categories for organizing plays
	Categories map[string][]string
}

// NewPlayLibrary creates a new play library
func NewPlayLibrary() *PlayLibrary {
	return &PlayLibrary{
		Plays:      make(map[string]*Play),
		Categories: make(map[string][]string),
	}
}

// Register adds a play to the library
func (pl *PlayLibrary) Register(play *Play) {
	pl.Plays[play.Name] = play

	// Add to categories based on tags
	for _, tag := range play.Tags {
		pl.Categories[tag] = append(pl.Categories[tag], play.Name)
	}
}

// Get retrieves a play by name
func (pl *PlayLibrary) Get(name string) (*Play, bool) {
	play, ok := pl.Plays[name]
	return play, ok
}

// GetByCategory returns all plays with a specific tag
func (pl *PlayLibrary) GetByCategory(category string) []*Play {
	names := pl.Categories[category]
	plays := make([]*Play, 0, len(names))
	for _, name := range names {
		if play, ok := pl.Plays[name]; ok {
			plays = append(plays, play)
		}
	}
	return plays
}

// All returns all registered plays
func (pl *PlayLibrary) All() []*Play {
	plays := make([]*Play, 0, len(pl.Plays))
	for _, play := range pl.Plays {
		plays = append(plays, play)
	}
	return plays
}

// StandardPlayLibrary returns a library with common test scenarios
func StandardPlayLibrary() *PlayLibrary {
	lib := NewPlayLibrary()

	// Play 1: Simple workflow with no faults (baseline)
	lib.Register(&Play{
		Name:        "SimpleBaseline",
		Description: "Execute simple workflow with no fault injection",
		Pitches:     []Pitch{},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "simple",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            30 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 1,
		},
		Tags: []string{"baseline", "simple"},
	})

	// Play 2: Transient matching failure on first task
	lib.Register(&Play{
		Name:        "MatchingFailureRetry",
		Description: "Transient matching failure should not prevent workflow completion",
		Pitches: []Pitch{
			{
				Target: "matchingservice.AddWorkflowTask",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  1,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "Fail first task addition with RESOURCE_EXHAUSTED",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "simple",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            30 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 1,
		},
		Tags: []string{"resilience", "matching", "retry"},
	})

	// Play 3: Transient history failure on workflow start
	lib.Register(&Play{
		Name:        "HistoryStartFailure",
		Description: "Transient history failure on start should not prevent workflow execution",
		Pitches: []Pitch{
			{
				Target: "historyservice.StartWorkflowExecution",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  1,
					Params: map[string]any{
						"error": pitcher.ErrorDeadlineExceeded,
					},
				},
				Description: "Fail first workflow start with DEADLINE_EXCEEDED",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "simple",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            30 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 1,
		},
		Tags: []string{"resilience", "history", "retry"},
	})

	// Play 4: Activity task polling failure
	lib.Register(&Play{
		Name:        "ActivityPollFailure",
		Description: "Transient failure during activity poll should not prevent workflow completion",
		Pitches: []Pitch{
			{
				Target: "matchingservice.PollActivityTaskQueue",
				Config: pitcher.PitchConfig{
					Action:      "fail",
					Probability: 0.3, // 30% chance
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "30% chance of failure on activity polling",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "default",
			Params: map[string]any{
				"activity_count": 5,
			},
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            60 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 5,
		},
		Tags: []string{"resilience", "matching", "activity", "stochastic"},
	})

	// Play 5: Latency injection
	lib.Register(&Play{
		Name:        "MatchingLatency",
		Description: "Workflow should complete despite matching service latency",
		Pitches: []Pitch{
			{
				Target: "matchingservice.AddWorkflowTask",
				Config: pitcher.PitchConfig{
					Action:      "delay",
					Probability: 0.5,
					Params: map[string]any{
						"duration": 500 * time.Millisecond,
						"jitter":   100 * time.Millisecond,
					},
				},
				Description: "50% chance of 500ms delay with 100ms jitter",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "default",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            60 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 3,
		},
		Tags: []string{"performance", "latency", "matching"},
	})

	// Play 6: Multiple concurrent failures
	lib.Register(&Play{
		Name:        "MultiPointFailure",
		Description: "Workflow should handle failures at multiple service boundaries",
		Pitches: []Pitch{
			{
				Target: "matchingservice.AddWorkflowTask",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  1,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "Fail first workflow task addition",
			},
			{
				Target: "matchingservice.AddActivityTask",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  1,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "Fail first activity task addition",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "default",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            60 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 3,
		},
		Tags: []string{"resilience", "multipoint", "complex"},
	})

	// Play 7: Workflow task completion failure
	lib.Register(&Play{
		Name:        "WorkflowTaskCompletionFailure",
		Description: "Workflow task completion failure should be retried",
		Pitches: []Pitch{
			{
				Target: "historyservice.RespondWorkflowTaskCompleted",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  2,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "Fail first 2 workflow task completions",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "simple",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            30 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 1,
		},
		Tags: []string{"resilience", "history", "workflow-task"},
	})

	// Play 8: Frontend signal delivery failure
	lib.Register(&Play{
		Name:        "FrontendSignalFailure",
		Description: "Signal delivery failure should not prevent workflow completion",
		Pitches: []Pitch{
			{
				Target: "workflowservice.SignalWorkflowExecution",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  1,
					Params: map[string]any{
						"error": pitcher.ErrorUnavailable,
					},
				},
				Description: "Fail first signal delivery with UNAVAILABLE",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "simple",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            30 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 1,
		},
		Tags: []string{"resilience", "frontend", "signal"},
	})

	// Play 9: High latency scenario
	lib.Register(&Play{
		Name:        "HighMatchingLatency",
		Description: "Workflow should complete despite very high matching latency",
		Pitches: []Pitch{
			{
				Target: "matchingservice.AddWorkflowTask",
				Config: pitcher.PitchConfig{
					Action: "delay",
					Count:  3,
					Params: map[string]any{
						"duration": 2 * time.Second,
						"jitter":   500 * time.Millisecond,
					},
				},
				Description: "2 second delay on first 3 task additions",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "simple",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            60 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 1,
		},
		Tags: []string{"performance", "latency", "matching", "stress"},
	})

	// Play 10: Cascading failures
	lib.Register(&Play{
		Name:        "CascadingFailures",
		Description: "Sequential failures across multiple service boundaries",
		Pitches: []Pitch{
			{
				Target: "historyservice.StartWorkflowExecution",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  1,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "Fail workflow start",
			},
			{
				Target: "matchingservice.AddWorkflowTask",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  1,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "Fail workflow task addition",
			},
			{
				Target: "historyservice.RespondWorkflowTaskCompleted",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  1,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "Fail workflow task completion",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "simple",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            60 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 1,
		},
		Tags: []string{"resilience", "complex", "cascading"},
	})

	// Play 11: Activity task addition failure
	lib.Register(&Play{
		Name:        "ActivityTaskAddFailure",
		Description: "Activity task addition failure should not prevent workflow completion",
		Pitches: []Pitch{
			{
				Target: "matchingservice.AddActivityTask",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  2,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "Fail first 2 activity task additions",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "default",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            60 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 3,
		},
		Tags: []string{"resilience", "activity", "matching"},
	})

	// Play 12: Workflow poll latency
	lib.Register(&Play{
		Name:        "WorkflowPollLatency",
		Description: "Workflow task polling latency should not prevent completion",
		Pitches: []Pitch{
			{
				Target: "matchingservice.PollWorkflowTaskQueue",
				Config: pitcher.PitchConfig{
					Action:      "delay",
					Probability: 0.4,
					Params: map[string]any{
						"duration": 1 * time.Second,
						"jitter":   200 * time.Millisecond,
					},
				},
				Description: "40% chance of 1 second polling delay",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "default",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            90 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 3,
		},
		Tags: []string{"performance", "latency", "polling", "matching"},
	})

	// Play 13: Stochastic multipoint failures
	lib.Register(&Play{
		Name:        "StochasticMultipoint",
		Description: "Random failures across multiple points with probability",
		Pitches: []Pitch{
			{
				Target: "matchingservice.AddWorkflowTask",
				Config: pitcher.PitchConfig{
					Action:      "fail",
					Probability: 0.2,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "20% chance of workflow task addition failure",
			},
			{
				Target: "matchingservice.AddActivityTask",
				Config: pitcher.PitchConfig{
					Action:      "fail",
					Probability: 0.2,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "20% chance of activity task addition failure",
			},
			{
				Target: "historyservice.RespondWorkflowTaskCompleted",
				Config: pitcher.PitchConfig{
					Action:      "fail",
					Probability: 0.15,
					Params: map[string]any{
						"error": pitcher.ErrorDeadlineExceeded,
					},
				},
				Description: "15% chance of workflow task completion failure",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "default",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            90 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 3,
		},
		Tags: []string{"resilience", "stochastic", "multipoint", "complex"},
	})

	// Play 14: Timeout injection
	lib.Register(&Play{
		Name:        "TimeoutInjection",
		Description: "Timeout errors should be handled gracefully",
		Pitches: []Pitch{
			{
				Target: "matchingservice.PollWorkflowTaskQueue",
				Config: pitcher.PitchConfig{
					Action:      "timeout",
					Probability: 0.3,
				},
				Description: "30% chance of timeout on workflow task poll",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "simple",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            60 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 1,
		},
		Tags: []string{"resilience", "timeout", "matching"},
	})

	// Play 15: Service unavailable
	lib.Register(&Play{
		Name:        "ServiceUnavailable",
		Description: "UNAVAILABLE errors should be retried successfully",
		Pitches: []Pitch{
			{
				Target: "matchingservice.AddWorkflowTask",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  3,
					Params: map[string]any{
						"error": pitcher.ErrorUnavailable,
					},
				},
				Description: "3 consecutive UNAVAILABLE errors",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "simple",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            30 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 1,
		},
		Tags: []string{"resilience", "error-handling", "matching"},
	})

	// Play 16: Internal error handling
	lib.Register(&Play{
		Name:        "InternalErrorHandling",
		Description: "INTERNAL errors should be handled gracefully",
		Pitches: []Pitch{
			{
				Target: "historyservice.StartWorkflowExecution",
				Config: pitcher.PitchConfig{
					Action: "fail",
					Count:  2,
					Params: map[string]any{
						"error": pitcher.ErrorInternal,
					},
				},
				Description: "2 consecutive INTERNAL errors on workflow start",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "simple",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            30 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 1,
		},
		Tags: []string{"resilience", "error-handling", "history"},
	})

	// Play 17: Mixed latency and failures
	lib.Register(&Play{
		Name:        "MixedLatencyAndFailures",
		Description: "Combination of latency and failures across services",
		Pitches: []Pitch{
			{
				Target: "matchingservice.AddWorkflowTask",
				Config: pitcher.PitchConfig{
					Action:      "delay",
					Probability: 0.3,
					Params: map[string]any{
						"duration": 500 * time.Millisecond,
						"jitter":   100 * time.Millisecond,
					},
				},
				Description: "30% chance of 500ms latency",
			},
			{
				Target: "matchingservice.AddActivityTask",
				Config: pitcher.PitchConfig{
					Action:      "fail",
					Probability: 0.2,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "20% chance of failure",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "default",
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            90 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 3,
		},
		Tags: []string{"resilience", "performance", "complex", "hybrid"},
	})

	// Play 18: Heavy activity workload with failures
	lib.Register(&Play{
		Name:        "HeavyActivityWorkload",
		Description: "Workflow with many activities should handle intermittent failures",
		Pitches: []Pitch{
			{
				Target: "matchingservice.PollActivityTaskQueue",
				Config: pitcher.PitchConfig{
					Action:      "fail",
					Probability: 0.25,
					Params: map[string]any{
						"error": pitcher.ErrorResourceExhausted,
					},
				},
				Description: "25% chance of activity poll failure",
			},
		},
		Workload: WorkloadConfig{
			Type:    "kitchensink",
			Variant: "default",
			Params: map[string]any{
				"activity_count": 10,
			},
		},
		ExpectedOutcome: OutcomeConfig{
			ExpectedState:          "completed",
			MaxDuration:            120 * time.Second,
			AllowedViolations:      []string{},
			MinActivitiesCompleted: 10,
		},
		Tags: []string{"resilience", "activity", "stress", "high-load"},
	})

	return lib
}
