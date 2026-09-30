package umpire_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/go/nexuscaller"
	"go.temporal.io/server/model/go/standaloneactivity"
	"go.temporal.io/server/model/go/umpire"
)

// Every witness the model packages' Queries report replays under its own Definition IDs, and none
// replays once one value is bound to another definition.
func TestEveryReportedWitnessReplaysUnderItsDefinitionIDs(t *testing.T) {
	queries := append(append([]*umpire.Query{}, standaloneactivity.FunctionalQueries...), nexuscaller.FunctionalQueries...)
	for _, q := range queries {
		a, err := q.Answer()
		require.NoError(t, err, q.Name)
		require.Equal(t, umpire.Found, a.Outcome, q.Name)
		require.NoError(t, q.Replay(a), q.Name)

		w := &umpire.Trace{Initial: a.Witness.Initial}
		for _, s := range a.Witness.Steps {
			s.Facts = append([]umpire.Atom{}, s.Facts...)
			w.Steps = append(w.Steps, s)
		}
		last := &w.Steps[len(w.Steps)-1]
		last.Action.ID = "test.elsewhere.action.other." + last.Action.Value
		rebound := a
		rebound.Witness = w
		require.ErrorContains(t, q.Replay(rebound), "is not the Definition ID", q.Name)
	}
}
