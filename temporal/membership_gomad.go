//go:build gomad || gomad_wasm_slim

package temporal

import (
	"errors"

	"go.uber.org/fx"
)

// Ringpop membership reaches syscall through tchannel and thrift; the native
// gomad build and the gomad_wasm_slim WASM guest build serve only clusters
// that supply StaticServiceHosts.
func defaultMembershipModule() fx.Option {
	return fx.Error(errors.New("ringpop membership is not built under the gomad or gomad_wasm_slim build tags; supply StaticServiceHosts"))
}
