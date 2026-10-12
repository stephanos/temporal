package archcheck

import (
	"slices"
	"strings"
	"testing"
)

func TestImportDirection(t *testing.T) {
	const core = "example.invalid/gomad"
	const wasm = "example.invalid/wasm"
	for _, test := range []struct {
		name, from, to string
		forbidden      bool
	}{
		{"shared-to-shared", core + "/world", core + "/record", false},
		{"shared-to-legacy-native", core + "/world", core + "/toolchain", true},
		{"shared-to-native", core + "/world", core + "/backends/native/build", true},
		{"shared-to-wasm", core + "/world", core + "/backends/wasm", true},
		{"native-to-shared", core + "/backends/native/build", core + "/world", false},
		{"native-to-native", core + "/backends/native/build", core + "/backends/native/toolchain", false},
		{"native-to-wasm", core + "/backends/native/build", core + "/backends/wasm", true},
		{"wasm-to-native", core + "/backends/wasm", core + "/backends/native/build", true},
		{"shared-to-cmd-named-backend", core + "/world", core + "/backends/cmd", true},
		{"native-to-cmd-named-backend", core + "/backends/native", core + "/backends/cmd", true},
		{"cmd-named-backend-to-native", core + "/backends/cmd", core + "/backends/native", true},
		{"shared-to-cmd", core + "/world", core + "/cmd/gomad/api", true},
		{"native-to-cmd", core + "/backends/native", core + "/cmd/gomad/api", true},
		{"wasm-to-cmd", core + "/backends/wasm", core + "/cmd/gomad/api", true},
		{"wasm-launcher-to-cmd", wasm + "/cmd/gomad-wasm", core + "/cmd/gomad/api", false},
		{"cmd-to-native", core + "/cmd/gomad", core + "/backends/native/build", false},
		{"cmd-to-wasm", core + "/cmd/gomad", core + "/backends/wasm", false},
		{"module-prefix-is-external", core + "/world", core + "extra/backends/native", false},
		{"legacy-prefix-is-shared", core + "/world", core + "/toolchainextra", false},
		{"external-wasm-public-import", wasm + "/backend", core + "/world", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := Config{Module: core, WASMModule: wasm, NativePackages: []string{"toolchain"}}
			if strings.HasPrefix(test.from, wasm+"/") {
				config.WASMImports = []string{test.to}
			}
			findings := config.CheckImports([]Package{{test.from, []string{test.to}}})
			if !test.forbidden {
				if len(findings) != 0 {
					t.Fatalf("legal import rejected: %v", findings)
				}
				return
			}
			if len(findings) != 1 || !strings.Contains(findings[0], test.from) || !strings.Contains(findings[0], test.to) {
				t.Fatalf("forbidden import must name both packages: %v", findings)
			}
		})
	}
}

func TestImportAllowlists(t *testing.T) {
	const core = "example.invalid/gomad"
	const wasm = "example.invalid/wasm"
	const from = core + "/runner"
	const to = core + "/toolchain"
	for _, test := range []struct {
		name       string
		exceptions []Exception
		imports    []string
		want       string
	}{
		{"existing-edge", []Exception{{from, to, "fn-164.3"}}, []string{to}, ""},
		{"deleted-exception", nil, []string{to}, "forbidden"},
		{"stale-exception", []Exception{{from, to, "fn-164.3"}}, nil, "stale"},
		{"missing-removal-task", []Exception{{from, to, ""}}, []string{to}, "removal task"},
		{"wildcard-exception", []Exception{{from, core + "/toolchain/*", "fn-164.3"}}, []string{to}, "exact"},
		{"duplicate-exception", []Exception{{from, to, "fn-164.3"}, {from, to, "fn-164.3"}}, []string{to}, "duplicate"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := Config{Module: core, WASMModule: wasm, NativePackages: []string{"toolchain"}, Exceptions: test.exceptions}
			findings := config.CheckImports([]Package{{from, test.imports}, {from, test.imports}})
			if test.want == "" {
				if len(findings) != 0 {
					t.Fatalf("existing exception rejected: %v", findings)
				}
				return
			}
			if !slices.ContainsFunc(findings, func(finding string) bool {
				return strings.Contains(finding, test.want) && strings.Contains(finding, from) && strings.Contains(finding, to)
			}) {
				t.Fatalf("missing %s diagnostic naming exception edge: %v", test.want, findings)
			}
		})
	}
	for _, test := range []struct {
		name, imported string
		allowed        []string
		want           string
	}{
		{"wasm-public-edge", core + "/world", []string{core + "/world"}, ""},
		{"wasm-outside-allowlist", core + "/toolchain", []string{core + "/world"}, "outside"},
		{"wasm-private-edge", core + "/internal/secret", []string{core + "/internal/secret"}, "public"},
		{"wasm-stale-entry", "", []string{core + "/world"}, "stale"},
		{"wasm-wildcard-entry", core + "/world", []string{core + "/*"}, "exact"},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := Config{Module: core, WASMModule: wasm, WASMImports: test.allowed}
			var imports []string
			if test.imported != "" {
				imports = []string{test.imported}
			}
			findings := config.CheckImports([]Package{{wasm + "/backend", imports}})
			if test.want == "" {
				if len(findings) != 0 {
					t.Fatalf("public import rejected: %v", findings)
				}
			} else if !slices.ContainsFunc(findings, func(finding string) bool { return strings.Contains(finding, test.want) }) {
				t.Fatalf("missing %s diagnostic: %v", test.want, findings)
			}
		})
	}
}
