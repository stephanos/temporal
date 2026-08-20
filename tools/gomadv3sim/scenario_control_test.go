package gomadv3sim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScenarioChoicePlanCanonicalRoundTripAndDetachedInput(t *testing.T) {
	overrides := []ScenarioChoiceOverride{{
		Ordinal: 0, ID: "route", Occurrence: 1, Alternatives: []string{"alpha", "beta"}, Selected: 0,
	}}
	plan, err := NewScenarioChoicePlan(overrides)
	require.NoError(t, err)
	overrides[0].Alternatives[0] = "changed"

	encoded, err := EncodeScenarioChoicePlan(plan)
	require.NoError(t, err)
	decoded, err := DecodeScenarioChoicePlan(encoded)
	require.NoError(t, err)
	require.Equal(t, plan, decoded)
	require.Equal(t, []string{"alpha", "beta"}, decoded.Overrides[0].Alternatives)

	_, err = DecodeScenarioChoicePlan(append(encoded, '\n'))
	require.Error(t, err)
}

func TestScenarioChoicePlanRejectsInvalidRankBoundOverrides(t *testing.T) {
	tests := []struct {
		name      string
		overrides []ScenarioChoiceOverride
	}{
		{name: "duplicate ordinal", overrides: []ScenarioChoiceOverride{
			{Ordinal: 1, ID: "first", Occurrence: 1, Alternatives: []string{"a", "b"}, Selected: 0},
			{Ordinal: 1, ID: "second", Occurrence: 1, Alternatives: []string{"a", "b"}, Selected: 1},
		}},
		{name: "zero occurrence", overrides: []ScenarioChoiceOverride{{Ordinal: 0, ID: "route", Alternatives: []string{"a", "b"}, Selected: 0}}},
		{name: "non-choice", overrides: []ScenarioChoiceOverride{{Ordinal: 0, ID: "route", Occurrence: 1, Alternatives: []string{"only"}, Selected: 0}}},
		{name: "rank overflow", overrides: []ScenarioChoiceOverride{{Ordinal: 0, ID: "route", Occurrence: 1, Alternatives: []string{"a", "b"}, Selected: 2}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewScenarioChoicePlan(test.overrides)
			require.Error(t, err)
		})
	}
}
