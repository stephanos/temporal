package wire

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/cluster/internal/history"
)

func TestRuntimeVolumeWireConfigProjectsMountedCapacitiesAndReplay(t *testing.T) {
	replay := &history.RuntimeReplay{Volumes: history.VolumeRecord{Transitions: []history.VolumeTransition{{Ordinal: 3, Kind: "write", Dependencies: []uint64{}, SelectedOperations: []uint64{}, Bytes: 2, Outcome: "ok"}}}}
	spec := history.RuntimeConfig{
		Seed: 7, Limits: history.RuntimeLimits{VolumeOperations: 9, VolumeTransitions: 11},
		Volumes: []history.RuntimeVolume{{ID: "v", CapacityBytes: 1024}},
		Nodes:   []history.RuntimeNode{{ID: "n", Volumes: []history.RuntimeMount{{Volume: "v", Path: "/data"}}}},
		Replay:  replay,
	}
	encoded, err := EncodeVolumeConfig(spec)
	require.NoError(t, err)
	decoder := newRuntimeVolumeWireDecoder(encoded, runtimeVolumeConfigMagic)
	value := decodeRuntimeVolumeConfigBody(decoder)
	require.NoError(t, decoder.finish())
	require.Equal(t, uint64(7), value.Seed)
	require.Equal(t, uint64(9), value.Limits.Operations)
	require.Equal(t, uint64(11), value.Limits.Transitions)
	require.Equal(t, []runtimeVolumeWireNode{{Node: "n", Volumes: []runtimeVolumeWireMount{{ID: "v", Path: "/data", CapacityBytes: 1024}}}}, value.Nodes)
	require.Equal(t, &replay.Volumes, value.Replay)
}

func TestRuntimeVolumeWireTypedReplayErrorPreservesPartialRecord(t *testing.T) {
	transition := history.VolumeTransition{Ordinal: 3, Kind: "write", Dependencies: []uint64{}, SelectedOperations: []uint64{}, Bytes: 2, Outcome: "ok"}
	record := history.VolumeRecord{Transitions: []history.VolumeTransition{transition}}
	encoder := &runtimeVolumeWireEncoder{data: []byte(runtimeVolumeFinishMagic)}
	encodeRuntimeVolumeRecord(encoder, record)
	encoder.boolean(true)
	encodeRuntimeVolumeErrorBody(encoder, history.RuntimeVolumeError{Kind: "replay", Message: "diverged", Ordinal: 3, ExpectedSHA256: "expected", ActualSHA256: "actual", Expected: &transition, Actual: &transition})
	require.NoError(t, encoder.err)
	got, bridge, err := DecodeVolumeFinish(encoder.data)
	require.NoError(t, err)
	require.Equal(t, record, got)
	require.NotNil(t, bridge)
	require.Equal(t, history.RuntimeVolumeError{Kind: "replay", Message: "diverged", Ordinal: 3, ExpectedSHA256: "expected", ActualSHA256: "actual", Expected: &transition, Actual: &transition}, *bridge)
}

func TestRuntimeVolumeWireRecordRoundTrip(t *testing.T) {
	record := history.VolumeRecord{
		Transitions: []history.VolumeTransition{{
			Ordinal: 3, Kind: "write", Handle: history.NodeHandle{Node: "node", Incarnation: 2}, Volume: "data",
			Operation: 7, Dependencies: []uint64{4, 6}, SelectedOperations: []uint64{}, Inode: 9, Offset: 2, Bytes: 3,
			PayloadSHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			EffectSHA256:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Outcome:       "ok",
		}},
		Snapshot: history.VolumeSnapshot{
			Volumes: []history.VolumeStateSnapshot{{
				Volume: "data", Node: "node", Mount: "/data", CapacityBytes: 1024,
				Persisted:         []history.VolumeEntrySnapshot{{Path: "/", Mode: 0o755, Kind: "directory"}},
				Volatile:          []history.VolumeEntrySnapshot{{Path: "/value", Mode: 0o600, Kind: "file", Size: 3, DataSHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}},
				PendingOperations: 1,
				PendingSHA256:     "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
				NextOperation:     8,
				Identity:          "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			}},
			TransitionSHA256: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			Identity:         "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		},
	}
	encoder := &runtimeVolumeWireEncoder{data: []byte(runtimeVolumeFinishMagic)}
	encodeRuntimeVolumeRecord(encoder, record)
	encoder.boolean(false)
	require.NoError(t, encoder.err)
	decoded, runtimeErr, err := DecodeVolumeFinish(encoder.data)
	require.NoError(t, err)
	require.Nil(t, runtimeErr)
	require.Equal(t, record, decoded)
}

func TestRuntimeVolumeWireEnumerationAndFrontierRoundTrip(t *testing.T) {
	frontier := history.VolumeCrashFrontier{
		Volume: "data", PendingSHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Cursor: []byte{0, 1}, Seen: []string{"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		Identity: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
	encodedFrontier, err := EncodeVolumeFrontier(&frontier)
	require.NoError(t, err)
	require.Equal(t, runtimeVolumeFrontierMagic, string(encodedFrontier[:len(runtimeVolumeFrontierMagic)]))

	encoder := &runtimeVolumeWireEncoder{data: []byte(runtimeVolumeEnumerationMagic)}
	encoder.uint64(1)
	encoder.string("data")
	encoder.string(frontier.PendingSHA256)
	encoder.uint64(1)
	encoder.uint64(2)
	encoder.uint64(1)
	encoder.string("/value")
	encoder.uint64(0o600)
	encoder.uint64(1)
	encoder.uint64(7)
	encoder.bytes([]byte("value"), (1 << 30))
	encoder.string("sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	encoder.boolean(true)
	encodeRuntimeVolumeFrontierBody(encoder, frontier)
	encoder.boolean(false)
	encoder.string(string(history.VolumeCrashCapacity("states")))
	require.NoError(t, encoder.err)
	page, err := DecodeVolumeEnumeration(encoder.data)
	require.NoError(t, err)
	require.Equal(t, []byte("value"), page.States[0].Entries[0].Data)
	require.Equal(t, frontier, *page.Frontier)
	require.Equal(t, history.VolumeCrashCapacity("states"), page.Capacity)
}

func TestRuntimeVolumeWireRejectsMalformedInput(t *testing.T) {
	_, _, err := DecodeVolumeFinish([]byte(runtimeVolumeFinishMagic))
	require.Error(t, err)
	_, err = DecodeVolumeEnumeration([]byte("invalid"))
	require.Error(t, err)
	_, err = EncodeVolumeFrontier(&history.VolumeCrashFrontier{Cursor: make([]byte, (1<<20)+1)})
	require.Error(t, err)
}
