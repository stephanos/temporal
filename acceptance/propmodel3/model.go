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
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	idSeparator       = "/"
	provisionalPrefix = "provisional-"
	initialized       = iota
)

var (
	specType = reflect.TypeOf((*Spec)(nil)).Elem()
)

type (
	Env struct {
		logger
		testingT *testing.T

		varTypes                 map[reflect.Type]struct{}
		modelTypes               map[reflect.Type]struct{}
		modelsByID               map[absoluteID]modelWrapper
		modelsByParentID         map[absoluteID]map[relativeID]modelWrapper
		propsByModelAndName      map[reflect.Type]map[string]reflect.Method
		varsByModelAndType       map[reflect.Type]map[reflect.Type]*variable[any]
		observersByModelAndInput map[reflect.Type]map[reflect.Type]reflect.Method
	}

	Model[T any, P modelWrapper] struct {
		Var[T]
		*model
	}
	model struct { // internal model representation without type annotations
		env    *Env
		typeOf reflect.Type
	}
	modelWrapper interface { // API for the user-defined struct
		getModel() *model
		setModel(*model)
		getID() relativeID
		getParentID() absoluteID
		getEnv() *Env
		isInitialised() bool
	}
	Root struct {
		env *Env
	}

	Var[T any] struct {
		varTag[T]
		*variable[T]
	}
	varTag[T any]               interface{} // for compile-type safety only
	SetVar[T protoreflect.Enum] struct {
		*variable[T]
	}
	variable[T any] struct {
		value T
		// TODO: history
	}

	Spec     interface{}
	Prop     interface{}
	PropEval interface{}

	EventHandler[T any] interface{}
	Gen[V varTag[V]]    interface{}

	// TODO: move out, model-specific
	Request[RQ proto.Message] struct {
		Ctx     context.Context
		Request RQ
	}
	Response[RQ, RS proto.Message] struct {
		Request[RQ]
		Response RS
		Error    error
	}
	relativeID string
	absoluteID string
)

func NewEnv(t *testing.T) *Env {
	return &Env{
		testingT:                 t,
		logger:                   newTestLogger(t),
		varTypes:                 make(map[reflect.Type]struct{}),
		modelTypes:               make(map[reflect.Type]struct{}),
		modelsByID:               make(map[absoluteID]modelWrapper),
		modelsByParentID:         make(map[absoluteID]map[relativeID]modelWrapper),
		propsByModelAndName:      make(map[reflect.Type]map[string]reflect.Method),
		varsByModelAndType:       make(map[reflect.Type]map[reflect.Type]*variable[any]),
		observersByModelAndInput: make(map[reflect.Type]map[reflect.Type]reflect.Method),
	}
}

func (e *Env) Send(payload any) {
	payloadVal := reflect.ValueOf(payload)

	// first, propagate to all models
	e.propagate(payloadVal, Root{e})

	// then, send to all observers of all models to find new models
	var newModels []modelWrapper
	for mdlType := range e.modelTypes {
		mdl := e.newMdl(mdlType)
		for _, observer := range e.observersByModelAndInput[mdlType] {
			if observer.Type.In(1) == payloadVal.Type() {
				if observer.Func.Call([]reflect.Value{mdl, payloadVal})[0].Bool() {
					newModels = append(newModels, mdl.Interface().(modelWrapper))
				}
			}
		}
	}

	// filter out existing models
	for _, mdl := range newModels {
		if !mdl.isInitialised() {
			// never initialised
			continue
		}

		absID := makeAbsoluteID(mdl.getParentID(), mdl.getID())
		if _, ok := e.modelsByID[absID]; ok {
			// already exists
			continue
		}

		e.modelsByID[absID] = mdl
		if _, ok := e.modelsByParentID[mdl.getParentID()]; !ok {
			e.modelsByParentID[mdl.getParentID()] = make(map[relativeID]modelWrapper)
		}
		e.modelsByParentID[mdl.getParentID()][mdl.getID()] = mdl
	}
}

func (e *Env) propagate(payload reflect.Value, mdl modelWrapper) {
	absID := makeAbsoluteID(mdl.getParentID(), mdl.getID())
	for _, child := range e.modelsByParentID[absID] {
		childVal := reflect.ValueOf(child.getModel())
		for _, observer := range e.observersByModelAndInput[child.getModel().typeOf] {
			if observer.Type.In(1) == payload.Type() {
				if observer.Func.Call([]reflect.Value{childVal, payload})[0].Bool() {
					e.propagate(payload, child)
				}
			}
		}
	}
}

func (e *Env) newMdl(mdl reflect.Type) reflect.Value {
	newMdl := reflect.New(mdl)
	newMdl.Interface().(modelWrapper).setModel(
		&model{
			env:    e,
			typeOf: mdl,
		},
	)
	return newMdl
}

func (m *Model[T, P]) setModel(mdl *model) {
	m.model = mdl
}

func (m model) getEnv() *Env {
	return m.env
}

func (m model) getID() relativeID {
	id, ok := Get[ID](m)
	if !ok {
		panic(fmt.Sprintf("model %q is missing an ID", m.typeOf))
	}
	return id
}

func (m model) getModel() *model {
	return &m
}

func (m model) setModel(mdl *model) {
	panic("not supported")
}

// isInitialised returns true if the ID and ParentID are set.
func (m model) isInitialised() bool {
	if _, ok := Get[ParentID](m); !ok {
		return false
	}
	if _, ok := Get[ID](m); !ok {
		return false
	}
	return true
}

func (m model) getParentID() absoluteID {
	// TODO: update, too, when the parent's ID changes
	parentID, ok := Get[ParentID](m)
	if !ok {
		panic(fmt.Sprintf("model %q missing parent ID", m.typeOf))
	}
	return parentID
}

func (r Root) getID() relativeID {
	return "/"
}

func (r Root) getParentID() absoluteID {
	return "/"
}

func (r Root) getEnv() *Env {
	return r.env
}

func (r Root) isInitialised() bool {
	return false
}

func (r Root) setModel(*model) {
	panic("not supported")
}

func (r Root) getModel() *model {
	panic("not supported")
}

func RegisterModel[T any](env *Env) {
	modelType := reflect.TypeFor[T]()
	if modelType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("model %q must be a struct", modelType))
	}
	if _, ok := env.modelTypes[modelType]; ok {
		panic(fmt.Sprintf("model %q already registered", modelType))
	}
	env.modelTypes[modelType] = struct{}{}

	// need pointer type for methods
	ptrModelType := reflect.TypeFor[*T]()
	if ptrModelType.NumMethod() == 0 {
		panic(fmt.Sprintf("model %q has no methods", modelType))
	}
	// TODO: better method recognition (ie use types, too)
	for i := 0; i < ptrModelType.NumMethod(); i++ {
		method := ptrModelType.Method(i)

		// find prop
		if strings.HasPrefix(method.Name, "Is") {
			if _, ok := env.propsByModelAndName[modelType]; !ok {
				env.propsByModelAndName[modelType] = make(map[string]reflect.Method)
			}
			env.propsByModelAndName[modelType][method.Name] = method
			continue
		}

		// find observer
		if strings.HasPrefix(method.Name, "On") {
			if method.Type.NumIn() != 2 {
				panic(fmt.Sprintf("observer %q on model %q must have exactly one input", method.Name, modelType))
			}
			if method.Type.NumOut() != 1 || method.Type.Out(0).Kind() != reflect.Bool {
				panic(fmt.Sprintf("observer %q on model %q must have return type `bool`", method.Name, modelType))
			}
			if _, ok := env.observersByModelAndInput[modelType]; !ok {
				env.observersByModelAndInput[modelType] = make(map[reflect.Type]reflect.Method)
			}
			inputType := method.Type.In(1)
			if _, ok := env.observersByModelAndInput[modelType][inputType]; ok {
				panic(fmt.Sprintf("duplicate observer %q on model %q", method.Name, modelType))
			}
			env.observersByModelAndInput[modelType][inputType] = method
			continue
		}

		// find generators
		if strings.HasPrefix(method.Name, "Gen") {
			// TODO
			continue
		}

		// find specs
		if method.Type.NumOut() > 0 && method.Type.Out(0) == specType {
			// TODO
			continue
		}

		panic(fmt.Sprintf("unsupported method %q on model %q", method.Name, modelType))
	}
}

func Get[V Var[T], T any](m modelWrapper) (T, bool) {
	var zero T
	typeOf := m.getModel().typeOf
	if _, ok := m.getEnv().varsByModelAndType[typeOf]; !ok {
		return zero, false
	}
	varType := reflect.TypeFor[T]()
	v, ok := m.getEnv().varsByModelAndType[typeOf][varType]
	if !ok {
		return zero, false
	}
	return v.getCurrent().(T), true
}

func Set[V Var[T], T any](m modelWrapper, val T) {
	typeOf := m.getModel().typeOf
	if _, ok := m.getEnv().varsByModelAndType[typeOf]; !ok {
		m.getEnv().varsByModelAndType[typeOf] = make(map[reflect.Type]*variable[any])
	}
	varType := reflect.TypeFor[T]()
	v, ok := m.getEnv().varsByModelAndType[typeOf][varType]
	if !ok {
		v = &variable[any]{}
		m.getEnv().varsByModelAndType[typeOf][varType] = v
	}
	v.value = val
}

func (v *variable[T]) getCurrent() T {
	return v.value
}

func (id relativeID) isProvisional() bool {
	return strings.HasPrefix(string(id), provisionalPrefix)
}

//func getByID(
//	parentID absoluteID,
//	id relativeID,
//) (modelWrapper, error) {
//	if parent == nil {
//		return nil, fmt.Errorf("no parent found")
//	}
//	mdl, ok := parent.getEnv().modelsByID[makeAbsoluteID(parent, id)]
//	if !ok {
//		return nil, fmt.Errorf("model with ID %q not found", id)
//	}
//	return mdl, nil
//}

func makeAbsoluteID(
	parentID absoluteID,
	id relativeID,
) absoluteID {
	return parentID + absoluteID(idSeparator+string(id))
}
