package archcheck

import (
	"strings"
	"testing"
)

func TestImportDirection(t *testing.T) {
	const core = "example.invalid/gomad"
	for _, test := range []struct {
		name, from, to string
		forbidden      bool
	}{
		{"shared-to-shared", core + "/internal/model/world", core + "/internal/evidence/record", false},
		{"shared-to-legacy", core + "/internal/model/world", core + "/internal/backends/legacy/build", true},
		{"shared-to-wasm", core + "/internal/model/world", core + "/internal/backends/wasm", true},
		{"legacy-to-shared", core + "/internal/backends/legacy/build", core + "/internal/model/world", false},
		{"legacy-to-legacy", core + "/internal/backends/legacy/build", core + "/internal/backends/legacy/toolchain", false},
		{"legacy-to-wasm", core + "/internal/backends/legacy/build", core + "/internal/backends/wasm", true},
		{"wasm-to-legacy", core + "/internal/backends/wasm", core + "/internal/backends/legacy/build", true},
		{"shared-to-cmd-named-backend", core + "/internal/model/world", core + "/internal/backends/cmd", true},
		{"legacy-to-cmd-named-backend", core + "/internal/backends/legacy", core + "/internal/backends/cmd", true},
		{"cmd-named-backend-to-legacy", core + "/internal/backends/cmd", core + "/internal/backends/legacy", true},
		{"shared-to-cmd", core + "/internal/model/world", core + "/cmd/gomad/api", true},
		{"legacy-to-cmd", core + "/internal/backends/legacy", core + "/cmd/gomad/api", true},
		{"wasm-to-cmd", core + "/internal/backends/wasm", core + "/cmd/gomad/api", true},
		{"cmd-to-legacy", core + "/cmd/gomad", core + "/internal/backends/legacy/build", false},
		{"cmd-to-wasm", core + "/cmd/gomad", core + "/internal/backends/wasm", false},
		{"module-prefix-is-external", core + "/internal/model/world", core + "extra/backends/legacy", false},
		{"external-importer", core + "extra/world", core + "/internal/backends/legacy", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			findings := Config{Module: core}.CheckImports([]Package{{test.from, []string{test.to}}})
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
