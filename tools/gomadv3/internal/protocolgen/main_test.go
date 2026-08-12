package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGeneratesAndChecksEveryEndpoint(t *testing.T) {
	root := t.TempDir()
	for _, relative := range []string{"protocol/iowire.json", "protocol/iowire.go.tmpl", "protocol/iowire_test.go.tmpl"} {
		contents, err := os.ReadFile(filepath.Join("..", "..", relative))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := run(root, false); err != nil {
		t.Fatal(err)
	}
	if err := run(root, true); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(root, "internal", "iowire", "wire_generated.go")
	if err := os.WriteFile(stale, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(root, true); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("run(check) error = %v", err)
	}
}

func TestReadSchemaRejectsTrailingData(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "protocol", "iowire.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "iowire.json")
	contents = append(contents, []byte("\n{}\n")...)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSchema(path); err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("readSchema() error = %v", err)
	}
}
