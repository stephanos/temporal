package gomad3sim

import (
	"crypto/sha256"
	"embed"
	"fmt"
)

//go:embed cluster.go controller.go exploration.go fault.go implementation.go inspect.go oracle.go record.go registry.go scenario.go scenario_control.go spec.go types.go wasm.go
var implementationSources embed.FS

// ImplementationSHA256 binds the reused cluster lifecycle and evidence owners.
func ImplementationSHA256() string {
	data := []byte("gomad3-cluster-implementation/v1\n")
	for _, name := range []string{"cluster.go", "controller.go", "exploration.go", "fault.go", "implementation.go", "inspect.go", "oracle.go", "record.go", "registry.go", "scenario.go", "scenario_control.go", "spec.go", "types.go", "wasm.go"} {
		source, err := implementationSources.ReadFile(name)
		if err != nil {
			return ""
		}
		data = fmt.Appendf(data, "%s:%x\n", name, sha256.Sum256(source))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
