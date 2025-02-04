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
)

type (
	Output[T any] struct {
		val T
	}
	output interface {
		valueAccessor
	}
)

func NewOutput[T any]() *Output[T] {
	return &Output[T]{}
}

func (o *Output[T]) Get(m outputAccessor) T {
	typeOf := getTypeOf[T]()
	if i, ok := m.outputs()[typeOf]; ok {
		return i.get().(T)
	}
	panic(fmt.Sprintf("output %q not found on model %q", typeOf, m))
}

func (o *Output[T]) get() any {
	return o.val
}

func (o *Output[T]) Set(m outputAccessor, v T) {
	typeOf := getTypeOf[T]()
	if o, ok := m.outputs()[typeOf]; ok {
		o.set(v)
		return
	}
	panic(fmt.Sprintf("output %q not found on model %q", typeOf, m))
}

func (o *Output[T]) set(v any) {
	o.val = v.(T)
}

func (o *Output[T]) add(model *model) {
	// NOTE: must copy the output to give each model its own instance
	model.outputMap[getTypeOf[T]()] = o.clone()
}

func (o *Output[T]) clone() *Output[T] {
	return &Output[T]{
		val: o.val,
	}
}
