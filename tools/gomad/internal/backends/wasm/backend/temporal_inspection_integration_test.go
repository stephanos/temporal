package backend

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/runner"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func TestTemporalInspectionRetainedFailure(t *testing.T) {
	retained := os.Getenv("GOMAD_WASM_TEMPORAL_INSPECTION_ROOT")
	if retained == "" {
		t.Skip("set GOMAD_WASM_TEMPORAL_INSPECTION_ROOT for the dedicated retained Temporal inspection gate")
	}
	server, err := filepath.Abs("../../../../../..")
	require.NoError(t, err)
	for _, fixture := range []struct {
		owner, test, dump, probe, snapshot string
	}{
		{"shard", "TestGomadShardDumpUsesSnapshotWithLockHeld", "temporal.shards", "temporal.shard.snapshot", "shard=7 range=19"},
		{"workflow", "TestGomadMutableStateDumpUsesSnapshotWithWorkflowLockHeld", "temporal.mutable_states", "temporal.mutable_state.snapshot", "persisted_next_event=11"},
		{"onebox", "TestReplayInspectionWorkflowFailure", "temporal.mutable_states", "temporal.*.snapshot", "snapshot="},
	} {
		t.Run(fixture.owner, func(t *testing.T) {
			provider := integrationProvider(t)
			provider.options.CompilerPath = stockgo.Compiler(t)
			provider.options.Config.Profile = wasi.CooperativeProfile
			provider.options.Config.Clock.ReadStepNanos = 0
			provider.options.Config.Environment = append(provider.options.Config.Environment, "GOMAD_DEBUG_CANARY=1")
			provider.options.MemoryBytes = 1 << 30
			provider.options.Fuel = 50_000_000_000
			provider.options.RuntimeRoot = filepath.Join(server, "tools/gomad/internal/model/runtimesource")
			executionTimeout, overallTimeout := 5*time.Minute, 15*time.Minute
			environment := []string{"GOMAD_DEBUG_CANARY=1"}
			original := filepath.Join(server, "service/history", fixture.owner)
			if fixture.owner == "onebox" {
				original = filepath.Join(server, "tools/gomad/internal/backends/wasm/testdata/prototype_temporal")
				inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(server, "schema/sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
				require.NoError(t, err)
				provider.options.Config.CapturedInputs = inputs
				provider.options.Config.Environment = append(provider.options.Config.Environment, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC", "TEMPORAL_TEST_LOG_LEVEL=warn")
				environment = append(environment, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TEMPORAL_TEST_LOG_LEVEL=warn")
				provider.options.Config.Limits = wasi.Limits{OutputBytes: 16 << 20, TranscriptBytes: 256 << 20, Calls: 1 << 28, Descriptors: 4096, Files: 100000, FilesystemBytes: 512 << 20, PendingEvents: 4096}
				provider.options.Config.NetworkV2 = &wasi.NetworkModelV2{Profile: wasi.NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: uint64(10 * time.Second), Transitions: 1 << 20}}
				provider.options.MemoryBytes = 4 << 30
				provider.options.Fuel = 1 << 50
				executionTimeout, overallTimeout = 30*time.Minute, 45*time.Minute
			}
			source, err := os.MkdirTemp(filepath.Join(server, "service/history"), ".gomad-inspection-"+fixture.owner+"-")
			require.NoError(t, err)
			t.Cleanup(func() {
				require.NoError(t, os.RemoveAll(source))
				require.NoError(t, os.RemoveAll(source+"-unavailable"))
			})
			entries, err := os.ReadDir(original)
			require.NoError(t, err)
			for _, entry := range entries {
				name := entry.Name()
				if entry.IsDir() || !strings.HasSuffix(name, ".go") || fixture.owner != "onebox" && strings.HasSuffix(name, "_test.go") && name != "gomad_debug_test.go" {
					continue
				}
				data, err := os.ReadFile(filepath.Join(original, name))
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(source, name), data, 0600))
			}
			root := filepath.Join(retained, fixture.owner)
			require.NoError(t, os.MkdirAll(root, 0700))
			provider.options.CacheRoot = filepath.Join(root, "cache")
			provider.options.CompileCacheRoot = filepath.Join(root, "engine-cache")
			require.NoError(t, os.MkdirAll(provider.options.CompileCacheRoot, 0700))
			targetSpec := target.Spec{Backend: Name, Kind: target.KindGoTest, Source: ".", WorkingDir: source, BuildTags: append(TemporalGuestBuildTags(), "gomad"), Args: []string{"-test.run=^" + fixture.test + "$", "-test.v"}}
			choiceBytes := uint64(8 << 20)
			if fixture.owner == "onebox" {
				choiceBytes = 0
				limits := cluster.DefaultLimits()
				limits.NetworkConnectionBytes = provider.options.Config.NetworkV2.Limits.ConnectionBytes
				limits.NetworkStallNanos = provider.options.Config.NetworkV2.Limits.StallNanos
				handle := cluster.NodeHandle{Node: "history", Incarnation: 1}
				scenario := simulation.OperatorScenario{Schema: simulation.OperatorSchema, Spec: cluster.Spec{Schema: cluster.SpecSchema, Backend: cluster.BackendWASM, Fidelity: cluster.FidelitySimulationModel, Limits: limits, Nodes: []cluster.NodeSpec{{ID: handle.Node, Boot: "main", Address: "127.0.0.1"}}}, Boots: []simulation.OperatorBoot{{ID: "main", Args: targetSpec.Args}}, Actions: []simulation.OperatorAction{{Action: "start", Handle: handle}, {Action: "wait", Handle: handle}}}
				targetSpec.Scenario, err = json.Marshal(scenario)
				require.NoError(t, err)
			}
			campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Environment: environment, Seeds: "7", Parallel: 1, ExecutionTimeout: executionTimeout, OverallTimeout: overallTimeout, TerminateGrace: time.Second, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, ChoiceTraceLimit: choiceBytes, Artifacts: filepath.Join(root, "artifacts"), RunnerBuild: "fn180-temporal-inspection", Coverage: campaignoptions.CoverageNone, Target: targetSpec})
			require.NoError(t, err)
			require.Equal(t, uint64(1), campaign.Failures)
			require.Len(t, campaign.Artifacts, 1)
			opened, err := artifact.OpenArtifact(campaign.Artifacts[0])
			require.NoError(t, err)
			defer func() { require.NoError(t, opened.Close()) }()
			if fixture.owner == "onebox" {
				stdout, err := os.ReadFile(filepath.Join(campaign.Artifacts[0], "stdout"))
				require.NoError(t, err)
				require.Contains(t, string(stdout), "workflow completed [step 0 attempt 2 step 1 attempt 2]")
				require.Contains(t, string(stdout), "Temporal inspection canary after workflow timers and activity retries completed")
			}
			before := opened.Manifest().RecordHash
			ref := opened.Manifest().Target.Backend.Evidence
			evidence, err := opened.ReadPayload(ref.File, uint64(ref.Bytes))
			require.NoError(t, err)
			require.NoError(t, os.Rename(source, source+"-unavailable"))
			provider.options.CompilerPath = filepath.Join(root, "compiler-unavailable")
			provider.options.RuntimeRoot = filepath.Join(root, "runtime-source-unavailable")
			if fixture.owner == "onebox" {
				payload := target.BackendPayload{Reference: *ref, Data: evidence}
				module := simulation.RetainedModule{SHA256: string(opened.Manifest().Target.SHA256), Copy: func(destination string) error {
					return opened.CopyPayload(opened.Manifest().Target.File, destination, 0700)
				}}
				observed, expected, _, err := simulation.ReplayBoundRetained(t.Context(), payload, module, root, simulation.InstalledHelper{Path: provider.options.HelperPath, SHA256: strings.TrimPrefix(provider.options.HelperSHA256, "sha256:"), Engine: provider.options.Engine})
				require.NoError(t, err)
				encoded, err := cluster.EncodeClusterRecord(observed.Record)
				require.NoError(t, err)
				require.Equal(t, expected, record.HashBytes(encoded))
			} else {
				replay, err := runner.Replay(t.Context(), runner.ReplaySpec{ArtifactPath: campaign.Artifacts[0], Backend: provider})
				require.NoError(t, err)
				require.True(t, replay.Match)
			}
			request := backends.InspectionRequest{Stop: backends.InspectionStop{Kind: "failure"}, Dumps: []string{"stacks", "heap", fixture.dump}, Probes: []string{fixture.probe}, MaximumBytes: backends.DefaultInspectionBytes, PerDumpBytes: backends.DefaultInspectionDumpBytes, OutputDirectory: filepath.Join(root, "inspection")}
			if fixture.owner == "onebox" {
				request.Instance, request.Incarnation = "history", 1
				request.Dumps = append(request.Dumps, "temporal.shards")
			}
			report, err := provider.Inspect(t.Context(), opened, request)
			require.NoError(t, err)
			require.Equal(t, backends.InspectionStatusInspectedPrefix, report.Status)
			require.True(t, report.PrefixVerified)
			require.True(t, report.WitnessMatch)
			encoded, err := os.ReadFile(report.OutputPath)
			require.NoError(t, err)
			var output wasi.DebugOutput
			require.NoError(t, json.Unmarshal(encoded, &output))
			require.NotEmpty(t, output.Stacks)
			heap, err := os.ReadFile(filepath.Join(filepath.Dir(report.OutputPath), "heap.pprof"))
			require.NoError(t, err)
			require.NotEmpty(t, heap)
			require.NotEmpty(t, output.Probes)
			if fixture.owner == "onebox" {
				probeNames := map[string]bool{}
				for _, probe := range output.Probes {
					probeNames[probe.Name] = true
				}
				require.True(t, probeNames["temporal.shard.snapshot"])
				require.True(t, probeNames["temporal.mutable_state.snapshot"])
			}
			found := false
			shards := false
			for _, dump := range output.Dumps {
				if dump.Name == fixture.dump {
					found = true
					require.Equal(t, "ok", dump.Status)
					require.True(t, bytes.Contains(dump.Data, []byte(fixture.snapshot)), string(dump.Data))
					require.Contains(t, string(dump.Data), "values may precede stop")
				}
				if fixture.owner == "onebox" && dump.Name == "temporal.shards" {
					shards = true
					require.Equal(t, "ok", dump.Status)
					require.Contains(t, string(dump.Data), "snapshot=")
				}
			}
			require.True(t, found)
			if fixture.owner == "onebox" {
				require.True(t, shards)
			}
			require.Equal(t, before, opened.Manifest().RecordHash)
			after, err := opened.ReadPayload(ref.File, uint64(ref.Bytes))
			require.NoError(t, err)
			require.Equal(t, evidence, after)
			reopened, err := artifact.OpenArtifact(campaign.Artifacts[0])
			require.NoError(t, err)
			require.Equal(t, before, reopened.Manifest().RecordHash)
			require.NoError(t, reopened.Close())
			t.Logf("retained Temporal failure artifact=%s record=%s debug=%s resolved=%+v", campaign.Artifacts[0], before, report.OutputPath, report.Resolved)
		})
	}
}
