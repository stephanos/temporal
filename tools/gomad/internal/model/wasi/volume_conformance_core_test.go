package wasi

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func TestVolumeCoreAPIContracts(t *testing.T) {
	for _, scenario := range volumeCoreAPIScripts(t) {
		t.Run(scenario.Name, func(t *testing.T) {
			steps := scenario.Steps
			last := steps[len(steps)-1]
			switch scenario.Name {
			case "captured-readonly-descendant-refuses-directory-rename":
				require.Equal(t, "cross-device", steps[3].Error)
				require.Equal(t, steps[2].Snapshots, steps[3].Snapshots)
				require.Equal(t, steps[2].Operations, steps[3].Operations)
				require.Equal(t, steps[2].State, steps[3].State)
				require.Equal(t, steps[2].UsedBytes, steps[3].UsedBytes)
				require.Equal(t, []byte("frozen"), steps[5].Data)
			case "clone-metadata-owned-object-isolation":
				require.Empty(t, steps[4].State.Volumes[0].Persisted.Owned)
				require.Equal(t, []uint64{4}, steps[7].State.Volumes[0].Persisted.Owned)
				require.Equal(t, uint32(0o600), last.State.Volumes[0].Persisted.Objects[1].Mode)
			case "lifecycle-invalid-selection-and-clone-isolation":
				for _, i := range []int{7, 8, 9} {
					require.NotEmpty(t, steps[i].Error)
					require.Equal(t, steps[6].Snapshots, steps[i].Snapshots)
				}
				require.Empty(t, steps[10].Selections)
				require.Equal(t, "stale", steps[12].Error)
				require.Equal(t, "stale", steps[13].Error)
				require.Equal(t, []byte("value"), steps[15].Data)
				require.Equal(t, []byte("value"), steps[21].Data)
				require.Len(t, steps[21].Operations["data"], 5)
			case "warm-digest-mutation-refusal-recovery":
				for i, step := range steps {
					if step.Error != "" {
						require.Equal(t, steps[i-1].Snapshots, step.Snapshots)
						require.Equal(t, steps[i-1].Operations, step.Operations)
					}
					if step.Op == "api-read" {
						require.Equal(t, []byte("durable"), step.Data)
					}
				}
				require.Zero(t, *last.UsedBytes)
			case "exact-volume-capacity-and-pending-refusal":
				require.Equal(t, &volume.VolumeCapacityError{Resource: "volume_bytes", Required: 4, Maximum: 3}, steps[2].Capacity)
				require.Equal(t, steps[1].Snapshots, steps[2].Snapshots)
				require.Equal(t, steps[1].Operations, steps[2].Operations)
				require.Empty(t, steps[3].Error)
				require.Equal(t, int64(3), steps[3].Value)
				require.Equal(t, &volume.VolumeCapacityError{Resource: "pending_operations", Required: 5, Maximum: 4}, last.Capacity)
			case "overlapping-writes-resize-dependencies":
				operations := last.Operations["data"]
				require.Equal(t, []uint64{1, 3}, operations[len(operations)-1].Dependencies)
			case "open-unlinked-data-accounting-last-close":
				require.Equal(t, []byte("abc"), steps[4].Data)
				require.Equal(t, uint64(5), *steps[5].UsedBytes)
				require.Equal(t, uint64(0), *last.UsedBytes)
				require.Empty(t, last.Snapshots[0].Volatile[1:])
			case "rename-open-target-byte-accounting":
				require.Equal(t, []byte("old"), steps[7].Data)
				require.Equal(t, []byte("new"), steps[9].Data)
				require.Equal(t, uint64(6), *steps[9].UsedBytes)
				require.Equal(t, uint64(3), *steps[10].UsedBytes)
				require.Equal(t, uint64(3), *last.UsedBytes)
			case "mkdir-all-preflight-planned-inode-consumption":
				require.Equal(t, &volume.VolumeCapacityError{Resource: "pending_operations", Required: 4, Maximum: 3}, last.Capacity)
				require.Equal(t, uint64(6), last.State.NextInode)
				require.Empty(t, last.State.Volumes[0].History)
				require.Equal(t, steps[0].Snapshots, last.Snapshots)
			case "append-positional-refusal-rename-handle-metadata":
				require.Equal(t, int64(4), steps[5].Value)
				require.Equal(t, "invalid", steps[6].Error)
				require.Equal(t, []byte("bcd"), steps[9].Data)
				entries := last.Snapshots[0].Volatile
				require.Equal(t, uint32(0o764), entries[1].Mode)
				require.Equal(t, int64(-8), entries[1].ModTime)
			case "seek-signed-wrap-and-negative-refusal":
				require.Equal(t, int64(2), steps[3].Value)
				require.Equal(t, "invalid", steps[4].Error)
				require.Equal(t, int64(2), steps[5].Value)
				require.Equal(t, int64(math.MaxInt64), steps[6].Value)
				require.Equal(t, "invalid", steps[7].Error)
			case "directory-handle-sorted-reads-original-name-inode-chdir":
				require.Equal(t, []string{"a"}, steps[5].Names)
				require.Equal(t, []string{"b"}, steps[6].Names)
				require.Equal(t, "eof", steps[7].Error)
				require.Equal(t, "/workspace/moved", steps[9].Cwd)
				require.Equal(t, "/workspace/moved", last.Cwd)
				require.Equal(t, "/workspace/dir", last.HandlePath)
			case "remove-all-reverse-namespace-effects":
				namespace := last.State.Volumes[0].Pending[len(last.State.Volumes[0].Pending)-1].Effect.Namespace
				require.Equal(t, []volume.ConformancePath{{Path: "/a/sub/file"}, {Path: "/a/sub"}, {Path: "/a"}}, namespace)
			case "rejected-bounds-preserve-data-history-and-offset":
				for _, i := range []int{3, 4, 5, 6} {
					require.Equal(t, steps[2].Snapshots, steps[i].Snapshots)
					require.Equal(t, steps[2].Operations, steps[i].Operations)
				}
				require.Equal(t, int64(3), last.Value)
			case "closed-open-handle-lifecycle-precedence":
				require.Equal(t, "bad-descriptor", steps[2].Error)
				require.Equal(t, "bad-descriptor", steps[4].Error)
				require.Equal(t, "stale", steps[6].Error)
				require.Equal(t, "stale", steps[7].Error)
			}
		})
	}
}

func volumeCoreGracefulLiteralScript(t *testing.T) volumeCoreScenario {
	t.Helper()
	scenario := volumeCoreScenario{Name: "graceful-lifecycle-literal-value-after-clone-isolation", Storage: StorageModel{
		Volumes: []volume.VolumeConfig{{ID: "data", Path: "/workspace", CapacityBytes: 1 << 20}},
		Limits:  volume.VolumeLimits{PendingOperations: 1000, Transitions: 1000},
	}}
	clock := &volume.Clock{}
	fs := volume.New()
	fs.SetClock(clock)
	capture := func(op string, input volumeCoreInput) {
		scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{Op: op, Input: input}))
	}
	require.NoError(t, fs.ConfigureVolumes(scenario.Storage.Volumes, scenario.Storage.Limits))
	capture("configure", volumeCoreInput{})
	flags := volume.OpenFlags{Read: true, Write: true, Create: true}
	clock.Value = 100
	file, err := fs.Open("/workspace/db", flags, 0o666)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/db", Handle: "db", Flags: &flags, Mode: 0o666})
	clock.Value = 117
	_, err = file.WriteAt([]byte("value"), 0)
	require.NoError(t, err)
	capture("write-at", volumeCoreInput{Handle: "db", Data: []byte("value")})
	require.NoError(t, file.Sync())
	capture("file-sync", volumeCoreInput{Handle: "db"})
	directory, err := fs.Open("/workspace", volume.OpenFlags{Read: true}, 0)
	require.NoError(t, err)
	require.NoError(t, directory.Sync())
	require.NoError(t, directory.Close())
	capture("directory-sync", volumeCoreInput{Path: "/workspace"})
	clock.Value = 200
	_, err = file.WriteAt([]byte("lost!"), 0)
	require.NoError(t, err)
	capture("write-at", volumeCoreInput{Handle: "db", Data: []byte("lost!")})
	fs, err = fs.AdvanceVolumeLifecycle(false, nil)
	require.NoError(t, err)
	capture("advance-crash", volumeCoreInput{})
	source := fs
	fs = fs.CloneDurable()
	capture("api-clone", volumeCoreInput{})
	file, err = fs.Open("/workspace/db", flags, 0o666)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/db", Handle: "cloned", Flags: &flags, Mode: 0o666})
	clock.Value = 300
	_, err = file.WriteAt([]byte("clone"), 0)
	require.NoError(t, err)
	capture("write-at", volumeCoreInput{Handle: "cloned", Data: []byte("clone")})
	fs = source
	capture("api-source", volumeCoreInput{})
	fs, err = fs.AdvanceVolumeLifecycle(true, nil)
	require.NoError(t, err)
	capture("advance-graceful", volumeCoreInput{})
	readFlags := volume.OpenFlags{Read: true}
	file, err = fs.Open("/workspace/db", readFlags, 0)
	require.NoError(t, err)
	capture("api-open", volumeCoreInput{Path: "/workspace/db", Handle: "graceful", Flags: &readFlags})
	data := make([]byte, 100)
	n, err := file.Read(data)
	require.NoError(t, err)
	data = data[:n]
	require.Equal(t, []byte("value"), data)
	scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{Op: "api-read", Input: volumeCoreInput{Handle: "graceful", Size: 100}, Data: data, Value: int64(n)}))
	volumeCoreImage(t, fs, scenario.Storage, &scenario.Steps[len(scenario.Steps)-1])
	return scenario
}

func volumeCoreAPIScripts(t *testing.T) []volumeCoreScenario {
	t.Helper()
	flags := volume.OpenFlags{Read: true, Write: true, Create: true}
	zero := uint64(0)
	appendFlags := flags
	appendFlags.Append = true
	open := func(path, handle string, f volume.OpenFlags) volumeCoreStep {
		return volumeCoreStep{Op: "api-open", Input: volumeCoreInput{Path: path, Handle: handle, Flags: &f, Mode: 0o600}}
	}
	op := func(name string, input volumeCoreInput) volumeCoreStep {
		return volumeCoreStep{Op: "api-" + name, Input: input}
	}
	file := func(name string) volumeCoreInput { return volumeCoreInput{Handle: name} }
	path := func(name string) volumeCoreInput { return volumeCoreInput{Path: name, Mode: 0o755} }
	definitions := []volumeCoreScenario{
		{Name: "unavailable-lifecycle-source-normalization-precedence", Steps: []volumeCoreStep{
			open("/workspace/file", "file", flags), op("advance", volumeCoreInput{}), op("source", volumeCoreInput{}),
			op("mkdir", path("/workspace/dir")), op("remove", path("/workspace/file")), op("remove-all", path("/workspace/dir")),
			open("/workspace/file", "stale", volume.OpenFlags{}), op("chmod", volumeCoreInput{Path: "/workspace/file", Mode: 0o600}),
		}},
		{Name: "captured-readonly-descendant-refuses-directory-rename", Steps: []volumeCoreStep{op("mkdir", path("/workspace/mutable")), op("captured-file", volumeCoreInput{Path: "/workspace/mutable/frozen", Mode: 0o444, Data: []byte("frozen")}), op("rename", volumeCoreInput{Path: "/workspace/mutable", Destination: "/workspace/moved"}), open("/workspace/mutable/frozen", "frozen", volume.OpenFlags{Read: true}), op("read", volumeCoreInput{Handle: "frozen", Size: 100})}},
		{Name: "clone-metadata-owned-object-isolation", Steps: []volumeCoreStep{open("/workspace/file", "file", flags), open("/workspace", "root", volume.OpenFlags{Read: true}), op("sync", file("root")), op("clone", volumeCoreInput{}), open("/workspace/file", "cloned", flags), op("handle-chmod", volumeCoreInput{Handle: "cloned", Mode: 0o644}), op("sync", file("cloned")), op("source", volumeCoreInput{})}},
		{Name: "shared-mount-and-temp-prefix-allocation", Storage: StorageModel{Volumes: []volume.VolumeConfig{{ID: "a", Path: "/shared/a", CapacityBytes: 1 << 20}, {ID: "b", Path: "/shared/b", CapacityBytes: 1 << 20}, {ID: "c", Path: "/tmp/cache", CapacityBytes: 1 << 20}}, Limits: volumeCoreStorage().Limits}, Steps: []volumeCoreStep{op("flush", volumeCoreInput{})}},
		{Name: "normalization-parent-escape-refusal", Steps: []volumeCoreStep{op("mkdir", path("/../escape")), op("remove", path("/../escape")), op("remove-all", path("/../escape"))}},
		{Name: "invalid-configuration-no-mount-allocation", Storage: StorageModel{Volumes: []volume.VolumeConfig{{ID: "data", Path: "/workspace/../workspace", CapacityBytes: 1}}, Limits: volumeCoreStorage().Limits}},
		{Name: "lifecycle-invalid-selection-and-clone-isolation", Steps: []volumeCoreStep{
			open("/workspace/db", "db", flags), op("write-at", volumeCoreInput{Handle: "db", Data: []byte("value")}), op("sync", file("db")), open("/workspace", "root", volume.OpenFlags{Read: true}), op("sync", file("root")), op("write-at", volumeCoreInput{Handle: "db", Data: []byte("lost!")}),
			op("crash", volumeCoreInput{Selections: map[string][]uint64{"data": {999}}}), op("crash", volumeCoreInput{Selections: map[string][]uint64{"data": {5, 5}}}), op("crash", volumeCoreInput{Selections: map[string][]uint64{"unknown": {}}}),
			op("select", volumeCoreInput{Seed: &zero}), op("advance", volumeCoreInput{}), op("sync", file("db")), op("source-stat", path("/workspace/db")), open("/workspace/db", "recovered", flags), op("read", volumeCoreInput{Handle: "recovered", Size: 100}),
			op("clone", volumeCoreInput{}), open("/workspace/db", "cloned", flags), op("write-at", volumeCoreInput{Handle: "cloned", Data: []byte("clone")}), op("source", volumeCoreInput{}), op("seek", file("recovered")), op("read", volumeCoreInput{Handle: "recovered", Size: 100}), op("flush", volumeCoreInput{}),
		}},
		{Name: "overlapping-writes-resize-dependencies", Steps: []volumeCoreStep{
			open("/workspace/file", "file", flags), op("write-at", volumeCoreInput{Handle: "file", Data: []byte("abc")}), op("write-at", volumeCoreInput{Handle: "file", Offset: 1, Data: []byte("XY")}),
		}},
		{Name: "exact-volume-capacity-and-pending-refusal", Storage: StorageModel{Volumes: []volume.VolumeConfig{{ID: "data", Path: "/workspace", CapacityBytes: 3}}, Limits: volume.VolumeLimits{PendingOperations: 4, Transitions: 100}}, Steps: []volumeCoreStep{
			open("/workspace/file", "file", flags), op("write-at", volumeCoreInput{Handle: "file", Data: []byte("four")}), op("write-at", volumeCoreInput{Handle: "file", Data: []byte("abc")}), op("resize", volumeCoreInput{Handle: "file", Size: 1}),
		}},
		{Name: "open-unlinked-data-accounting-last-close", Storage: StorageModel{Volumes: []volume.VolumeConfig{{ID: "data", Path: "/workspace", CapacityBytes: 5}}, Limits: volumeCoreStorage().Limits}, Steps: []volumeCoreStep{
			open("/workspace/file", "file", flags), op("write-at", volumeCoreInput{Handle: "file", Data: []byte("abc")}), op("remove", path("/workspace/file")), op("read", volumeCoreInput{Handle: "file", Size: 10}), op("write-at", volumeCoreInput{Handle: "file", Offset: 3, Data: []byte("xy")}), op("sync", file("file")), op("close", file("file")),
		}},
		{Name: "rename-open-target-byte-accounting", Steps: []volumeCoreStep{
			open("/workspace/source", "source", flags), open("/workspace/target", "target", flags), op("write", volumeCoreInput{Handle: "source", Data: []byte("new")}), op("write", volumeCoreInput{Handle: "target", Data: []byte("old")}), op("rename", volumeCoreInput{Path: "/workspace/source", Destination: "/workspace/target"}), op("seek", file("target")), op("read", volumeCoreInput{Handle: "target", Size: 10}), op("seek", file("source")), op("read", volumeCoreInput{Handle: "source", Size: 10}), op("close", file("target")), op("close", file("source")),
		}},
		{Name: "revoked-filesystem-normalization-precedence", Steps: []volumeCoreStep{open("/workspace/file", "file", flags), op("revoke", volumeCoreInput{}), op("mkdir", path("/workspace/dir")), op("remove", path("/workspace/file")), op("remove-all", path("/workspace/dir")), open("/workspace/file", "refused", flags), op("chmod", volumeCoreInput{Path: "/workspace/file", Mode: 0o600})}},
		{Name: "mkdir-all-preflight-planned-inode-consumption", Storage: StorageModel{Volumes: volumeCoreStorage().Volumes, Limits: volume.VolumeLimits{PendingOperations: 3, Transitions: 100}}, Steps: []volumeCoreStep{op("mkdir-all", volumeCoreInput{Path: "/workspace/a/b", Mode: 0o777})}},
		{Name: "append-positional-refusal-rename-handle-metadata", Steps: []volumeCoreStep{
			open("/workspace/file", "file", appendFlags), op("write", volumeCoreInput{Handle: "file", Data: []byte("abc")}), op("seek", file("file")), op("write", volumeCoreInput{Handle: "file", Data: []byte("d")}), op("seek", volumeCoreInput{Handle: "file", Whence: 1}), op("write-at", volumeCoreInput{Handle: "file", Data: []byte("x")}), op("rename", volumeCoreInput{Path: "/workspace/file", Destination: "/workspace/other"}), op("seek", volumeCoreInput{Handle: "file", Offset: 1}), op("read", volumeCoreInput{Handle: "file", Size: 100}), op("handle-chmod", volumeCoreInput{Handle: "file", Mode: 0o1764}), op("handle-chtimes", volumeCoreInput{Handle: "file", ModTime: -8}), op("seek", volumeCoreInput{Handle: "file", Whence: 1}),
		}},
		{Name: "seek-signed-wrap-and-negative-refusal", Steps: []volumeCoreStep{
			open("/workspace/file", "file", flags), op("write", volumeCoreInput{Handle: "file", Data: []byte("abc")}), op("seek", volumeCoreInput{Handle: "file", Offset: -1, Whence: 2}), op("seek", volumeCoreInput{Handle: "file", Offset: -3, Whence: 1}), op("seek", volumeCoreInput{Handle: "file", Whence: 1}), op("seek", volumeCoreInput{Handle: "file", Offset: math.MaxInt64}), op("seek", volumeCoreInput{Handle: "file", Offset: 1, Whence: 1}),
		}},
		{Name: "directory-handle-sorted-reads-original-name-inode-chdir", Steps: []volumeCoreStep{
			op("mkdir", path("/workspace/dir")), open("/workspace/dir/b", "b", flags), open("/workspace/dir/a", "a", flags), open("/workspace/dir", "dir", volume.OpenFlags{Read: true}), op("read-dir", volumeCoreInput{Handle: "dir", Size: 1}), op("read-dir", volumeCoreInput{Handle: "dir", Size: 1}), op("read-dir", volumeCoreInput{Handle: "dir", Size: 1}), op("chdir", path("/workspace/dir")), op("rename", volumeCoreInput{Path: "/workspace/dir", Destination: "/workspace/moved"}), op("chdir", path("/")), op("handle-chdir", file("dir")),
		}},
		{Name: "remove-all-reverse-namespace-effects", Steps: []volumeCoreStep{
			op("mkdir-all", path("/workspace/a/sub")), open("/workspace/a/sub/file", "file", flags), op("remove-all", path("/workspace/a")),
			open("/", "root", volume.OpenFlags{Read: true}), op("read-dir", volumeCoreInput{Handle: "root", Size: 100}),
			open("/tmp", "tmp", volume.OpenFlags{Read: true}), op("read-dir", volumeCoreInput{Handle: "tmp", Size: 100}),
			open("/workspace", "mount", volume.OpenFlags{Read: true}), op("read-dir", volumeCoreInput{Handle: "mount", Size: 100}),
		}},
		{Name: "rejected-bounds-preserve-data-history-and-offset", Steps: []volumeCoreStep{
			open("/workspace/file", "file", flags), op("write", volumeCoreInput{Handle: "file", Data: []byte("abc")}), op("write-at", volumeCoreInput{Handle: "file", Offset: (256 << 20) + 1, Data: []byte("xy")}), op("write-at", volumeCoreInput{Handle: "file", Offset: -1, Data: []byte("x")}), op("resize", volumeCoreInput{Handle: "file", Size: (256 << 20) + 1}), op("resize", volumeCoreInput{Handle: "file", Size: -1}), op("seek", volumeCoreInput{Handle: "file", Whence: 1}),
		}},
		{Name: "closed-open-handle-lifecycle-precedence", Steps: []volumeCoreStep{
			open("/workspace/file", "file", volume.OpenFlags{Read: true, Create: true}), op("resize", file("file")), op("close", file("file")), op("sync", file("file")), op("advance", volumeCoreInput{}), op("sync", file("file")), op("read", volumeCoreInput{Handle: "file", Size: 1}),
		}},
	}
	warm := volumeCoreScenario{Name: "warm-digest-mutation-refusal-recovery", Storage: StorageModel{Volumes: []volume.VolumeConfig{{ID: "data", Path: "/workspace", CapacityBytes: 64}}, Limits: volumeCoreStorage().Limits}}
	warm.Steps = append(warm.Steps, open("/workspace/file", "file", flags), open("/workspace", "root", volume.OpenFlags{Read: true}))
	for _, write := range []struct {
		offset int64
		data   []byte
	}{{0, []byte("abc")}, {1, []byte("xy")}, {5, []byte("z")}, {0, nil}} {
		warm.Steps = append(warm.Steps, op("write-at", volumeCoreInput{Handle: "file", Offset: write.offset, Data: write.data}), op("sync", file("file")), op("sync", file("root")))
	}
	for _, size := range []int64{2, 5, 5, 0} {
		warm.Steps = append(warm.Steps, op("resize", volumeCoreInput{Handle: "file", Size: size}), op("sync", file("file")))
	}
	warm.Steps = append(warm.Steps, op("write-at", volumeCoreInput{Handle: "file", Data: []byte("durable")}), op("handle-chmod", volumeCoreInput{Handle: "file", Mode: 0o640}), op("handle-chtimes", volumeCoreInput{Handle: "file", ModTime: 9}), op("rename", volumeCoreInput{Path: "/workspace/file", Destination: "/workspace/renamed"}), op("flush", volumeCoreInput{}), open("/workspace/renamed", "file", flags), op("write-at", volumeCoreInput{Handle: "file", Data: []byte("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")}), op("resize", volumeCoreInput{Handle: "file", Size: (256 << 20) + 1}), op("write-at", volumeCoreInput{Handle: "file", Data: []byte("lost")}), op("crash", volumeCoreInput{}), open("/workspace/renamed", "file", flags), op("read", volumeCoreInput{Handle: "file", Size: 100}), op("clone", volumeCoreInput{}), open("/workspace/renamed", "orphan", flags), op("remove", path("/workspace/renamed")), op("write", volumeCoreInput{Handle: "orphan", Data: []byte("orphan")}), op("close", file("orphan")))
	var observed []volumeCoreStep
	for _, step := range warm.Steps {
		observed = append(observed, op("observe", volumeCoreInput{}), step, op("observe", volumeCoreInput{}))
	}
	warm.Steps = observed
	definitions = append(definitions, warm)
	for i := range definitions {
		scenario := &definitions[i]
		if scenario.Storage.Volumes == nil {
			scenario.Storage = volumeCoreStorage()
		}
		clock := &volume.Clock{Value: 100}
		fs := volume.New()
		fs.SetClock(clock)
		configureError := fs.ConfigureVolumes(scenario.Storage.Volumes, scenario.Storage.Limits)
		if scenario.Name == "invalid-configuration-no-mount-allocation" {
			require.EqualError(t, configureError, "simulation volume has an invalid mount path")
		} else {
			require.NoError(t, configureError)
		}
		steps := append([]volumeCoreStep{{Op: "configure"}}, scenario.Steps...)
		handles := map[string]*volume.Handle{}
		var source *volume.FS
		for j, step := range steps {
			clock.Value = 100 + int64(j)*17
			input := step.Input
			handle := handles[input.Handle]
			var err error
			switch step.Op {
			case "configure":
				err = configureError
			case "api-observe":
			case "api-captured-file":
				loader, loaderErr := volume.NewCapturedSnapshot([]string{input.Path}, []volume.CapturedEntry{{Path: input.Path, Entry: volume.LoadEntry{Kind: volume.KindFile, Mode: input.Mode, Data: input.Data}}}, volume.CapturedSnapshotLimits{Entries: 8, Bytes: 1024, Lookups: 16})
				require.NoError(t, loaderErr)
				fs.SetCapturedSnapshot(loader)
				_, err = fs.Stat(input.Path)
			case "api-open":
				handles[input.Handle], err = fs.Open(input.Path, *input.Flags, input.Mode)
			case "api-write":
				var n int
				n, err = handle.Write(input.Data)
				step.Value = int64(n)
			case "api-write-at":
				var n int
				n, err = handle.WriteAt(input.Data, input.Offset)
				step.Value = int64(n)
			case "api-seek":
				step.Value, err = handle.Seek(input.Offset, input.Whence)
			case "api-read":
				data := make([]byte, input.Size)
				var n int
				n, err = handle.Read(data)
				if n > 0 {
					step.Data = data[:n]
				}
				step.Value = int64(n)
			case "api-read-dir":
				var entries []volume.Entry
				entries, err = handle.ReadDir(int(input.Size))
				for _, entry := range entries {
					step.Names = append(step.Names, entry.Name)
				}
			case "api-handle-chmod":
				err = handle.Chmod(input.Mode)
			case "api-handle-chtimes":
				err = handle.Chtimes(input.ModTime)
			case "api-handle-chdir":
				err = handle.Chdir()
			case "api-chdir":
				err = fs.Chdir(input.Path)
			case "api-mkdir":
				err = fs.Mkdir(input.Path, input.Mode)
			case "api-mkdir-all":
				err = fs.MkdirAll(input.Path, input.Mode)
			case "api-remove-all":
				err = fs.RemoveAll(input.Path)
			case "api-remove":
				err = fs.Remove(input.Path)
			case "api-chmod":
				err = fs.Chmod(input.Path, input.Mode)
			case "api-rename":
				err = fs.Rename(input.Path, input.Destination)
			case "api-resize":
				err = handle.Truncate(input.Size)
			case "api-close":
				err = handle.Close()
			case "api-sync":
				err = handle.Sync()
			case "api-advance":
				source = fs
				fs, err = fs.AdvanceVolumeLifecycle(false, nil)
			case "api-select":
				step.Selections = fs.SelectCrashOperations(*input.Seed)
			case "api-source-stat":
				_, err = source.Stat(input.Path)
			case "api-flush":
				err = fs.FlushVolumes()
			case "api-crash":
				err = fs.CrashVolumes(input.Selections)
			case "api-clone":
				source = fs
				fs = fs.CloneDurable()
			case "api-source":
				fs, source = source, fs
			case "api-revoke":
				_, err = fs.AdvanceVolumeLifecycle(false, nil)
			default:
				t.Fatalf("unknown core op %s", step.Op)
			}
			step.Error = volumeCoreError(err)
			if capacity, ok := err.(*volume.VolumeCapacityError); ok {
				step.Capacity = capacity
			}
			step.Cwd = fs.Getwd()
			if currentHandle := handles[input.Handle]; currentHandle != nil {
				step.HandlePath = currentHandle.Path()
			}
			used := fs.Statistics().UsedBytes
			step.UsedBytes = &used
			steps[j] = volumeCoreCapture(fs, clock, step)
			volumeCoreImage(t, fs, scenario.Storage, &steps[j])
		}
		scenario.Steps = steps
	}
	return definitions
}

func volumeCoreError(err error) string {
	if err == nil {
		return ""
	}
	for _, test := range []struct {
		err  error
		name string
	}{{syscall.EINVAL, "invalid"}, {syscall.EBADF, "bad-descriptor"}, {syscall.ESTALE, "stale"}, {syscall.EFBIG, "file-too-large"}, {syscall.ENOSPC, "no-space"}, {syscall.EXDEV, "cross-device"}, {io.EOF, "eof"}} {
		if errors.Is(err, test.err) {
			return test.name
		}
	}
	return err.Error()
}

func TestVolumeErrorMappingCorpus(t *testing.T) {
	type mapping struct {
		Kind     string `json:"kind"`
		Name     string `json:"name"`
		Errno    uint16 `json:"errno"`
		Boundary string `json:"boundary"`
	}
	var cases []mapping
	for _, test := range []struct {
		kind, name string
		err        error
	}{
		{"nil", "nil", nil}, {"errno", "eof", io.EOF}, {"errno", "closed", volume.ErrClosed},
		{"errno", "read-only-access", syscall.EACCES}, {"errno", "busy", syscall.EBUSY}, {"errno", "exists", syscall.EEXIST}, {"errno", "file-too-large", syscall.EFBIG}, {"errno", "invalid", syscall.EINVAL}, {"errno", "is-directory", syscall.EISDIR}, {"errno", "too-many-files", syscall.EMFILE}, {"errno", "no-memory", syscall.ENOMEM}, {"errno", "not-directory", syscall.ENOTDIR}, {"errno", "not-empty", syscall.ENOTEMPTY}, {"errno", "permission", syscall.EPERM}, {"errno", "read-only", syscall.EROFS}, {"errno", "cross-device", syscall.EXDEV}, {"errno", "unmapped", syscall.EINTR},
		{"errno", "stale", syscall.ESTALE}, {"errno", "bad-descriptor", syscall.EBADF}, {"errno", "no-entry", syscall.ENOENT}, {"errno", "no-space", syscall.ENOSPC}, {"errno", "io", syscall.EIO}, {"invalid", "x", errors.New("x")},
		{"capacity", "volume_bytes", &volume.VolumeCapacityError{Resource: "volume_bytes", Required: 3, Maximum: 2}},
		{"capacity", "pending_operations", &volume.VolumeCapacityError{Resource: "pending_operations", Required: 3, Maximum: 2}},
		{"capacity", "transitions", &volume.VolumeCapacityError{Resource: "transitions", Required: 3, Maximum: 2}},
	} {
		boundary := ""
		if err := storageBoundary(test.err); err != nil {
			boundary = err.Error()
		}
		cases = append(cases, mapping{test.kind, test.name, storageErrno(test.err), boundary})
	}
	data, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	name := filepath.Join("testdata", "volume-conformance", "vectors", "error-mapping.json")
	if *volumeConformanceOutput != "" {
		name = filepath.Join(*volumeConformanceOutput, "vectors", "error-mapping.json")
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(data) {
		t.Fatal("error mapping corpus drift")
	}
}
