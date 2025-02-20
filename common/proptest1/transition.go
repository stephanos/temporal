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

type (
	Transition interface {
		From() *State
		To() *State
		Event() string
		Invoke(any, any)
	}
	transition[M Model[M], I EventInput, O EventOutput] struct {
		from  *State
		event *EventHandler[M, I, O]
		to    *State
	}
	transitionOpt[M Model[M], I EventInput, O EventOutput] func(*transition[M, I, O])
)

func NewTransition[M Model[M], I EventInput, O EventOutput](
	from *State,
	event *EventHandler[M, I, O],
	to *State,
	opts ...transitionOpt[M, I, O],
) Transition {
	t := &transition[M, I, O]{from: from, event: event, to: to}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t transition[M, I, O]) From() *State {
	return t.from
}

func (t transition[M, I, O]) To() *State {
	return t.to
}

func (t transition[M, I, O]) Event() string {
	return t.event.label
}

func (t transition[M, I, O]) Invoke(m any, input any) {
	typedInput := input.(I)
	typedModel := m.(M)
	t.event.callback(typedModel, typedInput)
}
