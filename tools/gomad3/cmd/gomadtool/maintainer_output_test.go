package main

import (
	"bytes"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

func TestRunMaintainerOutputTerminalReports(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	maintainerWrite(t, fixture, "patch", "patch")
	maintainerWrite(t, fixture, "overlay/gomad.go", "package gomad")
	for _, test := range []struct {
		name      string
		arguments []string
		want      string
	}{
		{"patch-validate", []string{"patch-validate", "--root=" + root}, "gomad3 patch and overlay inputs are valid\n"},
		{"script-validate", []string{"script-validate", "--root=" + root}, "gomad3 script ownership is valid\n"},
		{"build-key", []string{"build-key", "--go-version=go1.26.4", "--archive-sha256=archive", "--patch=" + filepath.Join(fixture, "patch"), "--overlay=" + filepath.Join(fixture, "overlay"), "--host-os=darwin", "--host-arch=arm64", "--bootstrap-version=bootstrap", "--recipe-version=recipe", "--build-path=/usr/bin:/bin", "--bash-path=/bin/bash", "--bash-version=5.2"}, "959c6cfac1ff559191f442e7054ad210e69b770021c3723062ab7860c06fb1a2\n"},
	} {
		t.Run(test.name, func(t *testing.T) { maintainerReport(t, test.arguments, test.want, fixture) })
	}
}

func TestRunMaintainerOutputBoundaryDiscovery(t *testing.T) {
	arguments := []string{"boundary-generate", "--discover"}
	var stdout, stderr bytes.Buffer
	if status := run(arguments, &stdout, &stderr); status != 0 || stdout.Len() == 0 || stderr.Len() != 0 {
		t.Fatalf("healthy discovery status = %d, stdout = %q, stderr = %q", status, &stdout, &stderr)
	}
	t.Run("literal control", func(t *testing.T) {
		path := filepath.Join("testdata", "maintainer-output", "boundary-"+runtime.GOOS+"-"+runtime.GOARCH+"-"+runtime.Version()+".txt")
		want, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			t.Skip("literal stock-source candidate control is unavailable for this platform/toolchain")
		}
		if err != nil {
			t.Fatal(err)
		}
		if stdout.String() != string(want) {
			t.Fatalf("healthy discovery = %q, want %q", &stdout, want)
		}
	})
	output := newRefreshOutput(t, 0)
	status := run(arguments, output, &stderr)
	if !errors.Is(output.err, syscall.EBADF) {
		t.Fatalf("stdout write error = %v, want EBADF", output.err)
	}
	t.Log("stdout write observed EBADF")
	if status != 3 || stderr.Len() != 0 {
		t.Fatalf("stdout failure status = %d, want 3; stderr = %q", status, &stderr)
	}
}

func TestRunMaintainerOutputAuthoring(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	request, err := os.ReadFile(filepath.Join("testdata", "maintainer-output", "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	const approval = "sha256:365c49f6d1476a60efdc37b3f699ad5d0f0e00a813bcf0e1cfb269a9a7003152"
	for _, operation := range []string{"review", "generate-all", "generate-approved", "check"} {
		t.Run(operation, func(t *testing.T) {
			var healthy map[string]string
			for _, failed := range []bool{false, true} {
				packRoot := t.TempDir()
				maintainerWrite(t, packRoot, "requests/example-pack.json", string(request))
				base := []string{"compatibility-pack", "generate", "--root=" + root, "--compatibility-root=" + packRoot}
				approve := append(append([]string{}, base...), "--request=requests/example-pack.json", "--approve-review="+approval)
				if operation == "check" {
					var stdout, stderr bytes.Buffer
					if status := run(approve, &stdout, &stderr); status != 0 {
						t.Fatalf("seed approval status = %d: %s", status, &stderr)
					}
				}
				arguments, want := base, "generated compatibility packs\n"
				switch operation {
				case "review":
					arguments = []string{"compatibility-pack", "review", "--root=" + root, "--compatibility-root=" + packRoot, "--request=requests/example-pack.json", "--output=reports/example-pack.md"}
					want = approval + "\n"
				case "generate-approved":
					arguments, want = approve, "generated compatibility pack example-pack\n"
				case "check":
					arguments = []string{"compatibility-pack", "check", "--root=" + root, "--compatibility-root=" + packRoot}
					want = "compatibility packs are current\n"
				}
				before := maintainerSnapshot(t, packRoot)
				var stdout, stderr bytes.Buffer
				status, wantStatus := 0, 0
				if failed {
					output := newRefreshOutput(t, 0)
					status = run(arguments, output, &stderr)
					if !errors.Is(output.err, syscall.EBADF) {
						t.Fatalf("stdout write error = %v, want EBADF", output.err)
					}
					t.Log("stdout write observed EBADF")
					wantStatus = 3
				} else {
					status = run(arguments, &stdout, &stderr)
					if stdout.String() != want {
						t.Fatalf("healthy stdout = %q, want %q; stderr = %s", &stdout, want, &stderr)
					}
				}
				after := maintainerSnapshot(t, packRoot)
				if failed {
					if !maps.Equal(healthy, after) {
						t.Fatal("stdout failure changed completed authoring publication")
					}
				} else {
					healthy = after
				}
				if operation == "check" && !maps.Equal(before, after) {
					t.Fatal("check changed authoring state")
				}
				if operation == "generate-all" {
					if _, err := os.Stat(filepath.Join(packRoot, "packs", "example-pack.json")); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("unapproved pack was published: %v", err)
					}
				}
				if status != wantStatus || stderr.Len() != 0 {
					t.Fatalf("status = %d, want %d; stderr = %q", status, wantStatus, &stderr)
				}
			}
		})
	}
}

func TestRunMaintainerOutputPrimaryFailures(t *testing.T) {
	missing := t.TempDir()
	for _, command := range []string{"build-key", "patch-validate", "script-validate", "patch-materialize", "patch-regenerate", "test", "toolchain-build", "boundary-generate", "compatibility-pack", "upgrade-dossier"} {
		for _, test := range []struct {
			name      string
			arguments []string
			status    int
		}{
			{"invalid", []string{command, "--not-a-flag"}, 2},
			{"operation", []string{command, "--root=" + missing}, 1},
		} {
			if test.name == "operation" && (command != "patch-validate" && command != "script-validate" && command != "boundary-generate") {
				continue
			}
			t.Run(command+"/"+test.name, func(t *testing.T) {
				output, diagnostics := newRefreshOutput(t, 0), newRefreshOutput(t, 0)
				status := run(test.arguments, output, diagnostics)
				if output.err != nil || output.Len() != 0 {
					t.Fatalf("primary failure attempted stdout: %v", output.err)
				}
				if !errors.Is(diagnostics.err, syscall.EBADF) {
					t.Fatalf("stderr write error = %v, want EBADF", diagnostics.err)
				}
				if status != test.status {
					t.Fatalf("primary status = %d, want %d", status, test.status)
				}
			})
		}
	}
}

func maintainerReport(t *testing.T, arguments []string, want, publicationRoot string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if status := run(arguments, &stdout, &stderr); status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("healthy status = %d, stdout = %q, want %q, stderr = %q", status, &stdout, want, &stderr)
	}
	before := maintainerSnapshot(t, publicationRoot)
	output := newRefreshOutput(t, 0)
	status := run(arguments, output, &stderr)
	if !errors.Is(output.err, syscall.EBADF) {
		t.Fatalf("stdout write error = %v, want EBADF", output.err)
	}
	t.Log("stdout write observed EBADF")
	if after := maintainerSnapshot(t, publicationRoot); !maps.Equal(before, after) {
		t.Fatal("completed publication changed after stdout failure")
	}
	if status != 3 || stderr.Len() != 0 {
		t.Fatalf("stdout failure status = %d, want 3; stderr = %q", status, &stderr)
	}
}

func maintainerSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	if root == "" {
		return snapshot
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		if entry.IsDir() {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		snapshot[filepath.ToSlash(relative)] = string(contents)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func maintainerWrite(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
