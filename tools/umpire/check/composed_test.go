package check

// A composition read through the Realizer is the composition Check reads: the same table, by its
// Definition ID and Behavior Fingerprint, the same reasons for having none, and Properties that say
// of a witness's steps what Check's answer says.

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
)

func TestAComposedReadingIsTheCompositionCheckReads(t *testing.T) {
	for _, name := range []string{"activity-standalone", "activity-standalone-record"} {
		t.Run(name, func(t *testing.T) {
			m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", name+".json"))
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
					var rejected *RefinementError
					require.ErrorAs(t, err, &rejected, owner)
					require.Equal(t, x.Failure, rejected.Kind)
					table, err := r.TransitionTable(owner)
					require.NoError(t, err, "a refinement rejection does not prevent constructing transitions")
					require.NotEmpty(t, table.Reachable)
					continue
				case Unsupported, DeclarationError, ResourceLimit:
					continue
				default:
				}
				require.NoError(t, err, owner)
				require.Equal(t, x.Target, c.Table.Family.Target(c.Table.OwnerName()), owner)
				require.Equal(t, x.Fingerprint, c.Table.TargetFingerprint(), owner)
				require.Len(t, c.Table.Rows, x.TableRows, owner)
				table, err := r.TransitionTable(owner)
				require.NoError(t, err, owner)
				require.Equal(t, c.Table.TargetFingerprint(), table.TargetFingerprint(), owner)
				require.Equal(t, c.Table.Unknown, table.Unknown, owner)
				if x.Witness != nil {
					require.NoError(t, c.Table.Replay(x.Witness), "%s %s", owner, x.Key.Name)
				}
				read++
			}
			require.Positive(t, read)
		})
	}
}

func isComposition(m *umpirespb.Model, name string) bool {
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
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", "activity-standalone-record.json"))
	require.NoError(t, err)
	r, err := NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	c, err := r.Composition("trustingRecordOverQueue")
	require.NoError(t, err)
	require.Equal(t, "trustingRecordOverQueue", c.Decl.GetName())
	for _, row := range c.Table.Rows {
		source, err := c.State(row.Source)
		require.NoError(t, err)
		require.Equal(t, "temporal.features.activity.standalone.system.OverQueue", source.Type)
		require.Len(t, source.Fields, 2)
		for _, res := range row.Results {
			step, err := c.Step(res)
			require.NoError(t, err)
			target, err := c.State(res.State)
			require.NoError(t, err)
			require.True(t, step.Fields[1].Equal(target), row.Key)
			require.Equal(t, interp.Value{Kind: interp.TextValue, Text: res.Outcome}, step.Fields[0])
			require.Len(t, step.Fields[2].Items, len(res.Facts))
		}
	}
	names := make([]string, len(c.Properties))
	for i, p := range c.Properties {
		names[i] = p.Name
	}
	require.Equal(t, []string{"atMostOneActive", "failedCommitKeepsTheMessage", "trustingRecordOverQueue.pausedIsNotDispatched", "trustingRecordOverQueue.terminalStatesAreFinal"}, names)

	var counterexample *Receipt
	for _, x := range Check(m, DefaultScope).Receipts {
		if x.Subject == QuerySubject && x.Key.Owner == "trustingRecordOverQueue" && x.Property.Name == "atMostOneActive" && x.Kind == Counterexample && x.Monitor == "" {
			counterexample = &x
			break
		}
	}
	require.NotNil(t, counterexample, "Check finds atMostOneActive violated on trustingRecordOverQueue")
	property := c.Properties[0]
	state := counterexample.Witness.Initial.Value
	for i, step := range counterexample.Witness.Steps {
		var taken *interp.Result
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
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", "activity-standalone.json"))
	require.NoError(t, err)
	scope := DefaultScope
	scope.Compose.States = 3
	r, err := NewRealizer(m, scope)
	require.NoError(t, err)
	_, err = r.Composition("standaloneActivity")
	var limit *ComposeLimitError
	require.ErrorAs(t, err, &limit)
	require.Equal(t, "states", limit.Resource)
	_, err = r.TransitionTable("standaloneActivity")
	require.ErrorAs(t, err, &limit)
	require.Equal(t, "states", limit.Resource)

	r, err = NewRealizer(m, DefaultScope)
	require.NoError(t, err)
	_, err = r.Composition("activitySystem")
	require.Error(t, err)
	require.NotErrorAs(t, err, &limit)
	_, err = r.Composition("nothing")
	require.Error(t, err)
	_, err = r.TransitionTable("nothing")
	require.ErrorContains(t, err, "no machine or composition nothing")
	table, err := r.TransitionTable("activitySystem")
	require.NoError(t, err)
	require.Equal(t, r.Machine("activitySystem").Table.TargetFingerprint(), table.TargetFingerprint())
}
