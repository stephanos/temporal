package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitFileDistinguishesAnIntroducedFileFromAnInvalidRevision(t *testing.T) {
	repository := t.TempDir()
	root := filepath.Join(repository, "tools", "gomadv3")
	if err := os.MkdirAll(filepath.Join(root, "boundary"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "version.json"), []byte("version\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, arguments := range [][]string{
		{"init", "-q"},
		{"config", "user.name", "Gomad Test"},
		{"config", "user.email", "gomad@example.test"},
		{"add", "tools/gomadv3/version.json"},
		{"commit", "-q", "-m", "fixture"},
	} {
		command := exec.Command("git", append([]string{"-C", repository}, arguments...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
	}

	contents, found, err := gitFile(context.Background(), root, "HEAD", "version.json")
	if err != nil || !found || string(contents) != "version\n" {
		t.Fatalf("existing git file = %q, found = %t, error = %v", contents, found, err)
	}
	contents, found, err = gitFile(context.Background(), root, "HEAD", "boundary/manifest.json")
	if err != nil || found || contents != nil {
		t.Fatalf("introduced git file = %q, found = %t, error = %v", contents, found, err)
	}
	if _, _, err := gitFile(context.Background(), root, "missing-revision", "version.json"); err == nil {
		t.Fatal("gitFile accepted an invalid baseline revision")
	}
}
