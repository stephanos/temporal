package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/ir"
	"go.temporal.io/server/tools/umpire/lower"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestPartialPinnedPublicationPreservesUnselectedArtifacts(t *testing.T) {
	complete := modelTree(t)
	selected := kinds["functional"]
	pins := slices.DeleteFunc(slices.Clone(selected.pinned), func(pin lower.Selected) bool {
		return pin.Model != "nexus-workflow.json"
	})
	partial, err := lower.SelectCases(complete, pins)
	require.NoError(t, err)
	root := t.TempDir()
	directory := filepath.Join(root, filepath.FromSlash(selected.directory))
	require.NoError(t, os.MkdirAll(filepath.Dir(directory), 0o755))
	require.NoError(t, sync(root, complete, selected, true))
	unselectedFile := filepath.Join(directory, "nexus-workflow-control-forgedCompletion-case.json")
	unselectedBytes, err := os.ReadFile(unselectedFile)
	require.NoError(t, err)
	stamp := time.Unix(1234567890, 0)
	require.NoError(t, os.Chtimes(unselectedFile, stamp, stamp))
	unselectedInfo, err := os.Stat(unselectedFile)
	require.NoError(t, err)
	manifestFile := filepath.Join(directory, "manifest.json")
	originalManifest, err := os.ReadFile(manifestFile)
	require.NoError(t, err)
	originalRows, err := lower.DecodeManifest(originalManifest)
	require.NoError(t, err)
	require.NoError(t, os.Chtimes(manifestFile, stamp, stamp))
	manifestInfo, err := os.Stat(manifestFile)
	require.NoError(t, err)

	require.NoError(t, sync(root, partial, selected, false))
	checkedManifest, err := os.ReadFile(manifestFile)
	require.NoError(t, err)
	require.Equal(t, originalManifest, checkedManifest)
	checkedInfo, err := os.Stat(manifestFile)
	require.NoError(t, err)
	require.Equal(t, manifestInfo.ModTime(), checkedInfo.ModTime(), "a check writes nothing")

	changedFile := "nexus-workflow-syncCompletion-case.json"
	require.NoError(t, os.WriteFile(filepath.Join(directory, changedFile), []byte("{"), 0o644))
	require.ErrorContains(t, sync(root, partial, selected, false), changedFile)
	corrupt, err := os.ReadFile(filepath.Join(directory, changedFile))
	require.NoError(t, err)
	require.Equal(t, []byte("{"), corrupt, "a failed check does not repair the Case")
	checkedManifest, err = os.ReadFile(manifestFile)
	require.NoError(t, err)
	require.Equal(t, originalManifest, checkedManifest)

	changedCase, err := testpilot.DecodeCaseProtoJSON(partial[changedFile])
	require.NoError(t, err)
	changedCase.Provenance.ProducerVersion += "+scoped"
	partial[changedFile], err = protojson.Marshal(changedCase)
	require.NoError(t, err)
	partial[changedFile] = append(partial[changedFile], '\n')
	partialRows, err := lower.DecodeManifest(partial["manifest.json"])
	require.NoError(t, err)
	partialRows.Queries[0].Expected.Properties[0].ID += "-scoped"
	partial["manifest.json"], err = json.MarshalIndent(partialRows, "", "  ")
	require.NoError(t, err)
	partial["manifest.json"] = append(partial["manifest.json"], '\n')
	require.NoError(t, sync(root, partial, selected, true))
	repaired, err := os.ReadFile(filepath.Join(directory, changedFile))
	require.NoError(t, err)
	require.Equal(t, partial[changedFile], repaired)
	afterBytes, err := os.ReadFile(unselectedFile)
	require.NoError(t, err)
	require.Equal(t, unselectedBytes, afterBytes)
	afterInfo, err := os.Stat(unselectedFile)
	require.NoError(t, err)
	require.Equal(t, unselectedInfo.ModTime(), afterInfo.ModTime())
	updatedBytes, err := os.ReadFile(manifestFile)
	require.NoError(t, err)
	updated, err := lower.DecodeManifest(updatedBytes)
	require.NoError(t, err)
	expected := slices.DeleteFunc(slices.Clone(originalRows.Queries), func(entry lower.GeneratedCase) bool {
		return entry.Model == "nexus-workflow.json"
	})
	expected = append(expected, partialRows.Queries...)
	require.ElementsMatch(t, expected, updated.Queries)
	require.NoError(t, sync(root, partial, selected, false))
}

func TestPartialPinnedPublicationWithNoMatchingPinsCreatesNothing(t *testing.T) {
	partial, err := lower.SelectCases(modelTree(t), []lower.Selected{{Model: "activity-standalone.json", Query: "completion"}})
	require.NoError(t, err)
	root := t.TempDir()
	for _, update := range []bool{false, true} {
		require.NoError(t, sync(root, partial, kinds["functional"], update))
		entries, err := os.ReadDir(root)
		require.NoError(t, err)
		require.Empty(t, entries)
	}
}

func TestPartialPinnedPublicationRejectsMissingQueryOfSelectedModel(t *testing.T) {
	partial, err := lower.SelectCases(modelTree(t), []lower.Selected{{Model: "nexus-workflow.json", Query: "syncCompletion"}})
	require.NoError(t, err)
	root := t.TempDir()
	require.ErrorContains(t, sync(root, partial, kinds["functional"], true), "no Model declares the selected Query nexus-workflow.json/")
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestPartialPinnedPublicationRejectsUnsafeExistingTree(t *testing.T) {
	complete := modelTree(t)
	selected := kinds["functional"]
	pins := slices.DeleteFunc(slices.Clone(selected.pinned), func(pin lower.Selected) bool {
		return pin.Model != "nexus-workflow.json"
	})
	partial, err := lower.SelectCases(complete, pins)
	require.NoError(t, err)
	for _, fault := range []string{"lock", "unselected symlink", "missing identity", "manifest symlink", "unknown entry"} {
		t.Run(fault, func(t *testing.T) {
			root := t.TempDir()
			directory := filepath.Join(root, filepath.FromSlash(selected.directory))
			require.NoError(t, os.MkdirAll(filepath.Dir(directory), 0o755))
			require.NoError(t, sync(root, complete, selected, true))
			manifest := filepath.Join(directory, "manifest.json")
			var link string
			switch fault {
			case "lock":
				require.NoError(t, os.Mkdir(directory+".lock", 0o700))
			case "unselected symlink", "manifest symlink":
				link = manifest
				if fault == "unselected symlink" {
					link = filepath.Join(directory, "nexus-workflow-control-forgedCompletion-case.json")
				}
				encoded, err := os.ReadFile(link)
				require.NoError(t, err)
				target := filepath.Join(t.TempDir(), "external.json")
				require.NoError(t, os.WriteFile(target, encoded, 0o644))
				require.NoError(t, os.Remove(link))
				require.NoError(t, os.Symlink(target, link))
			case "missing identity":
				encoded, err := os.ReadFile(manifest)
				require.NoError(t, err)
				rows, err := lower.DecodeManifest(encoded)
				require.NoError(t, err)
				rows.Queries = slices.DeleteFunc(rows.Queries, func(entry lower.GeneratedCase) bool {
					return entry.Model == "nexus-workflow.json" && entry.Query.Name == "syncCompletion"
				})
				encoded, err = json.MarshalIndent(rows, "", "  ")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(manifest, append(encoded, '\n'), 0o644))
			case "unknown entry":
				require.NoError(t, os.WriteFile(filepath.Join(directory, "unexpected.json"), []byte("{}"), 0o644))
			}
			before := map[string][]byte{}
			entries, err := os.ReadDir(directory)
			require.NoError(t, err)
			for _, entry := range entries {
				before[entry.Name()], err = os.ReadFile(filepath.Join(directory, entry.Name()))
				require.NoError(t, err)
			}
			require.Error(t, sync(root, partial, selected, true))
			after, err := os.ReadDir(directory)
			require.NoError(t, err)
			require.Len(t, after, len(entries))
			for name, encoded := range before {
				actual, err := os.ReadFile(filepath.Join(directory, name))
				require.NoError(t, err)
				require.Equal(t, encoded, actual, name)
			}
			if link != "" {
				_, err := os.Readlink(link)
				require.NoError(t, err, "a refused publication preserves the symlink")
			}
		})
	}
}

func TestPinnedScopeSelectsOwnersAndRejectsInvalidLocations(t *testing.T) {
	root := t.TempDir()
	features := filepath.Join(root, ir.FeatureDirectory)
	owner := filepath.Join(features, "nexus", "workflow")
	flat := filepath.Join(features, "new_form")
	generated := filepath.Join(owner, "generated")
	for _, directory := range []string{generated, flat} {
		require.NoError(t, os.MkdirAll(directory, 0o755))
	}
	escape := filepath.Join(features, "escape")
	require.NoError(t, os.Symlink(t.TempDir(), escape))
	regular := filepath.Join(features, "regular-file")
	require.NoError(t, os.WriteFile(regular, []byte("not a feature folder"), 0o644))
	pins := kinds["functional"].pinned
	for _, item := range []struct {
		name, scope string
		want        []lower.Selected
		rejected    bool
	}{
		{"repository", ir.FeatureRoot(root), pins, false},
		{"features", features, pins, false},
		{"nested source owner", owner, pins, false},
		{"flat owner without pins", flat, nil, false},
		{"generated output", generated, nil, true},
		{"regular file", regular, nil, true},
		{"outside scope", t.TempDir(), nil, true},
		{"symlink escape", escape, nil, true},
	} {
		t.Run(item.name, func(t *testing.T) {
			actualRoot, actualPins, err := pinnedScope(item.scope, pins)
			if item.rejected {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, root, actualRoot)
			require.ElementsMatch(t, item.want, actualPins)
		})
	}
}

func TestPinnedCommandNoPinScopeSkipsIRAndRejectsOutsideScope(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "umpire-gen-cases")
	build := exec.CommandContext(t.Context(), "go", "build", "-tags", "test_dep", "-o", binary, ".")
	output, err := build.CombinedOutput()
	require.NoError(t, err, "%s", output)
	root := t.TempDir()
	owner := filepath.Join(root, ir.FeatureDirectory, "new_form")
	require.NoError(t, os.MkdirAll(owner, 0o755))
	outside := t.TempDir()
	for _, name := range []string{"functional", "canary"} {
		for _, update := range []bool{false, true} {
			for _, directory := range []string{owner, outside} {
				t.Run(name+"/"+filepath.Base(directory)+"/"+map[bool]string{false: "check", true: "update"}[update], func(t *testing.T) {
					arguments := []string{"--kind", name}
					if update {
						arguments = append(arguments, "--update")
					}
					command := exec.CommandContext(t.Context(), binary, arguments...)
					command.Dir = directory
					output, err := command.CombinedOutput()
					if directory == outside {
						var exit *exec.ExitError
						require.ErrorAs(t, err, &exit, "%s", output)
						require.Equal(t, 1, exit.ExitCode())
					} else {
						require.NoError(t, err, "%s", output)
					}
					entries, err := os.ReadDir(directory)
					require.NoError(t, err)
					require.Empty(t, entries)
				})
			}
		}
	}
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "model", entries[0].Name(), "no central pinned tree was created")
}
