//go:build test_dep

package volume

import (
	"fmt"
	"math"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConformanceRecoveryInodeHeadroomMatchesActualGoRebuild(t *testing.T) {
	for _, test := range []struct {
		paths       []string
		allocations uint64
	}{
		{nil, 1}, {[]string{"/workspace"}, 2}, {[]string{"/long/deep/mount"}, 4},
		{[]string{"/shared/a", "/shared/b", "/tmp/cache"}, 5},
	} {
		t.Run(fmt.Sprint(test.paths), func(t *testing.T) {
			fs := New()
			var configs []VolumeConfig
			for index, path := range test.paths {
				configs = append(configs, VolumeConfig{ID: fmt.Sprint(index), Path: path, CapacityBytes: 4096})
			}
			require.NoError(t, fs.ConfigureVolumes(configs, VolumeLimits{PendingOperations: 32, Transitions: 32}))
			if len(test.paths) > 0 {
				file, err := fs.Open(test.paths[0]+"/retained", OpenFlags{Write: true, Create: true}, 0o600)
				require.NoError(t, err)
				_, err = file.WriteAt([]byte("payload"), 0)
				require.NoError(t, err)
				require.NoError(t, file.Sync())
				require.NoError(t, file.Close())
				directory, err := fs.Open(test.paths[0], OpenFlags{Read: true}, 0)
				require.NoError(t, err)
				require.NoError(t, directory.Sync())
				require.NoError(t, directory.Close())
			}
			state := fs.ConformanceState()
			for _, volume := range state.Volumes {
				for _, object := range volume.Persisted.Objects {
					require.Greater(t, object.Inode, uint64(2))
				}
			}
			fs.nextInode = math.MaxUint64 - test.allocations
			next := fs.CloneDurable()
			require.Equal(t, uint64(math.MaxUint64), next.nextInode)
			require.Equal(t, uint64(math.MaxUint64)-test.allocations, fs.nextInode)
			if len(test.paths) > 0 {
				info, err := next.Stat(test.paths[0] + "/retained")
				require.NoError(t, err)
				require.Equal(t, []byte("payload"), info.Data)
			}
		})
	}
}

func conformanceFilesystem() *FS {
	fs := New()
	fs.nextInode = 41
	fs.generation = 7
	operation := &persistenceOperation{
		id: 9, kind: "namespace", deps: []uint64{8, 3}, stores: []string{"ns:/z", "ns:/a"},
		effect: persistenceEffect{
			kind: "namespace", inode: 13, parent: 2, parents: []uint64{12, 2},
			offset: 5, size: 11, mode: 0o640, modTime: -17, data: []byte("payload"),
			object:    &diskObject{inode: 13, mode: 0o600, kind: KindFile, modTime: -18, data: []byte("allocated")},
			namespace: []namespaceEffect{{path: "/z", inode: 0}, {path: "/a", inode: 13}},
		},
	}
	fs.volumes = map[string]*volumeState{
		"z": {
			id: "z", mount: "/first", capacityBytes: 4096,
			persisted: &diskState{
				objects: map[uint64]*diskObject{
					13: {inode: 13, mode: 0o600, kind: KindFile, modTime: -19, data: []byte("orphan")},
					2:  {inode: 2, mode: 0o755, kind: KindDirectory, modTime: 20},
					12: {inode: 12, mode: 0o644, kind: KindFile, modTime: 21, data: []byte("linked")},
				},
				paths: map[string]uint64{"/z": 12, "/": 2, "/a": 12},
				owned: map[uint64]struct{}{13: {}, 2: {}},
			},
			pending:       []*persistenceOperation{operation},
			history:       []*persistenceOperation{{id: 8, kind: "resize", effect: persistenceEffect{kind: "resize", inode: 13, size: 11}}, operation},
			nextOperation: 10, last: map[string]uint64{"ns:/z": 9, "alloc:13": 8},
		},
		"a": {id: "a", mount: "/second", capacityBytes: 8192, persisted: &diskState{}, nextOperation: 1},
	}
	return fs
}

func TestConformanceStateExportsFullRecoveryState(t *testing.T) {
	fs := conformanceFilesystem()
	state := fs.ConformanceState()
	require.Equal(t, uint64(41), state.NextInode)
	require.Equal(t, uint64(7), state.Generation)
	require.Len(t, state.Volumes, 2)
	require.Equal(t, "a", state.Volumes[0].ID)
	require.Equal(t, "/second", state.Volumes[0].Mount)
	require.Equal(t, uint64(8192), state.Volumes[0].CapacityBytes)
	require.Equal(t, uint64(1), state.Volumes[0].NextOperation)
	z := state.Volumes[1]
	require.Equal(t, "z", z.ID)
	require.Equal(t, "/first", z.Mount)
	require.Equal(t, uint64(4096), z.CapacityBytes)
	require.Equal(t, uint64(10), z.NextOperation)
	require.Equal(t, []ConformanceDiskObject{
		{Inode: 2, Mode: 0o755, Kind: KindDirectory, ModTime: 20},
		{Inode: 12, Mode: 0o644, Kind: KindFile, ModTime: 21, Data: []byte("linked")},
		{Inode: 13, Mode: 0o600, Kind: KindFile, ModTime: -19, Data: []byte("orphan")},
	}, z.Persisted.Objects)
	require.Equal(t, []ConformancePath{{Path: "/", Inode: 2}, {Path: "/a", Inode: 12}, {Path: "/z", Inode: 12}}, z.Persisted.Paths)
	require.Equal(t, []uint64{2, 13}, z.Persisted.Owned)
	require.Equal(t, []ConformanceLast{{Key: "alloc:13", Operation: 8}, {Key: "ns:/z", Operation: 9}}, z.Last)
	want := ConformancePersistenceOperation{
		ID: 9, Kind: "namespace", Dependencies: []uint64{8, 3}, Stores: []string{"ns:/z", "ns:/a"},
		Effect: ConformanceEffect{
			Kind: "namespace", Inode: 13, Parent: 2, Parents: []uint64{12, 2},
			Offset: 5, Size: 11, Mode: 0o640, ModTime: -17, Data: []byte("payload"),
			Object:    &ConformanceDiskObject{Inode: 13, Mode: 0o600, Kind: KindFile, ModTime: -18, Data: []byte("allocated")},
			Namespace: []ConformancePath{{Path: "/z", Inode: 0}, {Path: "/a", Inode: 13}},
		},
	}
	require.Equal(t, []ConformancePersistenceOperation{want}, z.Pending)
	require.Equal(t, []ConformancePersistenceOperation{
		{ID: 8, Kind: "resize", Effect: ConformanceEffect{Kind: "resize", Inode: 13, Size: 11}}, want,
	}, z.History)
}

func TestConformanceStateDoesNotMutateOrAliasModel(t *testing.T) {
	fs := conformanceFilesystem()
	before := fs.ConformanceState()
	changed := fs.ConformanceState()
	z := &changed.Volumes[1]
	z.Persisted.Objects[2].Data[0] = 'X'
	z.Persisted.Objects[0].Mode = 0
	z.Persisted.Paths[0].Path = "/changed"
	z.Persisted.Owned[0] = 99
	z.Last[0].Operation = 99
	for _, operations := range [][]ConformancePersistenceOperation{z.Pending, z.History[1:]} {
		operations[0].Dependencies[0] = 99
		operations[0].Stores[0] = "changed"
		operations[0].Effect.Parents[0] = 99
		operations[0].Effect.Data[0] = 'X'
		operations[0].Effect.Object.Data[0] = 'X'
		operations[0].Effect.Object.Mode = 0
		operations[0].Effect.Namespace[0].Path = "/changed"
	}
	changed.Volumes[0].ID = "changed"
	require.Equal(t, before, fs.ConformanceState())
	require.Equal(t, uint64(41), fs.nextInode)
	require.Equal(t, uint64(7), fs.generation)
	require.Equal(t, map[uint64]struct{}{2: {}, 13: {}}, fs.volumes["z"].persisted.owned)
	require.Same(t, fs.volumes["z"].pending[0], fs.volumes["z"].history[1])

	fs.volumes["z"].persisted.objects[13].data[0] = 'Y'
	fs.volumes["z"].pending[0].effect.data[0] = 'Y'
	require.Equal(t, []byte("orphan"), before.Volumes[1].Persisted.Objects[2].Data)
	require.Equal(t, []byte("payload"), before.Volumes[1].Pending[0].Effect.Data)
	require.Equal(t, []byte("payload"), before.Volumes[1].History[1].Effect.Data)
}

func TestConformanceStateRetainsSyncedOrphanAndActualOwnership(t *testing.T) {
	fs := New()
	fs.SetClock(&Clock{Value: 123})
	require.NoError(t, fs.ConfigureVolumes([]VolumeConfig{{ID: "data", Path: "/data", CapacityBytes: 4096}}, VolumeLimits{PendingOperations: 32, Transitions: 32}))
	file, err := fs.Open("/data/orphan", OpenFlags{Read: true, Write: true, Create: true}, 0o600)
	require.NoError(t, err)
	_, err = file.WriteAt([]byte("durable"), 0)
	require.NoError(t, err)
	require.NoError(t, file.Sync())
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	snapshots, operations, statistics := fs.VolumeSnapshots(), fs.Operations(), fs.Statistics()
	nextInode, generation := fs.nextInode, fs.generation
	state := fs.ConformanceState()
	require.Equal(t, snapshots, fs.VolumeSnapshots())
	require.Equal(t, operations, fs.Operations())
	require.Equal(t, statistics, fs.Statistics())
	require.Equal(t, nextInode, fs.nextInode)
	require.Equal(t, generation, fs.generation)
	require.Len(t, state.Volumes[0].Persisted.Paths, 1)
	require.Len(t, state.Volumes[0].Persisted.Objects, 2)
	require.Equal(t, []byte("durable"), state.Volumes[0].Persisted.Objects[1].Data)
	require.Len(t, state.Volumes[0].Persisted.Owned, 2)

	durable := fs.CloneDurable()
	_, err = durable.Stat("/data/orphan")
	require.ErrorIs(t, err, syscall.ENOENT)
	cloned := durable.ConformanceState()
	require.Equal(t, state.Volumes[0].Persisted.Objects, cloned.Volumes[0].Persisted.Objects)
	require.Empty(t, cloned.Volumes[0].Persisted.Owned)
	require.Empty(t, durable.volumes["data"].persisted.owned)
}
