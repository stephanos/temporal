package backend

import (
	"encoding/json"
	"fmt"
	"os"
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
	provider.options.HelperPath, provider.options.HelperSHA256 = helper, string(record.HashBytes(helperBytes))
	provider.options.Config.Profile = wasi.CooperativeProfile
	provider.options.Config.Clock.ReadStepNanos = 0
	provider.options.Config.Limits.Calls = 1 << 20
	provider.options.Fuel = 1 << 60
	provider.options.RuntimeRoot = t.TempDir()
	for _, name := range []string{"gomad_semantics.go.txt", "gomad_choicewire_generated.go.txt"} {
		data, err := os.ReadFile(filepath.Join("../../../model/runtimesource", name))
		require.NoError(t, err)
		if name == "gomad_semantics.go.txt" {
			anchor := "func gomadChoiceSeedRandom() {"
			require.Equal(t, 1, strings.Count(string(data), anchor))
			data = []byte(strings.Replace(string(data), anchor, anchor+`
	gomadDeterminismFixturePending = gomadDeterminismFixturePending || gomadSeed&(1<<63) != 0
	gomadSeed &= ^uint64(1<<63)
`, 1) + `
var gomadDeterminismFixturePending bool

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

func recordDeterminismArtifact(t *testing.T, provider *Provider, spec target.Spec, fault string) determinism.Run {
	t.Helper()
	var execution ExecutionObservation
	var configurationErr error
	provider.options.ObserveResults = func(result ExecutionObservation) { execution = result }
	provider.options.StressExecution = &StressExecution{Diagnostics: true, ConfigureRequest: func(request *wasi.Request) {
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
		metadata = execution.Cluster.InspectionMetadata
	}
	module, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	run := determinism.Run{Workload: spec.Source, Seed: 7, Perturbation: fault, ArtifactDir: campaign.Artifacts[0],
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
			expected := recordDeterminismArtifact(t, provider, spec, "")
			actual := recordDeterminismArtifact(t, provider, spec, test.fault)
			report, err := determinism.Compare(expected, actual)
			require.NoError(t, err)
			require.NotNil(t, report.First, "%s", report.Format())
			var event *determinism.Divergence
			for index := range report.BoundaryEvents {
				candidate := &report.BoundaryEvents[index]
				require.Equal(t, report.First.At.Shared, candidate.At.Shared)
				if candidate.Kind == test.kind {
					event = candidate
				}
			}
			require.NotNil(t, event, "%s", report.Format())
			require.NotEmpty(t, event.Instance)
			require.Equal(t, uint64(1), event.Incarnation)
			require.NotEqual(t, event.Expected, event.Actual)
			require.Positive(t, event.At.Shared)
			if test.fault == "draw" {
				require.Contains(t, event.Field, "runq_draws")
			}
			require.NotNil(t, report.Reproducer)
			require.Empty(t, report.Reproducer.PrefixUnavailable, "%s", report.Format())
			require.Contains(t, report.Reproducer.ReplayCommand, expected.ArtifactDir)
			if !test.cluster {
				require.GreaterOrEqual(t, len(expected.Boundaries), 9)
				require.Equal(t, expected.Boundaries[8].At, event.At)
				require.Equal(t, expected.Boundaries[7].Decision, report.Reproducer.Stop.Stop.Value)
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
				request.OutputDirectory, err = os.MkdirTemp(filepath.Join(root, t.Name()), "inspection-")
				require.NoError(t, err)
			}
			inspection, err := provider.Inspect(t.Context(), opened, request)
			require.NoError(t, err)
			require.Equal(t, backends.InspectionStatusInspectedPrefix, inspection.Status)
			require.True(t, inspection.PrefixVerified)
			require.True(t, inspection.WitnessMatch)
			require.Equal(t, report.Reproducer.Stop.Stop.Value, inspection.Resolved.Choice)
			if root := os.Getenv("GOMAD_WASM_DETERMINISM_EVIDENCE_ROOT"); root != "" {
				receipt, err := json.MarshalIndent(struct {
					Report     determinism.Report
					Inspection backends.InspectionResult
				}{report, inspection}, "", "  ")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(root, t.Name(), "prefix-receipt.json"), receipt, 0600))
			}
			t.Log(report.Format())
		})
	}
}
