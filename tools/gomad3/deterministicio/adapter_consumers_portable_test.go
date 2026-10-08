package deterministicio

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

func TestPortableRewrittenModuleConsumers(t *testing.T) {
	goCommand, cache := portableAdapterGo(t)
	for _, adapter := range rewrittenModuleAdapters {
		if adapter.consumer == nil {
			continue
		}
		t.Run(adapter.name, func(t *testing.T) {
			portableAdapterModule(t, cache, adapter.module, adapter.version)
			workingDirectory := t.TempDir()
			for _, name := range []string{"go.mod", "go.sum", adapter.name + "_test.go"} {
				contents, err := os.ReadFile(filepath.Join("testdata", adapter.name, name))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(workingDirectory, name), contents, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			identity := gomadversion.AdapterIdentity{Module: adapter.module, Version: adapter.version, Sum: adapter.sum}
			registry, err := newAdapterRegistry([]gomadversion.AdapterIdentity{identity}, []adapterImplementation{
				{module: adapter.module, prepare: adapter.prepare},
			})
			if err != nil {
				t.Fatal(err)
			}
			spec, adapters, err := registry.prepare(target.Spec{
				Kind: target.KindGoTest, Source: ".", WorkingDir: workingDirectory, PreparationRoot: t.TempDir(),
			}, cache)
			if err != nil {
				t.Fatal(err)
			}
			if len(adapters) != 1 || adapters[0].Module != adapter.module || adapters[0].PreparedSourceSetSHA256 != adapter.preparedSourceSetSHA256 {
				t.Fatalf("%s consumer adapter selection = %#v", adapter.name, adapters)
			}
			run := func(dir string, environment []string, args ...string) []byte {
				t.Helper()
				if len(args) > 0 && args[0] == "test" {
					args = append([]string{"test", "-tags=test_dep"}, args[1:]...)
				}
				command := exec.CommandContext(t.Context(), goCommand, args...)
				command.Dir = dir
				command.Env = append(os.Environ(), append([]string{"GOWORK=off", "GOFLAGS="}, environment...)...)
				output, err := command.CombinedOutput()
				if err != nil {
					t.Fatalf("prepared %s consumer %v: %v\n%s", adapter.name, args, err, output)
				}
				return output
			}
			consumer := adapter.consumer
			modFlags := []string{"-mod=readonly", "-modfile=" + spec.BuildModFile}
			testArguments := append(append(append([]string{"test", "-v"}, consumer.testFlags...), modFlags...), ".")
			t.Logf("prepared stock %s consumer:\n%s", adapter.name, run(workingDirectory, nil, testArguments...))
			if consumer.prepared != nil {
				consumer.prepared(t, run, adapters[0])
			}
			for _, platform := range []struct{ goos, goarch string }{{"darwin", "arm64"}, {"linux", "amd64"}} {
				host := platform.goos + "/" + platform.goarch
				environment := []string{"GOOS=" + platform.goos, "GOARCH=" + platform.goarch, "CGO_ENABLED=0"}
				listArguments := append(append(append([]string{"list"}, consumer.listFlags...), modFlags...), "-json", adapter.preparedPackage)
				var pkg consumerPackage
				if err := json.Unmarshal(run(workingDirectory, environment, listArguments...), &pkg); err != nil {
					t.Fatal(err)
				}
				pkg.workingDir, pkg.modFile = workingDirectory, spec.BuildModFile
				slices.Sort(pkg.GoFiles)
				sources := make([]compatibility.Source, len(pkg.GoFiles))
				for index, name := range pkg.GoFiles {
					contents, err := os.ReadFile(filepath.Join(pkg.Dir, name))
					if err != nil {
						t.Fatal(err)
					}
					sources[index] = compatibility.Source{Name: name, SHA256: digestBytes(contents)}
				}
				want := adapter.preparedSourceSetSHA256ByHost[host]
				if got := compatibility.DigestSources(sources); got != want {
					t.Fatalf("prepared %s source set on %s = %s, want %s", adapter.name, host, got, want)
				}
				if consumer.platform != nil {
					consumer.platform(t, run, environment, host, pkg)
				}
				t.Logf("prepared %s source set %s", host, want)
			}
		})
	}
}
