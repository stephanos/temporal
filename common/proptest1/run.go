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
	"fmt"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/maps"
	"pgregory.net/rapid"
)

var (
	_ Run = (*run)(nil)
)

type (
	Run interface {
		Logger
		Identity
		ModelParam
		RunContext
		T() *testing.T
		Helper()
		Seed() int
		Finish(string)
		Assert(string, ...any)
		WaitForState(*model, ...*State)
		update(EventInput) (EventOutput, bool)
		getModel(string) any
		newModel(string, EventOutput) any
		getModelByID(string, string) any
	}
	run struct {
		seed       int // TODO: init
		logger     Logger
		testingT   *testing.T // only present when running "example"
		rapidT     *rapid.T   // only present when running "fuzzer"
		modelTypes map[string]ModelType[any]
		models     map[string]map[string]*model // by type by ID
	}
)

func (r *run) set(a any) {
	//TODO implement me
	panic("implement me")
}

func (r *run) Finish(string) {
}

func (r *run) typeOf() string {
	return "<RUN>"
}

func (r *run) ID() ID {
	return "<RUN>"
}

func (r *run) get(_ seeder, s string) (any, bool) {
	if s == r.typeOf() {
		return r, true
	}
	return nil, false
}

func (r *run) T() *testing.T {
	return r.testingT
}

func (r *run) Debug(msg string, args ...any) {
	r.logger.Debug(msg, args...)
}

func (r *run) Info(msg string, args ...any) {
	r.logger.Info(msg, args...)
}

func (r *run) Error(msg string, args ...any) {
	r.logger.Error(msg, args...)
}

func (r *run) Fatal(msg string, args ...any) {
	r.logger.Fatal(msg, args...)
}

func (r *run) Seed() int {
	return r.seed
}

func (r *run) Helper() {
}

func (r *run) FailNow() {
	r.testingT.FailNow()
}

func (r *run) Run() Run {
	return r
}

func (r *run) Assert(name string, args ...any) {
	// TODO ...
}

func (r *run) WaitForState(m *model, desiredStates ...*State) {
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
			m.LogFatal("timed out waiting to reach any state of %v", desiredStatesDescr)
		default:
			// check if we are in the desired state
			currentState := m.CurrentState()
			for _, s := range desiredStates {
				if currentState == s {
					return
				}
			}

			if time.Since(lastPrint) > 1*time.Second {
				m.LogInfo("waiting to reach any state of %v\n", desiredStatesDescr)
				lastPrint = time.Now()
			}

			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (r *run) newModel(typeName string, input EventOutput) any {
	if _, ok := r.modelTypes[typeName]; !ok {
		panic(fmt.Sprintf("model type %q not found", typeName))
	}
	if _, ok := r.models[typeName]; !ok {
		r.models[typeName] = make(map[string]*model)
	}
	mdl := r.modelTypes[typeName].new(r, input)
	r.models[typeName][mdl.ID()] = mdl
	return mdl
}

func (r *run) update(in EventInput) (EventOutput, bool) {
	// TODO: make sure every event input has only one handler
	return nil, false
}

func (r *run) getModelByID(typeName string, id string) any {
	if m, ok := r.models[typeName][id]; ok {
		return m
	}
	panic(fmt.Sprintf("model with type %q and ID %q not found", typeName, id))
}

// TODO: use parent-child relationship to find closest model instance
func (r *run) getModel(typeName string) any {
	if models, ok := r.models[typeName]; ok && len(models) > 0 {
		if len(models) == 1 {
			return maps.Values(models)[0]
		}
		panic(fmt.Sprintf("more than one model with type %q found", typeName))
	}
	panic(fmt.Sprintf("model with type %q not found", typeName))
}

func (r *run) addModelType(t ModelType[any]) {
	if r.models == nil {
		r.models = make(map[string]map[string]*model)
	}
	r.models[t.typeOf()] = make(map[string]*model)
	if r.modelTypes == nil {
		r.modelTypes = make(map[string]ModelType[any])
	}
	r.modelTypes[t.typeOf()] = t
}

func GetByID[T ModelType[T]](r RunContext, id string) Model[T] {
	return r.Run().getModelByID(getTypeName[T](), id).(Model[T])
}

func Get[T ModelType[T]](r RunContext) Model[T] {
	return r.Run().getModel(getTypeName[T]()).(Model[T])
}

func Create[T ModelType[T]](r RunContext, input EventOutput) Model[T] {
	return r.Run().newModel(getTypeName[T](), input).(Model[T])
}

func Update[Out EventOutput, T Model[T], In EventInput](mdl T, input In) (Out, bool) {
	output, ok := mdl.Update(input)
	if !ok {
		var zero Out
		return zero, false
	}
	return output.(Out), true
}

func Signal[T Model[T], In EventInput](r RunContext, input In) {
	// TODO
}
