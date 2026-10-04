package lint

import (
	"strconv"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// short is the name an author wrote for a lifted name: what follows its last `.` or `$`, so
// `temporal.standaloneactivity.Protocol$.terminal` reads `terminal`.
func short(name string) string {
	if i := strings.LastIndexAny(name, ".$"); i >= 0 && i < len(name)-1 {
		return name[i+1:]
	}
	return name
}

// spell writes an expression back in the Scala it was lifted from, closely enough for a reader of a
// guard: names by their short names, operators infix, a nested operation in parentheses.
func spell(x *umpirespb.Expr) string {
	switch k := x.GetKind().(type) {
	case *umpirespb.Expr_Literal:
		return spellValue(k.Literal)
	case *umpirespb.Expr_Var:
		return k.Var
	case *umpirespb.Expr_Field:
		return operand(k.Field.GetBase()) + "." + k.Field.GetField()
	case *umpirespb.Expr_Call:
		return short(k.Call.GetFunction()) + "(" + spellAll(k.Call.GetArgs()) + ")"
	case *umpirespb.Expr_Construct:
		name := short(k.Construct.GetType())
		if k.Construct.GetCase() != "" {
			name = k.Construct.GetCase()
		}
		if len(k.Construct.GetArgs()) == 0 {
			return name
		}
		return name + "(" + spellAll(k.Construct.GetArgs()) + ")"
	case *umpirespb.Expr_Copy:
		updates := make([]string, len(k.Copy.GetUpdates()))
		for i, u := range k.Copy.GetUpdates() {
			updates[i] = u.GetName() + " = " + spell(u.GetValue())
		}
		return operand(k.Copy.GetBase()) + ".copy(" + strings.Join(updates, ", ") + ")"
	case *umpirespb.Expr_Unary:
		if k.Unary.GetOp() == umpirespb.Unary_OP_NEG {
			return "-" + operand(k.Unary.GetOperand())
		}
		return "!" + operand(k.Unary.GetOperand())
	case *umpirespb.Expr_Binary:
		left, right := operand(k.Binary.GetLeft()), operand(k.Binary.GetRight())
		if k.Binary.GetOp() == umpirespb.Binary_OP_CONTAINS {
			return right + ".contains(" + spell(k.Binary.GetLeft()) + ")"
		}
		return left + " " + binaryOps[k.Binary.GetOp()] + " " + right
	case *umpirespb.Expr_If:
		return "if " + spell(k.If.GetCondition()) + " then " + spell(k.If.GetThen()) + " else " + spell(k.If.GetElse())
	case *umpirespb.Expr_Match:
		return operand(k.Match.GetScrutinee()) + " match"
	case *umpirespb.Expr_Let:
		return spell(k.Let.GetBody())
	case *umpirespb.Expr_List:
		if len(k.List.GetItems()) == 0 {
			return "Nil"
		}
		return "List(" + spellAll(k.List.GetItems()) + ")"
	case *umpirespb.Expr_Hole:
		return "hole(" + k.Hole + ")"
	default:
		return "…"
	}
}

var binaryOps = map[umpirespb.Binary_Op]string{
	umpirespb.Binary_OP_EQ: "==", umpirespb.Binary_OP_NE: "!=", umpirespb.Binary_OP_AND: "&&", umpirespb.Binary_OP_OR: "||",
	umpirespb.Binary_OP_LT: "<", umpirespb.Binary_OP_LE: "<=", umpirespb.Binary_OP_GT: ">", umpirespb.Binary_OP_GE: ">=",
	umpirespb.Binary_OP_ADD: "+", umpirespb.Binary_OP_SUB: "-", umpirespb.Binary_OP_CONCAT: "++",
}

// operand spells an expression where it is an operand: a binary operation in parentheses.
func operand(x *umpirespb.Expr) string {
	if x.GetBinary() != nil || x.GetIf() != nil {
		return "(" + spell(x) + ")"
	}
	return spell(x)
}

func spellAll(xs []*umpirespb.Expr) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = spell(x)
	}
	return strings.Join(parts, ", ")
}

func spellValue(v *umpirespb.Value) string {
	switch k := v.GetKind().(type) {
	case *umpirespb.Value_Bool:
		return strconv.FormatBool(k.Bool)
	case *umpirespb.Value_Int:
		return strconv.FormatInt(k.Int, 10)
	case *umpirespb.Value_Text:
		return strconv.Quote(k.Text)
	case *umpirespb.Value_Enum:
		if len(k.Enum.GetFields()) == 0 {
			return k.Enum.GetCase()
		}
		fields := make([]string, len(k.Enum.GetFields()))
		for i, f := range k.Enum.GetFields() {
			fields[i] = spellValue(f)
		}
		return k.Enum.GetCase() + "(" + strings.Join(fields, ", ") + ")"
	case *umpirespb.Value_Record:
		fields := make([]string, len(k.Record.GetFields()))
		for i, f := range k.Record.GetFields() {
			fields[i] = spellValue(f)
		}
		return short(k.Record.GetType()) + "(" + strings.Join(fields, ", ") + ")"
	case *umpirespb.Value_List:
		items := make([]string, len(k.List.GetItems()))
		for i, f := range k.List.GetItems() {
			items[i] = spellValue(f)
		}
		return "List(" + strings.Join(items, ", ") + ")"
	default:
		return "…"
	}
}

// spellPattern writes a match case's pattern as Scala does.
func spellPattern(p *umpirespb.Pattern) string {
	switch k := p.GetKind().(type) {
	case *umpirespb.Pattern_Wildcard:
		return "_"
	case *umpirespb.Pattern_Bind:
		if k.Bind.GetPattern().GetWildcard() != nil {
			return k.Bind.GetName()
		}
		return k.Bind.GetName() + " @ " + spellPattern(k.Bind.GetPattern())
	case *umpirespb.Pattern_Literal:
		return spellValue(k.Literal)
	case *umpirespb.Pattern_Case:
		if len(k.Case.GetFields()) == 0 {
			return k.Case.GetCase()
		}
		fields := make([]string, len(k.Case.GetFields()))
		for i, f := range k.Case.GetFields() {
			fields[i] = spellPattern(f)
		}
		return k.Case.GetCase() + "(" + strings.Join(fields, ", ") + ")"
	case *umpirespb.Pattern_Alternatives:
		alts := make([]string, len(k.Alternatives.GetPatterns()))
		for i, a := range k.Alternatives.GetPatterns() {
			alts[i] = spellPattern(a)
		}
		return strings.Join(alts, " | ")
	default:
		return "…"
	}
}
