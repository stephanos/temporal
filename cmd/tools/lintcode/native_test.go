package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExplicitNativeLintRequiresSelection(t *testing.T) {
	err := lint(t.Context(), "HEAD", nativeModule, "test_dep")
	require.ErrorContains(t, err, "native lint module requires gomad_native selection")
}

func TestNestedNativeLintSelection(t *testing.T) {
	for _, selected := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "selected"}[selected], func(t *testing.T) {
			repo := newLintRepo(t)
			repo.write(nativeModule+"/go.mod", "module example.invalid/native\n\ngo 1.27.0\n")
			inventoryPath := "tools/gomad/internal/archcheck/inventory.go"
			inventory, err := os.ReadFile(filepath.Join(repo.root, inventoryPath))
			require.NoError(t, err)
			repo.write(inventoryPath, strings.Replace(string(inventory), "map[string]bool{", `map[string]bool{"internal/backends/native/go.mod":true,`, 1))
			args := []string{"lint-code-gomad"}
			if selected {
				args = append(args, "ALL_TEST_TAGS=test_dep,gomad_native")
			}
			repo.make(args...)
			calls := repo.calls()
			want := []string{gomadModule, gomadModule}
			if selected {
				want = append(want, nativeModule, nativeModule)
			}
			var owners []string
			for _, call := range calls {
				owners = append(owners, call.Dir)
				if selected {
					require.Contains(t, strings.Join(call.Args, " "), "gomad_native")
				} else {
					require.NotContains(t, strings.Join(call.Args, " "), "internal/backends/native")
				}
			}
			require.Equal(t, want, owners)
		})
	}
}
