package ir

// The source-derived Models the lifter's tests pin in model/irgen/testdata/lifts/expected,
// read as any other Model: admitted, then interpreted.

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
)

func lifted(t *testing.T, name string) *umpirespb.Model {
	t.Helper()
	m, err := Load(filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", name+".json"))
	require.NoError(t, err)
	return m
}

func TestLiftedModelsAreAdmitted(t *testing.T) {
	for _, name := range []string{"admission", "captured", "channels", "declarations", "hints", "presence", "realizations", "rejections", "rules", "taskqueue"} {
		t.Run(name, func(t *testing.T) {
			lifted(t, name)
		})
	}
}

// fn-126 R16: a machine object's rules lower to one step function per action, whose table is the
// one the same machine written in the core, with hand-written step functions, gives
// (model/irgen/testdata/lifts/Rules.scala: `Switch` and `CoreSwitch`).
func TestRulesLowerToTheCoreTables(t *testing.T) {
	machines, err := interp.Build(lifted(t, "rules"))
	require.NoError(t, err)
	table := func(name string) string {
		m, ok := machines[name]
		require.True(t, ok, name)
		tb := *m.Table
		tb.Machine = ""
		encoded, err := json.Marshal(tb)
		require.NoError(t, err)
		return string(encoded)
	}
	require.Equal(t, table("coreSwitch"), table("switch"))
	require.NotEmpty(t, machines["switch"].Table.Rows)
	// A bare binding keeps the rules' guards: a steady lamp's tick keeps it where a tick wears it out.
	require.Len(t, machines["steady"].Table.Rows, len(machines["switch"].Table.Rows))
}

// fn-139.1: machines on the framework's shared outcomes (model/irgen/testdata/lifts/Rejections.scala),
// whose `rejected` case carries a Rejection: the catalog keys each rejection apart, and the system,
// which rejects with `rejects`, has the table of the product it spells with `reject`, but for the
// explanation no key reads.
func TestSharedOutcomesKeyEachRejection(t *testing.T) {
	machines, err := interp.Build(lifted(t, "rejections"))
	require.NoError(t, err)
	system, product := machines["doorSystem"], machines["doorProduct"]
	require.Equal(t, []string{
		"accepted", "rejected-notFound", "rejected-alreadyExists", "rejected-failedPrecondition", "rejected-invalidArgument",
	}, system.Table.Outcomes)
	for _, key := range []string{"gone-open", "gone-ring", "shut-ring"} {
		require.Equal(t, "rejected-notFound", tableRow(t, system, key).Results[0].Outcome, key)
	}
	locked := tableRow(t, system, "locked-open").Results[0]
	require.Equal(t, "rejected-failedPrecondition", locked.Outcome)
	require.Equal(t, "the door is locked", locked.Because)
	rows := func(m *interp.Machine) string {
		encoded, err := json.Marshal(m.Table.Rows)
		require.NoError(t, err)
		return string(encoded)
	}
	require.Equal(t, rows(product), rows(system))
}

func tableRow(t *testing.T, m *interp.Machine, key string) interp.Row {
	t.Helper()
	for _, r := range m.Table.Rows {
		if r.Key == key {
			return r
		}
	}
	require.FailNow(t, "no row "+key)
	return interp.Row{}
}
