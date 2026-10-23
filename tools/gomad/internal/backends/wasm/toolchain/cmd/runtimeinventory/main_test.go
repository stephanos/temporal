package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPatchInventoryDerivesEveryRewriteAndRejectsUnknownExpressions(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "src/runtime"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "src/runtime/rand.go"), []byte("package runtime\nfunc rand() { old() }\n"), 0600))
	source := []byte("package toolchain\nconst before = `old()`\nfunc stockRewrites() map[string][]rewrite { return map[string][]rewrite{\"rand.go\": {{before, `gomadRuntimeRand()`, 1}, {`package runtime`, `package runtime // reviewed`, 1}}} }\n")
	patches, err := overlayPatches(source)
	require.NoError(t, err)
	require.Len(t, patches, 2)
	draws, others, err := patchReferences(root, patches, map[string]string{})
	require.NoError(t, err)
	require.Equal(t, map[string]int{"wasip1/wasm runtime/rand.go rand gomadRuntimeRand patch": 1}, draws)
	require.Equal(t, map[string]int{"wasip1/wasm runtime/rand.go package patch": 1}, others)
	_, err = overlayPatches([]byte("package toolchain\nfunc stockRewrites() map[string][]rewrite { return map[string][]rewrite{\"rand.go\": {{unknown(), `after`, 1}}} }"))
	require.ErrorContains(t, err, "operand")
	_, _, err = patchReferences(root, []overlayPatch{{file: "rand.go", from: "absent", to: "cheaprand()", count: 1}}, map[string]string{})
	require.ErrorContains(t, err, "anchor")
}

func TestClockInventoryHonorsBuildSelectionCommentsAndNewOverlayFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "src/runtime"), 0700))
	for file, source := range map[string]string{
		"clock.go":       "package runtime\n// cputicks ignored\nfunc clock() { cputicks(); cputicks() }\n",
		"clock_linux.go": "package runtime\nfunc host() { walltime() }\n",
		"clock_test.go":  "package runtime\nfunc test() { walltime() }\n",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "src/runtime", file), []byte(source), 0600))
	}
	added := filepath.Join(root, "added.go")
	require.NoError(t, os.WriteFile(added, []byte("package runtime\nfunc added() { nanotime() }\n"), 0600))
	got, err := clockReferences(root, map[string]string{filepath.Join(root, "src/runtime/gomad_wasm.go"): added}, map[string]string{})
	require.NoError(t, err)
	require.Equal(t, map[string]int{"wasip1/wasm runtime/clock.go cputicks": 2, "wasip1/wasm runtime/gomad_wasm.go nanotime": 1}, got)
}
