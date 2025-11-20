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

package pitcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPitcher_NoConfig(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Should return nil when no config exists
	err := p.Throw(ctx, "some.target")
	assert.NoError(t, err)
}

func TestPitcher_FailAction(t *testing.T) {
	p := New()
	ctx := context.Background()

	p.Configure("test.target", PitchConfig{
		Action: "fail",
		Params: map[string]any{
			"error": ErrorResourceExhausted,
		},
	})

	err := p.Throw(ctx, "test.target")
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
}

func TestPitcher_FailActionDefaultError(t *testing.T) {
	p := New()
	ctx := context.Background()

	p.Configure("test.target", PitchConfig{
		Action: "fail",
	})

	err := p.Throw(ctx, "test.target")
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestPitcher_DelayAction(t *testing.T) {
	p := New()
	ctx := context.Background()

	p.Configure("test.target", PitchConfig{
		Action: "delay",
		Params: map[string]any{
			"duration": 50 * time.Millisecond,
		},
	})

	start := time.Now()
	err := p.Throw(ctx, "test.target")
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 50*time.Millisecond)
	assert.Less(t, duration, 100*time.Millisecond) // sanity check
}

func TestPitcher_TimeoutAction(t *testing.T) {
	p := New()
	ctx := context.Background()

	p.Configure("test.target", PitchConfig{
		Action: "timeout",
		Params: map[string]any{
			"duration": 10 * time.Millisecond,
		},
	})

	err := p.Throw(ctx, "test.target")
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.DeadlineExceeded, st.Code())
}

func TestPitcher_CountLimit(t *testing.T) {
	p := New()
	ctx := context.Background()

	p.Configure("test.target", PitchConfig{
		Action: "fail",
		Count:  2, // Only inject twice
		Params: map[string]any{
			"error": ErrorUnavailable,
		},
	})

	// First two calls should fail
	err1 := p.Throw(ctx, "test.target")
	assert.Error(t, err1)

	err2 := p.Throw(ctx, "test.target")
	assert.Error(t, err2)

	// Third call should succeed (no injection)
	err3 := p.Throw(ctx, "test.target")
	assert.NoError(t, err3)
}

func TestPitcher_Probability(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Configure with 50% probability
	p.Configure("test.target", PitchConfig{
		Action:      "fail",
		Probability: 0.5,
		Params: map[string]any{
			"error": ErrorInternal,
		},
	})

	// Call 100 times and count failures
	failures := 0
	iterations := 100
	for i := 0; i < iterations; i++ {
		if err := p.Throw(ctx, "test.target"); err != nil {
			failures++
		}
	}

	// Should be roughly 50% (allow 20-80% range due to randomness)
	assert.Greater(t, failures, iterations/5)
	assert.Less(t, failures, iterations*4/5)
}

func TestPitcher_Reset(t *testing.T) {
	p := New()
	ctx := context.Background()

	p.Configure("test.target", PitchConfig{
		Action: "fail",
		Params: map[string]any{
			"error": ErrorInternal,
		},
	})

	// Should fail before reset
	err := p.Throw(ctx, "test.target")
	assert.Error(t, err)

	// Reset
	p.Reset()

	// Should not fail after reset
	err = p.Throw(ctx, "test.target")
	assert.NoError(t, err)
}

func TestPitcher_GlobalState(t *testing.T) {
	// Save and restore global state
	oldPitcher := Get()
	defer Set(oldPitcher)

	p := New()
	Set(p)

	assert.Equal(t, p, Get())

	Set(nil)
	assert.Nil(t, Get())
}

func TestPitcher_ContextCancellation(t *testing.T) {
	p := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	p.Configure("test.target", PitchConfig{
		Action: "delay",
		Params: map[string]any{
			"duration": 1 * time.Second,
		},
	})

	err := p.Throw(ctx, "test.target")
	assert.Equal(t, context.Canceled, err)
}

func TestPitcher_DelayWithJitter(t *testing.T) {
	p := New()
	ctx := context.Background()

	p.Configure("test.target", PitchConfig{
		Action: "delay",
		Params: map[string]any{
			"duration": 100 * time.Millisecond,
			"jitter":   0.5, // ±50% jitter
		},
	})

	// Measure several delays to check jitter is applied
	delays := make([]time.Duration, 10)
	for i := 0; i < len(delays); i++ {
		start := time.Now()
		err := p.Throw(ctx, "test.target")
		delays[i] = time.Since(start)
		assert.NoError(t, err)
	}

	// All delays should be different due to jitter
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}
	assert.False(t, allSame, "Expected delays to vary due to jitter")

	// All delays should be within expected range (50-150ms with some tolerance)
	for _, d := range delays {
		assert.GreaterOrEqual(t, d, 40*time.Millisecond)
		assert.LessOrEqual(t, d, 160*time.Millisecond)
	}
}
