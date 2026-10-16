package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func TestRunBuildKey(t *testing.T) {
	root := t.TempDir()
	patch := filepath.Join(root, "gomad.patch")
	overlay := filepath.Join(root, "overlay")
	if err := os.MkdirAll(overlay, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(patch, []byte("patch"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overlay, "gomad.go"), []byte("package gomad"), 0o600); err != nil {
		t.Fatal(err)
	}
	arguments := []string{
		"build-key", "--go-version=go1.26.4", "--archive-sha256=archive", "--patch=" + patch,
		"--overlay=" + overlay, "--host-os=darwin", "--host-arch=arm64", "--bootstrap-version=bootstrap",
		"--recipe-version=recipe", "--build-path=/usr/bin:/bin", "--bash-path=/bin/bash", "--bash-version=5.2",
	}
	first := runHosttool(t, arguments)
	if len(strings.TrimSpace(first)) != 64 {
		t.Fatalf("build key = %q", first)
	}
	if repeated := runHosttool(t, arguments); repeated != first {
		t.Fatalf("repeated key = %q, want %q", repeated, first)
	}
	if err := os.WriteFile(filepath.Join(overlay, "gomad.go"), []byte("package changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed := runHosttool(t, arguments); changed == first {
		t.Fatal("build key did not change with overlay contents")
	}
}

func TestRunBuildKeyRejectsIncompleteInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run([]string{"build-key"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status = %d, stderr = %q", status, stderr.String())
	}
}

func TestRunCompatibilityPackCheckReportsOperationalFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{"compatibility-pack", "check", "--root=" + t.TempDir()}, &stdout, &stderr)
	if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "compatibility-pack requests") {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
}

func TestRunCompatibilityPackGenerateWithoutApprovalRegeneratesRecordedRequests(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := run([]string{"compatibility-pack", "generate", "--root=" + t.TempDir()}, &stdout, &stderr)
	if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "compatibility-pack requests") {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
}

func TestRunRejectsRemovedTestMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run([]string{"test-mode", "--mode=test", "--output=tiers"}, &stdout, &stderr); status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "usage: gomadtool") {
		t.Fatalf("run(test-mode) = %d, stdout %q, stderr %q", status, stdout.String(), stderr.String())
	}
}

func TestRunPatchValidateChecksCurrentModule(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if status := run([]string{"patch-validate", "--root=" + root}, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
}

func TestRunScriptValidateChecksCurrentModule(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if status := run([]string{"script-validate", "--root=" + root}, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
}

func TestRunPatchCommandsRejectIncompleteInput(t *testing.T) {
	for _, arguments := range [][]string{{"patch-validate"}, {"patch-materialize", "--root=/tmp"}, {"patch-regenerate", "--root=/tmp"}} {
		var stdout, stderr bytes.Buffer
		if status := run(arguments, &stdout, &stderr); status != 2 {
			t.Fatalf("run(%q) status = %d, stderr = %q", arguments, status, stderr.String())
		}
	}
}

func TestRunToolchainBuildRejectsIncompleteInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run([]string{"toolchain-build"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status = %d, stderr = %q", status, stderr.String())
	}
}

func TestRunTestRejectsIncompleteInput(t *testing.T) {
	for _, arguments := range [][]string{
		{"test", "--mode=test-upstream"},
		{"test", "--root=/tmp", "--mode=test-interception", "--go=/tmp/go"},
	} {
		var stdout, stderr bytes.Buffer
		if status := run(arguments, &stdout, &stderr); status != 2 {
			t.Fatalf("run(%q) status = %d, stderr = %q", arguments, status, stderr.String())
		}
	}
}

func TestBootstrapGofmtResolvesInstalledTool(t *testing.T) {
	gofmt, err := bootstrapGofmt("")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(gofmt) != "gofmt" {
		t.Fatalf("bootstrapGofmt() = %q", gofmt)
	}
}

func TestRunCheckedRunWritesCompatibilityResult(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	status := run([]string{
		"checked-run", "5", "7", "fixture seed=1 mode=unit iteration=0", root, "--",
		"/bin/sh", "-c", "printf output; printf error >&2; exit 7",
	}, &stdout, &stderr)
	if status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
	for name, want := range map[string]string{
		"stdout": "output", "stderr": "error", "status": "7\n", "timed-out": "0\n", "output-truncated": "0\n",
	} {
		contents, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(contents) != want {
			t.Fatalf("%s = %q, want %q", name, contents, want)
		}
	}
}

func TestRunCheckedRunDistinguishesExit124FromTimeout(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	status := run([]string{"checked-run", "5", "124", "false timeout", root, "--", "/bin/sh", "-c", "exit 124"}, &stdout, &stderr)
	if status != 1 || !strings.Contains(stderr.String(), "status 124 was not a timeout") {
		t.Fatalf("status = %d, stderr = %q", status, stderr.String())
	}
}

func runHosttool(t *testing.T, arguments []string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if status := run(arguments, &stdout, &stderr); status != 0 {
		t.Fatalf("run() status = %d, stderr = %q", status, stderr.String())
	}
	return stdout.String()
}

func TestRunMaintainerOutputTerminalReports(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
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
		{"patch-validate", []string{"patch-validate", "--root=" + root}, "gomad patch and overlay inputs are valid\n"},
		{"script-validate", []string{"script-validate", "--root=" + root}, "gomad script ownership is valid\n"},
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
	root, err := filepath.Abs("../../../../..")
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
		invalidStatus := 2
		if command == "compatibility-pack" {
			invalidStatus = 1
		}
		for _, test := range []struct {
			name      string
			arguments []string
			status    int
		}{
			{"invalid", []string{command, "--not-a-flag"}, invalidStatus},
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
			descriptor := fmt.Sprintf(`{"schema_version":1,"go_version":"go1.26.4","archive":{"name":"go1.26.4.src.tar.gz","url":"https://go.dev/dl/go1.26.4.src.tar.gz","sha256":"%x"},"supported_platforms":["darwin/arm64"],"boundary_manifest_version":"go1.26.4-darwin-arm64-v1","patch":"internal/backends/native/toolchain/runtime/gomad.patch","adapters":[{"module":"modernc.org/libc","version":"v1.72.3","sum":"h1:test"}],"patch_allowlist":["src/runtime/proc.go"],"overlay_allowlist":["src/runtime/gomad.go"]}`, sha256.Sum256(archive))
			maintainerWrite(t, root, "internal/backends/native/toolchain/version/version.json", descriptor)
			maintainerWrite(t, root, "internal/backends/native/toolchain/runtime/gomad.patch", maintainerPatch)
			maintainerWrite(t, root, "internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad.go", "package runtime\n")
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
				want := "gomad patch materialized\n"
				if command == "patch-materialize" {
					arguments = append(arguments, "--source-root="+source)
				} else {
					gofmt, err := exec.LookPath("gofmt")
					if err != nil {
						t.Fatal(err)
					}
					arguments = append(arguments, "--candidate-root="+source, "--archive="+filepath.Join(root, "source.tar.gz"), "--output="+filepath.Join(outputRoot, "result.patch"), "--toolchain-root="+t.TempDir(), "--gofmt="+gofmt)
					want = "gomad patch regenerated\n"
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

type generatorDiagnosticOutput struct {
	writer   io.Writer
	attempts int
	contents []byte
}

func (output *generatorDiagnosticOutput) Write(contents []byte) (int, error) {
	output.attempts++
	output.contents = append(output.contents, contents...)
	return output.writer.Write(contents)
}

func TestRunGeneratorDiagnosticOutputPreservesStatus(t *testing.T) {
	for _, test := range []struct {
		name, input, contents, output, diagnostic string
		arguments                                 []string
		status                                    int
	}{
		{
			name: "missing manifest flags", output: "qualification/manifest.json",
			arguments: []string{"qualification-manifest-generate"}, status: 2,
			diagnostic: "qualification-manifest-generate requires --spec and --output\n",
		},
		{
			name: "manifest failure", input: "qualification/spec.json", contents: "{}\n", output: "qualification/manifest.json",
			arguments: []string{"qualification-manifest-generate", "--spec=qualification/spec.json"}, status: 1,
			diagnostic: "qualification manifest spec schema must be gomad.qualification-set-generator/v1\n",
		},
		{
			name: "protocol failure", input: "internal/backends/native/transcript/schema/iowire.json", contents: "{} {}\n", output: "internal/model/iowire/wire_generated.go",
			arguments: []string{"protocol-generate"}, status: 1,
			diagnostic: "I/O wire schema has trailing data\n",
		},
		{
			name: "version failure", input: "internal/backends/native/toolchain/version/version.json", contents: "{}\n", output: "version_generated.mk",
			arguments: []string{"version-generate"}, status: 1,
			diagnostic: "version descriptor schema version 0 is unsupported\n",
		},
		{
			name: "boundary failure", output: "compiler-overlay/overlay.json",
			arguments: []string{"boundary-generate"}, status: 1,
			diagnostic: "-goroot is required with -compiler-test-overlay\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, existing := range []bool{false, true} {
				name := "absent output"
				if existing {
					name = "existing output"
				}
				t.Run(name, func(t *testing.T) {
					root := t.TempDir()
					maintainerWrite(t, root, "sentinels/keep", "unchanged publication sentinel\n")
					if test.input != "" {
						maintainerWrite(t, root, test.input, test.contents)
					}
					outputPath := filepath.Join(root, filepath.FromSlash(test.output))
					if existing {
						maintainerWrite(t, root, test.output, "existing generated output sentinel\n")
					} else if _, err := os.Lstat(outputPath); !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("absent output stat = %v", err)
					}
					arguments := append(append([]string(nil), test.arguments...), "--root="+root)
					switch test.name {
					case "missing manifest flags", "manifest failure":
						arguments = append(arguments, "--output="+test.output)
					case "boundary failure":
						arguments = append(arguments, "--compiler-test-overlay="+outputPath)
					}
					for _, failed := range []bool{false, true} {
						name := "healthy stderr"
						if failed {
							name = "EBADF stderr"
						}
						t.Run(name, func(t *testing.T) {
							before := generatorDiagnosticSnapshot(t, root)
							var stdout, stderr bytes.Buffer
							output := &generatorDiagnosticOutput{writer: &stdout}
							diagnostics := &generatorDiagnosticOutput{writer: &stderr}
							var failedOutput *refreshOutput
							if failed {
								failedOutput = newRefreshOutput(t, 0)
								diagnostics.writer = failedOutput
							}
							status := run(arguments, output, diagnostics)
							if status != test.status || diagnostics.attempts != 1 || string(diagnostics.contents) != test.diagnostic {
								t.Fatalf("status=%d attempts=%d diagnostic=%q, want %d, 1, %q", status, diagnostics.attempts, diagnostics.contents, test.status, test.diagnostic)
							}
							if output.attempts != 0 || len(output.contents) != 0 || stdout.Len() != 0 {
								t.Fatalf("stdout attempts=%d attempted=%q written=%q", output.attempts, output.contents, stdout.String())
							}
							if failed {
								if !errors.Is(failedOutput.err, syscall.EBADF) || failedOutput.Len() != 0 || stderr.Len() != 0 {
									t.Fatalf("failed diagnostic error=%v written=%q healthy=%q", failedOutput.err, failedOutput.String(), stderr.String())
								}
								t.Logf("stderr Write returned %T: %v; errors.Is(EBADF)=true; attempted=%q", failedOutput.err, failedOutput.err, diagnostics.contents)
							} else if stderr.String() != test.diagnostic {
								t.Fatalf("written diagnostic=%q, want %q", stderr.String(), test.diagnostic)
							}
							if after := generatorDiagnosticSnapshot(t, root); !reflect.DeepEqual(before, after) {
								t.Fatalf("publication changed: before=%#v after=%#v", before, after)
							}
							if !existing {
								if _, err := os.Lstat(outputPath); !errors.Is(err, os.ErrNotExist) {
									t.Fatalf("absent output changed: stat=%v", err)
								}
							}
						})
					}
				})
			}
		})
	}
}

type generatorDiagnosticEntry struct {
	mode     fs.FileMode
	contents string
	children []string
}

func generatorDiagnosticSnapshot(t *testing.T, root string) map[string]generatorDiagnosticEntry {
	t.Helper()
	snapshot := map[string]generatorDiagnosticEntry{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, visitErr error) error {
		if visitErr != nil {
			return visitErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		value := generatorDiagnosticEntry{mode: info.Mode()}
		if entry.IsDir() {
			children, err := os.ReadDir(path)
			if err != nil {
				return err
			}
			for _, child := range children {
				value.children = append(value.children, child.Name())
			}
		} else {
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value.contents = string(contents)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		snapshot[filepath.ToSlash(relative)] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

type terminalDiagnosticWriter struct {
	output   bytes.Buffer
	file     *os.File
	failures map[int]bool
	attempts []string
	errors   []error
}

func (writer *terminalDiagnosticWriter) Write(contents []byte) (int, error) {
	writer.attempts = append(writer.attempts, string(contents))
	if writer.failures[len(writer.attempts)] {
		written, err := writer.file.Write(contents)
		writer.errors = append(writer.errors, err)
		return written, err
	}
	writer.errors = append(writer.errors, nil)
	return writer.output.Write(contents)
}

func terminalDiagnostics(t *testing.T, failures map[int]bool) *terminalDiagnosticWriter {
	t.Helper()
	return &terminalDiagnosticWriter{file: newRefreshOutput(t, 0).file, failures: failures}
}

func checkTerminalDiagnostics(t *testing.T, writer *terminalDiagnosticWriter, want []string) {
	t.Helper()
	if !reflect.DeepEqual(writer.attempts, want) {
		t.Fatalf("attempts = %q, want %q", writer.attempts, want)
	}
	var delivered strings.Builder
	for index, contents := range want {
		if writer.failures[index+1] {
			if !errors.Is(writer.errors[index], syscall.EBADF) {
				t.Fatalf("attempt %d error = %v, want actual EBADF", index+1, writer.errors[index])
			}
		} else {
			if writer.errors[index] != nil {
				t.Fatal(writer.errors[index])
			}
			delivered.WriteString(contents)
		}
	}
	if writer.output.String() != delivered.String() {
		t.Fatalf("delivered = %q, want %q", writer.output.String(), delivered.String())
	}
	contents, err := os.ReadFile(writer.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 0 {
		t.Fatalf("read-only file received %q", contents)
	}
}

func TestTerminalDiagnosticsPrimaryStatus(t *testing.T) {
	root := t.TempDir()
	maintainerWrite(t, root, "patch", "patch")
	maintainerWrite(t, root, "overlay/gomad.go", "package gomad")
	maintainerWrite(t, root, "internal/backends/native/compatibilitypack/requests/invalid.json", "{} {}\n")
	left, right := filepath.Join(root, "left.bin"), filepath.Join(root, "right.bin")
	if err := os.WriteFile(left, toolDiagnosticBytes(0), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(right, toolDiagnosticBytes(1)[:159], 0o600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(root, "missing")
	pack := []string{"compatibility-pack", "--root=" + root}
	for _, test := range []struct {
		name       string
		arguments  []string
		status     int
		diagnostic string
	}{
		{"usage", nil, 2, usage + "\n"},
		{"unknown command", []string{"missing-command"}, 2, usage + "\n"},
		{"checked usage", []string{"checked-run"}, 125, "usage: gomadtool checked-run <seconds> <expected-status> <label> <result-dir> -- <command> [args...]\n"},
		{"checked invalid timeout", []string{"checked-run", "0", "0", "fixture", root, "--", "/bin/sh"}, 125, "gomad checked runner requires a positive timeout and numeric expected status\n"},
		{"checked invalid root", []string{"checked-run", "5", "0", "fixture", "/", "--", "/bin/sh"}, 125, "gomad checked runner result directory is invalid: <nil>\n"},
		{"build key invalid identity", []string{"build-key", "--patch=" + filepath.Join(root, "patch"), "--overlay=" + filepath.Join(root, "overlay")}, 2, "build buildKeyIdentity field 0 is empty or contains a newline\n"},
		{"build key missing patch", []string{"build-key", "--patch=" + missing, "--overlay=" + filepath.Join(root, "overlay")}, 1, "hash patch: open " + missing + ": no such file or directory\n"},
		{"build key missing overlay", []string{"build-key", "--patch=" + filepath.Join(root, "patch"), "--overlay=" + missing}, 1, "hash overlay: lstat " + missing + ": no such file or directory\n"},
		{"patch validation", []string{"patch-validate", "--root=/"}, 1, "patch set root must be an absolute non-root directory\n"},
		{"script validation", []string{"script-validate", "--root=/"}, 1, "gomad script-policy root must be an absolute non-root directory\n"},
		{"patch materialize", []string{"patch-materialize", "--root=/", "--source-root=" + root}, 1, "patch set root must be an absolute non-root directory\n"},
		{"patch regenerate", []string{"patch-regenerate", "--root=/", "--candidate-root=" + root, "--gofmt=/bin/false"}, 1, "patch set root must be an absolute non-root directory\n"},
		{"toolchain ordinary failure", []string{"toolchain-build", "--root=/", "--bootstrap-go=/bin/false", "--build-bash=/bin/bash"}, 1, "gomad module root must be an absolute non-root directory\n"},
		{"test patch failure", []string{"test", "--root=/", "--mode=test-builder", "--go=/bin/false"}, 1, "patch set root must be an absolute non-root directory\n"},
		{"diagnostic usage", []string{"diagnostic-diff"}, 2, "usage: gomadtool diagnostic-diff [--json] EXPECTED_TRACE ACTUAL_TRACE\n"},
		{"diagnostic missing expected", []string{"diagnostic-diff", missing, left}, 2, "open " + missing + ": no such file or directory\n"},
		{"diagnostic missing actual", []string{"diagnostic-diff", left, missing}, 2, "open " + missing + ": no such file or directory\n"},
		{"diagnostic invalid actual", []string{"diagnostic-diff", left, right}, 2, "diagnostic trace length does not match header\n"},
		{"pack discover invalid paths", []string{pack[0], "discover", pack[1], "--request=../outside", "--working-dir=" + root}, 2, "compatibility-pack path is outside the Gomad root\n"},
		{"pack discover missing request", []string{pack[0], "discover", pack[1], "--request=internal/backends/native/compatibilitypack/requests/missing.json", "--working-dir=" + root}, 2, "compatibility-pack input is not a bounded regular file: " + filepath.Join(root, "internal/backends/native/compatibilitypack/requests/missing.json") + "\n"},
		{"pack review invalid output", []string{pack[0], "review", pack[1], "--request=internal/backends/native/compatibilitypack/requests/invalid.json", "--output=outside.md"}, 2, "compatibility-pack review output must be below internal/backends/native/compatibilitypack\n"},
		{"pack review invalid request", []string{pack[0], "review", pack[1], "--request=internal/backends/native/compatibilitypack/requests/invalid.json", "--output=internal/backends/native/compatibilitypack/reports/invalid.md"}, 2, "decode compatibility-pack request: unexpected trailing JSON token {\n"},
		{"pack generate outside request", []string{pack[0], "generate", pack[1], "--request=outside.json", "--approve-review=approval"}, 2, "compatibility-pack request must be below internal/backends/native/compatibilitypack\n"},
		{"pack qualify outside request", []string{pack[0], "qualify", pack[1], "--request=outside.json", "--working-dir=" + root}, 2, "compatibility-pack request must be below internal/backends/native/compatibilitypack\n"},
		{"pack invalid override", []string{pack[0], "check", pack[1], "--compatibility-root=relative"}, 2, "--compatibility-root \"relative\" must be an absolute, clean path\n"},
		{"upgrade missing baseline root", []string{"upgrade-dossier", "--root=" + missing, "--baseline-ref=HEAD"}, 1, "locate Gomad Git prefix: exit status 128\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				name := "healthy stderr"
				if failed {
					name = "EBADF stderr"
				}
				t.Run(name, func(t *testing.T) {
					before := generatorDiagnosticSnapshot(t, root)
					var stdout bytes.Buffer
					output := &generatorDiagnosticOutput{writer: &stdout}
					diagnostics := terminalDiagnostics(t, map[int]bool{1: failed})
					if status := run(test.arguments, output, diagnostics); status != test.status {
						t.Fatalf("status = %d, want %d; attempts = %q", status, test.status, diagnostics.attempts)
					}
					checkTerminalDiagnostics(t, diagnostics, []string{test.diagnostic})
					if output.attempts != 0 || stdout.Len() != 0 {
						t.Fatalf("stdout attempts = %d, delivered = %q", output.attempts, stdout.String())
					}
					if !reflect.DeepEqual(before, generatorDiagnosticSnapshot(t, root)) {
						t.Fatal("primary failure changed publication")
					}
				})
			}
		})
	}
}

func TestTerminalDiagnosticsPackQualificationClassification(t *testing.T) {
	for _, test := range []struct {
		name, input, contents, diagnostic string
		status                            int
	}{
		{"missing requests", "", "", "", 3},
		{"missing table", "", "", "compatibility-pack working-directory table working-directories.json is not a bounded regular file\n", 2},
		{"invalid request entry", "requests/.keep", "", "compatibility-pack request entry is invalid: .keep\n", 3},
		{"invalid table", "working-directories.json", "{}\n", "compatibility-pack working-directory table schema is unsupported\n", 2},
		{"empty table", "working-directories.json", "{\"schema\":\"gomad.compatibility-pack-working-directories/v1\",\"requests\":[]}\n", "no compatibility-pack request names " + runtime.GOOS + "/" + runtime.GOARCH + "\n", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if test.name != "missing requests" {
				if err := os.Mkdir(filepath.Join(root, "requests"), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if test.input != "" {
				maintainerWrite(t, root, test.input, test.contents)
			}
			want := test.diagnostic
			if test.name == "missing requests" {
				want = "read compatibility-pack requests: open " + filepath.Join(root, "requests") + ": no such file or directory\n"
			}
			for _, failed := range []bool{false, true} {
				before := generatorDiagnosticSnapshot(t, root)
				diagnostics := terminalDiagnostics(t, map[int]bool{1: failed})
				output := &generatorDiagnosticOutput{writer: io.Discard}
				status := run([]string{"compatibility-pack", "qualify", "--root=" + root, "--compatibility-root=" + root, "--all"}, output, diagnostics)
				if status != test.status || output.attempts != 0 {
					t.Fatalf("status = %d, want %d; stdout attempts = %d", status, test.status, output.attempts)
				}
				checkTerminalDiagnostics(t, diagnostics, []string{want})
				if !reflect.DeepEqual(before, generatorDiagnosticSnapshot(t, root)) {
					t.Fatal("qualification failure changed inputs")
				}
			}
		})
	}
}

func TestTerminalDiagnosticsCheckedRunSequence(t *testing.T) {
	for _, test := range []struct {
		name, command string
		status        string
		want          []string
	}{
		{"both payloads", "printf output; printf error >&2; exit 7", "7\n", []string{"gomad process failed: fixture: status 7, want 0\n", "--- stdout ---\noutput", "--- stderr ---\nerror"}},
		{"stdout only", "printf output; exit 7", "7\n", []string{"gomad process failed: fixture: status 7, want 0\n", "--- stdout ---\noutput"}},
		{"stderr only", "printf error >&2; exit 7", "7\n", []string{"gomad process failed: fixture: status 7, want 0\n", "--- stderr ---\nerror"}},
		{"no payload", "exit 7", "7\n", []string{"gomad process failed: fixture: status 7, want 0\n"}},
		{"false timeout", "exit 124", "124\n", []string{"gomad process failed: fixture: status 124 was not a timeout\n"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, failures := range []map[int]bool{nil, {1: true}, {1: true, 2: true, 3: true}, {2: true}} {
				root := t.TempDir()
				var stdout bytes.Buffer
				output := &generatorDiagnosticOutput{writer: &stdout}
				diagnostics := terminalDiagnostics(t, failures)
				expected := "0"
				if test.name == "false timeout" {
					expected = "124"
				}
				status := run([]string{"checked-run", "5", expected, "fixture", root, "--", "/bin/sh", "-c", test.command}, output, diagnostics)
				if status != 1 || output.attempts != 0 {
					t.Fatalf("status = %d, stdout attempts = %d", status, output.attempts)
				}
				checkTerminalDiagnostics(t, diagnostics, test.want)
				wantStdout, wantStderr := "", ""
				if strings.Contains(test.command, "printf output") {
					wantStdout = "output"
				}
				if strings.Contains(test.command, "printf error") {
					wantStderr = "error"
				}
				for name, want := range map[string]string{"stdout": wantStdout, "stderr": wantStderr, "status": test.status, "timed-out": "0\n", "output-truncated": "0\n"} {
					contents, err := os.ReadFile(filepath.Join(root, name))
					if err != nil {
						t.Fatal(err)
					}
					if string(contents) != want {
						t.Fatalf("completed %s = %q, want %q", name, contents, want)
					}
				}
			}
		})
	}
}

func TestTerminalDiagnosticsDiffSummary(t *testing.T) {
	root := t.TempDir()
	left, right := filepath.Join(root, "left.bin"), filepath.Join(root, "right.bin")
	for _, different := range []bool{false, true} {
		draw, wantStatus, want := byte(0), 0, "diagnostic traces match\n"
		if different {
			draw, wantStatus, want = 1, 1, "first-divergent-ordinal=0 fields=runtime_cheap_rand_draws\n"
		}
		for _, value := range []struct {
			path string
			draw byte
		}{{left, 0}, {right, draw}} {
			if err := os.WriteFile(value.path, toolDiagnosticBytes(value.draw), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		for _, failedStdout := range []bool{false, true} {
			for _, failedStderr := range []bool{false, true} {
				output := terminalDiagnostics(t, map[int]bool{1: failedStdout})
				diagnostics := terminalDiagnostics(t, map[int]bool{1: failedStderr})
				status := run([]string{"diagnostic-diff", left, right}, output, diagnostics)
				checkTerminalDiagnostics(t, output, []string{want})
				if failedStdout {
					if status != 3 {
						t.Fatalf("failed summary status = %d, want 3", status)
					}
					checkTerminalDiagnostics(t, diagnostics, []string{fmt.Sprintf("write %s: %s\n", output.file.Name(), syscall.EBADF)})
				} else if status != wantStatus || len(diagnostics.attempts) != 0 {
					t.Fatalf("healthy summary status = %d, stderr attempts = %q", status, diagnostics.attempts)
				}
			}
		}
	}
}

func TestTerminalDiagnosticsConformanceSequence(t *testing.T) {
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	maintainerWrite(t, root, "internal/backends/native/toolchain/version/version.json", `{"schema_version":1,"go_version":"go1.27.1","archive":{"name":"go1.27.1.src.tar.gz","url":"https://go.dev/dl/go1.27.1.src.tar.gz","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"supported_platforms":["darwin/arm64","linux/amd64"],"boundary_manifest_version":"go1.27.1-v1","patch":"internal/backends/native/toolchain/runtime/gomad.patch","adapters":[{"module":"modernc.org/libc","version":"v1.72.3","sum":"h1:test"}],"patch_allowlist":["src/runtime/proc.go"],"overlay_allowlist":["src/runtime/gomad.go"]}`)
	maintainerWrite(t, root, "internal/backends/native/toolchain/runtime/gomad.patch", maintainerPatch)
	maintainerWrite(t, root, "internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad.go", "package runtime\n")
	maintainerWrite(t, root, "go.mod", "module gomad.terminal-fixture\n\ngo 1.27.1\n")
	maintainerWrite(t, root, "toolchain/failure_test.go", "package toolchain\nvar = 1\n")
	const childStdout = "FAIL\tgomad.terminal-fixture/toolchain [setup failed]\nFAIL\n"
	const childStderr = "# gomad.terminal-fixture/toolchain\ntoolchain/failure_test.go:2:5: expected 'IDENT', found '='\n"
	want := []string{"gomad fixture toolchain-builder-unit failed with status 1: " + childStdout + childStderr + "\n", "--- stdout ---\n" + childStdout, "--- stderr ---\n" + childStderr}
	for _, failures := range []map[int]bool{nil, {1: true}, {1: true, 2: true, 3: true}, {2: true}} {
		before := generatorDiagnosticSnapshot(t, root)
		diagnostics := terminalDiagnostics(t, failures)
		output := &generatorDiagnosticOutput{writer: io.Discard}
		status := run([]string{"test", "--root=" + root, "--mode=test-builder", "--go=" + goCommand}, output, diagnostics)
		if status != 1 || output.attempts != 0 {
			t.Fatalf("status = %d, stdout attempts = %d", status, output.attempts)
		}
		checkTerminalDiagnostics(t, diagnostics, want)
		if !reflect.DeepEqual(before, generatorDiagnosticSnapshot(t, root)) {
			t.Fatal("failed fixture changed patch/module inputs")
		}
	}
}
