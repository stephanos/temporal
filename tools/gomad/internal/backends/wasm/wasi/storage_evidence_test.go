package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func TestStorageEvidenceLargeEnumerationOwnerParity(t *testing.T) {
	request := storageEnumerationRequest(t, false)
	var owners [2]Result
	for i := range owners {
		request.remoteModel = i == 0
		result, err := Run(t.Context(), request)
		require.NoError(t, err)
		requireStorageGuestExit(t, result)
		owners[i] = result
	}
	encoded, err := json.Marshal(owners[1].StorageCrashEnumeration)
	require.NoError(t, err)
	require.Greater(t, uint64(len(encoded)), 2*storageImageLimit(request.Config))
	withoutEnumeration := request
	withoutEnumeration.StorageCrashLimits = nil
	require.Greater(t, uint64(len(encoded)), storageStateLimit(withoutEnumeration))
	require.Empty(t, sameStorageEvidence(owners[0], owners[1]))
	require.NotNil(t, owners[1].GoFilesystemCalls)
	require.Empty(t, owners[1].GoFilesystemCalls)
	require.LessOrEqual(t, uint64(owners[1].StorageImage.Bytes()), storageImageLimit(request.Config))
	t.Logf("enumeration JSON=%d bytes, old state bound=%d, image=%d", len(encoded), 2*storageImageLimit(request.Config), owners[1].StorageImage.Bytes())

	recovery := storageEnumerationRequest(t, true)
	for i := range owners {
		recovery.remoteModel, recovery.StorageImage = i == 0, owners[i].StorageImage
		owners[i] = runStorageGuest(t, recovery)
		requireStorageGuestExit(t, owners[i])
		require.Equal(t, "x", string(owners[i].Stdout))
	}
	require.Empty(t, sameStorageEvidence(owners[0], owners[1]))
	require.Empty(t, owners[1].GoFilesystemCalls)
}

func TestStorageEvidenceRefusedRunOwnerParity(t *testing.T) {
	for _, test := range []struct {
		name, termination, message string
		writes                     int
		limits                     *volume.CrashEnumerationLimits
	}{
		{"image-capacity", "capacity", "WASI capacity: storage-image-bytes", 17, nil},
		{"enumeration", "invalid", "crash enumeration limits must be nonzero", 1, &volume.CrashEnumerationLimits{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := fmt.Sprintf(`i32.const 200 i32.const 4096 i32.store i32.const 204 i32.const 65536 i32.store
 loop $writes
 i32.const 4 i32.const 0 i32.const 0 i32.const 4 i32.const 9 i64.const 536870911 i64.const 0 i32.const 0 i32.const 100 call $open if unreachable end
 i32.const 100 i32.load i32.const 200 i32.const 1 i32.const 240 call $write if unreachable end
 local.get $i i32.const 1 i32.add local.tee $i i32.const %d i32.lt_u br_if $writes
 end`, test.writes)
			request := storageEvidenceRequest(t, body, `(data (i32.const 0) "file") (data (i32.const 4096) "x")`, 2)
			request.Config.Limits.FilesystemBytes = 64 << 10
			request.Config.Limits.TranscriptBytes = 8 << 20
			request.Config.Storage.Volumes[0].CapacityBytes = 64 << 10
			request.StorageCrashLimits = test.limits
			var owners [2]Result
			for i := range owners {
				request.remoteModel = i == 0
				result, err := Run(t.Context(), request)
				require.NoError(t, err)
				require.Equal(t, test.termination, result.Termination)
				require.Equal(t, test.message, result.Message)
				require.Nil(t, result.StorageImage)
				owners[i] = result
				t.Logf("owner remote=%v: transcript=%d snapshots=%d operations=%d enumerations=%v", request.remoteModel, len(result.StorageTranscript), len(result.StorageSnapshots), len(result.StorageOperations), result.StorageCrashEnumeration)
			}
			require.NotEmpty(t, owners[0].StorageTranscript)
			require.NotEmpty(t, owners[0].StorageOperations["data"])
			require.Equal(t, owners[0].StorageTranscript, owners[1].StorageTranscript)
			require.Equal(t, owners[0].StorageSnapshots, owners[1].StorageSnapshots)
			require.Equal(t, owners[0].StorageOperations, owners[1].StorageOperations)
			require.Equal(t, owners[0].StorageCrashEnumeration, owners[1].StorageCrashEnumeration)
			require.NotNil(t, owners[1].GoFilesystemCalls)
			require.Empty(t, owners[1].GoFilesystemCalls)
		})
	}
}

func TestStorageEvidencePartialCollectionAdmission(t *testing.T) {
	for _, test := range []struct {
		name, termination, message string
		stage                      byte
		enumerations               string
		identity                   string
		image                      []byte
		malformed                  string
		wantError                  bool
	}{
		{"enumeration", "invalid", "crash enumeration limits must be nonzero", 1, `{}`, "", nil, "", false},
		{"image", "capacity", "WASI capacity: storage-image-bytes", 2, `{"data":{"Complete":true}}`, "", nil, "", false},
		{"image-infrastructure", "infrastructure", "crash selection contains an unknown operation", 2, `{"data":{"Complete":true}}`, "", nil, "", false},
		{"success-terminal", "exit", "completed", 1, `{}`, "", nil, "", true},
		{"wrong-capacity", "capacity", "WASI capacity: transcript-bytes", 2, `{"data":{"Complete":true}}`, "", nil, "", true},
		{"complete-enumeration", "invalid", "failure", 1, `{"data":{"Complete":true}}`, "", nil, "", true},
		{"missing-enumeration", "capacity", "WASI capacity: storage-image-bytes", 2, `{}`, "", nil, "", true},
		{"image-identity", "invalid", "failure", 1, `{}`, strings.Repeat("a", 64), nil, "", true},
		{"image-bytes", "invalid", "failure", 1, `{}`, "", []byte{1}, "", true},
		{"unknown-stage", "invalid", "failure", 3, `{}`, "", nil, "", true},
		{"truncated", "invalid", "failure", 1, `{}`, "", nil, "truncated", true},
		{"corrupt", "invalid", "failure", 1, `{}`, "", nil, "corrupt", true},
		{"trailing", "invalid", "failure", 1, `{}`, "", nil, "trailing", true},
		{"count", "invalid", "failure", 1, `{}`, "", nil, "count", true},
		{"version", "invalid", "failure", 1, `{}`, "", nil, "version", true},
		{"stage-length", "invalid", "failure", 1, `{}`, "", nil, "stage-length", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			sections := validStorageSections()
			sections[2], sections[3] = []byte(test.enumerations), []byte(test.identity)
			sections = append(sections, []byte{test.stage})
			data := storageStateEnvelope(2, 6, sections...)
			switch test.malformed {
			case "truncated":
				data = data[:len(data)-1]
			case "corrupt":
				data[len(data)-1] ^= 1
			case "trailing":
				data = append(data, 0)
			case "count":
				data = storageStateEnvelope(2, 5, sections...)
			case "version":
				data = storageStateEnvelope(3, 6, sections...)
			case "stage-length":
				sections[5] = []byte{1, 2}
				data = storageStateEnvelope(2, 6, sections...)
			}
			sink := &frameSink{request: Request{Config: storageConfig(), StorageCrashLimits: &volume.CrashEnumerationLimits{}, ReturnStorageImage: true}, terminal: &resultFrame{Termination: test.termination, Message: test.message}}
			sink.helperEvidence.storageState, sink.helperEvidence.storageImage = data, test.image
			var result Result
			err := sink.collectHelperStorage(&result)
			if test.wantError {
				require.Error(t, err)
				require.Empty(t, result.StorageSnapshots)
			} else {
				require.NoError(t, err)
				require.Len(t, result.StorageSnapshots, 1)
				require.Len(t, result.StorageOperations["data"], 1)
			}
			require.Nil(t, result.StorageImage)
		})
	}
	t.Run("non-prefix-enumeration", func(t *testing.T) {
		config := storageConfig()
		config.Storage.Volumes = append(config.Storage.Volumes, volume.VolumeConfig{ID: "more", Path: "/other", CapacityBytes: 1024})
		sections := validStorageSections()
		var snapshots []volume.VolumeSnapshot
		require.NoError(t, json.Unmarshal(sections[0], &snapshots))
		encoded, err := json.Marshal(append(snapshots, volume.VolumeSnapshot{ID: "more"}))
		require.NoError(t, err)
		sections[0] = encoded
		sections[1] = []byte(`{"data":[],"more":[]}`)
		sections[2], sections[3] = []byte(`{"more":{"Complete":true}}`), nil
		sections = append(sections, []byte{1})
		sink := &frameSink{request: Request{Config: config, StorageCrashLimits: &volume.CrashEnumerationLimits{}}, terminal: &resultFrame{Termination: "invalid", Message: "failure"}}
		sink.helperEvidence.storageState = storageStateEnvelope(2, 6, sections...)
		var result Result
		require.EqualError(t, sink.collectHelperStorage(&result), "WASI invalid: storage evidence: enumeration failure prefix")
		require.Empty(t, result.StorageSnapshots)
	})
}

func TestStorageEvidenceLargeRefusedHistoryOwnerParity(t *testing.T) {
	body := `i32.const 4 i32.const 0 i32.const 0 i32.const 4 i32.const 1 i64.const 536870911 i64.const 0 i32.const 0 i32.const 100 call $open if unreachable end
 i32.const 200 i32.const 4096 i32.store i32.const 204 i32.const 1 i32.store
 loop $writes
 i32.const 100 i32.load i32.const 200 i32.const 1 i64.const 0 i32.const 240 call $pwrite if unreachable end
 local.get $i i32.const 1 i32.add local.tee $i i32.const 100000 i32.lt_u br_if $writes
 end`
	request := storageEvidenceRequest(t, body, `(data (i32.const 0) "file") (data (i32.const 4096) "x")`, 1)
	request.Config.Limits.FilesystemBytes = 64 << 10
	request.Config.Storage.Volumes[0].CapacityBytes = 64 << 10
	request.Config.Limits.Calls = 200000
	request.Config.Limits.TranscriptBytes = 128 << 20
	request.Config.Storage.Limits = volume.VolumeLimits{PendingOperations: 200000, Transitions: 200000}
	request.DigestTranscript = true
	var owners [2]Result
	for i := range owners {
		request.remoteModel = i == 0
		result, err := Run(t.Context(), request)
		require.NoError(t, err)
		t.Logf("owner remote=%v: termination=%s message=%s storage=%d", request.remoteModel, result.Termination, result.Message, len(result.StorageTranscript))
		require.Equal(t, "capacity", result.Termination)
		require.Equal(t, "WASI capacity: storage-image-bytes", result.Message)
		require.Nil(t, result.StorageImage)
		owners[i] = result
	}
	var operations [2][]byte
	for i := range owners {
		var err error
		operations[i], err = json.Marshal(owners[i].StorageOperations)
		require.NoError(t, err)
	}
	require.Greater(t, len(operations[0]), 256+18*(64<<10+1<<20))
	require.True(t, bytes.Equal(owners[0].StorageTranscript, owners[1].StorageTranscript), "storage transcript prefix differs")
	require.True(t, bytes.Equal(operations[0], operations[1]), "operation history prefix differs")
	require.Equal(t, owners[0].StorageSnapshots, owners[1].StorageSnapshots)
	require.Equal(t, owners[0].StorageCrashEnumeration, owners[1].StorageCrashEnumeration)
	require.NotNil(t, owners[1].GoFilesystemCalls)
	require.Empty(t, owners[1].GoFilesystemCalls)
	t.Logf("retained history JSON=%d bytes, previous image-derived state limit=%d", len(operations[0]), 256+18*(64<<10+1<<20))
}

func TestStorageEvidencePartialStreamRetainsIndependentCompleteBound(t *testing.T) {
	request := Request{Config: storageConfig(), ReturnStorageImage: true}
	request.Config.Limits.FilesystemBytes = 16 << 10
	request.Config.Storage.Volumes[0].CapacityBytes = 16 << 10
	request.Config.Limits.Calls, request.Config.Limits.Files = 1, 4
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprintf("partial=%v", partial), func(t *testing.T) {
			environment := newTestEnvironment(t, request.Config)
			environment.helperOwned, environment.namespaceOwned = true, true
			sink := &frameSink{request: request, environment: environment, started: &Started{}}
			header := storageStateEnvelope(1, 5, validStorageSections()...)[:16]
			if partial {
				header = storageStateEnvelope(2, 6, append(validStorageSections(), []byte{1})...)[:16]
			}
			send := func(data []byte) error {
				encoded, err := json.Marshal(struct {
					Type   string `json:"type"`
					Stream string `json:"stream"`
					Data   []byte `json:"data_base64"`
				}{"evidence", "storage_state", data})
				require.NoError(t, err)
				frame, err := newValidatedFrame(encoded)
				require.NoError(t, err)
				return sink.receiveHelperEvidence(frame)
			}
			// Version/count bytes may cross evidence-frame boundaries.
			require.NoError(t, send(header[:10]))
			require.NoError(t, send(header[10:]))
			limit := storageStateLimit(request)
			sink.helperEvidence.storageState = append(sink.helperEvidence.storageState, make([]byte, int(limit)-16)...)
			err := send([]byte{1})
			if partial {
				require.NoError(t, err, "explicit partial evidence must retain permitted history beyond the complete-image allowance")
				partialLimit := storagePartialStateLimit(request)
				require.Equal(t, uint64(21_141_076), partialLimit)
				sink.helperEvidence.storageState = append(sink.helperEvidence.storageState, make([]byte, int(partialLimit)-len(sink.helperEvidence.storageState)-1)...)
				require.NoError(t, send([]byte{1}))
				require.Equal(t, partialLimit, uint64(len(sink.helperEvidence.storageState)))
				require.EqualError(t, send([]byte{1}), "WASI capacity: helper evidence bytes")
				require.Equal(t, partialLimit, uint64(len(sink.helperEvidence.storageState)))
			} else {
				require.EqualError(t, err, "WASI capacity: helper evidence bytes")
			}
		})
	}
}

func TestStorageEvidencePartialHeaderBudgetRecognition(t *testing.T) {
	request := Request{Config: storageConfig()}
	request.Config.Limits.FilesystemBytes = 16 << 10
	request.Config.Storage.Volumes[0].CapacityBytes = 16 << 10
	request.Config.Limits.Calls, request.Config.Limits.Files = 1, 4
	header := storageStateEnvelope(2, 6, append(validStorageSections(), []byte{1})...)[:16]
	for length := 0; length < len(header); length++ {
		require.Equal(t, uint64(19_169_536), storageStateStreamLimit(request, header[:length], nil), "short prefix=%d", length)
	}
	for split := 0; split <= len(header); split++ {
		require.Equal(t, uint64(21_141_076), storageStateStreamLimit(request, header[:split], header[split:]), "split=%d", split)
	}
	for _, test := range []struct {
		name           string
		version, count uint32
		badMagic       bool
	}{{"complete", 1, 5, false}, {"version", 3, 6, false}, {"count", 2, 5, false}, {"magic", 2, 6, true}} {
		t.Run(test.name, func(t *testing.T) {
			data := storageStateEnvelope(test.version, test.count, append(validStorageSections(), []byte{1})...)[:16]
			if test.badMagic {
				data[0] ^= 1
			}
			require.Equal(t, uint64(19_169_536), storageStateStreamLimit(request, nil, data))
		})
	}
	request.Config.Limits.Calls = math.MaxUint64
	require.Equal(t, uint64(math.MaxUint64), storagePartialStateLimit(request))
	request.Config.Limits.Calls, request.Config.Limits.Files = 1, math.MaxUint64
	require.Equal(t, uint64(math.MaxUint64), storagePartialStateLimit(request))
	require.Equal(t, uint64(7), storageStateBudgetAdd(7, 0, math.MaxUint64))
	require.Equal(t, uint64(math.MaxUint64), storageStateBudgetAdd(math.MaxUint64-1, 1, 1))
	require.Equal(t, uint64(math.MaxUint64), storageStateBudgetAdd(math.MaxUint64-1, 1, 2))
}

func TestStorageEvidencePartialMetadataScalarBounds(t *testing.T) {
	hash := "sha256:" + strings.Repeat("f", 64)
	operation := volume.Operation{ID: math.MaxUint64, Kind: "namespace", Inode: math.MaxUint64, Offset: math.MaxUint64, Bytes: math.MaxUint64, PayloadSHA256: hash, EffectSHA256: hash}
	encoded, err := json.Marshal(operation)
	require.NoError(t, err)
	require.LessOrEqual(t, len(encoded)+1, 361)
	entry := volume.SnapshotEntry{Mode: math.MaxUint32, Kind: "directory", ModTime: math.MinInt64, Size: math.MaxUint64, DataSHA256: hash}
	encoded, err = json.Marshal(entry)
	require.NoError(t, err)
	require.LessOrEqual(t, len(encoded)+1, 195)
	operation.Path, operation.Destination = strings.Repeat("\x01", 4*volume.MaximumPathBytes), strings.Repeat("\x01", 4*volume.MaximumPathBytes)
	operation.Dependencies = make([]uint64, 2*4+3)
	for i := range operation.Dependencies {
		operation.Dependencies[i] = math.MaxUint64
	}
	encoded, err = json.Marshal(operation)
	require.NoError(t, err)
	require.LessOrEqual(t, len(encoded)+1, 361+2*6*4*volume.MaximumPathBytes+21*(2*4+3))
	entry.Path = operation.Path
	encoded, err = json.Marshal(entry)
	require.NoError(t, err)
	require.LessOrEqual(t, len(encoded)+1, 195+6*4*volume.MaximumPathBytes)
}

func TestStorageEvidenceEscapedMetadataOwnerParity(t *testing.T) {
	var data strings.Builder
	for i := range 50 {
		fmt.Fprintf(&data, `(data (i32.const %d) "%s%02d")`, 4096+4096*i, strings.Repeat(`\01`, 3998), i)
	}
	body := `loop $files
 i32.const 4 i32.const 0 i32.const 4096 local.get $i i32.const 4096 i32.mul i32.add i32.const 4000 i32.const 1 i64.const 536870911 i64.const 0 i32.const 0 i32.const 100 call $open if unreachable end
 local.get $i i32.const 1 i32.add local.tee $i i32.const 50 i32.lt_u br_if $files
 end
 i32.const 4 call $sync if unreachable end`
	request := storageEvidenceRequest(t, body, data.String(), 4)
	request.Config.Limits.FilesystemBytes = 16 << 10
	request.Config.Limits.TranscriptBytes = 8 << 20
	request.Config.Storage.Volumes[0].CapacityBytes = 16 << 10
	var owners [2]Result
	for i := range owners {
		request.remoteModel = i == 0
		result, err := Run(t.Context(), request)
		require.NoError(t, err)
		requireStorageGuestExit(t, result)
		owners[i] = result
	}
	snapshots, err := json.Marshal(owners[1].StorageSnapshots)
	require.NoError(t, err)
	operations, err := json.Marshal(owners[1].StorageOperations)
	require.NoError(t, err)
	require.Greater(t, uint64(len(snapshots)+len(operations)), 2*storageImageLimit(request.Config))
	require.Empty(t, sameStorageEvidence(owners[0], owners[1]))
	require.NotNil(t, owners[1].GoFilesystemCalls)
	require.Empty(t, owners[1].GoFilesystemCalls)
	t.Logf("escaped metadata JSON=%d, old state bound=%d, image=%d", len(snapshots)+len(operations), 2*storageImageLimit(request.Config), owners[1].StorageImage.Bytes())
}

func TestStorageEvidenceStateBudgetSaturatesAndCountsVolumes(t *testing.T) {
	request := Request{Config: storageConfig()}
	base := storageStateLimit(request)
	request.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 1, Operations: 1, Depth: 1, Bytes: 1, Work: 1}
	one := storageStateLimit(request)
	require.Greater(t, one, base)
	request.Config.Storage.Volumes = append(request.Config.Storage.Volumes, volume.VolumeConfig{ID: "more", Path: "/other", CapacityBytes: 1})
	request.Config.WritableDirectories = append(request.Config.WritableDirectories, "/other")
	require.Equal(t, base+2*(one-base), storageStateLimit(request))
	request.StorageCrashLimits.Bytes, request.StorageCrashLimits.States, request.StorageCrashLimits.Work = math.MaxUint64, math.MaxUint64, math.MaxUint64
	require.Equal(t, uint64(math.MaxUint64), storageStateLimit(request))

	request = storageEvidenceRequest(t, "", "", 1)
	request.StorageCrashLimits = &volume.CrashEnumerationLimits{States: math.MaxUint64, Operations: math.MaxUint64, Depth: math.MaxUint64, Bytes: math.MaxUint64, Work: math.MaxUint64}
	var owners [2]Result
	for i := range owners {
		request.remoteModel = i == 0
		result, err := Run(t.Context(), request)
		require.NoError(t, err)
		requireStorageGuestExit(t, result)
		owners[i] = result
	}
	require.Empty(t, sameStorageEvidence(owners[0], owners[1]))
}

func TestStorageEvidenceStreamAdmissionKeepsIndependentBounds(t *testing.T) {
	request := Request{Config: storageConfig(), ReturnStorageImage: true}
	request.Config.Limits.FilesystemBytes = 16 << 10
	request.Config.Storage.Volumes[0].CapacityBytes = 16 << 10
	for _, stream := range []string{"storage_state", "storage_image"} {
		t.Run(stream, func(t *testing.T) {
			environment := newTestEnvironment(t, request.Config)
			environment.helperOwned, environment.namespaceOwned = true, true
			sink := &frameSink{request: request, environment: environment, started: &Started{}}
			limit, target := storageStateLimit(request), &sink.helperEvidence.storageState
			if stream == "storage_image" {
				limit, target = storageImageLimit(request.Config), &sink.helperEvidence.storageImage
			}
			*target = make([]byte, limit-1)
			data, err := json.Marshal(struct {
				Type   string `json:"type"`
				Stream string `json:"stream"`
				Data   []byte `json:"data_base64"`
			}{"evidence", stream, []byte{1}})
			require.NoError(t, err)
			frame, err := newValidatedFrame(data)
			require.NoError(t, err)
			require.NoError(t, sink.receiveHelperEvidence(frame))
			require.Equal(t, limit, uint64(len(*target)))
			require.EqualError(t, sink.receiveHelperEvidence(frame), "WASI capacity: helper evidence bytes")
			require.Equal(t, limit, uint64(len(*target)))
		})
	}
}

func storageEnumerationRequest(t *testing.T, recoverImage bool) Request {
	t.Helper()
	body := `i32.const 4 i32.const 0 i32.const 0 i32.const 4 i32.const 1 i64.const 536870911 i64.const 0 i32.const 0 i32.const 100 call $open if unreachable end
 i32.const 200 i32.const 4096 i32.store i32.const 204 i32.const 65536 i32.store
 i32.const 100 i32.load i32.const 200 i32.const 1 i32.const 240 call $write if unreachable end
 i32.const 100 i32.load call $sync if unreachable end
 i32.const 4 call $sync if unreachable end
 loop $dirs
 i32.const 4 i32.const 20 local.get $i i32.const 2 i32.mul i32.add i32.const 2 call $mkdir if unreachable end
 local.get $i i32.const 1 i32.add local.tee $i i32.const 9 i32.lt_u br_if $dirs
 end`
	if recoverImage {
		body = `i32.const 4 i32.const 0 i32.const 0 i32.const 4 i32.const 0 i64.const 536870911 i64.const 0 i32.const 0 i32.const 100 call $open if unreachable end
 i32.const 200 i32.const 4096 i32.store i32.const 204 i32.const 1 i32.store
 i32.const 100 i32.load i32.const 200 i32.const 1 i64.const 0 i32.const 240 call $read if unreachable end
 i32.const 1 i32.const 200 i32.const 1 i32.const 240 call $write if unreachable end`
	}
	request := storageEvidenceRequest(t, body, `(data (i32.const 0) "file") (data (i32.const 20) "d0d1d2d3d4d5d6d7d8") (data (i32.const 4096) "x")`, 2)
	if !recoverImage {
		request.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 512, Operations: 32, Depth: 32, Bytes: 64 << 20, Work: 1 << 20}
	}
	return request
}

func storageEvidenceRequest(t *testing.T, body, data string, pages uint64) Request {
	t.Helper()
	wat := fmt.Sprintf(`(module
 (import "wasi_snapshot_preview1" "path_open" (func $open (param i32 i32 i32 i32 i32 i64 i64 i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "fd_sync" (func $sync (param i32) (result i32)))
 (import "wasi_snapshot_preview1" "path_create_directory" (func $mkdir (param i32 i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "fd_pread" (func $read (param i32 i32 i32 i64 i32) (result i32)))
 (import "wasi_snapshot_preview1" "fd_pwrite" (func $pwrite (param i32 i32 i32 i64 i32) (result i32)))
 (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "decision" (func (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "observation" (func (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "idle" (func (param i32 i32) (result i32)))
 (memory (export "memory") %d) %s
 (func (export "_start") (local $i i32)
 i32.const 512 i32.const 16 call $config drop
 %s
 i32.const 512 i32.const 16 call $finish drop))`, pages, data, body)
	imports := []Import{
		{Module: "wasi_snapshot_preview1", Name: "path_open", Params: []string{"i32", "i32", "i32", "i32", "i32", "i64", "i64", "i32", "i32"}, Results: []string{"i32"}},
		{Module: "wasi_snapshot_preview1", Name: "fd_write", Params: []string{"i32", "i32", "i32", "i32"}, Results: []string{"i32"}},
		{Module: "wasi_snapshot_preview1", Name: "fd_sync", Params: []string{"i32"}, Results: []string{"i32"}},
		{Module: "wasi_snapshot_preview1", Name: "path_create_directory", Params: []string{"i32", "i32", "i32"}, Results: []string{"i32"}},
		{Module: "wasi_snapshot_preview1", Name: "fd_pread", Params: []string{"i32", "i32", "i32", "i64", "i32"}, Results: []string{"i32"}},
		{Module: "wasi_snapshot_preview1", Name: "fd_pwrite", Params: []string{"i32", "i32", "i32", "i64", "i32"}, Results: []string{"i32"}},
	}
	for _, name := range []string{"config", "decision", "observation", "finish", "idle"} {
		imports = append(imports, Import{Module: "gomad_wasm_v1", Name: name, Params: []string{"i32", "i32"}, Results: []string{"i32"}})
	}
	request := fixtureRequest(t, wat, imports)
	request.Config = storageConfig()
	request.Config.Profile = CooperativeProfile
	request.Config.Limits.FilesystemBytes = 256 << 10
	request.Config.Limits.TranscriptBytes = 1 << 20
	request.Config.Storage.Volumes[0].CapacityBytes = 256 << 10
	request.InitialMemoryPages = pages
	request.ReturnStorageImage = true
	request.Runtime = &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: 8192, ExecutionIdentity: choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte(wat)), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}}}
	return request
}

func TestStorageEvidenceRunRefusesDamagedHelperStreams(t *testing.T) {
	request := storageGuestOperation(storageSQLiteRequest(t), "setup", "WAL", nil)
	request.ReturnStorageImage = true
	root := t.TempDir()
	wirePath := filepath.Join(root, "frames")
	helperPath := filepath.Join(root, "helper")
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }
	script := "#!/bin/sh\n" + quote(request.HelperPath) + " | /usr/bin/tee " + quote(wirePath) + "\n"
	require.NoError(t, os.WriteFile(helperPath, []byte(script), 0o700))
	request.HelperPath, request.HelperSHA256 = helperPath, fileHash(t, helperPath)
	recorded := runStorageGuest(t, request)
	requireStorageGuestExit(t, recorded)
	require.NotNil(t, recorded.GoFilesystemCalls)
	require.Empty(t, recorded.GoFilesystemCalls)
	wire, err := os.ReadFile(wirePath)
	require.NoError(t, err)
	frames := bytes.Split(bytes.TrimSpace(wire), []byte{'\n'})
	require.Greater(t, len(frames), 2)

	for _, damage := range []string{"none", "truncated", "corrupt", "trailing"} {
		t.Run(damage, func(t *testing.T) {
			var output bytes.Buffer
			changed := false
			for _, frame := range frames[1:] {
				var evidence struct {
					Type   string `json:"type"`
					Stream string `json:"stream"`
					Data   []byte `json:"data_base64"`
				}
				var header struct {
					Type string `json:"type"`
				}
				require.NoError(t, json.Unmarshal(frame, &header))
				if header.Type == "evidence" {
					require.NoError(t, json.Unmarshal(frame, &evidence))
					if evidence.Stream == "storage_state" && !changed {
						require.Greater(t, len(evidence.Data), 1)
						switch damage {
						case "truncated":
							evidence.Data = evidence.Data[:len(evidence.Data)-1]
						case "corrupt":
							evidence.Data[len(evidence.Data)-1] ^= 1
						case "trailing":
							evidence.Data = append(evidence.Data, 0)
						}
						changed = true
						frame, err = json.Marshal(evidence)
						require.NoError(t, err)
					}
				}
				output.Write(frame)
				output.WriteByte('\n')
			}
			require.True(t, changed, "actual helper emitted no storage-state envelope")
			// Drain the bootstrap, authorization and namespace before replaying
			// the recorded terminal streams, as the real helper does.
			script := "#!/bin/sh\nprintf '%s\\n' " + quote(string(frames[0])) + "\nwhile IFS= read -r frame; do\ncase \"$frame\" in\n*namespace_inputs_end*) break ;;\nesac\ndone\nprintf '%s' " + quote(output.String()) + "\n"
			require.NoError(t, os.WriteFile(helperPath, []byte(script), 0o700))
			request.HelperSHA256 = fileHash(t, helperPath)
			result, err := Run(t.Context(), request)
			require.NoError(t, err)
			require.True(t, result.Reaped)
			if damage == "none" {
				requireStorageGuestExit(t, result)
				require.Empty(t, sameStorageEvidence(recorded, result))
				return
			}
			require.Equal(t, "invalid", result.Termination)
			require.Contains(t, result.Message, "WASI invalid: storage evidence: envelope length or digest")
			require.Nil(t, result.StorageImage)
			require.Empty(t, result.StorageSnapshots)
		})
	}
}
