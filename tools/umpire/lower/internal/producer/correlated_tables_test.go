package producer

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
)

func TestCorrelatedTablesInternCompleteOrderedSemantics(t *testing.T) {
	build := func() *testpilotspb.CorrelatedContract {
		c := &testpilotspb.CorrelatedContract{}
		states, results := map[string]string{}, map[string]string{}
		atom := &testpilotspb.ModelValue{DefinitionId: "state", Value: "same"}
		first := &testpilotspb.ModelValue{DefinitionId: "field", Value: "first"}
		second := &testpilotspb.ModelValue{DefinitionId: "field", Value: "second"}
		state := func(fields ...*testpilotspb.ModelValue) string {
			return internState(c, states, &testpilotspb.CorrelatedState{Atom: atom, Fields: fields})
		}
		c.InitialStateId = state(first, second)
		require.Equal(t, "s1", c.InitialStateId)
		require.Equal(t, "s1", state(first, second))
		require.Equal(t, "s2", state(second, first))
		require.Equal(t, "s3", state(first))
		require.Equal(t, "s4", state())
		result := func(stateID string, facts ...*testpilotspb.ModelValue) string {
			return internResult(c, results, &testpilotspb.CorrelatedResult{Action: &testpilotspb.ModelValue{DefinitionId: "action", Value: "go"}, StateId: stateID, Outcome: &testpilotspb.ModelValue{DefinitionId: "outcome", Value: "ok"}, Facts: facts})
		}
		require.Equal(t, "r1", result("s1", first, second))
		require.Equal(t, "r1", result("s1", first, second))
		require.Equal(t, "r2", result("s2", first, second))
		require.Equal(t, "r3", result("s1", second, first))
		c.Transitions = []*testpilotspb.CorrelatedTransition{{PriorStateId: "s4", ResultId: "r1"}, {PriorStateId: "s1", ResultId: "r2"}}
		c.ProjectionRules = []*testpilotspb.CorrelatedProjectionRule{{Kind: "evidence", ResultIds: []string{result("s1", second, first), result("s1", first, second), result("s1", first, second)}}}
		require.Equal(t, []string{"r3", "r1", "r1"}, c.ProjectionRules[0].ResultIds)
		require.Len(t, c.States, 4)
		require.Len(t, c.Results, 3)
		require.True(t, proto.Equal(atom, c.States[0].Atom))
		require.True(t, proto.Equal(atom, c.States[1].Atom))
		return c
	}
	require.True(t, proto.Equal(build(), build()), "local IDs and table order are deterministic")
}
