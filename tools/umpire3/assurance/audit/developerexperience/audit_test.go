package developerexperience

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuditExercisesThePublicAuthoringBudgetDeterministically(t *testing.T) {
	promotion := `package promotion

import (
	"go.temporal.io/server/tools/umpire3/execution"
	"go.temporal.io/server/tools/umpire3/scenario"
	"go.temporal.io/server/tools/umpire3/regression"
)

func RequirePromoted(t regression.TestingT, factory execution.Factory) {
	authored := scenario.ProtocolAtomicScenario("promoted", []scenario.Resource{
		scenario.Callback("callback"),
	}, scenario.OnePath(
		scenario.RecordCallbackResponse("respond"),
		scenario.RequireCallbackResponseConsistency(),
	))
	regression.RequireRegression(t, authored, regression.WithEnvironment(factory))
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
	"go.temporal.io/server/tools/umpire3/protocol"
	"go.temporal.io/server/tools/umpire3/regression"
)
var _ = protocol.Experiment{}
var _ = regression.RequireRegression`)
	require.ErrorContains(t, err, "artifact plumbing")
}

func capabilities(cases []Case) []Capability {
	result := make([]Capability, len(cases))
	for index, item := range cases {
		result[index] = item.Capability
	}
	return result
}
