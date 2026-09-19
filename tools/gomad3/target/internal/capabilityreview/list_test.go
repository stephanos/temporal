package capabilityreview

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
