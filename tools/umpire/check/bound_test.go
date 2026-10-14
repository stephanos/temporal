package check

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
	"go.temporal.io/server/tools/umpire/interp"
)

// walked is what reading a witness step by step through a bound Query gives: per step, whether the
// Property is about it and holds on it, and each monitor's state and verdict where the walk stops.
type walked struct {
	about, held []bool
	monitors    []MonitorVerdict
}

// walkWitness takes a witness through the bound Query's own table and evaluators, as a reader of recorded
// steps does: nothing of the checker's search is used.
func walkWitness(t *testing.T, b *Bound, w *Trace) walked {
	t.Helper()
	var out walked
	state := w.Initial.Value
	require.Equal(t, b.Start, state, "the witness starts where the bound Query does")
	states := make([]string, len(b.Monitors))
	verdicts := make([]Verdict, len(b.Monitors))
	for i, mo := range b.Monitors {
		states[i], verdicts[i] = mo.Initial, umpire.MonitorUnread
	}
	for _, step := range w.Steps {
		var taken *interp.Result
		for _, row := range b.Table.RowsFrom(state) {
			for i, res := range row.Results {
				if row.Action == step.Action.Value && res.State == step.State.Value && res.Outcome == step.Outcome.Value && len(res.Facts) == len(step.Facts) {
					taken = &row.Results[i]
				}
			}
		}
		require.NotNil(t, taken, "step %s from %s is a result of the bound table", step.Action.Value, state)
		about := b.Property.About(step.Action.Value)
		holds := false
		if about {
			var err error
			holds, err = b.Property.Holds(state, *taken)
			require.NoError(t, err)
		}
		out.about, out.held = append(out.about, about), append(out.held, holds)
		for i, mo := range b.Monitors {
			next, err := mo.Next(states[i], state, *taken)
			require.NoError(t, err)
			states[i] = next
			read := !mo.AtEnds
			if read && mo.Read != nil {
				read, err = mo.Read(*taken)
				require.NoError(t, err)
			}
			if read && verdicts[i] != umpire.MonitorViolated {
				violated, err := mo.Violated(next)
				require.NoError(t, err)
				verdicts[i] = umpire.MonitorHeld
				if violated {
					verdicts[i] = umpire.MonitorViolated
				}
			}
		}
		state = step.State.Value
	}
	for i, mo := range b.Monitors {
		if mo.AtEnds && slices.Contains(b.Table.Ends, state) {
			violated, err := mo.Violated(states[i])
			require.NoError(t, err)
			verdicts[i] = umpire.MonitorHeld
			if violated {
				verdicts[i] = umpire.MonitorViolated
			}
		}
		out.monitors = append(out.monitors, MonitorVerdict{Name: mo.Name, State: states[i], Verdict: verdicts[i]})
	}
	return out
}

// Every Query Check answers with a witness is read again, step by step, through the bound Query a
// runtime reader is given, and says what the receipt says: a found trace holds on every step the
// Property is about and is about one; a counterexample fails on its last step and on none before, or
// leaves the monitor the receipt names violated; and every monitor ends in the state, and with the
// verdict, the receipt reports. A Query Check does not answer on one machine is refused.
func TestABoundQueryReadsStepsAsCheckDoes(t *testing.T) {
	witnesses, refused, atEnds := 0, 0, 0
	for _, name := range []string{"admission", "declarations", "realizations"} {
		m := lifted(t, name)
		realizer, err := NewRealizer(m, DefaultScope)
		require.NoError(t, err)
		for _, receipt := range Check(m, DefaultScope).Receipts {
			if receipt.Subject != QuerySubject {
				continue
			}
			bound, err := realizer.Bound(receipt.Key)
			declared, derr := realizer.Declared(receipt.Key)
			require.NoError(t, derr)
			if realizer.Machine(receipt.Key.Owner) == nil || declared.Query.GetThrough() || receipt.Kind == Unsupported || receipt.Kind == DeclarationError {
				require.Error(t, err, "%s %s %s %v", name, receipt.Key.Name, receipt.Kind, receipt.Key)
				require.Nil(t, bound)
				refused++
				continue
			}
			require.NoError(t, err, "%s %s", name, receipt.Key.Name)
			require.Same(t, realizer.Machine(receipt.Key.Owner).Decl, realizer.Machine(bound.Table.Machine).Decl)
			require.Len(t, bound.Table.Unknown, len(realizer.Machine(receipt.Key.Owner).Holes), "hole rows are the table's unknown pairs")
			for _, mo := range bound.Monitors {
				if !mo.AtEnds {
					continue
				}
				atEnds++
				for _, row := range bound.Table.Rows {
					for _, res := range row.Results {
						read, err := mo.Read(res)
						require.NoError(t, err)
						require.False(t, read, "%s is read at the end of a path, and after no step", mo.Name)
					}
				}
			}
			if receipt.Witness == nil || (receipt.Kind != Found && receipt.Kind != Counterexample) {
				continue
			}
			witnesses++
			read := walkWitness(t, bound, receipt.Witness)
			last := len(read.about) - 1
			failed := slices.IndexFunc(slices.Collect(func(yield func(int) bool) {
				for i := range read.about {
					if !yield(i) {
						return
					}
				}
			}), func(i int) bool { return read.about[i] && !read.held[i] })
			switch {
			case receipt.Kind == Found:
				require.Contains(t, read.about, true, "%s %s", name, receipt.Key.Name)
				require.Equal(t, -1, failed, "%s %s", name, receipt.Key.Name)
			case receipt.Monitor != "":
				require.Equal(t, -1, failed, "%s %s: the monitor is what fails", name, receipt.Key.Name)
				require.Contains(t, read.monitors, MonitorVerdict{Name: receipt.Monitor, State: stateOf(receipt, receipt.Monitor), Verdict: umpire.MonitorViolated})
			default:
				require.Equal(t, last, failed, "%s %s: the Property fails on the witness's last step and on none before", name, receipt.Key.Name)
			}
			for _, verdict := range receipt.Monitors {
				if verdict.Verdict == umpire.MonitorUnknown {
					continue
				}
				require.Contains(t, read.monitors, verdict, "%s %s", name, receipt.Key.Name)
			}
		}
	}
	// admission: the stale design's five counterexamples; declarations: one found; realizations: six
	// found.
	require.Equal(t, 12, witnesses)
	require.Positive(t, refused)
	require.Positive(t, atEnds)
}

func stateOf(r Receipt, monitor string) string {
	for _, verdict := range r.Monitors {
		if verdict.Name == monitor {
			return verdict.State
		}
	}
	return ""
}

// What a bound function could not read because it reached a hole is told apart from a function that
// is malformed.
func TestUnknownIsAHoleAndNothingElse(t *testing.T) {
	require.True(t, Unknown(&interp.Hole{ID: "h"}))
	require.False(t, Unknown(&interp.Error{Message: "not a Boolean"}))
	require.False(t, Unknown(nil))
}
