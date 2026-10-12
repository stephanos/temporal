package architecture

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestMergedLayoutOwnersPreserveBoundaries(t *testing.T) {
	const module = "go.temporal.io/server/tools/gomad3"
	for path, want := range map[string]string{
		"backends/native/build/internal/capabilitypolicy": "target",
		"backends/native/toolchain/version":               "toolchain",
		"backends/native/upgrade/pinimpact":               "upgrade",
		"backends/native/transcript":                      "deterministicio",
		"backends/native/conformance":                     "developer",
		"backends/native/generation/protocol":             "developer",
		"backends/native/qualification/set":               "qualification",
		"backends/native/preparation":                     "preparation",
		"captured":                                        "captured", "volume/internal/hashwire": "volume", "internal/iowire": "iowire",
		"backends/wasm/wasi": "wasm", "backends/wasm/internal/stockgo": "wasm",
		"backends/wasm/cmd/gomad-wasm":   "cli",
		"backends/native/toolchainextra": "", "backends/unknown": "",
	} {
		if got := Owner(module, module+"/"+path); got != want {
			t.Errorf("Owner(%q) = %q, want %q", path, got, want)
		}
	}
	for _, tc := range []struct {
		from, to string
		allowed  bool
	}{
		{"backends/wasm/toolchain", "runtimesemantics", true},
		{"backends/wasm/wasi", "captured", true},
		{"backends/wasm/wasi", "backends/native/transcript", false},
		{"backends/wasm/wasi", "qualification/set", false},
		{"backends/wasm/cmd/gomad-wasm", "cmd/gomad/api", true},
		{"backends/wasm/cmd/gomad-wasm", "hostexec", true},
		{"cmd/gomad", "hostexec", false},
		{"captured", "internal/iowire", true},
		{"volume", "internal/iowire", true},
		{"volume", "backends/native/toolchain", false},
		{"backends/native/build", "backends/native/toolchain/installation", true},
		{"backends/native/build", "backends/native/toolchain", false},
		{"backends/native/upgrade/pinimpact", "backends/native/transcript", true},
		{"backends/native/upgrade", "backends/native/transcript", false},
		{"internal/gomadtool/architecture", "internal/canonicaljson", true},
		{"internal/gomadtool/other", "internal/canonicaljson", false},
	} {
		from, to := module+"/"+tc.from, module+"/"+tc.to
		if got := OwnerMayImport(module, Owner(module, from), Owner(module, to), from, to); got != tc.allowed {
			t.Errorf("%s -> %s = %v, want %v", tc.from, tc.to, got, tc.allowed)
		}
	}
}

func TestMergedLayoutDiscoveryClassifiesOnlyFixtureAndCargoRoots(t *testing.T) {
	root := writeArchitectureSources(t, map[string]string{
		"backends/wasm/testdata/guest/source.go":                 "package fixture\n",
		"backends/wasm/wasi/testdata/guest/source.go":            "package fixture\n",
		"backends/wasm/toolchain/testdata/guest/source.go":       "package fixture\n",
		"backends/wasm/simulation/testdata/guest/source.go":      "package fixture\n",
		"backends/wasm/cmd/gomad-wasm/testdata/guest/source.go":  "package fixture\n",
		"backends/wasm/qualification/testdata/fixture/source.go": "package fixture\n",
		"backends/wasm/qualification/testdata/fixture/go.mod":    "module example.invalid/guest\n\ngo 1.27.1\n",
		"backends/wasm/wasmhost/target/.generated/source.go":     "package generated\n",
		"backends/wasm/wasmhost/target/cache/source.go":          "package generated\nimport _ \"example.invalid/generated/missing\"\n",
		"backends/wasm/wasmhost/target-extra/.hidden/source.go":  "package hidden\n",
		"backends/wasm/wasmhost/src/target/.hidden/source.go":    "package hidden\n",
	})
	inventory, findings, err := Discover(root, "go", Platform{"linux", "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(inventory.Sources, "backends/wasm/wasmhost/target/.generated/source.go") || slices.ContainsFunc(inventory.Packages, func(p Package) bool {
		return Within(p.ImportPath, "example.invalid/architecture/backends/wasm/wasmhost/target")
	}) {
		t.Fatal("Cargo generated cache entered the source inventory")
	}
	for _, finding := range findings {
		if finding.Category != "stale-exclusion" && finding.Path != "backends/wasm/wasmhost/target-extra/.hidden/source.go" && finding.Path != "backends/wasm/wasmhost/src/target/.hidden/source.go" {
			t.Errorf("explicit WASM fixture classification failed: %v", finding)
		}
	}
	for _, name := range []string{"backends/wasm/wasmhost/target-extra/.hidden/source.go", "backends/wasm/wasmhost/src/target/.hidden/source.go"} {
		if !slices.ContainsFunc(findings, func(f Finding) bool { return f.Category == "uncovered-source" && filepath.ToSlash(f.Path) == name }) {
			t.Errorf("Cargo-neighbor source escaped discovery: %s", name)
		}
	}
}

func TestMovedNativeFacadeStillRequiresReviewedEdges(t *testing.T) {
	const module = "go.temporal.io/server/tools/gomad3"
	imports := []string{
		module + "/backends/native/build/internal/build", module + "/backends/native/build/internal/provenance",
		module + "/backends/native/toolchain/installation", module + "/internal/sourceinventory",
	}
	findings := PackageEdges(module, []Package{{ImportPath: module + "/backends/native/build", Imports: imports}})
	if !slices.ContainsFunc(findings, func(f Finding) bool {
		return f.Category == "required-edge" && f.Path == module+"/backends/native/build" && f.Detail == module+"/backends/native/build/internal/capabilityreview"
	}) {
		t.Fatalf("moved native facade lost its missing-review edge guard: %v", findings)
	}
	shared := PackageEdges(module, []Package{{ImportPath: module + "/target", Imports: []string{module + "/record", module + "/internal/hostfs"}}})
	if len(shared) != 0 {
		t.Fatalf("shared target types retained native facade dependencies: %v", shared)
	}
}
