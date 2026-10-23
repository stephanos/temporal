package wasi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelperCorpusRejectsLLVMProfileStderrOnSuccessfulExit(t *testing.T) {
	for _, test := range []struct {
		name, diagnostic string
		accepted         bool
	}{
		{"empty", "", true},
		{"guest-profile-warning", "gomad-wasm-guest-profile warning: output unavailable", true},
		{"llvm-profile-error", "LLVM Profile Error: Failed to write file", false},
		{"llvm-profile-warning", "LLVM Profile Warning: Unable to merge profile data", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := filepath.Join(t.TempDir(), "helper")
			script := "#!/bin/sh\nwhile IFS= read -r line; do :; done\nprintf '%s' '" + test.diagnostic + "' >&2\nexit 0\n"
			if err := os.WriteFile(helper, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			run, err := runHelperCorpusCase(helper, map[string]any{
				"guest":       map[string]any{"wat": "(module)"},
				"request":     map[string]any{},
				"close_input": true,
			})
			if test.accepted {
				if err != nil {
					t.Fatal(err)
				}
				if run.exitCode != 0 || run.stderr != test.diagnostic {
					t.Fatalf("captured helper result=%+v", run)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "LLVM profile diagnostic") {
				t.Fatalf("successful helper profile diagnostic was accepted: run=%+v err=%v", run, err)
			}
		})
	}
}
