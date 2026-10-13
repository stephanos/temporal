package simulation

import (
	"crypto/sha256"
	"embed"
	"fmt"

	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/internal/model/clock"
	gomad3sim "go.temporal.io/server/tools/gomad3/internal/model/cluster"
	"go.temporal.io/server/tools/gomad3/internal/model/world"
)

//go:embed implementation.go oracles.go provider.go replay.go retained.go soft_assertion.go
var implementationSources embed.FS

func ImplementationSHA256() string {
	data := []byte("gomad3-wasm-simulation-implementation/v1\n")
	for _, name := range []string{"implementation.go", "oracles.go", "provider.go", "replay.go", "retained.go", "soft_assertion.go"} {
		source, err := implementationSources.ReadFile(name)
		if err != nil {
			return ""
		}
		data = fmt.Appendf(data, "%s:%x\n", name, sha256.Sum256(source))
	}
	data = fmt.Appendf(data, "wasi:%s\nclock:%s\nworld:%s\ncluster:%s\n", wasi.ImplementationSHA256(), clock.BrokerImplementationSHA256(), world.ImplementationSHA256(), gomad3sim.ImplementationSHA256())
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
