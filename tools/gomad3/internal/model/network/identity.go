package network

import (
	"crypto/sha256"
	"embed"
	"fmt"
)

//go:embed generated.go adapter.go identity.go generate/main.go
var implementationSources embed.FS

// ImplementationSHA256 binds the executable projection and its wait adapter,
// including the generator's admitted transformations of the native owner.
func ImplementationSHA256() string {
	var inventory []byte
	for _, name := range []string{"adapter.go", "generate/main.go", "generated.go", "identity.go"} {
		data, err := implementationSources.ReadFile(name)
		if err != nil {
			return ""
		}
		inventory = fmt.Appendf(inventory, "%s:%x\n", name, sha256.Sum256(data))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(inventory))
}
