package deterministicio

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

const cacheFixtureInventory = "sha256:dcb363f027b38ced3758987af61e21829381c3d0e4b706144859a00e750d53a7"

func cacheFixtureDefinition(t *testing.T, before func(string), primary error) adapterDefinition {
	t.Helper()
	identity := gomadversion.AdapterIdentity{Module: "example.com/adapter", Version: "v1.2.3", Sum: "h1:adapter"}
	return adapterDefinition{identity: identity, implementation: adapterImplementation{module: identity.Module,
		prepare: func(_, work string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
			if before != nil {
				before(work)
			}
			if primary != nil {
				return adapterPreparation{}, primary
			}
			replacement := filepath.Join(work, "adapter")
			if err := os.Mkdir(replacement, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(replacement, "adapter.go"), []byte("package adapter\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			inventory, err := digestAdapterSourceInventory(replacement)
			if err != nil || inventory != cacheFixtureInventory {
				t.Fatalf("fixture inventory = %q, %v", inventory, err)
			}
			return adapterPreparation{replacement: replacement, evidence: BuildAdapter{
				Module: identity.Module, Version: identity.Version, Sum: identity.Sum,
				ReplacementRoot: replacement, Replacement: filepath.Join(replacement, "adapter.go"),
				ReplacementSourceInventorySHA256: inventory,
			}}, nil
		}}}
}

func denyCacheScratchCleanup(t *testing.T, work string) {
	t.Helper()
	denied := filepath.Join(work, "denied")
	if err := os.Mkdir(denied, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(denied, "leaf"), []byte("scratch"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(denied, 0o700); err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Error(err)
		}
		if err := os.RemoveAll(work); err != nil {
			t.Error(err)
		}
	})
	if err := os.Chmod(denied, 0); err != nil {
		t.Fatal(err)
	}
	_, err := os.ReadDir(denied)
	if err == nil && os.Geteuid() == 0 {
		t.Skip("UID 0 bypasses mode-000 scratch permissions")
	}
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("scratch permission probe = %v (UID %d)", err, os.Geteuid())
	}
	t.Logf("UID %d actual scratch denial: %v", os.Geteuid(), err)
}

func assertCacheFixture(t *testing.T, cache string) string {
	t.Helper()
	published := filepath.Join(cache, "adapter@v1.2.3-dcb363f027b38ced")
	for _, item := range []struct {
		path string
		mode fs.FileMode
	}{{published, 0o700}, {filepath.Join(published, "adapter.go"), 0o600}} {
		info, err := os.Stat(item.path)
		if err != nil || info.Mode().Perm() != item.mode {
			t.Fatalf("cache mode %s = %v, %v", item.path, info, err)
		}
	}
	contents, err := os.ReadFile(filepath.Join(published, "adapter.go"))
	if err != nil || string(contents) != "package adapter\n" {
		t.Fatalf("published bytes = %q, %v", contents, err)
	}
	return published
}

func TestAdapterCacheCleanupOwnerErrors(t *testing.T) {
	for _, fault := range []bool{false, true} {
		for _, failPrepare := range []bool{false, true} {
			t.Run(fmtCacheCase(fault, failPrepare), func(t *testing.T) {
				cache := t.TempDir()
				var primary error
				if failPrepare {
					_, primary = os.ReadFile(filepath.Join(t.TempDir(), "missing"))
					if !errors.Is(primary, fs.ErrNotExist) {
						t.Fatal(primary)
					}
				}
				var work string
				definition := cacheFixtureDefinition(t, func(path string) {
					work = path
					if fault {
						denyCacheScratchCleanup(t, path)
					}
				}, primary)
				prepared, err := prepareCachedAdapter(definition, "unused", cache)
				if !fault {
					if err != primary {
						t.Fatalf("error = %v, want exact primary %v", err, primary)
					}
					if _, statErr := os.Stat(work); !errors.Is(statErr, fs.ErrNotExist) {
						t.Fatalf("scratch retained: %v", statErr)
					}
				} else {
					var cleanup *os.PathError
					if primary != nil {
						joined, ok := err.(interface{ Unwrap() []error })
						if !ok || len(joined.Unwrap()) != 2 || joined.Unwrap()[0] != primary {
							t.Fatalf("error = %v, want primary-first actual cleanup join", err)
						}
						if !errors.As(joined.Unwrap()[1], &cleanup) {
							t.Fatalf("cleanup = %v", joined.Unwrap()[1])
						}
					} else {
						var ok bool
						cleanup, ok = err.(*os.PathError)
						if !ok {
							t.Fatalf("error = %v, want direct sole OS cleanup error", err)
						}
					}
					if !errors.Is(cleanup, fs.ErrPermission) || !strings.HasPrefix(cleanup.Path, filepath.Join(work, "denied")) || IsInvalidBuildAdapterConfiguration(err) {
						t.Fatalf("cleanup cause/classification = %v", err)
					}
					t.Logf("actual RemoveAll failure: %v", cleanup)
				}
				if primary == nil {
					published := assertCacheFixture(t, cache)
					if prepared.replacement != published || prepared.evidence.ReplacementRoot != published || prepared.evidence.Replacement != filepath.Join(published, "adapter.go") {
						t.Fatalf("prepared result lost relocation: %#v", prepared)
					}
				}
			})
		}
	}
}

func fmtCacheCase(fault, primary bool) string {
	if primary {
		if fault {
			return "primary-and-cleanup"
		}
		return "exact-primary"
	}
	if fault {
		return "sole-cleanup"
	}
	return "healthy"
}

func TestAdapterCacheCleanupRegistryPublicationReuseRetry(t *testing.T) {
	for name, faults := range map[string][]bool{"healthy": {false, false}, "publication-retry": {true, false}, "reuse-retry": {false, true, false}} {
		t.Run(name, func(t *testing.T) {
			working := t.TempDir()
			module := "module example.com/target\n\nrequire example.com/adapter v1.2.3\n"
			sums := "example.com/adapter v1.2.3 h1:adapter\n"
			for name, contents := range map[string]string{"go.mod": module, "go.sum": sums} {
				if err := os.WriteFile(filepath.Join(working, name), []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			toolchain := t.TempDir()
			for _, fault := range faults {
				preparation := t.TempDir()
				var work string
				definition := cacheFixtureDefinition(t, func(path string) {
					work = path
					if fault {
						denyCacheScratchCleanup(t, path)
					}
				}, nil)
				registry := adapterRegistry{definitions: []adapterDefinition{definition}}
				prepared, evidence, err := registry.prepare(target.Spec{WorkingDir: working, PreparationRoot: preparation, ToolchainRoot: toolchain}, "unused")
				published := assertCacheFixture(t, filepath.Join(toolchain, "adapters"))
				if fault {
					if !errors.Is(err, fs.ErrPermission) || IsInvalidBuildAdapterConfiguration(err) || !reflect.DeepEqual(prepared, target.Spec{}) || evidence != nil {
						t.Fatalf("cleanup failure published target/evidence: %#v, %#v, %v", prepared, evidence, err)
					}
					for _, name := range []string{"gomad.mod", "gomad.sum"} {
						if _, statErr := os.Stat(filepath.Join(preparation, ".io-adapter", name)); !errors.Is(statErr, fs.ErrNotExist) {
							t.Fatalf("failed preparation published %s: %v", name, statErr)
						}
					}
					if chmodErr := os.Chmod(filepath.Join(work, "denied"), 0o700); chmodErr != nil {
						t.Fatal(chmodErr)
					}
					if removeErr := os.RemoveAll(work); removeErr != nil {
						t.Fatal(removeErr)
					}
				} else {
					if err != nil || len(evidence) != 1 || evidence[0].ReplacementRoot != published || evidence[0].Replacement != filepath.Join(published, "adapter.go") {
						t.Fatalf("healthy reuse = %#v, %v", evidence, err)
					}
					for name, want := range map[string]string{"gomad.mod": module + "\nreplace example.com/adapter v1.2.3 => " + published + "\n", "gomad.sum": sums} {
						got, readErr := os.ReadFile(filepath.Join(preparation, ".io-adapter", name))
						if readErr != nil || string(got) != want {
							t.Fatalf("%s = %q, %v; want %q", name, got, readErr, want)
						}
					}
				}
				entries, readErr := os.ReadDir(filepath.Join(toolchain, "adapters"))
				if readErr != nil || len(entries) != 1 || entries[0].Name() != "adapter@v1.2.3-dcb363f027b38ced" {
					t.Fatalf("cache entries = %v, %v", entries, readErr)
				}
				for name, want := range map[string]string{"go.mod": module, "go.sum": sums} {
					got, readErr := os.ReadFile(filepath.Join(working, name))
					if readErr != nil || string(got) != want {
						t.Fatalf("source %s changed: %q, %v", name, got, readErr)
					}
				}
			}
		})
	}
}

func TestAdapterCacheCleanupValidationControls(t *testing.T) {
	for _, kind := range []string{"root", "inventory", "corrupt-cache", "non-directory"} {
		t.Run(kind, func(t *testing.T) {
			cache := t.TempDir()
			definition := cacheFixtureDefinition(t, nil, nil)
			var preserved string
			if kind == "corrupt-cache" || kind == "non-directory" {
				prepared, err := prepareCachedAdapter(definition, "unused", cache)
				if err != nil {
					t.Fatal(err)
				}
				preserved = filepath.Join(prepared.replacement, "adapter.go")
				if kind == "non-directory" {
					if err := os.RemoveAll(prepared.replacement); err != nil {
						t.Fatal(err)
					}
					preserved = prepared.replacement
				}
				if err := os.WriteFile(preserved, []byte("corrupt"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			prepare := definition.implementation.prepare
			var work string
			definition.implementation.prepare = func(module, root string, identity gomadversion.AdapterIdentity) (adapterPreparation, error) {
				work = root
				prepared, err := prepare(module, root, identity)
				switch kind {
				case "root":
					prepared.evidence.ReplacementRoot = root
				case "inventory":
					prepared.evidence.ReplacementSourceInventorySHA256 = "sha256:short"
				}
				return prepared, err
			}
			_, err := prepareCachedAdapter(definition, "unused", cache)
			want := map[string]string{"root": "deterministic I/O adapter replacement root mismatch", "inventory": "deterministic I/O adapter replacement inventory is incomplete", "corrupt-cache": "published deterministic I/O adapter replacement ", "non-directory": "verify published deterministic I/O adapter replacement: "}[kind]
			if err == nil || !strings.HasPrefix(err.Error(), want) || IsInvalidBuildAdapterConfiguration(err) {
				t.Fatalf("validation error = %v, want %q", err, want)
			}
			if _, statErr := os.Stat(work); !errors.Is(statErr, fs.ErrNotExist) {
				t.Fatalf("scratch retained: %v", statErr)
			}
			if preserved != "" {
				contents, readErr := os.ReadFile(preserved)
				if readErr != nil || string(contents) != "corrupt" {
					t.Fatalf("existing cache changed: %q, %v", contents, readErr)
				}
			}
		})
	}
}

func TestAdapterCacheCleanupRegistryPrimary(t *testing.T) {
	for _, fault := range []bool{false, true} {
		t.Run(fmtCacheCase(fault, true), func(t *testing.T) {
			working := t.TempDir()
			module := "module example.com/target\n\nrequire example.com/adapter v1.2.3\n"
			sums := "example.com/adapter v1.2.3 h1:adapter\n"
			for name, contents := range map[string]string{"go.mod": module, "go.sum": sums} {
				if err := os.WriteFile(filepath.Join(working, name), []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			_, primary := os.ReadFile(filepath.Join(working, "missing"))
			if !errors.Is(primary, fs.ErrNotExist) {
				t.Fatal(primary)
			}
			definition := cacheFixtureDefinition(t, func(work string) {
				if fault {
					denyCacheScratchCleanup(t, work)
				}
			}, primary)
			preparation, toolchain := t.TempDir(), t.TempDir()
			registry := adapterRegistry{definitions: []adapterDefinition{definition}}
			prepared, evidence, err := registry.prepare(target.Spec{WorkingDir: working, PreparationRoot: preparation, ToolchainRoot: toolchain}, "unused")
			if !errors.Is(err, primary) || !reflect.DeepEqual(prepared, target.Spec{}) || evidence != nil || IsInvalidBuildAdapterConfiguration(err) {
				t.Fatalf("primary failure = %#v, %#v, %v", prepared, evidence, err)
			}
			if !fault && err != primary {
				t.Fatalf("primary identity changed: %v", err)
			}
			if fault && !errors.Is(err, fs.ErrPermission) {
				t.Fatalf("cleanup failure lost: %v", err)
			}
			for _, name := range []string{"gomad.mod", "gomad.sum"} {
				if _, statErr := os.Stat(filepath.Join(preparation, ".io-adapter", name)); !errors.Is(statErr, fs.ErrNotExist) {
					t.Fatalf("primary failure published %s: %v", name, statErr)
				}
			}
			entries, readErr := os.ReadDir(filepath.Join(toolchain, "adapters"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			for _, entry := range entries {
				if !fault || !strings.HasPrefix(entry.Name(), ".prepare-") {
					t.Fatalf("primary failure published %s", entry.Name())
				}
			}
			for name, want := range map[string]string{"go.mod": module, "go.sum": sums} {
				contents, readErr := os.ReadFile(filepath.Join(working, name))
				if readErr != nil || string(contents) != want {
					t.Fatalf("source %s changed: %q, %v", name, contents, readErr)
				}
			}
		})
	}
}
