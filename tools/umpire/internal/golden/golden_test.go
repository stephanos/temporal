package golden

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

func TestCaptureNeverReplacesAnExistingDestination(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "capture")
	files := map[string][]byte{"one.json": []byte("original\n")}
	require.NoError(t, Capture(dir, files))
	require.Error(t, Capture(dir, map[string][]byte{"one.json": []byte("changed\n")}))
	got, err := Read(dir)
	require.NoError(t, err)
	require.Equal(t, files["one.json"], got["one.json"])
}

func TestClosedMigrationRejectsUnlistedSourceChanges(t *testing.T) {
	cfg := Config{Paths: []Substitution{{Old: "old.scala", New: "new.scala"}}, Labels: []Substitution{{Old: "old model", New: "new model"}}}
	original := &umpirespb.Model{Source: "old model", Machines: []*umpirespb.Machine{{Position: &umpirespb.Position{File: "old.scala", Line: 12}}}}
	mapped, err := cfg.Migrate(original)
	require.NoError(t, err)
	require.Equal(t, "old.scala", original.Machines[0].Position.File)
	require.True(t, proto.Equal(&umpirespb.Position{File: "new.scala", Line: 12}, mapped.Machines[0].Position))
	moved, err := cfg.Match(original, mapped)
	require.NoError(t, err)
	require.True(t, moved)
	for _, change := range []func(*umpirespb.Model){
		func(m *umpirespb.Model) { m.Machines[0].Position.File = "other.scala" },
		func(m *umpirespb.Model) { m.Machines[0].Position.Line++ },
		func(m *umpirespb.Model) { m.Source += " unexpected" },
	} {
		changed := proto.CloneOf(mapped)
		change(changed)
		_, err := cfg.Match(original, changed)
		require.Error(t, err)
	}
	original.Machines[0].Position.File = "unlisted.scala"
	_, err = cfg.Migrate(original)
	require.ErrorContains(t, err, "unlisted.scala")
}

func TestCompareRequiresTheWholeInventory(t *testing.T) {
	original := map[string][]byte{"a": []byte("one"), "b": []byte("two")}
	require.NoError(t, Compare(original, original))
	for _, changed := range []map[string][]byte{
		{"a": []byte("one")},
		{"a": []byte("one"), "b": []byte("two"), "c": nil},
		{"a": []byte("changed"), "b": []byte("two")},
	} {
		require.Error(t, Compare(original, changed))
	}
}

func TestIRInventoryRejectsMissingAndUnknownFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "model", "scalav2", "ir")
	require.NoError(t, os.MkdirAll(dir, 0755))
	cfg := Config{Inventory: []string{"model/scalav2/ir/one.json"}}
	_, err := cfg.Inputs(root)
	require.ErrorContains(t, err, "missing IR inventory entry")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "one.json"), []byte(`{}`), 0644))
	inputs, err := cfg.Inputs(root)
	require.NoError(t, err)
	require.Len(t, inputs, 1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.json"), []byte(`{}`), 0644))
	_, err = cfg.Inputs(root)
	require.ErrorContains(t, err, "unknown IR inventory entry")
}
