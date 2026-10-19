package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/generation/protocol"
)

var gomadRoot = filepath.Join("..", "..", "..", "..")

func copyTree(t *testing.T, destination string, relatives ...string) {
	t.Helper()
	for _, relative := range relatives {
		source := filepath.Join(gomadRoot, filepath.FromSlash(relative))
		err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			inside, err := filepath.Rel(gomadRoot, path)
			if err != nil {
				return err
			}
			target := filepath.Join(destination, inside)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			return os.WriteFile(target, contents, 0o600)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestProtocolgenExitStatuses(t *testing.T) {
	var stderr bytes.Buffer
	if status := run([]string{"-check", "-root", gomadRoot}, &stderr); status != 0 {
		t.Fatalf("check of the tree = %d: %s", status, stderr.String())
	}
	root := t.TempDir()
	copyTree(t, root, "internal/model/schema", "internal/model/choice/schema", "internal/model/choice/trace.go",
		"internal/model/choice/tape.go", "internal/model/runtimesource/gomad_semantics.go.txt", "internal/model/world/schema")
	stderr.Reset()
	if status := run([]string{"-check", "-root", root}, &stderr); status != 1 {
		t.Fatalf("check before generate = %d: %s", status, stderr.String())
	}
	if lines := strings.Count(stderr.String(), "stale: "); lines != len(protocol.SharedTargets()) {
		t.Fatalf("check before generate reported %d stale outputs, want every one:\n%s", lines, stderr.String())
	}
	stderr.Reset()
	if status := run([]string{"-root", root}, &stderr); status != 0 {
		t.Fatalf("generate = %d: %s", status, stderr.String())
	}
	edited := protocol.SharedTargets()[len(protocol.SharedTargets())-1].Path
	path := filepath.Join(root, filepath.FromSlash(edited))
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(contents, "// edited\n"...), 0o600); err != nil {
		t.Fatal(err)
	}
	stderr.Reset()
	if status := run([]string{"-check", "-root", root}, &stderr); status != 1 || stderr.String() != "stale: "+edited+"\n" {
		t.Fatalf("check after edit = %d: %q", status, stderr.String())
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "model", "schema", "iowire.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, arguments := range [][]string{{"-check", "-root", root}, {"-unknown"}, {"-check", "extra"}} {
		stderr.Reset()
		if status := run(arguments, &stderr); status != 2 {
			t.Fatalf("run(%q) = %d, want 2: %s", arguments, status, stderr.String())
		}
	}
}

func TestModuleRootSkipsNestedModules(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "internal", "backends", "native")
	if err := os.MkdirAll(filepath.Join(nested, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+modulePath+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "go.mod"), []byte("module "+modulePath+"/internal/backends/native\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if found, err := moduleRoot(filepath.Join(nested, "cmd")); err != nil || found != root {
		t.Fatalf("module root = %q, %v; want %q", found, err, root)
	}
	if _, err := moduleRoot(t.TempDir()); err == nil {
		t.Fatal("found a gomad module outside one")
	}
}
