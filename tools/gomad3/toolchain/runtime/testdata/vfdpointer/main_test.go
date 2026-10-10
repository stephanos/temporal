package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCommandCancellationStopsDescendant(t *testing.T) {
	mode := os.Getenv("GOMAD_RUNNER_CANCEL_HELPER")
	if mode == "parent" {
		child := exec.Command(os.Args[0], "-test.run=^TestCommandCancellationStopsDescendant$")
		child.Env = append(os.Environ(), "GOMAD_RUNNER_CANCEL_HELPER=descendant")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		if err := child.Run(); err != nil {
			os.Exit(3)
		}
		os.Exit(0)
	}
	if mode == "descendant" {
		file, err := os.OpenFile(os.Getenv("GOMAD_RUNNER_CANCEL_MARKER"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			os.Exit(4)
		}
		if _, err := fmt.Fprintf(file, "%d\n", os.Getpid()); err != nil {
			os.Exit(5)
		}
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := file.WriteString("live\n"); err != nil {
				os.Exit(6)
			}
		}
		os.Exit(0)
	}
	directory := t.TempDir()
	marker := filepath.Join(directory, "descendant")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := runCommand(ctx, []string{os.Args[0], "-test.run=^TestCommandCancellationStopsDescendant$"}, append(os.Environ(), "GOMAD_RUNNER_CANCEL_HELPER=parent", "GOMAD_RUNNER_CANCEL_MARKER="+marker), filepath.Join(directory, "command.log"))
		done <- err
	}()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	var observed []byte
	for len(observed) == 0 {
		select {
		case <-deadline.C:
			t.Fatal("owned descendant did not start")
		case <-ticker.C:
			contents, err := os.ReadFile(marker)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
			if bytes.Contains(contents, []byte("live\n")) {
				observed = contents
			}
		}
	}
	pidText, _, _ := strings.Cut(string(observed), "\n")
	pid, err := strconv.Atoi(pidText)
	if err != nil || pid <= 0 {
		t.Fatalf("invalid observed descendant identity %q", pidText)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, syscall.ESRCH) {
			t.Errorf("owned descendant cleanup: %v", err)
		}
		if err := process.Release(); err != nil {
			t.Errorf("owned descendant handle: %v", err)
		}
	})
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation identity lost: %v", err)
		}
	case <-deadline.C:
		t.Fatal("cancelled command did not become terminal")
	}
	before, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	<-timer.C
	after, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("owned descendant kept writing after cancelled command returned")
	}
}

func TestInstrumentationPreservesHelper(t *testing.T) {
	original, err := os.ReadFile(filepath.Join("..", "..", "overlay", "src", "syscall", "gomad_vfd_unix.go"))
	if err != nil {
		t.Fatal(err)
	}
	instrumented, err := instrument(original)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(original, instrumented) {
		t.Fatal("helper was not instrumented")
	}
	for _, insertion := range insertions {
		if bytes.Count(instrumented, []byte(insertion.anchor+insertion.call)) != 1 {
			t.Fatalf("growth call %q is not unique", insertion.call)
		}
		instrumented = bytes.Replace(instrumented, []byte(insertion.call), nil, 1)
	}
	if !bytes.Equal(original, instrumented) {
		t.Fatal("removing growth calls did not recover original helper")
	}
	for _, source := range [][]byte{
		bytes.Replace(original, []byte("flags int) (int, error) {"), []byte("flags uint) (int, error) {"), 1),
		append(append([]byte{}, original...), original...),
	} {
		if _, err := instrument(source); err == nil {
			t.Fatal("accepted changed or duplicate helper signature")
		}
	}
}

func TestOverlayAgreementAndManifest(t *testing.T) {
	root, goroot, output := t.TempDir(), t.TempDir(), t.TempDir()
	source, err := os.ReadFile(filepath.Join("..", "..", "overlay", "src", "syscall", "gomad_vfd_unix.go"))
	if err != nil {
		t.Fatal(err)
	}
	repository := filepath.Join(root, "toolchain", "runtime", "overlay", "src", "syscall", "gomad_vfd_unix.go")
	selected := filepath.Join(goroot, "src", "syscall", "gomad_vfd_unix.go")
	writeTestFile(t, repository, source)
	writeTestFile(t, selected, source)
	for _, name := range []string{"growth.go.txt", "fixture_test.go.txt", "linux_export_test.go.txt", "darwin_export_test.go.txt"} {
		writeTestFile(t, filepath.Join(root, "toolchain", "runtime", "testdata", "vfdpointer", name), []byte("package syscall\n"))
	}
	prepared, err := prepareOverlay(root, goroot, output)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(prepared.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct{ Replace map[string]string }
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Replace) != 5 {
		t.Fatalf("overlay entries = %d, want 5", len(manifest.Replace))
	}
	for target, replacement := range manifest.Replace {
		if !filepath.IsAbs(target) || filepath.Dir(target) != filepath.Dir(selected) || !filepath.IsAbs(replacement) {
			t.Fatalf("non-private overlay mapping %q -> %q", target, replacement)
		}
		if _, err := os.ReadFile(replacement); err != nil {
			t.Fatal(err)
		}
	}
	unchanged, err := os.ReadFile(selected)
	if err != nil || !bytes.Equal(source, unchanged) {
		t.Fatalf("selected helper changed: %v", err)
	}
	if _, err := prepareOverlay(root, goroot, filepath.Dir(selected)); err == nil {
		t.Fatal("output inside toolchain sources was accepted")
	}
	if _, err := prepareOverlay(root, goroot, output); err == nil {
		t.Fatal("existing private overlay was overwritten")
	}
	writeTestFile(t, selected, append(source, '\n'))
	if _, err := prepareOverlay(root, goroot, t.TempDir()); err == nil || !strings.Contains(err.Error(), "differs") {
		t.Fatalf("source mismatch was not refused: %v", err)
	}
	writeTestFile(t, selected, source)
	if err := os.Remove(filepath.Join(root, "toolchain", "runtime", "testdata", "vfdpointer", "growth.go.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareOverlay(root, goroot, t.TempDir()); err == nil {
		t.Fatal("missing template was accepted")
	}
	if _, err := prepareOverlay(root, goroot, selected); err == nil {
		t.Fatal("non-directory output was accepted")
	}
}

func TestNativeAdmission(t *testing.T) {
	for _, test := range []struct {
		os, arch, help string
		admitted       bool
	}{
		{"linux", "amd64", "  -gomadguard\n", true},
		{"darwin", "arm64", "  -gomadguard\n", true},
		{"linux", "arm64", "  -gomadguard\n", false},
		{"linux", "amd64", "stock compiler", false},
	} {
		if err := admitNative(test.os, test.arch, test.help); (err == nil) != test.admitted {
			t.Fatalf("%s/%s admission=%v, want %v", test.os, test.arch, err, test.admitted)
		}
	}
}

func TestCommandFailureRetainsOutput(t *testing.T) {
	if os.Getenv("GOMAD_POINTER_RUNNER_CHILD") == "1" {
		if os.Args[len(os.Args)-1] != "two words" {
			os.Exit(8)
		}
		if _, err := os.Stdout.WriteString("intentional child failure\n"); err != nil {
			os.Exit(9)
		}
		os.Exit(7)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log := filepath.Join(t.TempDir(), "child.log")
	result, err := runCommand(ctx, []string{os.Args[0], "-test.run=^TestCommandFailureRetainsOutput$", "--", "two words"}, append(os.Environ(), "GOMAD_POINTER_RUNNER_CHILD=1"), log)
	if err == nil || result.ExitCode != 7 {
		t.Fatalf("child status=%d, error=%v", result.ExitCode, err)
	}
	contents, err := os.ReadFile(log)
	if err != nil || string(contents) != "intentional child failure\n" {
		t.Fatalf("failure log=%q, error=%v", contents, err)
	}
}

func TestFixtureResultRequiresSelectedTest(t *testing.T) {
	for _, test := range []struct {
		status int
		output string
		valid  bool
	}{
		{0, "=== RUN   TestGomadVFDPointerFixture\n--- PASS: TestGomadVFDPointerFixture (0.00s)\nPASS\n", true},
		{0, "PASS\n", false},
		{0, "=== RUN   AnotherTest\n--- PASS: AnotherTest (0.00s)\nPASS\n", false},
		{0, "=== RUN   TestGomadVFDPointerFixture\n--- SKIP: TestGomadVFDPointerFixture (0.00s)\nPASS\n", false},
		{1, "=== RUN   TestGomadVFDPointerFixture\n--- PASS: TestGomadVFDPointerFixture (0.00s)\n", false},
	} {
		if err := verifyFixture(test.status, []byte(test.output)); (err == nil) != test.valid {
			t.Fatalf("status=%d output=%q accepted=%v error=%v", test.status, test.output, test.valid, err)
		}
	}
}

func writeTestFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
}
