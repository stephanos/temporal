package wasi

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/clock"
	"go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/softassert"
	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

//go:embed cluster_network.go debug_output.go environment.go namespace.go namespace_image.go clock.go protocol.go transport.go runtime.go helper_model.go identity.go measurement.go storage_evidence.go storage_image.go storage_selection.go volume.go network_v2.go broker.go broker_network.go broker_process.go transcript.go
var implementationSources embed.FS

func ImplementationSHA256() string {
	return implementationSHA256(implementationSources)
}

func implementationSHA256(sources fs.ReadFileFS) string {
	return implementationSHA256WithStrictJSON(sources, strictjson.ImplementationSHA256())
}

func implementationSHA256WithStrictJSON(sources fs.ReadFileFS, strictJSON string) string {
	if strictJSON == "" {
		return ""
	}
	var inventory []byte
	for _, name := range []string{"broker.go", "broker_network.go", "broker_process.go", "clock.go", "cluster_network.go", "debug_output.go", "environment.go", "helper_model.go", "identity.go", "measurement.go", "namespace.go", "namespace_image.go", "network_v2.go", "transport.go", "protocol.go", "runtime.go", "storage_evidence.go", "storage_image.go", "storage_selection.go", "transcript.go", "volume.go"} {
		data, err := sources.ReadFile(name)
		if err != nil {
			return ""
		}
		inventory = fmt.Appendf(inventory, "%s:%x\n", name, sha256.Sum256(data))
	}
	inventory = fmt.Appendf(inventory, "volume-projection:%s\n", volume.ImplementationSHA256())
	inventory = fmt.Appendf(inventory, "network-projection:%s\n", network.ImplementationSHA256())
	inventory = fmt.Appendf(inventory, "interrupted-replay:%s\n", choice.InterruptedReplayImplementationSHA256())
	inventory = fmt.Appendf(inventory, "broker-clock:%s\n", clock.BrokerImplementationSHA256())
	inventory = fmt.Appendf(inventory, "broker-world:%s\n", world.ImplementationSHA256())
	inventory = fmt.Appendf(inventory, "strict-json:%s\n", strictJSON)
	inventory = fmt.Appendf(inventory, "soft-assertion:%s\n", softassert.ImplementationSHA256())
	return fmt.Sprintf("sha256:%x", sha256.Sum256(inventory))
}
