package goir

import (
	"testing"
 "path/filepath"
 "slices"
 "go.temporal.io/server/model/scalav2/goir/internal/golden"

	"github.com/stretchr/testify/require"
)

// Every witness the model packages' Queries report replays under its own Definition IDs, and none
// replays once one value is bound to another definition.
func TestEveryReportedWitnessReplaysUnderItsDefinitionIDs(t *testing.T) {
	root, err := golden.Root()
 require.NoError(t, err)
 var queries []*Query
 expected := map[string][]string{
  "activity": {"cancel", "cancelRequest", "completion", "nonRetryableFailure", "pauseResume", "retry", "scheduleToStartTimeout", "startToCloseTimeout", "terminate"},
  "nexus-caller": {"asyncCompletion", "asyncFailure", "handlerError", "retry", "scheduleToStartTimeout", "startToCloseTimeout", "syncCompletion"},
 }
 for _, name := range []string{"activity", "nexus-caller"} {
  model, err := Load(filepath.Join(root, "model", "scalav2", "ir", name+".json"))
  require.NoError(t, err)
  realizer, err := NewRealizer(model, DefaultScope)
  require.NoError(t, err)
  var found []string
  for _, receipt := range Check(model, DefaultScope).Receipts {
   if receipt.Subject != QuerySubject || receipt.Kind != Found { continue }
   query, err := realizer.Find(receipt.Key)
   require.NoError(t, err)
   queries = append(queries, query)
   found = append(found, receipt.Key.Name)
  }
  slices.Sort(found)
  require.Equal(t, expected[name], found)
 }
 require.Len(t, queries, 16)
	for _, q := range queries {
		a, err := q.Answer()
		require.NoError(t, err, q.Name)
		require.Equal(t, Outcome(Found), a.Outcome, q.Name)
		require.NoError(t, q.Replay(a), q.Name)

		w := &Trace{Initial: a.Witness.Initial}
		for _, s := range a.Witness.Steps {
			s.Facts = append([]Atom{}, s.Facts...)
			w.Steps = append(w.Steps, s)
		}
		last := &w.Steps[len(w.Steps)-1]
		last.Action.ID = "test.elsewhere.action.other." + last.Action.Value
		rebound := a
		rebound.Witness = w
		require.ErrorContains(t, q.Replay(rebound), "is not the Definition ID", q.Name)
	}
}
