// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
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

package stamp

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type handlerExample struct{}

func (*handlerExample) IntMethod(int) string {
	return "IntMethod"
}

func (*handlerExample) IntStringMethod(int, string) string {
	return "IntStringMethod"
}

func (*handlerExample) StringerMethod(fmt.Stringer) string {
	return "StringerMethod"
}

func TestModelHandler(t *testing.T) {
	t.Parallel()

	s := newHandlerSet()
	inst := &handlerExample{}
	ty := reflect.TypeOf(inst)
	for i := 0; i < ty.NumMethod(); i++ {
		method := ty.Method(i)
		s.add(newHandler(ty, method))
	}

	t.Run("Exact match", func(t *testing.T) {
		t.Run("single argument", func(t *testing.T) {
			require.Equal(t, []any{"IntMethod"},
				toVal(s.invokeBestMatch(inst, NewAction(5), false)))
		})

		t.Run("two arguments", func(t *testing.T) {
			require.Equal(t, []any{"IntStringMethod"},
				toVal(s.invokeBestMatch(inst, NewAction(5, "z"), false)))
		})

		t.Run("no match", func(t *testing.T) {
			require.Equal(t, []any{},
				toVal(s.invokeBestMatch(inst, NewAction(time.Now()), false)))
		})
	})

	t.Run("Fuzzy match", func(t *testing.T) {
		t.Run("single argument", func(t *testing.T) {
			require.Equal(t, []any{"StringerMethod"},
				toVal(s.invokeBestMatch(inst, NewAction(time.Now()), true)))
		})
	})
}

func toVal(vals []reflect.Value) []any {
	out := make([]any, len(vals))
	for i, v := range vals {
		out[i] = v.Interface()
	}
	return out
}
