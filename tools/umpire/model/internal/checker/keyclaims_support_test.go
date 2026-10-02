package checker

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
