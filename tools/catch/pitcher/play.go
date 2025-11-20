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

import "time"

// Play is an atomic action that can be initiated by a user or pitcher,
// and recorded by the scorebook. Play describes only the INTENTION
// (delay, fail, cancel, timeout) without knowledge of specific services.
type Play struct {
	// Type specifies what action to take
	Type PlayType

	// Params contains action-specific parameters
	Params map[string]any

	// Timestamp when the play was made (set by pitcher/scorebook)
	Timestamp time.Time

	// Target is set at runtime when play is made (for scorebook recording)
	Target string
}

// PlayType defines the types of plays that can be made
type PlayType string

const (
	// PlayDelay injects a delay before continuing
	PlayDelay PlayType = "delay"

	// PlayFail returns an error instead of continuing
	PlayFail PlayType = "fail"

	// PlayCancel cancels the operation
	PlayCancel PlayType = "cancel"

	// PlayTimeout simulates a timeout
	PlayTimeout PlayType = "timeout"

	// PlayDrop drops the request without response
	PlayDrop PlayType = "drop"
)

// Common parameter keys for plays
const (
	ParamDuration = "duration" // time.Duration for delays/timeouts
	ParamError    = "error"    // error code/message for failures
	ParamJitter   = "jitter"   // float64 jitter amount (0.0-1.0)
)

// NewPlay creates a new play with the specified type and parameters
func NewPlay(playType PlayType, params map[string]any) Play {
	if params == nil {
		params = make(map[string]any)
	}
	return Play{
		Type:      playType,
		Params:    params,
		Timestamp: time.Now(),
	}
}

// DelayPlay creates a delay play
func DelayPlay(duration time.Duration, jitter float64) Play {
	return NewPlay(PlayDelay, map[string]any{
		ParamDuration: duration,
		ParamJitter:   jitter,
	})
}

// FailPlay creates a fail play
func FailPlay(errorCode string) Play {
	return NewPlay(PlayFail, map[string]any{
		ParamError: errorCode,
	})
}

// TimeoutPlay creates a timeout play
func TimeoutPlay(duration time.Duration) Play {
	return NewPlay(PlayTimeout, map[string]any{
		ParamDuration: duration,
	})
}

// CancelPlay creates a cancel play
func CancelPlay() Play {
	return NewPlay(PlayCancel, nil)
}

// DropPlay creates a drop play
func DropPlay() Play {
	return NewPlay(PlayDrop, nil)
}
