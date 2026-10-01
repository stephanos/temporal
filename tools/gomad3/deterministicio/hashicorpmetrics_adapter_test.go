package deterministicio

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
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

func TestHashicorpMetricsRejectsChangedIdentity(t *testing.T) {
	for _, identity := range []gomadversion.AdapterIdentity{
		{Module: "github.com/armon/go-metrics", Version: hashicorpMetricsVersion, Sum: hashicorpMetricsSum},
		{Module: hashicorpMetricsModulePath, Version: "v0.5.3", Sum: hashicorpMetricsSum},
		{Module: hashicorpMetricsModulePath, Version: hashicorpMetricsVersion, Sum: "h1:changed"},
	} {
		if _, err := prepareHashicorpMetrics(t.TempDir(), t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
			t.Fatalf("changed identity %#v: %v", identity, err)
		}
	}
}

func TestHashicorpMetricsRewriteRejectsDrift(t *testing.T) {
	downloadPinnedModule(t, hashicorpMetricsModulePath, hashicorpMetricsVersion)
	moduleRoot := filepath.Join(pinnedModuleCache(t), "github.com", "hashicorp", "go-metrics@"+hashicorpMetricsVersion)
	source, err := readAdapterSource(hashicorpMetricsModulePath, moduleRoot, hashicorpMetricsSignalPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name         string
		contents     []byte
		anchor       []byte
		outputSHA256 string
		want         string
	}{
		{name: "upstream-edit", contents: append(append([]byte(nil), source...), '\n'), want: "source identity mismatch"},
		{name: "missing-anchor", anchor: []byte("missing signal source anchor"), want: "anchor mismatch"},
		{name: "ambiguous-anchor", anchor: []byte("\ti.stop"), want: "anchor mismatch"},
		{name: "replacement-edit", outputSHA256: hashicorpMetricsSignalSourceSHA256, want: "replacement identity mismatch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rewrite := hashicorpMetricsRewrites[0]
			rewrite.rewrites = append([]anchorRewrite(nil), rewrite.rewrites...)
			contents := source
			if test.contents != nil {
				contents = test.contents
			}
			if test.anchor != nil {
				rewrite.rewrites[0].anchor = test.anchor
			}
			if test.outputSHA256 != "" {
				rewrite.replacementSHA256 = test.outputSHA256
			}
			if _, err := rewriteAdapterSource(hashicorpMetricsModulePath, rewrite, contents); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("rewrite source: %v, want %s", err, test.want)
			}
		})
	}
}

func TestHashicorpMetricsRejectsUnrewrittenInventoryDrift(t *testing.T) {
	downloadPinnedModule(t, hashicorpMetricsModulePath, hashicorpMetricsVersion)
	moduleCache := t.TempDir()
	moduleRoot := filepath.Join(moduleCache, "github.com", "hashicorp", "go-metrics@"+hashicorpMetricsVersion)
	if err := os.MkdirAll(filepath.Dir(moduleRoot), 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(pinnedModuleCache(t), "github.com", "hashicorp", "go-metrics@"+hashicorpMetricsVersion)
	if err := copyAdapterModule(source, moduleRoot, nil, defaultAdapterCopyLimits); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(moduleRoot, "statsd.go")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(contents, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	identity := gomadversion.AdapterIdentity{Module: hashicorpMetricsModulePath, Version: hashicorpMetricsVersion, Sum: hashicorpMetricsSum}
	if _, err := prepareHashicorpMetrics(moduleCache, t.TempDir(), identity); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
		t.Fatalf("changed module inventory: %v", err)
	}
}

func TestProfileRejectsHashicorpMetricsConfigurationDrift(t *testing.T) {
	for _, test := range []struct {
		name         string
		version      string
		replace      string
		sum          string
		buildModFile bool
		want         string
	}{
		{name: "version", version: "v0.5.3", want: "unsupported github.com/hashicorp/go-metrics version"},
		{name: "replacement", replace: "replace github.com/hashicorp/go-metrics => ./metrics\n", want: "already replaces github.com/hashicorp/go-metrics"},
		{name: "version-replacement", replace: "replace github.com/hashicorp/go-metrics v0.5.4 => ./metrics\n", want: "already replaces github.com/hashicorp/go-metrics"},
		{name: "sum", sum: "h1:changed", want: "module sum"},
		{name: "missing-sum", want: "module sum"},
		{name: "build-modfile", buildModFile: true, want: "existing build modfile"},
	} {
		t.Run(test.name, func(t *testing.T) {
			workingDirectory := t.TempDir()
			version := test.version
			if version == "" {
				version = hashicorpMetricsVersion
			}
			modFile := "module example.test\n\ngo 1.27.1\n\nrequire " + hashicorpMetricsModulePath + " " + version + "\n" + test.replace
			if err := os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte(modFile), 0o600); err != nil {
				t.Fatal(err)
			}
			if test.sum != "" {
				if err := os.WriteFile(filepath.Join(workingDirectory, "go.sum"), []byte(hashicorpMetricsModulePath+" "+version+" "+test.sum+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			spec := target.Spec{WorkingDir: workingDirectory, PreparationRoot: t.TempDir()}
			if test.buildModFile {
				spec.BuildModFile = filepath.Join(workingDirectory, "existing.mod")
			}
			if _, _, err := Default().PrepareBuildAdapters(spec, t.TempDir()); !IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("prepare configuration: %v, want %s", err, test.want)
			}
		})
	}
}
