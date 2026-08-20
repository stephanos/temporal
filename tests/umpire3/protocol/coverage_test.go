package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedCoverageDenominatorClassifiesEveryCatalogTargetProperty(t *testing.T) {
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

	defined := 0
	undefined := 0
	var nexus CoverageTarget
	for _, target := range denominator.Targets {
		switch target.Status {
		case CoverageDenominatorDefined:
			defined++
		case CoverageDenominatorUndefined:
			undefined++
		default:
			require.FailNow(t, "unknown coverage denominator status", target.Status)
		}
		if target.Identifier == TargetIDFeatureNexus && target.Property == PropertyIDNexusOperationClosure {
			nexus = target
		}
	}
	require.Equal(t, 1, defined)
	require.Equal(t, targetProperties-1, undefined)
	require.Equal(t, CoverageDenominatorDefined, nexus.Status)
	require.Empty(t, nexus.Reason)
	require.Len(t, nexus.Edges, 17)
	identifiers := make(map[string]struct{}, len(nexus.Edges))
	for _, edge := range nexus.Edges {
		identifiers[edge.Identifier] = struct{}{}
	}
	require.Len(t, identifiers, 17)
}

func TestCoverageDenominatorRejectsUndefinedTargetWithEdges(t *testing.T) {
	denominator, err := DefaultCoverageDenominator()
	require.NoError(t, err)
	for index := range denominator.Targets {
		if denominator.Targets[index].Status == CoverageDenominatorUndefined {
			denominator.Targets[index].Edges = []CoverageEdge{{
				Identifier: "invented", FromState: "from", Action: "act", ToState: "to",
			}}
			require.ErrorContains(t, denominator.Validate(), "undefined")
			return
		}
	}
	require.FailNow(t, "generated denominator has no undefined target")
}
