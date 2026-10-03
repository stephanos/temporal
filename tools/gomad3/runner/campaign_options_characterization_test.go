package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

// campaignOptionsCharacterizationPath pins, for a table of campaign requests,
// what campaign validation returns and what the parent sends to the isolated
// coordinator. It was captured before campaign options gained one owner; the
// same table must keep producing the same selections, environments, error
// texts, error precedence and campaign plan records. The request bytes were
// re-pinned once when the options were nested by group; the legacy
// projection, which flattens the groups again, still equals the bytes pinned
// before.
const campaignOptionsCharacterizationPath = "testdata/campaign_options_characterization.json"

// campaignOptionsCharacterizationWrite rewrites the pinned table instead of
// comparing against it.
const campaignOptionsCharacterizationWrite = "GOMAD3_WRITE_CAMPAIGN_OPTIONS_CHARACTERIZATION"

// characterizationChildTimeout stands in for the child deadline that the
// isolated parent derives from its own remaining time.
const characterizationChildTimeout = 987654321 * time.Nanosecond

// characterizationCoordinator is a coordinator command that cannot start, so
// an isolated Explore that passes every request check stops at the process
// boundary with a stable error.
const characterizationCoordinator = "/nonexistent/gomad3-characterization-coordinator"

type campaignOptionsCase struct {
	name string
	spec CampaignSpec
	// dependencies are the private substitutions the request runs with.
	dependencies dependencies
	// frozenGuidance attaches the frozen guidance a resumed or sharded
	// campaign restores from its recorded plan.
	frozenGuidance *campaign.GuidancePlan
	// explore also runs Explore through the isolated path, whose request
	// checks precede and follow campaign validation.
	explore bool
	// exploreLocal also runs Explore through the local path.
	exploreLocal bool
	// pinBytes keeps the complete request bytes in the table, not only their
	// digest, so a representative request of each shape stays readable.
	pinBytes bool
}

type campaignOptionsObservation struct {
	Name           string        `json:"name"`
	Selection      string        `json:"selection"`
	SelectionCount uint64        `json:"selection_count"`
	Environment    string        `json:"environment"`
	Error          string        `json:"error"`
	IsolatedError  string        `json:"isolated_error,omitempty"`
	LocalError     string        `json:"local_error,omitempty"`
	RequestSHA256  record.SHA256 `json:"request_sha256"`
	Request        string        `json:"request,omitempty"`
	LegacySHA256   record.SHA256 `json:"legacy_request_sha256"`
	LegacyRequest  string        `json:"legacy_request,omitempty"`
	PlanSHA256     record.SHA256 `json:"plan_sha256,omitempty"`
	Plan           string        `json:"plan,omitempty"`
}

func TestCampaignOptionsCharacterization(t *testing.T) {
	cases := campaignOptionsCases(t)
	observed := make([]campaignOptionsObservation, 0, len(cases))
	for _, test := range cases {
		observed = append(observed, observeCampaignOptions(t, test))
	}
	if os.Getenv(campaignOptionsCharacterizationWrite) != "" {
		encoded, err := json.MarshalIndent(observed, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(campaignOptionsCharacterizationPath, append(encoded, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	contents, err := os.ReadFile(campaignOptionsCharacterizationPath)
	if err != nil {
		t.Fatal(err)
	}
	var pinned []campaignOptionsObservation
	if err := json.Unmarshal(contents, &pinned); err != nil {
		t.Fatal(err)
	}
	if len(pinned) != len(observed) {
		t.Fatalf("characterization table has %d cases, pinned %d", len(observed), len(pinned))
	}
	for index, got := range observed {
		want := pinned[index]
		t.Run(got.Name, func(t *testing.T) {
			if got.Name != want.Name {
				t.Fatalf("case %d = %q, pinned %q", index, got.Name, want.Name)
			}
			if got.Selection != want.Selection || got.SelectionCount != want.SelectionCount || got.Environment != want.Environment {
				t.Errorf("selection %q/%d environment %q, pinned %q/%d %q", got.Selection, got.SelectionCount, got.Environment, want.Selection, want.SelectionCount, want.Environment)
			}
			if got.Error != want.Error || got.IsolatedError != want.IsolatedError || got.LocalError != want.LocalError {
				t.Errorf("errors validate=%q isolated=%q local=%q, pinned validate=%q isolated=%q local=%q", got.Error, got.IsolatedError, got.LocalError, want.Error, want.IsolatedError, want.LocalError)
			}
			if got.RequestSHA256 != want.RequestSHA256 || got.Request != want.Request {
				t.Errorf("coordinator request %s %s, pinned %s %s", got.RequestSHA256, got.Request, want.RequestSHA256, want.Request)
			}
			if got.PlanSHA256 != want.PlanSHA256 || got.Plan != want.Plan {
				t.Errorf("campaign plan record %s %s, pinned %s %s", got.PlanSHA256, got.Plan, want.PlanSHA256, want.Plan)
			}
			if got.LegacySHA256 != want.LegacySHA256 || got.LegacyRequest != want.LegacyRequest {
				t.Errorf("legacy coordinator request projection %s %s, pinned %s %s", got.LegacySHA256, got.LegacyRequest, want.LegacySHA256, want.LegacyRequest)
			}
		})
	}
}

func observeCampaignOptions(t *testing.T, test campaignOptionsCase) campaignOptionsObservation {
	t.Helper()
	observation := campaignOptionsObservation{Name: test.name}
	selection, environment, err := characterizeValidation(test.spec, test.dependencies, test.frozenGuidance)
	observation.Selection, observation.SelectionCount = selection.String(), selection.Count()
	entries := make([]string, len(environment))
	for index, entry := range environment {
		entries[index] = entry.Name + "=" + entry.Value
	}
	observation.Environment = strings.Join(entries, " ")
	observation.Error = characterizationError(t, err)
	if err == nil && test.spec.ResumeCampaign == "" {
		plan, err := characterizePlanRecord(test.spec, test.frozenGuidance, selection, environment)
		if err != nil {
			t.Fatalf("%s: campaign plan record: %v", test.name, err)
		}
		observation.PlanSHA256 = record.HashBytes(plan)
		if test.pinBytes {
			observation.Plan = string(plan)
		}
	}
	request, legacy, err := characterizeCoordinatorRequest(test.spec)
	if err != nil {
		t.Fatalf("%s: encode coordinator request: %v", test.name, err)
	}
	observation.RequestSHA256, observation.LegacySHA256 = record.HashBytes(request), record.HashBytes(legacy)
	if test.pinBytes {
		observation.Request, observation.LegacyRequest = string(request), string(legacy)
	}
	if test.explore {
		isolated := test.spec
		isolated.CoordinatorCommand = []string{characterizationCoordinator, "__coordinator"}
		if len(test.spec.CoordinatorCommand) != 0 {
			isolated.CoordinatorCommand = test.spec.CoordinatorCommand
		}
		_, err := exploreWith(context.Background(), isolated, test.dependencies)
		observation.IsolatedError = characterizationError(t, err)
	}
	if test.exploreLocal {
		_, err := exploreWith(context.Background(), test.spec, test.dependencies)
		observation.LocalError = characterizationError(t, err)
	}
	return observation
}

// characterizationError removes the only host-dependent text an error can
// carry, the working directory a relative resume path resolves against.
func characterizationError(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		return ""
	}
	workingDir, wdErr := os.Getwd()
	if wdErr != nil {
		t.Fatal(wdErr)
	}
	return strings.ReplaceAll(err.Error(), workingDir, "$WORKDIR")
}

func campaignOptionsCases(t *testing.T) []campaignOptionsCase {
	t.Helper()
	seed := func() CampaignSpec {
		return CampaignSpec{
			Seeds: "7-9", Parallel: 2, ExecutionTimeout: 3 * time.Second, OverallTimeout: time.Minute, TerminateGrace: time.Second,
			OnFailure: PolicyAll, FailureBudget: 1, OutputLimit: 4096, WorldTransitionLimit: 8192,
			Artifacts: "/artifacts", Environment: []string{"ZETA=1", "ALPHA=2"},
			Target:            target.Spec{Kind: target.KindGoRun, Source: "./cmd/target", WorkingDir: "/", ToolchainRoot: "/toolchain", BuildTags: []string{"tag"}},
			SupervisorCommand: []string{"/bin/gomad", "__supervisor"}, RunnerBuild: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ProgressInterval: 250 * time.Millisecond,
		}
	}
	choiceSpec := func() CampaignSpec {
		spec := seed()
		spec.Strategy = StrategyChoiceExploration
		spec.Seeds = "7"
		spec.ChoiceTraceLimit = execution.MinimumChoiceTraceBytes
		spec.MaxExecutions = 8
		spec.MaxChoiceDepth = 4
		spec.ChoiceStartOrdinal = 2
		spec.MaxExplorationBytes = 1 << 20
		return spec
	}
	simulationSpec := func() CampaignSpec {
		spec := seed()
		spec.Strategy = StrategySimulationExploration
		spec.Seeds = "89"
		spec.OnFailure = PolicyBudget
		spec.FailureBudget = 4
		spec.ChoiceTraceLimit = 9 << 20
		spec.MaxExecutions = 16
		spec.MaxForcedDecisions = 3
		spec.MaxExplorationBytes = 2 << 20
		spec.MaxExplorationResultBytes = 3 << 20
		spec.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 5, Scenario: 11, Network: 13, Storage: 17, Fault: 19, Crash: 23}
		return spec
	}
	resumeSpec := func() CampaignSpec {
		return CampaignSpec{
			ResumeCampaign: "/campaigns/v1/resumed", RunnerBuild: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Target: target.Spec{ToolchainRoot: "/toolchain"}, SupervisorCommand: []string{"/bin/gomad", "__supervisor"}, ProgressInterval: time.Second,
		}
	}
	planIdentity := record.SHA256("sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	frozen := &campaign.GuidancePlan{Corpus: "/corpus", RequestedSelection: "7-9", RequestedCount: 3, AnsweredSeeds: []record.Uint64String{8}, AnsweredCount: 1, GuidedCount: 1}
	with := func(base func() CampaignSpec, configure func(*CampaignSpec)) CampaignSpec {
		spec := base()
		configure(&spec)
		return spec
	}
	falseOverride := false
	cases := []campaignOptionsCase{
		// Accepted requests for each strategy and campaign shape.
		{name: "seed default strategy", spec: seed(), explore: true, pinBytes: true},
		{name: "seed explicit strategy", spec: with(seed, func(spec *CampaignSpec) { spec.Strategy = StrategySeed }), explore: true},
		{name: "seed budget policy", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure, spec.FailureBudget = PolicyBudget, 3 })},
		{name: "seed first policy", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure = PolicyFirst })},
		{name: "seed observation", spec: with(seed, func(spec *CampaignSpec) {
			spec.ChoiceTraceLimit, spec.Diagnostics, spec.ClockTick = 1<<20, true, record.ClockTickForward
			spec.IOTranscriptLimit, spec.Coverage = 128<<20, CoverageSemanticChoice
		}), explore: true},
		{name: "seed strict clock tick", spec: with(seed, func(spec *CampaignSpec) { spec.ClockTick = record.ClockTickStrict })},
		{name: "seed read-only mounts", spec: with(seed, func(spec *CampaignSpec) { spec.IOROMounts = []string{"usr=/mnt/usr"} })},
		{name: "seed read-only mount limits", spec: with(seed, func(spec *CampaignSpec) {
			spec.IOROMounts = []string{"usr=/mnt/usr"}
			spec.IOROMountLimits = readonlymount.DefaultLimits()
		})},
		{name: "seed keep all successes", spec: with(seed, func(spec *CampaignSpec) {
			spec.KeepSuccesses, spec.SuccessArtifactLimit, spec.SuccessBytesLimit = KeepSuccessesAll, 2, 1<<20
		})},
		{name: "seed keep novel successes", spec: with(seed, func(spec *CampaignSpec) {
			spec.Coverage, spec.KeepSuccesses, spec.SuccessArtifactLimit, spec.SuccessBytesLimit = CoverageSemantic, KeepSuccessesNovel, 2, 1<<20
		})},
		{name: "seed execution evidence", spec: with(seed, func(spec *CampaignSpec) {
			spec.Seeds, spec.Coverage, spec.CollectExecutionEvidence = "7", CoverageSemantic, true
		})},
		{name: "guided", spec: with(seed, func(spec *CampaignSpec) {
			spec.Guide, spec.GuideRegression, spec.Corpus, spec.Coverage = true, true, "/corpus", CoverageSemantic
			spec.GuideSnapshotSHA256 = planIdentity
		}), explore: true},
		{name: "sharded", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 1, Count: 3}, planIdentity
		}), explore: true, pinBytes: true},
		{name: "sharded frozen guidance", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 0, Count: 2}, planIdentity
			spec.Guide, spec.Corpus, spec.Coverage = true, "/corpus", CoverageSemantic
		}), frozenGuidance: frozen},
		{name: "choice exploration", spec: choiceSpec(), explore: true, pinBytes: true},
		{name: "simulation exploration", spec: simulationSpec(), explore: true, pinBytes: true},
		{name: "resume", spec: resumeSpec(), pinBytes: true},
		{name: "resume explicit regression override", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.GuideRegressionOverride = &falseOverride })},
		{name: "resume missing campaign", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.ResumeCampaign = filepath.Join("missing", "v1", "resumed") }), explore: true, exploreLocal: true},

		// Isolated request checks around campaign validation.
		{name: "isolated injected executor", spec: seed(), dependencies: dependencies{executor: &fakeExecutor{}}, explore: true},
		{name: "isolated injected preparer", spec: with(seed, func(spec *CampaignSpec) { spec.Preparer = targetPreparer{} }), explore: true},
		{name: "isolated empty coordinator command", spec: with(seed, func(spec *CampaignSpec) { spec.CoordinatorCommand = []string{""} }), explore: true},
		{name: "isolated empty coordinator command after invalid request", spec: with(seed, func(spec *CampaignSpec) {
			spec.CoordinatorCommand, spec.Parallel = []string{""}, 0
		}), explore: true},

		// Resume rejections.
		{name: "resume with preparer", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.Preparer = targetPreparer{} })},
		{name: "resume without Runner build", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.RunnerBuild = "" })},
		{name: "resume without supervisor", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.SupervisorCommand = nil })},
		{name: "resume ignores fresh request faults", spec: with(resumeSpec, func(spec *CampaignSpec) { spec.Seeds, spec.Parallel, spec.Strategy = "bad", -1, "bogus" })},

		// Selection and sharding rejections.
		{name: "malformed seeds", spec: with(seed, func(spec *CampaignSpec) { spec.Seeds = "x" })},
		{name: "empty seeds", spec: with(seed, func(spec *CampaignSpec) { spec.Seeds = "" })},
		{name: "shard index outside count", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 3, Count: 3}, planIdentity
		})},
		{name: "shard without plan identity", spec: with(seed, func(spec *CampaignSpec) { spec.Shard = CampaignShard{Index: 0, Count: 2} })},
		{name: "shard malformed plan identity", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 0, Count: 2}, "sha256:short"
		})},
		{name: "shard choice strategy", spec: with(choiceSpec, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 0, Count: 2}, planIdentity
		})},
		{name: "shard guidance without frozen plan", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256 = CampaignShard{Index: 0, Count: 2}, planIdentity
			spec.Guide, spec.Corpus, spec.Coverage = true, "/corpus", CoverageSemantic
		})},
		{name: "shard first failure policy", spec: with(seed, func(spec *CampaignSpec) {
			spec.Shard, spec.PlanSHA256, spec.OnFailure = CampaignShard{Index: 0, Count: 2}, planIdentity, PolicyFirst
		})},

		// Strategy bound rejections.
		{name: "seed max executions", spec: with(seed, func(spec *CampaignSpec) { spec.MaxExecutions = 1 })},
		{name: "seed choice depth", spec: with(seed, func(spec *CampaignSpec) { spec.MaxChoiceDepth = 1 })},
		{name: "seed choice start ordinal", spec: with(seed, func(spec *CampaignSpec) { spec.ChoiceStartOrdinal = 1 })},
		{name: "seed forced decisions", spec: with(seed, func(spec *CampaignSpec) { spec.MaxForcedDecisions = 1 })},
		{name: "seed exploration bytes", spec: with(seed, func(spec *CampaignSpec) { spec.MaxExplorationBytes = 1 })},
		{name: "seed exploration result bytes", spec: with(seed, func(spec *CampaignSpec) { spec.MaxExplorationResultBytes = 1 })},
		{name: "seed dimension limits", spec: with(seed, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Crash = 1 })},
		{name: "choice forced decisions", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.MaxForcedDecisions = 1 })},
		{name: "choice result bytes", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.MaxExplorationResultBytes = 1 })},
		{name: "choice dimension limits", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Runtime = 1 })},
		{name: "choice multiple seeds", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.Seeds = "7-8" })},
		{name: "choice guidance", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.Guide, spec.Corpus, spec.Coverage = true, "/corpus", CoverageSemantic })},
		{name: "choice without trace", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.ChoiceTraceLimit = 0 })},
		{name: "choice without max executions", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.MaxExecutions = 0 })},
		{name: "choice without depth", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.MaxChoiceDepth = 0 })},
		{name: "choice without exploration bytes", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.MaxExplorationBytes = 0 })},
		{name: "simulation multiple seeds", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.Seeds = "89-90" })},
		{name: "simulation guidance", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.Guide, spec.Corpus, spec.Coverage = true, "/corpus", CoverageSemantic })},
		{name: "simulation without trace", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.ChoiceTraceLimit = 0 })},
		{name: "simulation without max executions", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.MaxExecutions = 0 }), explore: true},
		{name: "simulation choice depth", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.MaxChoiceDepth = 1 })},
		{name: "simulation choice start ordinal", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.ChoiceStartOrdinal = 1 })},
		{name: "simulation without forced decisions", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.MaxForcedDecisions = 0 }), explore: true},
		{name: "simulation without exploration bytes", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.MaxExplorationBytes = 0 })},
		{name: "simulation without result bytes", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.MaxExplorationResultBytes = 0 }), explore: true},
		{name: "simulation without runtime dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Runtime = 0 })},
		{name: "simulation without scenario dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Scenario = 0 })},
		{name: "simulation without network dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Network = 0 })},
		{name: "simulation without storage dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Storage = 0 })},
		{name: "simulation without fault dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Fault = 0 })},
		{name: "simulation without crash dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits.Crash = 0 }), explore: true},
		{name: "simulation without any dimension", spec: with(simulationSpec, func(spec *CampaignSpec) { spec.SimulationDimensionLimits = SimulationDimensionLimits{} })},
		{name: "unknown strategy", spec: with(seed, func(spec *CampaignSpec) { spec.Strategy = "bogus" }), explore: true},

		// Execution limit rejections.
		{name: "zero parallelism", spec: with(seed, func(spec *CampaignSpec) { spec.Parallel = 0 }), explore: true},
		{name: "zero execution timeout", spec: with(seed, func(spec *CampaignSpec) { spec.ExecutionTimeout = 0 })},
		{name: "zero overall timeout", spec: with(seed, func(spec *CampaignSpec) { spec.OverallTimeout = 0 })},
		{name: "negative termination grace", spec: with(seed, func(spec *CampaignSpec) { spec.TerminateGrace = -1 })},
		{name: "termination grace beyond execution timeout", spec: with(seed, func(spec *CampaignSpec) { spec.TerminateGrace = 4 * time.Second })},
		{name: "zero output limit", spec: with(seed, func(spec *CampaignSpec) { spec.OutputLimit = 0 })},
		{name: "zero World transition limit", spec: with(seed, func(spec *CampaignSpec) { spec.WorldTransitionLimit = 0 })},
		{name: "invalid I/O transcript limit", spec: with(seed, func(spec *CampaignSpec) { spec.IOTranscriptLimit = 1 })},

		// Observation rejections.
		{name: "diagnostics without choice trace", spec: with(seed, func(spec *CampaignSpec) { spec.Diagnostics = true })},
		{name: "diagnostics with choice exploration", spec: with(choiceSpec, func(spec *CampaignSpec) { spec.Diagnostics = true })},
		{name: "choice trace below minimum", spec: with(seed, func(spec *CampaignSpec) { spec.ChoiceTraceLimit = 1 })},
		{name: "choice trace above maximum", spec: with(seed, func(spec *CampaignSpec) { spec.ChoiceTraceLimit = execution.MaximumChoiceTraceBytes + 1 })},
		{name: "probes without coverage", spec: with(seed, func(spec *CampaignSpec) { spec.RequiredSemanticProbes = []string{"probe"} })},
		{name: "unknown required probe", spec: with(seed, func(spec *CampaignSpec) {
			spec.Coverage, spec.RequiredSemanticProbes = CoverageSemantic, []string{"gomad3.characterization.unknown"}
		})},
		{name: "probes with choice coverage", spec: with(seed, func(spec *CampaignSpec) {
			spec.Coverage, spec.ChoiceTraceLimit, spec.RequiredSemanticProbes = CoverageChoice, 1<<20, []string{"probe"}
		})},
		{name: "unknown coverage", spec: with(seed, func(spec *CampaignSpec) { spec.Coverage = "bogus" })},
		{name: "choice coverage without trace", spec: with(seed, func(spec *CampaignSpec) { spec.Coverage = CoverageChoice })},
		{name: "execution evidence with multiple seeds", spec: with(seed, func(spec *CampaignSpec) { spec.Coverage, spec.CollectExecutionEvidence = CoverageSemantic, true })},
		{name: "execution evidence without semantic coverage", spec: with(seed, func(spec *CampaignSpec) { spec.Seeds, spec.CollectExecutionEvidence = "7", true })},
		{name: "invalid clock tick", spec: with(seed, func(spec *CampaignSpec) { spec.ClockTick = "sometimes" })},

		// Retention rejections.
		{name: "disabled retention with count", spec: with(seed, func(spec *CampaignSpec) { spec.SuccessArtifactLimit = 1 })},
		{name: "disabled retention with bytes", spec: with(seed, func(spec *CampaignSpec) { spec.SuccessBytesLimit = 1 })},
		{name: "novel retention without coverage", spec: with(seed, func(spec *CampaignSpec) {
			spec.KeepSuccesses, spec.SuccessArtifactLimit, spec.SuccessBytesLimit = KeepSuccessesNovel, 1, 1
		})},
		{name: "novel retention without bytes", spec: with(seed, func(spec *CampaignSpec) {
			spec.Coverage, spec.KeepSuccesses, spec.SuccessArtifactLimit = CoverageSemantic, KeepSuccessesNovel, 1
		})},
		{name: "all retention without count", spec: with(seed, func(spec *CampaignSpec) { spec.KeepSuccesses, spec.SuccessBytesLimit = KeepSuccessesAll, 1 })},
		{name: "unknown retention", spec: with(seed, func(spec *CampaignSpec) { spec.KeepSuccesses = "bogus" })},
		{name: "empty artifact root", spec: with(seed, func(spec *CampaignSpec) { spec.Artifacts = "" })},
		{name: "empty Runner build", spec: with(seed, func(spec *CampaignSpec) { spec.RunnerBuild = "" })},

		// Guidance rejections.
		{name: "guidance without corpus", spec: with(seed, func(spec *CampaignSpec) { spec.Guide, spec.Coverage = true, CoverageSemantic })},
		{name: "guidance without coverage", spec: with(seed, func(spec *CampaignSpec) { spec.Guide, spec.Corpus = true, "/corpus" })},
		{name: "regression without guidance", spec: with(seed, func(spec *CampaignSpec) { spec.GuideRegression = true })},
		{name: "corpus without guidance", spec: with(seed, func(spec *CampaignSpec) { spec.Corpus = "/corpus" })},
		{name: "snapshot without guidance", spec: with(seed, func(spec *CampaignSpec) { spec.GuideSnapshotSHA256 = planIdentity })},

		// Failure policy rejections.
		{name: "all policy with budget", spec: with(seed, func(spec *CampaignSpec) { spec.FailureBudget = 2 })},
		{name: "first policy with budget", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure, spec.FailureBudget = PolicyFirst, 0 })},
		{name: "zero budget", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure, spec.FailureBudget = PolicyBudget, 0 })},
		{name: "unknown failure policy", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure = "bogus" })},

		// Process wiring and target environment rejections.
		{name: "no supervisor command", spec: with(seed, func(spec *CampaignSpec) { spec.SupervisorCommand = nil })},
		{name: "injected executor without supervisor command", spec: with(seed, func(spec *CampaignSpec) { spec.SupervisorCommand = nil }), dependencies: dependencies{executor: &fakeExecutor{}}},
		{name: "mounts without working directory", spec: with(seed, func(spec *CampaignSpec) { spec.IOROMounts, spec.Target.WorkingDir = []string{"usr=/mnt/usr"}, "" })},
		{name: "malformed mount", spec: with(seed, func(spec *CampaignSpec) { spec.IOROMounts = []string{"usr"} })},
		{name: "missing mount source", spec: with(seed, func(spec *CampaignSpec) { spec.IOROMounts = []string{"gomad3-characterization-missing=/mnt/missing"} })},
		{name: "invalid mount limits", spec: with(seed, func(spec *CampaignSpec) {
			spec.IOROMounts, spec.IOROMountLimits = []string{"usr=/mnt/usr"}, readonlymount.Limits{Files: 1}
		})},
		{name: "malformed environment entry", spec: with(seed, func(spec *CampaignSpec) { spec.Environment = []string{"NOVALUE"} })},
		{name: "reserved environment name", spec: with(seed, func(spec *CampaignSpec) { spec.Environment = []string{"GOMAXPROCS=2"} })},
		{name: "reserved environment prefix", spec: with(seed, func(spec *CampaignSpec) { spec.Environment = []string{"LD_AUDIT=x"} })},
		{name: "duplicate environment name", spec: with(seed, func(spec *CampaignSpec) { spec.Environment = []string{"A=1", "A=2"} })},
		{name: "clock tick as target environment", spec: with(seed, func(spec *CampaignSpec) { spec.Environment = []string{record.ClockTickEnvironment + "=forward"} })},

		// Simultaneous faults pin which check wins.
		{name: "precedence seeds before strategy", spec: with(seed, func(spec *CampaignSpec) { spec.Seeds, spec.Strategy, spec.Parallel = "x", "bogus", 0 })},
		{name: "precedence shard before strategy bounds", spec: with(seed, func(spec *CampaignSpec) { spec.Shard, spec.MaxExecutions = CampaignShard{Index: 0, Count: 2}, 1 })},
		{name: "precedence strategy before limits", spec: with(seed, func(spec *CampaignSpec) { spec.Strategy, spec.Parallel, spec.OutputLimit = "bogus", 0, 0 })},
		{name: "precedence limits before identity", spec: with(seed, func(spec *CampaignSpec) { spec.Parallel, spec.Artifacts, spec.RunnerBuild = 0, "", "" })},
		{name: "precedence coverage before retention", spec: with(seed, func(spec *CampaignSpec) { spec.Coverage, spec.KeepSuccesses = "bogus", "bogus" })},
		{name: "precedence policy before supervisor", spec: with(seed, func(spec *CampaignSpec) { spec.OnFailure, spec.SupervisorCommand = "bogus", nil })},
		{name: "precedence mounts before environment", spec: with(seed, func(spec *CampaignSpec) { spec.IOROMounts, spec.Environment = []string{"usr"}, []string{"NOVALUE"} })},
		{name: "precedence environment before clock tick", spec: with(seed, func(spec *CampaignSpec) { spec.Environment, spec.ClockTick = []string{"NOVALUE"}, "sometimes" })},
		{name: "precedence transcript before clock tick", spec: with(seed, func(spec *CampaignSpec) { spec.IOTranscriptLimit, spec.ClockTick = 1, "sometimes" })},
	}
	names := make(map[string]struct{}, len(cases))
	for _, test := range cases {
		if _, duplicate := names[test.name]; duplicate {
			t.Fatalf("duplicate characterization case %q", test.name)
		}
		names[test.name] = struct{}{}
	}
	return cases
}

// characterizeValidation, characterizeCoordinatorRequest and the exploreWith
// calls, which carry a case's private substitutions, are the only parts of
// this test that name the implementation under characterization.

// characterizationPrepared is a fixed prepared target, so a campaign plan
// record depends only on the campaign request it maps.
func characterizationPrepared() target.Prepared {
	return target.Prepared{
		Path: "/campaigns/v1/run/prepared/target", Kind: target.KindGoRun, Source: "./cmd/target",
		SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 4096, Argv: []string{"target"},
		BuildTags: []string{"tag"}, GoVersion: "go1.27.1", BuildKey: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", TargetGOOS: "linux", TargetGOARCH: "amd64",
	}
}

// characterizationPlan clears the plan members that identify this Runner's
// implementation rather than the campaign request: they legitimately change
// with a rebuilt toolchain or the development platform.
func characterizationPlan(plan campaign.CampaignPlan) ([]byte, error) {
	plan.IOProfile = deterministicio.Contract{}
	plan.ChoiceExplorationImplementationSHA256 = ""
	plan.SimulationExplorationImplementationSHA256 = ""
	if plan.ChoiceProfile != nil {
		profile := *plan.ChoiceProfile
		profile.ImplementationSHA256 = ""
		plan.ChoiceProfile = &profile
	}
	return canonicaljson.CanonicalJSON(plan)
}

func characterizationJournalPlan(spec CampaignSpec, selection SeedSelection) (campaign.ExecutionJournalPlan, error) {
	return campaign.DeriveExecutionJournalPlan(string(normalizedStrategy(spec.Strategy)), selection.Count(), spec.MaxExecutions, uint64(spec.Parallel))
}

func characterizationMounts(spec CampaignSpec) ([]readonlymount.Mapping, readonlymount.Limits, error) {
	mounts, err := readonlymount.ParseMappings(spec.IOROMounts, spec.Target.WorkingDir)
	limits := spec.IOROMountLimits
	if limits == (readonlymount.Limits{}) {
		limits = readonlymount.DefaultLimits()
	}
	return mounts, limits, err
}

func characterizeValidation(spec CampaignSpec, dependency dependencies, frozenGuidance *campaign.GuidancePlan) (SeedSelection, []record.Environment, error) {
	run := newCampaignRun(spec)
	run.dependencies, run.guidancePlan = dependency, frozenGuidance
	return validateConfig(run)
}

func characterizePlanRecord(spec CampaignSpec, frozenGuidance *campaign.GuidancePlan, selection SeedSelection, environment []record.Environment) ([]byte, error) {
	journalPlan, err := characterizationJournalPlan(spec, selection)
	if err != nil {
		return nil, err
	}
	mounts, limits, err := characterizationMounts(spec)
	if err != nil {
		return nil, err
	}
	run := newCampaignRun(spec)
	run.guidancePlan, run.IOROMountLimits = frozenGuidance, limits
	plan, err := campaignPlanRecord(run, journalPlan, "prepared/target", characterizationPrepared(), environment, mounts, selection.Count())
	if err != nil {
		return nil, err
	}
	return characterizationPlan(plan)
}

// characterizeCoordinatorRequest returns the request bytes the isolated
// parent sends and their projection onto the flat field layout the
// coordinator request had when this table was pinned, with the request's
// documented strategy default applied. The request now nests its options by
// group; flattening the groups must give back every pinned member and value.
func characterizeCoordinatorRequest(spec CampaignSpec) (request, legacy json.RawMessage, err error) {
	request, err = json.Marshal(newCoordinatorRequest(newCampaignRun(spec), characterizationChildTimeout))
	if err != nil {
		return nil, nil, err
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(request, &members); err != nil {
		return nil, nil, err
	}
	var groups map[string]map[string]json.RawMessage
	if err := json.Unmarshal(members["Options"], &groups); err != nil {
		return nil, nil, err
	}
	delete(members, "Options")
	for groupName, group := range groups {
		for name, value := range group {
			if _, duplicate := members[name]; duplicate {
				return nil, nil, fmt.Errorf("option %s.%s duplicates a request member", groupName, name)
			}
			members[name] = value
		}
	}
	legacy, err = legacyCoordinatorProjection(members)
	return request, legacy, err
}

func legacyCoordinatorProjection(members map[string]json.RawMessage) (json.RawMessage, error) {
	if string(members["Strategy"]) == `""` {
		members["Strategy"] = json.RawMessage(fmt.Sprintf("%q", StrategySeed))
	}
	return json.Marshal(members)
}
