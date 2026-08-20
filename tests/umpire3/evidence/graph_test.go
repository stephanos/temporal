package evidence

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBeforeRejectsCrossClockTimestamps(t *testing.T) {
	t.Parallel()

	graph := Graph{Facts: []Fact{
		fact("left", "source-a", "clock-a", 1, 10, "left-ref"),
		fact("right", "source-b", "clock-b", 1, 20, "right-ref"),
	}}

	before, err := graph.Before("left", "right")
	require.NoError(t, err)
	require.False(t, before)
}

func TestBeforeAcceptsSameSourceSequenceOrCausalReference(t *testing.T) {
	t.Parallel()

	left := fact("left", "source", "clock", 1, 20, "left-ref")
	right := fact("right", "source", "clock", 2, 10, "right-ref")
	graph := Graph{Facts: []Fact{left, right}}

	before, err := graph.Before("left", "right")
	require.NoError(t, err)
	require.True(t, before)

	right.SourceIdentity = "other-source"
	right.ClockDomain = "other-clock"
	right.CausalReferences = []string{"left-ref"}
	graph = Graph{Facts: []Fact{left, right}}
	before, err = graph.Before("left", "right")
	require.NoError(t, err)
	require.True(t, before)
}

func TestBuilderRejectsContradictoryIndependentEvidence(t *testing.T) {
	t.Parallel()

	builder := NewBuilder(Limits{MaxFacts: 4, MaxBytes: 1 << 20})
	first := fact("first", "public-api", "api-sequence", 1, 10, "first-ref")
	first.Kind = "operation-closed"
	second := fact("second", "history", "history-sequence", 1, 20, "second-ref")
	second.Kind = first.Kind
	second.Value = false
	require.NoError(t, builder.AddFact(first))
	require.NoError(t, builder.AddFact(second))

	_, err := builder.Build()
	var contradiction *ContradictionError
	require.ErrorAs(t, err, &contradiction)
	require.ElementsMatch(t, []string{"first", "second"}, contradiction.Facts)
}

func TestBuilderRequiresSourceIdentityAndLineage(t *testing.T) {
	t.Parallel()

	builder := NewBuilder(Limits{MaxFacts: 1, MaxBytes: 1 << 20})
	incomplete := fact("fact", "", "clock", 1, 10, "reference")
	incomplete.Lineage = nil

	require.ErrorContains(t, builder.AddFact(incomplete), "source identity")
}

func fact(identifier, source, clock string, sequence, observedAt int64, reference string) Fact {
	return Fact{
		Identifier: identifier, Kind: "observation", Value: true,
		SourceIdentity: source, ClockDomain: clock, SourceSequence: sequence,
		ObservedAtUnixNano: observedAt, Reference: reference,
		EntityIdentity: "entity", Lineage: []string{"namespace", "entity"},
	}
}
