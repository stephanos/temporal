package wasi

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVolumeRecoveryAdmissionCorpus(t *testing.T) {
	type fixture struct {
		Name         string                         `json:"name"`
		Session      volumeConformanceSession       `json:"session"`
		InitialReply []byte                         `json:"initial_reply_base64"`
		Operations   []volumeConformanceOperation   `json:"operations"`
		Expected     []volumeConformanceObservation `json:"observations"`
	}
	name := filepath.Join("testdata", "volume-conformance", "vectors", "recovery-admission.json")
	if *volumeConformanceOutput == "" {
		data, err := os.ReadFile(name)
		require.NoError(t, err)
		var cases []fixture
		require.NoError(t, json.Unmarshal(data, &cases))
		require.NotEmpty(t, cases)
		helper := namespaceConformanceHelper(t)
		for _, test := range cases {
			t.Run(test.Name, func(t *testing.T) {
				exchange, closeDriver := namespaceConformanceTransport(t, helper, test.Name)
				defer closeDriver()
				require.Equal(t, test.InitialReply, exchange(test.Session))
				for i, operation := range test.Operations {
					var got volumeConformanceObservation
					require.NoError(t, json.Unmarshal(exchange(operation), &got))
					require.Equal(t, test.Expected[i], got, "step=%d", i)
				}
			})
		}
		return
	}
	config := storageConfig()
	config.Storage.SQLiteLockDatabases = []string{"/workspace/db"}
	source := newTestEnvironment(t, config)
	image, err := source.CrashStorage(nil)
	require.NoError(t, err)
	valid, digest, err := volumeConformanceEncodeImage(image, storageImageLimit(config))
	require.NoError(t, err)
	wrongModel, _, err := encodeStorageImage(image.fs, []byte("{}"), image.occurrences, storageImageLimit(config))
	require.NoError(t, err)
	lockConfig := storageConfig()
	lockedSource := newTestEnvironment(t, lockConfig)
	fd := openPath(t, lockedSource, 4, "db.lock", 1, rightsAll)
	output(t, call(t, lockedSource, "fd_sync", fmt.Sprintf(`{"fd":%d}`, fd)), &struct{}{})
	output(t, call(t, lockedSource, "fd_sync", `{"fd":4}`), &struct{}{})
	lockImage, err := lockedSource.CrashStorage(nil)
	require.NoError(t, err)
	lockImage.model = image.model
	locked, _, err := volumeConformanceEncodeImage(lockImage, storageImageLimit(config))
	require.NoError(t, err)
	_, err = NewRecoveredEnvironment(config, lockImage)
	require.EqualError(t, err, "WASI invalid: durable image contains process-owned SQLite lock")
	wrong := *image
	wrong.model = []byte("{}")
	_, err = NewRecoveredEnvironment(config, &wrong)
	require.EqualError(t, err, "WASI invalid: recovery storage model identity")
	selectOp := volumeConformanceOp("select", map[string]any{"seed": 0})
	before := volumeConformanceRun(t, volumeConformanceScript{config: config, operations: []volumeConformanceOperation{selectOp}})
	validRecovery := volumeConformanceRun(t, volumeConformanceScript{config: config, operations: []volumeConformanceOperation{volumeConformanceOp("crash", map[string]any{"image": "good"}), volumeConformanceOp("recover", map[string]string{"image": "good"})}})
	require.Equal(t, valid, validRecovery.Observations[1].ImageData)
	require.Equal(t, digest, validRecovery.Observations[1].ImageDigest)
	recoverOp := func(data []byte, digest string) volumeConformanceOperation {
		return volumeConformanceOp("recover", map[string]any{"image_base64": data, "image_digest": digest})
	}
	var cases []fixture
	for _, test := range []struct {
		name, boundary string
		data           []byte
		digest         string
	}{
		{"model", "invalid:recovery storage model identity", wrongModel, ""}, {"lock", "invalid:durable image contains process-owned SQLite lock", locked, ""}, {"truncated", "invalid:durable storage image", valid[:len(valid)-1], ""}, {"digest", "invalid:durable storage image", valid, strings.Repeat("0", 64)},
	} {
		if test.digest == "" {
			test.digest = fmt.Sprintf("%x", sha256.Sum256(test.data))
		}
		refused := before.Observations[0]
		refused.Reply = []byte(fmt.Sprintf(`{"boundary":%q}`, test.boundary))
		refused.Selections = nil
		cases = append(cases, fixture{Name: test.name, Session: before.Session, InitialReply: before.InitialReply, Operations: []volumeConformanceOperation{selectOp, recoverOp(test.data, test.digest), recoverOp(valid, digest)}, Expected: []volumeConformanceObservation{before.Observations[0], refused, validRecovery.Observations[1]}})
	}
	volumeBoundaryWrite(t, "recovery-admission.json", cases)
}
