package runner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func TestCampaignRequestNormalizesTheRequest(t *testing.T) {
	provider := scriptProvider()
	spec := CampaignSpec{
		Environment: []string{"A=1"}, IOROMounts: []string{}, RequiredSemanticProbes: []string{"probe"},
		Backend: provider,
	}
	run := campaignRequestFromSpec(spec)
	if run.strategy() != StrategySeed {
		t.Fatalf("empty strategy normalized to %q, want %q", run.strategy(), StrategySeed)
	}
	if run.shard() != (CampaignShard{Index: 0, Count: 1}) {
		t.Fatalf("zero shard normalized to %#v, want the unsharded one-shard selection", run.shard())
	}
	if run.IOROMounts != nil {
		t.Fatalf("empty mount list normalized to %#v, want an absent list", run.IOROMounts)
	}
	spec.Environment[0], spec.RequiredSemanticProbes[0] = "B=2", "changed"
	want := []string{"A=1", "probe"}
	if got := []string{run.Environment[0], run.RequiredSemanticProbes[0]}; !reflect.DeepEqual(got, want) {
		t.Fatalf("run lists = %q after the request changed, want the run to own %q", got, want)
	}
	if run.Backend != provider {
		t.Fatal("campaign request changed its provider")
	}
	for _, strategy := range []Strategy{StrategySeed, StrategyChoiceExploration, StrategySimulationExploration, "unknown"} {
		if got := campaignRequestFromSpec(CampaignSpec{Strategy: strategy}).strategy(); got != strategy {
			t.Fatalf("strategy %q normalized to %q", strategy, got)
		}
	}
}

// A request that reaches the coordinator without a defaulted option means
// what the same request means to a local run.
func TestCoordinatorConfigNormalizesDecodedOptions(t *testing.T) {
	decoder := json.NewDecoder(bytes.NewReader([]byte(`{"Seeds":"7"}`)))
	decoder.DisallowUnknownFields()
	var request coordinatorConfig
	if err := decoder.Decode(&request); err != nil {
		t.Fatal(err)
	}
	provider := scriptProvider()
	got := request.campaignRequest(provider, "build")
	want := campaignRequestFromSpec(CampaignSpec{Seeds: "7", Backend: provider, RunnerBuild: "build"})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("coordinator campaign = %#v, want the local campaign %#v", got, want)
	}
}

// The isolated parent hands its coordinator exactly the options a local run
// would use, except the deadline it leaves to the child.
func TestCoordinatorConfigCarriesTheNormalizedOptions(t *testing.T) {
	run := campaignRequestFromSpec(distinctCampaignSpec(t))
	request := run.coordinatorConfig(time.Second)
	if run.OverallTimeout == time.Second {
		t.Fatal("distinct campaign already has the child deadline")
	}
	want := run.campaignOptions
	want.OverallTimeout = time.Second
	if !reflect.DeepEqual(request.campaignOptions, want) {
		t.Fatalf("coordinator options = %#v, want %#v", request.campaignOptions, want)
	}
	transported := request.campaignRequest(run.Backend, run.RunnerBuild)
	if transported.Backend != run.Backend || transported.RunnerBuild != run.RunnerBuild {
		t.Fatalf("coordinator runtime = %#v, want %#v", transported.campaignRuntime, run.campaignRuntime)
	}
	if run.OverallTimeout == want.OverallTimeout {
		t.Fatal("building the coordinator request changed the local deadline")
	}
}

func TestParseStrategyAndCoverageModeAreTheOneReading(t *testing.T) {
	for value, want := range map[string]Strategy{"": StrategySeed, "seed": StrategySeed, "choice-exploration": StrategyChoiceExploration, "simulation-exploration": StrategySimulationExploration} {
		if got, err := ParseStrategy(value); err != nil || got != want {
			t.Fatalf("ParseStrategy(%q) = %q, %v, want %q", value, got, err, want)
		}
	}
	if _, err := ParseStrategy("random"); err == nil || err.Error() != `unknown exploration strategy "random"` {
		t.Fatalf("ParseStrategy(random) error = %v", err)
	}
	for _, want := range []campaignoptions.CoverageMode{campaignoptions.CoverageNone, campaignoptions.CoverageSemantic, campaignoptions.CoverageChoice, campaignoptions.CoverageSemanticChoice} {
		if got, err := ParseCoverageMode(string(want)); err != nil || got != want {
			t.Fatalf("ParseCoverageMode(%q) = %q, %v", want, got, err)
		}
	}
	// Only an omitted CampaignSpec.Coverage means none; a spelled empty name is not a mode.
	for _, value := range []string{"", "all"} {
		if _, err := ParseCoverageMode(value); err == nil || err.Error() != fmt.Sprintf("unknown coverage mode %q", value) {
			t.Fatalf("ParseCoverageMode(%q) error = %v", value, err)
		}
	}
}

func TestParseSingleBaseSeedSharesRunnerStrategyRule(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "7", want: "7"},
		{input: "7-7", want: "7"},
		{input: "7,8"},
		{input: "bad"},
	} {
		selection, err := ParseSingleBaseSeed(test.input)
		if test.want == "" {
			if err == nil {
				t.Fatalf("ParseSingleBaseSeed(%q) = %q, want error", test.input, selection.String())
			}
			continue
		}
		if err != nil || selection.String() != test.want {
			t.Fatalf("ParseSingleBaseSeed(%q) = %q, %v; want %q", test.input, selection.String(), err, test.want)
		}
	}
}

func TestCampaignOptionErrorsPreserveKindThroughWrapping(t *testing.T) {
	_, singleErr := ParseSingleBaseSeed("7,8")
	var single *NotSingleBaseSeedError
	if !errors.As(fmt.Errorf("wrapped: %w", singleErr), &single) || singleErr.Error() != "requires exactly one base seed" {
		t.Fatalf("single-base-seed error = %T %v", singleErr, singleErr)
	}
	coverageErr := campaignoptions.ValidateCoverage(campaignoptions.CoverageNone, []string{"stdlib.os.openfile"})
	var required *campaignoptions.SemanticCoverageRequiredError
	if !errors.As(fmt.Errorf("wrapped: %w", coverageErr), &required) || coverageErr.Error() != "required semantic probes require semantic coverage" {
		t.Fatalf("semantic-coverage error = %T %v", coverageErr, coverageErr)
	}
}

func TestCampaignOptionNormalizationAndValidation(t *testing.T) {
	if got := normalizeStrategy(""); got != StrategySeed {
		t.Fatalf("normalizeStrategy(empty) = %q", got)
	}
	if got := campaignoptions.NormalizeCoverage("", true); got != campaignoptions.CoverageSemantic {
		t.Fatalf("campaignoptions.NormalizeCoverage(empty, guided) = %q", got)
	}
	if got := campaignoptions.NormalizeCoverage("", false); got != campaignoptions.CoverageNone {
		t.Fatalf("campaignoptions.NormalizeCoverage(empty, plain) = %q", got)
	}
	if got := campaignRequestFromSpec(CampaignSpec{Guide: true}).Coverage; got != "" {
		t.Fatalf("Runner changed absent guided coverage to %q", got)
	}
	for _, test := range []struct {
		mode     campaignoptions.CoverageMode
		required []string
		valid    bool
	}{
		{mode: campaignoptions.CoverageNone, valid: true},
		{mode: campaignoptions.CoverageSemantic, required: []string{"stdlib.os.openfile"}, valid: true},
		{mode: campaignoptions.CoverageNone, required: []string{"stdlib.os.openfile"}},
		{mode: campaignoptions.CoverageSemantic, required: []string{"unknown.probe"}},
		{mode: "unknown"},
	} {
		err := campaignoptions.ValidateCoverage(test.mode, test.required)
		if err == nil && len(test.required) != 0 {
			_, err = scriptProvider().Describe(t.Context(), backends.Configuration{RequiredSemanticProbes: test.required})
		}
		if (err == nil) != test.valid {
			t.Fatalf("campaignoptions.ValidateCoverage(%q, %v) = %v; valid=%t", test.mode, test.required, err, test.valid)
		}
	}
	for _, test := range []struct {
		limit uint64
		valid bool
	}{
		{limit: 0, valid: true},
		{limit: backends.MinimumChoiceTraceBytes, valid: true},
		{limit: backends.MaximumChoiceTraceBytes, valid: true},
		{limit: 1},
		{limit: backends.MaximumChoiceTraceBytes + 1},
	} {
		if err := campaignoptions.ValidateChoiceTraceLimit(test.limit); (err == nil) != test.valid {
			t.Fatalf("campaignoptions.ValidateChoiceTraceLimit(%d) = %v; valid=%t", test.limit, err, test.valid)
		}
	}
	if err := campaignoptions.ValidateChoiceCoverage(campaignoptions.CoverageChoice, 0); err == nil {
		t.Fatal("choice coverage without a trace succeeded")
	}
	if err := campaignoptions.ValidateChoiceCoverage(campaignoptions.CoverageChoice, backends.MinimumChoiceTraceBytes); err != nil {
		t.Fatal(err)
	}
	if err := campaignoptions.ValidateChoiceCoverage(campaignoptions.CoverageSemantic, 0); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfigRecordsOnlyTheForwardClockTick(t *testing.T) {
	for _, test := range []struct {
		clockTick string
		want      bool
	}{
		{clockTick: "", want: false},
		{clockTick: record.ClockTickStrict, want: false},
		{clockTick: record.ClockTickForward, want: true},
	} {
		config, _ := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
		config.ClockTick = test.clockTick
		_, environment, err := validateCampaignRequest(campaignRequestFromSpec(config))
		if err != nil {
			t.Fatalf("validateConfig(ClockTick=%q) = %v", test.clockTick, err)
		}
		recorded := slices.Contains(environment, record.Environment{Name: record.ClockTickEnvironment, Value: record.ClockTickForward})
		if recorded != test.want {
			t.Fatalf("validateConfig(ClockTick=%q) environment = %v", test.clockTick, environment)
		}
		if !slices.IsSortedFunc(environment, func(a, b record.Environment) int { return strings.Compare(a.Name, b.Name) }) {
			t.Fatalf("environment is not sorted: %v", environment)
		}
	}

	config, _ := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
	config.ClockTick = "sometimes"
	if _, _, err := validateCampaignRequest(campaignRequestFromSpec(config)); err == nil || !strings.Contains(err.Error(), "must be strict or forward") {
		t.Fatalf("validateConfig(ClockTick=sometimes) error = %v", err)
	}

	config, _ = testConfig(t, newFakePreparer(t), &fakeExecutor{}, "7", PolicyAll, 1)
	config.Environment = append(config.Environment, record.ClockTickEnvironment+"=forward")
	if _, _, err := validateCampaignRequest(campaignRequestFromSpec(config)); err == nil {
		t.Fatal("validateConfig() accepted the clock tick policy as a target --env entry")
	}
}
