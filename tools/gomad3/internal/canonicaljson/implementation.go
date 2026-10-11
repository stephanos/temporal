package canonicaljson

import (
	"crypto/sha256"
	"embed"
	"fmt"
)

//go:embed canonical.go implementation.go streaming.go
var implementationSources embed.FS

// ImplementationSHA256 binds both canonical encoders and their source inventory.
func ImplementationSHA256() string {
	data := []byte("gomad3-canonicaljson-implementation/v1\n")
	for _, name := range []string{"canonical.go", "implementation.go", "streaming.go"} {
		source, err := implementationSources.ReadFile(name)
		if err != nil {
			return ""
		}
		data = fmt.Appendf(data, "%s:%x\n", name, sha256.Sum256(source))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
