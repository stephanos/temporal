package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWorkingDirectory(t *testing.T) {
	module := t.TempDir()
	if err := os.WriteFile(filepath.Join(module, "go.mod"), []byte("module example.com/downstream\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	notModule := t.TempDir()
	current := func() (string, error) { return "/current", nil }

	if resolved, err := resolveWorkingDirectory("", current); err != nil || resolved != "/current" {
		t.Fatalf("resolveWorkingDirectory(default) = %q, %v", resolved, err)
	}
	if resolved, err := resolveWorkingDirectory(module, current); err != nil || resolved != module {
		t.Fatalf("resolveWorkingDirectory(module) = %q, %v", resolved, err)
	}
	for name, directory := range map[string]string{
		"relative":       "downstream",
		"unclean":        module + "/../" + filepath.Base(module),
		"missing":        filepath.Join(module, "absent"),
		"not-module":     notModule,
		"file-not-a-dir": filepath.Join(module, "go.mod"),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := resolveWorkingDirectory(directory, current)
			var invalid invalidWorkingDirectoryError
			if !errors.As(err, &invalid) {
				t.Fatalf("resolveWorkingDirectory(%q) error = %v, want invalid input", directory, err)
			}
		})
	}

	failure := errors.New("getwd failed")
	if _, err := resolveWorkingDirectory("", func() (string, error) { return "", failure }); !errors.Is(err, failure) {
		t.Fatalf("resolveWorkingDirectory(getwd failure) error = %v", err)
	}
}
