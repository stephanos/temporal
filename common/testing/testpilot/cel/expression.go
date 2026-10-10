// Package cel constructs canonical CEL expressions without runtime or descriptor authority.
// References use native optional value() and hasValue() receiver operations. Path selectors
// remain binding data checked against the Case catalog. The restricted runtime admits native
// logical, comparison, conditional, size and collection-index operators; it rejects direct
// message Select expressions, message constructors, comprehensions and arbitrary calls.
package cel

import (
	"fmt"

	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func Enum(value protoreflect.Enum) *celpb.Value {
	return &celpb.Value{Kind: &celpb.Value_EnumValue{EnumValue: &celpb.EnumValue{Type: string(value.Descriptor().FullName()), Value: int32(value.Number())}}}
}

func Literal(value *celpb.Value) *testpilotspb.Expression {
	if value == nil {
		return &testpilotspb.Expression{}
	}
	var constant *celpb.Constant
	switch v := value.Kind.(type) {
	case *celpb.Value_BoolValue:
		constant = &celpb.Constant{ConstantKind: &celpb.Constant_BoolValue{BoolValue: v.BoolValue}}
	case *celpb.Value_StringValue:
		constant = &celpb.Constant{ConstantKind: &celpb.Constant_StringValue{StringValue: v.StringValue}}
	case *celpb.Value_BytesValue:
		constant = &celpb.Constant{ConstantKind: &celpb.Constant_BytesValue{BytesValue: append([]byte(nil), v.BytesValue...)}}
	case *celpb.Value_Int64Value:
		constant = &celpb.Constant{ConstantKind: &celpb.Constant_Int64Value{Int64Value: v.Int64Value}}
	case *celpb.Value_Uint64Value:
		constant = &celpb.Constant{ConstantKind: &celpb.Constant_Uint64Value{Uint64Value: v.Uint64Value}}
	case *celpb.Value_DoubleValue:
		constant = &celpb.Constant{ConstantKind: &celpb.Constant_DoubleValue{DoubleValue: v.DoubleValue}}
	default:
		return &testpilotspb.Expression{Cel: &celpb.ParsedExpr{Expr: ident("v0")}, Bindings: []*testpilotspb.ExpressionBinding{{Variable: "v0", Input: &testpilotspb.ExpressionBinding_Literal{Literal: proto.CloneOf(value)}}}}
	}
	return &testpilotspb.Expression{Cel: &celpb.ParsedExpr{Expr: &celpb.Expr{Id: 1, ExprKind: &celpb.Expr_ConstExpr{ConstExpr: constant}}}}
}

func Ref(reference *testpilotspb.Reference) *testpilotspb.Expression {
	return member("value", &testpilotspb.Expression{Cel: &celpb.ParsedExpr{Expr: ident("v0")}, Bindings: []*testpilotspb.ExpressionBinding{{Variable: "v0", Input: &testpilotspb.ExpressionBinding_Reference{Reference: proto.CloneOf(reference)}}}})
}

func Path(source *testpilotspb.Expression, path string) *testpilotspb.Expression {
	result := proto.CloneOf(source)
	if result == nil || len(result.Bindings) != 1 || result.GetCel().GetExpr() == nil {
		return &testpilotspb.Expression{}
	}
	node := result.Cel.Expr
	if call := node.GetCallExpr(); call != nil && call.Function == "value" && len(call.Args) == 0 {
		node = call.Target
	}
	if node.GetIdentExpr() == nil {
		return &testpilotspb.Expression{}
	}
	binding := result.Bindings[0]
	if binding.Path != "" && path != "" {
		binding.Path += "."
	}
	binding.Path += path
	if result.Cel.Expr.GetIdentExpr() != nil && binding.Path != "" {
		result = member("value", result)
	}
	return result
}

func Present(source *testpilotspb.Expression) *testpilotspb.Expression {
	if source == nil || source.GetCel().GetExpr() == nil {
		return &testpilotspb.Expression{}
	}
	result := proto.CloneOf(source)
	if receiver := result.Cel.Expr.GetCallExpr(); receiver != nil && receiver.Function == "value" && receiver.Target != nil && len(receiver.Args) == 0 {
		receiver.Function = "hasValue"
		return result
	}
	return member("hasValue", call("optional.of", result))
}
func Size(source *testpilotspb.Expression) *testpilotspb.Expression { return call("size", source) }
func Not(source *testpilotspb.Expression) *testpilotspb.Expression  { return call("!_", source) }
func Compare(operator string, left, right *testpilotspb.Expression) *testpilotspb.Expression {
	if native := map[string]string{"==": "_==_", "!=": "_!=_", "<": "_<_", "<=": "_<=_", ">": "_>_", ">=": "_>=_", "in": "@in"}[operator]; native != "" {
		operator = native
	}
	return call(operator, left, right)
}
func All(operands ...*testpilotspb.Expression) *testpilotspb.Expression {
	return logical("_&&_", true, operands)
}
func Any(operands ...*testpilotspb.Expression) *testpilotspb.Expression {
	return logical("_||_", false, operands)
}
func logical(operator string, empty bool, operands []*testpilotspb.Expression) *testpilotspb.Expression {
	if len(operands) == 0 {
		return Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: empty}})
	}
	if len(operands) == 1 {
		return call(operator, operands[0], Literal(&celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: empty}}))
	}
	var tree func([]*testpilotspb.Expression) *testpilotspb.Expression
	tree = func(items []*testpilotspb.Expression) *testpilotspb.Expression {
		if len(items) == 1 {
			return items[0]
		}
		middle := len(items) / 2
		return call(operator, tree(items[:middle]), tree(items[middle:]))
	}
	return tree(operands)
}
func ident(name string) *celpb.Expr {
	return &celpb.Expr{Id: 1, ExprKind: &celpb.Expr_IdentExpr{IdentExpr: &celpb.Expr_Ident{Name: name}}}
}
func member(function string, source *testpilotspb.Expression) *testpilotspb.Expression {
	result := call(function, source)
	if receiver := result.GetCel().GetExpr().GetCallExpr(); receiver != nil {
		receiver.Target, receiver.Args = receiver.Args[0], nil
	}
	return result
}
func call(function string, operands ...*testpilotspb.Expression) *testpilotspb.Expression {
	result := &testpilotspb.Expression{Cel: &celpb.ParsedExpr{Expr: &celpb.Expr{ExprKind: &celpb.Expr_CallExpr{CallExpr: &celpb.Expr_Call{Function: function}}}}}
	for _, operand := range operands {
		copied := proto.CloneOf(operand)
		if copied == nil || copied.GetCel().GetExpr() == nil {
			return &testpilotspb.Expression{}
		}
		renames := map[string]string{}
		for _, binding := range copied.Bindings {
			name := fmt.Sprintf("v%d", len(result.Bindings))
			renames[binding.Variable] = name
			binding.Variable = name
			result.Bindings = append(result.Bindings, binding)
		}
		walk(copied.Cel.Expr, func(node *celpb.Expr) {
			if name := node.GetIdentExpr(); name != nil {
				if replacement, ok := renames[name.Name]; ok {
					name.Name = replacement
				}
			}
		})
		result.Cel.Expr.GetCallExpr().Args = append(result.Cel.Expr.GetCallExpr().Args, copied.Cel.Expr)
	}
	var id int64
	number(result.Cel.Expr, &id)
	return result
}
func number(node *celpb.Expr, next *int64) {
	if node == nil {
		return
	}
	*next++
	node.Id = *next
	if call := node.GetCallExpr(); call != nil {
		number(call.Target, next)
		for _, arg := range call.Args {
			number(arg, next)
		}
	}
	if selection := node.GetSelectExpr(); selection != nil {
		number(selection.Operand, next)
	}
	if list := node.GetListExpr(); list != nil {
		for _, element := range list.Elements {
			number(element, next)
		}
	}
	if mapping := node.GetStructExpr(); mapping != nil {
		for _, entry := range mapping.Entries {
			*next++
			entry.Id = *next
			number(entry.GetMapKey(), next)
			number(entry.Value, next)
		}
	}
}
func walk(node *celpb.Expr, visit func(*celpb.Expr)) {
	if node == nil {
		return
	}
	visit(node)
	if call := node.GetCallExpr(); call != nil {
		walk(call.Target, visit)
		for _, arg := range call.Args {
			walk(arg, visit)
		}
	}
	if selectExpr := node.GetSelectExpr(); selectExpr != nil {
		walk(selectExpr.Operand, visit)
	}
	if list := node.GetListExpr(); list != nil {
		for _, element := range list.Elements {
			walk(element, visit)
		}
	}
	if object := node.GetStructExpr(); object != nil {
		for _, entry := range object.Entries {
			walk(entry.GetMapKey(), visit)
			walk(entry.Value, visit)
		}
	}
}
