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
	"testing"

	"github.com/stretchr/testify/require"
)

type foo struct {
	Num int
}

type bar struct {
	Foo foo
}

func TestPropValidate(t *testing.T) {
	t.Parallel()

	t.Run("passed examples", func(t *testing.T) {
		r := PropExpr[bool](
			nil,
			`len(records) == 1`,
			WithExample(true, "one"),
			WithExample(false, "one", "two"))
		require.NoError(t, r.Validate())
	})

	t.Run("failed example", func(t *testing.T) {
		r := PropExpr[bool](
			nil,
			`len(records) == 0`,
			WithExample(true, "one"))
		require.ErrorContains(t, r.Validate(),
			`example #1 failed: expected 'true', got 'false'`)
	})

	t.Run("mixed passed/failed example", func(t *testing.T) {
		r := PropExpr[bool](
			nil,
			`len(records) == 0`,
			WithExample(true),
			WithExample(true, "not-empty"))
		require.ErrorContains(t, r.Validate(),
			`example #2 failed: expected 'true', got 'false'`)
	})
}

func TestPropEval(t *testing.T) {
	t.Parallel()

	t.Run("passing prop", func(t *testing.T) {
		r := PropExpr[bool](nil, `records | len() == 2`)
		res, err := r.eval(&EvalContext{
			Records: newRecorder(
				bar{},
				bar{},
			)})
		require.Equal(t, true, res)
		require.NoError(t, err)
	})

	t.Run("passing prop across mixed structs", func(t *testing.T) {
		r := PropExpr[bool](nil, `records | filter(.Num == 123) | len() == 1`)
		res, err := r.eval(&EvalContext{
			Records: newRecorder(
				foo{Num: 123},
				foo{Num: 999},
				bar{}, // does not have a field `Num`! without `safeFieldAccessPatcher` this would fail.
			)})
		require.Equal(t, true, res)
		require.NoError(t, err)
	})

	t.Run("failing prop", func(t *testing.T) {
		r := PropExpr[bool](nil, `records | len() == 2`)
		res, err := r.eval(&EvalContext{Records: newRecorder()})
		require.NoError(t, err)
		require.Equal(t, res, false)
	})

	t.Run("invalid syntax", func(t *testing.T) {
		res, err := PropExpr[bool](nil, `BOOM`).eval(nil)
		require.ErrorContains(t, err, "expected bool, but got unknown")
		require.Nil(t, res)
	})

	t.Run("non-boolean return", func(t *testing.T) {
		res, err := PropExpr[bool](nil, `log | filter(.Num == 123)`).eval(nil)
		require.ErrorContains(t, err, "expected bool, but got")
		require.Nil(t, res)
	})
}
