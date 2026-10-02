package lower

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

func MigrationFixture(m *modelirspb.Model, query string, identity cp.Identity) (*umpiremodel.Query, *cp.Realization, error) {
	p, err := NewProducer(m)
	if err != nil {
		return nil, nil, err
	}
	a, standing, err := p.ask(query)
	if err != nil {
		return nil, nil, err
	}
	if standing != Lowered {
		return nil, nil, fmt.Errorf("fixture %s: %s", query, standing)
	}
	l, problems := p.check(a, identity)
	if err := errors.Join(problems...); err != nil {
		return nil, nil, err
	}
	return l.query, l.realization, nil
}

func MigrationComparativeModel(t *testing.T, m *modelirspb.Model) {
	t.Helper()
	expectedKinds := []string{"temporal.nexus.caller.evidence.started", "temporal.nexus.caller.evidence.completed", "temporal.nexus.caller.evidence.failed", "temporal.nexus.caller.evidence.canceled", "temporal.nexus.caller.evidence.timedOut"}
	require.Len(t, m.GetRealizations(), 1)
	require.Equal(t, "asyncNexus", m.GetRealizations()[0].GetName())
	var exhaustive []string
	for _, e := range m.GetRealizations()[0].GetEvidence() {
		if e.GetExhaustive() {
			exhaustive = append(exhaustive, e.GetId())
			e.Exhaustive = false
		}
	}
	require.Equal(t, expectedKinds, exhaustive)
	closed := 0
	for _, script := range m.GetRealizations()[0].GetScripts() {
		for _, item := range script.GetItems() {
			if script.GetId() == "controller" && item.GetCommand().GetId() == "history" {
				require.Equal(t, expectedKinds, item.GetCommand().GetCloses())
				item.GetCommand().Closes = nil
				closed++
			}
		}
	}
	require.Equal(t, 1, closed)
}
