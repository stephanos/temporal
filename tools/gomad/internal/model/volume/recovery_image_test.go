package volume

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCrashRecoveryImageAdmissionCorpus(t *testing.T) {
	data, err := os.ReadFile("../wasi/testdata/volume-conformance/vectors/image-admission.json")
	require.NoError(t, err)
	var corpus struct {
		Cases []struct {
			Name    string `json:"name"`
			Storage struct {
				Volumes []VolumeConfig `json:"volumes"`
				Limits  VolumeLimits   `json:"limits"`
				Locks   []string       `json:"sqlite_lock_databases"`
			} `json:"storage"`
			Data    []byte `json:"image_base64"`
			Digest  string `json:"image_digest"`
			Model   string `json:"model"`
			Maximum uint64 `json:"maximum"`
			Error   string `json:"error"`
		} `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(data, &corpus))
	require.NotEmpty(t, corpus.Cases)
	for _, test := range corpus.Cases {
		t.Run(test.Name, func(t *testing.T) {
			_, err := DecodeCrashRecoveryImage(test.Data, test.Digest, test.Model, test.Storage.Volumes, test.Storage.Limits, test.Storage.Locks, test.Maximum)
			if test.Error == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.Error)
			}
		})
	}
}

func TestCrashRecoveryImagePreservesOriginalEnumeration(t *testing.T) {
	fs := New()
	configs := []VolumeConfig{{ID: "sqlite", Path: "/volume", CapacityBytes: 4 << 20}}
	limits := VolumeLimits{PendingOperations: 10000, Transitions: 100000}
	require.NoError(t, fs.ConfigureVolumes(configs, limits))
	file, err := fs.Open("/volume/db", OpenFlags{Write: true, Create: true}, 0600)
	require.NoError(t, err)
	_, err = file.WriteAt([]byte("persisted"), 0)
	require.NoError(t, err)
	require.NoError(t, file.Sync())
	directory, err := fs.Open("/volume", OpenFlags{Read: true}, 0)
	require.NoError(t, err)
	require.NoError(t, directory.Sync())
	_, err = file.WriteAt([]byte("new"), 2)
	require.NoError(t, err)
	require.NoError(t, file.Truncate(12))
	model := fmt.Sprintf("sha256:%064x", 1)
	data := crashImageTestEncode(fs.ConformanceState(), model)
	imported, err := DecodeCrashRecoveryImage(data, fmt.Sprintf("%x", sha256.Sum256(data)), model, configs, limits, nil, 9<<20)
	require.NoError(t, err)
	require.Equal(t, crashImageTestEncode(fs.ConformanceState(), model), crashImageTestEncode(imported.ConformanceState(), model))
	require.Equal(t, fs.Operations(), imported.Operations())
	require.Equal(t, fs.VolumeSnapshots()[0].PendingSHA256, imported.VolumeSnapshots()[0].PendingSHA256)
	bound := CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}
	want, err := fs.EnumerateCrashStates("sqlite", bound, nil)
	require.NoError(t, err)
	got, err := imported.EnumerateCrashStates("sqlite", bound, nil)
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.True(t, got.Complete)
	require.Greater(t, len(got.States), 1)
	data[len(data)-1] ^= 1
	require.Equal(t, crashImageTestEncode(fs.ConformanceState(), model), crashImageTestEncode(imported.ConformanceState(), model))
}

func TestCrashRecoveryImageRefusesMalformedPendingState(t *testing.T) {
	fs := New()
	configs := []VolumeConfig{{ID: "sqlite", Path: "/volume", CapacityBytes: 4 << 20}}
	limits := VolumeLimits{PendingOperations: 10000, Transitions: 100000}
	require.NoError(t, fs.ConfigureVolumes(configs, limits))
	file, err := fs.Open("/volume/db", OpenFlags{Write: true, Create: true}, 0600)
	require.NoError(t, err)
	_, err = file.WriteAt([]byte("payload"), 0)
	require.NoError(t, err)
	model := fmt.Sprintf("sha256:%064x", 1)
	data := crashImageTestEncode(fs.ConformanceState(), model)
	t.Run("pending-limit", func(t *testing.T) {
		small := limits
		small.PendingOperations = 1
		_, err := DecodeCrashRecoveryImage(data, fmt.Sprintf("%x", sha256.Sum256(data)), model, configs, small, nil, 9<<20)
		require.EqualError(t, err, "recovery operation limits")
	})
	t.Run("history-limit", func(t *testing.T) {
		small := limits
		small.Transitions = 1
		_, err := DecodeCrashRecoveryImage(data, fmt.Sprintf("%x", sha256.Sum256(data)), model, configs, small, nil, 9<<20)
		require.EqualError(t, err, "recovery operation limits")
	})
	for _, test := range []struct {
		name    string
		mutate  func(*ConformanceState)
		message string
	}{
		{"pending-effect-forged", func(s *ConformanceState) { s.Volumes[0].Pending[0].Effect.Mode++ }, "recovery pending operation"},
		{"pending-order", func(s *ConformanceState) {
			s.Volumes[0].Pending[0], s.Volumes[0].Pending[1] = s.Volumes[0].Pending[1], s.Volumes[0].Pending[0]
		}, "recovery state ordering"},
		{"future-dependency", func(s *ConformanceState) { s.Volumes[0].History[0].Dependencies = []uint64{100} }, "recovery operation dependencies"},
		{"missing-alloc-object", func(s *ConformanceState) { s.Volumes[0].History[0].Effect.Object = nil }, "recovery persistence effect"},
		{"overflowing-effect", func(s *ConformanceState) { s.Volumes[0].History[0].Effect.Offset = ^uint64(0) }, "recovery persistence effect"},
		{"forged-last", func(s *ConformanceState) { s.Volumes[0].Last[0].Operation++ }, "recovery pending operation stores"},
		{"unsupported-kind", func(s *ConformanceState) { s.Volumes[0].Persisted.Objects[0].Kind = 3 }, "recovery object kind"},
		{"invalid-path", func(s *ConformanceState) { s.Volumes[0].Persisted.Paths[0].Path = "/../escape" }, "recovery durable tree"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := fs.ConformanceState()
			test.mutate(&state)
			data := crashImageTestEncode(state, model)
			_, err := DecodeCrashRecoveryImage(data, fmt.Sprintf("%x", sha256.Sum256(data)), model, configs, limits, nil, 9<<20)
			require.EqualError(t, err, test.message)
		})
	}
}

// This test writer follows the published image field order, independently of the decoder.
func crashImageTestEncode(state ConformanceState, model string) []byte {
	var payload bytes.Buffer
	var write func(reflect.Value)
	number := func(n uint64) {
		requireWrite := binary.Write(&payload, binary.LittleEndian, n)
		if requireWrite != nil {
			panic(requireWrite)
		}
	}
	write = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				write(v.Field(i))
			}
		case reflect.String:
			number(uint64(v.Len()))
			payload.WriteString(v.String())
		case reflect.Slice:
			number(uint64(v.Len()))
			for i := 0; i < v.Len(); i++ {
				if v.Type().Elem().Kind() == reflect.Uint8 {
					payload.WriteByte(byte(v.Index(i).Uint()))
				} else {
					write(v.Index(i))
				}
			}
		case reflect.Pointer:
			if v.IsNil() {
				number(0)
			} else {
				number(1)
				write(v.Elem())
			}
		case reflect.Int64:
			number(uint64(v.Int()))
		default:
			number(v.Uint())
		}
	}
	write(reflect.ValueOf(state))
	stateBytes := append([]byte(nil), payload.Bytes()...)
	payload.Reset()
	write(reflect.ValueOf(model))
	modelBytes := append([]byte(nil), payload.Bytes()...)
	payload.Reset()
	payload.WriteString("GMVOLIMG")
	if err := binary.Write(&payload, binary.LittleEndian, uint32(1)); err != nil {
		panic(err)
	}
	if err := binary.Write(&payload, binary.LittleEndian, uint32(3)); err != nil {
		panic(err)
	}
	for i, section := range [][]byte{modelBytes, make([]byte, 8), stateBytes} {
		if err := binary.Write(&payload, binary.LittleEndian, uint32(i+1)); err != nil {
			panic(err)
		}
		number(uint64(len(section)))
		payload.Write(section)
	}
	return payload.Bytes()
}
