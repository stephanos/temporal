package volume

import (
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"fmt"
	"io/fs"
	"slices"
)

//go:embed conformance_state.go fs_generated.go handles_generated.go local_handles_generated.go volume_generated.go provenance_generated.go types_generated.go runtime.go metadata.go clock.go captured.go recorder.go identity.go internal/hashwire/hash_generated.go internal/hashwire/wire_generated.go
var implementationSources embed.FS

var implementationPaths = []string{
	"captured.go",
	"clock.go",
	"conformance_state.go",
	"fs_generated.go",
	"handles_generated.go",
	"identity.go",
	"internal/hashwire/hash_generated.go",
	"internal/hashwire/wire_generated.go",
	"local_handles_generated.go",
	"metadata.go",
	"provenance_generated.go",
	"recorder.go",
	"runtime.go",
	"types_generated.go",
	"volume_generated.go",
}

// ImplementationSHA256 binds the compiled projection, host adapters, clock shim,
// and complete private hash codec independently of their original native inputs.
func ImplementationSHA256() string {
	identity, err := implementationSHA256(implementationSources)
	if err != nil {
		// Every required file is embedded above; a missing entry is invalid identity.
		return ""
	}
	return identity
}

func implementationSHA256(sources fs.ReadFileFS) (string, error) {
	paths := slices.Clone(implementationPaths)
	slices.Sort(paths)
	input := []byte("gomad.host-volume-implementation/v1\x00")
	for _, path := range paths {
		content, err := sources.ReadFile(path)
		if err != nil {
			return "", err
		}
		input = binary.BigEndian.AppendUint64(input, uint64(len(path)))
		input = append(input, path...)
		input = binary.BigEndian.AppendUint64(input, uint64(len(content)))
		input = append(input, content...)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(input)), nil
}
