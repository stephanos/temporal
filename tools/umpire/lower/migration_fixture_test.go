package lower_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/tools/umpire/internal/golden"
	"go.temporal.io/server/tools/umpire/lower"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestOriginalNexusFixturesThroughAdmittedIR(t *testing.T) {
	files, err := golden.Read(filepath.Join("testdata", "migration"))
	require.NoError(t, err)
	m := new(modelirspb.Model)
	require.NoError(t, protojson.Unmarshal(files["original/inputs/ir/nexus-caller.json"], m))
	lower.MigrationComparativeModel(t, m)
	actual := map[string][]byte{}
	expected := map[string][]byte{}
	for _, query := range m.GetQueries() {
		name := query.GetName()
		key := "oracles/nexus/" + name + "/typed"
		var identity cp.Identity
		var source cp.Source
		require.NoError(t, json.Unmarshal(files[key+"/identity-input.json"], &identity))
		require.NoError(t, json.Unmarshal(files[key+"/source.json"], &source))
		q, r, err := lower.MigrationFixture(m, name, identity)
		require.NoError(t, err)
		c, err := cp.Produce(q, identity, r, source)
		require.NoError(t, err)
		captureLegacyQuery(t, actual, key, q)
		putCase(t, actual, key, c)
	}
	for k := range actual {
		expected[k] = files[k]
	}
	require.NoError(t, golden.Compare(expected, actual))
}
