//go:build linux || darwin

package hostfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRenameNoReplacePreservesExistingDestination(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source")
	destination := filepath.Join(directory, "destination")
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("destination"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RenameNoReplace(source, destination); !errors.Is(err, os.ErrExist) {
		t.Fatalf("RenameNoReplace() error = %v, want destination-exists", err)
	}
	if contents, err := os.ReadFile(source); err != nil || string(contents) != "source" {
		t.Fatalf("source = %q, %v", contents, err)
	}
	if contents, err := os.ReadFile(destination); err != nil || string(contents) != "destination" {
		t.Fatalf("destination = %q, %v", contents, err)
	}
}
