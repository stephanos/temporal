package explore

import (
	"testing"

	"github.com/stretchr/testify/require"
	_ "go.temporal.io/api/workflowservice/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

func TestFiniteVariationsArePrioritizedAndRelowered(t *testing.T) {
	m, err := umpiremodel.Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	var q *umpirespb.Query
	for _, declared := range m.Queries {
		if declared.Name == "syncCompletion" {
			q = declared
		}
	}
	require.NotNil(t, q)
	var scenario *umpirespb.Scenario
	for _, s := range m.Scenarios {
		if s.Name == q.Scenario.Name && s.Machine == q.Scenario.Machine {
			scenario = s
		}
	}
	require.NotNil(t, scenario)
	q.Exploration = &umpirespb.Exploration{Name: "deadlines", Runs: 1, Edits: 1, DropPrefix: true, Variations: []*umpirespb.Variation{{Index: 0, Choices: []*umpirespb.Alternative{
		{Name: "default", Actions: scenario.Actions[:1]},
		{Name: "preferred", Priority: 10, Actions: scenario.Actions[:1]},
	}}}}
	plan, err := New(m, "deadlines")
	require.NoError(t, err)
	require.Len(t, plan.Candidates, 2)
	require.Equal(t, "preferred", plan.Candidates[0].Key)
	require.Equal(t, "default", plan.Candidates[1].Key)
	require.Empty(t, plan.Candidates[0].Rejection)
	require.NotEmpty(t, plan.Candidates[0].Bytes)
	again, err := New(m, "deadlines")
	require.NoError(t, err)
	require.Equal(t, plan.Candidates[0].Bytes, again.Candidates[0].Bytes)
	_, err = plan.Reduce(plan.Candidates[0], 0)
	require.Error(t, err, "dropping schedule must not yield an executable completion")
}

func TestAlternativeNamesCannotAliasAnotherTuple(t *testing.T) {
	m, err := umpiremodel.Load("../../../model/ir/nexus-caller.json")
	require.NoError(t, err)
	for _, q := range m.Queries {
		if q.Name == "syncCompletion" {
			q.Exploration = &umpirespb.Exploration{Name: "ambiguous", Runs: 1, Variations: []*umpirespb.Variation{{Index: 0, Choices: []*umpirespb.Alternative{{Name: "a+b"}}}}}
		}
	}
	_, err = New(m, "ambiguous")
	require.ErrorContains(t, err, "alternative name")
}
