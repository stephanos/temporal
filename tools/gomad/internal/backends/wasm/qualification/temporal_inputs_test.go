package qualification

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

// TestPrepareTemporalQualificationInputs reconstructs configuration-only inputs
// from the fn-178 producer; it never prepares a module or dispatches a helper.
func TestPrepareTemporalQualificationInputs(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_INPUTS_ROOT")
	if root == "" {
		t.Skip("set an absolute fresh GOMAD_WASM_QUALIFICATION_INPUTS_ROOT to retain current configuration")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == string(filepath.Separator) {
		t.Fatal("fresh absolute non-root configuration directory required")
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	checkout, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(checkout, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	network := &wasi.NetworkModelV2{Profile: wasi.NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}}
	broker := wasi.BrokerConfig{
		Nodes: []wasi.BrokerNode{{ID: "active", Incarnation: 1}, {ID: "standby", Incarnation: 1}}, Seed: 7,
		Limits: world.CompressedSnapshotProfile().Limits, WorldProfile: world.CompressedSnapshotSchema,
		Network: network, Addresses: map[string]string{"active": "10.0.0.1", "standby": "10.0.0.2"}, Diagnostics: true,
		Faults: []wasi.BrokerNetworkFault{
			{Name: "temporal-active-to-standby-partition", Action: "partition", From: "active", To: "standby", AtNanos: 1},
			{Name: "temporal-active-to-standby-heal", Action: "heal", From: "active", To: "standby", AtNanos: uint64(30 * time.Second)},
		},
	}
	if _, err := wasi.NewBroker(broker); err != nil {
		t.Fatal(err)
	}
	files := map[string]any{"two-cluster-broker-config.json": broker}
	for index, role := range []string{"active", "standby"} {
		peer := []string{"10.0.0.2", "10.0.0.1"}[index]
		config := wasi.Config{
			Profile: wasi.CooperativeProfile, WorkingDirectory: "/workspace",
			Args:                []string{"replication.test", "-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestIndependentClusterReplicationRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"},
			Environment:         []string{"PWD=/workspace", "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "GOMAD_TEMPORAL_DATABASE=/volume/temporal.db", "GOMAD_TEMPORAL_SCHEMA=/workspace/schema/sqlite/v3", "GOMAD_TEMPORAL_CLUSTER=" + role, "GOMAD_TEMPORAL_PHASE=commit", "GOMAD_TEMPORAL_REMOTE_FRONTEND=" + peer + ":7233", "GOMAD_TEMPORAL_REMOTE_HTTP=" + peer + ":7243", "GOMAD_TEMPORAL_REMOTE_HISTORY=" + peer + ":7234"},
			WritableDirectories: []string{"/workspace", "/tmp", "/volume"}, CapturedInputs: inputs,
			EntropyKey: sha256.Sum256([]byte("gomad-wasm-temporal-replication/v1/" + role)),
			Clock:      wasi.ClockPolicy{EpochNanos: 1700000000000000000},
			Limits:     wasi.Limits{OutputBytes: 8 << 20, TranscriptBytes: 128 << 20, Calls: 4000000, Descriptors: 4096, Files: 10000, FilesystemBytes: 512 << 20, PendingEvents: 4096},
			Storage:    &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: role, Path: "/volume", CapacityBytes: 256 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 100000, Transitions: 10000000}, SQLiteLockDatabases: []string{"/volume/temporal.db"}},
			NetworkV2:  network,
		}
		if _, err := wasi.NewEnvironment(config); err != nil {
			t.Fatal(err)
		}
		files[role+"-cluster-commit.request.json"] = wasi.Request{MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 15 * time.Minute, DigestTranscript: true, ReturnStorageImage: true}
	}
	for name, value := range files {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		t.Logf("configuration=%s sha256=%s bytes=%d", path, record.HashBytes(data), len(data))
	}
}
