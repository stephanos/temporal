package ir

// Load decodes one ProtoJSON file and admits what it decoded. What it refuses names the file, or is
// what Validate says of the decoded Model.

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func writeModel(t *testing.T, encoded string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "model.json")
	require.NoError(t, os.WriteFile(path, []byte(encoded), 0o600))
	return path
}

func TestLoadDecodesTheFileItAdmits(t *testing.T) {
	path := filepath.Join("..", "..", "..", "model", "irgen", "testdata", "lifts", "expected", "presence.json")
	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	want := &umpirespb.Model{}
	require.NoError(t, protojson.Unmarshal(encoded, want))

	got, err := Load(path)
	require.NoError(t, err)
	require.True(t, proto.Equal(want, got))
}

func TestLoadRefusesAFileItCannotRead(t *testing.T) {
	m, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.Nil(t, m)
}

func TestIRPathsSkipsOnlyCurrentMetadataSidecars(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"model.json", "model.lint.json", "model.waivers.json", "model.laws.json"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o600))
	}
	paths, err := IRPaths(dir)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(dir, "model.json"), filepath.Join(dir, "model.laws.json")}, paths)
}

// What is no ProtoJSON of the schema is refused at the file: it has no position inside a Model.
func TestLoadRefusesWhatTheSchemaDoesNotHave(t *testing.T) {
	for name, test := range map[string]struct{ encoded, says string }{
		"a field the schema lacks": {`{"noSuchField": 1}`, `unknown field "noSuchField"`},
		"what is no JSON":          {`{`, "unexpected EOF"},
	} {
		t.Run(name, func(t *testing.T) {
			path := writeModel(t, test.encoded)
			m, err := Load(path)
			require.Nil(t, m)
			var located *interp.Error
			require.ErrorAs(t, err, &located)
			require.Equal(t, path, located.Position)
			require.Contains(t, located.Message, test.says)
		})
	}
}

// A file that decodes is admitted as any Model is: Load reports what Validate reports, and no Model.
func TestLoadReportsWhatValidateRefuses(t *testing.T) {
	path := writeModel(t, `{"source": "written", "version": 7, "machines": [{"name": "m"}, {"name": "m"}]}`)
	decoded := &umpirespb.Model{Source: "written", Version: 7, Machines: []*umpirespb.Machine{{Name: "m"}, {Name: "m"}}}
	want := Validate(decoded)
	require.Error(t, want)

	m, err := Load(path)
	require.Nil(t, m)
	require.Equal(t, want, err)
}
