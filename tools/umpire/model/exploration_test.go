package model

import (
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestExplorationDeclarationsHaveAnIRForm(t *testing.T) {
	var q modelirspb.Query
	require.NoError(t, protojson.Unmarshal([]byte(`{"name":"sample","exploration":{"name":"sampleSpace","runs":1,"edits":2,"variations":[{"index":0,"choices":[{"name":"bounded","priority":10,"actions":[]}]}],"dropPrefix":true}}`), &q))
}

func TestExplorationRejectsInvalidDomains(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(*modelirspb.Exploration)
		message string
	}{
		{"budget", func(e *modelirspb.Exploration) { e.Runs = 0 }, "positive runs"},
		{"final action", func(e *modelirspb.Exploration) { e.Variations[0].Index = 1 }, "non-prefix index"},
		{"empty domain", func(e *modelirspb.Exploration) { e.Variations[0].Choices = nil }, "empty domain"},
		{"repeated index", func(e *modelirspb.Exploration) { e.Variations = append(e.Variations, e.Variations[0]) }, "repeated/non-prefix"},
		{"repeated name", func(e *modelirspb.Exploration) {
			e.Variations[0].Choices = append(e.Variations[0].Choices, e.Variations[0].Choices[0])
		}, "duplicate alternative"},
		{"unknown action", func(e *modelirspb.Exploration) { e.Variations[0].Choices[0].Actions[0].Action = "missing" }, "missing"},
		{"finite ceiling", func(e *modelirspb.Exploration) { e.Variations[0].Choices = make([]*modelirspb.Alternative, 4097) }, "4096 combinations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := Load("../../../model/ir/nexus-caller.json")
			require.NoError(t, err)
			for _, q := range m.Queries {
				if q.Name == "syncCompletion" {
					tc.change(q.Exploration)
				}
			}
			require.ErrorContains(t, Validate(m), tc.message)
		})
	}
}
