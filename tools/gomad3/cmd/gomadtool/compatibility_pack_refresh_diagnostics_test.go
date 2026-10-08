package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack/authoring"
)

func TestRunCompatibilityPackRefreshDiagnosticsPreservePrimaryStatus(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	goCommand, _ := refreshSourceGo(t)
	for _, name := range []string{"usage", "authoring root", "missing table", "missing requests", "missing Go", "live impact", "saved impact", "authoring refresh"} {
		t.Run(name, func(t *testing.T) {
			packRoot := t.TempDir()
			maintainerWrite(t, packRoot, authoring.WorkingDirectoriesFile, `{"requests":[],"schema":"`+authoring.WorkingDirectoriesSchema+`"}`+"\n")
			requests := filepath.Join(packRoot, "requests")
			if err := os.Mkdir(requests, 0o700); err != nil {
				t.Fatal(err)
			}
			arguments := []string{"compatibility-pack", "refresh", "--root=" + root, "--compatibility-root=" + packRoot, "--go=" + goCommand}
			status, diagnostic := 2, ""
			switch name {
			case "usage":
				arguments, diagnostic = []string{"compatibility-pack", "refresh"}, compatibilityPackRefreshUsage+"\n"
			case "authoring root":
				arguments = []string{"compatibility-pack", "refresh", "--root=" + root, "--compatibility-root=relative"}
				diagnostic = "--compatibility-root \"relative\" must be an absolute, clean path\n"
			case "missing table":
				if err := os.Remove(filepath.Join(packRoot, authoring.WorkingDirectoriesFile)); err != nil {
					t.Fatal(err)
				}
				diagnostic = "compatibility-pack working-directory table working-directories.json is not a bounded regular file\n"
			case "missing requests":
				if err := os.Remove(requests); err != nil {
					t.Fatal(err)
				}
				_, readErr := os.ReadDir(requests)
				if !errors.Is(readErr, os.ErrNotExist) {
					t.Fatalf("request directory refusal = %v", readErr)
				}
				status, diagnostic = 3, fmt.Sprintf("read compatibility-pack requests: %v\n", readErr)
			case "missing Go":
				missing := filepath.Join(packRoot, "missing-go")
				arguments = append(arguments, "--go="+missing)
				_, pathErr := exec.LookPath(missing)
				if !errors.Is(pathErr, os.ErrNotExist) {
					t.Fatalf("Go executable refusal = %v", pathErr)
				}
				status, diagnostic = 3, fmt.Sprintf("compatibility-pack refresh requires a go command; set GOMAD3_BOOTSTRAP_GO or pass --go: %v\n", pathErr)
			case "live impact":
				fixture := newResolvedRefreshFixture(t)
				packRoot = fixture.packRoot
				fixture.table(`{"directory":"../missing","request":"pack-a"},{"directory":"../pack-b","request":"pack-b"}`)
				arguments = []string{"compatibility-pack", "refresh", "--root=" + root, "--compatibility-root=" + packRoot, "--go=" + goCommand}
				_, readErr := os.ReadFile(filepath.Join(filepath.Dir(packRoot), "missing", "go.mod"))
				if !errors.Is(readErr, os.ErrNotExist) {
					t.Fatalf("live module refusal = %v", readErr)
				}
				diagnostic = fmt.Sprintf("read module file: %v\n", readErr)
			case "saved impact":
				path := filepath.Join(packRoot, "invalid-impact.json")
				maintainerWrite(t, packRoot, "invalid-impact.json", "invalid JSON")
				arguments = append(arguments, "--impact-report="+path)
				diagnostic = "invalid pin-impact report\n"
			case "authoring refresh":
				previous := compatibilityPackReviewer
				compatibilityPackReviewer = func(string) authoring.Reviewer {
					if err := os.Remove(requests); err != nil {
						t.Fatal(err)
					}
					return func(context.Context, authoring.Request, string) (authoring.CapabilityReview, error) {
						t.Fatal("missing request directory must refuse before discovery")
						return authoring.CapabilityReview{}, nil
					}
				}
				t.Cleanup(func() { compatibilityPackReviewer = previous })
				status = 3
				diagnostic = fmt.Sprintf("read compatibility-pack requests: %v\n", &os.PathError{Op: "open", Path: requests, Err: syscall.ENOENT})
			}
			before := maintainerSnapshot(t, packRoot)
			for _, failed := range []bool{false, true} {
				if name == "authoring refresh" && failed {
					if err := os.Mkdir(requests, 0o700); err != nil {
						t.Fatal(err)
					}
				}
				var stdout, stderr bytes.Buffer
				actual := 0
				if failed {
					writer := newRefreshOutput(t, 0)
					actual = run(arguments, &stdout, writer)
					if !errors.Is(writer.err, syscall.EBADF) {
						t.Fatalf("diagnostic write error = %v, want actual EBADF", writer.err)
					}
					t.Log("diagnostic write executed and returned EBADF")
				} else {
					actual = run(arguments, &stdout, &stderr)
					if stderr.String() != diagnostic {
						t.Fatalf("healthy diagnostic = %q, want %q", &stderr, diagnostic)
					}
					t.Logf("healthy status %d diagnostic %q", actual, &stderr)
				}
				if actual != status || stdout.Len() != 0 || !maps.Equal(before, maintainerSnapshot(t, packRoot)) {
					t.Fatalf("failed=%t primary status=%d stdout=%q, want %d and preserved files", failed, actual, &stdout, status)
				}
			}
		})
	}
	t.Run("absolute root", func(t *testing.T) {
		directory := t.TempDir()
		t.Chdir(directory)
		t.Setenv("PWD", "")
		if err := os.Remove(directory); err != nil {
			t.Fatal(err)
		}
		_, workingDirectoryError := os.Getwd()
		if !errors.Is(workingDirectoryError, os.ErrNotExist) {
			t.Fatalf("deleted working directory error = %v", workingDirectoryError)
		}
		arguments := []string{"compatibility-pack", "refresh", "--root=."}
		var stdout, stderr bytes.Buffer
		if status := run(arguments, &stdout, &stderr); status != 2 || stderr.String() != workingDirectoryError.Error()+"\n" || stdout.Len() != 0 {
			t.Fatalf("healthy absolute-root refusal status=%d stdout=%q stderr=%q", status, &stdout, &stderr)
		}
		diagnostics := newRefreshOutput(t, 0)
		if status := run(arguments, &stdout, diagnostics); status != 2 || !errors.Is(diagnostics.err, syscall.EBADF) || stdout.Len() != 0 {
			t.Fatalf("failed absolute-root diagnostic status=%d error=%v", status, diagnostics.err)
		}
		t.Logf("healthy status 2 diagnostic %q; failed diagnostic executed actual EBADF", &stderr)
	})
}
