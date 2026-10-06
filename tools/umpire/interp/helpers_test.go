package interp

import (
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

func admLiteral(at *umpirespb.Expr, v *umpirespb.Value) *umpirespb.Expr {
	return &umpirespb.Expr{Position: at.GetPosition(), Kind: &umpirespb.Expr_Literal{Literal: v}}
}

func admEnum(typ, c string) *umpirespb.Value {
	return &umpirespb.Value{Kind: &umpirespb.Value_Enum{Enum: &umpirespb.EnumValue{Type: typ, Case: c}}}
}

func function(m *umpirespb.Model, suffix string) *umpirespb.Function {
	for _, f := range m.GetFunctions() {
		if strings.HasSuffix(f.GetName(), suffix) {
			return f
		}
	}
	return nil
}

const activityIR = "../../../model/ir/activity-standalone.json"

// stuck is the first reachable state that is not an end and has no row with a result, or "". It reads
// the rows themselves, not the table's index of them, which a test that edits the rows leaves stale.
func stuck(t *Table) string {
	for _, s := range t.Reachable {
		if !slices.Contains(t.Ends, s) && !slices.ContainsFunc(t.Rows, func(r Row) bool { return r.Source == s && len(r.Results) > 0 }) {
			return s
		}
	}
	return ""
}
