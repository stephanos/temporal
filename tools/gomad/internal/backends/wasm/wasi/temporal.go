package wasi

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"time"

	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
)

// TemporalConfig captures the immutable schema and the admitted Temporal guest defaults.
func TemporalConfig(root, cwd string, args ...string) (Config, error) {
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(root, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
	if err != nil {
		return Config{}, fmt.Errorf("capture Temporal schema/sqlite: %w", err)
	}
	return Config{
		Profile: CooperativeProfile, WorkingDirectory: cwd,
		Args:                append([]string{"temporal.test", "-test.v", "-test.count=1", "-test.timeout=72h"}, args...),
		Environment:         []string{"PWD=" + cwd, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn"},
		WritableDirectories: []string{"/workspace", "/tmp", cwd},
		CapturedInputs:      inputs,
		EntropyKey:          sha256.Sum256([]byte("gomad-wasm-prototype/v1")),
		Clock:               ClockPolicy{EpochNanos: 946684800000000000},
		Limits:              Limits{OutputBytes: 256 << 20, TranscriptBytes: 1 << 30, Calls: maxCooperativeCalls, Descriptors: 4096, Files: 100000, FilesystemBytes: 512 << 20, PendingEvents: 4096},
	}, nil
}

// TemporalNetwork returns the bounded network configuration of the one-box fixture.
func TemporalNetwork() *NetworkModelV2 {
	return &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: uint64(10 * time.Second), Transitions: 1 << 20}}
}
