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
		_, _, err := runCommand(ctx, directory, []string{os.Args[0], "-test.run=^TestCommandCancellationStopsDescendant$"}, append(os.Environ(), "GOMAD_RUNNER_CANCEL_HELPER=parent", "GOMAD_RUNNER_CANCEL_MARKER="+marker), filepath.Join(directory, "command.log"))
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

func TestNativeAdmission(t *testing.T) {
	for _, tc := range []struct {
		os, arch, help string
		ok             bool
	}{
		{"linux", "amd64", "  -gomadguard\n    guard capability boundary\n", true},
		{"darwin", "arm64", "  -gomadguard\n", true},
		{"linux", "arm64", "  -gomadguard\n", false},
		{"linux", "amd64", "stock compiler mentions -gomadguard in prose\n", false},
		{"darwin", "arm64", "", false},
	} {
		t.Run(tc.os+"/"+tc.arch+"/"+tc.help, func(t *testing.T) {
			if err := admitNative(tc.os, tc.arch, tc.help); (err == nil) != tc.ok {
				t.Fatalf("admission error=%v, accepted=%v", err, tc.ok)
			}
		})
	}
}

func TestEnvironmentKeepsArgumentsOutOfShellAndRemovesActivation(t *testing.T) {
	env := probeEnvironment([]string{"PATH=/a path", "TMPDIR=/tmp/private", "GOMADSEED=77", "GOMAD3_IO_PROFILE=foreign", "GOMAD3_CHILD_SEED=9", "GOMAD3_CLOCK_TICK=forward", "GOOS=foreign", "GOARCH=foreign", "GOFLAGS=-race", "GOEXPERIMENT=greenteagc", "CGO_ENABLED=1"})
	values := make(map[string]string)
	for _, value := range env {
		key, val, _ := strings.Cut(value, "=")
		if _, found := values[key]; found {
			t.Fatalf("duplicate key %s", key)
		}
		values[key] = val
	}
	for key := range values {
		if strings.HasPrefix(key, "GOMAD") || key == "GOOS" || key == "GOARCH" {
			t.Fatalf("ambient activation or cross target retained: %s", key)
		}
	}
	for key, want := range map[string]string{"PATH": "/a path", "TMPDIR": "/tmp/private", "GOEXPERIMENT": "nogreenteagc", "CGO_ENABLED": "0", "GOFLAGS": "", "GOENV": "off", "GOWORK": "off", "GOTOOLCHAIN": "local", "TZ": "UTC"} {
		if values[key] != want {
			t.Fatalf("%s=%q, want %q", key, values[key], want)
		}
	}
}

func TestChildResultRejectsEmptyWrongAndSkippedSelection(t *testing.T) {
	name := "TestGomadNativeTCP"
	for _, tc := range []struct {
		output string
		code   int
		ok     bool
	}{
		{"=== RUN   TestGomadNativeTCP\n--- PASS: TestGomadNativeTCP (0.00s)\nPASS\n", 0, true},
		{"", 0, false},
		{"=== RUN   TestOther\n--- PASS: TestOther (0.00s)\nPASS\n", 0, false},
		{"=== RUN   TestGomadNativeTCP\n--- SKIP: TestGomadNativeTCP (0.00s)\nPASS\n", 0, false},
		{"=== RUN   TestGomadNativeTCP\n--- PASS: TestGomadNativeTCP (0.00s)\nPASS\n", 1, false},
	} {
		if err := checkChild([]byte(tc.output), name, tc.code); (err == nil) != tc.ok {
			t.Fatalf("code=%d output=%q: error=%v accepted=%v", tc.code, tc.output, err, tc.ok)
		}
	}
}

func TestRefuseInstrumentationPreservesReportingBody(t *testing.T) {
	original, err := os.ReadFile(filepath.Join("..", "..", "overlay", "src", "internal", "gomadio", "descriptor_backend.go"))
	if err != nil {
		t.Fatal(err)
	}
	instrumented, err := instrumentRefuse(original)
	if err != nil {
		t.Fatal(err)
	}
	call := []byte("\tgomadNativeObserveRefuse(operation)\n")
	if bytes.Count(instrumented, call) != 1 {
		t.Fatal("actual reporting seam was not observed exactly once")
	}
	if !bytes.Equal(bytes.Replace(instrumented, call, nil, 1), original) {
		t.Fatal("removing reporting observer did not recover actual backend")
	}
	for _, changed := range [][]byte{
		bytes.Replace(original, []byte("net.syscall.refused"), []byte("wrong.refusal.operation"), 1),
		append(append([]byte{}, original...), original...),
	} {
		if _, err := instrumentRefuse(changed); err == nil {
			t.Fatal("changed or duplicated reporting source was accepted")
		}
	}
}

func TestOverlayBindsSourcesAndCreatesOnlyPrivateReplacements(t *testing.T) {
	repo, goroot, output := nativeInputTree(t)
	prepared, err := prepareOverlay(repo, goroot, output)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(prepared.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct{ Replace map[string]string }
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Replace) != 6 {
		t.Fatalf("replacements=%d, want 5 synthetic files and actual backend instrumentation", len(manifest.Replace))
	}
	for target, replacement := range manifest.Replace {
		if strings.HasSuffix(target, "/descriptor_backend.go") {
			original, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			contents, err := os.ReadFile(replacement)
			if err != nil || !bytes.Equal(bytes.Replace(contents, []byte("\tgomadNativeObserveRefuse(operation)\n"), nil, 1), original) {
				t.Fatalf("backend instrumentation changed actual body: %v", err)
			}
		} else if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Fatalf("original synthetic file exists: %s: %v", target, err)
		}
		if filepath.Dir(replacement) != output {
			t.Fatalf("replacement escaped private output: %s", replacement)
		}
		info, err := os.Stat(replacement)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("replacement permissions=%v", info.Mode())
		}
	}
	if len(prepared.Hashes) < 10 {
		t.Fatalf("missing template or source identities: %v", prepared.Hashes)
	}
	if _, err := prepareOverlay(repo, goroot, output); err == nil {
		t.Fatal("reused output accepted")
	}
}

func TestOverlayRejectsMismatchSymlinkAndSourceOutput(t *testing.T) {
	for _, kind := range []string{"mismatch", "symlink", "source-output"} {
		t.Run(kind, func(t *testing.T) {
			repo, goroot, output := nativeInputTree(t)
			path := filepath.Join(goroot, "src/runtime/gomad_vfd.go")
			switch kind {
			case "mismatch":
				if err := os.WriteFile(path, []byte("foreign runtime"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(repo, "toolchain/runtime/overlay/src/runtime/gomad_vfd.go"), path); err != nil {
					t.Fatal(err)
				}
			case "source-output":
				output = filepath.Join(goroot, "src", "private-output")
			}
			if _, err := prepareOverlay(repo, goroot, output); err == nil {
				t.Fatal("unsafe overlay input accepted")
			}
		})
	}
}

func nativeInputTree(t *testing.T) (string, string, string) {
	t.Helper()
	base := t.TempDir()
	repo, goroot, output := filepath.Join(base, "repo"), filepath.Join(base, "goroot"), filepath.Join(base, "evidence")
	put := func(path, value string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{"runtime/gomad_vfd.go", "runtime/gomad.go", "syscall/gomad_vfd_unix.go", "syscall/gomad_vfd_linux_amd64.go", "syscall/gomad_vfd_darwin_arm64.go", "internal/gomadio/descriptor_backend.go", "internal/gomadio/network.go", "internal/gomadio/gomadio.go", "internal/gomadvfd/descriptor.go", "net/gomad.go"} {
		contents := "frozen " + path
		if path == "internal/gomadio/descriptor_backend.go" {
			actual, err := os.ReadFile(filepath.Join("..", "..", "overlay", "src", path))
			if err != nil {
				t.Fatal(err)
			}
			contents = string(actual)
		}
		put(filepath.Join(repo, "toolchain/runtime/overlay/src", path), contents)
		put(filepath.Join(goroot, "src", path), contents)
	}
	for _, file := range []string{"fixture_test.go.txt", "runtime_probe.go.txt", "linux_probe.go.txt", "darwin_probe.go.txt", "gomadio_probe.go.txt"} {
		put(filepath.Join(repo, "toolchain/runtime/testdata/vfdnative", file), "template "+file)
	}
	return repo, goroot, output
}
