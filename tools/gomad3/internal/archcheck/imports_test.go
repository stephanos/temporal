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
		{"shared-to-native", core + "/internal/model/world", core + "/internal/backends/native/build", true},
		{"shared-to-wasm", core + "/internal/model/world", core + "/internal/backends/wasm", true},
		{"native-to-shared", core + "/internal/backends/native/build", core + "/internal/model/world", false},
		{"native-to-native", core + "/internal/backends/native/build", core + "/internal/backends/native/toolchain", false},
		{"native-to-wasm", core + "/internal/backends/native/build", core + "/internal/backends/wasm", true},
		{"wasm-to-native", core + "/internal/backends/wasm", core + "/internal/backends/native/build", true},
		{"shared-to-cmd-named-backend", core + "/internal/model/world", core + "/internal/backends/cmd", true},
		{"native-to-cmd-named-backend", core + "/internal/backends/native", core + "/internal/backends/cmd", true},
		{"cmd-named-backend-to-native", core + "/internal/backends/cmd", core + "/internal/backends/native", true},
		{"shared-to-cmd", core + "/internal/model/world", core + "/cmd/gomad/api", true},
		{"native-to-cmd", core + "/internal/backends/native", core + "/cmd/gomad/api", true},
		{"wasm-to-cmd", core + "/internal/backends/wasm", core + "/cmd/gomad/api", true},
		{"cmd-to-native", core + "/cmd/gomad", core + "/internal/backends/native/build", false},
		{"cmd-to-wasm", core + "/cmd/gomad", core + "/internal/backends/wasm", false},
		{"module-prefix-is-external", core + "/internal/model/world", core + "extra/backends/native", false},
		{"external-importer", core + "extra/world", core + "/internal/backends/native", false},
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
