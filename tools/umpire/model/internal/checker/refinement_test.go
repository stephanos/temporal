package checker_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpire "go.temporal.io/server/tools/umpire/model/internal/checker"
)

func seesOpened(f string) bool { return f == "opened" }

// seesOpenedOfAbstract reads the door as the abstract one, which sees that it opened.
var seesOpenedOfAbstract = umpire.RefinementSpec{MapState: abstractKeyOf, SeesFact: seesOpened}

// chattyDoorTable locks with a claim that it opened, a fact the abstract door sees.
func chattyDoorTable(name string, extend ...func(*umpire.TableSpec)) *umpire.Table {
	return doorTable(name, append([]func(*umpire.TableSpec){func(s *umpire.TableSpec) {
		s.Evidence = nil
		s.Rows = []umpire.Row{
			rowOf("closed-false", "lock", resultOf("ok", "locked-false", "opened")),
			rowOf("closed-false", "turn-right-true", resultOf("ok", "open-false", "opened", "creaked-true")),
			rowOf("closed-true", "lock", resultOf("ok", "locked-true", "opened")),
			rowOf("closed-true", "turn-right-true", resultOf("ok", "open-true", "opened", "creaked-false")),
			rowOf("open-false", "push", resultOf("ok", "closed-false")),
			rowOf("open-true", "push", resultOf("ok", "closed-true")),
		}
	}}, extend...)...)
}

func refinementError(t *testing.T, err error) *umpire.RefinementError {
	t.Helper()
	var re *umpire.RefinementError
	require.ErrorAs(t, err, &re)
	return re
}

// requireReplaysFromAStart checks that a diagnostic's witness is a path of the refining machine from
// one of its starts.
func requireReplaysFromAStart(t *testing.T, tb *umpire.Table, w *umpire.Trace) {
	t.Helper()
	require.NotNil(t, w)
	require.NoError(t, tb.Replay(w))
	require.Contains(t, tb.Starts, w.Initial.Value)
}

func TestAnInvisibleStutterRefinesAndTheLegacyRuleIsUnchanged(t *testing.T) {
	legacy, err := umpire.RefineTables(doorTable("concrete", refiningAbstract), abstractTable(true), readsAbstract)
	require.NoError(t, err)
	visible, err := umpire.RefineTables(doorTable("concrete", refiningAbstract), abstractTable(true),
		umpire.RefinementSpec{MapState: abstractKeyOf, SeesFact: seesOpened,
			SeesOutcome: func(string) bool { return false }, CoverStarts: true})
	require.NoError(t, err)
	require.Equal(t, legacy.Rows, visible.Rows, "locking records nothing the abstract door sees")
}

func TestAStutterRecordingAVisibleFactIsRejected(t *testing.T) {
	_, err := umpire.RefineTables(chattyDoorTable("chatty", refiningAbstract), abstractTable(true), readsAbstract)
	require.NoError(t, err, "without a projection, a stutter may record any fact")

	m := chattyDoorTable("chatty", refiningAbstract)
	_, err = umpire.RefineTables(m, abstractTable(true), seesOpenedOfAbstract)
	require.EqualError(t, err, "chatty refines abstract: the row 'closed-false-lock' steps from 'closed-false' to "+
		"'locked-false', which both read as 'shut' in abstract, and it records opened, which abstract sees; "+
		"a stutter changes nothing abstract sees, and abstract has no step that records it")
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementVisibleStutter, re.Kind)
	requireReplaysFromAStart(t, m, re.Witness)
	require.Equal(t, []string{"lock"}, actionsOf(re.Witness))
}

func TestAStutterWithAVisibleOutcomeIsRejected(t *testing.T) {
	m := doorTable("concrete", refiningAbstract)
	_, err := umpire.RefineTables(m, abstractTable(true),
		umpire.RefinementSpec{MapState: abstractKeyOf, SeesOutcome: func(o string) bool { return o == "ok" }})
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementVisibleStutter, re.Kind)
	require.ErrorContains(t, err, "and its outcome ok is one abstract sees")
	require.Equal(t, []string{"lock"}, actionsOf(re.Witness))
	requireReplaysFromAStart(t, m, re.Witness)
}

func TestACarrierMustRecordEveryVisibleFact(t *testing.T) {
	_, err := umpire.RefineTables(doorTable("concrete", refiningAbstract), abstractTable(false), readsAbstract)
	require.NoError(t, err, "without a projection, the abstract turn that records nothing carries the concrete one")

	m := doorTable("concrete", refiningAbstract)
	_, err = umpire.RefineTables(m, abstractTable(false), seesOpenedOfAbstract)
	require.ErrorContains(t, err, "neither a step of abstract nor a stutter")
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementUnmatched, re.Kind)
	require.Equal(t, []string{"turn-right-true"}, actionsOf(re.Witness))
	requireReplaysFromAStart(t, m, re.Witness)
}

func TestInitialCorrespondenceCoversEveryProductStart(t *testing.T) {
	twoStarts := func() *umpire.Table { return abstractTable(true, startsAt("shut", "ajar")) }
	_, err := umpire.RefineTables(doorTable("concrete", refiningAbstract), twoStarts(), readsAbstract)
	require.NoError(t, err, "without CoverStarts, a refinement may start in fewer states than its product")

	product := twoStarts()
	_, err = umpire.RefineTables(doorTable("concrete", refiningAbstract), product,
		umpire.RefinementSpec{MapState: abstractKeyOf, CoverStarts: true})
	require.EqualError(t, err, "concrete refines abstract: abstract starts at 'ajar', which no start of concrete reads as")
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementInitial, re.Kind)
	require.Nil(t, re.Witness)
	require.NoError(t, product.Replay(re.ProductWitness))
	require.Equal(t, "ajar", re.ProductWitness.Initial.Value)
	require.Empty(t, re.ProductWitness.Steps)
}

func TestAStartOutsideTheProductStartsHasAReplayableWitness(t *testing.T) {
	m := opensFirst()
	_, err := umpire.RefineTables(m, abstractTable(true), readsAbstract)
	re := refinementError(t, err)
	require.Equal(t, umpire.RefinementInitial, re.Kind)
	requireReplaysFromAStart(t, m, re.Witness)
	require.Equal(t, "open-false", re.Witness.Initial.Value)
}

// A key-level refinement names what the refined table sees beside the map that reads a state as its
// state: a projection with no map refines nothing.
func TestAProjectionWithoutARefinementIsRejected(t *testing.T) {
	_, err := umpire.RefineTables(doorTable("alone"), abstractTable(true), umpire.RefinementSpec{SeesFact: seesOpened})
	require.EqualError(t, err, "alone refines abstract: a refinement names how a state reads as a state of abstract")
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
