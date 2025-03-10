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

package rule

import (
	"cmp"
	"errors"
	"fmt"

	"github.com/expr-lang/expr"
)

var (
	RuleViolated = errors.New("violated")
)

type (
	EvalContext struct {
		Log Log
	}
	Log  []any
	Rule struct {
		name            string
		expr            string
		examples        []*EvalContext
		counterExamples []*EvalContext
	}
	ruleErr struct {
		rule Rule
		err  error
	}
	ruleOption func(*Rule)
)

func WithExample(log ...any) ruleOption {
	return func(rule *Rule) {
		rule.examples = append(rule.examples, &EvalContext{Log: log})
	}
}

func WithCounterExample(log ...any) ruleOption {
	return func(rule *Rule) {
		rule.counterExamples = append(rule.counterExamples, &EvalContext{Log: log})
	}
}

// See https://expr-lang.org/docs/language-definition
func NewRule(
	expr string,
	opts ...ruleOption,
) *Rule {
	rule := &Rule{expr: expr}
	for _, opt := range opts {
		opt(rule)
	}
	return rule
}

func (r *Rule) SetName(name string) {
	r.name = name
}

func (r Rule) String() string {
	return cmp.Or(r.name, r.expr)
}

func (r Rule) Validate() error {
	for i, e := range r.examples {
		if err := r.Eval(e); err != nil {
			return fmt.Errorf("example #%d failed: %w", i+1, err)
		}
	}
	for i, ce := range r.counterExamples {
		if err := r.Eval(ce); err == nil {
			return fmt.Errorf("counter-example #%d passed", i+1)
		}
	}
	return nil
}

func (r Rule) Eval(evalCtx *EvalContext) error {
	// compile
	program, err := expr.Compile(
		r.expr,
		expr.AsBool(),
		expr.WarnOnAny(),
		expr.Patch(safeFieldAccessPatcher{}),
	)
	if err != nil {
		return &ruleErr{rule: r, err: err}
	}

	// evaluate
	input := map[string]any{"log": evalCtx.Log}
	output, err := expr.Run(program, input)
	if err != nil {
		return &ruleErr{rule: r, err: err}
	}

	// assess
	if !output.(bool) { // already checked that result is bool
		return &ruleErr{rule: r, err: RuleViolated}
	}
	return nil
}

func (l *Log) Add(entry any) *Log {
	*l = append(*l, entry)
	return l
}

func (e ruleErr) Error() string {
	return fmt.Errorf("rule %q %w", e.rule, e.err).Error()
}
