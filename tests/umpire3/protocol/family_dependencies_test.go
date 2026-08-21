package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFamilyDependencyGraphSelectsTargetsFromTransitiveLeanSources(t *testing.T) {
	t.Parallel()

	graph, err := DefaultFamilyDependencyGraph()
	require.NoError(t, err)
	catalog, err := DefaultCatalog()
	require.NoError(t, err)

	require.Equal(t, []TargetID{TargetIDNexusCancellation},
		graph.AffectedTargets([]string{"Temporal/Product/Nexus.lean"}))
	require.Len(t, graph.AffectedTargets([]string{"Umpire3/Behavior.lean"}), len(catalog.Targets))
	require.Empty(t, graph.AffectedTargets([]string{"docs/unrelated.md"}))
}

func TestFamilyDependencyGraphDeclaresTargetSpecificCheckerPortfolio(t *testing.T) {
	t.Parallel()

	graph, err := DefaultFamilyDependencyGraph()
	require.NoError(t, err)

	nexus, found := graph.Family(TargetIDNexusCancellation)
	require.True(t, found)
	require.Equal(t, []string{"exact", "native", "veil"}, nexus.Checkers)
	require.NotEmpty(t, nexus.LeanTests)

	delivery, found := graph.Family(TargetIDFoundationDeliverySafety)
	require.True(t, found)
	require.Equal(t, []string{"exact", "lean-temporal"}, delivery.Checkers)
}
