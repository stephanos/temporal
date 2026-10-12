package simulation

import (
	"crypto/sha256"
	"embed"
	"fmt"

	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/world"
	"go.temporal.io/server/tools/gomad3sim"
)

//go:embed implementation.go oracles.go provider.go replay.go retained.go
var implementationSources embed.FS

func ImplementationSHA256() string {
	data := []byte("gomad3-wasm-simulation-implementation/v1\n")
	for _, name := range []string{"implementation.go", "oracles.go", "provider.go", "replay.go", "retained.go"} {
		source, err := implementationSources.ReadFile(name)
		if err != nil {
			return ""
		}
		data = fmt.Appendf(data, "%s:%x\n", name, sha256.Sum256(source))
	}
	data = fmt.Appendf(data, "wasi:%s\nclock:%s\nworld:%s\ncluster:%s\n", wasi.ImplementationSHA256(), runner.SimulationClockImplementationSHA256(), world.ImplementationSHA256(), gomad3sim.ImplementationSHA256())
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
