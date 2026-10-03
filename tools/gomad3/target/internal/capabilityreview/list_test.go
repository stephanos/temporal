package capabilityreview

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
)

func TestListOwnsBoundedGoListTransport(t *testing.T) {
	command := filepath.Join(t.TempDir(), "go")
	contents := []byte("#!/bin/sh\nprintf '%s' '{\"ImportPath\":\"example.com/pkg\",\"Name\":\"pkg\"}'\n")
	if err := os.WriteFile(command, contents, 0o700); err != nil {
		t.Fatal(err)
	}
	packages, err := List(context.Background(), Request{GoCommand: command, Directory: t.TempDir(), Package: "./pkg", OutputLimit: 1024, PackageLimit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 1 || packages[0].ImportPath != "example.com/pkg" {
		t.Fatalf("List() = %#v", packages)
	}
}

func TestListRejectsOverflowBeforeDecodingValidPrefix(t *testing.T) {
	command := listFixtureCommand(t, "printf '%s' '{\"ImportPath\":\"example.com/pkg\"}'; printf '%s' 'extra'")
	_, err := List(context.Background(), Request{GoCommand: command, Directory: t.TempDir(), Package: ".", OutputLimit: 36, PackageLimit: 2})
	var overflow *gocommand.OverflowError
	if !errors.As(err, &overflow) || overflow.Stream != "stdout" {
		t.Fatalf("List() error = %T %v, want stdout overflow", err, err)
	}
}

func TestListDistinguishesMalformedListingAndInvalidInput(t *testing.T) {
	malformed := listFixtureCommand(t, "printf '%s' '{bad json}'")
	_, err := List(context.Background(), Request{GoCommand: malformed, Directory: t.TempDir(), Package: ".", OutputLimit: 1024, PackageLimit: 2})
	if err == nil || !strings.Contains(err.Error(), "decode target capability closure") {
		t.Fatalf("malformed List() error = %v", err)
	}
	invalid := listFixtureCommand(t, "printf '%s' 'no required module provides package' >&2; exit 1")
	_, err = List(context.Background(), Request{GoCommand: invalid, Directory: t.TempDir(), Package: ".", OutputLimit: 1024, PackageLimit: 2})
	var commandErr *CommandError
	if !errors.As(err, &commandErr) || !commandErr.InvalidInput {
		t.Fatalf("invalid List() error = %T %v", err, err)
	}
}

func TestListPreservesDeadline(t *testing.T) {
	command := listFixtureCommand(t, "sleep 1000")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := List(ctx, Request{GoCommand: command, Directory: t.TempDir(), Package: ".", OutputLimit: 1024, PackageLimit: 2})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("List() error = %v, want deadline exceeded", err)
	}
}

func listFixtureCommand(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "go")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
