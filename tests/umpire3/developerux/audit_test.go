package developerux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuditExercisesThePublicAuthoringBudgetDeterministically(t *testing.T) {
	promotion := `package promotion

import (
	"go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/scenario"
	"go.temporal.io/server/tests/umpire3/umpire3test"
)

func RequirePromoted(t umpire3test.TestingT, factory execution.Factory) {
	authored := scenario.ProtocolAtomicRegression("promoted", []scenario.Resource{
		scenario.Callback("callback"),
	}, scenario.OnePath(
		scenario.RecordCallbackResponse("respond"),
		scenario.RequireCallbackResponseConsistency(),
	))
	umpire3test.RequireRegression(t, authored, umpire3test.WithEnvironment(factory))
}`

	first, err := Run(promotion)
	require.NoError(t, err)
	second, err := Run(promotion)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.NoError(t, first.Validate())
	require.Equal(t, []Capability{
		CapabilityFirstRegression,
		CapabilityPartialOrder,
		CapabilityRuntimeIdentity,
		CapabilityTypedFault,
	}, capabilities(first.Cases))
	require.Equal(t, 2, first.Cases[1].PathCount)
	require.Positive(t, first.Cases[2].IdentityCount)
	require.Positive(t, first.Cases[3].FaultCount)
	require.True(t, first.Promotion.RequireRegression)
	require.NotEmpty(t, first.ArtifactDigest)
}

func TestAuditRejectsPromotionArtifactPlumbing(t *testing.T) {
	_, err := Run(`package promotion
import (
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/umpire3test"
)
var _ = protocol.Experiment{}
var _ = umpire3test.RequireRegression`)
	require.ErrorContains(t, err, "artifact plumbing")
}

func capabilities(cases []Case) []Capability {
	result := make([]Capability, len(cases))
	for index, item := range cases {
		result[index] = item.Capability
	}
	return result
}
