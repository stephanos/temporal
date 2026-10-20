package wasi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventOrderCaptureIsPrivateToDiagnosticRoles(t *testing.T) {
	ordinary := Request{Engine: PinnedEngine()}
	before, err := json.Marshal(ordinary)
	require.NoError(t, err)
	ordinary.SetEventOrderDiagnostics(true)
	require.Equal(t, uint64(64<<10), ordinary.helperOutputLimit())
	ordinary.SetStressHostConditions("", false)
	ordinary.SetEventOrderDiagnostics(true)
	require.Equal(t, uint64((32<<20)+(64<<10)), ordinary.helperOutputLimit())
	require.Contains(t, ordinary.stressHostEnvironment, "GOMAD_WASM_EVENT_ORDER=1")
	require.NoError(t, ordinary.SetDeterminismFixtureConditions(""))
	ordinary.SetEventOrderDiagnostics(true)
	require.Equal(t, uint64((32<<20)+(64<<10)), ordinary.helperOutputLimit())
	after, err := json.Marshal(ordinary)
	require.NoError(t, err)
	require.Equal(t, before, after)
	ordinary.SetEventOrderDiagnostics(false)
	require.NotContains(t, ordinary.stressHostEnvironment, "GOMAD_WASM_EVENT_ORDER=1")
}

func TestClusterEventOrderAccessorIsDetached(t *testing.T) {
	result := ClusterResult{helperStderr: []byte("sidecar")}
	detached := result.HelperStderr()
	detached[0] = 'X'
	require.Equal(t, "sidecar", string(result.HelperStderr()))
}
