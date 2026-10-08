package deterministicio

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

func TestPortableMetricsConsumer(t *testing.T) {
	goCommand, cache := portableAdapterGo(t)
	portableAdapterModule(t, cache, hashicorpMetricsModulePath, hashicorpMetricsVersion)
	working := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum", "metrics_test.go"} {
		contents, err := os.ReadFile(filepath.Join("testdata", "hashicorpmetrics", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(working, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	spec, adapters, err := deterministicAdapters.prepare(target.Spec{Kind: target.KindGoTest, Source: ".", WorkingDir: working, PreparationRoot: t.TempDir(), BuildTags: []string{"hashicorpmetrics", "test_dep"}}, cache)
	if err != nil {
		t.Fatal(err)
	}
	if len(adapters) != 1 || adapters[0].Module != hashicorpMetricsModulePath {
		t.Fatalf("metrics fixture adapters = %+v", adapters)
	}
	command := exec.CommandContext(t.Context(), goCommand, "test", "-v", "-p=2", "-mod=readonly", "-modfile="+spec.BuildModFile, "-tags=test_dep,hashicorpmetrics", ".")
	command.Dir = working
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("prepared stock metrics consumer: %v\n%s", err, output)
	}
	t.Logf("prepared stock metrics consumer:\n%s", output)
	contents, err := os.ReadFile(adapters[0].Replacement)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "\"os/signal\"") {
		t.Fatal("prepared metrics consumer retains the unsupported signal dependency")
	}
	for _, platform := range []string{"darwin/arm64", "linux/amd64"} {
		goos, goarch, _ := strings.Cut(platform, "/")
		got, err := target.AdapterPreparedSourceSetSHA256(t.Context(), goCommand, adapters[0].ReplacementRoot, hashicorpMetricsModulePath, goos, goarch)
		if err != nil || got != hashicorpMetricsPreparedSourceSetSHA256ByHost[platform] {
			t.Fatalf("stock metrics %s source set = %s (%v)", platform, got, err)
		}
	}
}
