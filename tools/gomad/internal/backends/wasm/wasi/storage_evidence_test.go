package wasi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
