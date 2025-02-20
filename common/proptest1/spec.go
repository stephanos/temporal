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
	"reflect"
	"testing"
)

type (
	Spec interface {
		Setup(Run)
		Teardown(Run)
	}
	specSettings struct {
	}
	specOption func(*specSettings)
)

//	func WithTimeout(t time.Duration) specOption {
//		return func(s *specSettings) {
//			s.timeout = t
//		}
//	}

func RunExample[T Spec](t *testing.T, fn func(T), opts ...specOption) {
	var spec T
	specVal := reflect.New(reflect.TypeOf(spec).Elem())
	spec = specVal.Interface().(T)

	r := NewExampleRun(t, nil)
	spec.Setup(r)
	fn(spec)
	r.Finish(t.Name())
	spec.Teardown(r)
}

//func RunCodegen[T *specCtx](t *testing.T, opts ...specOption) {
//	s := newSpecCtx("example", opts...)
//	s.run = NewCodegen(t)
//
//	if setup, ok := s.(SpecSetup); ok {
//		setup.Setup()
//	}
//	//s.fn(env)
//	s.run.Finish(s.label)
//	if setup, ok := s.(SpecTeardown); ok {
//		setup.Teardown()
//	}
//}
