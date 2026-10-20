package cluster

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
)

//go:embed cluster.go controller.go doc.go exploration.go fault.go implementation.go inspect.go internal/history/types.go internal/wire/domain.go internal/wire/keys.go internal/wire/network.go internal/wire/process.go internal/wire/volume.go oracle.go process_model.go process_node.go record.go registry.go runtime_config.go runtime_domain.go runtime_network.go runtime_process.go runtime_process_model.go runtime_time.go runtime_volume.go scenario.go scenario_control.go spec.go types.go wasm.go wasm_inspection.go wasm_inspection_codec.go
var implementationSources embed.FS

var implementationSourceNames = []string{
	"cluster.go", "controller.go", "doc.go", "exploration.go", "fault.go", "implementation.go", "inspect.go",
	"internal/history/types.go", "internal/wire/domain.go", "internal/wire/keys.go", "internal/wire/network.go", "internal/wire/process.go", "internal/wire/volume.go",
	"oracle.go", "process_model.go", "process_node.go", "record.go", "registry.go", "runtime_config.go", "runtime_domain.go", "runtime_network.go",
	"runtime_process.go", "runtime_process_model.go", "runtime_time.go", "runtime_volume.go", "scenario.go", "scenario_control.go", "spec.go", "types.go", "wasm.go", "wasm_inspection.go", "wasm_inspection_codec.go",
}

// ImplementationSHA256 binds the reused cluster lifecycle and evidence owners.
func ImplementationSHA256() string {
	return implementationSHA256(implementationSources)
}

func implementationSHA256(sources fs.FS) string {
	data := []byte("gomad-cluster-implementation/v1\n")
	for _, name := range implementationSourceNames {
		source, err := fs.ReadFile(sources, name)
		if err != nil {
			return ""
		}
		data = fmt.Appendf(data, "%s:%x\n", name, sha256.Sum256(source))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
