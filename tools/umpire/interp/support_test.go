package interp

// What only the reader's tests ask of its results: no consumer reads a report or a machine this way.

import (
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// BuildWithin is Build within explicit ceilings.
func BuildWithin(m *umpirespb.Model, c Ceilings) (map[string]*Machine, error) {
	in := NewInterpreterWithin(m, c)
	return in.Build(m)
}

// disabled is whether a state and class of the machine are a disabled pair: an empty list of steps,
// neither a row nor a hole row.
func disabled(m *Machine, state, class string) bool {
	if _, ok := m.State(state); !ok || !slices.ContainsFunc(m.Classes, func(c Class) bool { return c.Key == class }) {
		return false
	}
	key := state + "-" + class
	return !slices.ContainsFunc(m.Table.RowsFrom(state), func(r Row) bool { return r.Key == key }) &&
		!slices.ContainsFunc(m.Holes, func(h HoleRow) bool { return h.Row == key })
}
