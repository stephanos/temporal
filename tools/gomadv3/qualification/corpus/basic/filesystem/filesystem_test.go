package filesystem

import (
	"os"
	"testing"
)

func TestFileLifecyclePreservesContentsAndMetadata(t *testing.T) {
	if err := os.Mkdir("workspace", 0o750); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile("workspace/pending", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o640)
	if err != nil {
		t.Fatal(err)
	}
	if written, writeErr := file.Write([]byte("committed-state")); writeErr != nil || written != len("committed-state") {
		t.Fatalf("write = %d, %v", written, writeErr)
	}
	if err = file.Sync(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename("workspace/pending", "workspace/committed"); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile("workspace/committed")
	if err != nil || string(contents) != "committed-state" {
		t.Fatalf("contents = %q, %v", contents, err)
	}
	info, err := os.Stat("workspace/committed")
	if err != nil || info.Mode().Perm() != 0o640 || info.Size() != int64(len(contents)) {
		t.Fatalf("file info = %#v, %v", info, err)
	}
	if err = os.Remove("workspace/committed"); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat("workspace/committed"); !os.IsNotExist(err) {
		t.Fatalf("removed file stat error = %v", err)
	}
}
