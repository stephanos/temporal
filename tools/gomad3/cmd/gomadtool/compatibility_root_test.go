package main

import (
	"path/filepath"
	"testing"
)

func TestCompatibilityPackPathsHonorAnExternalRoot(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()

	_, compatibilityRoot, request, err := resolveCompatibilityPackPaths(root, "", "internal/compatibilitypack/requests/x.json")
	if err != nil || compatibilityRoot != filepath.Join(root, "internal", "compatibilitypack") || request != filepath.Join(root, "internal", "compatibilitypack", "requests", "x.json") {
		t.Fatalf("default paths = %q, %q, %v", compatibilityRoot, request, err)
	}

	_, compatibilityRoot, request, err = resolveCompatibilityPackPaths(root, external, "requests/x.json")
	if err != nil || compatibilityRoot != external || request != filepath.Join(external, "requests", "x.json") {
		t.Fatalf("external paths = %q, %q, %v", compatibilityRoot, request, err)
	}
	if !pathWithin(compatibilityRoot, request) {
		t.Fatal("external request is not below the external root")
	}

	if _, _, _, err := resolveCompatibilityPackPaths(root, external, filepath.Join(root, "requests", "x.json")); err == nil {
		t.Fatal("a request outside the external root was accepted")
	}
	for _, override := range []string{"relative/root", external + "/../" + filepath.Base(external)} {
		if _, _, _, err := resolveCompatibilityPackPaths(root, override, "requests/x.json"); err == nil {
			t.Fatalf("--compatibility-root %q was accepted", override)
		}
	}
}

func TestCompatibilityPathBaseKeepsDefaultPathsRootRelative(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if base := compatibilityPathBase(root, filepath.Join(root, "internal", "compatibilitypack"), ""); base != root {
		t.Fatalf("default base = %q, want %q", base, root)
	}
	if base := compatibilityPathBase(root, external, external); base != external {
		t.Fatalf("external base = %q, want %q", base, external)
	}
}
