package verification

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

func TestCorrelatedPrepareRejectsInvalidStateFields(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
	c.Correlated.States[0].Fields = []*testpilotspb.ModelValue{{Value: "missing-definition"}}
	_, err := Prepare(c, catalog, view, ceiling, correlated)
	require.Equal(t, &ir.Error{Category: ir.Malformed, Path: "contract", Detail: "invalid or duplicate correlated state"}, err)
}

func TestCorrelatedPrepareNormalizedTables(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.CorrelatedContract){
		"duplicate-state-id": func(s *testpilotspb.CorrelatedContract) { s.States = append(s.States, proto.CloneOf(s.States[0])) },
		"conflicting-state-id": func(s *testpilotspb.CorrelatedContract) {
			state := proto.CloneOf(s.States[0])
			state.Fields = []*testpilotspb.ModelValue{{DefinitionId: "phase", Value: "other"}}
			s.States = append(s.States, state)
		},
		"duplicate-complete-state": func(s *testpilotspb.CorrelatedContract) {
			state := proto.CloneOf(s.States[0])
			state.StateId = "alias"
			s.States = append(s.States, state)
		},
		"duplicate-result-id": func(s *testpilotspb.CorrelatedContract) { s.Results = append(s.Results, proto.CloneOf(s.Results[0])) },
		"duplicate-complete-result": func(s *testpilotspb.CorrelatedContract) {
			result := proto.CloneOf(s.Results[0])
			result.ResultId = "alias"
			s.Results = append(s.Results, result)
		},
		"conflicting-result-id": func(s *testpilotspb.CorrelatedContract) {
			result := proto.CloneOf(s.Results[0])
			result.Outcome.Value = "other"
			s.Results = append(s.Results, result)
		},
		"dangling-initial":           func(s *testpilotspb.CorrelatedContract) { s.InitialStateId = "missing" },
		"dangling-result-state":      func(s *testpilotspb.CorrelatedContract) { s.Results[0].StateId = "missing" },
		"dangling-prior":             func(s *testpilotspb.CorrelatedContract) { s.Transitions[0].PriorStateId = "missing" },
		"dangling-transition-result": func(s *testpilotspb.CorrelatedContract) { s.Transitions[0].ResultId = "missing" },
		"dangling-projection-result": func(s *testpilotspb.CorrelatedContract) { s.ProjectionRules[0].ResultIds = []string{"missing"} },
		"unauthorized-projection-result": func(s *testpilotspb.CorrelatedContract) {
			result := proto.CloneOf(s.Results[0])
			result.ResultId = "unreferenced"
			result.Outcome.Value = "unreferenced"
			s.Results = append(s.Results, result)
			s.ProjectionRules[0].ResultIds = []string{result.ResultId}
		},
		"duplicate-transition": func(s *testpilotspb.CorrelatedContract) {
			s.Transitions = append(s.Transitions, proto.CloneOf(s.Transitions[0]))
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
			mutate(c.Correlated)
			_, err := Prepare(c, catalog, view, ceiling, correlated)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, ir.Malformed, diagnostic.Category)
		})
	}
}

func TestCorrelatedNormalizedExpansionLimits(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Contract, *testpilotspb.ContractLimits){
		"result-count": func(c *testpilotspb.Contract, limits *testpilotspb.ContractLimits) {
			limits.MaxTransitions = int64(len(c.Correlated.Transitions))
			result := proto.CloneOf(c.Correlated.Results[0])
			result.ResultId, result.Outcome.Value = "extra", "extra"
			c.Correlated.Results = append(c.Correlated.Results, result)
		},
		"expanded-state-bytes": func(c *testpilotspb.Contract, _ *testpilotspb.ContractLimits) {
			c.Correlated.States[0].Fields = []*testpilotspb.ModelValue{{DefinitionId: "large", Value: strings.Repeat("x", 20000)}}
		},
		"expanded-projection-bytes": func(c *testpilotspb.Contract, _ *testpilotspb.ContractLimits) {
			c.Correlated.Results[0].Facts = []*testpilotspb.ModelValue{{DefinitionId: "large", Value: strings.Repeat("x", 20000)}}
			c.Correlated.ProjectionRules[0].ResultIds = []string{"request", "request", "request", "request", "request"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
			mutate(c, ceiling)
			correlated.MaxSemanticTransitions = min(correlated.MaxSemanticTransitions, ceiling.MaxTransitions)
			_, err := Prepare(c, catalog, view, ceiling, correlated)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, ir.LimitExceeded, diagnostic.Category)
		})
	}
}

func TestCorrelatedPrepareCompleteStateCeiling(t *testing.T) {
	c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
	state := proto.CloneOf(c.Correlated.States[0])
	state.StateId, state.Fields = "other", []*testpilotspb.ModelValue{{DefinitionId: "phase", Value: "other"}}
	c.Correlated.States = append(c.Correlated.States, state)
	ceiling.MaxStates = 1
	_, err := Prepare(c, catalog, view, ceiling, correlated)
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, ir.LimitExceeded, diagnostic.Category)
}
