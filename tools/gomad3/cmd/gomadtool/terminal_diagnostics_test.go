package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

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
	maintainerWrite(t, root, "internal/compatibilitypack/requests/invalid.json", "{} {}\n")
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
		{"checked invalid timeout", []string{"checked-run", "0", "0", "fixture", root, "--", "/bin/sh"}, 125, "gomad3 checked runner requires a positive timeout and numeric expected status\n"},
		{"checked invalid root", []string{"checked-run", "5", "0", "fixture", "/", "--", "/bin/sh"}, 125, "gomad3 checked runner result directory is invalid: <nil>\n"},
		{"build key invalid identity", []string{"build-key", "--patch=" + filepath.Join(root, "patch"), "--overlay=" + filepath.Join(root, "overlay")}, 2, "build buildKeyIdentity field 0 is empty or contains a newline\n"},
		{"build key missing patch", []string{"build-key", "--patch=" + missing, "--overlay=" + filepath.Join(root, "overlay")}, 1, "hash patch: open " + missing + ": no such file or directory\n"},
		{"build key missing overlay", []string{"build-key", "--patch=" + filepath.Join(root, "patch"), "--overlay=" + missing}, 1, "hash overlay: lstat " + missing + ": no such file or directory\n"},
		{"patch validation", []string{"patch-validate", "--root=/"}, 1, "patch set root must be an absolute non-root directory\n"},
		{"script validation", []string{"script-validate", "--root=/"}, 1, "gomad3 script-policy root must be an absolute non-root directory\n"},
		{"patch materialize", []string{"patch-materialize", "--root=/", "--source-root=" + root}, 1, "patch set root must be an absolute non-root directory\n"},
		{"patch regenerate", []string{"patch-regenerate", "--root=/", "--candidate-root=" + root, "--gofmt=/bin/false"}, 1, "patch set root must be an absolute non-root directory\n"},
		{"toolchain ordinary failure", []string{"toolchain-build", "--root=/", "--bootstrap-go=/bin/false", "--build-bash=/bin/bash"}, 1, "gomad3 module root must be an absolute non-root directory\n"},
		{"test patch failure", []string{"test", "--root=/", "--mode=test-builder", "--go=/bin/false"}, 1, "patch set root must be an absolute non-root directory\n"},
		{"diagnostic usage", []string{"diagnostic-diff"}, 2, "usage: gomadtool diagnostic-diff [--json] EXPECTED_TRACE ACTUAL_TRACE\n"},
		{"diagnostic missing expected", []string{"diagnostic-diff", missing, left}, 2, "open " + missing + ": no such file or directory\n"},
		{"diagnostic missing actual", []string{"diagnostic-diff", left, missing}, 2, "open " + missing + ": no such file or directory\n"},
		{"diagnostic invalid actual", []string{"diagnostic-diff", left, right}, 2, "diagnostic trace length does not match header\n"},
		{"pack discover invalid paths", []string{pack[0], "discover", pack[1], "--request=../outside", "--working-dir=" + root}, 2, "compatibility-pack path is outside the Gomad v3 root\n"},
		{"pack discover missing request", []string{pack[0], "discover", pack[1], "--request=internal/compatibilitypack/requests/missing.json", "--working-dir=" + root}, 2, "compatibility-pack input is not a bounded regular file: " + filepath.Join(root, "internal/compatibilitypack/requests/missing.json") + "\n"},
		{"pack review invalid output", []string{pack[0], "review", pack[1], "--request=internal/compatibilitypack/requests/invalid.json", "--output=outside.md"}, 2, "compatibility-pack review output must be below internal/compatibilitypack\n"},
		{"pack review invalid request", []string{pack[0], "review", pack[1], "--request=internal/compatibilitypack/requests/invalid.json", "--output=internal/compatibilitypack/reports/invalid.md"}, 2, "decode compatibility-pack request: unexpected trailing JSON token {\n"},
		{"pack generate outside request", []string{pack[0], "generate", pack[1], "--request=outside.json", "--approve-review=approval"}, 2, "compatibility-pack request must be below internal/compatibilitypack\n"},
		{"pack qualify outside request", []string{pack[0], "qualify", pack[1], "--request=outside.json", "--working-dir=" + root}, 2, "compatibility-pack request must be below internal/compatibilitypack\n"},
		{"pack invalid override", []string{pack[0], "check", pack[1], "--compatibility-root=relative"}, 2, "--compatibility-root \"relative\" must be an absolute, clean path\n"},
		{"upgrade missing baseline root", []string{"upgrade-dossier", "--root=" + missing, "--baseline-ref=HEAD"}, 1, "locate Gomad v3 Git prefix: exit status 128\n"},
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
		{"empty table", "working-directories.json", "{\"schema\":\"gomad3.compatibility-pack-working-directories/v1\",\"requests\":[]}\n", "no compatibility-pack request names " + runtime.GOOS + "/" + runtime.GOARCH + "\n", 1},
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
		{"both payloads", "printf output; printf error >&2; exit 7", "7\n", []string{"gomad3 process failed: fixture: status 7, want 0\n", "--- stdout ---\noutput", "--- stderr ---\nerror"}},
		{"stdout only", "printf output; exit 7", "7\n", []string{"gomad3 process failed: fixture: status 7, want 0\n", "--- stdout ---\noutput"}},
		{"stderr only", "printf error >&2; exit 7", "7\n", []string{"gomad3 process failed: fixture: status 7, want 0\n", "--- stderr ---\nerror"}},
		{"no payload", "exit 7", "7\n", []string{"gomad3 process failed: fixture: status 7, want 0\n"}},
		{"false timeout", "exit 124", "124\n", []string{"gomad3 process failed: fixture: status 124 was not a timeout\n"}},
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
	maintainerWrite(t, root, "toolchain/version/version.json", `{"schema_version":1,"go_version":"go1.27.1","archive":{"name":"go1.27.1.src.tar.gz","url":"https://go.dev/dl/go1.27.1.src.tar.gz","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"supported_platforms":["darwin/arm64","linux/amd64"],"boundary_manifest_version":"go1.27.1-v1","patch":"toolchain/runtime/gomad.patch","adapters":[{"module":"modernc.org/libc","version":"v1.72.3","sum":"h1:test"}],"patch_allowlist":["src/runtime/proc.go"],"overlay_allowlist":["src/runtime/gomad.go"]}`)
	maintainerWrite(t, root, "toolchain/runtime/gomad.patch", maintainerPatch)
	maintainerWrite(t, root, "toolchain/runtime/overlay/src/runtime/gomad.go", "package runtime\n")
	maintainerWrite(t, root, "go.mod", "module gomad3.terminal-fixture\n\ngo 1.27.1\n")
	maintainerWrite(t, root, "toolchain/failure_test.go", "package toolchain\nvar = 1\n")
	const childStdout = "FAIL\tgomad3.terminal-fixture/toolchain [setup failed]\nFAIL\n"
	const childStderr = "# gomad3.terminal-fixture/toolchain\ntoolchain/failure_test.go:2:5: expected 'IDENT', found '='\n"
	want := []string{"gomad3 fixture toolchain-builder-unit failed with status 1: " + childStdout + childStderr + "\n", "--- stdout ---\n" + childStdout, "--- stderr ---\n" + childStderr}
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
