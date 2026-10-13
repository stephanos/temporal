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
	appendField := func(value string) {
		input = binary.BigEndian.AppendUint64(input, uint64(len(value)))
		input = append(input, value...)
	}
	appendField(dependencyPath)
	appendField(version)
	appendField(sum)
	for _, path := range implementationPaths {
		content, err := sources.ReadFile(path)
		if err != nil {
			return "", err
		}
		appendField(path)
		input = binary.BigEndian.AppendUint64(input, uint64(len(content)))
		input = append(input, content...)
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(input)), nil
}
