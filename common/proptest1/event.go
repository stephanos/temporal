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
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, IORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package proptest1

import (
	"fmt"
)

var (
	eventHandlerByType = map[string]map[*State]map[string]eventHandler{} // modelType -> State -> eventInputType -> eventHandler
)

type (
	EventHandler[M Model[M], I EventInput, O EventOutput] struct {
		label       string
		callback    func(any, I) O
		inputTypeOf string
		initsModel  bool
	}
	eventHandler interface {
		update(*model, EventInput) any
	}
	EventInput  any
	EventOutput any
	//eventOpt[M Model[M], I any] struct {
	//	apply func(*EventHandler[M, I])
	//}
)

func newEventHandler[M Model[M], I EventInput, O EventOutput](
	label string,
	states []*State,
	fn func(M, I) O,
) *EventHandler[M, I, O] {
	hdl := &EventHandler[M, I, O]{
		label: label,
		callback: func(m any, input I) O {
			return fn(m.(M), input)
		},
		inputTypeOf: getTypeName[I](),
	}

	//for _, opt := range opts {
	//	opt.apply(hdl)
	//}

	modelTypeOf := getTypeName[M]()
	if _, ok := eventHandlerByType[modelTypeOf]; !ok {
		eventHandlerByType[modelTypeOf] = map[*State]map[string]eventHandler{}
	}
	if len(states) == 0 {
		panic(fmt.Sprintf("event handler must be registered with at least one state"))
	}
	for _, state := range states {
		if _, ok := eventHandlerByType[modelTypeOf][state]; !ok {
			eventHandlerByType[modelTypeOf][state] = map[string]eventHandler{}
		}
		if _, ok := eventHandlerByType[modelTypeOf][state][hdl.inputTypeOf]; ok {
			panic(fmt.Sprintf("event handler already registered for modelType %q, state %q and inputType %q", modelTypeOf, state, hdl.inputTypeOf))
		}
		eventHandlerByType[modelTypeOf][state][hdl.inputTypeOf] = hdl
	}
	return hdl
}

func NewInitHandler[M Model[M], I EventInput](
	label string,
	fn func(M, I) ID,
) *EventHandler[M, I, ID] {
	return newEventHandler(label, []*State{initState}, fn)
}

func NewSignalHandler[M Model[M], I EventInput](
	label string,
	states []*State,
	fn func(M, I),
) *EventHandler[M, I, any] {
	return newEventHandler(label, states, func(m M, input I) any {
		fn(m, input)
		return nil
	})
}

func NewQueryHandler[M Model[M], O EventOutput](
	label string,
	states []*State,
	fn func(M) O,
) *EventHandler[M, any, O] {
	return newEventHandler(label, states, func(m M, _ any) O {
		return fn(m)
	})
}

func NewUpdateHandler[M Model[M], I EventInput, O EventOutput](
	label string,
	states []*State,
	fn func(M, I) O,
) *EventHandler[M, I, O] {
	return newEventHandler(label, states, func(m M, input I) O {
		return fn(m, input)
	})
}

func (e *EventHandler[M, I, O]) update(mdl *model, input EventInput) any {
	return e.callback(mdl, input.(I))
}

//func ModelInit[M modelType]() *eventOpt[M] {
//	return &eventOpt[M]{
//		apply: func(evt *EventHandler[M]) {
//			evt.initsModel = true
//		},
//	}
//}

//func Callback[M modelType, I any](callback func(M, I)) *eventOpt[M] {
//	return &eventOpt[M]{
//		apply: func(evt *EventHandler[M]) {
//			evt.callbacks[getTypeName[I]()] = func(m M, input any) {
//				callback(m, input.(I))
//			}
//		},
//	}
//}

func (e *EventHandler[M, I, O]) String() string {
	return e.label
}
