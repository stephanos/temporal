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

package proptest

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/server/common/proptest/expect"
	. "go.temporal.io/server/common/proptest/internal"
)

var (
	All              = expect.All
	Any              = expect.Any
	Either           = expect.Either
	Rule             = expect.NewRule
	scopeTypePattern = regexp.MustCompile(`Scope\[(.*)]`)
)

type (
	Env struct {
		logger
		lock              sync.Mutex
		testingT          *testing.T
		currentTick       int
		currentMdlCounter modelID
		mdlTypeIdx        map[wrapperType]struct{}
		mdlTypeByName     map[string]wrapperType
		mdlChildTypes     map[wrapperType][]wrapperType
		callbackIdx       map[wrapperType]map[callbackType][]*callback
		ruleIdx           map[wrapperType][]expect.Rule
		childrenIdx       map[scope]map[ID]modelWrapper
		varIdx            map[modelID]map[VarType]*Variable
	}
	modelID int // internal identifier only - different from public domain ID
)

func NewEnv(t *testing.T) *Env {
	return &Env{
		testingT:   t,
		logger:     newTestLogger(t),
		mdlTypeIdx: make(map[wrapperType]struct{}),
		mdlTypeByName: map[string]wrapperType{
			rootTypeName: reflect.TypeOf(root),
		},
		mdlChildTypes: make(map[wrapperType][]wrapperType),
		ruleIdx:       make(map[wrapperType][]expect.Rule),
		childrenIdx:   make(map[scope]map[ID]modelWrapper),
		callbackIdx:   make(map[wrapperType]map[callbackType][]*callback),
		varIdx:        make(map[modelID]map[VarType]*Variable),
	}
}

// TODO: type check parent matches model somehow?
func RegisterModel[T any, M Model[T]](
	env *Env,
) {
	mdlType := reflect.TypeFor[T]()
	if mdlType.Kind() != reflect.Struct {
		env.Fatal(fmt.Sprintf("model %q must be a struct", mdlType))
	}
	if _, ok := env.mdlTypeIdx[mdlType]; ok {
		env.Fatal(fmt.Sprintf("model %q already registered", mdlType))
	}
	env.mdlTypeIdx[mdlType] = struct{}{}
	env.callbackIdx[mdlType] = make(map[callbackType][]*callback)
	exampleMdl := newModel(env, mdlType)
	env.mdlTypeByName[qualifiedTypeName(mdlType)] = mdlType

	for i := 0; i < mdlType.NumField(); i++ {
		field := mdlType.Field(i)
		scopeMatch := scopeTypePattern.FindStringSubmatch(field.Type.String())
		if len(scopeMatch) == 2 {
			scopeMdlTypeName := scopeMatch[1]
			scopeMdlType, ok := env.mdlTypeByName[scopeMdlTypeName]
			if !ok {
				env.Fatal(fmt.Sprintf("model %q field %q must be a registered model", mdlType, field.Name))
			}
			env.mdlChildTypes[scopeMdlType] = append(env.mdlChildTypes[scopeMdlType], mdlType)
		}
	}

	// TODO: init SM

	// need pointer type for methods
	ptrModelType := reflect.TypeFor[*T]()
	if ptrModelType.NumMethod() == 0 {
		env.Fatal(fmt.Sprintf("model %q has no methods", mdlType))
	}
	// TODO: better method recognition (ie use types, too)
	for i := 0; i < ptrModelType.NumMethod(); i++ {
		method := ptrModelType.Method(i)
		methodName := strings.ToLower(method.Name)
		switch {
		case strings.HasPrefix(methodName, "id"):
			cb := newCallback(mdlType, method)
			if len(cb.outTypes) == 0 {
				env.Fatal(fmt.Sprintf("method %q on model %q must return at least one value", method.Name, mdlType))
			}
			env.callbackIdx[mdlType][identityCallback] = append(env.callbackIdx[mdlType][identityCallback], cb)
		case strings.HasPrefix(methodName, "get") || strings.HasPrefix(methodName, "is"):
			cb := newCallback(mdlType, method)
			if len(cb.outTypes) == 0 {
				env.Fatal(fmt.Sprintf("method %q on model %q must return at least one value", method.Name, mdlType))
			}
			env.callbackIdx[mdlType][variableCallback] = append(env.callbackIdx[mdlType][variableCallback], cb)
		case strings.HasPrefix(methodName, "on"):
			cb := newCallback(mdlType, method)
			if len(cb.outTypes) != 1 {
				env.Fatal(fmt.Sprintf("method %q on model %q must return at least 1 value", method.Name, mdlType))
			}
			env.callbackIdx[mdlType][observerCallback] = append(env.callbackIdx[mdlType][observerCallback], cb)
		case strings.HasPrefix(methodName, "verify"):
			cb := newCallback(mdlType, method)
			if len(cb.inTypes) != 0 {
				env.Fatal(fmt.Sprintf("method %q on model %q must not have any parameters", method.Name, mdlType))
			}
			if len(cb.outTypes) != 1 {
				env.Fatal(fmt.Sprintf("method %q on model %q must only return `expect.Rule`", method.Name, mdlType))
			}
			env.ruleIdx[mdlType] = append(env.ruleIdx[mdlType],
				cb.method.Func.Call([]reflect.Value{reflect.ValueOf(exampleMdl)})[0].Interface().(expect.Rule))
		case strings.HasPrefix(methodName, "gen"):
			// TODO
		default:
			env.Fatal(fmt.Sprintf("unsupported method %q on model %q", method.Name, mdlType))
		}
	}
}

func Make[T any](e *Env) T {
	panic("not implemented")
}

func (e *Env) Send(payloads ...any) expect.Report {
	e.lock.Lock()
	defer func() {
		e.currentTick += 1
		e.lock.Unlock()
	}()

	payloadVals := map[reflect.Type]reflect.Value{}
	for _, payload := range payloads {
		if payload == nil {
			continue
		}
		t := reflect.TypeOf(payload)
		v := reflect.ValueOf(payload)
		payloadVals[t] = v
	}

	// send payloads to models
	e.walk(func(parent, child modelWrapper) (cont bool) {
		if child == nil {
			// create new children
			for _, mdlType := range e.mdlChildTypes[parent.getType()] {
				newChild := newModel(e, mdlType) // TODO: use tmp ID instead
				cleanup := func() {
					delete(e.varIdx, newChild.getID())
				}

				// find and verify model's identity
				invokeIdentityCallback(newChild, payloadVals)
				id := getVar[ID](newChild.getModel()).CurrentOrDefault().(ID)
				if id == "" {
					cleanup()
					continue
				}
				if _, ok := e.childrenIdx[parent.getScope()][id]; ok {
					cleanup()
					continue
				}

				// keep new child
				e.updateID(newChild, parent, "", id)
				e.Info(fmt.Sprintf("%s created", newChild.str()))

				// invoke observer
				invokeObserverCallback(newChild, payloadVals)
			}

			return true
		} else {
			invokeIdentityCallback(child, payloadVals)
			id := getVar[ID](child.getModel()).CurrentOrDefault().(ID)
			if id == "" {
				return false
			}
			if _, ok := e.childrenIdx[parent.getScope()][id]; !ok {
				return false
			}

			// update identity
			e.updateID(child, parent, id, id)

			// invoke observer
			invokeObserverCallback(child, payloadVals)

			return true
		}
	})

	// update variables
	e.walk(func(_, child modelWrapper) (cont bool) {
		if child != nil {
			invokeVariableCallback(child)
		}
		return true
	})

	// verify spec
	var res expect.Report
	e.walk(func(_, child modelWrapper) (cont bool) {
		if child != nil {
			res.Merge(verifySpecs(child))
		}
		return true
	})
	return res
}

func (e *Env) walk(fn func(modelWrapper, modelWrapper) bool) {
	visited := map[modelID]struct{}{}

	var walkModelsFn func(modelWrapper)
	walkModelsFn = func(mw modelWrapper) {
		if _, ok := visited[mw.getID()]; ok {
			return
		}
		visited[mw.getID()] = struct{}{}

		if fn(mw, nil) {
			for _, child := range e.childrenIdx[mw.getScope()] {
				if cont := fn(mw, child); cont {
					walkModelsFn(child)
				}
			}
		}
	}
	walkModelsFn(root)
}

func (e *Env) updateID(
	mw modelWrapper,
	parent modelWrapper,
	curDomainID ID,
	newDomainID ID,
) {
	mdl := mw.getModel()

	parentScope := parent.getScope()
	if _, ok := e.childrenIdx[parentScope]; !ok {
		e.childrenIdx[parentScope] = make(map[ID]modelWrapper)
	}

	// no updateID needed if domain IDs are the same
	if newDomainID == curDomainID {
		return
	}

	// only allow ID change if previous ID was temporary
	if curDomainID != "" { // && !curDomainID.isTemp()
		e.Fatal(fmt.Sprintf("%s tried to change domain ID from %q to %q",
			mdl.str(), curDomainID, newDomainID))

		if _, ok := e.childrenIdx[parentScope][newDomainID]; ok {
			e.Fatal(fmt.Sprintf("%s already has a child of type %q with domain ID %q",
				parent.str(), mdl.getType(), newDomainID))
		}
	}

	set[ID](mdl, newDomainID)

	// delete old reference, if it exists
	if curDomainID != "" {
		if _, ok := e.childrenIdx[parentScope][curDomainID]; !ok {
			e.Fatal(fmt.Sprintf("%s does not have previous domain ID %q",
				mdl.str(), curDomainID))
		}
		delete(e.childrenIdx[parentScope], curDomainID)

		// migrate its children to new parent
		e.childrenIdx[scope{id: newDomainID, typeOf: mw.getType()}] =
			e.childrenIdx[scope{id: curDomainID, typeOf: mw.getType()}]

		e.Info(fmt.Sprintf("%s new ID (was %q)", mdl.str(), curDomainID))
	}

	e.childrenIdx[parentScope][newDomainID] = mw
}

func (e *Env) nextID() modelID {
	e.currentMdlCounter++
	return e.currentMdlCounter
}

func (e *Env) Seed() int {
	panic("implement me")
}
