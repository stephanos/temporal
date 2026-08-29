package artifactio

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImmutableDirectoryInterruptionExposesNoDigestDirectory(t *testing.T) {
	root := t.TempDir()
	directory, files, digest := immutableDirectoryFixture()

	destination, err := publishImmutableDirectoryWithHooks(directory, root, digest, files, immutablePublishHooks{
		beforeInstall: func() error { return errSimulatedInterruption },
	})
	require.ErrorIs(t, err, errSimulatedInterruption)
	require.Empty(t, destination)
	_, err = os.Lstat(filepath.Join(root, "sets", digest))
	require.ErrorIs(t, err, os.ErrNotExist)

	destination, err = directory.Publish(root, digest, files)
	require.NoError(t, err)
	loaded, err := directory.Read(destination)
	require.NoError(t, err)
	require.Equal(t, files, loaded)

	interruptedDestination, err := publishImmutableDirectoryWithHooks(directory, root, digest, files, immutablePublishHooks{
		beforeInstall: func() error { return errSimulatedInterruption },
	})
	require.ErrorIs(t, err, errSimulatedInterruption)
	require.Empty(t, interruptedDestination)
	loaded, err = directory.Read(destination)
	require.NoError(t, err)
	require.Equal(t, files, loaded)
}

func TestImmutableDirectoryRecoversUnreachableStaging(t *testing.T) {
	root := t.TempDir()
	directory, files, digest := immutableDirectoryFixture()
	setsRoot := filepath.Join(root, "sets")
	require.NoError(t, os.Mkdir(setsRoot, 0o700))
	stale := filepath.Join(setsRoot, immutableSetStagingPrefix+digest+"-stale")
	require.NoError(t, os.Mkdir(stale, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(stale, "partial"), []byte("partial"), 0o600))

	_, err := directory.Publish(root, digest, files)
	require.NoError(t, err)
	_, err = os.Lstat(stale)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestImmutableDirectoryRejectsConcurrentWriter(t *testing.T) {
	root := t.TempDir()
	directory, files, digest := immutableDirectoryFixture()
	entered := make(chan struct{})
	release := make(chan struct{})
	directory.Validate = func(map[string][]byte) error {
		select {
		case <-entered:
		default:
			close(entered)
			<-release
		}
		return nil
	}
	finished := make(chan error, 1)
	go func() {
		_, err := directory.Publish(root, digest, files)
		finished <- err
	}()
	<-entered

	_, err := directory.Publish(root, digest, files)
	require.ErrorContains(t, err, "concurrent writer")
	close(release)
	require.NoError(t, <-finished)
}

func immutableDirectoryFixture() (ImmutableDirectory, map[string][]byte, string) {
	manifest := []byte("manifest\n")
	files := map[string][]byte{
		"manifest.json":    manifest,
		"artifacts/a.json": []byte("a\n"),
		"artifacts/b.json": []byte("b\n"),
	}
	directory := ImmutableDirectory{
		ManifestPath:     "manifest.json",
		MaximumFileBytes: 1024,
		MemberPaths: func(encoded []byte) ([]string, error) {
			if string(encoded) != string(manifest) {
				return nil, errors.New("invalid manifest")
			}
			return []string{"artifacts/a.json", "artifacts/b.json"}, nil
		},
		Validate: func(candidate map[string][]byte) error {
			for path, expected := range files {
				if string(candidate[path]) != string(expected) {
					return errors.New("conflicting bytes")
				}
			}
			return nil
		},
	}
	hash := sha256.Sum256(manifest)
	return directory, files, hex.EncodeToString(hash[:])
}
