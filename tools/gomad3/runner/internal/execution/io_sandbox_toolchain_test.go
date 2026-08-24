package execution_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestProfilePassesHostCapabilitySandbox(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	tests := []struct {
		name             string
		source           string
		workingDirectory string
		wantOutput       string
		hostEscape       bool
		denyWrites       bool
		libcAdapter      bool
		guarded          bool
	}{
		{name: "filesystem", source: "./io_filesystem", workingDirectory: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"), wantOutput: "isolated\n", hostEscape: true, denyWrites: true},
		{name: "network", source: "./io_net", workingDirectory: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"), wantOutput: "ok\n"},
		{name: "signal stop", source: "./io_signal", workingDirectory: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"), wantOutput: "ok\n", guarded: true},
		{name: "user lookup", source: "./io_user", workingDirectory: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata"), wantOutput: "ok\n", guarded: true},
		{name: "modernc libc", source: ".", workingDirectory: filepath.Join("..", "..", "..", "internal", "gomadtool", "conformance", "testdata", "libc_adapter"), wantOutput: "ok\n", denyWrites: true, libcAdapter: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := target.Spec{
				Kind: target.KindGoRun, Source: test.source, WorkingDir: test.workingDirectory,
				PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
			}
			if test.guarded {
				spec.CapabilityMode = target.CapabilityModeGuarded
			}
			if test.libcAdapter {
				moduleCache, cacheErr := target.ReadModuleCache(context.Background(), toolchainRoot)
				if cacheErr != nil {
					t.Fatal(cacheErr)
				}
				spec, _, err = profile.PrepareBuildAdapters(spec, moduleCache)
				if err != nil {
					t.Fatal(err)
				}
			}
			prepared, err := target.Prepare(context.Background(), spec)
			if err != nil {
				t.Fatal(err)
			}
			canonicalTarget, err := filepath.EvalSymlinks(prepared.Path)
			if err != nil {
				t.Fatal(err)
			}
			frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
			if err != nil {
				t.Fatal(err)
			}
			runDirectory := t.TempDir()
			environment := []string{"GOMAD3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"}
			policy := fmt.Sprintf("(version 1)(allow default)(deny network*)(deny process-fork)(deny process-exec)(allow process-exec (literal %s))", strconv.Quote(canonicalTarget))
			var targetArguments []string
			if test.hostEscape {
				hostFile := filepath.Join(t.TempDir(), "host-file")
				if err := os.WriteFile(hostFile, []byte("host"), 0o600); err != nil {
					t.Fatal(err)
				}
				policy += fmt.Sprintf("(deny file-read-data (literal %s))", strconv.Quote(hostFile))
				targetArguments = append(targetArguments, hostFile)
			}
			if test.denyWrites {
				policy += fmt.Sprintf("(deny file-write* (subpath %s))", strconv.Quote(runDirectory))
			}
			arguments := append([]string{"-p", policy, prepared.Path}, targetArguments...)
			result, err := execution.Run(context.Background(), execution.Spec{
				SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
				BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
				Command:           "/usr/bin/sandbox-exec", Args: arguments, Argv0: "sandbox-exec", Dir: runDirectory, Env: environment,
				ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 4096,
				World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
				IO:    &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
			})
			if err != nil {
				t.Fatalf("execution.Run() error = %v, result = %#v, stderr = %q", err, result, result.Stderr.Bytes)
			}
			if result.Termination != execution.TerminationExit || result.ExitCode != 0 || string(result.Stdout.Bytes) != test.wantOutput {
				t.Fatalf("result = %#v, stderr = %q", result, result.Stderr.Bytes)
			}
		})
	}
}

func TestGuardedCapabilityPublishesIOCompletionBeforeTermination(t *testing.T) {
	result, err := runIOFailureFixture(t, "guarded-io-terminal", "package main\n\nimport \"os/exec\"\n\nfunc main() { _ = exec.Command(\"true\") }\n", target.CapabilityModeGuarded)
	if err != nil {
		t.Fatalf("execution.Run() error = %v, result = %#v, stderr = %q", err, result, result.Stderr.Bytes)
	}
	if result.Termination != execution.TerminationExit || result.ExitCode == 0 || !result.IOTranscript.Complete || !strings.HasPrefix(string(result.Stderr.Bytes), "fatal error: GOMAD_CAPABILITY_DENIED") {
		t.Fatalf("result = %#v, stderr = %q", result, result.Stderr.Bytes)
	}
}

func TestTargetPanicPublishesIOCompletionBeforeTermination(t *testing.T) {
	result, err := runIOFailureFixture(t, "panic-io-terminal", "package main\n\nimport \"os\"\n\nfunc main() { _, _ = os.Stat(\"/missing\"); panic(\"boom\") }\n", target.CapabilityModeClosure)
	if err != nil {
		t.Fatalf("execution.Run() error = %v, result = %#v, stderr = %q", err, result, result.Stderr.Bytes)
	}
	if result.Termination != execution.TerminationExit || result.ExitCode == 0 || !result.IOTranscript.Complete || !strings.HasPrefix(string(result.Stderr.Bytes), "panic: boom") {
		t.Fatalf("result = %#v, stderr = %q", result, result.Stderr.Bytes)
	}
}

func runIOFailureFixture(t *testing.T, moduleName, source string, capabilityMode target.CapabilityMode) (execution.Result, error) {
	t.Helper()
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module example.com/"+moduleName+"\n\ngo 1.26.4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "main.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: ".", WorkingDir: workingDirectory,
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot, CapabilityMode: capabilityMode,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := deterministicio.Default()
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	return execution.Run(context.Background(), execution.Spec{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
		Command:           prepared.Path, Argv0: prepared.Argv[0], Dir: t.TempDir(),
		Env:              []string{"GOMAD3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"},
		ExecutionTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 4096,
		World: execution.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &execution.IOCapability{Config: frame, Transcript: &execution.IOTranscriptCapability{Limit: 64 << 20}},
	})
}
