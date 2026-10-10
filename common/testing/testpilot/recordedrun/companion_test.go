package recordedrun

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
)

func TestCompanionMigrationInventory(t *testing.T) {
	root := "../../../.."
	encoded, err := os.ReadFile("testdata/companion-migration.json")
	require.NoError(t, err)
	var inventory struct {
		CaseRoots  []string `json:"caseRoots"`
		Companions []struct {
			Case string `json:"case"`
			Run  string `json:"run"`
		} `json:"companions"`
	}
	require.NoError(t, json.Unmarshal(encoded, &inventory))
	require.NotEmpty(t, inventory.CaseRoots)
	require.NotEmpty(t, inventory.Companions)
	var records []string
	caseIdentities := map[string]string{}
	for _, directory := range inventory.CaseRoots {
		before := len(caseIdentities)
		err := filepath.WalkDir(filepath.Join(root, directory), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".json") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var fields map[string]json.RawMessage
			if json.Unmarshal(data, &fields) != nil {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if fields["case"] != nil && fields["run"] != nil && fields["identity"] != nil {
				records = append(records, filepath.ToSlash(relative))
			}
			if fields["program"] == nil || fields["contract"] == nil {
				return nil
			}
			identity, err := CaseIdentity(data)
			if err != nil {
				return err
			}
			caseIdentities[filepath.ToSlash(relative)] = identity
			return nil
		})
		require.NoError(t, err)
		require.Greater(t, len(caseIdentities), before, "Case root %s must not be empty", directory)
	}
	var inventoried []string
	for _, companion := range inventory.Companions {
		identity, exists := caseIdentities[companion.Case]
		require.True(t, exists, "uninventoried Case %s", companion.Case)
		encoded, err := os.ReadFile(filepath.Join(root, companion.Run))
		require.NoError(t, err)
		decoded, err := DecodeForCase(encoded, identity)
		require.NoError(t, err, "companion %s", companion.Run)
		caseBytes, err := os.ReadFile(filepath.Join(root, companion.Case))
		require.NoError(t, err)
		source, err := testpilot.DecodeCaseProtoJSON(caseBytes)
		require.NoError(t, err)
		require.Empty(t, Crossed(source, decoded.Run))
		inventoried = append(inventoried, companion.Run)
	}
	slices.Sort(records)
	slices.Sort(inventoried)
	require.Equal(t, records, inventoried, "every checked-in record needs an exact companion")
	t.Logf("migration inventory: %d Cases in %d roots; %d exact Run companions", len(caseIdentities), len(inventory.CaseRoots), len(records))
}
