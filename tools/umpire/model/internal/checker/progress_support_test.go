package checker

// Assumes lists the assumptions every check of the machine relies on.
func (m *Machine[S, O, F]) Assumes(assumptions ...Assumption) *Machine[S, O, F] {
	m.assumptions = append(m.assumptions, assumptions...)
	return m
}

// NewProgress declares a progress claim over a machine whose state type is S.
func NewProgress[S any](name string, from, to func(S) bool, within int, assumptions ...Assumption) *Progress {
	typed := func(f func(S) bool) func(*Table, string) (bool, error) {
		return func(t *Table, key string) (bool, error) {
			v := t.stateValue[key]
			s, ok := v.(S)
			if !ok {
				return false, errorf("progress "+name, "the state %s of %s is not a %T", key, t.Machine, s)
			}
			return f(s), nil
		}
	}
	return &Progress{Name: name, Within: within, Assumptions: assumptions, from: typed(from), to: typed(to)}
}

// KeyProgress declares a progress claim over state keys, for a table with no typed values.
func KeyProgress(name string, from, to func(state string) bool, within int, assumptions ...Assumption) *Progress {
	keyed := func(f func(string) bool) func(*Table, string) (bool, error) {
		return func(_ *Table, key string) (bool, error) { return f(key), nil }
	}
	return &Progress{Name: name, Within: within, Assumptions: assumptions, from: keyed(from), to: keyed(to)}
}

// Incomplete reports that some kind of violation was ruled out while the check read an unknown
// pair, behind which one may lie. A violation found stands whatever was unknown.
func (a ProgressAnswer) Incomplete() bool {
	verified := func(v ProgressVerdict) bool { return v.Outcome == VerifiedWithinLimits }
	return len(a.Unknown) > 0 && (verified(a.Deadlock) || verified(a.Cycle) || verified(a.Deadline))
}
