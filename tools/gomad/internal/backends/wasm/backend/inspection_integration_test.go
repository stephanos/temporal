package backend

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/runner"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func inspectionArtifactFixture(t *testing.T) (*Provider, string) {
	t.Helper()
	provider := integrationProvider(t)
	helper := filepath.Join(t.TempDir(), "inspection-private-helper")
	original, err := os.Open(provider.options.HelperPath)
	require.NoError(t, err)
	copy, err := os.OpenFile(helper, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0500)
	require.NoError(t, err)
	_, err = io.Copy(copy, original)
	require.NoError(t, err)
	require.NoError(t, copy.Close())
	require.NoError(t, original.Close())
	provider.options.HelperPath = helper
	provider.options.CompilerPath = stockgo.Compiler(t)
	provider.options.Config.Profile = wasi.CooperativeProfile
	provider.options.Config.Clock.ReadStepNanos = 0
	provider.options.Fuel = 1 << 60
	owner, err := filepath.Abs("../../../model/runtimesource")
	require.NoError(t, err)
	provider.options.RuntimeRoot = owner
	server, err := filepath.Abs("../../../../../..")
	require.NoError(t, err)
	source, err := os.MkdirTemp(filepath.Join(server, "common/gomaddebug/testdata"), "inspection-")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(source))
		require.NoError(t, os.RemoveAll(source+"-unavailable"))
	})
	fixture, err := os.ReadFile(filepath.Join(server, "common/gomaddebug/testdata/fixture/main.go"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(source, "main.go"), fixture, 0600))
	return provider, source
}

func inspectionSearchWithReapedProcess(t *testing.T, provider *Provider, opened *artifact.Opened, request backends.InspectionRequest) backends.InspectionResult {
	t.Helper()
	if runtime.GOOS == "windows" {
		result, err := provider.Inspect(t.Context(), opened, request)
		require.NoError(t, err)
		return result
	}
	type completion struct {
		report backends.InspectionResult
		err    error
	}
	completed := make(chan completion, 1)
	go func() {
		report, err := provider.Inspect(t.Context(), opened, request)
		completed <- completion{report, err}
	}()
	var pid string
	observeContext, cancelObserve := context.WithTimeout(t.Context(), 4*time.Second)
	defer cancelObserve()
	poll := time.NewTicker(25 * time.Millisecond)
	defer poll.Stop()
	for pid == "" {
		processes, err := exec.Command("ps", "-eo", "pid,args").Output()
		if err == nil {
			for _, line := range strings.Split(string(processes), "\n") {
				if strings.Contains(line, provider.options.HelperPath) {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						pid = fields[0]
						break
					}
				}
			}
		}
		if pid != "" {
			break
		}
		select {
		case <-observeContext.Done():
			t.Fatal("inspection helper process was not observed before deadline")
		case <-poll.C:
		}
	}
	result := <-completed
	require.NoError(t, result.err)
	process, err := exec.Command("ps", "-p", pid, "-o", "pid=").Output()
	require.Error(t, err, "inspection helper PID %s survived cleanup: %s", pid, process)
	return result.report
}

func TestInspectionRetainedArtifactWithoutSourceOrCompiler(t *testing.T) {
	provider, source := inspectionArtifactFixture(t)
	root := t.TempDir()
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 5 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, ChoiceTraceLimit: 1 << 20, Artifacts: filepath.Join(root, "artifacts"), RunnerBuild: "inspection-retained", Coverage: campaignoptions.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, BuildTags: []string{"gomad", "test_dep"}}})
	require.NoError(t, err)
	require.Equal(t, uint64(1), campaign.Failures)
	require.Len(t, campaign.Artifacts, 1)
	opened, err := artifact.OpenArtifact(campaign.Artifacts[0])
	require.NoError(t, err)
	defer func() { require.NoError(t, opened.Close()) }()
	before := opened.Manifest().RecordHash
	reference := opened.Manifest().Target.Backend.Evidence
	retained, err := opened.ReadPayload(reference.File, uint64(reference.Bytes))
	require.NoError(t, err)
	prepared, err := provider.ValidateReplay(t.Context(), opened)
	require.NoError(t, err)
	pr, err := decodeProvenance(prepared.BackendPayloads[0].Data)
	require.NoError(t, err)
	execution, _, err := provider.standaloneInspection(opened, pr, retained)
	require.NoError(t, err)
	require.NotEmpty(t, execution.Inspection.Boundaries)
	last := execution.Inspection.Boundaries[len(execution.Inspection.Boundaries)-1]
	require.NoError(t, os.Rename(source, source+"-unavailable"))
	provider.options.CompilerPath = filepath.Join(root, "compiler-unavailable")
	provider.options.RuntimeRoot = filepath.Join(root, "runtime-source-unavailable")
	request := backends.InspectionRequest{Stop: backends.InspectionStop{Kind: "choice", Value: last.Decision}, Dumps: []string{"stacks", "heap", "fixture.snapshot", "fixture.panic"}, MaximumBytes: backends.DefaultInspectionBytes, PerDumpBytes: backends.DefaultInspectionDumpBytes}
	for range 2 {
		report, err := provider.Inspect(t.Context(), opened, request)
		require.NoError(t, err)
		require.Equal(t, backends.InspectionStatusInspectedPrefix, report.Status)
		require.True(t, report.PrefixVerified)
		require.True(t, report.WitnessMatch)
		require.Equal(t, last.Decision, report.Resolved.Choice)
		debug, err := os.ReadFile(report.OutputPath)
		require.NoError(t, err)
		require.Contains(t, string(debug), `"name":"fixture.snapshot"`)
		require.Contains(t, string(debug), `"status":"panic"`)
		heap, err := os.ReadFile(filepath.Join(filepath.Dir(report.OutputPath), "heap.pprof"))
		require.NoError(t, err)
		require.NotEmpty(t, heap)
	}
	var first uint64
	for _, boundary := range execution.Inspection.Boundaries {
		if boundary.Order >= execution.Inspection.Incarnations[0].Registry.ClosedOrder {
			first = boundary.Decision
			break
		}
	}
	search := backends.InspectionRequest{MaximumBytes: backends.DefaultInspectionBytes, PerDumpBytes: backends.DefaultInspectionDumpBytes, Search: &backends.InspectionSearchRequest{Check: "fixture.before_three", First: first, Last: last.Decision, Mode: "scan", AttemptBudget: 3, WallTime: 30 * time.Second}}
	var firstEvaluations []backends.InspectionSearchEvaluation
	for range 2 {
		report, err := provider.Inspect(t.Context(), opened, search)
		require.NoError(t, err)
		require.NotEmpty(t, report.Search.Evaluations)
		for _, evaluation := range report.Search.Evaluations {
			require.True(t, evaluation.PrefixVerified, "%+v", evaluation)
			require.Contains(t, []string{"pass", "fail"}, evaluation.Outcome)
		}
		if firstEvaluations == nil {
			firstEvaluations = report.Search.Evaluations
		} else {
			require.Equal(t, firstEvaluations, report.Search.Evaluations)
		}
	}
	search.Search.First, search.Search.Last = last.Decision, last.Decision
	search.Search.Mode = "bisect"
	conditional, err := provider.Inspect(t.Context(), opened, search)
	require.NoError(t, err)
	require.Equal(t, "already_failing_at_first", conditional.Search.Status)
	require.True(t, conditional.Search.ConditionalOnMonotonicity)
	search.Search.Check = "fixture.recovering"
	_, err = provider.Inspect(t.Context(), opened, search)
	var monotonicPreflight *backends.InspectionPreflightError
	require.ErrorAs(t, err, &monotonicPreflight)
	search.Search.Mode, search.Search.Check = "scan", "fixture.panic_check"
	search.Search.AttemptBudget = 1
	panicked, err := provider.Inspect(t.Context(), opened, search)
	require.NoError(t, err)
	require.Equal(t, "invariant_evaluation_failure", panicked.Search.Status)
	require.Nil(t, panicked.Search.FirstFailure)
	require.Len(t, panicked.Search.Evaluations, 1)
	require.Equal(t, "error", panicked.Search.Evaluations[0].Outcome)
	require.Equal(t, "panic", panicked.Search.Evaluations[0].Detail)
	search.Search.Check, search.Search.WallTime = "fixture.timeout_check", 5*time.Second
	started := time.Now()
	timedOut := inspectionSearchWithReapedProcess(t, provider, opened, search)
	require.Equal(t, "incomplete", timedOut.Search.Status)
	require.Nil(t, timedOut.Search.FirstFailure)
	require.Len(t, timedOut.Search.Evaluations, 1)
	require.Equal(t, "timeout", timedOut.Search.Evaluations[0].Outcome)
	require.NotNil(t, timedOut.Search.Remaining)
	require.Less(t, time.Since(started), 10*time.Second)
	if runtime.GOOS != "windows" {
		processes, err := exec.Command("ps", "-eo", "pid,args").Output()
		require.NoError(t, err)
		require.NotContains(t, string(processes), provider.options.HelperPath)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	request.Dumps = []string{"fixture.timeout"}
	_, err = provider.Inspect(ctx, opened, request)
	require.Error(t, err)
	if runtime.GOOS != "windows" {
		processes, err := exec.Command("ps", "-eo", "pid,args").Output()
		require.NoError(t, err)
		require.NotContains(t, string(processes), provider.options.HelperPath)
	}
	require.Equal(t, before, opened.Manifest().RecordHash)
	after, err := opened.ReadPayload(reference.File, uint64(reference.Bytes))
	require.NoError(t, err)
	require.Equal(t, retained, after)
}

func TestInspectionRetainedClusterTargetsRestartedIncarnation(t *testing.T) {
	provider, source := inspectionArtifactFixture(t)
	a1, a2, b1 := cluster.NodeHandle{Node: "a", Incarnation: 1}, cluster.NodeHandle{Node: "a", Incarnation: 2}, cluster.NodeHandle{Node: "b", Incarnation: 1}
	scenario := simulation.OperatorScenario{Schema: simulation.OperatorSchema, Spec: cluster.Spec{Schema: cluster.SpecSchema, Backend: cluster.BackendWASM, Fidelity: cluster.FidelitySimulationModel, Limits: cluster.DefaultLimits(), Nodes: []cluster.NodeSpec{{ID: "a", Boot: "main", Address: "10.0.0.1"}, {ID: "b", Boot: "main", Address: "10.0.0.2"}}}, Boots: []simulation.OperatorBoot{{ID: "main"}}, Actions: []simulation.OperatorAction{{Action: "start", Handle: a1}, {Action: "start", Handle: b1}, {Action: "wait", Handle: a1}, {Action: "wait", Handle: b1}, {Action: "restart", Handle: a2}, {Action: "wait", Handle: a2}}}
	encoded, err := json.Marshal(scenario)
	require.NoError(t, err)
	campaign, err := runner.Explore(t.Context(), runner.CampaignSpec{Backend: provider, Seeds: "7", Parallel: 1, ExecutionTimeout: time.Minute, OverallTimeout: 5 * time.Minute, TerminateGrace: 100 * time.Millisecond, OnFailure: runner.PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), RunnerBuild: "inspection-cluster", Coverage: campaignoptions.CoverageNone, Target: target.Spec{Backend: Name, Kind: target.KindGoRun, Source: ".", WorkingDir: source, BuildTags: []string{"gomad", "test_dep"}, Scenario: encoded}})
	require.NoError(t, err)
	require.Len(t, campaign.Artifacts, 1)
	opened, err := artifact.OpenArtifact(campaign.Artifacts[0])
	require.NoError(t, err)
	defer func() { require.NoError(t, opened.Close()) }()
	ref := *opened.Manifest().Target.Backend.Evidence
	data, err := opened.ReadPayload(ref.File, uint64(ref.Bytes))
	require.NoError(t, err)
	_, retained, err := simulation.VerifyRetained(target.BackendPayload{Reference: ref, Data: data})
	require.NoError(t, err)
	var last cluster.WASMInspectionBoundary
	for _, boundary := range retained.WASM.Inspection.Boundaries {
		if boundary.Handle == a2 {
			last = boundary
		}
	}
	require.Equal(t, a2, last.Handle)
	require.NoError(t, os.Rename(source, source+"-unavailable"))
	provider.options.CompilerPath = filepath.Join(t.TempDir(), "unavailable")
	request := backends.InspectionRequest{Instance: "a", Incarnation: 2, Stop: backends.InspectionStop{Kind: "choice", Value: last.Decision}, Dumps: []string{"stacks", "fixture.snapshot"}, MaximumBytes: backends.DefaultInspectionBytes, PerDumpBytes: backends.DefaultInspectionDumpBytes}
	report, err := provider.Inspect(t.Context(), opened, request)
	require.NoError(t, err)
	require.True(t, report.PrefixVerified)
	require.Equal(t, uint64(2), report.Resolved.Incarnation)
	require.Len(t, report.PrefixCuts, 3)
	for _, cut := range report.PrefixCuts {
		require.NotEqual(t, "never", cut.Instance)
	}
	request.Instance, request.Incarnation = "", 0
	_, err = provider.Inspect(t.Context(), opened, request)
	var preflight *backends.InspectionPreflightError
	require.ErrorAs(t, err, &preflight)
	request.Instance, request.Incarnation = "a", 2
	request.Stop = backends.InspectionStop{}
	request.Dumps = nil
	request.Search = &backends.InspectionSearchRequest{Check: "fixture.before_three", First: last.Decision, Last: last.Decision, Mode: "scan", AttemptBudget: 1, WallTime: 30 * time.Second}
	search, err := provider.Inspect(t.Context(), opened, request)
	require.NoError(t, err)
	require.Len(t, search.Search.Evaluations, 1)
	require.True(t, search.Search.Evaluations[0].PrefixVerified)
	request.Search.Check = "fixture.panic_check"
	panicked, err := provider.Inspect(t.Context(), opened, request)
	require.NoError(t, err)
	require.Equal(t, "invariant_evaluation_failure", panicked.Search.Status)
	require.Nil(t, panicked.Search.FirstFailure)
	require.Equal(t, "panic", panicked.Search.Evaluations[0].Detail)
	request.Search.Check, request.Search.WallTime = "fixture.timeout_check", 5*time.Second
	started := time.Now()
	timedOut := inspectionSearchWithReapedProcess(t, provider, opened, request)
	require.Equal(t, "incomplete", timedOut.Search.Status)
	require.Nil(t, timedOut.Search.FirstFailure)
	require.Len(t, timedOut.Search.Evaluations, 1)
	require.Equal(t, "timeout", timedOut.Search.Evaluations[0].Outcome)
	require.Less(t, time.Since(started), 10*time.Second)
	request.Search = nil
	request.Stop = backends.InspectionStop{Kind: "choice", Value: last.Decision}
	request.Dumps = []string{"fixture.timeout"}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	_, err = provider.Inspect(ctx, opened, request)
	require.Error(t, err)
	if runtime.GOOS != "windows" {
		processes, err := exec.Command("ps", "-eo", "pid,args").Output()
		require.NoError(t, err)
		require.NotContains(t, string(processes), provider.options.HelperPath)
	}
}
