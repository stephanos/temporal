package conformance

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// wide is a machine with one state and one step from it back to it that records n facts, with one
// observation of each: the step can explain any of the 2^n subsets of the observations.
func wide(n int, limits Limits) (*plan, *ordered) {
	p := &plan{limits: limits, states: []string{"only"}, ends: []bool{true}, steps: [][]step{{{target: 0}}}, holes: [][]string{nil}}
	o := &ordered{before: make([]uint64, n)}
	for i := range n {
		name := fmt.Sprint("fact", i)
		p.steps[0][0].facts = append(p.steps[0][0].facts, name)
		o.evidence = append(o.evidence, &observation{identity: name, kind: &kind{records: name}})
	}
	return p, o
}

// A ceiling is consulted before the work it bounds is done or made room for. A step that could
// explain 2^22 sets of observations, and one that could explain 2^64, more than any memory holds,
// are both stopped at a work ceiling of one after allocating next to nothing: the sets are tried one
// at a time, each charged before it is built, and never listed.
func TestACeilingIsReachedBeforeTheWorkItBoundsIsDone(t *testing.T) {
	for _, n := range []int{22, 64} {
		t.Run(fmt.Sprint(n, " facts"), func(t *testing.T) {
			limits := generous
			limits.MaxWork, limits.MaxCandidates = 1, 1
			p, o := wide(n, limits)
			spent := 0
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			read, err := p.explore(o, regime{event: 9}, &spent)
			runtime.ReadMemStats(&after)
			require.Nil(t, read)
			var reached *LimitError
			require.ErrorAs(t, err, &reached)
			require.Equal(t, &LimitError{Resource: "work", Ceiling: 1, Event: 9}, reached)
			require.Equal(t, 1, spent, "the count is the work done")
			require.Less(t, after.TotalAlloc-before.TotalAlloc, uint64(1<<16), "bytes allocated")
		})
	}
}

// The same step within ceilings that admit it is explored whole: with three facts, every one of the
// eight subsets is a candidate, and the work is counted as done, one unit for each observation tried
// and one for each step taken.
func TestEverySetOfObservationsAStepCanExplainIsTried(t *testing.T) {
	p, o := wide(3, generous)
	spent := 0
	read, err := p.explore(o, regime{}, &spent)
	require.NoError(t, err)
	// Only the candidate that has explained all three explains the evidence.
	require.Equal(t, 1, read.candidates)
	// A candidate with m observations left takes the step once for each of the 2^m sets of them, and
	// tries an observation once at each of the 2^m - 1 branches that add one to a set. Over the 8
	// candidates, C(3,m) of them for each m: 27 steps and 19 tries.
	require.Equal(t, 46, spent)
}

// A step that records one fact twice reaches the set of one observation two ways, and takes it once:
// the unobserved step, one unit; the observation tried for the second fact and the step taken with
// it, two units; the observation tried for the first fact, one unit, and the same set not taken again.
func TestASetReachedTwoWaysIsTakenOnce(t *testing.T) {
	p := &plan{limits: generous, states: []string{"before", "after"}, ends: []bool{false, true},
		steps: [][]step{{{target: 1, facts: []string{"f", "f"}}}, nil}, holes: [][]string{nil, nil}}
	spent := 0
	read, err := p.explore(&ordered{evidence: []*observation{{identity: "e", kind: &kind{records: "f"}}}, before: make([]uint64, 1)}, regime{}, &spent)
	require.NoError(t, err)
	require.Equal(t, 1, read.candidates)
	require.Equal(t, 4, spent)
}
