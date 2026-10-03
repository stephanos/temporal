package runner

import (
	"context"
	"debug/buildinfo"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestReplayBuildInfoRejectsMatchingCoverageInstrumentation(t *testing.T) {
	info := &debug.BuildInfo{GoVersion: "go1.27.1", Path: "example.com/target", Settings: []debug.BuildSetting{{Key: "-cover", Value: "true"}}}
	err := validateBuildInfo(info, target.ProjectBuildInfo(info))
	if err == nil || !strings.Contains(err.Error(), "coverage") {
		t.Fatalf("matching coverage build-info rejection = %v", err)
	}
}

func TestReplayAndMinimizeRejectRetainedCoverageBinary(t *testing.T) {
	module := t.TempDir()
	for name, contents := range map[string]string{"go.mod": "module example.com/coveragefixture\n\ngo 1.27.1\n", "main.go": "package main\nfunc main() {}\n"} {
		if err := os.WriteFile(filepath.Join(module, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	binary := filepath.Join(t.TempDir(), "target")
	command := exec.Command(filepath.Join(toolchainRoot(t), "bin", "go"), "build", "-tags=test_dep", "-cover", "-o", binary, ".")
	command.Dir = module
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "GOROOT", "GOMADSEED", "GOMAD3_CHILD_SEED", "GOFLAGS", "GOENV", "CGO_ENABLED", "GOEXPERIMENT", "GOTOOLCHAIN", "GOWORK":
			continue
		}
		command.Env = append(command.Env, entry)
	}
	command.Env = append(command.Env, "GOFLAGS=", "GOENV=off", "CGO_ENABLED=0", "GOEXPERIMENT=nogreenteagc", "GOTOOLCHAIN=local", "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build covered target: %v\n%s", err, output)
	}
	contents, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	build, err := buildinfo.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	original, _ := replayArtifact(t)
	opened, err := artifact.OpenArtifact(original)
	if err != nil {
		t.Fatal(err)
	}
	manifest := opened.Manifest()
	manifest.Target.SHA256 = record.HashBytes(contents)
	manifest.Target.Size = record.Uint64String(len(contents))
	manifest.Target.BuildInfo = target.ProjectBuildInfo(build)
	stdout, err := opened.ReadPayload(manifest.Streams.Stdout.File, uint64(manifest.Streams.Stdout.RetainedBytes))
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := opened.ReadPayload(manifest.Streams.Stderr.File, uint64(manifest.Streams.Stderr.RetainedBytes))
	if err != nil {
		t.Fatal(err)
	}
	world, err := readWorldPayloads(opened)
	if err != nil {
		t.Fatal(err)
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{Manifest: manifest, TargetPath: binary, Stdout: stdout, Stderr: stderr, World: world})
	if err != nil {
		t.Fatal(err)
	}
	for _, verifyOnly := range []bool{false, true} {
		executor := &fakeReplayExecutor{}
		_, err := replayWith(context.Background(), ReplaySpec{ArtifactPath: published.Path, VerifyOnly: verifyOnly, ToolchainRoot: toolchainRoot(t), SupervisorCommand: []string{"unused"}}, executionDependencies{executor: executor})
		if err == nil || !strings.Contains(err.Error(), "stored target uses unsupported coverage instrumentation") || executor.calls != 0 {
			t.Fatalf("covered replay: error = %v, calls = %d", err, executor.calls)
		}
	}
	_, err = minimizeWith(context.Background(), MinimizeSpec{ArtifactPath: published.Path, OutputRoot: t.TempDir(), AttemptBudget: 1, ToolchainRoot: toolchainRoot(t)}, executionDependencies{})
	if err == nil || !strings.Contains(err.Error(), "stored target uses unsupported coverage instrumentation") {
		t.Fatalf("covered minimization: error = %v", err)
	}
}
