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

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	root     ModelParent = Root{}
	specType             = reflect.TypeOf((*Spec)(nil)).Elem()
)

type (
	Env struct {
		varTypes                 map[reflect.Type]struct{}
		modelTypes               map[reflect.Type]struct{}
		modelsByID               map[absoluteID]model
		modelsByParentID         map[absoluteID][]model
		propsByModelAndName      map[reflect.Type]map[string]reflect.Method
		observersByModelAndInput map[reflect.Type]map[reflect.Type]reflect.Method
	}

	Model[T any, P ModelParent] struct {
		Var[T]
		model
	}
	model struct {
		typeOf reflect.Type
	}

	ModelParent interface {
		absoluteID() absoluteID
	}
	Root struct{}

	Var[T any]    interface{}
	SetVar[T any] interface {
		Var[T]
		Descriptor() protoreflect.EnumDescriptor
	}
	variable struct {
	}

	Spec     interface{}
	Prop     interface{}
	PropEval interface{}

	EventHandler[T any] interface{}
	Gen[T Var[T]]       interface{}

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
)

func NewEnv() *Env {
	return &Env{
		varTypes:                 make(map[reflect.Type]struct{}),
		modelTypes:               make(map[reflect.Type]struct{}),
		modelsByID:               make(map[absoluteID]model),
		modelsByParentID:         make(map[absoluteID][]model),
		propsByModelAndName:      make(map[reflect.Type]map[string]reflect.Method),
		observersByModelAndInput: make(map[reflect.Type]map[reflect.Type]reflect.Method),
	}
}

func (e *Env) Send(payload any) {
	payloadVal := reflect.ValueOf(payload)

	// send to all observers of all models to find new models
	var newModels []model
	for mdlType := range e.modelTypes {
		if e.observe(mdlType, payloadVal) {
			newModels = append(newModels, model{typeOf: mdlType})
		}
	}

	// dedupe
	// TODO

	// propagate to all models
	e.propagate(payloadVal, root)
}

func (e *Env) propagate(payload reflect.Value, parent ModelParent) {
	parentID := parent.absoluteID()
	for _, child := range e.modelsByParentID[parentID] {
		e.observe(child.typeOf, payload)
	}
}

func (e *Env) observe(mdl reflect.Type, payload reflect.Value) bool {
	observers := e.observersByModelAndInput[mdl]
	for _, observer := range observers {
		if observer.Type.In(1) == payload.Type() {
			mdl := reflect.New(mdl).Elem()
			matched := observer.Func.Call([]reflect.Value{
				mdl,
				payload,
			})[0].Bool()
			if matched {
				return true // TODO: safe? requires observers to be ordered?
			}
		}
	}
	return false
}

func (m model) absoluteID() absoluteID {
	return "" // TODO
}

func (r Root) absoluteID() absoluteID {
	return "/"
}

func RegisterModel[M Model[T, P], T any, P ModelParent](env *Env, mdl M) {
	modelType := reflect.TypeFor[T]()
	if _, ok := env.modelTypes[modelType]; ok {
		panic(fmt.Sprintf("model %q already registered", modelType))
	}

	// TODO: better method recognition (ie use types, too)
	for i := 0; i < modelType.NumMethod(); i++ {
		method := modelType.Method(i)

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
		if method.Type.Out(0) == specType {
			// TODO
			continue
		}

		panic(fmt.Sprintf("unhandled method %q on model %q", method.Name, modelType))
	}
	env.modelTypes[modelType] = struct{}{}
}
