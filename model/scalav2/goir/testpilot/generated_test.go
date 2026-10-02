package testpilot

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot/recordedrun"
)

func generatedFiles(t *testing.T) map[string][]byte {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "..", "cases", "*.json"))
	require.NoError(t, err)
	files := map[string][]byte{}
	for _, path := range paths {
		files[filepath.Base(path)], err = os.ReadFile(path)
		require.NoError(t, err)
	}
	return files
}

func TestGeneratedCasesAreCheckedIn(t *testing.T) {
	files, err := GenerateCases(filepath.Join("..", "..", "ir"))
	require.NoError(t, err)
	require.NoError(t, SyncCases(filepath.Join("..", "..", "cases"), files, false))
	manifest, err := DecodeManifest(files["manifest.json"])
	require.NoError(t, err)
	lowered := 0
	for _, entry := range manifest.Queries {
		if entry.Standing != Lowered {
			continue
		}
		lowered++
		_, err := recordedrun.CaseIdentity(files[entry.File])
		require.NoError(t, err)
		require.NotNil(t, entry.Expected)
	}
	require.Positive(t, lowered)
}

func TestManifestRejectsInvalidMetadata(t *testing.T) {
	encoded := generatedFiles(t)["manifest.json"]
	for name, mutate := range map[string]func(*Manifest){
		"version":         func(m *Manifest) { m.Version++ },
		"duplicate query": func(m *Manifest) { m.Queries = append(m.Queries, m.Queries[0]) },
		"duplicate path": func(m *Manifest) {
			for i := range m.Queries {
				if m.Queries[i].Standing == Lowered {
					other := m.Queries[i]
					other.Query.Name += "-other"
					m.Queries = append(m.Queries, other)
					break
				}
			}
		},
		"traversal": func(m *Manifest) { m.Queries[0].Model = "../model.json" },
		"missing expected": func(m *Manifest) {
			for i := range m.Queries {
				if m.Queries[i].Standing == Lowered {
					m.Queries[i].Expected = nil
					break
				}
			}
		},
		"unknown status": func(m *Manifest) {
			for i := range m.Queries {
				if m.Queries[i].Standing == Lowered {
					m.Queries[i].Expected.Properties[0].Status = "maybe"
					break
				}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			m, err := DecodeManifest(encoded)
			require.NoError(t, err)
			mutate(m)
			invalid, err := json.Marshal(m)
			require.NoError(t, err)
			_, err = DecodeManifest(invalid)
			require.Error(t, err)
		})
	}
	for _, invalid := range [][]byte{[]byte(`{"version":1,"unknown":true}`), append(append([]byte{}, encoded...), []byte("{}")...)} {
		_, err := DecodeManifest(invalid)
		require.Error(t, err)
	}
}

func TestCasePublicationDetectsDriftAndPreservesTheOldTreeOnInvalidInput(t *testing.T) {
	files := generatedFiles(t)
	for _, drift := range []string{"stale", "missing", "obsolete", "symlink"} {
		t.Run(drift, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "cases")
			require.NoError(t, SyncCases(root, files, true))
			switch drift {
			case "stale":
				require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), []byte("{}"), 0644))
			case "missing":
				require.NoError(t, os.Remove(filepath.Join(root, "manifest.json")))
			case "obsolete":
				require.NoError(t, os.WriteFile(filepath.Join(root, "obsolete.json"), []byte("{}"), 0644))
			case "symlink":
				require.NoError(t, os.Remove(filepath.Join(root, "manifest.json")))
				require.NoError(t, os.Symlink("absent", filepath.Join(root, "manifest.json")))
			default:
				t.Fatal(drift)
			}
			require.Error(t, SyncCases(root, files, false))
			require.NoError(t, SyncCases(root, files, true))
			require.NoError(t, SyncCases(root, files, false))
			broken := maps.Clone(files)
			for name := range broken {
				if name != "manifest.json" {
					broken[name] = []byte("{")
					break
				}
			}
			require.Error(t, SyncCases(root, broken, true))
			require.NoError(t, SyncCases(root, files, false))
		})
	}
}
