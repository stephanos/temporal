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
	"time"
)

type (
	Actor[M ModelType[M]] interface {
		modelType
		Do(Action[M])
	}
	ActorDef[M ModelType[M]] struct {
		modelType
		label string
	}
	actionOpt struct {
		apply func(*actionSettings)
	}
	ActionTemplate[M ModelType[M], I any, O any] struct {
		*actionSettings
		label string
		fn    func(I) O
	}
	actionSettings struct {
		async bool
	}
	Action[M ModelType[M]] struct {
		Template *ActionTemplate[M, any, any]
		Model    modelType
	}
	Handler struct {
		label string
	}
)

func NewAction[M Model[M], I any, O any](
	label string,
	inputs []ModelVar,
	fn func(I) O,
	opts ...*actionOpt,
) func(args ...any) O {
	t := &ActionTemplate[M, I, O]{
		label:          label,
		actionSettings: &actionSettings{},
		fn:             fn,
	}

	for _, opt := range opts {
		opt.apply(t.actionSettings)
	}

	return func(args ...any) O {
		// TODO
		panic("unimplemented")
	}
}

func NewInitAction[M Model[M]](
	label string,
	inputs []ModelVar,
	fn func(Readonly[M]) ID,
	opts ...*actionOpt,
) func(args ...any) Handle[M] {
	return NewAction[M, Readonly[M], Handle[M]](
		label,
		inputs,
		func(m Readonly[M]) Handle[M] {
			panic("unimplemented")
		},
		opts...)
}

func RequireState(states ...*State) *actionOpt {
	return &actionOpt{
		apply: func(settings *actionSettings) {
			// TODO
		},
	}
}

func WaitForState(timeout time.Duration) *actionOpt {
	return &actionOpt{
		apply: func(settings *actionSettings) {
			// TODO
		},
	}
}

//func NewSignalHandler[MT any, A Actor[M], M ModelDef[MT]](label string, handler func(A, MT)) *Handler {
//	return &Handler{
//		label: label,
//	}
//}

//func (t *ActionTemplate[M, T]) Do() M {
//	// TODO
//	//action := Action[M]{
//	//	Template: &ActionTemplate[M, any]{
//	//		label: t.label,
//	//		fn: func(m any) any {
//	//			return t.fn(m)
//	//		},
//	//	},
//	//}
//	//// TODO
//	panic("unimplemented")
//}

//func (t *ActionTemplate[M, T]) Payload(m M) T {
//	return t.fn(m)
//}
//
//func (t *ActionTemplate[M, T]) String() string {
//	return t.label
//}
