package ioprofile

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/process"
	"go.temporal.io/server/tools/gomadv3/internal/romount"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestProfileFilesystemStaysInMemory(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./io_filesystem", WorkingDir: filepath.Join("..", "..", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := Default()
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	runDirectory := t.TempDir()
	result, err := process.Run(context.Background(), process.Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"},
		BootstrapCommand:  []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
		Command:           prepared.Path, Argv0: prepared.Argv[0], Dir: runDirectory, Env: []string{"GOMADV3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"},
		RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 4096,
		World: process.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO:    &process.IOCapability{Config: frame, Transcript: &process.IOTranscriptCapability{Limit: 64 << 20}},
	})
	if err != nil {
		t.Fatalf("process.Run() error = %v, result = %#v, stderr = %q", err, result, result.Stderr.Bytes)
	}
	if result.Termination != process.TerminationExit || result.ExitCode != 0 || string(result.Stdout.Bytes) != "ok\n" {
		t.Fatalf("result = %#v, stderr = %q", result, result.Stderr.Bytes)
	}
	if _, err = os.Stat(filepath.Join(runDirectory, "workspace")); !os.IsNotExist(err) {
		t.Fatalf("profile created host filesystem state: %v", err)
	}
	if _, err = os.Stat(filepath.Join(runDirectory, "renamed")); !os.IsNotExist(err) {
		t.Fatalf("profile created host file state: %v", err)
	}
}

func TestDirectSeedFilesystemStartsEmpty(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./io_filesystem", WorkingDir: filepath.Join("..", "..", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	hostFile := filepath.Join(t.TempDir(), "host-file")
	if err := os.WriteFile(hostFile, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(prepared.Path)
	command.Env = []string{"GOMADSEED=7", "GOMADV3_HOST_ESCAPE=" + hostFile, "TZ=UTC"}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run direct-seed target: %v: %s", err, output)
	}
	if string(output) != "isolated\n" {
		t.Fatalf("output = %q", output)
	}
}

func TestProfileReadOnlyMountServesCapturedFilesInMemory(t *testing.T) {
	toolchainRoot, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := target.Prepare(context.Background(), target.Spec{
		Kind: target.KindGoRun, Source: "./io_ro_mount", WorkingDir: filepath.Join("..", "..", "testdata"),
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile := Default()
	frame, err := profile.BootstrapFrame(prepared, "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "schema.sql"), []byte("select 1;\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "empty"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	runDirectory := t.TempDir()
	if err := os.WriteFile(filepath.Join(runDirectory, "undeclared"), []byte("host secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := process.Run(context.Background(), process.Request{
		SupervisorCommand: []string{os.Args[0], "-test.run=TestEntropySupervisorHelper"}, BootstrapCommand: []string{os.Args[0], "-test.run=TestEntropyBootstrapHelper"},
		Command: prepared.Path, Argv0: prepared.Argv[0], Dir: runDirectory, Env: []string{"GOMADV3_IO_PROFILE=" + profile.Name(), "GOMADSEED=7", "TZ=UTC"},
		RunTimeout: 10 * time.Second, TerminateGrace: time.Second, OutputLimit: 4096,
		World: process.WorldCapability{RecordLimit: 1 << 20, TransitionLimit: 1 << 20, Seed: 7},
		IO: &process.IOCapability{Config: frame, Transcript: &process.IOTranscriptCapability{Limit: 64 << 20},
			ReadOnlyMount: &process.ReadOnlyMountCapability{Mappings: []romount.Mapping{{Source: source, Target: "/mounted"}}, Limits: romount.DefaultLimits()}},
	})
	if err != nil {
		t.Fatalf("process.Run() error = %v, result = %#v, stderr = %q", err, result, result.Stderr.Bytes)
	}
	if result.ExitCode != 0 || string(result.Stdout.Bytes) != "ok\n" {
		t.Fatalf("result = %#v, stderr = %q", result, result.Stderr.Bytes)
	}
}
