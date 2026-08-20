package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/parallelsuite"
	umpire3execution "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/migration"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/scenario"
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
	require.Equal(t, umpire3execution.ClaimConforming, result.Claim.Kind)
	require.Equal(t, umpire3execution.EvidenceProfileDualHistory, result.Environment.EvidenceProfile)
	require.Contains(t, umpire3ObservationSources(result.Observations), "temporal-public-history")
	require.Contains(t, umpire3ObservationSources(result.Observations), "temporal-history-service")
}

func TestUmpire3WorkflowTaskOwnershipFencingUsesStaleTaskToken(t *testing.T) {
	runUmpire3Regression(t, scenario.NewScenario(
		"umpire3-workflow-task-ownership-fencing",
		protocol.TargetIDFoundationOwnershipFencing,
		[]scenario.Resource{{Identifier: "workflow-task", Kind: protocol.EntityKindWorkflowTask}},
		scenario.OnePath(
			scenario.Action("fence-owner", protocol.ActionKindFenceWorkflowOwner),
			scenario.Require(protocol.PropertyIDWorkflowTaskOwnershipFencing),
		),
	))
}

func umpire3ObservationSources(observations []umpire3execution.Observation) []string {
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

func evaluateUmpire3Behavior(t *testing.T, behavior string, variant string, negativeControl bool) umpire3execution.Result {
	t.Helper()
	scenario, err := migration.Scenario(behavior, variant)
	require.NoError(t, err)
	return evaluateUmpire3Regression(t, scenario, variant, negativeControl)
}

func evaluateUmpire3BehaviorIn(
	t *testing.T,
	behavior string,
	variant string,
	factory umpire3execution.Factory,
) umpire3execution.Result {
	t.Helper()
	scenario, err := migration.Scenario(behavior, variant)
	require.NoError(t, err)
	return evaluateUmpire3RegressionIn(t, scenario, factory)
}

func runUmpire3Regression(t *testing.T, authored scenario.Scenario, variant ...string) {
	t.Helper()
	umpire3test.RequireRegression(t, authored, umpire3test.WithEnvironment(newUmpire3RootEnvironment(t, false, variant...)))
}

func evaluateUmpire3Regression(
	t *testing.T,
	authored scenario.Scenario,
	variant string,
	negativeControl bool,
) umpire3execution.Result {
	t.Helper()
	return evaluateUmpire3RegressionIn(t, authored, newUmpire3RootEnvironment(t, negativeControl, variant))
}

func evaluateUmpire3RegressionIn(
	t *testing.T,
	authored scenario.Scenario,
	factory umpire3execution.Factory,
) umpire3execution.Result {
	t.Helper()
	suite, err := scenario.Compile(context.Background(), authored, scenario.Limits{
		MaxPaths: 32, MaxActions: 128, MaxStates: 10000, MaxMemoryBytes: 16 << 20,
		MaxTime: 10 * time.Second,
	})
	require.NoError(t, err)
	require.Len(t, suite.Experiments, 1)
	result, err := umpire3execution.Run(context.Background(), umpire3execution.Request{
		Experiment: suite.Experiments[0], Environment: factory,
	})
	require.NoError(t, err)
	return result
}

func umpire3ProgressScenario(identifier string, target protocol.TargetID, kind protocol.EntityKind) scenario.Scenario {
	return umpire3AssuranceScenario(identifier, target, protocol.PropertyIDEntityProgress,
		kind, protocol.ActionKindProgressEntity)
}

func umpire3AssuranceScenario(
	identifier string,
	target protocol.TargetID,
	property protocol.PropertyID,
	resourceKind protocol.EntityKind,
	action protocol.ActionKind,
) scenario.Scenario {
	return scenario.NewScenario(identifier, target,
		[]scenario.Resource{{Identifier: identifier + "-resource", Kind: resourceKind}},
		scenario.OnePath(scenario.Action(identifier+"-action", action), scenario.Require(property)))
}

func umpire3NexusCancellationScenario(identifier string) scenario.Scenario {
	return umpire3AssuranceScenario(identifier, protocol.TargetIDNexusCancellation,
		protocol.PropertyIDNexusCancellationWonExcludesSuccess,
		protocol.EntityKindNexusOperation, protocol.ActionKindPersistSuccess)
}

func umpire3UpdateScenario(identifier string) scenario.Scenario {
	return umpire3AssuranceScenario(identifier, protocol.TargetIDWorkflowUpdateLifecycle,
		protocol.PropertyIDWorkflowUpdateAcceptedCompletesThroughHistory,
		protocol.EntityKindWorkflowUpdate, protocol.ActionKindCompleteUpdate)
}

func newUmpire3RootEnvironment(t *testing.T, negativeControl bool, variant ...string) umpire3execution.Factory {
	t.Helper()
	return newUmpire3SDKRootFactory(t, negativeControl, variant...)
}
