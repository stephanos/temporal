package mutation

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
)

func TestCoveragePointsRequireEveryMonitorObservationBeforeCoveringEvidence(t *testing.T) {
	t.Parallel()

	denominator, err := protocolcatalog.DefaultCoverageDenominator()
	require.NoError(t, err)
	experiment := loadMutationExperiment(t)

	points, err := CoveragePointsForExperiment(denominator, experiment)
	require.NoError(t, err)
	require.True(t, slices.ContainsFunc(points, func(point protocolcatalog.ModelCoveragePoint) bool {
		return point.Dimension == protocolcatalog.CoverageEvidence &&
			point.Identifier == "nexus-cancellation/evidence:causal"
	}))

	experiment.Checkpoints[2].Observation = "cancellation-accepted"
	points, err = CoveragePointsForExperiment(denominator, experiment)
	require.NoError(t, err)
	require.False(t, slices.ContainsFunc(points, func(point protocolcatalog.ModelCoveragePoint) bool {
		return point.Dimension == protocolcatalog.CoverageEvidence
	}))
}
