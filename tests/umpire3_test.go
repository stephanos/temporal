package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/parallelsuite"
	"go.temporal.io/server/tests/umpire3/compiler"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/migration"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/regress"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
	"go.temporal.io/server/tests/umpire3/umpire3test"
)

// Umpire3TestSuite is an end-to-end test of both halves of the umpire together:
// the compiler produces checked experiments, an environment realizes them, and
// generated monitors qualify the resulting evidence.
type Umpire3TestSuite struct {
	parallelsuite.Suite[*Umpire3TestSuite]
}

func TestUmpire3TestSuite(t *testing.T) {
	parallelsuite.Run(t, &Umpire3TestSuite{})
}

func TestUmpire3IndependentHistoryCorroboratesPublicHistory(t *testing.T) {
	result := evaluateUmpire3Behavior(t, "PlanAndDriveWorkflowToCompletion", "", false)
	require.Equal(t, umpire3runtime.ClaimConforming, result.Claim.Kind)
	require.Equal(t, environment.EvidenceProfileDualHistory, result.Environment.EvidenceProfile)
	require.Contains(t, umpire3ObservationSources(result.Observations), "temporal-public-history")
	require.Contains(t, umpire3ObservationSources(result.Observations), "temporal-history-service")
}

func TestUmpire3WorkflowTaskOwnershipFencingUsesStaleTaskToken(t *testing.T) {
	runUmpire3Regression(t, regress.NewScenario(
		"umpire3-workflow-task-ownership-fencing",
		protocol.TargetIDFoundationOwnershipFencing,
		[]regress.Resource{{Identifier: "workflow-task", Kind: protocol.EntityKindWorkflowTask}},
		regress.OnePath(
			regress.Action("fence-owner", protocol.ActionKindFenceWorkflowOwner),
			regress.Require(protocol.PropertyIDWorkflowTaskOwnershipFencing),
		),
	))
}

func umpire3ObservationSources(observations []environment.Observation) []string {
	sources := make([]string, len(observations))
	for index, observation := range observations {
		sources[index] = observation.Source
	}
	return sources
}

func (s *Umpire3TestSuite) TestPlanAndDriveWorkflowToCompletion() {
	runUmpire3Behavior(s.T(), "PlanAndDriveWorkflowToCompletion", "")
}

// TestPlanAndDriveKitchenSinkWorkflow is the same compile -> drive -> judge loop, but records the
// participant-program migration of the kitchensink workflow as a distinct regression.
func (s *Umpire3TestSuite) TestPlanAndDriveKitchenSinkWorkflow() {
	runUmpire3Behavior(s.T(), "PlanAndDriveKitchenSinkWorkflow", "")
}

// TestPlanAndDriveNexusOperationCHASM retains the CHASM Nexus semantic outcome while Umpire3
// qualifies it through its own catalog, compiler, environment, and evidence graph.
func (s *Umpire3TestSuite) TestPlanAndDriveNexusOperationCHASM() {
	runUmpire3Behavior(s.T(), "PlanAndDriveNexusOperationCHASM", "chasm")
}

// TestPlanAndDriveKitchenSinkNexusOperation keeps the independently named kitchensink path in the
// root functional suite while sharing only Umpire3's normal semantic execution seam.
func (s *Umpire3TestSuite) TestPlanAndDriveKitchenSinkNexusOperation() {
	runUmpire3Behavior(s.T(), "PlanAndDriveKitchenSinkNexusOperation", "chasm")
}

func runUmpire3Behavior(t *testing.T, behavior string, variant string) {
	t.Helper()
	scenario, err := migration.Scenario(behavior, variant)
	require.NoError(t, err)
	runUmpire3Regression(t, scenario, variant)
}

func evaluateUmpire3Behavior(t *testing.T, behavior string, variant string, negativeControl bool) umpire3runtime.Result {
	t.Helper()
	scenario, err := migration.Scenario(behavior, variant)
	require.NoError(t, err)
	return evaluateUmpire3Regression(t, scenario, variant, negativeControl)
}

func evaluateUmpire3BehaviorIn(
	t *testing.T,
	behavior string,
	variant string,
	factory environment.Factory,
) umpire3runtime.Result {
	t.Helper()
	scenario, err := migration.Scenario(behavior, variant)
	require.NoError(t, err)
	return evaluateUmpire3RegressionIn(t, scenario, factory)
}

func runUmpire3Regression(t *testing.T, scenario compiler.Scenario, variant ...string) {
	t.Helper()
	umpire3test.RequireRegression(t, scenario, umpire3test.WithEnvironment(newUmpire3RootEnvironment(t, false, variant...)))
}

func evaluateUmpire3Regression(
	t *testing.T,
	scenario compiler.Scenario,
	variant string,
	negativeControl bool,
) umpire3runtime.Result {
	t.Helper()
	return evaluateUmpire3RegressionIn(t, scenario, newUmpire3RootEnvironment(t, negativeControl, variant))
}

func evaluateUmpire3RegressionIn(
	t *testing.T,
	scenario compiler.Scenario,
	factory environment.Factory,
) umpire3runtime.Result {
	t.Helper()
	suite, err := compiler.Compile(context.Background(), scenario, compiler.Limits{
		MaxPaths: 32, MaxActions: 128, MaxStates: 10000, MaxMemoryBytes: 16 << 20,
		MaxTime: 10 * time.Second,
	})
	require.NoError(t, err)
	require.Len(t, suite.Experiments, 1)
	result, err := umpire3runtime.Run(context.Background(), umpire3runtime.Request{
		Experiment: suite.Experiments[0], Environment: factory,
	})
	require.NoError(t, err)
	return result
}

func umpire3ProgressScenario(identifier string, target protocol.TargetID, kind protocol.EntityKind) compiler.Scenario {
	return umpire3AssuranceScenario(identifier, target, protocol.PropertyIDEntityProgress,
		kind, protocol.ActionKindProgressEntity)
}

func umpire3AssuranceScenario(
	identifier string,
	target protocol.TargetID,
	property protocol.PropertyID,
	resourceKind protocol.EntityKind,
	action protocol.ActionKind,
) compiler.Scenario {
	return regress.NewScenario(identifier, target,
		[]regress.Resource{{Identifier: identifier + "-resource", Kind: resourceKind}},
		regress.OnePath(regress.Action(identifier+"-action", action), regress.Require(property)))
}

func umpire3NexusCancellationScenario(identifier string) compiler.Scenario {
	return umpire3AssuranceScenario(identifier, protocol.TargetIDNexusCancellation,
		protocol.PropertyIDNexusCancellationWonExcludesSuccess,
		protocol.EntityKindNexusOperation, protocol.ActionKindPersistSuccess)
}

func umpire3UpdateScenario(identifier string) compiler.Scenario {
	return umpire3AssuranceScenario(identifier, protocol.TargetIDWorkflowUpdateLifecycle,
		protocol.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory,
		protocol.EntityKindWorkflowUpdate, protocol.ActionKindCompleteUpdate)
}

func newUmpire3RootEnvironment(t *testing.T, negativeControl bool, variant ...string) environment.Factory {
	t.Helper()
	return newUmpire3SDKRootFactory(t, negativeControl, variant...)
}
