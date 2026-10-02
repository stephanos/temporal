//go:build test_dep && integration

package tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tests/testcore"
	testpilotcore "go.temporal.io/server/tests/testcore/testpilot"
)

func TestTestpilotScalaNexus(t *testing.T) {
	for _, value := range testpilotcore.NexusImplementationSwitch() {
		t.Run(value.Name, func(t *testing.T) {
			var options []testcore.TestOption
			for _, setting := range value.Settings {
				options = append(options, testcore.WithDynamicConfig(setting.Setting, setting.Value))
			}
			env := newTestpilotTestEnvironment(t, options...)
			for _, query := range nexusCallerQueries[:2] {
				t.Run(query.name, func(t *testing.T) {
					fixture := scalaFixture(t, "nexus-caller", "temporal.nexus.caller", "nexusProtocol", "nexusCallerTests", query.name)
					lives := make([]testpilotLiveCase, 2)
					endpoints := make([]string, len(lives))
					for i := range lives {
						name := "scala-nexus-" + uuid.NewString()
						lives[i] = bindCase(t, env, fixture.Source, CaseBinding{Identity: name, Namespace: name, TaskQueue: name, NexusEndpoint: name, CreateEndpoint: true, DynamicConfig: value.Configuration()})
						endpoints[i] = name
					}
					require.NotEqual(t, lives[0].prepared.Identity().Bindings, lives[1].prepared.Identity().Bindings)
					ids, learned := map[string]bool{}, map[string]bool{}
					for range 2 {
						for i, result := range runScalaCases(t, env, fixture, lives) {
							property, status := "syncSucceeds", testpilot.PropertySatisfied
							if query.name == "asyncCompletion" {
								property, status = "completionSucceeds", testpilot.PropertyInconclusive
							}
							requireScalaAssessment(t, fixture, lives[i], result, property, status)
							requireNexusCallerVerdict(t, query, result.run, result.verdict, endpoints[i])
							require.NotContains(t, ids, result.run.GetRunId())
							ids[result.run.GetRunId()] = true
							described, err := lives[i].client.DescribeWorkflowExecution(t.Context(), result.run.GetRunId(), "")
							require.NoError(t, err)
							runID := described.GetWorkflowExecutionInfo().GetExecution().GetRunId()
							_, err = uuid.Parse(runID)
							require.NoError(t, err)
							require.NotContains(t, learned, runID)
							learned[runID] = true
							for other := range lives {
								if other == i {
									continue
								}
								_, err := lives[other].client.DescribeWorkflowExecution(t.Context(), result.run.GetRunId(), runID)
								var missing *serviceerror.NotFound
								require.ErrorAs(t, err, &missing)
							}
						}
					}
				})
			}
		})
	}
}
