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
	Input[T any] struct { // TODO: split into InputTemplate and Input
		run         Run
		label       string
		val         T
		optional    bool
		generator   *Generator[T]
		initialised bool
	}
	modelInput[M ModelType[M]] struct {
		*Input[M]
	}
	inputOpt[T any] func(*Input[T])
	input           interface {
		ModelParam
		valueAccessor
		isOptional() bool
		isInitialised() bool
	}
)

func NewVar[T any](
	label string,
	opts ...inputOpt[T],
) *Input[T] {
	inp := &Input[T]{label: label}
	for _, opt := range opts {
		opt(inp)
	}
	return inp
}

func Default[T any](gen *Generator[T]) inputOpt[T] {
	return func(i *Input[T]) {
		i.generator = gen
	}
}

func Optional[T any]() inputOpt[T] {
	return func(i *Input[T]) {
		i.optional = true
	}
}

func (i *Input[T]) Val(v T) *Input[T] {
	return i.Gen(Just(v))
}

func (i *Input[T]) String() string {
	return fmt.Sprintf("Input[%v]", i.label)
}

func (i *Input[T]) Gen(gen *Generator[T]) *Input[T] {
	clone := i.clone()
	clone.generator = gen
	return clone
}

func (i *Input[T]) Get(m ModelType[any]) T {
	//if val, ok := m.get(m.Run(), i.label); ok {
	//	return val.(T)
	//}
	panic(fmt.Sprintf("input %q not found on modelType %q", i, m))
}

func (i *Input[T]) ID() ID {
	return i.label
}

func (i *Input[T]) get(s seeder, typeOf string) (any, bool) {
	if typeOf != i.typeOf() {
		return nil, false
	}

	if !i.initialised {
		if i.generator == nil {
			return nil, false
		}
		i.set(i.generator.Next(s))
	}

	return i.val, true
}

func (i *Input[T]) typeOf() string {
	return i.label
}

func (i *Input[T]) add(model modelType) {
	// NOTE: must copy the input to give each modelType its own instance
	model.inputs()[i.ID()] = i.clone()
}

func (i *Input[T]) clone() *Input[T] {
	return &Input[T]{
		label:       i.label,
		val:         i.val,
		optional:    i.optional,
		generator:   i.generator,
		initialised: i.initialised,
	}
}

func (i *Input[T]) isInitialised() bool {
	return i.initialised
}

func (i *Input[T]) set(v any) {
	if v == nil {
		panic(fmt.Sprintf("cannot set nil value on input %q", i))
	}
	i.val = v.(T)
	i.initialised = true
}

func (i *Input[T]) isOptional() bool {
	return i.optional
}

func (i *Input[T]) With(opts ...inputOpt[T]) ModelVar {
	clone := i.clone()
	for _, opt := range opts {
		opt(clone)
	}
	return clone
}

func (i *Input[T]) Set(m ModelType[any], val T) {
	m.set(val)
}

func (i *modelInput[M]) set(v any) {
	if v == nil {
		panic(fmt.Sprintf("cannot set nil value on modelType input %q", i))
	}
	i.Input.val = v.(M)
}

func (i *modelInput[M]) get(s seeder, typeOf string) (any, bool) {
	model := i.Input.val
	return model.get(s, typeOf)
}

func (i *modelInput[M]) add(model modelType) {
	model.inputs()[i.ID()] = &modelInput[M]{ // cloning
		Input: i.Input.clone(),
	}
}
