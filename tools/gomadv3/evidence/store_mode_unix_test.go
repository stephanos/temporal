//go:build unix

package evidence

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestPublishEnforcesModesIndependentOfUmask(t *testing.T) {
	input := artifactInput(t)
	store := Store{Root: t.TempDir()}
	oldUmask := syscall.Umask(0o777)
	defer syscall.Umask(oldUmask)

	published, err := store.PublishArtifact(input)
	if err != nil {
		t.Fatal(err)
	}
	for path, mode := range map[string]os.FileMode{
		".":                         0o700,
		"manifest.json":             0o600,
		"target":                    0o700,
		"stdout":                    0o600,
		"world":                     0o700,
		"world/snapshot.json":       0o600,
		"world/final-snapshot.json": 0o600,
		"world/transitions.jsonl":   0o600,
	} {
		info, statErr := os.Stat(filepath.Join(published.Path, path))
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm() != mode {
			t.Fatalf("%s mode = %#o, want %#o", path, info.Mode().Perm(), mode)
		}
	}
}
