//go:build gomad || gomad_wasm_slim

package temporal

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestDefaultMembershipRefusesUnbuiltRingpop(t *testing.T) {
	err := fx.New(defaultMembershipModule(), fx.NopLogger).Err()
	require.ErrorContains(t, err, "ringpop membership is not built under the gomad or gomad_wasm_slim build tags; supply StaticServiceHosts")
}
