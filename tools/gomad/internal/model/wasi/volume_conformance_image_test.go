package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func TestVolumeImageIdentityCorpus(t *testing.T) {
	config := storageConfig()
	e := newTestEnvironment(t, config)
	fd := openPath(t, e, 4, "db", 1, rightsAll)
	writeStorage(t, e, fd, 0, "durable")
	call(t, e, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd))
	call(t, e, "fd_sync", `{"fd":4}`)
	image, err := e.CrashStorage(nil)
	require.NoError(t, err)
	type identityCase struct {
		Name        string            `json:"name"`
		Data        []byte            `json:"image_base64"`
		Digest      string            `json:"image_digest"`
		NextInode   uint64            `json:"next_inode"`
		Occurrences map[string]uint64 `json:"occurrences"`
	}
	var cases []identityCase
	capture := func(name string, image *StorageImage) {
		data, digest, err := volumeConformanceEncodeImage(image, 1<<20)
		require.NoError(t, err)
		require.True(t, bytes.HasPrefix(data, []byte("GMVOLIMG\x01\x00\x00\x00\x03\x00\x00\x00")))
		require.Equal(t, fmt.Sprintf("%x", sha256.Sum256(data)), digest)
		_, _, err = volumeConformanceEncodeImage(image, uint64(len(data)-1))
		require.Error(t, err)
		cases = append(cases, identityCase{name, data, digest, image.fs.ConformanceState().NextInode, cloneStorageOccurrences(image.occurrences)})
	}
	capture("original", image)
	require.Equal(t, image.data, cases[0].Data)
	require.Equal(t, image.Identity(), cases[0].Digest)
	cloned := &StorageImage{fs: image.fs.CloneDurable(), model: slices.Clone(image.model), occurrences: cloneStorageOccurrences(image.occurrences)}
	capture("next-inode-continuation", cloned)
	require.NotEqual(t, cases[0].NextInode, cases[1].NextInode)
	require.NotEqual(t, cases[0].Digest, cases[1].Digest)
	image.occurrences["/workspace/db\x00write"]++
	capture("fault-occurrence-continuation", image)
	require.NotEqual(t, cases[0].Digest, cases[2].Digest)
	require.Equal(t, cases[0].NextInode, cases[2].NextInode)
	data, err := json.MarshalIndent(cases, "", "  ")
	require.NoError(t, err)
	data = append(data, '\n')
	name := filepath.Join("testdata", "volume-conformance", "vectors", "image-identity.json")
	if *volumeConformanceOutput != "" {
		name = filepath.Join(*volumeConformanceOutput, "vectors", "image-identity.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o755))
		require.NoError(t, os.WriteFile(name, data, 0o644))
		return
	}
	want, err := os.ReadFile(name)
	require.NoError(t, err)
	require.Equal(t, string(want), string(data))
}

type volumeImageAdmissionCase struct {
	Name               string       `json:"name"`
	Storage            StorageModel `json:"storage"`
	Data               []byte       `json:"image_base64"`
	Digest             string       `json:"image_digest"`
	Model              string       `json:"model"`
	Maximum            uint64       `json:"maximum"`
	Error              string       `json:"error"`
	RecoveredNextInode *uint64      `json:"recovered_next_inode,omitempty"`
	RecoveredTempMode  *uint32      `json:"recovered_temp_mode,omitempty"`
	RefuseAllocation   bool         `json:"refuse_allocation,omitempty"`
}

// Inputs originate in the Go encoder. Expected refusals pin the Rust decoder
// contract: the Go owner deliberately has no decoder for helper-owned images.
func TestVolumeImageAdmissionCorpus(t *testing.T) {
	storage := volumeCoreStorage()
	fs := volume.New()
	require.NoError(t, fs.ConfigureVolumes(storage.Volumes, storage.Limits))
	file, err := fs.Open("/workspace/db", volume.OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	require.NoError(t, err)
	_, err = file.WriteAt([]byte("durable"), 0)
	require.NoError(t, err)
	require.NoError(t, file.Sync())
	directory, err := fs.Open("/workspace", volume.OpenFlags{Read: true}, 0)
	require.NoError(t, err)
	require.NoError(t, directory.Sync())
	modelJSON, err := json.Marshal(storage)
	require.NoError(t, err)
	bytes, digest, err := encodeStorageImage(fs, modelJSON, nil, 1<<20)
	require.NoError(t, err)
	modelHash := sha256.Sum256(modelJSON)
	model := fmt.Sprintf("sha256:%x", modelHash)
	var cases []volumeImageAdmissionCase
	add := func(name string, data []byte, expected string) {
		cases = append(cases, volumeImageAdmissionCase{Name: name, Storage: storage, Data: data, Digest: fmt.Sprintf("%x", sha256.Sum256(data)), Model: model, Maximum: 1 << 20, Error: expected})
	}
	add("authentic-go-image", bytes, "")
	for _, offset := range []int{0, 8, 12, 16} {
		changed := slices.Clone(bytes)
		changed[offset] ^= 0xff
		expected := "recovery image format"
		if offset == 16 {
			expected = "recovery image sections"
		}
		add(fmt.Sprintf("invalid-frame-offset-%d", offset), changed, expected)
	}
	changed := slices.Clone(bytes)
	binary.LittleEndian.PutUint64(changed[20:28], ^uint64(0))
	add("impossible-section-length", changed, "truncated recovery image")
	add("trailing-byte", append(slices.Clone(bytes), 0), "recovery image trailing bytes")
	add("truncated", slices.Clone(bytes[:len(bytes)-1]), "truncated recovery image")
	changed = slices.Clone(bytes)
	changed[len(changed)-1] ^= 1
	add("tampered-digest", changed, "recovery image digest mismatch")
	cases[len(cases)-1].Digest = digest
	add("model-mismatch", bytes, "recovery storage model identity")
	cases[len(cases)-1].Model = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	add("exact-image-capacity-refused", bytes, "capacity:image-bytes")
	cases[len(cases)-1].Maximum = uint64(len(bytes) - 1)
	add("volume-config-mismatch", bytes, "recovery volume configuration")
	cases[len(cases)-1].Storage.Volumes = slices.Clone(storage.Volumes)
	cases[len(cases)-1].Storage.Volumes[0].CapacityBytes++
	state := fs.ConformanceState()
	require.Equal(t, bytes, volumeImageMutateState(bytes, state))
	for _, counter := range []uint64{^uint64(0), ^uint64(0) - 1, 0, 1, 4} {
		changed := state
		changed.NextInode = counter
		add(fmt.Sprintf("invalid-next-inode-%d", counter), volumeImageMutateState(bytes, changed), "recovery filesystem counters")
	}
	for _, directory := range []bool{false, true} {
		changed := fs.ConformanceState()
		v := &changed.Volumes[0]
		inode := v.Persisted.Paths[0].Inode
		if !directory {
			inode = v.Persisted.Paths[1].Inode
		}
		v.Persisted.Paths = append(v.Persisted.Paths, volume.ConformancePath{Path: "/alias", Inode: inode})
		slices.SortFunc(v.Persisted.Paths, func(a, b volume.ConformancePath) int {
			if a.Path < b.Path {
				return -1
			}
			if a.Path > b.Path {
				return 1
			}
			return 0
		})
		add(fmt.Sprintf("path-inode-alias-directory-%t", directory), volumeImageMutateState(bytes, changed), "recovery path inode alias")
	}
	for _, inode := range []uint64{1, 2} {
		changed := fs.ConformanceState()
		v := &changed.Volumes[0]
		v.Persisted.Objects[0].Inode = inode
		add(fmt.Sprintf("reserved-object-inode-%d", inode), volumeImageMutateState(bytes, changed), "recovery object inode alias")
	}
	locked := fs.ConformanceState()
	locked.Volumes[0].Persisted.Paths[1].Path = "/db.lock"
	add("sqlite-lock-path", volumeImageMutateState(bytes, locked), "durable image contains process-owned SQLite lock")
	cases[len(cases)-1].Storage.SQLiteLockDatabases = []string{"/workspace/db"}
	multiStorage := StorageModel{Volumes: []volume.VolumeConfig{{ID: "left", Path: "/left", CapacityBytes: 4096}, {ID: "right", Path: "/right", CapacityBytes: 4096}}, Limits: storage.Limits}
	multiFS := volume.New()
	require.NoError(t, multiFS.ConfigureVolumes(multiStorage.Volumes, multiStorage.Limits))
	multiModelJSON, err := json.Marshal(multiStorage)
	require.NoError(t, err)
	multiBytes, _, err := encodeStorageImage(multiFS, multiModelJSON, nil, 1<<20)
	require.NoError(t, err)
	multiModelHash := sha256.Sum256(multiModelJSON)
	for _, test := range []struct {
		inode  uint64
		orphan bool
	}{{3, false}, {3, true}, {1, false}, {2, false}} {
		state := multiFS.ConformanceState()
		right := &state.Volumes[1].Persisted
		object := right.Objects[0]
		object.Inode = test.inode
		if test.orphan {
			right.Objects = append([]volume.ConformanceDiskObject{object}, right.Objects...)
			right.Owned = append([]uint64{test.inode}, right.Owned...)
		} else {
			right.Objects = []volume.ConformanceDiskObject{object}
			right.Owned = []uint64{test.inode}
			right.Paths[0].Inode = test.inode
		}
		add(fmt.Sprintf("cross-volume-object-alias-%d-orphan-%t", test.inode, test.orphan), volumeImageMutateState(multiBytes, state), "recovery object inode alias")
		cases[len(cases)-1].Storage = multiStorage
		cases[len(cases)-1].Model = fmt.Sprintf("sha256:%x", multiModelHash)
	}
	for _, test := range []struct {
		name     string
		mounts   []string
		required uint64
	}{{"no-volumes", nil, 1}, {"workspace", []string{"/workspace"}, 2}, {"deep-mount", []string{"/long/deep/mount"}, 4}, {"shared-and-temp", []string{"/shared/a", "/shared/b", "/tmp/cache"}, 5}} {
		storage := StorageModel{Volumes: []volume.VolumeConfig{}, Limits: volumeCoreStorage().Limits}
		for i, mount := range test.mounts {
			storage.Volumes = append(storage.Volumes, volume.VolumeConfig{ID: fmt.Sprint(i), Path: mount, CapacityBytes: 1 << 20})
		}
		fs := volume.New()
		require.NoError(t, fs.ConfigureVolumes(storage.Volumes, storage.Limits))
		if len(test.mounts) != 0 {
			file, err := fs.Open(test.mounts[0]+"/retained", volume.OpenFlags{Read: true, Write: true, Create: true}, 0o600)
			require.NoError(t, err)
			_, err = file.WriteAt([]byte("payload"), 0)
			require.NoError(t, err)
			require.NoError(t, file.Sync())
			dir, err := fs.Open(test.mounts[0], volume.OpenFlags{Read: true}, 0)
			require.NoError(t, err)
			require.NoError(t, dir.Sync())
		}
		modelJSON, err := json.Marshal(storage)
		require.NoError(t, err)
		encoded, _, err := encodeStorageImage(fs, modelJSON, nil, 1<<20)
		require.NoError(t, err)
		for _, overflow := range []bool{false, true} {
			state := fs.ConformanceState()
			state.NextInode = ^uint64(0) - test.required
			expected := ""
			if overflow {
				state.NextInode++
				expected = "recovery filesystem counters"
			}
			add(fmt.Sprintf("exact-rebuild-headroom-%s-overflow-%t", test.name, overflow), volumeImageMutateState(encoded, state), expected)
			row := &cases[len(cases)-1]
			row.Storage = storage
			row.Model = fmt.Sprintf("sha256:%x", sha256.Sum256(modelJSON))
			if !overflow {
				maximum := ^uint64(0)
				row.RecoveredNextInode = &maximum
				temporary, err := fs.Stat("/tmp")
				require.NoError(t, err)
				row.RecoveredTempMode = &temporary.Mode
				row.RefuseAllocation = true
			}
		}
	}
	historicalBytes, err := os.ReadFile(filepath.Join("testdata", "volume-conformance", "sync-crash-reopen.json"))
	require.NoError(t, err)
	var historical volumeConformanceCorpus
	require.NoError(t, json.Unmarshal(historicalBytes, &historical))
	require.Greater(t, len(historical.Observations), 3)
	orphanBoundary := slices.Clone(historical.Observations[3].ImageData)
	foundState := false
	for offset := 16; offset+12 <= len(orphanBoundary); {
		tag := binary.LittleEndian.Uint32(orphanBoundary[offset : offset+4])
		length := binary.LittleEndian.Uint64(orphanBoundary[offset+4 : offset+12])
		offset += 12
		require.LessOrEqual(t, length, uint64(len(orphanBoundary)-offset))
		if tag == 3 {
			require.GreaterOrEqual(t, length, uint64(8))
			binary.LittleEndian.PutUint64(orphanBoundary[offset:offset+8], ^uint64(0)-2)
			foundState = true
			break
		}
		offset += int(length)
	}
	require.True(t, foundState)
	add("orphan-object-exact-rebuild-headroom", orphanBoundary, "")
	require.NotNil(t, historical.Session.Storage)
	row := &cases[len(cases)-1]
	row.Storage = *historical.Session.Storage
	historicalModel, err := json.Marshal(row.Storage)
	require.NoError(t, err)
	row.Model = fmt.Sprintf("sha256:%x", sha256.Sum256(historicalModel))
	maximum, temporaryMode := ^uint64(0), uint32(0o777)
	row.RecoveredNextInode = &maximum
	row.RecoveredTempMode = &temporaryMode
	row.RefuseAllocation = true
	data, err := json.MarshalIndent(struct {
		Schema string                     `json:"schema"`
		Cases  []volumeImageAdmissionCase `json:"cases"`
	}{"gomad.volume-image-admission/v1", cases}, "", "  ")
	require.NoError(t, err)
	data = append(data, '\n')
	name := filepath.Join("testdata", "volume-conformance", "vectors", "image-admission.json")
	if *volumeConformanceOutput != "" {
		name = filepath.Join(*volumeConformanceOutput, "vectors", "image-admission.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o755))
		require.NoError(t, os.WriteFile(name, data, 0o644))
		return
	}
	want, err := os.ReadFile(name)
	require.NoError(t, err)
	require.Equal(t, string(want), string(data))
}

func volumeImageMutateState(image []byte, state volume.ConformanceState) []byte {
	var fs volumeImageEncoder
	fs.number(state.NextInode)
	fs.number(state.Generation)
	fs.number(uint64(len(state.Volumes)))
	for _, v := range state.Volumes {
		fs.text(v.ID)
		fs.text(v.Mount)
		fs.number(v.CapacityBytes)
		fs.number(uint64(len(v.Persisted.Objects)))
		for _, object := range v.Persisted.Objects {
			fs.object(object)
		}
		fs.paths(v.Persisted.Paths)
		fs.numbers(v.Persisted.Owned)
		fs.operations(v.Pending)
		fs.operations(v.History)
		fs.number(v.NextOperation)
		fs.number(uint64(len(v.Last)))
		for _, entry := range v.Last {
			fs.text(entry.Key)
			fs.number(entry.Operation)
		}
	}
	offset := 16
	for i := 0; i < 2; i++ {
		offset += 12 + int(binary.LittleEndian.Uint64(image[offset+4:offset+12]))
	}
	result := slices.Clone(image[:offset+4])
	result = binary.LittleEndian.AppendUint64(result, uint64(fs.Len()))
	return append(result, fs.Bytes()...)
}
