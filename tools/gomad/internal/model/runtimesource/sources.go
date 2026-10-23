// Package runtimesemantics owns the sources compiled verbatim into each runtime overlay.
package runtimesemantics

import (
	"bytes"
	"fmt"
	"path/filepath"

	hostfs "go.temporal.io/server/tools/gomad/internal/support/fs"
)

// Read returns package-runtime sources keyed by their overlay filename.
func Read(root string) (map[string][]byte, error) {
	sources := make(map[string][]byte, 2)
	for _, name := range []string{"gomad_semantics.go", "gomad_choicewire_generated.go"} {
		data, err := hostfs.ReadBounded(filepath.Join(root, name+".txt"), 4<<20)
		if err != nil {
			return nil, fmt.Errorf("runtime semantics %s: %w", name, err)
		}
		sources[name] = data
	}
	return sources, nil
}

// Verify rejects an overlay copy that differs from its identity-bound sources.
func Verify(sources map[string][]byte, runtimeRoot string) error {
	for name, expected := range sources {
		data, err := hostfs.ReadBounded(filepath.Join(runtimeRoot, name), uint64(len(expected)))
		if err != nil {
			return fmt.Errorf("runtime semantics copy %s: %w", name, err)
		}
		if !bytes.Equal(data, expected) {
			return fmt.Errorf("runtime semantics copy %s bytes changed", name)
		}
	}
	return nil
}
