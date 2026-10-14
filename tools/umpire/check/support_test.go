package check

// What only the reader's tests ask of its results: no consumer reads a report or a machine this way.

import (
	"slices"
	"testing"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
)

// IsCheck reports whether a receipt of this kind is the answer of a check that ran: a result, one
// left incomplete by a hole, or one a limit of its own cut. An unsupported declaration, an error and
// work a ceiling refused are none.
func (k ReceiptKind) IsCheck() bool {
	switch k {
	case Verified, Found, NotFound, Counterexample, RefinementRejected, Incomplete, LimitReached, Unresolved:
		return true
	default:
		return false
	}
}

// Checks lists the receipts that are answers of checks that ran.
func (r *Report) Checks() []Receipt {
	var out []Receipt
	for _, x := range r.Receipts {
		if x.Kind.IsCheck() {
			out = append(out, x)
		}
	}
	return out
}

// Unsupported lists the declarations this reader did not check.
func (r *Report) Unsupported() []Receipt {
	var out []Receipt
	for _, x := range r.Receipts {
		if x.Kind == Unsupported {
			out = append(out, x)
		}
	}
	return out
}

// refinementOf is a refining machine's refinement as Check reads it: the rows RefineTables pairs
// with the product's steps, and why the refinement is not established.
func refinementOf(t *testing.T, m *umpirespb.Model, machine string) ([]RefinementRow, error) {
	t.Helper()
	b := bind(m, DefaultScope)
	r := b.refinement(b.subject(machine))
	if r.ref == nil {
		return nil, r.err
	}
	return r.ref.Rows, r.err
}

// disabled is whether a state and class of the machine are a disabled pair: an empty list of steps,
// neither a row nor a hole row.
func disabled(m *interp.Machine, state, class string) bool {
	if _, ok := m.State(state); !ok || !slices.ContainsFunc(m.Classes, func(c interp.Class) bool { return c.Key == class }) {
		return false
	}
	key := state + "-" + class
	return !slices.ContainsFunc(m.Table.RowsFrom(state), func(r interp.Row) bool { return r.Key == key }) &&
		!slices.ContainsFunc(m.Holes, func(h interp.HoleRow) bool { return h.Row == key })
}
