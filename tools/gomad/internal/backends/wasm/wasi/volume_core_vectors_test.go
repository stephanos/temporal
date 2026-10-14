//go:build test_dep

package wasi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

type volumeCoreVectors struct {
	Schema    string               `json:"schema"`
	Model     string               `json:"model"`
	Scenarios []volumeCoreScenario `json:"scenarios"`
}

type volumeCoreScenario struct {
	Name    string           `json:"name"`
	Storage StorageModel     `json:"storage"`
	Steps   []volumeCoreStep `json:"steps"`
}

type volumeCoreInput struct {
	Path        string              `json:"path,omitempty"`
	Destination string              `json:"destination,omitempty"`
	Handle      string              `json:"handle,omitempty"`
	Flags       *volume.OpenFlags   `json:"flags,omitempty"`
	Mode        uint32              `json:"mode,omitempty"`
	Offset      int64               `json:"offset,omitempty"`
	Size        int64               `json:"size,omitempty"`
	ModTime     int64               `json:"mod_time,omitempty"`
	Data        []byte              `json:"data_base64,omitempty"`
	Seed        *uint64             `json:"seed,omitempty"`
	Selections  map[string][]uint64 `json:"selections,omitempty"`
}

type volumeCoreStep struct {
	Op            string                        `json:"op"`
	Now           int64                         `json:"now"`
	Input         volumeCoreInput               `json:"input"`
	Snapshots     []volume.VolumeSnapshot       `json:"snapshots"`
	Operations    map[string][]volume.Operation `json:"operations"`
	State         volume.ConformanceState       `json:"state"`
	Selections    map[string][]uint64           `json:"selections"`
	Accepted      *bool                         `json:"accepted,omitempty"`
	ImageDigest   string                        `json:"image_digest,omitempty"`
	ImageData     []byte                        `json:"image_base64,omitempty"`
	ImageIdentity string                        `json:"image_identity,omitempty"`
}

func TestVolumeCoreVectors(t *testing.T) {
	got := volumeCoreVectors{
		Schema: "gomad3.volume-core-vectors/v1", Model: volumeConformanceModel,
		Scenarios: []volumeCoreScenario{volumeCoreScript(t), volumeCoreNestedOrphanScript(t)},
	}
	if *volumeConformanceOutput != "" {
		require.True(t, filepath.IsAbs(*volumeConformanceOutput), "volume-conformance-output must be an absolute external directory")
		directory := filepath.Join(*volumeConformanceOutput, "vectors")
		require.NoError(t, os.MkdirAll(directory, 0o755))
		data, err := json.MarshalIndent(got, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(directory, "core.json"), append(data, '\n'), 0o644))
		return
	}
	data, err := os.ReadFile(filepath.Join("testdata", "volume-conformance", "vectors", "core.json"))
	require.NoError(t, err, "Go core vectors missing; generate into an external directory for review")
	var want volumeCoreVectors
	require.NoError(t, json.Unmarshal(data, &want))
	require.Equal(t, want, got, "Go pure volume vector drift; inspect before updating vectors/core.json")
}

func volumeCoreStorage() StorageModel {
	return StorageModel{
		Volumes: []volume.VolumeConfig{{ID: "data", Path: "/workspace", CapacityBytes: 65536}},
		Limits:  volume.VolumeLimits{PendingOperations: 128, Transitions: 256},
	}
}

func volumeCoreCapture(fs *volume.FS, clock *volume.Clock, step volumeCoreStep) volumeCoreStep {
	step.Now = clock.Value
	step.Snapshots = fs.VolumeSnapshots()
	step.Operations = fs.Operations()
	step.State = fs.ConformanceState()
	return step
}

func volumeCoreImage(t *testing.T, fs *volume.FS, storage StorageModel, step *volumeCoreStep) {
	t.Helper()
	model, err := json.Marshal(storage)
	require.NoError(t, err)
	image := &StorageImage{fs: fs, model: model, occurrences: map[string]uint64{}}
	step.ImageData, step.ImageDigest, err = volumeConformanceEncodeImage(image, 1<<20)
	require.NoError(t, err)
	step.ImageIdentity = volumeConformanceBinaryImageIdentity
}

func volumeCoreScript(t *testing.T) volumeCoreScenario {
	t.Helper()
	scenario := volumeCoreScenario{Name: "metadata-rename-detached-lifecycle", Storage: volumeCoreStorage()}
	clock := &volume.Clock{Value: 100}
	fs := volume.New()
	fs.SetClock(clock)
	capture := func(op string, input volumeCoreInput) {
		scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{Op: op, Input: input}))
	}
	require.NoError(t, fs.ConfigureVolumes(scenario.Storage.Volumes, scenario.Storage.Limits))
	capture("configure", volumeCoreInput{})
	flags := volume.OpenFlags{Read: true, Write: true, Create: true}
	file, err := fs.Open("/workspace/db", flags, 0o666)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/db", Handle: "db", Flags: &flags, Mode: 0o666})
	clock.Value = 117
	n, err := file.WriteAt([]byte("value"), 0)
	require.NoError(t, err)
	require.Equal(t, 5, n)
	capture("write-at", volumeCoreInput{Handle: "db", Data: []byte("value")})
	clock.Value = 121
	require.NoError(t, fs.Chmod("/workspace/db", 0o600))
	capture("chmod", volumeCoreInput{Path: "/workspace/db", Mode: 0o600})
	require.NoError(t, fs.Chtimes("/workspace/db", -7))
	capture("chtimes", volumeCoreInput{Path: "/workspace/db", ModTime: -7})
	clock.Value = 133
	require.NoError(t, fs.MkdirAll("/workspace/a/b", 0o755))
	capture("mkdir-all", volumeCoreInput{Path: "/workspace/a/b", Mode: 0o755})
	clock.Value = 150
	require.NoError(t, fs.Rename("/workspace/db", "/workspace/a/b/moved"))
	capture("rename", volumeCoreInput{Path: "/workspace/db", Destination: "/workspace/a/b/moved"})
	for _, seed := range []uint64{0, 1, 7, 42} {
		scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{
			Op: "select", Input: volumeCoreInput{Seed: &seed}, Selections: fs.SelectCrashOperations(seed),
		}))
	}
	require.NoError(t, file.Sync())
	capture("file-sync", volumeCoreInput{Handle: "db"})
	for _, directory := range []string{"/workspace", "/workspace/a", "/workspace/a/b"} {
		clock.Value++
		handle, err := fs.Open(directory, volume.OpenFlags{Read: true}, 0)
		require.NoError(t, err)
		require.NoError(t, handle.Sync())
		require.NoError(t, handle.Close())
		capture("directory-sync", volumeCoreInput{Path: directory})
	}
	clock.Value = 171
	require.NoError(t, fs.Remove("/workspace/a/b/moved"))
	capture("remove", volumeCoreInput{Path: "/workspace/a/b/moved"})
	clock.Value = 173
	require.NoError(t, file.Truncate(3))
	capture("truncate", volumeCoreInput{Handle: "db", Size: 3})
	clock.Value = 177
	n, err = file.WriteAt([]byte("end"), 0)
	require.NoError(t, err)
	require.Equal(t, 3, n)
	capture("write-at", volumeCoreInput{Handle: "db", Data: []byte("end")})
	clock.Value = 179
	require.NoError(t, file.Sync())
	capture("file-sync", volumeCoreInput{Handle: "db"})
	require.NoError(t, file.Close())
	capture("close", volumeCoreInput{Handle: "db"})
	clock.Value = 182
	fs = fs.CloneDurable()
	capture("clone-durable", volumeCoreInput{})
	clock.Value = 191
	file, err = fs.Open("/workspace/next", flags, 0o600)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/next", Handle: "next", Flags: &flags, Mode: 0o600})
	clock.Value = 193
	n, err = file.WriteAt([]byte("pending"), 0)
	require.NoError(t, err)
	require.Equal(t, 7, n)
	capture("write-at", volumeCoreInput{Handle: "next", Data: []byte("pending")})
	for _, seed := range []uint64{0, 1, 7, 42} {
		scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{
			Op: "select", Input: volumeCoreInput{Seed: &seed}, Selections: fs.SelectCrashOperations(seed),
		}))
	}
	require.NoError(t, file.Close())
	capture("close", volumeCoreInput{Handle: "next"})
	clock.Value = 197
	selections := fs.SelectCrashOperations(7)
	fs, err = fs.AdvanceVolumeLifecycle(false, selections)
	require.NoError(t, err)
	capture("advance-crash", volumeCoreInput{Selections: selections})
	clock.Value = 201
	file, err = fs.Open("/workspace/after-crash", flags, 0o600)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/after-crash", Handle: "after-crash", Flags: &flags, Mode: 0o600})
	clock.Value = 203
	n, err = file.WriteAt([]byte("tail"), 0)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	capture("write-at", volumeCoreInput{Handle: "after-crash", Data: []byte("tail")})
	require.NoError(t, file.Close())
	capture("close", volumeCoreInput{Handle: "after-crash"})
	clock.Value = 211
	fs, err = fs.AdvanceVolumeLifecycle(true, nil)
	require.NoError(t, err)
	capture("advance-graceful", volumeCoreInput{})
	volumeCoreImage(t, fs, scenario.Storage, &scenario.Steps[len(scenario.Steps)-1])
	return scenario
}

func volumeCoreNestedOrphanScript(t *testing.T) volumeCoreScenario {
	t.Helper()
	scenario := volumeCoreScenario{Name: "nested-orphan-path-image", Storage: volumeCoreStorage()}
	clock := &volume.Clock{Value: 100}
	fs := volume.New()
	fs.SetClock(clock)
	capture := func(op string, input volumeCoreInput) {
		scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{Op: op, Input: input}))
	}
	require.NoError(t, fs.ConfigureVolumes(scenario.Storage.Volumes, scenario.Storage.Limits))
	capture("configure", volumeCoreInput{})
	require.NoError(t, fs.Mkdir("/workspace/dir", 0o755))
	capture("mkdir", volumeCoreInput{Path: "/workspace/dir", Mode: 0o755})
	flags := volume.OpenFlags{Read: true, Write: true, Create: true}
	file, err := fs.Open("/workspace/dir/db", flags, 0o666)
	require.NoError(t, err)
	capture("open", volumeCoreInput{Path: "/workspace/dir/db", Handle: "db", Flags: &flags, Mode: 0o666})
	clock.Value = 117
	n, err := file.WriteAt([]byte("value"), 0)
	require.NoError(t, err)
	require.Equal(t, 5, n)
	capture("write-at", volumeCoreInput{Handle: "db", Data: []byte("value")})
	require.NoError(t, file.Close())
	capture("close", volumeCoreInput{Handle: "db"})
	selections := map[string][]uint64{"data": {1, 3, 4, 5, 6}}
	fs, err = fs.AdvanceVolumeLifecycle(false, selections)
	require.NoError(t, err, "Go permits durable child path without its parent's namespace operation")
	accepted := true
	scenario.Steps = append(scenario.Steps, volumeCoreCapture(fs, clock, volumeCoreStep{
		Op: "advance-crash", Input: volumeCoreInput{Selections: selections}, Accepted: &accepted,
	}))
	volumeCoreImage(t, fs, scenario.Storage, &scenario.Steps[len(scenario.Steps)-1])
	return scenario
}
