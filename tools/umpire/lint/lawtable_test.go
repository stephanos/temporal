package lint

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/model"
)

// lawTablesOf reads the tables of an IR file of model/ir with the law sidecar beside it.
func lawTablesOf(t *testing.T, path string) *Result {
	t.Helper()
	m, err := Read(path, unrealizedByMachine, Options{})
	require.NoError(t, err)
	require.NotNil(t, m.Laws)
	tables, _, err := m.holes()
	require.NoError(t, err)
	return &Result{File: path, Tables: tables, Laws: m.compositionLaws()}
}

func tableOf(t *testing.T, r *Result, machine string) *Table {
	t.Helper()
	i := slices.IndexFunc(r.Tables, func(t *Table) bool { return t.Machine == machine })
	require.GreaterOrEqual(t, i, 0, machine)
	return r.Tables[i]
}

func lawOf(t *testing.T, lt *LawTable, claim string) LawView {
	t.Helper()
	require.NotNil(t, lt)
	i := slices.IndexFunc(lt.Laws, func(lv LawView) bool { return lv.Claim.Name == claim })
	require.GreaterOrEqual(t, i, 0, claim)
	return lt.Laws[i]
}

func TestLawsPinTheCellsOfTheirCapabilitiesActions(t *testing.T) {
	r := lawTablesOf(t, "../../../model/ir/activity.json")
	product := tableOf(t, r, "activityProduct")

	// The pair law pins the paused cells of the dispatching action as MUST NOT, read from the
	// sidecar with what it promises, beside the cell the step function wrote.
	paused := lawOf(t, product.Laws, "activityProduct.pausedIsNotDispatched")
	require.Equal(t, MustNot, paused.Modality)
	require.Equal(t, []string{"Pausable", "Pollable"}, paused.Claim.Capabilities)
	require.Equal(t, "attemptStart", paused.Claim.Actions["Pollable.dispatch"])
	require.Contains(t, paused.Entry.Promises, "no step from paused lands in running")
	require.NotEmpty(t, paused.Entry.DoesNotPromise)
	i := slices.IndexFunc(paused.Pins, func(c LawCell) bool { return c.Class == "attemptStart" })
	require.GreaterOrEqual(t, i, 0)
	require.Equal(t, "paused", paused.Pins[i].Label)
	// The step function is silent there, a worker's poll it does not answer; the law says MUST NOT.
	require.Equal(t, Silent, paused.Pins[i].Modality)
	require.Equal(t, []string{"Pollable.dispatch"}, paused.Pins[i].Fields)
	for _, c := range paused.Pins {
		require.Contains(t, []string{"attemptStart", "control-pause", "control-unpause"}, c.Class, "only its capabilities' actions")
	}

	// Closable names no action: its laws pin the terminal cells of every class.
	closed := lawOf(t, product.Laws, "activityProduct.closedIsRejectedUniformly")
	require.Equal(t, MustNot, closed.Modality)
	require.Len(t, closed.Pins, 1)
	require.Equal(t, EveryClass, closed.Pins[0].Class)
	for _, phase := range []string{"completed", "failed", "canceled", "terminated", "timedOut"} {
		require.Contains(t, closed.Pins[0].Label, phase)
	}

	// The cells of a capability's action no law pins are listed: a poll of a scheduled activity.
	j := slices.IndexFunc(product.Laws.Unpinned, func(c LawCell) bool { return c.Class == "attemptStart" && c.Modality == May })
	require.GreaterOrEqual(t, j, 0)
	require.Contains(t, product.Laws.Unpinned[j].Label, "scheduled")
	for _, c := range product.Laws.Unpinned {
		require.NotEqual(t, "paused", c.Label, "a paused cell of the actions is pinned")
	}

	// A law never adds, removes or rewrites a row: the table reads as it does with no sidecar.
	bare := read(t, activityIR)
	tables, _, err := bare.holes()
	require.NoError(t, err)
	for _, table := range tables {
		require.Nil(t, table.Laws)
		require.Equal(t, table.Rules, tableOf(t, r, table.Machine).Rules)
	}
}

func TestProtocolMarksTheProductsLawsInherited(t *testing.T) {
	protocol := tableOf(t, lawTablesOf(t, "../../../model/ir/activity.json"), "activityProtocol")

	inherited := lawOf(t, protocol.Laws, "activityProduct.pausedIsNotDispatched")
	require.True(t, inherited.Inherited)
	require.True(t, inherited.Unchecked, "no Query over the protocol's Scenarios asks the product's law")
	require.NotEmpty(t, inherited.Pins, "read through the refinement")

	// A functional law is a same-step MUST its find witnesses on one path.
	settles := lawOf(t, protocol.Laws, "activityProtocol.terminateSettles")
	require.Equal(t, Must, settles.Modality)
	require.True(t, settles.Witnessed)
	require.False(t, settles.Inherited)
	require.Equal(t, "control-terminate", settles.Claim.Actions["Terminable.terminate"])
	for _, c := range settles.Pins {
		require.Equal(t, "control-terminate", c.Class)
	}
}

func TestLawTablesNameWaiversAndCompositions(t *testing.T) {
	r := lawTablesOf(t, "../../../model/ir/activity-system.json")
	// The admission record waives closedIsRejectedUniformly with its reason.
	record := tableOf(t, r, "currentAdmission")
	require.True(t, slices.ContainsFunc(record.Laws.Excepted, func(w model.LawWaiver) bool {
		return w.Law == "closedIsRejectedUniformly" && strings.TrimSpace(w.Because) != ""
	}))
	// A composition has no table: its laws are printed with what they say.
	require.NotEmpty(t, r.Laws)
	for _, lt := range r.Laws {
		require.NotEmpty(t, lt.Laws, lt.Owner)
		for _, lv := range lt.Laws {
			require.Empty(t, lv.Pins)
			require.NotNil(t, lv.Entry)
		}
	}

	// An override is printed with the def that states it and its reason.
	n := lawTablesOf(t, "../../../model/ir/nexus-operation.json")
	override := lawOf(t, tableOf(t, n, "nexusOperation").Laws, "nexusOperation.closedIsRejectedUniformly")
	require.NotNil(t, override.Overridden)
	require.Contains(t, override.Overridden.By, "closedRejectsOrRepeats")
}

func TestLawTablesAreWritten(t *testing.T) {
	r := lawTablesOf(t, "../../../model/ir/activity.json")
	var out strings.Builder
	require.NoError(t, WriteTables(&out, r))
	text := out.String()
	require.Contains(t, text, "\nlaws ../../../model/ir/activity.json activityProduct\n")
	require.Regexp(t, `\n  activityProduct\.pausedIsNotDispatched  pausedIsNotDispatched of Pausable and Pollable, MUST NOT  model/temporal/features/standaloneactivity/Capabilities\.scala:\d+\n`, text)
	require.Contains(t, text, "\n    promises: while an entity is paused no work is handed to a worker")
	require.Regexp(t, `\n    attemptStart \(Pollable\.dispatch\) +paused +MUST NOT +cell: \? s\.phase != scheduled\n`, text)
	require.Regexp(t, `\n    every class +completed, failed, canceled, terminated, timedOut +MUST NOT\n`, text)
	require.Contains(t, text, "\n  no law pins\n")
	require.Contains(t, text, "inherited from activityProduct, unchecked")
	require.Contains(t, text, "MUST on its find's path only")
}
