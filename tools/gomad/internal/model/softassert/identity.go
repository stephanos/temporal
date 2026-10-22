package softassert

import (
	"crypto/sha256"
	"embed"
	"fmt"

	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
)

//go:embed identity.go softassert.go
var sources embed.FS

func ImplementationSHA256() string {
	inventory := []byte("gomad-soft-assertion/v1\n")
	for _, name := range []string{"identity.go", "softassert.go"} {
		data, err := sources.ReadFile(name)
		if err != nil {
			return ""
		}
		inventory = fmt.Appendf(inventory, "%s:%x\n", name, sha256.Sum256(data))
	}
	inventory = fmt.Appendf(inventory, "strict-json:%s\n", strictjson.ImplementationSHA256())
	return fmt.Sprintf("sha256:%x", sha256.Sum256(inventory))
}
