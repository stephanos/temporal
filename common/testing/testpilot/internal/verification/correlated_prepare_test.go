package verification

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
)

// Lean decodes every correlated state as its atom plus its fields and checks each field the way any
// model value is checked, so admission rejects an invalid entry in any of the three field lists.
func TestCorrelatedPrepareRejectsInvalidStateFields(t *testing.T) {
	invalidField := &testpilotspb.ModelValue{Value: "missing-definition"}
	for name, tc := range map[string]struct {
		mutate func(*testpilotspb.CorrelatedContract)
		detail string
	}{
		"initial-state-fields": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.InitialStateFields = []*testpilotspb.ModelValue{invalidField}
			},
			detail: "invalid correlated projection binding",
		},
		"prior-fields": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Transitions[0].PriorFields = []*testpilotspb.ModelValue{invalidField}
			},
			detail: "invalid correlated transition value",
		},
		"state-fields": {
			mutate: func(s *testpilotspb.CorrelatedContract) {
				s.Transitions[0].StateFields = []*testpilotspb.ModelValue{invalidField}
			},
			detail: "invalid correlated transition value",
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
			tc.mutate(c.Correlated)
			_, err := Prepare(c, catalog, view, ceiling, correlated)
			require.Equal(t, &ir.Error{Category: ir.Malformed, Path: "contract", Detail: tc.detail}, err)
		})
	}
}

// An output row is authorized only by a transition whose result equals it as Lean's Result does,
// the resulting state's fields included and in order.
func TestCorrelatedPrepareComparesOutputStateFields(t *testing.T) {
	fields := []*testpilotspb.ModelValue{{DefinitionId: "phase", Value: "ready"}, {DefinitionId: "attempts", Value: "0"}}
	for name, tc := range map[string]struct {
		output   []*testpilotspb.ModelValue
		admitted bool
	}{
		"equal":     {output: fields, admitted: true},
		"absent":    {output: nil},
		"reordered": {output: []*testpilotspb.ModelValue{fields[1], fields[0]}},
		"different": {output: []*testpilotspb.ModelValue{fields[0], {DefinitionId: "attempts", Value: "1"}}},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, view, ceiling, correlated := correlatedFixture(t, 1)
			for _, tr := range c.Correlated.Transitions {
				tr.StateFields = fields
			}
			for _, r := range c.Correlated.ProjectionRules {
				for _, out := range r.Outputs {
					out.StateFields = fields
				}
			}
			c.Correlated.ProjectionRules[0].Outputs[0].StateFields = tc.output
			_, err := Prepare(c, catalog, view, ceiling, correlated)
			if tc.admitted {
				require.NoError(t, err)
				return
			}
			require.Equal(t, &ir.Error{Category: ir.Malformed, Path: "contract", Detail: "projection output absent from transition table"}, err)
		})
	}
}
