package measure

import (
	"crypto/sha256"
	"path/filepath"

	wasi "go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func standaloneConfig() wasi.Config {
	return wasi.Config{Profile: wasi.StockProfile, Args: []string{"guest", "literal $arg"}, Environment: []string{"PWD=/workspace", "TZ=UTC", "ONLY=captured"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, EntropyKey: [32]byte{1, 2, 3}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: 1000}, Limits: wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 8 << 20, Calls: 10000, Descriptors: 64, Files: 1000, FilesystemBytes: 8 << 20, PendingEvents: 4096}}
}

func temporalRecoveryConfig(root string) (wasi.Config, error) {
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(root, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
	if err != nil {
		return wasi.Config{}, err
	}
	return wasi.Config{WorkingDirectory: "/workspace", Environment: []string{"PWD=/workspace", "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "GOMAD_TEMPORAL_DATABASE=/volume/temporal.db", "GOMAD_TEMPORAL_SCHEMA=/workspace/schema/sqlite/v3"}, WritableDirectories: []string{"/workspace", "/tmp", "/volume"}, CapturedInputs: inputs, EntropyKey: sha256.Sum256([]byte("gomad-wasm-temporal-recovery/v1")), Clock: wasi.ClockPolicy{EpochNanos: 1700000000000000000, ReadStepNanos: 0}, Limits: wasi.Limits{OutputBytes: 8 << 20, TranscriptBytes: 128 << 20, Calls: 4000000, Descriptors: 4096, Files: 10000, FilesystemBytes: 512 << 20, PendingEvents: 4096}, Storage: &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: "temporal", Path: "/volume", CapacityBytes: 256 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 100000, Transitions: 10000000}, SQLiteLockDatabases: []string{"/volume/temporal.db"}}}, nil
}
