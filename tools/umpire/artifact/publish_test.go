package artifact_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
)

func TestPublishSetLoadsOneCompleteImmutableDirectory(t *testing.T) {
	root := t.TempDir()
	admitted, err := artifact.AdmitSet(artifactSetFixtureMembers(t))
	require.NoError(t, err)

	destination, err := artifact.PublishSet(root, admitted)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "sets", strings.TrimPrefix(admitted.ManifestSHA256(), "sha256:")), destination)

	loaded, err := artifact.LoadSet(destination)
	require.NoError(t, err)
	require.Equal(t, admitted.Identity(), loaded.Identity())
	require.Equal(t, admitted.Checksum(), loaded.Checksum())
	require.Equal(t, admitted.ManifestSHA256(), loaded.ManifestSHA256())
	require.Equal(t, admitted.ManifestBytes(), loaded.ManifestBytes())

	identicalDestination, err := artifact.PublishSet(root, admitted)
	require.NoError(t, err)
	require.Equal(t, destination, identicalDestination)
}

func TestAdmitSetFilesOwnsExactInputSnapshot(t *testing.T) {
	members := artifactSetFixtureMembers(t)
	admitted, err := artifact.AdmitSet(members)
	require.NoError(t, err)
	files := map[string][]byte{"manifest.json": admitted.ManifestBytes()}
	for _, member := range members {
		files[member.Path] = bytes.Clone(member.Encoded)
	}

	fromFiles, err := artifact.AdmitSetFiles(files)
	require.NoError(t, err)
	wantManifest := fromFiles.ManifestBytes()
	files["manifest.json"][0] = '['
	files[members[0].Path][0] = '['

	require.Equal(t, admitted.Identity(), fromFiles.Identity())
	require.Equal(t, wantManifest, fromFiles.ManifestBytes())
}

func TestLoadSetRejectsUnsafeOrConflictingDestinations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{
			name: "conflicting member bytes",
			mutate: func(t *testing.T, destination string) {
				t.Helper()
				path := filepath.Join(destination, "artifacts", "experiment.json")
				encoded, err := os.ReadFile(path)
				require.NoError(t, err)
				encoded = bytes.Clone(encoded)
				encoded[0] = '['
				require.NoError(t, os.WriteFile(path, encoded, 0o600))
			},
		},
		{
			name: "unexpected file",
			mutate: func(t *testing.T, destination string) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(destination, "extra.json"), []byte("{}\n"), 0o600))
			},
		},
		{
			name: "permissive member",
			mutate: func(t *testing.T, destination string) {
				t.Helper()
				require.NoError(t, os.Chmod(filepath.Join(destination, "manifest.json"), 0o644))
			},
		},
		{
			name: "non-regular member",
			mutate: func(t *testing.T, destination string) {
				t.Helper()
				path := filepath.Join(destination, "manifest.json")
				require.NoError(t, os.Remove(path))
				require.NoError(t, os.Mkdir(path, 0o700))
			},
		},
		{
			name: "permissive sets directory",
			mutate: func(t *testing.T, destination string) {
				t.Helper()
				require.NoError(t, os.Chmod(filepath.Dir(destination), 0o755))
			},
		},
		{
			name: "symlinked member",
			mutate: func(t *testing.T, destination string) {
				t.Helper()
				path := filepath.Join(destination, "manifest.json")
				external := filepath.Join(t.TempDir(), "manifest.json")
				require.NoError(t, os.WriteFile(external, []byte("{}\n"), 0o600))
				require.NoError(t, os.Remove(path))
				if err := os.Symlink(external, path); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			admitted, err := artifact.AdmitSet(artifactSetFixtureMembers(t))
			require.NoError(t, err)
			destination, err := artifact.PublishSet(root, admitted)
			require.NoError(t, err)
			test.mutate(t, destination)

			loaded, err := artifact.LoadSet(destination)
			require.Error(t, err)
			require.Empty(t, loaded.Identity())
			_, publishErr := artifact.PublishSet(root, admitted)
			require.Error(t, publishErr)
		})
	}
}

func TestPublishSetDoesNotRepairConflictingDestination(t *testing.T) {
	root := t.TempDir()
	admitted, err := artifact.AdmitSet(artifactSetFixtureMembers(t))
	require.NoError(t, err)
	destination, err := artifact.PublishSet(root, admitted)
	require.NoError(t, err)
	path := filepath.Join(destination, "artifacts", "experiment.json")
	conflict := []byte("conflicting bytes\n")
	require.NoError(t, os.WriteFile(path, conflict, 0o600))

	_, err = artifact.PublishSet(root, admitted)
	require.Error(t, err)
	got, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	require.Equal(t, conflict, got)
}

func TestLoadSetRequiresExactDigestDirectory(t *testing.T) {
	root := t.TempDir()
	admitted, err := artifact.AdmitSet(artifactSetFixtureMembers(t))
	require.NoError(t, err)
	destination, err := artifact.PublishSet(root, admitted)
	require.NoError(t, err)

	alias := filepath.Join(root, "sets", strings.Repeat("0", 64))
	require.NoError(t, os.Rename(destination, alias))
	loaded, err := artifact.LoadSet(alias)
	require.Error(t, err)
	require.Empty(t, loaded.Identity())

	outside := filepath.Join(root, filepath.Base(alias))
	require.NoError(t, os.Rename(alias, outside))
	loaded, err = artifact.LoadSet(outside)
	require.Error(t, err)
	require.Empty(t, loaded.Identity())
}

func TestPublishSetReadersObserveAbsenceOrOneCompleteSet(t *testing.T) {
	root := t.TempDir()
	members := artifactSetFixtureMembers(t)
	oldSet, err := artifact.AdmitSet(members)
	require.NoError(t, err)
	oldDestination, err := artifact.PublishSet(root, oldSet)
	require.NoError(t, err)
	newSet, err := artifact.AdmitSet(members[:2])
	require.NoError(t, err)
	newDestination := filepath.Join(root, "sets", strings.TrimPrefix(newSet.ManifestSHA256(), "sha256:"))

	start := make(chan struct{})
	errorsSeen := make(chan error, 64)
	var readers sync.WaitGroup
	for range 32 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			<-start
			for range 25 {
				loadedOld, loadOldErr := artifact.LoadSet(oldDestination)
				if loadOldErr != nil || loadedOld.Identity() != oldSet.Identity() {
					errorsSeen <- loadOldErr
					return
				}
				loadedNew, loadNewErr := artifact.LoadSet(newDestination)
				if loadNewErr == nil && loadedNew.Identity() != newSet.Identity() {
					errorsSeen <- fmt.Errorf("loaded new set identity %q, want %q", loadedNew.Identity(), newSet.Identity())
					return
				}
				if loadNewErr != nil && !errors.Is(loadNewErr, os.ErrNotExist) {
					errorsSeen <- loadNewErr
					return
				}
			}
		}()
	}
	close(start)
	published, err := artifact.PublishSet(root, newSet)
	require.NoError(t, err)
	require.Equal(t, newDestination, published)
	readers.Wait()
	close(errorsSeen)
	for readErr := range errorsSeen {
		require.NoError(t, readErr)
	}
}
