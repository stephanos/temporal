package wasi

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	captured "go.temporal.io/server/tools/gomad/internal/model/capture"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/support/temporalinput"
)

func TestTemporalConfigPreservesPrototypeBytes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	schema := filepath.Join(root, "schema/sqlite")
	require.NoError(t, os.MkdirAll(schema, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(schema, "schema.sql"), []byte("CREATE TABLE example(id INTEGER);"), 0600))
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: schema, Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
	require.NoError(t, err)
	cwd := "/workspace/tests/gomadfunctional"
	want := Config{
		Profile: CooperativeProfile, WorkingDirectory: cwd,
		Args:                []string{"temporal.test", "-test.v", "-test.count=1", "-test.timeout=72h", "-test.run=^TestFrontendSystemInfo$"},
		Environment:         []string{"PWD=" + cwd, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn"},
		WritableDirectories: []string{"/workspace", "/tmp", cwd}, CapturedInputs: inputs,
		EntropyKey: sha256.Sum256([]byte("gomad-wasm-prototype/v1")), Clock: ClockPolicy{EpochNanos: 946684800000000000},
		Limits: Limits{OutputBytes: 256 << 20, TranscriptBytes: 1 << 30, Calls: 1 << 28, Descriptors: 4096, Files: 100000, FilesystemBytes: 512 << 20, PendingEvents: 4096},
	}
	schemaInputs, err := temporalinput.CaptureSchema(root)
	require.NoError(t, err)
	config, err := TemporalConfig(schemaInputs, cwd, TemporalPrototype, "-test.run=^TestFrontendSystemInfo$")
	require.NoError(t, err)
	wantBytes, err := json.Marshal(want)
	require.NoError(t, err)
	actual, err := json.Marshal(config)
	require.NoError(t, err)
	require.Equal(t, string(wantBytes), string(actual))
	_, err = NewEnvironment(config)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(schema, "schema.sql"), []byte("changed schema"), 0600))
	changedInputs, err := temporalinput.CaptureSchema(root)
	require.NoError(t, err)
	changed, err := TemporalConfig(changedInputs, cwd, TemporalPrototype)
	require.NoError(t, err)
	require.NotEqual(t, config.CapturedInputs, changed.CapturedInputs)
}

func TestTemporalConfigPoliciesPreserveCompleteBytes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	schema := filepath.Join(root, "schema/sqlite")
	require.NoError(t, os.MkdirAll(schema, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(schema, "schema.sql"), []byte("CREATE TABLE example(id INTEGER);"), 0600))
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: schema, Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
	require.NoError(t, err)
	cwd := "/workspace/tests/gomadfunctional"
	schemaInputs, err := temporalinput.CaptureSchema(root)
	require.NoError(t, err)
	prototype := Config{
		Profile: CooperativeProfile, WorkingDirectory: cwd,
		Args:                []string{"temporal.test", "-test.v", "-test.count=1", "-test.timeout=72h"},
		Environment:         []string{"PWD=" + cwd, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn"},
		WritableDirectories: []string{"/workspace", "/tmp", cwd}, CapturedInputs: inputs,
		EntropyKey: sha256.Sum256([]byte("gomad-wasm-prototype/v1")), Clock: ClockPolicy{EpochNanos: 946684800000000000},
		Limits: Limits{OutputBytes: 256 << 20, TranscriptBytes: 1 << 30, Calls: 1 << 28, Descriptors: 4096, Files: 100000, FilesystemBytes: 512 << 20, PendingEvents: 4096},
	}
	network := &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}}
	installed := prototype
	installed.NetworkV2 = network
	qualification := prototype
	qualification.Args = nil
	inspection := Config{
		Profile: CooperativeProfile, WorkingDirectory: "/workspace",
		Environment:         []string{"PWD=/workspace", "GOMAD_DEBUG_CANARY=1", "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn"},
		WritableDirectories: []string{"/workspace", "/tmp"}, CapturedInputs: inputs,
		Clock:     ClockPolicy{EpochNanos: 946684800000000000},
		Limits:    Limits{OutputBytes: 16 << 20, TranscriptBytes: 256 << 20, Calls: 1 << 28, Descriptors: 4096, Files: 100000, FilesystemBytes: 512 << 20, PendingEvents: 4096},
		NetworkV2: network,
	}
	for _, test := range []struct {
		policy TemporalPolicy
		want   Config
	}{
		{TemporalPrototype, prototype}, {TemporalInstallation, installed}, {TemporalInspection, inspection}, {TemporalQualification, qualification},
	} {
		t.Run(string(test.policy), func(t *testing.T) {
			t.Parallel()
			config, err := TemporalConfig(schemaInputs, test.want.WorkingDirectory, test.policy)
			require.NoError(t, err)
			actual, err := json.Marshal(config)
			require.NoError(t, err)
			want, err := json.Marshal(test.want)
			require.NoError(t, err)
			require.Equal(t, string(want), string(actual))
			_, err = NewEnvironment(config)
			require.NoError(t, err)
		})
	}
	_, err = TemporalConfig(schemaInputs, cwd, TemporalPolicy("unknown"))
	require.ErrorContains(t, err, "Temporal configuration policy")
	for _, policy := range []TemporalPolicy{TemporalInspection, TemporalQualification} {
		_, err = TemporalConfig(schemaInputs, cwd, policy, "-test.v")
		require.ErrorContains(t, err, "takes arguments from Runner")
	}
}
