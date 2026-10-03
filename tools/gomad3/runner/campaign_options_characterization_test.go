package runner

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio/readonlymount"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
)

type campaignOptionsObservation struct {
	Name        string               `json:"name"`
	Selection   string               `json:"selection"`
	Count       uint64               `json:"count"`
	Environment []record.Environment `json:"environment"`
	Error       string               `json:"error"`
	Request     string               `json:"request"`
}

type campaignOptionsRequestBytes struct {
	Name    string `json:"name"`
	Request string `json:"request"`
}

func campaignOptionsCases() []struct {
	name   string
	config CampaignSpec
} {
	base := CampaignSpec{
		Seeds: "7", Parallel: 1, ExecutionTimeout: 30 * time.Second, OverallTimeout: time.Minute,
		TerminateGrace: time.Second, OnFailure: PolicyAll, FailureBudget: 1,
		OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20,
		Artifacts: "/tmp/gomad-campaign-options", Environment: []string{"MODE=baseline"},
		SupervisorCommand: []string{"supervisor"},
		RunnerBuild:       "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	choice := base
	choice.Strategy = StrategyChoiceExploration
	choice.ChoiceTraceLimit = 1 << 20
	choice.MaxExecutions, choice.MaxChoiceDepth, choice.MaxExplorationBytes = 8, 4, 1<<20
	simulation := base
	simulation.Strategy = StrategySimulationExploration
	simulation.ChoiceTraceLimit = 1 << 20
	simulation.MaxExecutions, simulation.MaxForcedDecisions = 8, 4
	simulation.MaxExplorationBytes, simulation.MaxExplorationResultBytes = 1<<20, 2<<20
	simulation.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 1, Scenario: 2, Network: 3, Storage: 4, Fault: 5, Crash: 6}
	guided := base
	guided.Seeds = "7-10"
	guided.Guide, guided.GuideRegression = true, true
	guided.Corpus, guided.Coverage = "/tmp/gomad-corpus", CoverageSemantic
	sharded := base
	sharded.Shard = CampaignShard{Index: 1, Count: 2}
	sharded.PlanSHA256 = record.SHA256("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	resume := base
	resume.ResumeCampaign = "/tmp/gomad-resume"
	cases := []struct {
		name   string
		config CampaignSpec
	}{
		{"seed-default", base}, {"choice", choice}, {"simulation", simulation}, {"guided", guided}, {"sharded", sharded}, {"resume", resume},
	}
	add := func(name string, source CampaignSpec, change func(*CampaignSpec)) {
		change(&source)
		cases = append(cases, struct {
			name   string
			config CampaignSpec
		}{name, source})
	}
	add("resume-preparer", resume, func(c *CampaignSpec) { c.Preparer = targetPreparer{} })
	add("resume-build", resume, func(c *CampaignSpec) { c.RunnerBuild = "" })
	add("resume-command", resume, func(c *CampaignSpec) { c.SupervisorCommand = nil })
	add("seeds", base, func(c *CampaignSpec) { c.Seeds = "" })
	add("shard-index", sharded, func(c *CampaignSpec) { c.Shard.Index = 2 })
	add("shard-identity-missing", sharded, func(c *CampaignSpec) { c.PlanSHA256 = "" })
	add("shard-identity-invalid", sharded, func(c *CampaignSpec) { c.PlanSHA256 = "bad" })
	add("shard-strategy", sharded, func(c *CampaignSpec) { c.Strategy = StrategyChoiceExploration })
	add("shard-guidance", sharded, func(c *CampaignSpec) { c.Guide = true })
	add("shard-policy", sharded, func(c *CampaignSpec) { c.OnFailure = PolicyFirst })
	add("seed-bounds", base, func(c *CampaignSpec) { c.MaxExecutions = 1 })
	add("choice-simulation-bounds", choice, func(c *CampaignSpec) { c.MaxForcedDecisions = 1 })
	add("choice-seeds", choice, func(c *CampaignSpec) { c.Seeds = "7-8" })
	add("choice-guide", choice, func(c *CampaignSpec) { c.Guide = true })
	add("choice-trace", choice, func(c *CampaignSpec) { c.ChoiceTraceLimit = 0 })
	add("choice-executions", choice, func(c *CampaignSpec) { c.MaxExecutions = 0 })
	add("choice-depth", choice, func(c *CampaignSpec) { c.MaxChoiceDepth = 0 })
	add("choice-bytes", choice, func(c *CampaignSpec) { c.MaxExplorationBytes = 0 })
	add("simulation-seeds", simulation, func(c *CampaignSpec) { c.Seeds = "7-8" })
	add("simulation-guide", simulation, func(c *CampaignSpec) { c.Guide = true })
	add("simulation-trace", simulation, func(c *CampaignSpec) { c.ChoiceTraceLimit = 0 })
	add("simulation-executions", simulation, func(c *CampaignSpec) { c.MaxExecutions = 0 })
	add("simulation-choice-depth", simulation, func(c *CampaignSpec) { c.MaxChoiceDepth = 1 })
	add("simulation-start-ordinal", simulation, func(c *CampaignSpec) { c.ChoiceStartOrdinal = 1 })
	add("simulation-forced", simulation, func(c *CampaignSpec) { c.MaxForcedDecisions = 0 })
	add("simulation-bytes", simulation, func(c *CampaignSpec) { c.MaxExplorationBytes = 0 })
	add("simulation-result-bytes", simulation, func(c *CampaignSpec) { c.MaxExplorationResultBytes = 0 })
	for _, dimension := range []struct {
		name string
		zero func(*SimulationDimensionLimits)
	}{
		{"runtime", func(l *SimulationDimensionLimits) { l.Runtime = 0 }},
		{"scenario", func(l *SimulationDimensionLimits) { l.Scenario = 0 }},
		{"network", func(l *SimulationDimensionLimits) { l.Network = 0 }},
		{"storage", func(l *SimulationDimensionLimits) { l.Storage = 0 }},
		{"fault", func(l *SimulationDimensionLimits) { l.Fault = 0 }},
		{"crash", func(l *SimulationDimensionLimits) { l.Crash = 0 }},
	} {
		add("simulation-dimension-"+dimension.name, simulation, func(c *CampaignSpec) { dimension.zero(&c.SimulationDimensionLimits) })
	}
	add("strategy-unknown", base, func(c *CampaignSpec) { c.Strategy = "unknown" })
	add("parallel", base, func(c *CampaignSpec) { c.Parallel = 0 })
	add("timeout", base, func(c *CampaignSpec) { c.ExecutionTimeout = 0 })
	add("grace", base, func(c *CampaignSpec) { c.TerminateGrace = time.Minute })
	add("output", base, func(c *CampaignSpec) { c.OutputLimit = 0 })
	add("diagnostics", base, func(c *CampaignSpec) { c.Diagnostics = true })
	add("choice-capacity", base, func(c *CampaignSpec) { c.ChoiceTraceLimit = 1 })
	add("artifacts", base, func(c *CampaignSpec) { c.Artifacts = "" })
	add("coverage-probes", base, func(c *CampaignSpec) { c.RequiredSemanticProbes = []string{"probe"} })
	add("coverage-unknown-probe", base, func(c *CampaignSpec) { c.Coverage = CoverageSemantic; c.RequiredSemanticProbes = []string{"probe"} })
	add("coverage-choice-probes", base, func(c *CampaignSpec) { c.Coverage = CoverageChoice; c.RequiredSemanticProbes = []string{"probe"} })
	add("coverage-choice-trace", base, func(c *CampaignSpec) { c.Coverage = CoverageChoice })
	add("coverage-unknown", base, func(c *CampaignSpec) { c.Coverage = "unknown" })
	add("retention-disabled-limit", base, func(c *CampaignSpec) { c.SuccessArtifactLimit = 1 })
	add("retention-novel", base, func(c *CampaignSpec) { c.KeepSuccesses = KeepSuccessesNovel })
	add("retention-all", base, func(c *CampaignSpec) { c.KeepSuccesses = KeepSuccessesAll })
	add("retention-unknown", base, func(c *CampaignSpec) { c.KeepSuccesses = "unknown" })
	add("execution-evidence", base, func(c *CampaignSpec) { c.CollectExecutionEvidence = true })
	add("guide-corpus", base, func(c *CampaignSpec) { c.Guide = true })
	add("unguided-corpus", base, func(c *CampaignSpec) { c.Corpus = "/tmp/corpus" })
	add("failure-budget-first", base, func(c *CampaignSpec) { c.FailureBudget = 2 })
	add("failure-budget-zero", base, func(c *CampaignSpec) { c.OnFailure = PolicyBudget; c.FailureBudget = 0 })
	add("failure-policy-unknown", base, func(c *CampaignSpec) { c.OnFailure = "unknown" })
	add("supervisor", base, func(c *CampaignSpec) { c.SupervisorCommand = nil })
	add("mount-working-dir", base, func(c *CampaignSpec) { c.IOROMounts = []string{"/tmp=data"} })
	add("mount-invalid", base, func(c *CampaignSpec) { c.Target.WorkingDir = "/"; c.IOROMounts = []string{"invalid"} })
	add("mount-limits", base, func(c *CampaignSpec) {
		c.Target.WorkingDir = "/"
		c.IOROMounts = []string{"/tmp=data"}
		c.IOROMountLimits = readonlymount.Limits{PathBytes: 1}
	})
	add("environment-invalid", base, func(c *CampaignSpec) { c.Environment = []string{"bad"} })
	add("environment-reserved", base, func(c *CampaignSpec) { c.Environment = []string{"GOMADSEED=1"} })
	add("environment-duplicate", base, func(c *CampaignSpec) { c.Environment = []string{"MODE=one", "MODE=two"} })
	add("transcript", base, func(c *CampaignSpec) { c.IOTranscriptLimit = 1 })
	add("clock", base, func(c *CampaignSpec) { c.ClockTick = "unknown" })
	add("seeds-before-parallel", base, func(c *CampaignSpec) { c.Seeds = ""; c.Parallel = 0 })
	add("simulation-forced-before-result", simulation, func(c *CampaignSpec) { c.MaxForcedDecisions = 0; c.MaxExplorationResultBytes = 0 })
	add("retention-before-environment", base, func(c *CampaignSpec) { c.SuccessArtifactLimit = 1; c.Environment = []string{"bad"} })
	add("resume-build-before-command", resume, func(c *CampaignSpec) { c.RunnerBuild = ""; c.SupervisorCommand = nil })
	add("guided-frozen-selection", guided, func(c *CampaignSpec) {
		c.guidancePlan = &campaign.GuidancePlan{RequestedCount: 4, AnsweredCount: 1}
		c.Seeds = "7-9"
	})
	return cases
}

func TestCampaignOptionsCharacterization(t *testing.T) {
	var observed []campaignOptionsObservation
	cases := campaignOptionsCases()
	for _, test := range cases {
		selection, environment, err := validateConfig(test.config)
		request, marshalErr := json.Marshal(campaignRequestFromSpec(test.config).coordinatorConfig(test.config.OverallTimeout))
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		row := campaignOptionsObservation{Name: test.name, Selection: selection.String(), Count: selection.Count(), Environment: environment, Request: string(request)}
		if err != nil {
			row.Error = err.Error()
		}
		observed = append(observed, row)
	}
	const fixture = "testdata/campaign-options-before.json"
	if os.Getenv("GOMAD3_CAPTURE_OPTIONS_BASELINE") == "1" {
		data, err := json.MarshalIndent(observed, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fixture, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var baseline []campaignOptionsObservation
	if err := json.Unmarshal(data, &baseline); err != nil {
		t.Fatal(err)
	}
	if len(observed) != len(baseline) {
		t.Fatalf("characterization rows = %d, want %d", len(observed), len(baseline))
	}
	var requestBytes []campaignOptionsRequestBytes
	for index, got := range observed {
		want := baseline[index]
		if want.Name == "mount-limits" {
			if source, err := os.Lstat("/tmp"); err == nil && source.IsDir() {
				want.Error = "read-only mount limits must be positive"
			}
		}
		gotRequest, wantRequest := got.Request, want.Request
		requestBytes = append(requestBytes, campaignOptionsRequestBytes{Name: got.Name, Request: gotRequest})
		got.Request, want.Request = "", ""
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s selection/environment/error = %#v, want %#v", want.Name, got, want)
		}
		var gotFields, wantFields map[string]json.RawMessage
		if err := json.Unmarshal([]byte(gotRequest), &gotFields); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(wantRequest), &wantFields); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotFields, wantFields) {
			t.Errorf("%s coordinator fields changed: got %s, want %s", want.Name, gotRequest, wantRequest)
		}
		decoder := json.NewDecoder(bytes.NewReader([]byte(gotRequest)))
		decoder.DisallowUnknownFields()
		var decoded coordinatorConfig
		if err := decoder.Decode(&decoded); err != nil {
			t.Fatal(err)
		}
		original := campaignRequestFromSpec(cases[index].config)
		transported := decoded.campaignRequest()
		if !reflect.DeepEqual(transported.campaignOptions, original.campaignOptions) || !reflect.DeepEqual(transported.SupervisorCommand, original.SupervisorCommand) || transported.RunnerBuild != original.RunnerBuild {
			t.Errorf("%s coordinator decode changed the campaign request", want.Name)
		}
	}
	const afterFixture = "testdata/campaign-options-request-after.json"
	if os.Getenv("GOMAD3_CAPTURE_OPTIONS_AFTER") == "1" {
		data, err := json.MarshalIndent(requestBytes, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(afterFixture, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	data, err = os.ReadFile(afterFixture)
	if err != nil {
		t.Fatal(err)
	}
	var expectedRequests []campaignOptionsRequestBytes
	if err := json.Unmarshal(data, &expectedRequests); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(requestBytes, expectedRequests) {
		t.Fatal("coordinator request bytes changed")
	}
}
