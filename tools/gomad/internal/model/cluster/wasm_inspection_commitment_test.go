package cluster

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWASMInspectionCommitmentsUseOwnerDomainsAndLengthFraming(t *testing.T) {
	for _, test := range []struct {
		kind     string
		material []byte
		golden   string
	}{
		{"storage", []byte("abc"), "bc21ac8235dac2e82b5ff9401aec3720cbe19d167569211b906be2b290c51b7a"},
		{"network", nil, "9cf01abb216e9758d088460097ca46d1fdde9e1f2045bf8b719450202698a51e"},
	} {
		digest, err := WASMInspectionStateCommitment(test.kind, test.material)
		require.NoError(t, err)
		require.Equal(t, test.golden, hex.EncodeToString(digest[:]))
	}
	storage, err := WASMInspectionStateCommitment("storage", []byte("abc"))
	require.NoError(t, err)
	network, err := WASMInspectionStateCommitment("network", []byte("abc"))
	require.NoError(t, err)
	require.NotEqual(t, storage, network)
	_, err = WASMInspectionStateCommitment("unknown", nil)
	require.Error(t, err)
}

func TestWASMInspectionCommitmentEncodingAndFramingRefusals(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	for _, test := range []struct {
		name   string
		mutate func(*WASMInspectionEvidence)
	}{
		{"old raw schema", func(m *WASMInspectionEvidence) { m.Schema = "gomad.wasm-inspection/v1" }},
		{"missing World discriminator", func(m *WASMInspectionEvidence) { m.WorldEncoding = "" }},
		{"old full-wire World encoding", func(m *WASMInspectionEvidence) { m.WorldEncoding = "gomad.inspection-world-sha256/v1" }},
		{"unknown World discriminator", func(m *WASMInspectionEvidence) { m.WorldEncoding = "unknown" }},
		{"raw World snapshot", func(m *WASMInspectionEvidence) { m.World = []byte("raw snapshot") }},
		{"World length", func(m *WASMInspectionEvidence) {
			binary.LittleEndian.PutUint64(m.World[len(WASMInspectionWorldHeader):], 31)
		}},
		{"World partial frame", func(m *WASMInspectionEvidence) { m.World = m.World[:len(m.World)-1] }},
		{"World trailing bytes", func(m *WASMInspectionEvidence) { m.World = append(m.World, 0) }},
		{"World interior cut", func(m *WASMInspectionEvidence) { m.Boundaries[0].WorldBytes-- }},
		{"missing state discriminator", func(m *WASMInspectionEvidence) { m.Boundaries[0].Cuts[0].StateEncoding = "" }},
		{"unknown state discriminator", func(m *WASMInspectionEvidence) { m.Boundaries[0].Cuts[0].StateEncoding = "unknown" }},
		{"raw storage snapshot", func(m *WASMInspectionEvidence) { m.Boundaries[0].Cuts[0].StorageState = []byte("state") }},
		{"missing recovery commitment", func(m *WASMInspectionEvidence) { m.Boundaries[0].Cuts[0].RecoveryState = nil }},
		{"oversized network commitment", func(m *WASMInspectionEvidence) { m.Boundaries[0].Cuts[0].NetworkState = make([]byte, 33) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutated := CloneWASMInspection(metadata)
			test.mutate(mutated)
			require.ErrorIs(t, ValidateWASMInspection(mutated, nil), ErrWASMInspectionUnsupported)
		})
	}
	observed := CloneWASMInspection(metadata)
	observed.World[len(WASMInspectionWorldHeader)+8] ^= 1
	require.ErrorIs(t, VerifyWASMInspectionPrefix(metadata, observed, metadata.Boundaries[1]), ErrWASMInstrumentationDivergence)
}

func TestWASMInspectionCommitmentsKeepEveryBoundaryWithinCapacity(t *testing.T) {
	metadata, _ := inspectionFixture(t)
	handle := metadata.Incarnations[0].Handle
	metadata.Incarnations[0].TerminalOrder = 3000
	metadata.Incarnations[0].Streams = WASMInspectionStreams{Choice: make([]byte, 2000*96)}
	state, err := WASMInspectionStateCommitment("storage", bytes.Repeat([]byte{'s'}, 64<<10))
	require.NoError(t, err)
	metadata.World = []byte(WASMInspectionWorldHeader)
	metadata.Boundaries = nil
	for ordinal := uint64(0); ordinal < 2000; ordinal++ {
		metadata.World = binary.LittleEndian.AppendUint64(metadata.World, sha256.Size)
		digest := sha256.Sum256([]byte{byte(ordinal), byte(ordinal >> 8)})
		metadata.World = append(metadata.World, digest[:]...)
		cut := WASMInspectionCut{Handle: handle, StateEncoding: WASMInspectionStateEncoding, ChoiceRecords: ordinal, ChoiceBytes: ordinal * 96, NetworkState: make([]byte, 32), StorageState: append([]byte{}, state[:]...), RecoveryState: make([]byte, 32), Witness: &WASMRuntimeWitness{Goroutines: 2, RuntimeState: make([]byte, 128)}}
		metadata.Boundaries = append(metadata.Boundaries, WASMInspectionBoundary{Handle: handle, Decision: ordinal, Trace: ordinal, Order: ordinal + 3, WorldBytes: uint64(len(metadata.World)), LifecycleBytes: 1, Cuts: []WASMInspectionCut{cut}})
	}
	commitInspectionFixture(t, metadata)
	require.NoError(t, ValidateWASMInspection(metadata, nil))
	encoded, err := json.Marshal(metadata)
	require.NoError(t, err)
	require.Less(t, len(encoded), 8<<20)
	decoded, err := DecodeWASMInspection(encoded)
	require.NoError(t, err)
	require.Len(t, decoded.Boundaries, 2000)
}
