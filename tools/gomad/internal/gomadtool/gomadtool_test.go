package gomadtool

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGomadExecutable(t *testing.T) {
	t.Run("environment override", func(t *testing.T) {
		t.Setenv("GOMADTOOL", "/path/to/gomad")

		path, err := findGomadExecutable()

		if err != nil {
			t.Fatal(err)
		}
		if path != "/path/to/gomad" {
			t.Fatalf("findGomadExecutable() = %q, want %q", path, "/path/to/gomad")
		}
	})

	t.Run("path", func(t *testing.T) {
		t.Setenv("GOMADTOOL", "")
		binDir := t.TempDir()
		binPath := filepath.Join(binDir, "gomad")
		if err := os.WriteFile(binPath, nil, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", binDir)

		path, err := findGomadExecutable()

		if err != nil {
			t.Fatal(err)
		}
		if path != binPath {
			t.Fatalf("findGomadExecutable() = %q, want %q", path, binPath)
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Setenv("GOMADTOOL", "")
		t.Setenv("PATH", t.TempDir())

		_, err := findGomadExecutable()

		if err == nil {
			t.Fatal("findGomadExecutable() succeeded, want error")
		}
	})
}
