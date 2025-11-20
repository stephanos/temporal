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

// Package pitcher provides fault injection capabilities for tests.
// It follows the interceptor pattern to inject failures, delays, and other
// chaos into RPC calls and persistence operations during testing.
//
// Pitcher is only active during tests and uses a global state pattern
// similar to testhooks.TestHooks.
package pitcher

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Pitcher provides fault injection for tests.
// It is thread-safe and designed to be used as a global singleton.
type Pitcher interface {
	// Throw executes a pitch (atomic fault injection action).
	// Returns error to inject (nil = no injection).
	// The target identifies what's being intercepted (e.g., "matchingservice.AddWorkflowTask").
	Throw(ctx context.Context, target string) error

	// Configure sets pitch configuration for a target.
	Configure(target string, config PitchConfig)

	// Reset clears all configurations.
	Reset()
}

// PitchConfig defines how to inject faults for a specific target.
type PitchConfig struct {
	// Action specifies what to do: "fail", "delay", "timeout", etc.
	Action string

	// Probability controls random injection (0.0-1.0, where 1.0 = always inject).
	Probability float64

	// Count limits how many times to inject (0 = unlimited, N = inject N times then stop).
	Count int

	// Params contains action-specific parameters.
	Params map[string]any
}

// Common error codes for fault injection
const (
	ErrorResourceExhausted = "RESOURCE_EXHAUSTED"
	ErrorDeadlineExceeded  = "DEADLINE_EXCEEDED"
	ErrorUnavailable       = "UNAVAILABLE"
	ErrorInternal          = "INTERNAL"
	ErrorCanceled          = "CANCELED"
	ErrorAborted           = "ABORTED"
)

// DelayParams specifies delay configuration
type DelayParams struct {
	Duration time.Duration // How long to delay
	Jitter   float64       // Random jitter (0.0-1.0)
}

// pitcherImpl implements the Pitcher interface
type pitcherImpl struct {
	mu      sync.RWMutex
	configs map[string]*pitchState
	rand    *rand.Rand
}

// pitchState tracks execution state for a configured pitch
type pitchState struct {
	config    PitchConfig
	executed  int // how many times this pitch has been executed
	mu        sync.Mutex
}

// Global pitcher instance (only active in tests)
var (
	globalPitcher Pitcher
	pitcherMu     sync.RWMutex
)

// New creates a new Pitcher instance
func New() Pitcher {
	return &pitcherImpl{
		configs: make(map[string]*pitchState),
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Get returns the global pitcher (nil if not in test mode)
func Get() Pitcher {
	pitcherMu.RLock()
	defer pitcherMu.RUnlock()
	return globalPitcher
}

// Set configures the global pitcher (test setup only)
func Set(p Pitcher) {
	pitcherMu.Lock()
	defer pitcherMu.Unlock()
	globalPitcher = p
}

// Throw implements Pitcher.Throw
func (p *pitcherImpl) Throw(ctx context.Context, target string) error {
	p.mu.RLock()
	state, ok := p.configs[target]
	p.mu.RUnlock()

	if !ok {
		return nil // no configuration for this target
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	// Check if we've exceeded the count limit
	if state.config.Count > 0 && state.executed >= state.config.Count {
		return nil
	}

	// Check probability
	if state.config.Probability < 1.0 {
		if p.rand.Float64() >= state.config.Probability {
			return nil
		}
	}

	// Execute the pitch
	state.executed++
	return p.executePitch(ctx, target, state.config)
}

// executePitch performs the actual fault injection
func (p *pitcherImpl) executePitch(ctx context.Context, target string, config PitchConfig) error {
	switch config.Action {
	case "fail":
		return p.executeFail(config)
	case "delay":
		return p.executeDelay(ctx, config)
	case "timeout":
		return p.executeTimeout(ctx, config)
	default:
		return fmt.Errorf("pitcher: unknown action %q for target %q", config.Action, target)
	}
}

// executeFail returns a gRPC error
func (p *pitcherImpl) executeFail(config PitchConfig) error {
	errorCode, ok := config.Params["error"].(string)
	if !ok {
		errorCode = ErrorInternal
	}

	code := parseErrorCode(errorCode)
	message := fmt.Sprintf("pitcher injected failure: %s", errorCode)
	if msg, ok := config.Params["message"].(string); ok {
		message = msg
	}

	return status.Error(code, message)
}

// executeDelay sleeps for the configured duration
func (p *pitcherImpl) executeDelay(ctx context.Context, config PitchConfig) error {
	duration, ok := config.Params["duration"].(time.Duration)
	if !ok {
		duration = 100 * time.Millisecond // default delay
	}

	// Apply jitter if specified
	if jitter, ok := config.Params["jitter"].(float64); ok && jitter > 0 {
		jitterAmount := float64(duration) * jitter * (p.rand.Float64() - 0.5) * 2
		duration += time.Duration(jitterAmount)
	}

	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// executeTimeout returns a deadline exceeded error after a delay
func (p *pitcherImpl) executeTimeout(ctx context.Context, config PitchConfig) error {
	// First delay, then return timeout error
	if err := p.executeDelay(ctx, config); err != nil {
		return err
	}
	return status.Error(codes.DeadlineExceeded, "pitcher injected timeout")
}

// Configure implements Pitcher.Configure
func (p *pitcherImpl) Configure(target string, config PitchConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Validate config
	if config.Action == "" {
		config.Action = "fail" // default action
	}
	if config.Probability == 0 {
		config.Probability = 1.0 // default to always inject
	}

	p.configs[target] = &pitchState{
		config:   config,
		executed: 0,
	}
}

// Reset implements Pitcher.Reset
func (p *pitcherImpl) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.configs = make(map[string]*pitchState)
}

// parseErrorCode converts string error codes to gRPC codes
func parseErrorCode(errorCode string) codes.Code {
	switch errorCode {
	case ErrorResourceExhausted:
		return codes.ResourceExhausted
	case ErrorDeadlineExceeded:
		return codes.DeadlineExceeded
	case ErrorUnavailable:
		return codes.Unavailable
	case ErrorInternal:
		return codes.Internal
	case ErrorCanceled:
		return codes.Canceled
	case ErrorAborted:
		return codes.Aborted
	default:
		return codes.Internal
	}
}
