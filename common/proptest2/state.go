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
// THE SOFTMARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package proptest

import (
	"fmt"
	"strings"
)

var (
	InitState    State = "<init>" // sentinel state that represents the initial state
	AnyState     State = "<any>"  // sentinel state that matches any state
	StateDraft   State = "draft"
	StateAlive   State = "alive"
	StateDead    State = "dead"
	StateInvalid       = StateDead.SubState("invalid")
)

var (
	TriggerCommitted   Trigger = "committed"
	TriggerInvalidated Trigger = "invalidated"
)

type (
	State                 string
	Trigger               string
	stateConfig[M entity] struct {
		modelOption[M]
		from        State
		transitions []transition
	}
	transition struct {
		to      State
		trigger Trigger
	}
)

func (s State) String() string {
	return underlineStr(string(s))
}

func (s State) SubState(sub State) State {
	return State(fmt.Sprintf("%s:%s", s, sub))
}

func (s State) IsTerminal() bool {
	return strings.HasPrefix(string(s), string(StateDead))
}

func WithState[M entity](from State) *stateConfig[M] {
	return &stateConfig[M]{from: from}
}

func (t *stateConfig[M]) Allow(to State, trigger Trigger) *stateConfig[M] {
	t.transitions = append(t.transitions, transition{to: to, trigger: trigger})
	return t
}

func (t *stateConfig[M]) On(fn any) *stateConfig[M] {
	return t
}

func (t *stateConfig[M]) apply(entity M) {
	if t.from.IsTerminal() {
		entity.getEnv().Fatal(fmt.Sprintf("cannot add transition from terminal state %q", t.from))
	}
	for _, set := range t.transitions {
		entity.sm().Configure(t.from).Permit(set.trigger, set.to)
	}
}

//m.sm().Configure(to).OnEntryFrom(trigger, func(ctx context.Context, args ...any) error {
//	if len(args) != 1 {
//		return fmt.Errorf("expected 1 argument, got %d", len(args))
//	}
//	tr, ok := args[0].(T)
//	if !ok {
//		return fmt.Errorf(fmt.Sprintf("cannot cast %v to %T", args[0], tr))
//	}
//	return callback(m)
//})
