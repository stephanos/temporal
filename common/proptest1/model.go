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

package proptest1

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/qmuntal/stateless"
	"github.com/stretchr/testify/require"
)

var _ Model[ModelType[any]] = &model{}

type (
	modelType struct {
		run       Run
		id        string
		label     string
		inputMap  map[string]input
		outputMap map[string]output
	}
	ModelType[T any] interface {
		ModelParam

		new(Run, EventInput) *model
		inputs() map[ID]input
		outputs() map[ID]output
	}
	Model[M ModelType[M]] interface {
		RunContext
		ModelType[M]
		require.TestingT

		CurrentState() *State
		LogInfo(string, ...any)
		LogError(string, ...any)
		LogFatal(string, ...any)
		Update(EventInput) (EventOutput, bool)
		Signal(EventInput)
	}
	model struct {
		*modelType
		run Run
		sm  *stateless.StateMachine
	}
	Handle[M ModelType[M]] interface {
	}
	ModelVar interface {
		ModelParam
		add(modelType)
		isOptional() bool
	}
	Readonly[M Model[M]] interface {
		Model[M] // TODO: don't allow Set()
	}
	readonly[M ModelType[M]] struct {
		*modelType
	}
	RunContext interface {
		Run() Run
	}
	ModelParam interface {
		TypeOf
		set(any)
		get(seeder, string) (any, bool)
	}
	valueAccessor interface {
		set(any)
	}
	generator interface {
		Next(Run) any
		Variants() []any
	}
)

func (m *model) NewState(state *State) {

}

func NewModelType[T any]() ModelType[T] {
	m := &modelType{
		label: getTypeName[T](),
	}

	//if len(transitions) > 0 {
	//	eventLabels := map[string]struct{}{}
	//	for _, t := range transitions {
	//		if _, ok := eventLabels[t.EventHandler()]; ok {
	//			panic(fmt.Sprintf("duplicate event %q found in modelType %q", t.EventHandler(), label))
	//		}
	//		m.sm.Configure(t.From()).Permit(t.EventHandler(), t.To())
	//		m.sm.Configure(t.To()).OnEntryFrom(t.EventHandler(), func(ctx context.Context, args ...any) error {
	//			t.Invoke(m, args[0])
	//			return nil
	//		})
	//	}
	//}

	//paramsByType := map[ID]ModelParam{}
	//for _, param := range params {
	//	typeOf := param.typeOf()
	//	if typeOf == "" {
	//		panic(fmt.Sprintf("typeOf is empty for %T", param))
	//	}
	//	switch t := param.(type) {
	//	case Run:
	//		m.setRun(t)
	//		continue
	//	case ModelType:
	//		m.setRun(t.Run())
	//	}
	//	paramsByType[typeOf] = param
	//}
	//if m.r == nil {
	//	panic(fmt.Sprintf("Run has not been provided to modelType %q", label))
	//}
	//
	//// resolve input values
	//var missingInputs []string
	//for _, inp := range inputs {
	//	typeOf := inp.typeOf()
	//
	//	// try to find matching parameter
	//	if param, ok := paramsByType[typeOf]; ok {
	//		if val, ok := param.get(m.r, typeOf); ok {
	//			inp.set(val)
	//			inp.add(m)
	//			delete(paramsByType, typeOf)
	//			continue
	//		}
	//	}
	//
	//	// try default value
	//	// TODO: move into param.get
	//	if val, ok := inp.get(m.r, typeOf); ok {
	//		inp.set(val)
	//		inp.add(m)
	//		delete(paramsByType, typeOf)
	//		continue
	//	}
	//
	//	// skip if optional
	//	if inp.isOptional() {
	//		delete(paramsByType, typeOf)
	//		inp.add(m)
	//		continue
	//	}
	//
	//	missingInputs = append(missingInputs, typeOf)
	//}
	//if len(missingInputs) > 0 || len(paramsByType) > 0 {
	//	panic(fmt.Sprintf("incomplete modelType %q:\n undefined inputs: %s\n unknown parameters: %s",
	//		label, missingInputs, maps.Keys(paramsByType)))
	//}

	// TODO: or lookup cached modelType for same ID

	return m
}

func (m *modelType) new(run Run, input EventInput) *model {
	mdl := &model{
		modelType: &modelType{
			label: m.label,
		},
		run: run,
	}

	mdl.sm = stateless.NewStateMachine(initState)
	mdl.sm.OnTransitioning(func(_ context.Context, t stateless.Transition) {
		mdl.LogInfo("transition to %q\n", t.Destination)
	})

	id, ok := mdl.Update(input)
	if !ok {
		panic(fmt.Sprintf("model %q failed to initialise", m.label))
	}
	mdl.id = id.(string)

	return mdl
}

func (m *model) SetID(id string) {
	m.id = id
}

func (m *model) ID() string {
	return m.id
}

func (m *modelType) Get() string {
	return m.id
}

func (m *model) String() string {
	var args []string
	if s := m.sm.MustState(); s != nil {
		args = append(args, fmt.Sprintf("state=%v", s))
	}
	return fmt.Sprintf("%s{%v}", m.label, strings.Join(args, ", "))
}

func (m *modelType) inputs() map[ID]input {
	return m.inputMap
}

func (m *modelType) outputs() map[ID]output {
	return m.outputMap
}

func (m *model) Run() Run {
	return m.run
}

func (m *modelType) add(_ modelType) {
	panic("implement me")
}

func (m *modelType) set(v any) {
	panic("implement me")
}

func (m *model) CurrentState() *State {
	return m.sm.MustState().(*State)
}

func ToInput[M ModelType[M]](opts ...inputOpt[M]) *modelInput[M] {
	return &modelInput[M]{
		Input: NewVar[M](getTypeName[M](), opts...),
	}
}

func (m *model) Update(input EventInput) (EventOutput, bool) {
	inputTypeOf := typeToStr(input, reflect.TypeOf(input))
	if _, ok := eventHandlerByType[m.typeOf()]; !ok {
		m.LogFatal("no event handlers found for model %q", m)
		return nil, false
	}
	if _, ok := eventHandlerByType[m.typeOf()][m.CurrentState()]; !ok {
		m.LogFatal("no event handlers found for model %q with state %q", m, m.CurrentState())
		return nil, false
	}
	if _, ok := eventHandlerByType[m.typeOf()][m.CurrentState()][inputTypeOf]; !ok {
		m.LogFatal("no event handler found for model %q with state %q and input %q", m, m.CurrentState(), inputTypeOf)
		return nil, false
	}
	evt, _ := eventHandlerByType[m.typeOf()][m.CurrentState()][inputTypeOf]
	return evt.update(m, input), true
}

func (m *model) Signal(input EventInput) {
	panic("implement me")
}

func (m *modelType) get(_ seeder, typeOf string) (any, bool) {
	//if m.typeOf() == typeOf {
	//	return m.val, true
	//}

	// search inputs
	if inp, ok := m.inputs()[typeOf]; ok {
		v, ok := inp.get(m.run, typeOf)
		if !ok {
			panic(fmt.Sprintf("input %q in modelType %q is not initialised", inp, m))
		}
		return v, true
	}

	// search outputs
	if outp, ok := m.outputs()[typeOf]; ok {
		v, ok := outp.get(m.run, typeOf)
		if ok {
			return v, true
		}
	}

	// search recursively
	for _, inp := range m.inputs() {
		if v, ok := inp.get(m.run, typeOf); ok {
			return v, true
		}
	}

	return nil, false
}

func (m *modelType) typeOf() string {
	return m.label
}

func (m *model) LogInfo(format string, args ...any) {
	m.Run().Info("%v %v", m, fmt.Sprintf(format, args...))
}

func (m *model) LogError(format string, args ...any) {
	m.Run().Error("%v %v", m, fmt.Sprintf(format, args...))
}

func (m *model) LogFatal(format string, args ...any) {
	m.Run().Fatal("%v %v", m, fmt.Sprintf(format, args...))
}

func (m *model) Errorf(format string, args ...any) {
	m.LogError(format, args...)
}

func (m *model) FailNow() {
	m.Run().T().FailNow()
}

func (m *model) setRun(r *run) {
	m.run = r
}
