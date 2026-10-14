package check

// Named choices (model/SEMANTICS.md): each result of a step may carry the name of the alternative it
// is, written on its step record's construct. The names are inert: a row's results report them, and
// nothing else a Model derives changes with them. The fixture is the lifted admission specimen, whose
// committed admission `admitted` either consumes the message or, besides that, retains it for another
// delivery: two results of one row.

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
)

// TestNamedChoicesAreReportedAndInert names the alternatives of the admission specimen's steps. Each
// result reports the name of the step record it was read from, in the order of the results, and an
// unnamed one reports none. Every table, as encoded, every Definition ID and Behavior Fingerprint, every
// refinement, every Property's answer on every row and every receipt, Query answers included, is the
// unnamed Model's.
func TestNamedChoicesAreReportedAndInert(t *testing.T) {
	plain, named := namedAdmission(t)
	require.NoError(t, ir.Validate(named))
	before, err := interp.Build(plain)
	require.NoError(t, err)
	after, err := interp.Build(named)
	require.NoError(t, err)
	require.Len(t, after, len(before))
	seen := map[string]int{}
	for name, mm := range after {
		was := before[name].Table
		is := mm.Table
		wasJSON, err := json.Marshal(was)
		require.NoError(t, err)
		isJSON, err := json.Marshal(is)
		require.NoError(t, err)
		require.Equal(t, sha256.Sum256(wasJSON), sha256.Sum256(isJSON), "%s: the table encodes to other bytes", name)
		require.Equal(t, was.IDs(), is.IDs(), name)
		require.Equal(t, was.TargetFingerprint(), is.TargetFingerprint(), name)
		require.Len(t, is.Rows, len(was.Rows), name)
		for i, row := range is.Rows {
			unnamed := choicesOf(was.Rows[i])
			require.Equal(t, make([]string, len(unnamed)), unnamed, "the unnamed Model names no result")
			want := make([]string, len(row.Results))
			switch {
			case name == "activityProduct":
			case row.Action == "dispatch":
				want = []string{"enqueued"}
			case row.Action == "poll" && len(row.Results) == 2:
				want = []string{"consumed", "retained"}
			case row.Action == "poll" && slices.Contains(row.Results[0].Facts, "admissionRejected"):
				require.Equal(t, "activityRecord", name, "only admitCurrent rejects a stale message")
				want = []string{""}
			case row.Action == "poll":
				want = []string{"consumed"}
			default:
			}
			require.Equal(t, want, choicesOf(row), "%s %s", name, row.Key)
			seen[fmt.Sprintf("%s %s %d %s", name, row.Action, len(row.Results), choicesOf(row)[0])]++
		}
	}
	for _, kind := range []string{
		"activityRecord dispatch 1 enqueued", "trustingActivityRecord dispatch 1 enqueued",
		"activityRecord poll 2 consumed", "trustingActivityRecord poll 2 consumed",
		"activityRecord poll 1 consumed", "trustingActivityRecord poll 1 consumed",
		"activityRecord poll 1 ",
	} {
		require.Positive(t, seen[kind], "no row of the kind %q", kind)
	}
	// What Check derives of the Model besides its tables: each refinement, each Property on every row of
	// its machine and of the machines that refine it, and every receipt, Query answers among them.
	refined := 0
	for _, machine := range plain.GetMachines() {
		if machine.GetRefines() == nil {
			continue
		}
		refined++
		wasRows, wasErr := refinementOf(t, plain, machine.GetName())
		isRows, isErr := refinementOf(t, named, machine.GetName())
		require.Equal(t, fmt.Sprint(wasErr), fmt.Sprint(isErr), machine.GetName())
		require.Equal(t, wasRows, isRows, machine.GetName())
	}
	require.Positive(t, refined)
	require.Equal(t, propertyAnswers(t, plain), propertyAnswers(t, named))
	require.Equal(t, causesAsText(checked(t, plain).Receipts), causesAsText(checked(t, named).Receipts))
}

// propertyAnswers is each Property's answer on each result of each row of its machine it is about: the
// row and result, and whether it holds there or why it cannot be read.
func propertyAnswers(t *testing.T, m *umpirespb.Model) map[string][]string {
	t.Helper()
	b := bind(m, DefaultScope)
	out := map[string][]string{}
	for _, p := range m.GetProperties() {
		subject := b.subject(p.GetMachine())
		require.NoError(t, subject.err, p.GetName())
		reading, err := b.propertyReads(subject, p)
		require.NoError(t, err, p.GetName())
		bound := boundProperty(p, reading)
		answers := []string{}
		for _, row := range subject.table.Rows {
			if !bound.About(row.Action) {
				continue
			}
			for i, result := range row.Results {
				holds, err := bound.Holds(row.Source, result)
				answers = append(answers, fmt.Sprintf("%s %d: %t %v", row.Key, i, holds, err))
			}
		}
		out[p.GetMachine()+" "+p.GetName()] = answers
	}
	require.NotEmpty(t, out)
	return out
}

// causesAsText is the receipts with each cause read as its message.
func causesAsText(receipts []Receipt) []Receipt {
	out := slices.Clone(receipts)
	for i := range out {
		if out[i].Cause != nil {
			out[i].Cause = errors.New(out[i].Cause.Error())
		}
		out[i].Also = causesAsText(out[i].Also)
	}
	return out
}
