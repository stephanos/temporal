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

package rule_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/proptest/rule"
)

type foo struct {
	Num int
}

type bar struct {
	Foo foo
}

func TestRuleValidate(t *testing.T) {
	t.Parallel()

	t.Run("passed example", func(t *testing.T) {
		r := rule.NewRule(
			`len(log) == 1`,
			rule.WithExample(
				"one",
			))
		require.NoError(t, r.Validate())
	})

	t.Run("failed example", func(t *testing.T) {
		r := rule.NewRule(
			`len(log) == 0`,
			rule.WithExample(
				"one",
			))
		require.ErrorContains(t, r.Validate(),
			`example #1 failed: rule "len(log) == 0" violated`)
	})

	t.Run("mixed passed/failed example", func(t *testing.T) {
		r := rule.NewRule(
			`len(log) == 0`,
			rule.WithExample(),            // pass
			rule.WithExample("not-empty")) // fail
		require.ErrorContains(t, r.Validate(),
			`example #2 failed: rule "len(log) == 0" violated`)
	})

	t.Run("passed counter-example", func(t *testing.T) {
		r := rule.NewRule(
			`len(log) == 1`,
			rule.WithCounterExample())
		require.NoError(t, r.Validate())
	})

	t.Run("failed counter-example", func(t *testing.T) {
		r := rule.NewRule(
			`len(log) == 0`,
			rule.WithCounterExample())
		require.ErrorContains(t, r.Validate(), "counter-example #1 passed")
	})

	t.Run("mixed passed/failed counter-example", func(t *testing.T) {
		r := rule.NewRule(
			`len(log) == 0`,
			rule.WithCounterExample("not-empty"), // pass
			rule.WithCounterExample())            // fail
		require.ErrorContains(t, r.Validate(), "counter-example #2 passed")
	})
}

func TestRuleEval(t *testing.T) {
	t.Parallel()

	t.Run("passing rule", func(t *testing.T) {
		r := rule.NewRule(`log | len() == 2`)
		err := r.Eval(&rule.EvalContext{Log: []any{
			bar{},
			bar{},
		}})
		require.NoError(t, err)
	})

	t.Run("passing rule across mixed structs", func(t *testing.T) {
		r := rule.NewRule(`log | filter(.Num == 123) | len() == 1`)
		err := r.Eval(&rule.EvalContext{Log: []any{
			foo{Num: 123},
			foo{Num: 999},
			bar{}, // does not have a field `Num`! without `safeFieldAccessPatcher` this would fail.
		}})
		require.NoError(t, err)
	})

	t.Run("failing rule", func(t *testing.T) {
		r := rule.NewRule(`log | len() == 2`)
		err := r.Eval(&rule.EvalContext{Log: []any{}})
		require.ErrorContains(t, err, rule.RuleViolated.Error())
	})

	t.Run("invalid syntax", func(t *testing.T) {
		err := rule.NewRule(`BOOM`).Eval(nil)
		require.ErrorContains(t, err, "expected bool, but got unknown")
	})

	t.Run("non-boolean return", func(t *testing.T) {
		err := rule.NewRule(`log | filter(.Num == 123)`).Eval(nil)
		require.ErrorContains(t, err, "expected bool, but got")
	})
}
