package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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

func storageEnumerationRequest(t *testing.T, recoverImage bool) Request {
	t.Helper()
	body := `i32.const 4 i32.const 0 i32.const 0 i32.const 4 i32.const 1 i64.const 536870911 i64.const 0 i32.const 0 i32.const 100 call $open if unreachable end
 i32.const 200 i32.const 4096 i32.store i32.const 204 i32.const 65536 i32.store
 i32.const 100 i32.load i32.const 200 i32.const 1 i32.const 240 call $write if unreachable end
 i32.const 100 i32.load call $sync if unreachable end
 i32.const 4 call $sync if unreachable end
 loop $dirs
 i32.const 4 i32.const 20 local.get $i i32.const 2 i32.mul i32.add i32.const 2 call $mkdir if unreachable end
 local.get $i i32.const 1 i32.add local.tee $i i32.const 7 i32.lt_u br_if $dirs
 end`
	if recoverImage {
		body = `i32.const 4 i32.const 0 i32.const 0 i32.const 4 i32.const 0 i64.const 536870911 i64.const 0 i32.const 0 i32.const 100 call $open if unreachable end
 i32.const 200 i32.const 4096 i32.store i32.const 204 i32.const 1 i32.store
 i32.const 100 i32.load i32.const 200 i32.const 1 i64.const 0 i32.const 240 call $read if unreachable end
 i32.const 1 i32.const 200 i32.const 1 i32.const 240 call $write if unreachable end`
	}
	wat := fmt.Sprintf(`(module
 (import "wasi_snapshot_preview1" "path_open" (func $open (param i32 i32 i32 i32 i32 i64 i64 i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "fd_write" (func $write (param i32 i32 i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "fd_sync" (func $sync (param i32) (result i32)))
 (import "wasi_snapshot_preview1" "path_create_directory" (func $mkdir (param i32 i32 i32) (result i32)))
 (import "wasi_snapshot_preview1" "fd_pread" (func $read (param i32 i32 i32 i64 i32) (result i32)))
 (import "gomad_wasm_v1" "config" (func $config (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "decision" (func (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "observation" (func (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "finish" (func $finish (param i32 i32) (result i32)))
 (import "gomad_wasm_v1" "idle" (func (param i32 i32) (result i32)))
 (memory (export "memory") 2) (data (i32.const 0) "file")
 (data (i32.const 20) "d0d1d2d3d4d5d6") (data (i32.const 4096) "x")
 (func (export "_start") (local $i i32)
 i32.const 512 i32.const 16 call $config drop
 %s
 i32.const 512 i32.const 16 call $finish drop))`, body)
	imports := []Import{
		{Module: "wasi_snapshot_preview1", Name: "path_open", Params: []string{"i32", "i32", "i32", "i32", "i32", "i64", "i64", "i32", "i32"}, Results: []string{"i32"}},
		{Module: "wasi_snapshot_preview1", Name: "fd_write", Params: []string{"i32", "i32", "i32", "i32"}, Results: []string{"i32"}},
		{Module: "wasi_snapshot_preview1", Name: "fd_sync", Params: []string{"i32"}, Results: []string{"i32"}},
		{Module: "wasi_snapshot_preview1", Name: "path_create_directory", Params: []string{"i32", "i32", "i32"}, Results: []string{"i32"}},
		{Module: "wasi_snapshot_preview1", Name: "fd_pread", Params: []string{"i32", "i32", "i32", "i64", "i32"}, Results: []string{"i32"}},
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
	request.InitialMemoryPages = 2
	request.ReturnStorageImage = true
	request.Runtime = &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, Limit: 8192, ExecutionIdentity: choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte(wat)), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}}}
	if !recoverImage {
		request.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 128, Operations: 32, Depth: 32, Bytes: 64 << 20, Work: 1 << 20}
	}
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
