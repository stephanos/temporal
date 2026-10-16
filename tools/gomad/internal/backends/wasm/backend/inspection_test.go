package backend

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestInspectionProviderRefusesMissingArtifactBeforeExecution(t *testing.T) {
	p := &Provider{}
	_, err := p.Inspect(t.Context(), nil, backends.InspectionRequest{Stop: backends.InspectionStop{Kind: "choice"}, MaximumBytes: backends.DefaultInspectionBytes, PerDumpBytes: backends.DefaultInspectionDumpBytes})
	var preflight *backends.InspectionPreflightError
	require.ErrorAs(t, err, &preflight)
	require.ErrorContains(t, err, "opened artifact")
}

func TestInspectionOutputBoundsIncludeJSONAndHeap(t *testing.T) {
	output := wasi.DebugOutput{HeapProfile: bytes.Repeat([]byte{1}, 1024), Dumps: []wasi.DebugDump{{Name: "snapshot", Status: "ok", Data: bytes.Repeat([]byte{2}, 4096)}}}
	encoded, heap, truncated, err := boundInspectionOutput(output, 512)
	require.NoError(t, err)
	require.True(t, truncated)
	require.LessOrEqual(t, len(encoded)+len(heap), 512)
	var result struct {
		Truncated     bool             `json:"truncated"`
		Dumps         []wasi.DebugDump `json:"dumps"`
		HeapTruncated bool             `json:"heap_truncated"`
	}
	require.NoError(t, json.Unmarshal(encoded, &result))
	require.True(t, result.Truncated)
	require.True(t, result.HeapTruncated)
	require.Len(t, result.Dumps, 1)
	require.Equal(t, "truncated", result.Dumps[0].Status)
	_, _, _, err = boundInspectionOutput(output, 1)
	require.Error(t, err)
}

func TestStandaloneInspectionRefusesOldEvidenceWithoutLaunching(t *testing.T) {
	p := &Provider{}
	for _, evidence := range []string{"old raw WASI bytes", `{"schema":"gomad-wasm.modeled-evidence/v1"}`, `{"schema":"gomad-wasm.modeled-evidence/v2","inspection":null}`} {
		_, _, err := p.standaloneInspection(nil, provenance{}, []byte(evidence))
		require.ErrorIs(t, err, cluster.ErrWASMInspectionUnsupported)
	}
}
