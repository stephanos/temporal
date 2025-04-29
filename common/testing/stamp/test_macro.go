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

package stamp

import (
	"fmt"
	"testing"

	"go.temporal.io/server/common/log"
)

type (
	ScenarioMacro struct {
		t          *testing.T
		loggerFunc func(testing.TB) log.Logger
		fn         func(*MacroScenario)
		opts       []ScenarioOption
		genCache   map[string]any
	}
	MacroScenario struct {
		*Scenario
		parent         *ScenarioMacro
		explorationRun bool
	}
	LabeledFunction struct {
		Label string
		Fn    func()
	}
)

func newScenarioMacro(
	t *testing.T,
	loggerFunc func(testing.TB) log.Logger,
	fn func(*MacroScenario),
	opts ...ScenarioOption,
) *ScenarioMacro {
	return &ScenarioMacro{
		t:          t,
		loggerFunc: loggerFunc,
		fn:         fn,
		opts:       opts,
		genCache:   map[string]any{},
	}
}

func (sm *ScenarioMacro) newScenario(t *testing.T) *MacroScenario {
	s := newScenario(t, sm.loggerFunc(t), sm.opts...)
	return &MacroScenario{
		Scenario: s,
		parent:   sm,
	}
}

// TODO: check there is no duplicate scenario run
func (sm *ScenarioMacro) Run() {
	sm.t.Logf("starting exploration run")

	type genInfo struct {
		choices int
		index   int
	}
	genIdx := map[string]genInfo{}

	// exploration run to collect all choice generators
	var failed bool
	sm.t.Run("exploration run", func(t *testing.T) {
		s := sm.newScenario(t)
		s.explorationRun = true

		s.genCtx.pickChoiceFn = func(id string, choices int) int {
			sm.t.Logf("found choice generator: %s", id)
			if choices <= 0 {
				panic("no choices")
			}
			genIdx[id] = genInfo{
				choices: choices,
				index:   len(genIdx),
			}
			return 0 // always pick the first choice on the first run
		}

		sm.fn(s)
		failed = t.Failed()
	})
	if failed {
		sm.t.Fatalf("exploration run failed")
	}

	// calculate the total number of scenarios
	total := 1
	for _, info := range genIdx {
		total *= info.choices
	}
	sm.t.Logf("found %d scenarios across %d generators", total, len(genIdx))

	// run all other scenarios
	for i := 0; i < total; i++ {
		sm.t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			s := sm.newScenario(t)

			s.genCtx.pickChoiceFn = func(id string, choices int) int {
				info, exists := genIdx[id]
				if !exists {
					panic(fmt.Sprintf("found unknown generator after exploration: %s", id))
				}

				divisor := 1
				for otherId, otherInfo := range genIdx {
					if otherInfo.index > info.index {
						if otherInfo.choices <= 0 {
							panic(fmt.Sprintf("generator %s has invalid variant count: %d", otherId, otherInfo.choices))
						}
						divisor *= otherInfo.choices
					}
				}

				pick := (i / divisor) % info.choices
				return pick
			}

			sm.fn(s)
		})
	}
}

func (sm *ScenarioMacro) VerifyOnce() {
	// TODO
}

// TODO: panic if same name is used twice witin same macro scenario
func (ms *MacroScenario) Maybe(name string, fn func()) {
	ms.t.Helper()

	name = fmt.Sprintf("Maybe(%s)", name)
	gen := GenChoice(name, true, false)
	choice := resolveChoice(ms, gen, name)
	if choice {
		fn()
	}
}

func (ms *MacroScenario) Switch(name string, choices ...LabeledFunction) {
	ms.t.Helper()

	name = fmt.Sprintf("Switch(%s)", name)
	gen := GenChoice(name, choices...)
	choice := resolveChoice(ms, gen, name)
	choice.Fn()
}

func resolveChoice[T any](ms *MacroScenario, gen Gen[T], name string) T {
	if cached, ok := ms.parent.genCache[name]; ok {
		return cached.(T)
	}
	res := gen.Next(ms)
	ms.parent.genCache[name] = res
	return res
}
