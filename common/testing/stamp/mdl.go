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
	"hash/fnv"
	"reflect"
)

type (
	Model[T modelWrapper] struct {
		*internalModel
	}
	internalModel struct { // model representation without type annotations
		id       modelID
		domainID ID
		env      *ModelEnv
		typeOf   modelType
		evalCtx  *EvalContext
		recorder modelRecorder
		genCache map[any]any
	}
	modelType    reflect.Type
	modelWrapper interface { // API for the user-defined struct
		GetID() string
		Record(...any)

		str() string
		getID() modelID
		setParent(modelWrapper)
		getScope() modelWrapper
		getType() modelType
		getEvalCtx() *EvalContext
		getModel() *internalModel // read-only!
		setModel(*internalModel)
		getDomainID() ID
		setDomainID(id ID)
	}
	Scope[T modelWrapper] struct {
		mw T
	}
	ID string
)

func newModel(env *ModelEnv, mdlType modelType, parent modelWrapper) modelWrapper {
	mw := reflect.New(mdlType.Elem()).Interface().(modelWrapper)
	mw.setModel(
		&internalModel{
			env:     env,
			id:      env.nextID(),
			typeOf:  mdlType,
			evalCtx: &EvalContext{},
		})
	mw.setParent(parent)
	return mw
}

// Note: not `String` to avoid concurrency issues when being printed by testing goroutine
func (m *internalModel) str() string {
	return boldStr(fmt.Sprintf("[%s(%s)]", m.typeOf, m.getDomainID()))
}

func (m *Model[T]) setModel(mdl *internalModel) {
	m.internalModel = mdl
}

func (m *internalModel) Seed() int {
	h := fnv.New32a()
	h.Write([]byte(fmt.Sprintf("%s_%d", m.typeOf.String(), m.env.Seed())))
	return int(h.Sum32())
}

func (m *internalModel) getEnv() *ModelEnv {
	return m.env
}

func (m *internalModel) getType() modelType {
	return m.typeOf
}

func (m *internalModel) getDomainID() ID {
	return m.domainID
}

func (m *internalModel) setDomainID(id ID) {
	m.domainID = id
}

// GetID returns the domain ID of the model.
func (m *internalModel) GetID() string {
	return string(m.getDomainID())
}

func (m *internalModel) getID() modelID {
	if m.id == 0 {
		m.getEnv().Fatal(fmt.Sprintf("%q is missing an ID", m.typeOf))
	}
	return m.id
}

func (m *internalModel) getModel() *internalModel {
	return m
}

func (m *internalModel) getEvalCtx() *EvalContext {
	return m.evalCtx
}

func (m *internalModel) setModel(_ *internalModel) {
	panic("not implemented")
}

func (m *internalModel) is_model() {
	panic("not implemented")
}

func (m *internalModel) Record(records ...any) {
	m.recorder.Add(records...)
}

func (id ID) Gen() Gen[ID] {
	return GenName[ID]()
}

// TODO: GetScope[T] function that finds the requested type in the parent chain

func (s Scope[T]) getScope() modelWrapper {
	return s.mw
}

func (s Scope[T]) GetScope() T {
	return s.mw
}

func (s *Scope[T]) setParent(mw modelWrapper) {
	s.mw = mw.(T)
}
