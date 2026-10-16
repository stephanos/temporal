package simulation

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/clock"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

//go:embed implementation.go inspection.go inspection_control.go inspection_network.go operator.go oracles.go provider.go replay.go replay_inspection.go replay_inspection_search.go retained.go soft_assertion.go
var implementationSources embed.FS

func ImplementationSHA256() string {
	return implementationSHA256(implementationSources)
}

func implementationSHA256(sources fs.ReadFileFS) string {
	data := []byte("gomad-wasm-simulation-implementation/v1\n")
	for _, name := range []string{"implementation.go", "inspection.go", "inspection_control.go", "inspection_network.go", "operator.go", "oracles.go", "provider.go", "replay.go", "replay_inspection.go", "replay_inspection_search.go", "retained.go", "soft_assertion.go"} {
		source, err := sources.ReadFile(name)
		if err != nil {
			return ""
		}
		data = fmt.Appendf(data, "%s:%x\n", name, sha256.Sum256(source))
	}
	data = fmt.Appendf(data, "wasi:%s\nclock:%s\nworld:%s\ncluster:%s\n", wasi.ImplementationSHA256(), clock.BrokerImplementationSHA256(), world.ImplementationSHA256(), gomadsim.ImplementationSHA256())
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
