package readonlymount

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCaptureReadOnlyMountInventoryBindsCompleteTree(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "nested", "value")
	if err := os.WriteFile(path, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	mappings := []Mapping{{Source: root, Target: "/input"}}
	first, err := CaptureReadOnlyMountInventory(mappings, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if first.Entries != 3 || first.TotalBytes != 5 || first.SHA256 == "" {
		t.Fatalf("first inventory = %#v", first)
	}
	if err := os.WriteFile(path, []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := CaptureReadOnlyMountInventory(mappings, DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if first.SHA256 == second.SHA256 {
		t.Fatal("mount inventory identity did not change with file content")
	}
}
