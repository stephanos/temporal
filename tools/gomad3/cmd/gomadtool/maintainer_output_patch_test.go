package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

const maintainerPatch = "diff --git a/src/runtime/proc.go b/src/runtime/proc.go\n--- a/src/runtime/proc.go\n+++ b/src/runtime/proc.go\n@@ -2,2 +2,2 @@\n \n-func target() {}\n+func replacement() {}\n"

const maintainerRegeneratedPatch = "diff --git a/src/runtime/proc.go b/src/runtime/proc.go\nindex f85fc2f..5287f8a 100644\n--- a/src/runtime/proc.go\n+++ b/src/runtime/proc.go\n@@ -2,2 +2,2 @@ package runtime\n \n-func target() {}\n+func replacement() {}\n"

func TestRunMaintainerOutputPatchPublication(t *testing.T) {
	for _, command := range []string{"patch-materialize", "patch-regenerate"} {
		t.Run(command, func(t *testing.T) {
			for _, name := range []string{"patch", "git", "gofmt"} {
				if _, err := exec.LookPath(name); err != nil {
					t.Skipf("required fixture executable %s is unavailable: %v", name, err)
				}
			}
			root, source := t.TempDir(), t.TempDir()
			archive := maintainerArchive(t)
			maintainerWrite(t, root, "source.tar.gz", string(archive))
			descriptor := fmt.Sprintf(`{"schema_version":1,"go_version":"go1.26.4","archive":{"name":"go1.26.4.src.tar.gz","url":"https://go.dev/dl/go1.26.4.src.tar.gz","sha256":"%x"},"supported_platforms":["darwin/arm64"],"boundary_manifest_version":"go1.26.4-darwin-arm64-v1","patch":"toolchain/runtime/gomad.patch","adapters":[{"module":"modernc.org/libc","version":"v1.72.3","sum":"h1:test"}],"patch_allowlist":["src/runtime/proc.go"],"overlay_allowlist":["src/runtime/gomad.go"]}`, sha256.Sum256(archive))
			maintainerWrite(t, root, "toolchain/version/version.json", descriptor)
			maintainerWrite(t, root, "toolchain/runtime/gomad.patch", maintainerPatch)
			maintainerWrite(t, root, "toolchain/runtime/overlay/src/runtime/gomad.go", "package runtime\n")
			inputs := maintainerSnapshot(t, root)
			outputRoot := t.TempDir()
			for _, failed := range []bool{false, true} {
				contents := "package runtime\n\nfunc target() {}\n"
				if command == "patch-regenerate" {
					contents = "package runtime\n\nfunc replacement() {}\n"
				}
				maintainerWrite(t, source, "VERSION", "go1.26.4\n")
				maintainerWrite(t, source, "README", "fixture\n")
				maintainerWrite(t, source, "src/runtime/proc.go", contents)
				candidate := maintainerSnapshot(t, source)
				arguments := []string{command, "--root=" + root}
				want := "gomad3 patch materialized\n"
				if command == "patch-materialize" {
					arguments = append(arguments, "--source-root="+source)
				} else {
					gofmt, err := exec.LookPath("gofmt")
					if err != nil {
						t.Fatal(err)
					}
					arguments = append(arguments, "--candidate-root="+source, "--archive="+filepath.Join(root, "source.tar.gz"), "--output="+filepath.Join(outputRoot, "result.patch"), "--toolchain-root="+t.TempDir(), "--gofmt="+gofmt)
					want = "gomad3 patch regenerated\n"
				}
				var stdout, stderr bytes.Buffer
				status := 0
				if failed {
					output := newRefreshOutput(t, 0)
					status = run(arguments, output, &stderr)
					if !errors.Is(output.err, syscall.EBADF) {
						t.Fatalf("stdout write error = %v, want EBADF; status = %d; stderr = %s", output.err, status, &stderr)
					}
					t.Log("stdout write observed EBADF")
				} else {
					status = run(arguments, &stdout, &stderr)
					if stdout.String() != want {
						t.Fatalf("healthy stdout = %q, want %q; stderr = %s", &stdout, want, &stderr)
					}
				}
				if !maps.Equal(inputs, maintainerSnapshot(t, root)) {
					t.Fatal("patch inputs changed")
				}
				if command == "patch-materialize" {
					candidate["src/runtime/proc.go"] = "package runtime\n\nfunc replacement() {}\n"
				} else {
					contents, err := os.ReadFile(filepath.Join(outputRoot, "result.patch"))
					if err != nil {
						t.Fatal(err)
					}
					if string(contents) != maintainerRegeneratedPatch {
						t.Fatalf("regenerated patch = %q, want %q", contents, maintainerRegeneratedPatch)
					}
				}
				if !maps.Equal(candidate, maintainerSnapshot(t, source)) {
					t.Fatal("source publication differs from the literal healthy result")
				}
				wantStatus := 0
				if failed {
					wantStatus = 3
				}
				if status != wantStatus || stderr.Len() != 0 {
					t.Fatalf("status = %d, want %d; stderr = %q", status, wantStatus, &stderr)
				}
			}
		})
	}
}

func maintainerArchive(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	zipper := gzip.NewWriter(&output)
	archive := tar.NewWriter(zipper)
	for _, directory := range []string{"go/", "go/src/", "go/src/runtime/"} {
		if err := archive.WriteHeader(&tar.Header{Name: directory, Mode: 0o755, Typeflag: tar.TypeDir}); err != nil {
			t.Fatal(err)
		}
	}
	for _, entry := range []struct{ name, contents string }{
		{"go/VERSION", "go1.26.4\n"}, {"go/README", "fixture\n"}, {"go/src/runtime/proc.go", "package runtime\n\nfunc target() {}\n"},
	} {
		if err := archive.WriteHeader(&tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.contents)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write([]byte(entry.contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
