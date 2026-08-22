package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedCoverageDenominatorDefinesEveryCatalogTargetProperty(t *testing.T) {
	t.Parallel()

	denominator, err := DefaultCoverageDenominator()
	require.NoError(t, err)
	catalog, err := DefaultCatalog()
	require.NoError(t, err)
	targetProperties := 0
	for _, target := range catalog.Targets {
		targetProperties += len(target.Properties)
	}
	require.Len(t, denominator.Targets, targetProperties)

	kinds := make(map[CoverageDimension]struct{})
	var nexus CoverageTarget
	for _, target := range denominator.Targets {
		require.Equal(t, CoverageDenominatorDefined, target.Status)
		require.NotEmpty(t, target.Points)
		for _, point := range target.Points {
			kinds[point.Dimension] = struct{}{}
		}
		if target.Identifier == TargetIDFeatureNexus && target.Property == PropertyIDNexusOperationClosure {
			nexus = target
		}
	}
	require.Equal(t, map[CoverageDimension]struct{}{
		CoverageTransition:  {},
		CoverageRelation:    {},
		CoverageProperty:    {},
		CoverageFault:       {},
		CoverageObservation: {},
		CoverageRefinement:  {},
		CoverageEvidence:    {},
	}, kinds)
	require.Equal(t, CoverageDenominatorDefined, nexus.Status)
	require.Empty(t, nexus.Reason)
	require.Len(t, nexus.Edges, 17)
	identifiers := make(map[string]struct{}, len(nexus.Edges))
	for _, edge := range nexus.Edges {
		identifiers[edge.Identifier] = struct{}{}
	}
	require.Len(t, identifiers, 17)
}

func TestCoverageDenominatorRejectsDefinedTargetWithoutPoints(t *testing.T) {
	denominator, err := DefaultCoverageDenominator()
	require.NoError(t, err)
	denominator.Targets[0].Points = nil
	require.ErrorContains(t, denominator.Validate(), "points")
}
