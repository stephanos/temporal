package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedNexusCoverageDenominatorHasSeventeenUniqueEdges(t *testing.T) {
	t.Parallel()

	denominator, err := DefaultCoverageDenominator()
	require.NoError(t, err)
	require.Len(t, denominator.Targets, 1)
	require.Equal(t, TargetIDFeatureNexus, denominator.Targets[0].Identifier)
	require.Equal(t, PropertyIDNexusOperationClosure, denominator.Targets[0].Property)
	require.Len(t, denominator.Targets[0].Edges, 17)
	identifiers := make(map[string]struct{}, len(denominator.Targets[0].Edges))
	for _, edge := range denominator.Targets[0].Edges {
		identifiers[edge.Identifier] = struct{}{}
	}
	require.Len(t, identifiers, 17)
}
