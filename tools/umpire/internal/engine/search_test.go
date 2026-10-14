package engine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func endCatalogQuery(t *testing.T, ends bool, holds bool) *Query {
	t.Helper()
	states := []string{"start"}
	for i := range 1024 {
		states = append(states, fmt.Sprintf("end-%d", i))
	}
	spec := TableSpec{Machine: "ends", States: states, Actions: []string{"go"}, Outcomes: []string{"accepted"},
		Starts: []string{"start"}, Rows: []Row{{Key: "start-go", Source: "start", Action: "go",
			Results: []Result{{Outcome: "accepted", State: "end-0"}}}}}
	if ends {
		spec.Ends = states[1:]
	}
	table := NewTable(spec)
	require.NoError(t, table.Err())
	property := KeyTransitionProperty(table, "holds", func(string, Result) (bool, error) { return holds, nil })
	return KeyVerify("q", property, KeyScenario(table, "one", "start", "go"), Limits{Name: "one", Steps: 1, Actions: 1, Search: 2})
}

func TestUnmonitoredAnswersDoNotAllocateTheUnusedEndCatalog(t *testing.T) {
	without, many := endCatalogQuery(t, false, false), endCatalogQuery(t, true, false)
	expected, err := without.Answer()
	require.NoError(t, err)
	require.Equal(t, CounterexampleFound, expected.Outcome)
	require.Equal(t, []string{"start-go"}, expected.Rows)
	require.True(t, expected.Exercised)
	require.NoError(t, without.Replay(expected))
	measure := func(q *Query) float64 {
		return testing.AllocsPerRun(3, func() {
			answer, err := q.Answer()
			require.NoError(t, err)
			require.Equal(t, expected, answer)
		})
	}
	plain, catalog := measure(without), measure(many)
	t.Logf("no-monitor allocations: no ends=%g, 1024 ends=%g", plain, catalog)
	require.LessOrEqual(t, catalog, plain+1, "unused terminal keys must not allocate per Query")
	require.NoError(t, many.Replay(expected))
}

// originalEndCatalogSearcher preserves the pre-repair preparation of the same search algorithm.
func originalEndCatalogSearcher(q *Query) (*searcher, error) {
	if err := q.check(); err != nil {
		return nil, err
	}
	table, err := q.Scenario.Machine.Table()
	if err != nil {
		return nil, err
	}
	if err := q.checkScenario(table); err != nil {
		return nil, err
	}
	var ref *Refinement
	if q.refinement != nil {
		if ref, err = q.refinement(); err != nil {
			return nil, err
		}
		if err := q.checkRefined(table, ref); err != nil {
			return nil, err
		}
	}
	ends := map[string]bool{}
	for _, end := range table.Ends {
		ends[end] = true
	}
	return &searcher{q: q, t: table, ref: ref, ends: ends, monRead: make([]bool, len(q.monitors))}, nil
}

func TestEndCatalogPreparationPreservesTheOriginalAnswersAndReplay(t *testing.T) {
	for _, test := range []struct {
		name string
		at   *Evaluation
	}{
		{name: "no monitors"},
		{name: "every step", at: new(EveryStep())},
		{name: "at ends", at: new(AtEnds())},
		{name: "after step", at: new(AfterKey(func(r Result) (bool, error) { return r.State == "end-0", nil }))},
	} {
		t.Run(test.name, func(t *testing.T) {
			q := endCatalogQuery(t, true, test.at != nil)
			if test.at != nil {
				q.Watch(KeyMonitor("end", "start", func(_, _ string, r Result) (string, error) { return r.State, nil },
					func(state string) (bool, error) { return state == "end-0", nil }, *test.at))
			}
			original, err := originalEndCatalogSearcher(q)
			require.NoError(t, err)
			expected, err := original.run()
			require.NoError(t, err)
			expected.Unknown, expected.Expanded = original.unknown, original.expanded
			answer, err := q.Answer()
			require.NoError(t, err)
			require.Equal(t, expected, answer)
			require.NoError(t, q.Replay(answer))
		})
	}
}
