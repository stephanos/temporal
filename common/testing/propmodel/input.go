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
	"fmt"

	"go.temporal.io/server/common/testing/propcheck"
)

type (
	Input[T any] struct {
		val        T
		optional   bool
		genDefault *propcheck.Gen[T]
	}
	inputOpt[T any] func(*Input[T])
	input           interface {
		valueAccessor
		isOptional() bool
		getDefault(propcheck.Run) (any, bool)
	}
)

func NewInput[T any](opts ...inputOpt[T]) *Input[T] {
	inp := &Input[T]{}
	for _, opt := range opts {
		opt(inp)
	}
	return inp
}

func Default[T any](gen *propcheck.Gen[T]) inputOpt[T] {
	return func(i *Input[T]) {
		i.genDefault = gen
	}
}

func Optional[T any]() inputOpt[T] {
	return func(i *Input[T]) {
		i.optional = true
	}
}

func (i *Input[T]) Get(m inputAccessor) T {
	typeOf := getTypeOf[T]()
	if i, ok := m.inputs()[typeOf]; ok {
		return i.get().(T)
	}
	// TODO: search other models
	panic(fmt.Sprintf("input %q not found on model %q", typeOf, m))
}

func (i *Input[T]) get() any {
	return i.val
}

func (i *Input[T]) add(model *model) {
	// NOTE: must copy the input to give each model its own instance
	model.inputMap[getTypeOf[T]()] = i.clone()
}

func (i *Input[T]) clone() *Input[T] {
	return &Input[T]{
		val:        i.val,
		optional:   i.optional,
		genDefault: i.genDefault,
	}
}

func (i *Input[T]) getDefault(run propcheck.Run) (any, bool) {
	if i.genDefault == nil {
		return nil, false
	}
	return i.genDefault.Next(run), true
}

func (i *Input[T]) set(v any) {
	i.val = v.(T)
}

func (i *Input[T]) isOptional() bool {
	return i.optional
}
