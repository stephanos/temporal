package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func TestSQLiteCrashCatalogueUsesOriginalEnumeration(t *testing.T) {
	fs := volume.New()
	model := &StorageModel{Volumes: []volume.VolumeConfig{{ID: "sqlite", Path: "/volume", CapacityBytes: 4 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 10000, Transitions: 100000}, Faults: []StorageFault{{Name: "acknowledged-crash", Resource: "/volume/crash-trigger", Operation: "open", Occurrence: 1, Action: "crash-before"}}, SQLiteLockDatabases: []string{"/volume/db"}}
	require.NoError(t, fs.ConfigureVolumes(model.Volumes, model.Limits))
	file, err := fs.Open("/volume/db", volume.OpenFlags{Create: true, Write: true}, 0600)
	require.NoError(t, err)
	_, err = file.WriteAt([]byte("old-data"), 0)
	require.NoError(t, err)
	require.NoError(t, file.Sync())
	directory, err := fs.Open("/volume", volume.OpenFlags{Read: true}, 0)
	require.NoError(t, err)
	require.NoError(t, directory.Sync())
	_, err = file.WriteAt([]byte("new"), 0)
	require.NoError(t, err)
	require.NoError(t, file.Truncate(12))
	modelJSON, err := json.Marshal(model)
	require.NoError(t, err)
	image, _, err := encodeStorageImage(fs, modelJSON, nil, 9<<20)
	require.NoError(t, err)
	modelIdentity := fmt.Sprintf("sha256:%x", sha256.Sum256(modelJSON))
	enumeration, err := fs.EnumerateCrashStates("sqlite", volume.CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}, nil)
	require.NoError(t, err)
	require.Len(t, enumeration.States, 4)
	operations := fs.Operations()
	context := StorageCrashContext{PendingSHA256: fs.VolumeSnapshots()[0].PendingSHA256, Operations: append([]volume.Operation{}, operations["sqlite"][len(operations["sqlite"])-2:]...)}
	sections := make([][]byte, 6)
	marshal := func(v any) []byte { data, err := json.Marshal(v); require.NoError(t, err); return data }
	sections[0] = marshal(fs.VolumeSnapshots())
	sections[1] = marshal(operations)
	sections[2] = marshal(map[string]volume.CrashEnumeration{"sqlite": enumeration})
	sections[3] = []byte(fmt.Sprintf("%064x", 1))
	sections[4] = make([]byte, 40)
	sum := sha256.Sum256(nil)
	copy(sections[4][8:], sum[:])
	sections[5] = marshal(map[string]StorageCrashContext{"sqlite": context})
	envelope := func() []byte {
		var out bytes.Buffer
		out.WriteString(storageStateMagic)
		require.NoError(t, binary.Write(&out, binary.LittleEndian, uint32(3)))
		require.NoError(t, binary.Write(&out, binary.LittleEndian, uint32(6)))
		for i, section := range sections {
			require.NoError(t, binary.Write(&out, binary.LittleEndian, uint32(i+1)))
			require.NoError(t, binary.Write(&out, binary.LittleEndian, uint64(len(section))))
			out.Write(section)
		}
		body := append([]byte(nil), out.Bytes()...)
		require.NoError(t, binary.Write(&out, binary.LittleEndian, uint64(len(body))))
		hash := sha256.Sum256(body)
		out.Write(hash[:])
		return out.Bytes()
	}
	state := envelope()
	originalCatalogue := sections[2]
	got, observed, err := ProjectSQLiteCrashCatalogue(state, nil, image, modelIdentity)
	require.NoError(t, err)
	require.Equal(t, enumeration, got)
	require.True(t, context.Equal(observed))
	t.Run("forged-catalogue", func(t *testing.T) {
		enumeration.States[0].Entries[0].Mode ^= 1
		sections[2] = marshal(map[string]volume.CrashEnumeration{"sqlite": enumeration})
		_, _, err := ProjectSQLiteCrashCatalogue(envelope(), nil, image, modelIdentity)
		require.ErrorContains(t, err, "original volume model")
		sections[2] = originalCatalogue
	})
	t.Run("forged-context-and-history", func(t *testing.T) {
		context.Operations[0].EffectSHA256 = fmt.Sprintf("sha256:%064x", 2)
		operations["sqlite"][len(operations["sqlite"])-2] = context.Operations[0]
		sections[1] = marshal(operations)
		sections[5] = marshal(map[string]StorageCrashContext{"sqlite": context})
		_, _, err := ProjectSQLiteCrashCatalogue(envelope(), nil, image, modelIdentity)
		require.ErrorContains(t, err, "original volume model")
	})
	t.Run("missing-image", func(t *testing.T) {
		_, _, err := ProjectSQLiteCrashCatalogue(state, nil, nil, modelIdentity)
		require.Error(t, err)
	})
	t.Run("wrong-model", func(t *testing.T) {
		_, _, err := ProjectSQLiteCrashCatalogue(state, nil, image, fmt.Sprintf("sha256:%064x", 3))
		require.ErrorContains(t, err, "model identity")
	})
}
