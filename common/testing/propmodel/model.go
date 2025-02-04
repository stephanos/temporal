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
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/qmuntal/stateless"
	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/propcheck"
)

type (
	model struct {
		*Model
		label     string
		r         propcheck.Run
		sm        *stateless.StateMachine
		inputMap  map[reflect.Type]input
		outputMap map[reflect.Type]output
		debugInfo map[string]any
	}
	Model struct {
		*require.Assertions
		m *model // reference to the actual model
	}
	ModelProvider interface {
		model() *model
	}
	ModelInput interface {
		add(*model)
	}
	inputAccessor interface {
		inputs() map[reflect.Type]input
	}
	outputAccessor interface {
		outputs() map[reflect.Type]output
	}
	valueAccessor interface {
		get() any
		set(any)
	}
)

func NewModel(
	label string,
	transitions []Transition,
	inputs []ModelInput,
	params []any,
) *Model {
	m := &model{
		label:     label,
		inputMap:  map[reflect.Type]input{},
		outputMap: map[reflect.Type]output{},
		debugInfo: map[string]any{},
	}
	m.Model = &Model{m: m}

	for _, inp := range inputs {
		inp.add(m)
	}

	if len(transitions) > 0 {
		var initState *State
		for _, t := range transitions {
			from := t.From()
			if from.init {
				if initState != nil && initState != from {
					panic(fmt.Sprintf("multiple initial states found in model %q", label))
				}
				initState = from
			}
		}
		if initState == nil {
			panic(fmt.Sprintf("initial state not found in model %q", label))
		}

		sm := stateless.NewStateMachine(initState)
		sm.OnTransitioning(func(_ context.Context, t stateless.Transition) {
			m.Infof("transition to %q\n", t.Destination)
		})

		eventLabels := map[string]struct{}{}
		for _, t := range transitions {
			if _, ok := eventLabels[t.Event()]; ok {
				panic(fmt.Sprintf("duplicate event %q found in model %q", t.Event(), label))
			}
			sm.Configure(t.From()).Permit(t.Event(), t.To())
			sm.Configure(t.To()).OnEntryFrom(t.Event(), func(ctx context.Context, args ...any) error {
				t.Invoke(m.Model, args[0])
				return nil
			})
		}
		m.sm = sm
	}

	reqInputs := map[reflect.Type]input{}
	for typeOf, inp := range m.inputMap {
		reqInputs[typeOf] = inp
	}
	ackInput := func(typeOf reflect.Type) {
		if _, ok := reqInputs[typeOf]; !ok {
			panic(fmt.Sprintf("unknown parameter %q provided in model %q", typeOf, label))
		}
		delete(reqInputs, typeOf)
	}

	providedParams := map[reflect.Type]any{}
	for _, param := range params {
		providedParams[reflect.TypeOf(param)] = param
	}
	ackParam := func(typeOf reflect.Type) {
		delete(providedParams, typeOf)
	}

	// resolve Run input
	for typeOf := range maps.Keys(providedParams) {
		switch t := providedParams[typeOf].(type) {
		case *propcheck.FakeRun:
			m.setRun(t)
			ackParam(typeOf)
			break
		case propcheck.Run:
			m.setRun(t)
			ackParam(typeOf)
			break
		case ModelProvider:
			m.setRun(t.model().r)
			break
		}
	}
	if m.r == nil {
		panic(fmt.Sprintf("Run has not been provided to model %q", label))
	}

	// resolve inputs
	for paramTypeOf := range maps.Keys(providedParams) {
		param := providedParams[paramTypeOf]

		// if it's a generator, invoke it
		if gen, ok := param.(propcheck.Generator); ok {
			param = gen.Next(m.r)
			ackParam(paramTypeOf) // need to remove generator type now
			paramTypeOf = reflect.TypeOf(param)
		}

		for inputTypeOf, inp := range reqInputs {
			if paramTypeOf.AssignableTo(inputTypeOf) {
				inp.set(param)
				ackInput(inputTypeOf)
				ackParam(paramTypeOf)
			}
		}
	}
	if len(providedParams) > 0 {
		panic(fmt.Sprintf("unknown parameters in model %q: %s",
			label, slices.Collect(maps.Keys(providedParams))))
	}

	// assign default values
	for typeOf, inp := range reqInputs {
		if defaultVal, ok := inp.getDefault(m.r); ok {
			inp.set(defaultVal)
			ackInput(typeOf)
		}
	}
	if len(reqInputs) > 0 {
		panic(fmt.Sprintf("missing required parameters in model %q: %s",
			label, slices.Collect(maps.Keys(reqInputs))))
	}

	// TODO: or lookup cached model for same ID

	return m.Model
}

func (m *Model) String() string {
	var args []string
	if s := m.m.sm.MustState(); s != nil {
		args = append(args, fmt.Sprintf("state=%v", s))
	}
	for k, v := range m.m.debugInfo {
		args = append(args, fmt.Sprintf("%v=%v", k, v))
	}
	return fmt.Sprintf("%s{%v}", m.m.label, strings.Join(args, ", "))
}

func (m *Model) inputs() map[reflect.Type]input {
	return m.m.inputMap
}

func (m *Model) outputs() map[reflect.Type]output {
	return m.m.outputMap
}

func (m *Model) model() *model {
	return m.m
}

func (m *Model) Run() propcheck.Run {
	return m.m.r
}

func (m *Model) Infof(msg string, args ...any) {
	m.Run().Logf("%v %v", m, fmt.Sprintf(msg, args...))
}

func (m *Model) Errorf(msg string, args ...any) {
	m.Run().Logf("🔴 %v %v", m, fmt.Sprintf(msg, args...))
}

func (m *Model) Fatalf(msg string, args ...any) {
	m.Run().Fatalf("💥 %v %v", m, fmt.Sprintf(msg, args...))
}

func (m *model) setRun(r propcheck.Run) {
	m.r = r
	m.Assertions = require.New(r)
}

func (m *model) model() *model {
	return m
}
