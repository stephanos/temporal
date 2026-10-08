package preparation

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/toolchain/installation"
)

func sourceInstallation(t *testing.T) (installation.Description, string) {
	t.Helper()
	goCommand := os.Getenv("GOMAD3_SOURCE_GO")
	if goCommand == "" {
		t.Skip("supported-source cross-build control requested separately")
	}
	profile := deterministicio.Default().TargetContract()
	root := t.TempDir()
	key := strings.Repeat("7", 64)
	count := filepath.Join(root, "build-count")
	script := "#!/bin/sh\nset -eu\ncase \"$1\" in build) printf 'build\\n' >> " + strconv.Quote(count) + ";; test) if [ \"${2:-}\" = -c ]; then printf 'build\\n' >> " + strconv.Quote(count) + "; fi;; esac\nGOOS=" + profile.GOOS + " GOARCH=" + profile.GOARCH + " exec " + strconv.Quote(goCommand) + " \"$@\"\n"
	for _, name := range []string{filepath.Join(root, "bin", "go"), filepath.Join(root, "builds", key, "bin", "go")} {
		if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, filepath.Join(root, "build-key"), key+"\n")
	description, err := installation.Describe(root)
	if err != nil {
		t.Fatal(err)
	}
	return description, count
}

func sourceBuildCount(t *testing.T, path string) int {
	t.Helper()
	contents, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(contents), "build\n")
}

func TestPrepareSourceFreshCacheAndSharedDurableRoot(t *testing.T) {
	installation, count := sourceInstallation(t)
	cache := installation.PinnedBuild().PreparedTargets()
	if _, err := os.Lstat(cache); !os.IsNotExist(err) || sourceBuildCount(t, count) != 0 {
		t.Fatalf("initial cache is not empty: %v", err)
	}
	module := t.TempDir()
	dependency := t.TempDir()
	writeTestFile(t, filepath.Join(dependency, "go.mod"), "module example.com/local\n\ngo 1.27.1\n")
	writeTestFile(t, filepath.Join(dependency, "value.go"), "package local\nfunc Value() int { return 7 }\n")
	writeTestFile(t, filepath.Join(module, "go.mod"), "module example.com/external\n\ngo 1.27.1\n\nrequire example.com/local v0.0.0\nreplace example.com/local => "+dependency+"\n")
	writeTestFile(t, filepath.Join(module, "main.go"), "package main\nimport \"example.com/local\"\nvar value = local.Value()\nfunc main() {}\n")
	root := t.TempDir()
	sibling := filepath.Join(root, ".adapter-work-caller")
	writeTestFile(t, sibling, "caller-owned")
	var first target.Prepared
	for index := 0; index < 2; index++ {
		prepared, err := Prepare(t.Context(), Request{Target: target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: module, PreparationRoot: root, ToolchainRoot: installation.Root()}})
		if err != nil {
			t.Fatal(err)
		}
		if err := prepared.Verify(); err != nil {
			t.Fatal(err)
		}
		if sourceBuildCount(t, count) != 1 {
			t.Fatalf("preparation %d: build count = %d, want 1", index, sourceBuildCount(t, count))
		}
		if prepared.BuildInfo.MainModule != "example.com/external" || prepared.Adapters == nil || len(prepared.Adapters) != 0 {
			t.Fatalf("prepared = %#v", prepared)
		}
		if index == 0 {
			first = prepared
		} else {
			if first.Path == prepared.Path {
				t.Fatal("independent preparations share a destination")
			}
			if err := first.Verify(); err != nil {
				t.Fatalf("second preparation changed first binary: %v", err)
			}
			prepared.Path = first.Path
			if !reflect.DeepEqual(prepared, first) || !reflect.DeepEqual(prepared.RecordTarget(), first.RecordTarget()) {
				t.Fatalf("fresh and cache target differ: %#v %#v", first, prepared)
			}
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".adapter-work-") && entry.Name() != filepath.Base(sibling) {
				t.Fatalf("private workspace survived: %s", entry.Name())
			}
		}
	}
	data, err := os.ReadFile(sibling)
	if err != nil || string(data) != "caller-owned" {
		t.Fatalf("caller sibling = %q, %v", data, err)
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 1 {
		t.Fatalf("prepared cache entries = %v, %v", entries, err)
	}
	lock, err := hostfs.Try(filepath.Join(installation.PinnedBuild().TargetCache(), "gomad-cache.lock"))
	if err != nil {
		t.Fatalf("build cache lock remains held: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	t.Logf("empty prepared cache -> one real cross-build -> one cache restore; entries=%d; lock released", len(entries))
}

func TestPrepareSourceAdapterErrorsCleanWorkspace(t *testing.T) {
	installation, count := sourceInstallation(t)
	fixture := filepath.Join(testModuleRoot(t), "deterministicio", "testdata", "sprig")
	mod, err := os.ReadFile(filepath.Join(fixture, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile(filepath.Join(fixture, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"missing sum", "invalid sum", "replacement conflict"} {
		t.Run(name, func(t *testing.T) {
			module := t.TempDir()
			moduleFile := string(mod)
			if name == "replacement conflict" {
				moduleFile += "\nreplace github.com/Masterminds/sprig/v3 => ./other\n"
			}
			writeTestFile(t, filepath.Join(module, "go.mod"), moduleFile)
			var sums string
			if name != "missing sum" {
				sums = string(sum)
				if name == "invalid sum" {
					sums = strings.ReplaceAll(sums, "h1:", "h1:modified")
				}
				writeTestFile(t, filepath.Join(module, "go.sum"), sums)
			}
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, ".adapter-work-caller"), "caller-owned")
			_, err := Prepare(t.Context(), Request{Target: target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: module, PreparationRoot: root, ToolchainRoot: installation.Root()}})
			if StageOf(err) != StageAdapters || !deterministicio.IsInvalidBuildAdapterConfiguration(err) {
				t.Fatalf("adapter error = %v, stage = %s", err, StageOf(err))
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 1 || entries[0].Name() != ".adapter-work-caller" {
				t.Fatalf("adapter failure cleanup = %v, %v", entries, err)
			}
			actual, err := os.ReadFile(filepath.Join(module, "go.mod"))
			if err != nil || string(actual) != moduleFile {
				t.Fatalf("target module changed: %q, %v", actual, err)
			}
			actual, err = os.ReadFile(filepath.Join(module, "go.sum"))
			if name == "missing sum" {
				if !os.IsNotExist(err) {
					t.Fatalf("missing sum file changed: %q, %v", actual, err)
				}
			} else if err != nil || string(actual) != sums {
				t.Fatalf("target sums changed: %q, %v", actual, err)
			}
		})
	}
	if sourceBuildCount(t, count) != 0 {
		t.Fatal("invalid adapter inputs reached target compilation")
	}
}
