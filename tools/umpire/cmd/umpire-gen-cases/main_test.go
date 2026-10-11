package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/ir"
	"go.temporal.io/server/tools/umpire/lower"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const repository = "../../../.."

func TestFeaturePublicationPreservesIRLegacyAndSiblingTrees(t *testing.T) {
	complete := modelTree(t)
	selected, err := lower.SelectCases(complete, kinds["canary"].pinned)
	require.NoError(t, err)
	root := t.TempDir()
	directory := filepath.Join(root, "model", "temporal", "features", "nexus", "workflow", "generated")
	sibling := filepath.Join(root, "model", "temporal", "features", "future_form", "generated")
	for _, dir := range []string{directory, sibling, filepath.Join(directory, "legacy")} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	preserved := map[string][]byte{
		filepath.Join(directory, "ir.json"):                     []byte("immutable IR"),
		filepath.Join(directory, "ir.lint.json"):                []byte("immutable lint"),
		filepath.Join(directory, "legacy", "supplemental.json"): []byte("immutable historical payload"),
		filepath.Join(sibling, "ir.json"):                       []byte("unselected sibling"),
	}
	for path, data := range preserved {
		require.NoError(t, os.WriteFile(path, data, 0o644))
	}
	require.NoError(t, lower.SyncCases(directory, selected, true))
	require.NoError(t, lower.SyncCases(directory, selected, false))
	for path, data := range preserved {
		actual, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, data, actual, path)
	}
	encoded, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	require.NoError(t, err)
	manifest, err := lower.DecodeManifest(encoded)
	require.NoError(t, err)
	for _, entry := range manifest.Queries {
		require.Equal(t, entry.Query.Name+".case.json", entry.File)
		actual, err := os.ReadFile(filepath.Join(directory, entry.File))
		require.NoError(t, err)
		require.Equal(t, complete[strings.TrimSuffix(entry.Model, ".json")+"-"+entry.Query.Name+"-case.json"], actual)
	}
}

func TestUnscopedPublicationDiscoversEveryAddedFeature(t *testing.T) {
	root := t.TempDir()
	model, err := ir.Load(ir.ModelPath(repository, "nexus-workflow.json"))
	require.NoError(t, err)
	model.Queries = slices.DeleteFunc(model.Queries, func(q *umpirespb.Query) bool { return q.GetName() != "syncCompletion" })
	require.Len(t, model.Queries, 1)
	encoded, err := protojson.Marshal(model)
	require.NoError(t, err)
	write := func(name string) string {
		directory := filepath.Join(root, ir.FeatureDirectory, name, "generated")
		require.NoError(t, os.MkdirAll(directory, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(directory, "ir.json"), encoded, 0o644))
		return directory
	}
	first, second := write("first_form"), write("second")
	require.NoError(t, modelCases(ir.FeatureRoot(root), true))
	require.NoError(t, modelCases(ir.FeatureRoot(root), false))
	third := write("new_feature")
	require.NoError(t, modelCases(ir.FeatureRoot(root), true))
	for _, directory := range []string{first, second, third} {
		manifestBytes, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
		require.NoError(t, err)
		manifest, err := lower.DecodeManifest(manifestBytes)
		require.NoError(t, err)
		require.Len(t, manifest.Queries, 1)
		require.Equal(t, ir.ModelName(filepath.Join(directory, "ir.json")), manifest.Queries[0].Model)
		require.FileExists(t, filepath.Join(directory, "syncCompletion.case.json"))
	}
	siblingBytes, err := os.ReadFile(filepath.Join(second, "manifest.json"))
	require.NoError(t, err)
	siblingInfo, err := os.Stat(filepath.Join(second, "manifest.json"))
	require.NoError(t, err)
	require.NoError(t, modelCases(first, true))
	after, err := os.ReadFile(filepath.Join(second, "manifest.json"))
	require.NoError(t, err)
	afterInfo, err := os.Stat(filepath.Join(second, "manifest.json"))
	require.NoError(t, err)
	require.Equal(t, siblingBytes, after)
	require.Equal(t, siblingInfo.ModTime(), afterInfo.ModTime())
}

// modelTree is the checked-in complete tree, which the lowering package's own test holds equal to
// what the checked IR lowers to.
func modelTree(t *testing.T) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	manifest := lower.Manifest{Version: 1}
	err := filepath.WalkDir(filepath.Join(repository, ir.FeatureDirectory), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "manifest.json" {
			return nil
		}
		encoded, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		part, err := lower.DecodeManifest(encoded)
		if err != nil {
			return err
		}
		for _, item := range part.Queries {
			if item.Standing == lower.Lowered {
				physical := filepath.Join(filepath.Dir(path), item.File)
				item.File = strings.TrimSuffix(item.Model, ".json") + "-" + item.Query.Name + "-case.json"
				files[item.File], err = os.ReadFile(physical)
				if err != nil {
					return err
				}
			}
			manifest.Queries = append(manifest.Queries, item)
		}
		return nil
	})
	require.NoError(t, err)
	slices.SortFunc(manifest.Queries, func(a, b lower.GeneratedCase) int {
		if a.Model != b.Model {
			return strings.Compare(a.Model, b.Model)
		}
		return strings.Compare(a.Query.Name, b.Query.Name)
	})
	files["manifest.json"], err = json.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)
	files["manifest.json"] = append(files["manifest.json"], '\n')
	return files
}

// Every pinned tree is checked in as the complete tree's selection, so a functional fixture and the
// canary's pin of one Query are the same bytes as the model tree's Case.
func TestEveryPinnedTreeIsCheckedInAsTheModelTreesSelection(t *testing.T) {
	complete := modelTree(t)
	pinned := 0
	for name, selected := range kinds {
		if len(selected.pinned) == 0 {
			continue
		}
		pinned++
		t.Run(name, func(t *testing.T) {
			pinnedFiles, err := lower.SelectCases(complete, selected.pinned)
			require.NoError(t, err)
			original, err := os.ReadFile(filepath.Join(repository, selected.directory, "manifest.json"))
			require.NoError(t, err)
			require.Equal(t, string(original), string(pinnedFiles["manifest.json"]))
			require.NoError(t, sync(repository, complete, selected, false))
			encoded, err := os.ReadFile(filepath.Join(repository, selected.directory, "manifest.json"))
			require.NoError(t, err)
			manifest, err := lower.DecodeManifest(encoded)
			require.NoError(t, err)
			require.Len(t, manifest.Queries, len(selected.pinned))
			for _, entry := range manifest.Queries {
				checkedIn, err := os.ReadFile(filepath.Join(repository, selected.directory, entry.File))
				require.NoError(t, err)
				require.Equal(t, complete[entry.File], checkedIn, entry.File)
			}
		})
	}
	require.Equal(t, 2, pinned)

	shared := "nexus-workflow-syncCompletion-case.json"
	functional, err := os.ReadFile(filepath.Join(repository, kinds["functional"].directory, shared))
	require.NoError(t, err)
	canary, err := os.ReadFile(filepath.Join(repository, kinds["canary"].directory, shared))
	require.NoError(t, err)
	require.Equal(t, functional, canary, "the canary pins the functional fixture's bytes")
}

// A check of a tree that differs from the selection fails and writes nothing; an update publishes
// the whole tree, the check that follows finds no difference, and a selection that is refused
// leaves the published tree as it was.
func TestAPinnedTreeIsPublishedWholeAndThenChecksClean(t *testing.T) {
	complete := modelTree(t)
	root := t.TempDir()
	selected := kinds["canary"]
	parent := filepath.Dir(filepath.Join(root, selected.directory))
	require.NoError(t, os.MkdirAll(parent, 0o755))

	require.Error(t, sync(root, complete, selected, false), "an absent tree is stale")
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	require.Empty(t, entries, "a failed check leaves nothing behind")

	require.NoError(t, sync(root, complete, selected, true))
	require.NoError(t, sync(root, complete, selected, false))
	manifest := filepath.Join(root, selected.directory, "manifest.json")
	published, err := os.ReadFile(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifest, append(published, '\n'), 0o644))
	require.ErrorContains(t, sync(root, complete, selected, false), "manifest.json is stale")
	require.NoError(t, sync(root, complete, selected, true))
	republished, err := os.ReadFile(manifest)
	require.NoError(t, err)
	require.Equal(t, published, republished)

	unknown := selected
	unknown.pinned = []lower.Selected{{Model: "nexus-workflow.json", Query: "absent"}}
	require.ErrorContains(t, sync(root, complete, unknown, true), "no Model declares the selected Query")
	require.NoError(t, sync(root, complete, selected, false), "a refused selection publishes nothing")
}

// Cases are generated only from IR whose every Query declares its total: each Query without one is
// refused with its count, in every IR file at once, and IR declaring every total is admitted.
func TestEveryQueryWithoutATotalIsRefused(t *testing.T) {
	directory := t.TempDir()
	queries := 0
	for _, name := range []string{"nexus-workflow.json", "nexus-workflow-control.json"} {
		m, err := ir.Load(ir.ModelPath(repository, name))
		require.NoError(t, err)
		m = ir.WithoutTotals(m)
		queries += len(m.GetQueries())
		encoded, err := protojson.Marshal(m)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(directory, name), encoded, 0o644))
	}
	err := requireTotals(directory)
	require.ErrorContains(t, err, "nexus-workflow.json")
	require.ErrorContains(t, err, "nexus-workflow-control.json")
	require.ErrorContains(t, err, "query forgedCompletion declares no total: its static combination count is ")
	require.Len(t, err.(interface{ Unwrap() []error }).Unwrap(), 2, "one refusal per IR file")
	require.Equal(t, queries, strings.Count(err.Error(), "declares no total"), "every Query is reported")

	for _, name := range []string{"nexus-workflow.json", "nexus-workflow-control.json"} {
		path := filepath.Join(directory, name)
		m, err := ir.Load(path)
		require.NoError(t, err)
		declared := proto.CloneOf(m)
		for _, q := range declared.GetQueries() {
			q.Total = wrapperspb.Int64(0)
		}
		totaled, err := ir.WithTotals(declared)
		require.NoError(t, err)
		require.NoError(t, ir.Validate(totaled))
		encoded, err := protojson.Marshal(totaled)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, encoded, 0o644))
	}
	require.NoError(t, requireTotals(directory))
}
