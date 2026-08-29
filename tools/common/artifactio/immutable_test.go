package artifactio

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImmutableDirectoryRejectsManifestABADuringRead(t *testing.T) {
	root := t.TempDir()
	directory, filesA, digestA := immutableDirectoryFixture()
	destination, err := directory.Publish(root, digestA, filesA)
	require.NoError(t, err)
	manifestB := []byte("manifest-b\n")
	filesB := map[string][]byte{
		"manifest.json":    manifestB,
		"artifacts/a.json": []byte("a-b\n"),
		"artifacts/b.json": []byte("b-b\n"),
	}
	directory.MemberPaths = func(encoded []byte) ([]string, error) {
		if !bytes.Equal(encoded, filesA["manifest.json"]) && !bytes.Equal(encoded, manifestB) {
			return nil, errors.New("invalid manifest")
		}
		return []string{"artifacts/a.json", "artifacts/b.json"}, nil
	}
	directory.Validate = func(candidate map[string][]byte) error {
		expected := filesA
		if bytes.Equal(candidate["manifest.json"], manifestB) {
			expected = filesB
		}
		if !equalFileMaps(candidate, expected) {
			return errors.New("manifest and members are mixed")
		}
		return nil
	}

	aRead := make(chan struct{})
	bInstalled := make(chan struct{})
	reopen := make(chan struct{})
	aRestored := make(chan struct{})
	writerErr := make(chan error, 1)
	go func() {
		<-aRead
		for path, encoded := range filesB {
			if err := os.WriteFile(filepath.Join(destination, filepath.FromSlash(path)), encoded, 0o600); err != nil {
				writerErr <- err
				return
			}
		}
		close(bInstalled)
		<-reopen
		if err := os.WriteFile(filepath.Join(destination, "manifest.json"), filesA["manifest.json"], 0o600); err != nil {
			writerErr <- err
			return
		}
		close(aRestored)
		writerErr <- nil
	}()

	loaded, err := directory.readWithHooks(destination, digestA, true, immutableReadHooks{
		afterManifest: func() {
			close(aRead)
			<-bInstalled
		},
		beforeManifestReopen: func() {
			close(reopen)
			<-aRestored
		},
	})
	require.Error(t, err)
	require.Nil(t, loaded)
	require.NoError(t, <-writerErr)
}

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
