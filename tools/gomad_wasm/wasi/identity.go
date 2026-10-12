package wasi

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/volume"
	"go.temporal.io/server/tools/gomad3/runner"
	"go.temporal.io/server/tools/gomad3/simulation/network"
	"go.temporal.io/server/tools/gomad3/world"
)

//go:embed environment.go namespace.go clock.go protocol.go process.go runtime.go network.go helper_model.go identity.go volume.go network_v2.go broker.go broker_network.go broker_process.go
var implementationSources embed.FS

func ImplementationSHA256() string {
	return implementationSHA256(implementationSources)
}

func implementationSHA256(sources fs.ReadFileFS) string {
	var inventory []byte
	for _, name := range []string{"broker.go", "broker_network.go", "broker_process.go", "clock.go", "environment.go", "helper_model.go", "identity.go", "namespace.go", "network.go", "network_v2.go", "process.go", "protocol.go", "runtime.go", "volume.go"} {
		data, err := sources.ReadFile(name)
		if err != nil {
			return ""
		}
		inventory = fmt.Appendf(inventory, "%s:%x\n", name, sha256.Sum256(data))
	}
	inventory = fmt.Appendf(inventory, "volume-projection:%s\n", volume.ImplementationSHA256())
	inventory = fmt.Appendf(inventory, "network-projection:%s\n", network.ImplementationSHA256())
	inventory = fmt.Appendf(inventory, "interrupted-replay:%s\n", choice.InterruptedReplayImplementationSHA256())
	inventory = fmt.Appendf(inventory, "broker-clock:%s\n", runner.SimulationClockImplementationSHA256())
	inventory = fmt.Appendf(inventory, "broker-world:%s\n", world.ImplementationSHA256())
	return fmt.Sprintf("sha256:%x", sha256.Sum256(inventory))
}
