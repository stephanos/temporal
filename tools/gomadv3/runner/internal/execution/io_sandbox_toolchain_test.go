package execution_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/runner/internal/execution"
	"go.temporal.io/server/tools/gomadv3/target"
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
	}{
		{name: "filesystem", source: "./io_filesystem", workingDirectory: filepath.Join("..", "..", "..", "toolchain", "internal", "conformance", "testdata"), wantOutput: "isolated\n", hostEscape: true, denyWrites: true},
		{name: "network", source: "./io_net", workingDirectory: filepath.Join("..", "..", "..", "toolchain", "internal", "conformance", "testdata"), wantOutput: "ok\n"},
		{name: "modernc libc", source: ".", workingDirectory: filepath.Join("..", "..", "..", "toolchain", "internal", "conformance", "testdata", "libc_adapter"), wantOutput: "ok\n", denyWrites: true, libcAdapter: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := target.Spec{
				Kind: target.KindGoRun, Source: test.source, WorkingDir: test.workingDirectory,
				PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
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
			environment := []string{"GOMADV3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"}
			policy := fmt.Sprintf("(version 1)(allow default)(deny network*)(deny process-fork)(deny process-exec)(allow process-exec (literal %s))", strconv.Quote(canonicalTarget))
			if test.hostEscape {
				hostFile := filepath.Join(t.TempDir(), "host-file")
				if err := os.WriteFile(hostFile, []byte("host"), 0o600); err != nil {
					t.Fatal(err)
				}
				policy += fmt.Sprintf("(deny file-read-data (literal %s))", strconv.Quote(hostFile))
				environment = append(environment, "GOMADV3_HOST_ESCAPE="+hostFile)
			}
			if test.denyWrites {
				policy += fmt.Sprintf("(deny file-write* (subpath %s))", strconv.Quote(runDirectory))
			}
			result, err := execution.Run(context.Background(), execution.Spec{
				SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
				BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
				Command:           "/usr/bin/sandbox-exec", Args: []string{"-p", policy, prepared.Path}, Argv0: "sandbox-exec", Dir: runDirectory, Env: environment,
				RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 4096,
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
