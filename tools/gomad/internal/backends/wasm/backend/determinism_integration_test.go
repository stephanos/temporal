package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/determinism"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/determinism/observed"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/runner"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func determinismRetainedDirectory(t *testing.T, prefix string) string {
	t.Helper()
	root := os.Getenv("GOMAD_WASM_DETERMINISM_EVIDENCE_ROOT")
	if root == "" {
		return t.TempDir()
	}
	require.True(t, filepath.IsAbs(root))
	base := filepath.Join(root, t.Name())
	require.NoError(t, os.MkdirAll(base, 0700))
	directory, err := os.MkdirTemp(base, prefix)
	require.NoError(t, err)
	return directory
}

func determinismReplayPrefix(t *testing.T, provider *Provider) []string {
	t.Helper()
	directory := determinismRetainedDirectory(t, "operator-")
	binary := filepath.Join(directory, "gomad")
	build := exec.CommandContext(t.Context(), provider.options.CompilerPath, "build", "-p=1", "-trimpath", "-o", binary, "../../../../cmd/gomad")
	output, err := build.CombinedOutput()
	require.NoError(t, err, "build actual replay CLI: %s", output)
	config, err := json.Marshal(provider.options.Config)
	require.NoError(t, err)
	configPath := filepath.Join(directory, "config.json")
	require.NoError(t, os.WriteFile(configPath, config, 0600))
	manifest := filepath.Join(directory, "gomad-wasm.json")
	install := exec.CommandContext(t.Context(), binary, "install", "--compiler", provider.options.CompilerPath, "--helper", provider.options.HelperPath, "--runtime-root", provider.options.RuntimeRoot, "--cache", provider.options.CacheRoot, "--profile", wasi.CooperativeProfile, "--config", configPath, "--memory-bytes", fmt.Sprint(provider.options.MemoryBytes), "--fuel", fmt.Sprint(provider.options.Fuel), "--output", manifest)
	output, err = install.CombinedOutput()
	require.NoError(t, err, "install actual replay CLI: %s", output)
	return []string{binary, "--installation", manifest, "replay", "--determinism-fixture"}
}

func determinismArtifactFixture(t *testing.T) (*Provider, target.Spec) {
	t.Helper()
	if os.Getenv("GOMAD_WASM_DETERMINISM_FIXTURES") != "1" {
		t.Skip("requires the deliberate-fault helper and an explicit live-fixture resource slot")
	}
	helper := os.Getenv("GOMAD_WASM_DETERMINISM_HELPER")
	require.True(t, filepath.IsAbs(helper), "build --features determinism-fixture and set GOMAD_WASM_DETERMINISM_HELPER")
	provider := integrationProvider(t)
	helperBytes, err := os.ReadFile(helper)
	require.NoError(t, err)
	pinnedHelper := filepath.Join(determinismRetainedDirectory(t, "helper-"), "gomad-wasmhost")
	require.NoError(t, os.WriteFile(pinnedHelper, helperBytes, 0700))
	provider.options.HelperPath, provider.options.HelperSHA256 = pinnedHelper, string(record.HashBytes(helperBytes))
	provider.options.Config.Profile = wasi.CooperativeProfile
	provider.options.Config.Clock.ReadStepNanos = 0
	provider.options.Config.Limits.Calls = 1 << 20
	provider.options.Fuel = 1 << 60
	provider.options.RuntimeRoot = determinismRetainedDirectory(t, "runtime-")
	for _, name := range []string{"gomad_semantics.go.txt", "gomad_choicewire_generated.go.txt"} {
		data, err := os.ReadFile(filepath.Join("../../../model/runtimesource", name))
		require.NoError(t, err)
		if name == "gomad_semantics.go.txt" {
			anchor := "func gomadChoiceSeedRandom() {"
			require.Equal(t, 1, strings.Count(string(data), anchor))
			data = []byte(strings.Replace(string(data), anchor, anchor+`
	gomadDeterminismFixturePending = gomadDeterminismFixturePending || gomadSeed&(1<<63) != 0
	gomadDeterminismFixtureOutputPending = gomadDeterminismFixtureOutputPending || gomadSeed&(1<<62) != 0
	gomadSeed &= ^uint64((1<<63)|(1<<62))
`, 1) + `
var gomadDeterminismFixturePending bool
var gomadDeterminismFixtureOutputPending bool

//go:linkname gomadDeterminismFixtureOutput
func gomadDeterminismFixtureOutput() bool { return gomadDeterminismFixtureOutputPending }

//go:linkname gomadDeterminismFixtureDraw
func gomadDeterminismFixtureDraw() {
	if gomadDeterminismFixturePending {
		gomadDeterminismFixturePending = false
		gomadChoiceRunqSeeded(2)
	}
}
`)
		}
		require.NoError(t, os.WriteFile(filepath.Join(provider.options.RuntimeRoot, name), data, 0600))
	}
	source, err := filepath.Abs("../../../../../../common/gomaddebug/testdata/determinism")
	require.NoError(t, err)
	return provider, target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, BuildTags: []string{"gomad", "test_dep"}}
}

func recordDeterminismArtifact(t *testing.T, provider *Provider, spec target.Spec, fault string, replayPrefix []string) determinism.Run {
	t.Helper()
	var execution ExecutionObservation
	var configurationErr error
	provider.options.ObserveResults = func(result ExecutionObservation) { execution = result }
	provider.options.StressExecution = &StressExecution{Diagnostics: true, EventOrder: true, ChoiceTraceLimit: 1 << 20, ConfigureRequest: func(request *wasi.Request) {
		configurationErr = request.SetDeterminismFixtureConditions(fault)
	}}
	choices, diagnostics := uint64(4<<20), true
	if len(spec.Scenario) != 0 {
		choices, diagnostics = 0, false
	}
	artifacts := t.TempDir()
	if root := os.Getenv("GOMAD_WASM_DETERMINISM_EVIDENCE_ROOT"); root != "" {
		require.True(t, filepath.IsAbs(root))
		cell := fault
		if cell == "" {
			cell = "reference"
		}
		artifacts = filepath.Join(root, t.Name(), cell)
		require.NoError(t, os.MkdirAll(artifacts, 0700))
	}
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{
		Backend: provider, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 5 * time.Minute,
		TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1,
		OutputLimit: 1 << 20, WorldTransitionLimit: 4 << 20, IOTranscriptLimit: 64 << 20,
		ChoiceTraceLimit: choices, Diagnostics: diagnostics, Artifacts: artifacts,
		RunnerBuild: "determinism-executed-fixtures", Coverage: campaignoptions.CoverageNone, Target: spec,
	})
	require.NoError(t, err)
	require.NoError(t, configurationErr)
	require.Equal(t, uint64(1), campaign.Failures)
	require.Len(t, campaign.Artifacts, 1)
	require.NotEmpty(t, execution.Incarnations)
	metadata := execution.Incarnations[0].Result.InspectionMetadata
	if execution.Cluster != nil {
		opened, err := artifact.OpenArtifact(campaign.Artifacts[0])
		require.NoError(t, err)
		reference := *opened.Manifest().Target.Backend.Evidence
		data, readErr := opened.ReadPayload(reference.File, uint64(reference.Bytes))
		closeErr := opened.Close()
		require.NoError(t, readErr)
		require.NoError(t, closeErr)
		_, retained, err := simulation.VerifyRetained(target.BackendPayload{Reference: reference, Data: data})
		require.NoError(t, err)
		require.NotNil(t, retained.WASM)
		require.NotNil(t, retained.WASM.Inspection)
		metadata, err = json.Marshal(retained.WASM.Inspection)
		require.NoError(t, err)
	}
	module, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	run := determinism.Run{Workload: spec.Source, Seed: 7, Perturbation: fault, ArtifactDir: campaign.Artifacts[0],
		ReplayPrefix:   replayPrefix,
		Command:        []string{"env", "GOMAD_WASM_DETERMINISM_FIXTURES=1", "GOMAD_WASM_DETERMINISM_HELPER=" + provider.options.HelperPath, "GOMAD_WASM_TEST_HELPER=" + provider.options.HelperPath, "GOMAD_STOCK_GO=" + os.Getenv("GOMAD_STOCK_GO"), "go", "-C", module, "test", "-tags", "test_dep", "./internal/backends/wasm/backend", "-run", "^" + t.Name() + "$", "-count=1"},
		Identities:     map[string]string{"module": execution.Target.SHA256, "source": execution.Target.BuildKey, "helper": provider.options.HelperSHA256, "model": provider.options.ModelSHA256, "engine": fmt.Sprint(provider.options.Engine)},
		EvidenceDigest: string(record.HashBytes(metadata)),
	}
	for _, incarnation := range execution.Incarnations {
		require.Equal(t, "exit", incarnation.Result.Termination)
		require.NotEmpty(t, incarnation.Result.DiagnosticTrace.Bytes)
		run.Incarnations = append(run.Incarnations, determinism.Incarnation{Instance: string(incarnation.Handle.Node), Incarnation: incarnation.Handle.Incarnation, Diagnostic: incarnation.Result.DiagnosticTrace.Bytes})
	}
	run, err = observed.FromInspection(run, metadata)
	require.NoError(t, err)
	stderr := execution.Incarnations[0].Result.HelperStderr()
	if execution.Cluster != nil {
		stderr = execution.Cluster.HelperStderr()
	}
	log, err := determinism.ParseEventOrder(stderr)
	require.NoError(t, err)
	require.True(t, log.Complete, "%s", log.Reason)
	run.OrderLog = &log
	return run
}

func TestDeterminismExecutedDivergenceArtifacts(t *testing.T) {
	for _, test := range []struct {
		fault   string
		kind    determinism.EventKind
		cluster bool
	}{
		{"draw", determinism.KindDrawCounter, false},
		{"clock", determinism.KindClock, false},
		{"handoff", determinism.KindWorldOrder, true},
		{"output-clock", determinism.KindOutput, false},
	} {
		t.Run(test.fault, func(t *testing.T) {
			provider, spec := determinismArtifactFixture(t)
			if test.cluster {
				a, b := model.NodeHandle{Node: "a", Incarnation: 1}, model.NodeHandle{Node: "b", Incarnation: 1}
				scenario := simulation.OperatorScenario{Schema: simulation.OperatorSchema,
					Spec:  model.Spec{Schema: model.SpecSchema, Backend: model.BackendWASM, Fidelity: model.FidelitySimulationModel, Limits: model.DefaultLimits(), Nodes: []model.NodeSpec{{ID: "a", Boot: "main", Address: "10.0.0.1"}, {ID: "b", Boot: "main", Address: "10.0.0.2"}}},
					Boots: []simulation.OperatorBoot{{ID: "main"}}, Actions: []simulation.OperatorAction{{Action: "start", Handle: a}, {Action: "start", Handle: b}, {Action: "wait", Handle: a}, {Action: "wait", Handle: b}},
				}
				var err error
				spec.Scenario, err = json.Marshal(scenario)
				require.NoError(t, err)
			}
			replayPrefix := determinismReplayPrefix(t, provider)
			expected := recordDeterminismArtifact(t, provider, spec, "", replayPrefix)
			actual := recordDeterminismArtifact(t, provider, spec, test.fault, replayPrefix)
			report, err := determinism.Compare(expected, actual)
			require.NoError(t, err)
			require.NotNil(t, report.First, "%s", report.Format())
			event := report.First
			require.Equal(t, "helper-event", report.Ordering, "%s", report.Format())
			require.Equal(t, test.kind, event.Kind, "%s", report.Format())
			require.NotEmpty(t, event.Instance)
			require.Equal(t, uint64(1), event.Incarnation)
			require.NotEqual(t, event.Expected, event.Actual)
			require.Positive(t, event.At.Shared)
			if test.fault == "draw" {
				var before, after determinism.OrderedEvent
				require.NoError(t, json.Unmarshal([]byte(event.Expected), &before))
				require.NoError(t, json.Unmarshal([]byte(event.Actual), &after))
				var left, right map[string]uint64
				require.NoError(t, json.Unmarshal(before.Value, &left))
				require.NoError(t, json.Unmarshal(after.Value, &right))
				require.Equal(t, left["runq_draws"]+1, right["runq_draws"])
			}
			require.NotNil(t, report.Reproducer)
			require.Empty(t, report.Reproducer.PrefixUnavailable, "%s", report.Format())
			require.Contains(t, report.Reproducer.ReplayCommand, expected.ArtifactDir)
			if test.fault == "output-clock" {
				var output, clock, nextBoundary uint64
				output = event.At.Shared
				for _, marker := range actual.OrderLog.Boundaries {
					if marker.Order >= output && (nextBoundary == 0 || marker.Order < nextBoundary) {
						nextBoundary = marker.Order
					}
				}
				for _, candidate := range actual.OrderLog.Events {
					if candidate.Order > output && candidate.Kind == determinism.KindClock {
						clock = candidate.Order
						break
					}
				}
				require.Greater(t, clock, output, "guest output must commit before its later clock read")
				require.LessOrEqual(t, clock, nextBoundary, "both real effects must share one retained boundary interval")
			}
			if !test.cluster && test.fault != "output-clock" {
				require.GreaterOrEqual(t, len(expected.Boundaries), 9)
				difference, err := choice.DiffDiagnostics(expected.Incarnations[0].Diagnostic, actual.Incarnations[0].Diagnostic)
				require.NoError(t, err)
				require.NotNil(t, difference)
				require.NotNil(t, difference.Expected)
				require.NotNil(t, difference.Actual)
				if test.fault == "draw" {
					require.Equal(t, []string{"runq_draws"}, difference.Fields)
					require.Equal(t, difference.Expected.RunqDraws+1, difference.Actual.RunqDraws)
				} else {
					require.Equal(t, []string{"virtual_time"}, difference.Fields)
					require.Equal(t, difference.Expected.VirtualTime+1, difference.Actual.VirtualTime)
				}
			}
			provider.options.StressExecution.ConfigureRequest = func(request *wasi.Request) { require.NoError(t, request.SetDeterminismFixtureConditions("")) }
			opened, err := artifact.OpenArtifact(expected.ArtifactDir)
			require.NoError(t, err)
			defer func() { require.NoError(t, opened.Close()) }()
			request := report.Reproducer.Stop
			if root := os.Getenv("GOMAD_WASM_DETERMINISM_EVIDENCE_ROOT"); root != "" {
				request.OutputDirectory = filepath.Join(determinismRetainedDirectory(t, "inspection-"), "output")
			}
			inspection, err := provider.Inspect(t.Context(), opened, request)
			require.NoError(t, err)
			require.Equal(t, backends.InspectionStatusInspectedPrefix, inspection.Status)
			require.True(t, inspection.PrefixVerified)
			require.True(t, inspection.WitnessMatch)
			require.Equal(t, report.Reproducer.Stop.Stop.Value, inspection.Resolved.Choice)
			cliCommand := report.Reproducer.ReplayCommand
			cli := exec.CommandContext(t.Context(), cliCommand[0], cliCommand[1:]...)
			cliOutput, err := cli.CombinedOutput()
			require.NoError(t, err, "literal emitted replay command %q: %s", cliCommand, cliOutput)
			var cliInspection backends.InspectionResult
			require.NoError(t, json.Unmarshal(cliOutput, &cliInspection))
			require.Equal(t, backends.InspectionStatusInspectedPrefix, cliInspection.Status)
			require.True(t, cliInspection.PrefixVerified)
			require.True(t, cliInspection.WitnessMatch)
			require.Equal(t, report.Reproducer.Stop.Stop.Value, cliInspection.Resolved.Choice)
			if root := os.Getenv("GOMAD_WASM_DETERMINISM_EVIDENCE_ROOT"); root != "" {
				receipt, err := json.MarshalIndent(struct {
					Report        determinism.Report
					Inspection    backends.InspectionResult
					CLIInspection backends.InspectionResult
				}{report, inspection, cliInspection}, "", "  ")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(root, t.Name(), "prefix-receipt.json"), receipt, 0600))
			}
			t.Log(report.Format())
		})
	}
}
