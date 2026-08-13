package scriptpolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAcceptsReviewedScriptOwnership(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(root); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsUnownedAndBashPolicyScripts(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, path, contents, want string
	}{
		{name: "unowned", path: "new-policy.sh", contents: "#!/bin/sh\n", want: "unowned script"},
		{name: "bash policy", path: "exec.sh", contents: "#!/usr/bin/env bash\n[[ -n $value ]]\n", want: "must be strict POSIX shell"},
		{name: "perl policy", path: "exec.sh", contents: "#!/bin/sh\nperl -e policy\n", want: "must not invoke Perl"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := t.TempDir()
			copyReviewedScripts(t, root, fixture)
			path := filepath.Join(fixture, test.path)
			if err := os.WriteFile(path, []byte(test.contents), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := Validate(fixture); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v, want %q", err, test.want)
			}
		})
	}
}

func copyReviewedScripts(t *testing.T, source, destination string) {
	t.Helper()
	for path := range reviewedScripts {
		contents, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(destination, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, contents, 0o700); err != nil {
			t.Fatal(err)
		}
	}
}
