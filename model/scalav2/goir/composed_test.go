package goir

// A composition read through the Realizer is the composition Check reads: the same table, by its
// Definition ID and Behavior Fingerprint, the same reasons for having none, and Properties that say
// of a witness's steps what Check's answer says.

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/go/umpire"
)

func TestAComposedReadingIsTheCompositionCheckReads(t *testing.T) {
	for _, name := range []string{"activity", "activity-system"} {
		t.Run(name, func(t *testing.T) {
			m, err := Load(filepath.Join("..", "ir", name+".json"))
			require.NoError(t, err)
			r, err := NewRealizer(m, DefaultScope)
			require.NoError(t, err)
			report := Check(m, DefaultScope)
			read := 0
			for _, x := range report.Receipts {
				owner := x.Key.Owner
				if !isComposition(m, owner) {
					continue
				}
				c, err := r.Composition(owner)
				switch x.Kind {
				case RefinementRejected:
					// The composition has no table, for the reason its receipt gives.
					var rejected *umpire.RefinementError
					require.ErrorAs(t, err, &rejected, owner)
					require.Equal(t, x.Failure, rejected.Kind)
					continue
				case Unsupported, DeclarationError, ResourceLimit:
					continue
				default:
				}
				require.NoError(t, err, owner)
				require.Equal(t, x.Target, c.Table.Family.Target(c.Table.OwnerName()), owner)
				require.Equal(t, x.Fingerprint, c.Table.TargetFingerprint(), owner)
				require.Len(t, c.Table.Rows, x.TableRows, owner)
				if x.Witness != nil {
					require.NoError(t, c.Table.Replay(x.Witness), "%s %s", owner, x.Key.Name)
				}
				read++
			}
			require.Positive(t, read)
		})
	}
}

func isComposition(m *modelirspb.Model, name string) bool {
	for _, c := range m.GetCompositions() {
		if c.GetName() == name {
			return true
		}
	}
	return false
}

// The reading decodes what the composed table keys: a state is the composition's state record, one
// member state per field, and a step record carries that state with the composed outcome and facts
// as strings. Its Properties are the Model's, read as Check reads them: the stale design over the
// queue violates atMostOneActive on the last step of Check's own counterexample, and on no earlier one.
func TestAComposedReadingDecodesStatesStepsAndProperties(t *testing.T) {
	m, err := Load(filepath.Join("..", "ir", "activity-system.json"))
	require.NoError(t, err)
	r, err := NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	c, err := r.Composition("staleOverQueue")
	require.NoError(t, err)
	require.Equal(t, "staleOverQueue", c.Decl.GetName())
	for _, row := range c.Table.Rows {
		source, err := c.State(row.Source)
		require.NoError(t, err)
		require.Equal(t, "temporal.standaloneactivity.OverQueue", source.Type)
		require.Len(t, source.Fields, 2)
		for _, res := range row.Results {
			step, err := c.Step(res)
			require.NoError(t, err)
			target, err := c.State(res.State)
			require.NoError(t, err)
			require.True(t, step.Fields[1].Equal(target), row.Key)
			require.Equal(t, Value{Kind: TextValue, Text: res.Outcome}, step.Fields[0])
			require.Len(t, step.Fields[2].Items, len(res.Facts))
		}
	}
	names := make([]string, len(c.Properties))
	for i, p := range c.Properties {
		names[i] = p.Name
	}
	require.Equal(t, []string{"atMostOneActive", "failedCommitKeepsTheMessage", "notAdmittedWhilePaused", "terminalStays"}, names)

	var counterexample *Receipt
	for _, x := range Check(m, DefaultScope).Receipts {
		if x.Subject == QuerySubject && x.Key.Owner == "staleOverQueue" && x.Property.Name == "atMostOneActive" && x.Kind == Counterexample && x.Monitor == "" {
			counterexample = &x
			break
		}
	}
	require.NotNil(t, counterexample, "Check finds atMostOneActive violated on staleOverQueue")
	property := c.Properties[0]
	state := counterexample.Witness.Initial.Value
	for i, step := range counterexample.Witness.Steps {
		var taken *umpire.Result
		for _, row := range c.Table.RowsFrom(state) {
			for _, res := range row.Results {
				if row.Action == step.Action.Value && res.State == step.State.Value && res.Outcome == step.Outcome.Value {
					taken = &res
				}
			}
		}
		require.NotNil(t, taken, "step %d", i+1)
		require.True(t, property.About(step.Action.Value))
		held, err := property.Holds(state, *taken)
		require.NoError(t, err)
		require.Equal(t, i < len(counterexample.Witness.Steps)-1, held, "step %d", i+1)
		state = step.State.Value
	}
}

// A composition past the scope's ceiling has no reading, and the error is the one Check's
// resource-limit receipt is made of; a name that is no composition names nothing.
func TestAComposedReadingKeepsTheCeilingAndTheNames(t *testing.T) {
	m, err := Load(filepath.Join("..", "ir", "activity.json"))
	require.NoError(t, err)
	scope := DefaultScope
	scope.Compose.States = 3
	r, err := NewRealizer(m, scope)
	require.NoError(t, err)
	_, err = r.Composition("standaloneActivity")
	var limit *umpire.ComposeLimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, "states", limit.Resource)

	r, err = NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	_, err = r.Composition("activityProtocol")
	require.Error(t, err)
	require.NotErrorAs(t, err, &limit)
	_, err = r.Composition("nothing")
	require.Error(t, err)
}
