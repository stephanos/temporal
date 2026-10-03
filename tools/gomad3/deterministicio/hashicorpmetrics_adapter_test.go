package deterministicio

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

func TestHashicorpMetricsAdapterConsumer(t *testing.T) {
	downloadPinnedModule(t, hashicorpMetricsModulePath, hashicorpMetricsVersion)
	workingDirectory := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "metrics_test.go"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "hashicorpmetrics", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	toolchainRoot, err := filepath.Abs(filepath.Join("..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := Default().PrepareBuildAdapters(target.Spec{
		Kind: target.KindGoTest, Source: ".", WorkingDir: workingDirectory,
		PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot, BuildTags: []string{"hashicorpmetrics", "test_dep"},
	}, pinnedModuleCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != "github.com/hashicorp/go-metrics" {
		t.Fatalf("metrics fixture adapters = %#v", adapters)
	}
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), goCommand,
		"test", "-v", "-p=2", "-mod=readonly", "-modfile="+spec.BuildModFile, "-tags=test_dep,hashicorpmetrics", ".")
	command.Dir = workingDirectory
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("prepared metrics consumer: %v\n%s", err, output)
	}
	t.Logf("prepared stock metrics consumer:\n%s", output)
	if _, err := target.ReviewCapabilities(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(adapters[0].Replacement)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "\"os/signal\"") {
		t.Fatal("prepared metrics consumer retains the unsupported signal dependency")
	}
}
