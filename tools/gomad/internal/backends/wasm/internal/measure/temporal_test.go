package measure

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
)

func TestPrototypeTemporalConfigurationPreservesCompleteBytes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	schema := filepath.Join(root, "schema", "sqlite")
	require.NoError(t, os.MkdirAll(schema, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(schema, "schema.sql"), []byte("CREATE TABLE example(id INTEGER);"), 0600))
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: schema, Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
	require.NoError(t, err)
	cwd := "/workspace/tests"
	want := wasi.Config{
		Profile: wasi.CooperativeProfile, WorkingDirectory: cwd,
		Args:                []string{"temporal.test", "-test.v", "-test.count=1", "-test.timeout=72h", "-test.run=^TestActivityTestSuite$", "-persistenceType=sql", "-persistenceDriver=sqlite"},
		Environment:         []string{"PWD=" + cwd, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn"},
		WritableDirectories: []string{"/workspace", "/tmp", cwd}, CapturedInputs: inputs,
		EntropyKey: sha256.Sum256([]byte("gomad-wasm-prototype/v1")), Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000},
		Limits: wasi.Limits{OutputBytes: 256 << 20, TranscriptBytes: 1 << 30, Calls: 1 << 28, Descriptors: 4096, Files: 100000, FilesystemBytes: 512 << 20, PendingEvents: 4096},
	}
	config, err := prototypeTemporalConfig(t.Context(), root, cwd, "-test.run=^TestActivityTestSuite$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	require.NoError(t, err)
	actual, err := json.Marshal(config)
	require.NoError(t, err)
	expected, err := json.Marshal(want)
	require.NoError(t, err)
	require.Equal(t, string(expected), string(actual))
}

func TestPrototypeTemporalConfigurationPreservesCaptureError(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, want := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(root, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
	require.Error(t, want)
	_, err := prototypeTemporalConfig(t.Context(), root, "/workspace/tests")
	require.EqualError(t, err, want.Error())
}
