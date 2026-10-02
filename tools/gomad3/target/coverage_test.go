package target

import (
	"context"
	"debug/buildinfo"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	targetbuild "go.temporal.io/server/tools/gomad3/target/internal/build"
)

func TestPinnedCoverageBuildSettingsRejectInstrumentation(t *testing.T) {
	module := writeModule(t, map[string]string{
		"go.mod":       "module example.com/coveragefixture\n\ngo 1.27.1\n",
		"main.go":      "package main\nfunc main() {}\n",
		"main_test.go": "package main\nimport \"testing\"\nfunc TestMainBody(t *testing.T) { main() }\n",
	})
	identity, err := ReadToolchainIdentity(toolchainRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name    string
		args    []string
		covered bool
	}{
		{name: "build", args: []string{"build", "-tags=test_dep"}},
		{name: "build-cover", args: []string{"build", "-tags=test_dep", "-cover"}, covered: true},
		{name: "test-cover", args: []string{"test", "-tags=test_dep", "-c", "-cover"}, covered: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "target")
			args := append(append([]string(nil), test.args...), "-o", binary, ".")
			command := exec.Command(filepath.Join(toolchainRoot(t), "bin", "go"), args...)
			command.Dir = module
			command.Env = targetbuild.Environment()
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("build: %v\n%s", err, output)
			}
			info, err := buildinfo.ReadFile(binary)
			if err != nil {
				t.Fatal(err)
			}
			projected := ProjectBuildInfo(info)
			t.Logf("pinned %s settings: %#v", test.name, projected.Settings)
			cover := ""
			for _, setting := range projected.Settings {
				if setting.Key == "-cover" {
					cover = setting.Value
				}
			}
			if test.covered && cover != "true" {
				t.Fatalf("coverage build omitted -cover=true: %#v", projected.Settings)
			}

			contents, err := os.ReadFile(binary)
			if err != nil {
				t.Fatal(err)
			}
			wire := provenanceWire{Schema: provenanceSchema, SchemaVersion: 3, GoVersion: identity.GoVersion, BuildKey: identity.BuildKey, TargetGOOS: identity.TargetGOOS, TargetGOARCH: identity.TargetGOARCH, BinarySHA256: string(record.HashBytes(contents)), BinarySize: record.Uint64String(len(contents)), BuildInfo: projected, CapabilityClosure: validCapabilityClosure(), CapabilityMode: CapabilityModeClosure}
			if test.covered {
				encoded, err := canonicaljson.CanonicalJSON(wire)
				if err != nil {
					t.Fatal(err)
				}
				provenance := filepath.Join(t.TempDir(), "provenance.json")
				if err := os.WriteFile(provenance, encoded, 0o600); err != nil {
					t.Fatal(err)
				}
				_, err = Prepare(context.Background(), Spec{Kind: KindExec, Source: binary, Provenance: provenance, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t)})
				if err == nil || !strings.Contains(err.Error(), "exec provenance uses unsupported coverage instrumentation") {
					t.Fatalf("exec coverage rejection = %v", err)
				}
			}
			err = validateDeterministicBuildInfo(projected)
			if test.covered {
				if err == nil || !strings.Contains(err.Error(), "coverage") {
					t.Fatalf("coverage rejection = %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
