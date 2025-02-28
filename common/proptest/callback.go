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

	. "go.temporal.io/server/common/proptest/internal"
)

const (
	observerCallback callbackType = iota
	variableCallback              = iota
	identityCallback              = iota
)

var (
	idType = reflect.TypeOf(ID(""))
)

type (
	callback struct {
		method   reflect.Method
		inTypes  []reflect.Type
		outTypes []reflect.Type
	}
	callbackType int
	evalContext  struct {
		mw modelWrapper
	}
)

func newCallback(
	modelType reflect.Type,
	method reflect.Method,
) *callback {
	if method.Type.NumIn() == 0 {
		panic(fmt.Sprintf("method %q on model %q must have at least one argument", method.Name, modelType))
	}

	knownTypes := map[reflect.Type]struct{}{}
	var inTypes []reflect.Type
	for j := 1; j < method.Type.NumIn(); j++ {
		inType := method.Type.In(j)
		if _, ok := knownTypes[inType]; ok {
			panic(fmt.Sprintf("method %q on model %q has duplicate argument type %q", method.Name, modelType, inType))
		}
		inTypes = append(inTypes, inType)
		knownTypes[inType] = struct{}{}
	}

	var outTypes []reflect.Type
	for j := 0; j < method.Type.NumOut(); j++ {
		outType := method.Type.Out(j)
		outTypes = append(outTypes, outType)
	}

	return &callback{
		method:   method,
		inTypes:  inTypes,
		outTypes: outTypes,
	}
}

func invokeIdentityCallback(
	mw modelWrapper,
	args map[reflect.Type]reflect.Value,
) {
	mdl := mw.getModel()
	env := mdl.env

	idCallbacks := env.callbackIdx[mw.getType()][identityCallback]
	cb, cbArgs := findBestMatch(idCallbacks, args)
	if cb == nil {
		return
	}
	invokeAndApplyVars(mw, cb, cbArgs)
}

func invokeObserverCallback(
	mw modelWrapper,
	args map[reflect.Type]reflect.Value,
) {
	mdl := mw.getModel()
	env := mdl.env

	callbacks := env.callbackIdx[mw.getType()][observerCallback]
	cb, cbArgs := findBestMatch(callbacks, args)
	if cb == nil {
		return
	}
	invokeAndApplyVars(mw, cb, cbArgs)
}

func invokeVariableCallback(mw modelWrapper) {
	mdl := mw.getModel()
	env := mdl.env

	for _, cb := range env.callbackIdx[mw.getType()][variableCallback] {
		var args []reflect.Value
		for _, inType := range cb.inTypes {
			v := getVarByType(mdl, inType)
			args = append(args, reflect.ValueOf(v.CurrentOrDefault()))
		}
		invokeAndApplyVars(mw, cb, args)
	}
}

func findBestMatch(
	callbacks []*callback,
	args map[reflect.Type]reflect.Value,
) (*callback, []reflect.Value) {
	matches := map[*callback][]reflect.Value{}
	for _, cb := range callbacks {
		var methodArgs []reflect.Value
		for _, inType := range cb.inTypes {
			if _, ok := args[inType]; ok {
				// exact match
				methodArgs = append(methodArgs, args[inType])
			} else if inType.Kind() == reflect.Interface {
				// interface match
				for _, arg := range args {
					if arg.Type().Implements(inType) {
						methodArgs = append(methodArgs, arg)
						break
					}
				}
			}
		}
		if len(methodArgs) == len(cb.inTypes) {
			matches[cb] = methodArgs
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}

	var bestMatch *callback
	for cb, numArgs := range matches {
		if bestMatch == nil || len(numArgs) > len(matches[bestMatch]) {
			bestMatch = cb
		}
	}
	return bestMatch, matches[bestMatch]
}

func invokeAndApplyVars(
	mw modelWrapper,
	cb *callback,
	cbArgs []reflect.Value,
) {
	mdl := mw.getModel()

	res := cb.method.Func.Call(append(
		[]reflect.Value{reflect.ValueOf(mw)},
		cbArgs...,
	))
	for i, outType := range cb.outTypes {
		val := res[i].Interface()
		if outType == idType && (val == Unknown || val == Ignored) {
			// ignore empty IDs
			continue
		}
		setVarByType(mdl, outType, val)
	}
}

func (e *evalContext) Resolve(varType VarType) *Variable {
	return getVarByType(e.mw.getModel(), varType)
}
