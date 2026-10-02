package umpire_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/model/go/umpire"
)

func seesOpened(f doorFact) bool { return f == opened{} }

// chattyDoor locks with a claim that it opened, a fact the abstract door sees.
func chattyDoor(name string) *umpire.Machine[door, doorOutcome, doorFact] {
	return umpire.NewMachine[door, doorOutcome, doorFact]("test.door", name).
		Starts(door{Phase: closed}).
		Ends(func(d door) bool { return d.Phase == locked }).
		Step1(turn, turnStep).Step0(push, pushStep).
		Step0(lock, func(d door) []doorStep {
			if d.Phase != closed {
				return nil
			}
			return []doorStep{{Outcome: ok, State: door{Phase: locked, Oiled: d.Oiled}, Facts: []doorFact{opened{}}}}
		})
}

func refinementError(t *testing.T, err error) *umpire.RefinementError {
	t.Helper()
	var re *umpire.RefinementError
	require.ErrorAs(t, err, &re)
	return re
}

// requireReplaysFromAStart checks that a diagnostic's witness is a path of the refining machine from
// one of its starts.
func requireReplaysFromAStart(t *testing.T, m umpire.Model, w *umpire.Trace) {
	t.Helper()
	tb := tableOf(t, m)
	require.NotNil(t, w)
	require.NoError(t, tb.Replay(w))
	require.Contains(t, tb.Starts, w.Initial.Value)
}

func TestAnInvisibleStutterRefinesAndTheLegacyRuleIsUnchanged(t *testing.T) {
	legacy, err := newDoor("concrete").Refines(abstractMachine(true), abstractOf).Refinement()
	require.NoError(t, err)
	visible, err := newDoor("concrete").Refines(abstractMachine(true), abstractOf).
		Visible(seesOpened).VisibleOutcomes(func(doorOutcome) bool { return false }).CoversStarts().Refinement()
	require.NoError(t, err)
	require.Equal(t, legacy.Rows, visible.Rows, "locking records nothing the abstract door sees")
}

func TestAStutterRecordingAVisibleFactIsRejected(t *testing.T) {
	_, err := chattyDoor("chatty").Refines(abstractMachine(true), abstractOf).Refinement()
	require.NoError(t, err, "without a projection, a stutter may record any fact")

	m := chattyDoor("chatty").Refines(abstractMachine(true), abstractOf).Visible(seesOpened)
	_, err = m.Refinement()
	require.EqualError(t, err, "chatty refines abstract: the row 'closed-false-lock' steps from 'closed-false' to "+
		"'locked-false', which both read as 'shut' in abstract, and it records opened, which abstract sees; "+
		"a stutter changes nothing abstract sees, and abstract has no step that records it")
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementVisibleStutter, re.Kind)
	requireReplaysFromAStart(t, m, re.Witness)
	require.Equal(t, []string{"lock"}, actionsOf(re.Witness))
}

func TestAStutterWithAVisibleOutcomeIsRejected(t *testing.T) {
	m := newDoor("concrete").Refines(abstractMachine(true), abstractOf).
		VisibleOutcomes(func(o doorOutcome) bool { return o == ok })
	_, err := m.Refinement()
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementVisibleStutter, re.Kind)
	require.ErrorContains(t, err, "and its outcome ok is one abstract sees")
	require.Equal(t, []string{"lock"}, actionsOf(re.Witness))
	requireReplaysFromAStart(t, m, re.Witness)
}

func TestACarrierMustRecordEveryVisibleFact(t *testing.T) {
	_, err := newDoor("concrete").Refines(abstractMachine(false), abstractOf).Refinement()
	require.NoError(t, err, "without a projection, the abstract turn that records nothing carries the concrete one")

	m := newDoor("concrete").Refines(abstractMachine(false), abstractOf).Visible(seesOpened)
	_, err = m.Refinement()
	require.ErrorContains(t, err, "neither a step of abstract nor a stutter")
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementUnmatched, re.Kind)
	require.Equal(t, []string{"turn-right-true"}, actionsOf(re.Witness))
	requireReplaysFromAStart(t, m, re.Witness)
}

func TestInitialCorrespondenceCoversEveryProductStart(t *testing.T) {
	twoStarts := func() *umpire.Machine[abstractDoor, doorOutcome, abstractFact] {
		return abstractMachine(true).Starts(abstractDoor{shut}, abstractDoor{ajar})
	}
	_, err := newDoor("concrete").Refines(twoStarts(), abstractOf).Refinement()
	require.NoError(t, err, "without CoversStarts, a refinement may start in fewer states than its product")

	product := twoStarts()
	_, err = newDoor("concrete").Refines(product, abstractOf).CoversStarts().Refinement()
	require.EqualError(t, err, "concrete refines abstract: abstract starts at 'ajar', which no start of concrete reads as")
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementInitial, re.Kind)
	require.Nil(t, re.Witness)
	require.NoError(t, tableOf(t, product).Replay(re.ProductWitness))
	require.Equal(t, "ajar", re.ProductWitness.Initial.Value)
	require.Empty(t, re.ProductWitness.Steps)
}

func TestAStartOutsideTheProductStartsHasAReplayableWitness(t *testing.T) {
	m := umpire.NewMachine[door, doorOutcome, doorFact]("test.door", "opensFirst").
		Starts(door{Phase: open}).Step0(push, pushStep).
		Refines(abstractMachine(true), abstractOf)
	_, err := m.Refinement()
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementInitial, re.Kind)
	requireReplaysFromAStart(t, m, re.Witness)
	require.Equal(t, "open-false", re.Witness.Initial.Value)
}

func TestAProjectionWithoutARefinementIsRejected(t *testing.T) {
	_, err := newDoor("alone").Visible(seesOpened).Table()
	require.EqualError(t, err, "alone: the machine names what a refined machine sees, and refines none")
}

func TestRefineTablesChecksKeyOnlyTables(t *testing.T) {
	product := umpire.NewTable(umpire.TableSpec{Machine: "product", Family: "test.keys",
		States: []string{"idle", "done"}, Actions: []string{"finish"}, Outcomes: []string{"ok"},
		Facts: []string{"finished"}, Starts: []string{"idle"}, Ends: []string{"done"},
		Rows: []umpire.Row{{Key: "idle-finish", Source: "idle", Action: "finish",
			Results: []umpire.Result{{Outcome: "ok", State: "done", Facts: []string{"finished"}}}}}})
	detail := umpire.NewTable(umpire.TableSpec{Machine: "detail", Family: "test.keys",
		States: []string{"idle", "working", "done"}, Actions: []string{"begin", "finish"}, Outcomes: []string{"ok"},
		Facts: []string{"finished"}, Starts: []string{"idle"}, Ends: []string{"done"},
		Rows: []umpire.Row{
			{Key: "idle-begin", Source: "idle", Action: "begin",
				Results: []umpire.Result{{Outcome: "ok", State: "working", Facts: []string{"finished"}}}},
			{Key: "working-finish", Source: "working", Action: "finish",
				Results: []umpire.Result{{Outcome: "ok", State: "done", Facts: []string{"finished"}}}},
		}})
	read := func(s string) (string, error) {
		if s == "working" {
			return "idle", nil
		}
		return s, nil
	}
	ref, err := umpire.RefineTables(detail, product, umpire.RefinementSpec{MapState: read})
	require.NoError(t, err)
	require.Nil(t, ref.Rows[0].Product, "beginning is a stutter")

	_, err = umpire.RefineTables(detail, product, umpire.RefinementSpec{MapState: read,
		SeesFact: func(string) bool { return true }})
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementVisibleStutter, re.Kind, "beginning claims the finish the product sees")
	require.NoError(t, detail.Replay(re.Witness))
}

func actionsOf(w *umpire.Trace) []string {
	var out []string
	for _, s := range w.Steps {
		out = append(out, s.Action.Value)
	}
	return out
}
