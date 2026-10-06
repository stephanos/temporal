package engine

// KeyProgress declares a progress claim over state keys whose from and to always answer.
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

// Incomplete reports that the answer found no witness and explored an unknown, so what it says is
// absent may lie behind one. A found witness or counterexample stands whatever was unknown.
func (a Answer) Incomplete() bool {
	return (a.Outcome == VerifiedWithinLimits || a.Outcome == NotFound) && len(a.Unknown) > 0
}

// Model is the table as a Model, which claims declared over its keys name as their machine. A table
// has one Model, so a Property and a Scenario over one table name the same machine.
func (t *Table) Model() Model { return t.model }

// UnknownFrom lists the unknown pairs at this state, in the order the table lists them.
func (t *Table) UnknownFrom(state string) []UnknownPair { return t.unknownsFrom(state) }
