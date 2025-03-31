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
	"cmp"
	"errors"
	"fmt"
	"reflect"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
)

var (
	_            prop = Prop[bool]{}
	propType          = reflect.TypeFor[prop]()
	PropViolated      = errors.New("violated")
)

type (
	EvalContext struct {
		Records modelRecorder
	}
	Prop[T propTypes] struct {
		_        T // have to use the type parameter to enforce compile-time checking
		owner    modelWrapper
		name     string
		typeOf   reflect.Type
		evalExpr string
		examples []example
	}
	prop interface {
		SetName(string)
		Validate() error
		eval(*EvalContext) (any, error)
		String() string
	}
	propTypes interface {
		bool | int
	}
	example struct {
		outcome any
		evalCtx *EvalContext
	}
	propErr struct {
		prop string
		err  error
	}
	propOption[T propTypes] func(*Prop[T])
)

func WithExample[T propTypes](outcome T, records ...any) propOption[T] {
	return func(prop *Prop[T]) {
		prop.examples = append(prop.examples, example{
			outcome: outcome,
			evalCtx: &EvalContext{Records: newRecorder(records...)},
		})
	}
}

// See https://expr-lang.org/docs/language-definition
func PropExpr[T propTypes](
	owner modelWrapper,
	expr string,
	opts ...propOption[T],
) Prop[T] {
	r := Prop[T]{owner: owner, evalExpr: expr, typeOf: reflect.TypeFor[T]()}
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

func (p Prop[T]) SetName(name string) {
	p.name = name
}

func (p Prop[T]) String() string {
	return cmp.Or(p.name, p.evalExpr)
}

func (p Prop[T]) Validate() error {
	for i, e := range p.examples {
		res, err := p.eval(e.evalCtx)
		if err != nil {
			return fmt.Errorf("example #%d failed: %w", i+1, err)
		}
		if res != e.outcome {
			return fmt.Errorf("example #%d failed: expected '%v', got '%v'", i+1, e.outcome, res)
		}
	}
	return nil
}

func (p Prop[T]) Eval() (any, error) {
	return p.eval(p.owner.getEvalCtx())
}

func (p Prop[T]) eval(evalCtx *EvalContext) (any, error) {
	// configure
	var typeCheckOption expr.Option
	switch p.typeOf {
	case reflect.TypeFor[bool]():
		typeCheckOption = expr.AsBool()
	case reflect.TypeFor[int]():
		typeCheckOption = expr.AsInt()
	default:
		return nil, fmt.Errorf("unsupported prop type %q", p.typeOf.String())
	}

	// compile
	program, err := expr.Compile(
		p.evalExpr,
		typeCheckOption,
		expr.WarnOnAny(),
		expr.Patch(safeFieldAccessPatcher{}),
	)
	if err != nil {
		return nil, &propErr{prop: p.String(), err: err}
	}

	// evaluate
	params := map[string]any{"records": evalCtx.Records.Get()}
	output, err := expr.Run(program, params)
	if err != nil {
		return nil, &propErr{prop: p.String(), err: err}
	}

	return output, nil
}

func (e propErr) Error() string {
	return fmt.Errorf("prop %q %w", e.prop, e.err).Error()
}

type safeFieldAccessPatcher struct{}

func (p safeFieldAccessPatcher) Visit(node *ast.Node) {
	memberNode, ok := (*node).(*ast.MemberNode)
	if !ok {
		return
	}
	ast.Patch(node, &ast.MemberNode{
		Node: &ast.ConditionalNode{
			Cond: &ast.BinaryNode{
				Operator: "in",
				Left:     memberNode.Property,
				Right:    memberNode.Node,
			},
			Exp1: memberNode.Node,
			Exp2: &ast.MapNode{},
		},
		Property: memberNode.Property,
		Optional: memberNode.Optional,
		Method:   memberNode.Method,
	})
}
