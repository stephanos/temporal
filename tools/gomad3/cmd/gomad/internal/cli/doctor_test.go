package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCheckReportsAvailableContract(t *testing.T) {
	root, runner, artifacts := writeDoctorFixture(t, runtime.GOOS, runtime.GOARCH)
	report := Check(Config{
		ToolchainRoot: filepath.Join(root, ".toolchain"), InstallationSource: "test", RepairInstruction: "repair test toolchain",
		RunnerPath: runner, ArtifactRoot: artifacts, HostOS: runtime.GOOS, HostArch: runtime.GOARCH,
	})
	if !report.Available || report.GoVersion != "go1.27.1" || report.ToolchainBuild != strings.Repeat("a", 64) {
		t.Fatalf("report = %#v", report)
	}
	gotAdapters := make([]string, len(report.Adapters))
	for index, adapter := range report.Adapters {
		gotAdapters[index] = adapter.Module + ":" + adapter.Status
	}
	wantAdapters := []string{"github.com/Masterminds/sprig/v3:available", "github.com/cactus/go-statsd-client/v5:available", "github.com/cockroachdb/pebble:available", "github.com/getsentry/sentry-go:available", "github.com/go-playground/validator/v10:available", "github.com/hashicorp/go-metrics:available", "github.com/hashicorp/go-sockaddr:available", "github.com/hashicorp/memberlist:available", "go.opentelemetry.io/otel/sdk:available", "go.temporal.io/sdk:available", "go.uber.org/fx:available", "golang.org/x/net:available", "google.golang.org/grpc:available", "modernc.org/libc:available", "modernc.org/memory:available"}
	if report.RunnerBuild == "" || report.BoundaryManifestVersion == "" || strings.Join(gotAdapters, ",") != strings.Join(wantAdapters, ",") {
		t.Fatalf("identity = %#v", report)
	}
	if report.ArtifactDirectory != artifacts || report.InstallationSource != "test" || report.RepairInstruction != "repair test toolchain" || len(report.Checks) != 19 {
		t.Fatalf("diagnostics = %#v", report)
	}
	for _, check := range report.Checks {
		if check.Status != "ok" {
			t.Fatalf("check = %#v", check)
		}
	}
}

func TestCheckExplainsMissingToolchain(t *testing.T) {
	root := t.TempDir()
	runner := filepath.Join(root, ".bin", "gomad")
	if err := os.MkdirAll(filepath.Dir(runner), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runner, []byte("runner"), 0o700); err != nil {
		t.Fatal(err)
	}
	report := Check(Config{
		ToolchainRoot: filepath.Join(root, ".toolchain"), InstallationSource: "test", RepairInstruction: "repair test toolchain",
		RunnerPath: runner, ArtifactRoot: filepath.Join(root, "artifacts"), HostOS: "darwin", HostArch: "arm64",
	})
	if report.Available || report.RepairInstruction != "repair test toolchain" {
		t.Fatalf("report = %#v", report)
	}
	toolchain := findCheck(t, report, "toolchain")
	if toolchain.Status != "error" || !strings.Contains(toolchain.Detail, report.RepairInstruction) {
		t.Fatalf("toolchain check = %#v", toolchain)
	}
}

func TestCheckRejectsUnsupportedHostAndUnwritableArtifacts(t *testing.T) {
	root, runner, artifacts := writeDoctorFixture(t, "darwin", "arm64")
	if err := os.MkdirAll(artifacts, 0o700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(artifacts, "not-a-directory")
	if err := os.WriteFile(file, []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	report := Check(Config{
		ToolchainRoot: filepath.Join(root, ".toolchain"), InstallationSource: "test", RepairInstruction: "repair test toolchain",
		RunnerPath: runner, ArtifactRoot: filepath.Join(file, "child"), HostOS: "linux", HostArch: "arm64",
	})
	if report.Available || findCheck(t, report, "host").Status != "error" || findCheck(t, report, "artifacts").Status != "error" {
		t.Fatalf("report = %#v", report)
	}
}

func findCheck(t *testing.T, report Report, name string) CheckResult {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("check %q is absent", name)
	return CheckResult{}
}

func writeDoctorFixture(t *testing.T, goos, goarch string) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	key := strings.Repeat("a", 64)
	for _, directory := range []string{filepath.Join(root, ".toolchain", "bin"), filepath.Join(root, ".toolchain", "builds", key, "bin"), filepath.Join(root, ".bin")} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	goScript := "#!/bin/sh\nprintf 'go1.27.1\\n" + goos + "\\n" + goarch + "\\n0\\n'\n"
	for _, path := range []string{filepath.Join(root, ".toolchain", "bin", "go"), filepath.Join(root, ".toolchain", "builds", key, "bin", "go")} {
		if err := os.WriteFile(path, []byte(goScript), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".toolchain", "build-key"), []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := filepath.Join(root, ".bin", "gomad")
	if err := os.WriteFile(runner, []byte("runner"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root, runner, filepath.Join(root, "artifacts")
}
