package wasi

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
)

//go:embed environment.go namespace.go clock.go protocol.go process.go runtime.go
var implementationSources embed.FS

func ImplementationSHA256() string {
	return implementationSHA256(implementationSources)
}

func implementationSHA256(sources fs.ReadFileFS) string {
	var inventory []byte
	for _, name := range []string{"clock.go", "environment.go", "namespace.go", "process.go", "protocol.go", "runtime.go"} {
		data, err := sources.ReadFile(name)
		if err != nil {
			return ""
		}
		inventory = fmt.Appendf(inventory, "%s:%x\n", name, sha256.Sum256(data))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(inventory))
}
