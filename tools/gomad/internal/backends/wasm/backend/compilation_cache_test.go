package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

func compilationCacheProvider(t *testing.T) (*Provider, target.Spec, *[][]string) {
	t.Helper()
	root := t.TempDir()
	write := func(name string, data []byte, mode os.FileMode) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, mode); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"tools/asm", "tools/compile", "tools/link", "pkg/include/textflag.h", "fixture_test.go"} {
		write(name, []byte("fixture input"), 0600)
	}
	write("module.wasm", []byte{0, 97, 115, 109, 1, 0, 0, 0, 5, 3, 1, 0, 1}, 0600)
	environment, err := json.Marshal(map[string]string{"GOTOOLDIR": filepath.Join(root, "tools"), "GOROOT": root})
	if err != nil {
		t.Fatal(err)
	}
	listed, err := json.Marshal(map[string]any{"Dir": root, "TestGoFiles": []string{"fixture_test.go"}})
	if err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf("#!/bin/sh\ncase \"$1\" in\nversion) printf 'go version go1.27.1 linux/amd64\\n';;\nenv) printf '%%s' '%s';;\nlist) printf '%%s' '%s';;\ntest) [ \"$2\" = '-c' ] && [ \"$3\" = '-mod=readonly' ] && [ \"$4\" = '-trimpath' ] && [ \"$5\" = '-buildvcs=false' ] && [ \"$6\" = '-pgo=off' ] && [ \"$7\" = '-o' ] || exit 91\n/bin/cp '%s' \"$8\";;\n*) exit 92;;\nesac\n", environment, listed, filepath.Join(root, "module.wasm"))
	write("go", []byte(script), 0700)
	write("helper", []byte("helper"), 0700)
	options := portableProviderOptions(wasi.StockProfile)
	options.CompilerPath, options.CompilerSHA256 = filepath.Join(root, "go"), string(record.HashBytes([]byte(script)))
	options.HelperPath, options.HelperSHA256 = filepath.Join(root, "helper"), string(record.HashBytes([]byte("helper")))
	options.CacheRoot = filepath.Join(root, "cache")
	var commands [][]string
	options.ObserveCompilerCommand = func(metrics CompilerCommandMetrics) { commands = append(commands, metrics.Arguments) }
	p, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	return p, target.Spec{Backend: Name, Kind: target.KindGoTest, Source: ".", WorkingDir: root, PreparationRoot: t.TempDir(), Args: []string{"-test.run=^TestFirst$"}}, &commands
}

func TestCompilationCacheReusesModuleWithCurrentExecutionProvenance(t *testing.T) {
	t.Parallel()
	p, spec, commands := compilationCacheProvider(t)
	first, err := p.Prepare(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"selector", "configuration", "helper", "limits", "soft-assertion"} {
		switch change {
		case "selector":
			spec.Args = []string{"-test.run=^TestSecond$"}
		case "configuration":
			p.options.Config.Clock.ReadStepNanos++
		case "helper":
			if err := os.WriteFile(p.options.HelperPath, []byte("new helper"), 0700); err != nil {
				t.Fatal(err)
			}
			p.options.HelperSHA256 = string(record.HashBytes([]byte("new helper")))
		case "limits":
			p.options.Fuel++
		case "soft-assertion":
			spec.SoftAssertionReportOnly = true
		}
		*commands = nil
		spec.PreparationRoot = t.TempDir()
		prepared, err := p.Prepare(t.Context(), spec)
		if err != nil {
			t.Fatalf("%s: %v", change, err)
		}
		for _, args := range *commands {
			if args[0] == "test" || args[0] == "build" {
				t.Fatalf("%s recompiled: %q", change, args)
			}
		}
		if first.SHA256 != prepared.SHA256 || first.BuildKey == prepared.BuildKey {
			t.Fatalf("%s lost module reuse or distinct execution identity", change)
		}
		pr, err := decodeProvenance(prepared.BackendPayloads[0].Data)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(pr.Config, p.options.Config) || !reflect.DeepEqual(pr.Args, spec.Args) || pr.HelperSHA256 != p.options.HelperSHA256 || pr.Fuel != p.options.Fuel || pr.SoftAssertionReportOnly != spec.SoftAssertionReportOnly {
			t.Fatalf("%s retained stale execution provenance", change)
		}
		if err := p.ValidatePrepared(spec, prepared, nil); err != nil {
			t.Fatal(err)
		}
		first = prepared
	}
}

func TestCompilationCacheInvalidatesChangedCompileInputs(t *testing.T) {
	t.Parallel()
	for _, change := range []string{"source", "tags", "compiler", "compiler-tool", "header"} {
		t.Run(change, func(t *testing.T) {
			t.Parallel()
			p, spec, commands := compilationCacheProvider(t)
			first, err := p.Prepare(t.Context(), spec)
			if err != nil {
				t.Fatal(err)
			}
			*commands = nil
			var path string
			switch change {
			case "source":
				path = filepath.Join(spec.WorkingDir, "fixture_test.go")
			case "tags":
				spec.BuildTags = []string{"test_dep"}
			case "compiler":
				path = p.options.CompilerPath
			case "compiler-tool":
				path = filepath.Join(spec.WorkingDir, "tools/compile")
			case "header":
				path = filepath.Join(spec.WorkingDir, "pkg/include/textflag.h")
			}
			if path != "" {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				data = append(data, '\n')
				if err := os.WriteFile(path, data, 0700); err != nil {
					t.Fatal(err)
				}
				if change == "compiler" {
					p.options.CompilerSHA256 = string(record.HashBytes(data))
				}
			}
			spec.PreparationRoot = t.TempDir()
			second, err := p.Prepare(t.Context(), spec)
			if err != nil {
				t.Fatal(err)
			}
			builds := 0
			for _, args := range *commands {
				if args[0] == "test" {
					builds++
				}
			}
			if builds != 1 || first.BuildKey == second.BuildKey {
				t.Fatalf("%s did not invalidate compilation: builds=%d", change, builds)
			}
		})
	}
}

func TestCompilationKeyIncludesRuntimeOverlayAndBuildEffects(t *testing.T) {
	t.Parallel()
	base := provenance{Profile: wasi.CooperativeProfile, RuntimeSHA256: "runtime", RuntimeImplementationSHA256: "implementation", BuildFlags: []string{"-overlay", "overlay.json"}, BuildEnvironment: []string{"GOOS=wasip1"}, Kind: target.KindGoTest, Source: "."}
	first, err := compilationFor(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"runtime", "runtime-implementation", "overlay", "environment", "kind", "source"} {
		t.Run(change, func(t *testing.T) {
			t.Parallel()
			pr := base
			switch change {
			case "runtime":
				pr.RuntimeSHA256 = "changed"
			case "runtime-implementation":
				pr.RuntimeImplementationSHA256 = "changed"
			case "overlay":
				pr.BuildFlags = []string{"-overlay", "changed.json"}
			case "environment":
				pr.BuildEnvironment = []string{"GOOS=other"}
			case "kind":
				pr.Kind = target.KindGoRun
			case "source":
				pr.Source = "./other"
			}
			second, err := compilationFor(pr)
			if err != nil {
				t.Fatal(err)
			}
			if first.BuildKey == second.BuildKey {
				t.Fatal("changed compile input reused key")
			}
		})
	}
}

func compilationCachePath(t *testing.T, p *Provider, prepared target.Prepared) string {
	t.Helper()
	pr, err := decodeProvenance(prepared.BackendPayloads[0].Data)
	if err != nil {
		t.Fatal(err)
	}
	compilation, err := compilationFor(pr)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(p.options.CacheRoot, "compilation-v1", compilation.BuildKey)
}

func TestCompilationCacheRejectsCorruption(t *testing.T) {
	t.Parallel()
	for _, corruption := range []string{"module", "receipt", "size", "imports", "memory", "inputs", "key", "schema"} {
		t.Run(corruption, func(t *testing.T) {
			t.Parallel()
			p, spec, commands := compilationCacheProvider(t)
			prepared, err := p.Prepare(t.Context(), spec)
			if err != nil {
				t.Fatal(err)
			}
			cache := compilationCachePath(t, p, prepared)
			path := filepath.Join(cache, "receipt.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var receipt compilationReceipt
			if err := json.Unmarshal(data, &receipt); err != nil {
				t.Fatal(err)
			}
			switch corruption {
			case "module":
				path, data = filepath.Join(cache, "module.wasm"), []byte("corrupt")
			case "receipt":
				data = []byte("corrupt")
			default:
				switch corruption {
				case "size":
					receipt.ModuleBytes++
				case "imports":
					receipt.Imports = []wasi.Import{{Module: "unknown", Name: "unknown"}}
				case "memory":
					receipt.InitialMemoryPages++
				case "inputs":
					receipt.Inputs.Source = "other"
				case "key":
					receipt.BuildKey = "other"
				case "schema":
					receipt.Schema = "other"
				}
				data, err = json.Marshal(receipt)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Chmod(path, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			*commands = nil
			spec.PreparationRoot = t.TempDir()
			if _, err := p.Prepare(t.Context(), spec); err == nil {
				t.Fatal("corrupt cache accepted")
			}
			for _, args := range *commands {
				if args[0] == "test" {
					t.Fatal("corrupt cache silently rebuilt")
				}
			}
		})
	}
}
