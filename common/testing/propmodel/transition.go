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

type (
	Transition interface {
		From() *State
		To() *State
		Event() string
		Invoke(*Model, any)
	}
	transition[T any] struct {
		from  *State
		event *Event[T]
		to    *State
	}
	transitionOpt[T any] func(*transition[T])
)

func NewTransition[T any](
	from *State,
	event *Event[T],
	to *State,
	opts ...transitionOpt[T],
) *transition[T] {
	t := &transition[T]{from: from, event: event, to: to}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t transition[T]) From() *State {
	return t.from
}

func (t transition[T]) To() *State {
	return t.to
}

func (t transition[T]) Event() string {
	return t.event.label
}

func (t transition[T]) Invoke(m *Model, arg any) {
	typedArg, _ := arg.(T)
	t.event.callback(m, typedArg)
}
