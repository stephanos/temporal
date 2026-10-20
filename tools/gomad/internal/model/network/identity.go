package network

import (
	"crypto/sha256"
	"embed"
	"fmt"
)

//go:embed simulation.go adapter.go identity.go stream_status_generated.go
var implementationSources embed.FS

// ImplementationSHA256 binds the executable simulation, its wait adapter and the
// generated stream-status codec.
func ImplementationSHA256() string {
	var inventory []byte
	for _, name := range []string{"adapter.go", "identity.go", "simulation.go", "stream_status_generated.go"} {
		data, err := implementationSources.ReadFile(name)
		if err != nil {
			return ""
		}
		inventory = fmt.Appendf(inventory, "%s:%x\n", name, sha256.Sum256(data))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(inventory))
}
