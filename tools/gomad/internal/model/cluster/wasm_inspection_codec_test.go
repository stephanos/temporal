package cluster

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWASMInspectionBinaryBoundaryRoundTripAndTruncation(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	frames, err := encodeInspectionBoundaries(metadata.Boundaries)
	require.NoError(t, err)
	decoded, err := decodeInspectionBoundaries(frames)
	require.NoError(t, err)
	require.Equal(t, metadata.Boundaries, decoded)
	for length := range len(frames) {
		_, err := decodeInspectionBoundaries(frames[:length])
		require.Error(t, err, "accepted partial frame at byte %d", length)
	}
	decoded[0].Cuts[0].Witness.RuntimeState[0] ^= 1
	require.NotEqual(t, frames[0], decoded[0].Cuts[0].Witness.RuntimeState[0])
	unchanged, err := decodeInspectionBoundaries(frames)
	require.NoError(t, err)
	require.Equal(t, metadata.Boundaries, unchanged)
}

func TestWASMInspectionBinaryBoundaryRejectsMalformedMaterial(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	metadata.Boundaries = metadata.Boundaries[:1]
	metadata.Boundaries[0].Handle = NodeHandle{Node: "a", Incarnation: 1}
	metadata.Boundaries[0].Cuts[0].Handle = metadata.Boundaries[0].Handle
	metadata.Boundaries[0].Cuts[0].ClockRun = nil
	frames, err := encodeInspectionBoundaries(metadata.Boundaries)
	require.NoError(t, err)
	// Header/count/frame length occupy 16 bytes. Both inline handles occupy 13.
	const cutAt = 16 + 13 + 48 + 4
	const clockAt = cutAt + 13 + 56
	const witnessAt = clockAt + 4 + 96
	for _, test := range []struct {
		name string
		edit func([]byte) []byte
	}{
		{"old version", func(b []byte) []byte { b[5] = 0; return b }},
		{"huge count", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[8:], 65537); return b }},
		{"huge frame", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[12:], ^uint32(0)); return b }},
		{"empty node", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[16:], 0); return b }},
		{"huge node", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[16:], 1025); return b }},
		{"invalid UTF8", func(b []byte) []byte { b[20] = 255; return b }},
		{"newline node", func(b []byte) []byte { b[20] = '\n'; return b }},
		{"NUL node", func(b []byte) []byte { b[20] = 0; return b }},
		{"zero incarnation", func(b []byte) []byte { binary.LittleEndian.PutUint64(b[21:], 0); return b }},
		{"huge cuts", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[77:], 65537); return b }},
		{"zero cut incarnation", func(b []byte) []byte { binary.LittleEndian.PutUint64(b[cutAt+5:], 0); return b }},
		{"huge clock", func(b []byte) []byte { binary.LittleEndian.PutUint32(b[clockAt:], 1025); return b }},
		{"choices exceed records", func(b []byte) []byte { binary.BigEndian.PutUint64(b[witnessAt+104:], ^uint64(0)); return b }},
		{"zero goroutines", func(b []byte) []byte { binary.BigEndian.PutUint64(b[witnessAt+112:], 0); return b }},
		{"trailing bytes", func(b []byte) []byte { return append(b, 0) }},
		{"frame padding", func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[12:], binary.LittleEndian.Uint32(b[12:])+1)
			return append(b, 0)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeInspectionBoundaries(test.edit(bytes.Clone(frames)))
			require.Error(t, err)
		})
	}
}

func TestWASMInspectionBinaryWitnessProjectsAllRawCounters(t *testing.T) {
	raw := make([]byte, 128)
	for at, value := range map[int]uint64{0: 50, 16: 11, 40: ^uint64(0), 48: 2, 56: 3, 64: 4, 72: 5, 80: 6, 88: 7, 96: 13, 104: 20, 112: 9, 120: 10} {
		binary.BigEndian.PutUint64(raw[at:], value)
	}
	binary.BigEndian.PutUint32(raw[24:], 12)
	raw[28] = 2
	binary.BigEndian.PutUint32(raw[32:], 8)
	witness, err := decodeInspectionWitness(raw)
	require.NoError(t, err)
	require.Equal(t, &WASMRuntimeWitness{Allocations: 11, AllocatedBytes: 13, GCCycles: 12, GCPhase: 2, Goroutines: 9, RunQueue: 8, GlobalRunQueue: 10, SeededDraws: 26, SeededDrawCounters: [7]uint64{^uint64(0), 2, 3, 4, 5, 6, 7}, Choices: 20, Observations: 30, RuntimeState: raw}, witness)
	metadata, _ := inspectionFixture(t)
	metadata.Boundaries[0].Cuts[0].Witness = witness
	_, err = encodeInspectionBoundaries(metadata.Boundaries)
	require.NoError(t, err)
	witness.Allocations++
	_, err = encodeInspectionBoundaries(metadata.Boundaries)
	require.ErrorContains(t, err, "projection differs")
}

func TestWASMInspectionBinaryEmptyClockRemainsBase64InControlJSON(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	metadata.Boundaries[0].Cuts[0].ClockRun = nil
	frames, err := encodeInspectionBoundaries(metadata.Boundaries)
	require.NoError(t, err)
	boundaries, err := decodeInspectionBoundaries(frames)
	require.NoError(t, err)
	boundary := cloneInspectionBoundary(boundaries[0])
	encoded, err := json.Marshal(boundary.Cuts[0])
	require.NoError(t, err)
	var control struct {
		ClockRun string `json:"clock_run"`
	}
	require.NoError(t, json.Unmarshal(encoded, &control))
	require.Contains(t, string(encoded), `"clock_run":""`)
}

func TestWASMInspectionBinaryWitnessAuthenticatesUnprojectedRawBytes(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	observed := CloneWASMInspection(metadata)
	observed.Boundaries[1].Cuts[0].Witness.RuntimeState[8] ^= 1
	require.ErrorIs(t, VerifyWASMInspectionPrefix(metadata, observed, metadata.Boundaries[1]), ErrWASMInstrumentationDivergence)
}

func TestWASMInspectionBinaryLifecyclePreservesEveryAction(t *testing.T) {
	data := []byte(WASMInspectionLifecycleHeader)
	ends := make(map[int]bool)
	for action := byte(1); action <= 10; action++ {
		node, incarnation := "a", uint64(1)
		if action == 7 {
			node, incarnation = "controller", 0
		}
		data = append(data, action)
		data = binary.LittleEndian.AppendUint32(data, uint32(len(node)))
		data = append(data, node...)
		data = binary.LittleEndian.AppendUint64(data, incarnation)
		ends[len(data)] = true
	}
	cuts, err := inspectionLifecycleCuts(data)
	require.NoError(t, err)
	require.Len(t, cuts, 10)
	for end, cut := range cuts {
		require.True(t, ends[int(end)])
		require.Equal(t, uint64(cut.action), cut.order)
	}
	for length := range len(data) {
		_, err := inspectionLifecycleCuts(data[:length])
		if length == len(WASMInspectionLifecycleHeader) || ends[length] {
			require.NoError(t, err)
		} else {
			require.Error(t, err, "accepted partial lifecycle at byte %d", length)
		}
	}
	for _, edit := range []func([]byte){
		func(b []byte) { b[5] = 0 },
		func(b []byte) { b[8] = 0 },
		func(b []byte) { b[8] = 11 },
		func(b []byte) { b[8] = 7 },
		func(b []byte) { b[13] = 255 },
		func(b []byte) { binary.LittleEndian.PutUint64(b[14:], 0) },
	} {
		mutated := bytes.Clone(data)
		edit(mutated)
		_, err := inspectionLifecycleCuts(mutated)
		require.Error(t, err)
	}
}

func TestWASMInspectionBinaryEvidenceRequiresVersionedDiscriminators(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	encoded, err := json.Marshal(metadata)
	require.NoError(t, err)
	for _, field := range []string{"boundary_encoding", "lifecycle_encoding", "state_encoding", "world_encoding", "schema"} {
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(encoded, &fields))
		for _, value := range []json.RawMessage{json.RawMessage(`"unknown"`), json.RawMessage(`null`)} {
			fields[field] = value
			mutated, err := json.Marshal(fields)
			require.NoError(t, err)
			_, err = DecodeWASMInspection(mutated)
			require.Error(t, err)
		}
		delete(fields, field)
		mutated, err := json.Marshal(fields)
		require.NoError(t, err)
		_, err = DecodeWASMInspection(mutated)
		require.Error(t, err)
	}
	for _, version := range []string{"gomad.wasm-inspection/v1", "gomad.wasm-inspection/v2"} {
		_, err := DecodeWASMInspection(bytes.Replace(encoded, []byte(WASMInspectionSchema), []byte(version), 1))
		require.ErrorIs(t, err, ErrWASMInspectionUnsupported)
	}
	_, err = DecodeWASMInspection(append([]byte(`{"boundary_encoding":"duplicate",`), encoded[1:]...))
	require.Error(t, err)
}
