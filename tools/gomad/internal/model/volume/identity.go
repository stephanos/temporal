package volume

import (
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"fmt"
	"io/fs"
	"slices"
)

//go:embed conformance_state.go fs.go handles.go local_handles.go volume.go types.go runtime.go metadata.go clock.go captured.go recorder.go identity.go internal/hashwire/hash.go internal/hashwire/wire_generated.go
var implementationSources embed.FS

var implementationPaths = []string{
	"captured.go",
	"clock.go",
	"conformance_state.go",
	"fs.go",
	"handles.go",
	"identity.go",
	"internal/hashwire/hash.go",
	"internal/hashwire/wire_generated.go",
	"local_handles.go",
	"metadata.go",
	"recorder.go",
	"runtime.go",
	"types.go",
	"volume.go",
}

// ImplementationSHA256 binds the compiled filesystem model, host adapters, clock
// shim, and complete private hash codec.
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
