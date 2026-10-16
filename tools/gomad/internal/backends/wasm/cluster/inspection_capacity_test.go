package simulation

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestInspectionCapacityRetainsTemporalScaleBoundariesAndEffects(t *testing.T) {
	const decisions, effects = 51_000, 376_000
	handle := gomadsim.NodeHandle{Node: "history", Incarnation: 1}
	metadata := &gomadsim.WASMInspectionEvidence{Schema: gomadsim.WASMInspectionSchema, Complete: true, WorldEncoding: gomadsim.WASMInspectionWorldEncoding, World: []byte(gomadsim.WASMInspectionWorldHeader), Lifecycle: []byte(gomadsim.WASMInspectionLifecycleHeader)}
	var order uint64
	appendEffect := func(action byte) {
		metadata.Lifecycle = append(metadata.Lifecycle, action)
		metadata.Lifecycle = binary.LittleEndian.AppendUint32(metadata.Lifecycle, uint32(len(handle.Node)))
		metadata.Lifecycle = append(metadata.Lifecycle, string(handle.Node)...)
		metadata.Lifecycle = binary.LittleEndian.AppendUint64(metadata.Lifecycle, handle.Incarnation)
		order++
	}
	states := make(map[string][]byte)
	for _, kind := range []string{"network", "storage", "recovery"} {
		digest, err := gomadsim.WASMInspectionStateCommitment(kind, nil)
		require.NoError(t, err)
		states[kind] = digest[:]
	}
	appendEffect(1)
	completedEffects := 0
	for decision := range decisions {
		for completedEffects < (decision+1)*effects/decisions {
			appendEffect(6)
			completedEffects++
		}
		appendEffect(2)
		metadata.World = binary.LittleEndian.AppendUint64(metadata.World, sha256.Size)
		metadata.World = append(metadata.World, make([]byte, sha256.Size)...)
		witness := &gomadsim.WASMRuntimeWitness{Goroutines: 2, Choices: uint64(decision), RuntimeState: make([]byte, 128)}
		binary.BigEndian.PutUint64(witness.RuntimeState, uint64(decision))
		binary.BigEndian.PutUint64(witness.RuntimeState[104:], uint64(decision))
		binary.BigEndian.PutUint64(witness.RuntimeState[112:], 2)
		cut := gomadsim.WASMInspectionCut{Handle: handle, ChoiceRecords: uint64(decision), ChoiceBytes: uint64(decision * 96), WASIBytes: uint64(8 + completedEffects*16), ClockRun: []byte{}, StateEncoding: gomadsim.WASMInspectionStateEncoding, NetworkState: states["network"], StorageState: states["storage"], RecoveryState: states["recovery"], Witness: witness}
		metadata.Boundaries = append(metadata.Boundaries, gomadsim.WASMInspectionBoundary{Handle: handle, Decision: uint64(decision), Trace: uint64(decision), Order: order, WorldBytes: uint64(len(metadata.World)), LifecycleBytes: uint64(len(metadata.Lifecycle)), Cuts: []gomadsim.WASMInspectionCut{cut}})
	}
	appendEffect(3)
	streams := gomadsim.WASMInspectionStreams{Choice: make([]byte, decisions*96), WASI: make([]byte, 8+effects*16)}
	metadata.Incarnations = []gomadsim.WASMInspectionIncarnation{{Handle: handle, StartedOrder: 1, TerminalOrder: order, Streams: streams}}
	execution := &gomadsim.WASMExecutionEvidence{Inspection: metadata, Incarnations: []gomadsim.WASMIncarnationEvidence{{Handle: handle, ChoiceTrace: streams.Choice, ChoiceTape: make([]byte, decisions*96), WASISHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(streams.WASI)), NetworkSHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(nil)), StorageSHA256: fmt.Sprintf("sha256:%x", sha256.Sum256(nil))}}}
	require.Equal(t, effects, completedEffects)
	require.Equal(t, uint64(decisions+effects+2), order)
	require.Len(t, metadata.Lifecycle, 8+(decisions+effects+2)*20)
	require.NoError(t, gomadsim.ValidateWASMInspection(metadata, execution))
	encoded, err := json.Marshal(metadata)
	require.NoError(t, err)
	require.LessOrEqual(t, len(encoded), gomadsim.MaximumWASMInspectionBytes)
	decoded, err := gomadsim.DecodeWASMInspection(encoded)
	require.NoError(t, err)
	require.Len(t, decoded.Boundaries, decisions)
	require.Equal(t, metadata.Lifecycle, decoded.Lifecycle)
	require.Equal(t, streams, decoded.Incarnations[0].Streams)
	for i := range metadata.Boundaries {
		require.Equal(t, metadata.Boundaries[i], decoded.Boundaries[i])
	}
	record, err := json.Marshal(gomadsim.ClusterRecord{Backend: gomadsim.BackendWASM, WASM: execution})
	require.NoError(t, err)
	require.NoError(t, retainedCapacityLowerBound(uint64(len(record))))
	bundle, err := json.Marshal(retainedExecution{Schema: RetainedSchema, Record: record})
	require.NoError(t, err)
	require.LessOrEqual(t, len(bundle), 512<<20)
	t.Logf("inspection=%d record=%d retained_bundle=%d boundaries=%d lifecycle_effects=%d", len(encoded), len(record), len(bundle), len(decoded.Boundaries), order)
}
