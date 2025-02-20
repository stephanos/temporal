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

type (
	varTag[T any] interface{} // for compile-type safety only
)

func MustGet[T any](mw modelWrapper) T {
	mdl := mw.getModel()
	v, ok := get[T](mdl)
	if !ok {
		mdl.getEnv().Fatal(fmt.Sprintf("%v has no var %v", mdl.str(), reflect.TypeFor[T]()))
	}
	return v
}

func get[T any](m *internalModel) (T, bool) {
	var zero T
	val, ok := getVar[T](m).Get()
	if !ok {
		return zero, false
	}
	return val.(T), true
}

func getVar[T any](m *internalModel) *Variable {
	return getVarByType(m, reflect.TypeFor[T]())
}

func getVarByType(m *internalModel, vType VarType) *Variable {
	env := m.getEnv()
	if _, ok := env.varIdx[m.getID()]; !ok {
		return &Variable{TypeOf: vType}
	}
	v, ok := env.varIdx[m.getID()][vType]
	if !ok {
		return &Variable{TypeOf: vType}
	}
	return v
}

func set[T any](m *internalModel, val T) {
	setVarByType(m, reflect.TypeFor[T](), val)
}

func setVarByType(m *internalModel, vType reflect.Type, val any) {
	env := m.getEnv()
	mdlID := m.getID()
	if _, ok := env.varIdx[mdlID]; !ok {
		env.varIdx[mdlID] = make(map[VarType]*Variable)
	}
	v, ok := env.varIdx[mdlID][vType]
	if !ok {
		v = &Variable{TypeOf: vType}
		env.varIdx[mdlID][vType] = v
	}
	v.Set(val, env.currentTick)
}
