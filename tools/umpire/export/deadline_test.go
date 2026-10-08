package export

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
)

func deadlineSlice(t *testing.T) *Slice {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", "deadlineCapabilities.json"))
	require.NoError(t, err)
	s := openSlice(t, m)
	s.Name = "deadlineCapabilities"
	return s
}

func TestDeadlineSelectorsAndBeforeStateAgreeWithTheReader(t *testing.T) {
	s := deadlineSlice(t)
	r := check.Check(s.Model, check.DefaultScope)
	require.Len(t, r.Receipts, 9)
	for _, receipt := range r.Receipts {
		require.Equal(t, check.Verified, receipt.Kind, receipt.Explanation)
		require.True(t, receipt.Exercised, receipt.Key.Name)
	}
	selected, skipped, eligibleRetries := 0, 0, 0
	for _, mm := range s.Model.GetMachines() {
		view, err := s.view(s.machines[mm.GetName()])
		require.NoError(t, err)
		for _, receipt := range r.Receipts {
			if receipt.Key.Owner != mm.GetName() {
				continue
			}
			bound, err := s.bound.Bound(receipt.Key)
			require.NoError(t, err)
			property := slices.Index(view.Properties, receipt.Key.Name)
			require.GreaterOrEqual(t, property, 0)
			for state, by := range view.Claims {
				for action, steps := range by {
					about := bound.Property.About(action)
					for i, reads := range steps {
						if !about {
							require.Equal(t, claimRead{Holds: true}, reads[property])
							skipped++
							continue
						}
						rows := bound.Table.RowsFrom(state)
						row := slices.IndexFunc(rows, func(row interp.Row) bool { return row.Action == action })
						require.GreaterOrEqual(t, row, 0)
						held, err := bound.Property.Holds(state, rows[row].Results[i])
						require.NoError(t, err)
						require.Equal(t, claimRead{About: true, Holds: held}, reads[property])
						require.True(t, held)
						selected++
						if receipt.Key.Name == "deadlineTimers.start.deadlineReturnsToWaiting" && state == "held-true-true-true-false-false" {
							require.Equal(t, "waiting-true-true-false-false-false", rows[row].Results[i].State)
							require.Empty(t, rows[row].Results[i].Facts)
							eligibleRetries++
						}
					}
				}
			}
		}
	}
	require.Positive(t, selected)
	require.Positive(t, skipped)
	require.Equal(t, 1, eligibleRetries)
	x := exported(t, s)
	receipts, err := s.QuintAgreement(x, encodeDump(t, s, x, nil))
	require.NoError(t, err)
	for _, machine := range x.Machines {
		r := only(t, receipts, PropertyAgreement, machine)
		require.Equal(t, Agreed, r.Kind, "%v", r.Differences)
		require.Positive(t, r.About)
		require.Equal(t, Agreed, only(t, receipts, TransitionAgreement, machine).Kind)
	}
}

func TestQuintReadsDeadlineSelectorsAndBeforeState(t *testing.T) {
	needs(t, QuintTool)
	s := deadlineSlice(t)
	x := exported(t, s)
	receipts, err := s.QuintAgreement(x, quintDump(t, x))
	require.NoError(t, err)
	for _, machine := range x.Machines {
		for _, claim := range []Claim{TransitionAgreement, PropertyAgreement} {
			r := only(t, receipts, claim, machine)
			require.Equal(t, Agreed, r.Kind, "%v", r.Differences)
			require.Positive(t, r.Reads+r.Enabled)
			report(t, r)
		}
	}
}
