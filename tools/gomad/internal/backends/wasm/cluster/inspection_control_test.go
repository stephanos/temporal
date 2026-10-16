package simulation

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestInspectionControlAuthenticatesRawCutAndRegistry(t *testing.T) {
	execution := inspectionSearchFixture(t)
	resolved, err := ResolveInspection(execution, cluster.WASMInspectionRequest{Stop: cluster.WASMInspectionStop{Kind: "choice", Value: 2}}, false)
	require.NoError(t, err)
	encoded, err := InspectionControl(execution.Inspection, resolved, "healthy")
	require.NoError(t, err)
	var control struct {
		Boundary   cluster.WASMInspectionBoundary   `json:"expected"`
		World      string                           `json:"world_sha256"`
		Streams    []map[string]string              `json:"stream_sha256"`
		Registries []cluster.WASMInspectionRegistry `json:"registries"`
		Commands   []struct{ Kind, ID uint32 }      `json:"commands"`
	}
	require.NoError(t, json.Unmarshal(encoded, &control))
	expectedBoundary := resolved.Boundary
	require.Equal(t, expectedBoundary, control.Boundary)
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(execution.Inspection.World[:resolved.Boundary.WorldBytes])), control.World)
	cut := resolved.Boundary.Cuts[0]
	require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(execution.Inspection.Incarnations[0].Streams.Choice[:cut.ChoiceBytes])), control.Streams[0]["choice"])
	require.Equal(t, "healthy", control.Registries[0].Entries[0].Name)
	require.Len(t, control.Commands, 1)
	require.Equal(t, uint32(4), control.Commands[0].Kind)
	resolved.Boundary.Order++
	_, err = InspectionControl(execution.Inspection, resolved, "healthy")
	require.Error(t, err)
}

func TestInspectionControlRefusesUnregisteredAndIncompleteBeforeReplay(t *testing.T) {
	execution := inspectionSearchFixture(t)
	resolved, err := ResolveInspection(execution, cluster.WASMInspectionRequest{Stop: cluster.WASMInspectionStop{Kind: "choice"}}, false)
	require.NoError(t, err)
	_, err = InspectionControl(execution.Inspection, resolved, "missing")
	require.ErrorContains(t, err, "unknown or unavailable")
	execution.Inspection.Complete = false
	_, err = InspectionControl(execution.Inspection, resolved, "healthy")
	require.ErrorIs(t, err, cluster.ErrWASMInspectionUnsupported)
}
