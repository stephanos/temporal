package deterministicio

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

func TestPortableServerGraphPreparedSourceIdentities(t *testing.T) {
	goCommand, cache := portableAdapterGo(t)
	working := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum"} {
		contents, err := os.ReadFile(filepath.Join("..", "..", "..", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(working, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var imports strings.Builder
	for _, adapter := range rewrittenModuleAdapters {
		if !adapter.outsideServerGraph {
			imports.WriteString("import _ \"" + adapter.importPath + "\"\n")
		}
	}
	if err := os.WriteFile(filepath.Join(working, "main.go"), []byte("package main\n\n"+imports.String()+"\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, adapters, err := deterministicAdapters.prepare(target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: working, PreparationRoot: t.TempDir()}, cache)
	if err != nil {
		t.Fatal(err)
	}
	selected := map[string]BuildAdapter{}
	for _, adapter := range adapters {
		selected[adapter.Module] = adapter
	}
	for _, adapter := range rewrittenModuleAdapters {
		if !adapter.outsideServerGraph {
			if _, found := selected[adapter.module]; !found {
				t.Fatalf("%s adapter was not selected: %#v", adapter.name, adapters)
			}
		}
	}
	for _, prepared := range adapters {
		var pins map[string]string
		for _, definition := range deterministicAdapters.definitions {
			if definition.identity.Module != prepared.Module {
				continue
			}
			pins = libcPreparedSourceSetSHA256ByHost
			if definition.implementation.rewritten != nil {
				pins = definition.implementation.rewritten.preparedSourceSetSHA256ByHost
			}
		}
		if len(pins) != 2 {
			t.Fatalf("prepared graph module %s has no dual source pins", prepared.Module)
		}
		for _, platform := range []string{"darwin/arm64", "linux/amd64"} {
			t.Run(prepared.Module+"/"+platform, func(t *testing.T) {
				goos, goarch, _ := strings.Cut(platform, "/")
				command := exec.CommandContext(t.Context(), goCommand, "list", "-mod=readonly", "-modfile="+spec.BuildModFile, "-tags=test_dep", "-json", prepared.PreparedPackage)
				command.Dir = working
				command.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0", "GOWORK=off", "GOFLAGS=")
				var stderr bytes.Buffer
				command.Stderr = &stderr
				output, err := command.Output()
				if err != nil {
					t.Fatalf("server graph package listing: %v\n%s\n%s", err, output, &stderr)
				}
				var pkg consumerPackage
				if err := json.Unmarshal(output, &pkg); err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(pkg.Dir, prepared.ReplacementRoot+string(filepath.Separator)) && pkg.Dir != prepared.ReplacementRoot {
					t.Fatalf("server graph selected unprepared package %s", pkg.Dir)
				}
				got, err := target.AdapterPreparedSourceSetSHA256(t.Context(), goCommand, pkg.Dir, prepared.PreparedPackage, goos, goarch)
				if err != nil || got != pins[platform] {
					t.Fatalf("server graph %s source set = %s (%v), want %s", prepared.Module, got, err, pins[platform])
				}
				t.Logf("root graph selected %s from %s; %s source owner digest %s", prepared.PreparedPackage, pkg.Dir, platform, got)
			})
		}
	}
}
