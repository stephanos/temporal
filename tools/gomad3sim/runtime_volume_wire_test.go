package gomad3sim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuntimeVolumeWireConfigProjectsMountedCapacitiesAndReplay(t *testing.T) {
	replay := &ReplayPlan{Volumes: VolumeRecord{Transitions: []VolumeTransition{{Ordinal: 3, Kind: VolumeOperationWrite, Dependencies: []uint64{}, SelectedOperations: []uint64{}, Bytes: 2, Outcome: VolumeOutcomeOK}}}}
	spec := Spec{
		Seed: 7, Limits: Limits{VolumeOperations: 9, VolumeTransitions: 11},
		Volumes: []VolumeSpec{{ID: "v", CapacityBytes: 1024}},
		Nodes:   []NodeSpec{{ID: "n", Volumes: []VolumeMount{{Volume: "v", Path: "/data"}}}},
		Replay:  replay,
	}
	encoded, err := encodeRuntimeVolumeConfig(spec)
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
	transition := VolumeTransition{Ordinal: 3, Kind: VolumeOperationWrite, Dependencies: []uint64{}, SelectedOperations: []uint64{}, Bytes: 2, Outcome: VolumeOutcomeOK}
	record := VolumeRecord{Transitions: []VolumeTransition{transition}}
	encoder := &runtimeVolumeWireEncoder{data: []byte(runtimeVolumeFinishMagic)}
	encodeRuntimeVolumeRecord(encoder, record)
	encoder.boolean(true)
	encodeRuntimeVolumeErrorBody(encoder, runtimeVolumeError{Kind: "replay", Message: "diverged", Ordinal: 3, ExpectedSHA256: "expected", ActualSHA256: "actual", Expected: &transition, Actual: &transition})
	require.NoError(t, encoder.err)
	got, bridge, err := decodeRuntimeVolumeFinish(encoder.data)
	require.NoError(t, err)
	require.Equal(t, record, got)
	require.NotNil(t, bridge)
	var divergence *ReplayDivergenceError
	err = runtimeVolumeErrorValue(*bridge)
	require.ErrorAs(t, err, &divergence)
	require.ErrorIs(t, err, ErrReplayDiverged)
	require.Equal(t, ReplayDivergence{
		Dimension: ReplayDimensionVolume, Ordinal: 3, ExpectedSHA256: "expected", ActualSHA256: "actual",
		ExpectedVolume: &transition, ActualVolume: &transition,
	}, divergence.Divergence)
}

func TestRuntimeVolumeWireRecordRoundTrip(t *testing.T) {
	record := VolumeRecord{
		Transitions: []VolumeTransition{{
			Ordinal: 3, Kind: VolumeOperationWrite, Handle: NodeHandle{Node: "node", Incarnation: 2}, Volume: "data",
			Operation: 7, Dependencies: []uint64{4, 6}, SelectedOperations: []uint64{}, Inode: 9, Offset: 2, Bytes: 3,
			PayloadSHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			EffectSHA256:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Outcome:       VolumeOutcomeOK,
		}},
		Snapshot: VolumeSnapshot{
			Volumes: []VolumeStateSnapshot{{
				Volume: "data", Node: "node", Mount: "/data", CapacityBytes: 1024,
				Persisted:         []VolumeEntrySnapshot{{Path: "/", Mode: 0o755, Kind: "directory"}},
				Volatile:          []VolumeEntrySnapshot{{Path: "/value", Mode: 0o600, Kind: "file", Size: 3, DataSHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}},
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
	decoded, runtimeErr, err := decodeRuntimeVolumeFinish(encoder.data)
	require.NoError(t, err)
	require.Nil(t, runtimeErr)
	require.Equal(t, record, decoded)
}

func TestRuntimeVolumeWireEnumerationAndFrontierRoundTrip(t *testing.T) {
	frontier := VolumeCrashFrontier{
		Volume: "data", PendingSHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Cursor: []byte{0, 1}, Seen: []string{"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		Identity: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
	encodedFrontier, err := encodeRuntimeVolumeFrontier(&frontier)
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
	encoder.bytes([]byte("value"), MaximumVolumeCapacityBytes)
	encoder.string("sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	encoder.boolean(true)
	encodeRuntimeVolumeFrontierBody(encoder, frontier)
	encoder.boolean(false)
	encoder.string(string(VolumeCrashCapacityStates))
	require.NoError(t, encoder.err)
	page, err := decodeRuntimeVolumeEnumeration(encoder.data)
	require.NoError(t, err)
	require.Equal(t, []byte("value"), page.States[0].Entries[0].Data)
	require.Equal(t, frontier, *page.Frontier)
	require.Equal(t, VolumeCrashCapacityStates, page.Capacity)
}

func TestRuntimeVolumeWireRejectsMalformedInput(t *testing.T) {
	_, _, err := decodeRuntimeVolumeFinish([]byte(runtimeVolumeFinishMagic))
	require.Error(t, err)
	_, err = decodeRuntimeVolumeEnumeration([]byte("invalid"))
	require.Error(t, err)
	_, err = encodeRuntimeVolumeFrontier(&VolumeCrashFrontier{Cursor: make([]byte, MaximumVolumeOperations+1)})
	require.Error(t, err)
}
