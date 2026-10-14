package ir

import (
	"cmp"
	"fmt"
	"slices"

	celpb "cel.dev/expr"
	"google.golang.org/protobuf/proto"
)

// CanonicalCEL snapshots the stored CEL AST, assigning IDs from one in preorder.
// Calls visit target then arguments; selects visit operand; lists retain order;
// literal maps visit sorted entries, each entry then key then value. SourceInfo is
// diagnostic for evaluation, but remains in exact artifact bytes, with IDs remapped.
// This is identity canonicalization, not environment or type admission.
func CanonicalCEL(source *celpb.ParsedExpr) (*celpb.ParsedExpr, error) {
	if err := CheckSurface(source, DefaultLimits()); err != nil {
		return nil, err
	}
	if len(source.GetSourceInfo().GetMacroCalls()) != 0 || len(source.GetSourceInfo().GetExtensions()) != 0 {
		return nil, Invalid(Unsupported, "source_info", "CEL macros and extensions are unsupported")
	}
	result := proto.CloneOf(source)
	ids := map[int64]int64{}
	var next int64
	assign := func(id *int64) error {
		if *id <= 0 || ids[*id] != 0 {
			return Invalid(Malformed, "expr.id", "CEL IDs must be positive and unique")
		}
		next++
		ids[*id] = next
		*id = next
		return nil
	}
	var walk func(*celpb.Expr) error
	walk = func(node *celpb.Expr) error {
		if node == nil {
			return Invalid(Malformed, "expr", "CEL expression is required")
		}
		if err := assign(&node.Id); err != nil {
			return err
		}
		var children []*celpb.Expr
		switch kind := node.ExprKind.(type) {
		case *celpb.Expr_ConstExpr:
			if kind == nil || !celScalar(kind.ConstExpr) {
				return Invalid(Unsupported, "expr.const_expr", "CEL scalar constant is required")
			}
		case *celpb.Expr_IdentExpr:
			if kind == nil || kind.IdentExpr == nil {
				return Invalid(Malformed, "expr.ident_expr", "CEL identifier is required")
			}
		case *celpb.Expr_SelectExpr:
			children = append(children, node.GetSelectExpr().GetOperand())
		case *celpb.Expr_CallExpr:
			call := node.GetCallExpr()
			if call == nil {
				return Invalid(Malformed, "expr.call_expr", "CEL call is required")
			}
			if call.Target != nil {
				children = append(children, call.Target)
			}
			children = append(children, call.Args...)
		case *celpb.Expr_ListExpr:
			list := node.GetListExpr()
			if list == nil || len(list.OptionalIndices) != 0 {
				return Invalid(Unsupported, "expr.list_expr", "CEL optional list entries are unsupported")
			}
			children = append(children, list.Elements...)
		case *celpb.Expr_StructExpr:
			literal := node.GetStructExpr()
			if err := canonicalCELMap(literal); err != nil {
				return err
			}
			for _, entry := range literal.Entries {
				if err := assign(&entry.Id); err != nil {
					return err
				}
				if err := walk(entry.GetMapKey()); err != nil {
					return err
				}
				if err := walk(entry.Value); err != nil {
					return err
				}
			}
		default:
			return Invalid(Unsupported, "expr", "unsupported CEL identity node")
		}
		for _, child := range children {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(result.Expr); err != nil {
		return nil, err
	}
	if info := result.SourceInfo; info != nil {
		positions := make(map[int64]int32, len(info.Positions))
		for original, position := range info.Positions {
			id, ok := ids[original]
			if !ok {
				return nil, Invalid(Malformed, "source_info.positions", "CEL source position names an unknown ID")
			}
			positions[id] = position
		}
		info.Positions = positions
	}
	return result, nil
}

func celScalar(value *celpb.Constant) bool {
	if value == nil || IsNil(value.ConstantKind) {
		return false
	}
	switch value.ConstantKind.(type) {
	case *celpb.Constant_BoolValue, *celpb.Constant_Int64Value, *celpb.Constant_Uint64Value,
		*celpb.Constant_DoubleValue, *celpb.Constant_StringValue, *celpb.Constant_BytesValue:
		return true
	default:
		return false
	}
}

func canonicalCELMap(literal *celpb.Expr_CreateStruct) error {
	if literal == nil || literal.MessageName != "" {
		return Invalid(Unsupported, "expr.struct_expr", "CEL message construction is unsupported")
	}
	var keyType string
	for _, entry := range literal.Entries {
		if entry == nil || entry.OptionalEntry || !celScalar(entry.GetValue().GetConstExpr()) {
			return Invalid(Unsupported, "expr.struct_expr", "canonical CEL maps require scalar literal values")
		}
		key := entry.GetMapKey().GetConstExpr()
		if key == nil || IsNil(key.ConstantKind) {
			return Invalid(Unsupported, "expr.struct_expr", "canonical CEL maps require scalar literal keys")
		}
		switch key.ConstantKind.(type) {
		case *celpb.Constant_BoolValue, *celpb.Constant_Int64Value, *celpb.Constant_Uint64Value, *celpb.Constant_StringValue:
		default:
			return Invalid(Unsupported, "expr.struct_expr", "unsupported CEL map key type")
		}
		current := fmt.Sprintf("%T", key.ConstantKind)
		if keyType != "" && current != keyType {
			return Invalid(Unsupported, "expr.struct_expr", "canonical CEL maps require one key type")
		}
		keyType = current
	}
	slices.SortFunc(literal.Entries, func(a, b *celpb.Expr_CreateStruct_Entry) int {
		return compareCELKeys(a.GetMapKey().GetConstExpr(), b.GetMapKey().GetConstExpr())
	})
	for i := 1; i < len(literal.Entries); i++ {
		if compareCELKeys(literal.Entries[i-1].GetMapKey().GetConstExpr(), literal.Entries[i].GetMapKey().GetConstExpr()) == 0 {
			return Invalid(Malformed, "expr.struct_expr", "duplicate CEL map key")
		}
	}
	return nil
}

func compareCELKeys(a, b *celpb.Constant) int {
	switch a.ConstantKind.(type) {
	case *celpb.Constant_BoolValue:
		if a.GetBoolValue() == b.GetBoolValue() {
			return 0
		}
		if !a.GetBoolValue() {
			return -1
		}
		return 1
	case *celpb.Constant_Int64Value:
		return cmp.Compare(a.GetInt64Value(), b.GetInt64Value())
	case *celpb.Constant_Uint64Value:
		return cmp.Compare(a.GetUint64Value(), b.GetUint64Value())
	default:
		return cmp.Compare(a.GetStringValue(), b.GetStringValue())
	}
}
