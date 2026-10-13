package hostfs_test

import (
	"context"
	"errors"
	"io"
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
	if _, _, err := hostfs.OpenPath(link); !errors.Is(err, hostfs.ErrSymbolicLink) {
		t.Fatalf("symlink open: %v", err)
	}
}

func TestPublicStagingAndLinkCountDelegateToSharedHostOwner(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "result")
	staged, err := hostfs.StageContext(context.Background(), directory, ".result-*", 0o600, func(_ context.Context, writer io.Writer) error {
		_, err := io.WriteString(writer, "published")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := staged.Cleanup(); err != nil {
			t.Errorf("Cleanup() = %v", err)
		}
	}()
	published, err := staged.PublishNoReplaceContext(context.Background(), destination)
	if err != nil || !published {
		t.Fatalf("PublishNoReplaceContext() = %t, %v", published, err)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if links, known := hostfs.LinkCount(info); known && links != 1 {
		t.Fatalf("LinkCount() = %d, %t", links, known)
	}
}
