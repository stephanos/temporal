package hostfs_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/hostfs"
)

func TestPublicReadAcceptsCompleteBoundedBytesAndRejectsOverflowOrLinks(t *testing.T) {
	name := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(name, []byte("frozen"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := hostfs.ReadBounded(name, 6)
	if err != nil || string(data) != "frozen" {
		t.Fatalf("bounded input = %q, %v", data, err)
	}
	if _, err := hostfs.ReadBounded(name, 5); err == nil {
		t.Fatal("oversize input accepted")
	}
	link := name + "-link"
	if err := os.Symlink(name, link); err != nil {
		t.Fatal(err)
	}
	if _, err := hostfs.ReadBounded(link, 6); err == nil {
		t.Fatal("symlink input accepted")
	}
}
