package lower

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func generatedFiles(t *testing.T) map[string][]byte {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "model", "cases", "*.json"))
	require.NoError(t, err)
	files := map[string][]byte{}
	for _, path := range paths {
		files[filepath.Base(path)], err = os.ReadFile(path)
		require.NoError(t, err)
	}
	return files
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

// A selection is the complete tree's own bytes for the Queries it names, whatever order names them,
// and it publishes and checks as a tree of its own.
func TestSelectedCasesAreTheCompleteTreesBytes(t *testing.T) {
	// The checked-in tree is what the checked IR lowers to, which TestGeneratedCasesAreCheckedIn holds.
	files, again := generatedFiles(t), generatedFiles(t)

	selected, err := SelectCases(files, []Selected{
		{Model: "nexus-control.json", Query: "forgedCompletion"},
		{Model: "nexus-caller.json", Query: "syncCompletion"},
	})
	require.NoError(t, err)
	reordered, err := SelectCases(again, []Selected{
		{Model: "nexus-caller.json", Query: "syncCompletion"},
		{Model: "nexus-control.json", Query: "forgedCompletion"},
	})
	require.NoError(t, err)
	require.Equal(t, selected, reordered)

	manifest, err := DecodeManifest(selected["manifest.json"])
	require.NoError(t, err)
	complete, err := DecodeManifest(files["manifest.json"])
	require.NoError(t, err)
	expected := map[string][]byte{"manifest.json": selected["manifest.json"]}
	var entries []GeneratedCase
	for _, entry := range complete.Queries {
		if entry.File == "nexus-caller-syncCompletion-case.json" || entry.File == "nexus-control-forgedCompletion-case.json" {
			expected[entry.File] = files[entry.File]
			entries = append(entries, entry)
		}
	}
	require.Equal(t, &Manifest{Version: 1, Queries: entries}, manifest)
	require.Equal(t, expected, selected)

	root := filepath.Join(t.TempDir(), "pinned")
	require.NoError(t, SyncCases(root, selected, true))
	require.NoError(t, SyncCases(root, reordered, false))
	require.Error(t, SyncCases(root, files, false), "the complete tree is another tree")
}

// A Query with no Case cannot be pinned: one no Model declares, one named twice, one that lowers to
// nothing, and an empty selection are each refused, and nothing is selected.
func TestSelectingRefusesAQueryWithNoCase(t *testing.T) {
	files := generatedFiles(t)
	manifest, err := DecodeManifest(files["manifest.json"])
	require.NoError(t, err)
	var unlowered Selected
	for _, entry := range manifest.Queries {
		if entry.Standing != Lowered {
			unlowered = Selected{Model: entry.Model, Query: entry.Query.Name}
			break
		}
	}
	require.NotEmpty(t, unlowered.Query, "the model tree accounts for a Query that does not lower")
	pinned := Selected{Model: "nexus-caller.json", Query: "syncCompletion"}
	for name, test := range map[string]struct {
		selected []Selected
		detail   string
	}{
		"nothing":           {nil, "no Query selected"},
		"an undeclared one": {[]Selected{pinned, {Model: "nexus-caller.json", Query: "absent"}}, "no Model declares the selected Query nexus-caller.json/absent"},
		"another Model's":   {[]Selected{{Model: "nexus-control.json", Query: "syncCompletion"}}, "no Model declares the selected Query nexus-control.json/syncCompletion"},
		"one named twice":   {[]Selected{pinned, pinned}, "named twice"},
		"one with no Case":  {[]Selected{pinned, unlowered}, "has no Case: " + string(manifestStanding(manifest, unlowered))},
		"a missing Case":    {[]Selected{{Model: "nexus-control.json", Query: "forgedCompletion"}}, "generated Case nexus-control-forgedCompletion-case.json is missing"},
		"a broken manifest": {[]Selected{pinned}, "manifest"},
	} {
		t.Run(name, func(t *testing.T) {
			input := maps.Clone(files)
			switch name {
			case "a missing Case":
				delete(input, "nexus-control-forgedCompletion-case.json")
			case "a broken manifest":
				input["manifest.json"] = []byte(`{"version":2,"queries":[]}`)
			default:
			}
			selected, err := SelectCases(input, test.selected)
			require.ErrorContains(t, err, test.detail)
			require.Nil(t, selected)
		})
	}
}

func manifestStanding(manifest *Manifest, query Selected) Standing {
	for _, entry := range manifest.Queries {
		if entry.Model == query.Model && entry.Query.Name == query.Query {
			return entry.Standing
		}
	}
	return ""
}
