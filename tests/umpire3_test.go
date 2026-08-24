package tests

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/parallelsuite"
	umpire3execution "go.temporal.io/server/tools/umpire3/execution"
	"go.temporal.io/server/tools/umpire3/regression"
	"go.temporal.io/server/tools/umpire3/scenario"
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
	result := evaluateUmpire3Regression(t, umpire3WorkflowProgressRegression("independent-history"), "", false)
	require.Equal(t, umpire3execution.ClaimConforming, result.Claim.Kind, result.Claim.Reason)
	require.Equal(t, umpire3execution.EvidenceProfileDualHistory, result.Environment.EvidenceProfile)
	identities := umpire3ObservationSourceIdentities(result.Observations)
	require.Len(t, identities, 2)
	require.NotEqual(t, identities[0], identities[1])
	require.True(t, slices.ContainsFunc(identities, func(identity string) bool {
		return !strings.HasSuffix(identity, "/history-service")
	}), identities)
	require.True(t, slices.ContainsFunc(identities, func(identity string) bool {
		return strings.HasSuffix(identity, "/history-service")
	}), identities)
}

func TestUmpire3WorkflowTaskOwnershipFencingUsesStaleTaskToken(t *testing.T) {
	runUmpire3Regression(t, scenario.FoundationOwnershipFencingScenario(
		"umpire3-workflow-task-ownership-fencing",
		[]scenario.Resource{scenario.WorkflowTask("workflow-task")},
		scenario.OnePath(
			scenario.FenceWorkflowOwner("fence-owner"),
			scenario.RequireWorkflowTaskOwnershipFencing(),
		),
	))
}

func umpire3ObservationSourceIdentities(observations []umpire3execution.Observation) []string {
	sources := make([]string, len(observations))
	for index, observation := range observations {
		sources[index] = observation.SourceIdentity
	}
	return sources
}

func (s *Umpire3TestSuite) TestPlanAndDriveWorkflowToCompletion() {
	runUmpire3Regression(s.T(), umpire3WorkflowProgressRegression("PlanAndDriveWorkflowToCompletion"))
}

// TestPlanAndDriveKitchenSinkWorkflow is the same compile -> drive -> judge loop, but records the
// participant-program migration of the kitchensink workflow as a distinct regression.
func (s *Umpire3TestSuite) TestPlanAndDriveKitchenSinkWorkflow() {
	runUmpire3Regression(s.T(), umpire3WorkflowProgressRegression("PlanAndDriveKitchenSinkWorkflow"))
}

// TestPlanAndDriveNexusOperationCHASM retains the CHASM Nexus semantic outcome while Umpire3
// qualifies it through its own catalog, compiler, environment, and evidence graph.
func (s *Umpire3TestSuite) TestPlanAndDriveNexusOperationCHASM() {
	runUmpire3Regression(s.T(), umpire3NexusClosureRegression("PlanAndDriveNexusOperationCHASM",
		scenario.ScheduleOperation("schedule"),
		scenario.WorkerReturnsSuccess("success"),
		scenario.PersistSuccess("persist"),
	), "chasm")
}

// TestPlanAndDriveKitchenSinkNexusOperation keeps the independently named kitchensink path in the
// root functional suite while sharing only Umpire3's normal semantic execution seam.
func (s *Umpire3TestSuite) TestPlanAndDriveKitchenSinkNexusOperation() {
	runUmpire3Regression(s.T(), umpire3NexusClosureRegression("PlanAndDriveKitchenSinkNexusOperation",
		scenario.ScheduleOperation("schedule"),
		scenario.WorkerReturnsSuccess("success"),
		scenario.PersistSuccess("persist"),
	), "chasm")
}

func runUmpire3Regression(t *testing.T, authored scenario.Scenario, variant ...string) {
	t.Helper()
	regression.RequireRegression(t, authored, regression.WithEnvironment(newUmpire3RootEnvironment(t, false, variant...)))
}

func requireUmpire3Violation(t *testing.T, authored scenario.Scenario, variant ...string) {
	t.Helper()
	regression.RequireRegression(t, authored,
		regression.WithEnvironment(newUmpire3RootEnvironment(t, false, variant...)),
		regression.ExpectViolation())
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

func umpire3WorkflowProgressRegression(identifier string) scenario.Scenario {
	return scenario.FoundationDeliverySafetyScenario(identifier,
		[]scenario.Resource{scenario.Workflow(identifier + "-workflow")},
		scenario.OnePath(
			scenario.ProgressEntity(identifier+"-progress"),
			scenario.RequireEntityProgress(),
		))
}

func umpire3NexusClosureRegression(identifier string, actions ...scenario.Term) scenario.Scenario {
	actions = append(actions, scenario.RequireNexusOperationClosure())
	return scenario.FeatureNexusScenario(identifier,
		[]scenario.Resource{
			scenario.NexusOperation(identifier + "-operation"),
			scenario.Workflow(identifier + "-workflow"),
		},
		scenario.OnePath(actions...))
}

func umpire3NexusProgressRegression(identifier string, actions ...scenario.Term) scenario.Scenario {
	actions = append(actions, scenario.RequireNexusOperationProgress())
	return scenario.FeatureNexusProgressScenario(identifier,
		[]scenario.Resource{
			scenario.NexusOperation(identifier + "-operation"),
			scenario.Workflow(identifier + "-workflow"),
		},
		scenario.OnePath(actions...))
}

func umpire3NexusRPCFault(
	identifier string,
	build func(string, ...scenario.FaultOption) scenario.FaultIntent,
	options ...scenario.FaultOption,
) scenario.FaultIntent {
	defaults := []scenario.FaultOption{
		scenario.OnEndpoints("umpire3-nexus-endpoint"),
		scenario.OnTaskQueues("umpire3-nexus-task-queue"),
		scenario.OnServices("nexus"),
		scenario.OnRoutes("/service/operation"),
		scenario.OnAttempts(1),
		scenario.AtOccurrence(1, 1),
	}
	return build(identifier, append(defaults, options...)...)
}

type umpire3ScenarioAction func(string, ...scenario.ActionOption) scenario.Term

func umpire3NexusActionRegression(identifier string, action umpire3ScenarioAction) scenario.Scenario {
	return scenario.FeatureNexusScenario(identifier,
		[]scenario.Resource{scenario.NexusOperation(identifier + "-operation")},
		scenario.OnePath(
			action(identifier+"-action"),
			scenario.RequireNexusOperationClosure(),
		))
}

func newUmpire3RootEnvironment(t *testing.T, negativeControl bool, variant ...string) umpire3execution.Factory {
	t.Helper()
	return newUmpire3SDKRootFactory(t, negativeControl, variant...)
}
