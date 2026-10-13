package strictjson

import (
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"fmt"
	"io/fs"
)

const (
	dependencyPath    = "github.com/go-json-experiment/json"
	dependencyVersion = "v0.0.0-20260820222146-c27c302e5fc3"
	dependencySum     = "h1:UADEEmDKgfXbtnGJZ97beY5XLo9ZechG1nlU4KnRrkE="
)

//go:embed decode.go identity.go
var implementationSources embed.FS

var implementationPaths = []string{"decode.go", "identity.go"}

// ImplementationSHA256 binds the shared decoder and its pinned library input.
func ImplementationSHA256() string {
	identity, err := implementationSHA256(implementationSources, dependencyVersion, dependencySum)
	if err != nil {
		return ""
	}
	return identity
}

func implementationSHA256(sources fs.ReadFileFS, version, sum string) (string, error) {
	input := []byte("gomad3.strict-json-implementation/v1\x00")
	input = appendIdentityField(input, dependencyPath)
	input = appendIdentityField(input, version)
	input = appendIdentityField(input, sum)
	for _, path := range implementationPaths {
		//gomad:dynamic reviewed 2026-10-13 unexported callers provide embedded or test in-memory filesystems
		content, err := sources.ReadFile(path)
		if err != nil {
			return "", err
		}
		input = appendIdentityField(input, path)
		input = binary.BigEndian.AppendUint64(input, uint64(len(content)))
		input = append(input, content...)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(input)), nil
}

func appendIdentityField(input []byte, value string) []byte {
	input = binary.BigEndian.AppendUint64(input, uint64(len(value)))
	return append(input, value...)
}
