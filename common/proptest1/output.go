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
)

type (
	Output[T any] struct {
		Identity
		label       string
		val         T
		initialised bool
	}
	output interface {
		ModelParam
		valueAccessor
	}
)

func (o *Output[T]) isOptional() bool {
	return true
}

func NewOutput[T any](label string) *Output[T] {
	return &Output[T]{label: label}
}

func (o *Output[T]) Get(m ModelType[any]) T {
	//if val, ok := m.get(m.Run(), o.label); ok {
	//	return val.(T)
	//}
	panic(fmt.Sprintf("output %q not found on modelType %q", o, m))
}

//func (o *Output[T]) GetOrDefault(m ModelType, defaultVal T) T {
//	if val, ok := m.get(m.Run(), o.label); ok {
//		return val.(T)
//	}
//	return defaultVal
//}

func (o *Output[T]) typeOf() string {
	return o.label
}

func (o *Output[T]) String() string {
	return fmt.Sprintf("Output(%s)", o.label)
}

func (o *Output[T]) get(r seeder, typeOf string) (any, bool) {
	if o.typeOf() != typeOf {
		return nil, false
	}
	return o.val, o.initialised
}

func (o *Output[T]) Set(m ModelType[any], v T) {
	if o, ok := m.outputs()[o.ID()]; ok {
		o.set(v)
		return
	}
	panic(fmt.Sprintf("output %q not found on modelType %q", o.ID(), m))
}

func (o *Output[T]) set(v any) {
	o.val = v.(T)
	o.initialised = true
}

func (o *Output[T]) ID() ID {
	return o.label
}

func (o *Output[T]) add(model modelType) {
	// NOTE: must copy the output to give each modelType its own instance
	model.outputs()[o.ID()] = o.clone()
}

func (o *Output[T]) clone() *Output[T] {
	return &Output[T]{
		val: o.val,
	}
}
