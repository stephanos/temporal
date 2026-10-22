package wasi

import (
	"crypto/sha256"
	"fmt"

	captured "go.temporal.io/server/tools/gomad/internal/model/capture"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
)

type TemporalPolicy string

const (
	TemporalPrototype     TemporalPolicy = "prototype"
	TemporalInstallation  TemporalPolicy = "installation"
	TemporalInspection    TemporalPolicy = "inspection"
	TemporalQualification TemporalPolicy = "qualification"
)

// TemporalConfig applies one complete guest configuration policy to captured schema inputs.
func TemporalConfig(inputs captured.CapturedInputs, cwd string, policy TemporalPolicy, args ...string) (Config, error) {
	config := Config{
		Profile: CooperativeProfile, WorkingDirectory: cwd,
		Args:                append([]string{"temporal.test", "-test.v", "-test.count=1", "-test.timeout=72h"}, args...),
		Environment:         []string{"PWD=" + cwd, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn"},
		WritableDirectories: []string{"/workspace", "/tmp", cwd},
		CapturedInputs:      inputs,
		EntropyKey:          sha256.Sum256([]byte("gomad-wasm-prototype/v1")),
		Clock:               ClockPolicy{EpochNanos: 946684800000000000},
		Limits:              Limits{OutputBytes: 256 << 20, TranscriptBytes: 1 << 30, Calls: maxCooperativeCalls, Descriptors: 4096, Files: 100000, FilesystemBytes: 512 << 20, PendingEvents: 4096},
	}
	switch policy {
	case TemporalPrototype:
	case TemporalInstallation:
		config.NetworkV2 = temporalNetwork()
	case TemporalInspection:
		config.Args = nil
		config.Environment = append([]string{"PWD=" + cwd, "GOMAD_DEBUG_CANARY=1"}, config.Environment[1:]...)
		config.WritableDirectories = []string{"/workspace", "/tmp"}
		config.EntropyKey = [32]byte{}
		config.Limits.OutputBytes = 16 << 20
		config.Limits.TranscriptBytes = 256 << 20
		config.NetworkV2 = temporalNetwork()
	case TemporalQualification:
		config.Args = nil
	default:
		return Config{}, fmt.Errorf("unknown Temporal configuration policy %q", policy)
	}
	if len(args) != 0 && (policy == TemporalInspection || policy == TemporalQualification) {
		return Config{}, fmt.Errorf("the Temporal configuration policy %q takes arguments from Runner", policy)
	}
	return config, nil
}

func temporalNetwork() *NetworkModelV2 {
	return &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}}
}
