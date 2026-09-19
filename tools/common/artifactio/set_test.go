package artifactio

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetPublishValidatesBeforeReplacingManagedRoots(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	set := fixtureSet()
	writeFixtureTree(t, root, "old")
	authored := filepath.Join(root, "Temporal", "Authored.lean")
	require.NoError(t, os.WriteFile(authored, []byte("authored"), 0o600))
	roots, paths, err := validateSet(set, fixtureSetArtifacts("new"))
	require.NoError(t, err)
	resolvedRoot, err := resolveSetRoot(root)
	require.NoError(t, err)
	runtimeLikeSibling := filepath.Join(root, ".temporal-artifact-set-"+setIdentity(resolvedRoot, roots, paths))
	require.NoError(t, os.Mkdir(runtimeLikeSibling, 0o700))
	runtimeLikeAuthored := filepath.Join(runtimeLikeSibling, "Authored.txt")
	require.NoError(t, os.WriteFile(runtimeLikeAuthored, []byte("authored runtime-like sibling"), 0o600))

	err = set.Publish(root, fixtureSetArtifacts("new"), func(candidateRoot string) error {
		require.FileExists(t, filepath.Join(candidateRoot, "Temporal", "DynamicConfig.lean"))
		require.FileExists(t, filepath.Join(candidateRoot, "Temporal", "DynamicConfig", "Types.lean"))
		require.True(t, strings.HasPrefix(candidateRoot, root+string(filepath.Separator)))
		return errors.New("invalid Lean")
	})
	require.ErrorContains(t, err, "validate candidate: invalid Lean")
	requireFixtureTree(t, root, "old")

	require.NoError(t, set.Publish(root, fixtureSetArtifacts("new"), nil))
	requireFixtureTree(t, root, "new")
	authoredBytes, err := os.ReadFile(authored)
	require.NoError(t, err)
	require.Equal(t, []byte("authored"), authoredBytes)
	runtimeLikeBytes, err := os.ReadFile(runtimeLikeAuthored)
	require.NoError(t, err)
	require.Equal(t, []byte("authored runtime-like sibling"), runtimeLikeBytes)
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Name() != filepath.Base(runtimeLikeSibling) {
			require.False(t, strings.HasPrefix(entry.Name(), ".temporal-artifact-set-"), entry.Name())
		}
	}
}

func TestSetPublishRejectsIncompleteAndUnsafeSets(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tests := []struct {
		name      string
		set       Set
		artifacts map[string][]byte
		want      string
	}{
		{
			name: "missing path",
			set:  fixtureSet(),
			artifacts: map[string][]byte{
				"Temporal/DynamicConfig.lean":       []byte("facade"),
				"Temporal/DynamicConfig/Types.lean": []byte("types"),
			},
			want: "exactly the managed paths",
		},
		{
			name: "unexpected path",
			set:  fixtureSet(),
			artifacts: map[string][]byte{
				"Temporal/DynamicConfig.lean":          []byte("facade"),
				"Temporal/DynamicConfig/Types.lean":    []byte("types"),
				"Temporal/DynamicConfig/Settings.lean": []byte("settings"),
				"Temporal/DynamicConfig/Extra.lean":    []byte("extra"),
			},
			want: "exactly the managed paths",
		},
		{
			name: "traversal",
			set: Set{
				Roots: []string{"../Temporal"},
				Paths: []string{"../Temporal/DynamicConfig.lean"},
			},
			artifacts: map[string][]byte{"../Temporal/DynamicConfig.lean": []byte("unsafe")},
			want:      "unsafe",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.set.Publish(root, tt.artifacts, nil)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestSetPublishRejectsSymlinkedManagedPaths(t *testing.T) {
	t.Run("managed parent", func(t *testing.T) {
		root := t.TempDir()
		external := t.TempDir()
		if err := os.Symlink(external, filepath.Join(root, "Temporal")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		err := fixtureSet().Publish(root, fixtureSetArtifacts("new"), nil)
		require.ErrorContains(t, err, "symlink")
		entries, readErr := os.ReadDir(external)
		require.NoError(t, readErr)
		require.Empty(t, entries)
	})
	t.Run("managed artifact leaf", func(t *testing.T) {
		root := t.TempDir()
		external := filepath.Join(t.TempDir(), "outside.lean")
		require.NoError(t, os.WriteFile(external, []byte("outside"), 0o600))
		leaf := filepath.Join(root, "Temporal", "DynamicConfig", "Types.lean")
		require.NoError(t, os.MkdirAll(filepath.Dir(leaf), 0o700))
		if err := os.Symlink(external, leaf); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		err := fixtureSet().Publish(root, fixtureSetArtifacts("new"), nil)
		require.ErrorContains(t, err, "symlink")
		externalBytes, readErr := os.ReadFile(external)
		require.NoError(t, readErr)
		require.Equal(t, []byte("outside"), externalBytes)
	})
	t.Run("output root parent", func(t *testing.T) {
		directory := t.TempDir()
		physicalParent := filepath.Join(directory, "physical")
		physicalRoot := filepath.Join(physicalParent, "output")
		require.NoError(t, os.MkdirAll(physicalRoot, 0o700))
		alias := filepath.Join(directory, "alias")
		if err := os.Symlink(physicalParent, alias); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		err := fixtureSet().Publish(filepath.Join(alias, "output"), fixtureSetArtifacts("new"), nil)
		require.ErrorContains(t, err, "symlink")
		entries, readErr := os.ReadDir(physicalRoot)
		require.NoError(t, readErr)
		require.Empty(t, entries)
	})
}

func TestSetPublishRejectsConcurrentWriter(t *testing.T) {
	root := t.TempDir()
	set := fixtureSet()
	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- set.Publish(root, fixtureSetArtifacts("first"), func(string) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	err := set.Publish(root, fixtureSetArtifacts("second"), nil)
	require.ErrorContains(t, err, "concurrent writer")
	close(release)
	require.NoError(t, <-finished)
	requireFixtureTree(t, root, "first")
}

func TestSetPublishRollsBackHandledInstallationFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	set := fixtureSet()
	writeFixtureTree(t, root, "old")

	err := publishSetWithHooks(set, root, fixtureSetArtifacts("new"), nil, publishHooks{
		beforeInstall: func(index int, _ string) error {
			if index == 1 {
				return errors.New("injected failure")
			}
			return nil
		},
	})
	require.ErrorContains(t, err, "injected failure")
	requireFixtureTree(t, root, "old")
}

func TestSetPublishRecoversInterruptedInstallation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	set := fixtureSet()
	writeFixtureTree(t, root, "old")

	err := publishSetWithHooks(set, root, fixtureSetArtifacts("partial"), nil, publishHooks{
		beforeInstall: func(index int, _ string) error {
			if index == 1 {
				return errSimulatedInterruption
			}
			return nil
		},
	})
	require.ErrorIs(t, err, errSimulatedInterruption)

	err = set.Publish(root, fixtureSetArtifacts("new"), func(string) error {
		return errors.New("stop after recovery")
	})
	require.ErrorContains(t, err, "stop after recovery")
	requireFixtureTree(t, root, "old")
	require.NoError(t, set.Publish(root, fixtureSetArtifacts("new"), nil))
	requireFixtureTree(t, root, "new")
}

func TestSetPublishRecoversInterruptedCleanup(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		removeBackup string
	}{
		{name: "before cleanup"},
		{name: "after partial backup cleanup", removeBackup: "Temporal/DynamicConfig.lean"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			set := fixtureSet()
			writeFixtureTree(t, root, "old")

			err := publishSetWithHooks(set, root, fixtureSetArtifacts("new"), nil, publishHooks{
				beforeCleanup: func(transactionRoot string) error {
					state, exists, stateErr := readPublicationState(transactionRoot)
					require.NoError(t, stateErr)
					require.True(t, exists)
					require.True(t, state.Committed)
					if tt.removeBackup != "" {
						relative := filepath.FromSlash("backup/" + tt.removeBackup)
						require.NoError(t, os.RemoveAll(filepath.Join(transactionRoot, relative)))
					}
					return errSimulatedInterruption
				},
			})
			require.ErrorIs(t, err, errSimulatedInterruption)
			requireFixtureTree(t, root, "new")

			err = set.Publish(root, fixtureSetArtifacts("later"), func(string) error {
				return errors.New("stop after recovery")
			})
			require.ErrorContains(t, err, "stop after recovery")
			requireFixtureTree(t, root, "new")
		})
	}
}

func fixtureSet() Set {
	return Set{
		Roots: []string{
			"Temporal/DynamicConfig.lean",
			"Temporal/DynamicConfig",
		},
		Paths: []string{
			"Temporal/DynamicConfig.lean",
			"Temporal/DynamicConfig/Types.lean",
			"Temporal/DynamicConfig/Settings.lean",
		},
	}
}

func fixtureSetArtifacts(value string) map[string][]byte {
	return map[string][]byte{
		"Temporal/DynamicConfig.lean":          []byte("facade-" + value),
		"Temporal/DynamicConfig/Types.lean":    []byte("types-" + value),
		"Temporal/DynamicConfig/Settings.lean": []byte("settings-" + value),
	}
}

func writeFixtureTree(t *testing.T, root string, value string) {
	t.Helper()
	for path, encoded := range fixtureSetArtifacts(value) {
		absolute := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(absolute), 0o700))
		require.NoError(t, os.WriteFile(absolute, encoded, 0o600))
	}
	stale := filepath.Join(root, "Temporal", "DynamicConfig", "Stale.lean")
	require.NoError(t, os.WriteFile(stale, []byte("stale-"+value), 0o600))
}

func requireFixtureTree(t *testing.T, root string, value string) {
	t.Helper()
	for path, want := range fixtureSetArtifacts(value) {
		encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		require.NoError(t, err)
		require.Equal(t, want, encoded)
	}
	_, err := os.Stat(filepath.Join(root, "Temporal", "DynamicConfig", "Stale.lean"))
	if value == "old" {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, os.ErrNotExist)
	}
}
