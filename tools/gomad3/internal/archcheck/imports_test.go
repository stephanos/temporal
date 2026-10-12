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
		{"shared-to-shared", core + "/world", core + "/record", false},
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
		{"cmd-to-native", core + "/cmd/gomad", core + "/backends/native/build", false},
		{"cmd-to-wasm", core + "/cmd/gomad", core + "/backends/wasm", false},
		{"module-prefix-is-external", core + "/world", core + "extra/backends/native", false},
		{"external-importer", core + "extra/world", core + "/backends/native", false},
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
