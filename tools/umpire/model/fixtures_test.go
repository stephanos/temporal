package model

// The source-derived Models the lifter's tests pin in model/irgen/testdata/lifts/expected,
// read as any other Model: admitted, then interpreted.

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
)

func lifted(t *testing.T, name string) *umpirespb.Model {
	t.Helper()
	m, err := Load(filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", name+".json"))
	require.NoError(t, err)
	return m
}

func TestLiftedModelsAreAdmitted(t *testing.T) {
	for _, name := range []string{"admission", "captured", "channels", "declarations", "hints", "presence", "realizations", "rules", "taskqueue"} {
		t.Run(name, func(t *testing.T) {
			lifted(t, name)
		})
	}
}

// fn-126 R16: a machine object's rules lower to one step function per action, whose table is the
// one the same machine written in the core, with hand-written step functions, gives
// (model/irgen/testdata/lifts/Rules.scala: `Switch` and `CoreSwitch`).
func TestRulesLowerToTheCoreTables(t *testing.T) {
	machines, err := Build(lifted(t, "rules"))
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
