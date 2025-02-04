// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
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

package propmodel

import (
	"strings"
	"time"
)

type (
	State struct {
		init  bool
		Label string
	}
	stateOpt func(*State)
)

func NewState(label string, opts ...stateOpt) *State {
	s := &State{Label: label}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

var Init = func(s *State) {
	s.init = true
}

func (s *State) String() string {
	return s.Label
}

func WaitForState(m *Model, desiredStates ...*State) {
	ctx := NewContext()

	desiredStatesStr := make([]string, len(desiredStates))
	for i, s := range desiredStates {
		desiredStatesStr[i] = s.String()
	}
	desiredStatesDescr := "[" + strings.Join(desiredStatesStr, ", ") + "]"

	lastPrint := time.Now()
	for {
		select {
		case <-ctx.Done():
			m.Fatalf("timed out waiting to reach any state of %v", desiredStatesDescr)
		default:
			// check if we are in the desired state
			currentState := m.m.sm.MustState()
			for _, s := range desiredStates {
				if currentState == s {
					return
				}
			}

			if time.Since(lastPrint) > 1*time.Second {
				m.Infof("waiting to reach any state of %v\n", desiredStatesDescr)
				lastPrint = time.Now()
			}

			time.Sleep(50 * time.Millisecond)
		}
	}
}
