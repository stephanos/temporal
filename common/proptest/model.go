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

package proptest

import (
	"fmt"
	"reflect"
)

var (
	Ignored ID = "<<ignored>>"
	Unknown ID = ""
)

type (
	Model[T any] struct {
		*internalModel
	}
	internalModel struct { // internal model representation without type annotations
		id     modelID
		env    *Env
		typeOf wrapperType
		sm     *StateMachine
	}
	wrapperType  = reflect.Type
	modelWrapper interface { // API for the user-defined struct
		str() string
		GetID() string
		getID() modelID
		getScope() scope
		getType() wrapperType
		getModel() *internalModel // read-only!
		setModel(*internalModel)
		getDomainID() ID
	}
	Scope[T any] struct {
		modelWrapper
	}
	scope struct {
		id     ID
		typeOf wrapperType
	}
	ID string
)

func newModel(
	env *Env,
	mdlType reflect.Type,
) modelWrapper {
	mw := reflect.New(mdlType).Interface().(modelWrapper)
	mw.setModel(
		&internalModel{
			env:    env,
			id:     env.nextID(),
			typeOf: mdlType,
		})
	return mw
}

// Note: not `String` to avoid concurrency issues when being printed by testing goroutine
func (m *internalModel) str() string {
	return boldStr(fmt.Sprintf("[%s(%s)]", m.typeOf.Name(), m.getDomainID()))
}

func (m *Model[T]) setModel(mdl *internalModel) {
	m.internalModel = mdl
}

func (m *internalModel) getScope() scope {
	return scope{id: m.getDomainID(), typeOf: m.getType()}
}

func (m *internalModel) getEnv() *Env {
	return m.env
}

func (m *internalModel) getType() wrapperType {
	return m.typeOf
}

func (m *internalModel) getDomainID() ID {
	v := getVar[ID](m)
	if v == nil {
		return Unknown
	}
	current := v.Current()
	if current == nil {
		return Unknown
	}
	return current.(ID)
}

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

func (m *internalModel) setModel(_ *internalModel) {
	panic("not implemented")
}
