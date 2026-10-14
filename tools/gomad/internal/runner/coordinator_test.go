package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	backend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/native"
	"go.temporal.io/server/tools/gomad/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func TestDecodeCoordinatorMessagesForwardsProgressAndFinalResponse(t *testing.T) {
	payload := []byte("{\"type\":\"progress\",\"progress\":{\"Phase\":\"preparing\",\"campaign_path\":\"/campaign\",\"Selected\":3,\"Attempted\":0,\"Running\":0,\"Succeeded\":0,\"Failures\":0,\"Watchdogs\":0,\"Cancelled\":0,\"DistinctFailures\":0,\"Artifacts\":null}}\n{\"type\":\"result\",\"response\":{\"CampaignResult\":{\"campaign_path\":\"/campaign\",\"SelectionCount\":3,\"Attempted\":3,\"Succeeded\":3,\"Failures\":0,\"Watchdogs\":0,\"Cancelled\":0,\"DistinctFailures\":0,\"StopReason\":\"seeds_exhausted\",\"Artifacts\":null},\"ErrorReason\":\"\",\"ErrorDetail\":\"\"}}\n")
	var progress []CampaignEvent
	response, err := decodeCoordinatorMessages(bytes.NewReader(payload), func(update CampaignEvent) error {
		progress = append(progress, update)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(progress) != 1 || progress[0].Phase != ProgressPreparing || response.CampaignResult.Attempted != 3 {
		t.Fatalf("progress = %#v, response = %#v", progress, response)
	}
}

func TestDecodeCoordinatorMessagesRejectsProgressAfterResult(t *testing.T) {
	payload := []byte("{\"type\":\"result\",\"response\":{\"CampaignResult\":{},\"ErrorReason\":\"\",\"ErrorDetail\":\"\"}}\n{\"type\":\"progress\",\"progress\":{\"Phase\":\"running\"}}\n")
	if _, err := decodeCoordinatorMessages(bytes.NewReader(payload), nil); err == nil {
		t.Fatal("decodeCoordinatorMessages() succeeded")
	}
}

func TestDecodeCoordinatorMessagesPreservesDrainFailure(t *testing.T) {
	drainFailure := errors.New("drain failed")
	input := &sequencedReader{reads: []readerResult{{data: []byte("x")}, {err: drainFailure}}}
	_, err := decodeCoordinatorMessages(input, nil)
	if err == nil || !errors.Is(err, drainFailure) {
		t.Fatalf("decodeCoordinatorMessages() error = %v", err)
	}
}

type cancellationCoordinatorProvider struct {
	*native.Provider
	observed backend.ExecutionResult
}

func (provider *cancellationCoordinatorProvider) Execute(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	result, err := provider.Provider.Execute(ctx, request)
	provider.observed = result
	return result, err
}

func TestCoordinatorOutputFailureCancelsAndReapsBeforeDraining(t *testing.T) {
	for _, test := range []struct {
		mode, reason string
	}{
		{mode: "malformed", reason: "coordinator_decode"},
		{mode: "progress", reason: "progress_output"},
	} {
		t.Run(test.mode, func(t *testing.T) {
			t.Setenv("GOMAD3_COORDINATOR_CANCEL_CASE", test.mode)
			config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
			config.OverallTimeout = 2 * time.Second
			provider := &cancellationCoordinatorProvider{Provider: native.New(native.Config{
				CoordinatorCommand: []string{os.Args[0], "-test.run=^TestCoordinatorCancellationHelper$"},
				RunnerBuild:        config.RunnerBuild,
			})}
			config.Backend = provider
			progressFailure := errors.New("progress failed")
			config.Progress = func(CampaignEvent) error { return progressFailure }
			_, err := runIsolated(t.Context(), campaignRequestFromSpec(config))
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != test.reason || !provider.observed.Cancelled || !provider.observed.GroupGone {
				t.Fatalf("coordinator failure = %v, transport = %#v", err, provider.observed)
			}
			if test.mode == "progress" && !errors.Is(err, progressFailure) {
				t.Fatalf("progress cause was lost: %v", err)
			}
		})
	}
}

func TestCoordinatorCancellationHelper(t *testing.T) {
	mode := os.Getenv("GOMAD3_COORDINATOR_CANCEL_CASE")
	if mode == "" {
		t.Skip("coordinator subprocess only")
	}
	payload := "malformed\n"
	if mode == "progress" {
		payload = "{\"type\":\"progress\",\"progress\":{\"Phase\":\"running\"}}\n"
	}
	if _, err := io.WriteString(os.Stdout, payload); err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	<-timer.C
}

type readerResult struct {
	data []byte
	err  error
}

type sequencedReader struct {
	reads []readerResult
}

func (reader *sequencedReader) Read(output []byte) (int, error) {
	if len(reader.reads) == 0 {
		return 0, io.EOF
	}
	result := reader.reads[0]
	reader.reads = reader.reads[1:]
	return copy(output, result.data), result.err
}

// TestMain lets the test binary serve the Runner's private modes, so an
// isolated campaign started by a test crosses the same process boundaries as
// the gomad executable: coordinator, supervisor and target bootstrap.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "__coordinator", "__target_bootstrap", "__supervisor":
			var err error
			if os.Args[1] == "__coordinator" {
				var payload []byte
				payload, err = io.ReadAll(os.Stdin)
				if err == nil {
					config, shared, decodeErr := native.CoordinatorInput(payload)
					if decodeErr != nil {
						config = native.Config{SupervisorCommand: []string{os.Args[0], "__supervisor"}, RunnerBuild: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
						shared = payload
					}
					err = CoordinatorMain(bytes.NewReader(shared), os.Stdout, native.New(config))
				}
			} else {
				_, err = native.DispatchPrivateMode(os.Args[1], os.Stdin, os.Stdout)
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(3)
			}
			os.Exit(0)
		}
	}
	os.Exit(m.Run())
}

// coordinatorLocalOnlyFields are the CampaignSpec fields that deliberately do
// not cross the coordinator transport, with the reason each stays local.
var coordinatorLocalOnlyFields = map[string]string{
	"Backend":              "each process receives its configured provider",
	"RunnerBuild":          "the provider supplies its local Runner identity",
	"Progress":             "replaced in the child by a callback that forwards events to the parent",
	"Preparer":             "injected preparation is rejected for isolated campaigns",
	"Replayer":             "injected replay is rejected for isolated campaigns",
	"resumePreflight":      "the child opens its own resume preflight",
	"guidancePlan":         "selected locally after target preparation; resume restores it from the recorded plan",
	"failureArtifactLimit": "the child derives it from the campaign plan",
	"failureBytesLimit":    "the child derives it from the campaign plan",
}

func TestExploreRejectsPrivateExecutionForIsolatedCampaign(t *testing.T) {
	_, err := exploreWith(context.Background(), CampaignSpec{Backend: native.New(native.Config{CoordinatorCommand: []string{"unused"}})}, executionDependencies{executor: &fakeExecutor{}})
	if err == nil || !strings.Contains(err.Error(), "isolated Runner does not accept injected preparation or execution") {
		t.Fatalf("isolated execution injection error = %v", err)
	}
}

func TestCoordinatorTransportCoversEveryCampaignSpecField(t *testing.T) {
	specType := reflect.TypeFor[CampaignSpec]()
	wireType := reflect.TypeFor[coordinatorConfig]()
	for index := range specType.NumField() {
		field := specType.Field(index)
		wireField, transported := wireType.FieldByName(field.Name)
		if _, local := coordinatorLocalOnlyFields[field.Name]; local {
			if transported {
				t.Errorf("CampaignSpec.%s is listed as local-only but is transported", field.Name)
			}
			continue
		}
		if !transported {
			t.Errorf("CampaignSpec.%s is neither transported to the coordinator nor listed as local-only", field.Name)
			continue
		}
		if wireField.Type != field.Type {
			t.Errorf("CampaignSpec.%s has type %s but is transported as %s", field.Name, field.Type, wireField.Type)
		}
	}
	for _, field := range reflect.VisibleFields(wireType) {
		if field.PkgPath != "" {
			continue
		}
		if _, found := specType.FieldByName(field.Name); !found {
			t.Errorf("coordinator transport field %s has no CampaignSpec field", field.Name)
		}
	}
	for name := range coordinatorLocalOnlyFields {
		if _, found := specType.FieldByName(name); !found {
			t.Errorf("local-only field %s is not a CampaignSpec field", name)
		}
	}
}

func TestCoordinatorTransportRoundTripsEveryTransportedField(t *testing.T) {
	const childTimeout = 987654321 * time.Nanosecond
	distinct := distinctCampaignSpec(t)
	falseOverride := false
	explicitFalse := distinct
	explicitFalse.GuideRegressionOverride = &falseOverride
	for _, test := range []struct {
		name string
		spec CampaignSpec
	}{
		{name: "distinct nonzero values", spec: distinct},
		{name: "zero values", spec: CampaignSpec{}},
		{name: "explicit false regression override", spec: explicitFalse},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := json.Marshal(campaignRequestFromSpec(test.spec).coordinatorConfig(childTimeout))
			if err != nil {
				t.Fatal(err)
			}
			decoder := json.NewDecoder(bytes.NewReader(encoded))
			decoder.DisallowUnknownFields()
			var wire coordinatorConfig
			if err := decoder.Decode(&wire); err != nil {
				t.Fatal(err)
			}
			got := wire.campaignRequest(test.spec.Backend, test.spec.RunnerBuild)
			want := campaignRequestFromSpec(test.spec)
			want.OverallTimeout = childTimeout
			want.Progress, want.Preparer, want.executor, want.Replayer = nil, nil, nil, nil
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("coordinator transport changed:\n got %#v\nwant %#v", got, want)
			}
		})
	}
}

// distinctCampaignSpec assigns a different nonzero value to every exported
// leaf of a CampaignSpec, so a field added later is exercised by the transport
// round trip without editing a literal.
func distinctCampaignSpec(t *testing.T) CampaignSpec {
	t.Helper()
	var spec CampaignSpec
	next := uint64(100)
	value := reflect.ValueOf(&spec).Elem()
	for index := range value.NumField() {
		field := value.Type().Field(index)
		if _, local := coordinatorLocalOnlyFields[field.Name]; local {
			continue
		}
		fillDistinct(t, field.Name, value.Field(index), &next)
	}
	return spec
}

func fillDistinct(t *testing.T, path string, value reflect.Value, next *uint64) {
	t.Helper()
	*next++
	switch value.Kind() {
	case reflect.String:
		value.SetString(fmt.Sprintf("%s-%d", path, *next))
	case reflect.Int, reflect.Int64:
		value.SetInt(int64(*next))
	case reflect.Uint64:
		value.SetUint(*next)
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
		fillDistinct(t, path+"*", value.Elem(), next)
	case reflect.Slice:
		value.Set(reflect.MakeSlice(value.Type(), 2, 2))
		for index := range value.Len() {
			fillDistinct(t, fmt.Sprintf("%s[%d]", path, index), value.Index(index), next)
		}
	case reflect.Struct:
		for index := range value.NumField() {
			field := value.Type().Field(index)
			if !field.IsExported() {
				t.Fatalf("%s.%s is unexported and cannot cross the coordinator transport", path, field.Name)
			}
			fillDistinct(t, path+"."+field.Name, value.Field(index), next)
		}
	default:
		t.Fatalf("%s has kind %s, which the coordinator transport test cannot populate", path, value.Kind())
	}
}

func differingExportedFields(got, want CampaignSpec) []string {
	gotValue, wantValue := reflect.ValueOf(got), reflect.ValueOf(want)
	var names []string
	for index := range gotValue.NumField() {
		field := gotValue.Type().Field(index)
		if field.IsExported() && !reflect.DeepEqual(gotValue.Field(index).Interface(), wantValue.Field(index).Interface()) {
			names = append(names, field.Name)
		}
	}
	return names
}

func TestIsolatedRunnerExecutesSeedStrategyWithSuppliedLimits(t *testing.T) {
	config := isolatedCampaign(t, conformanceTarget(t, "./choice_exploration"))
	config.ExecutionTimeout = 31 * time.Second
	config.TerminateGrace = 3 * time.Second
	config.OutputLimit = 5 << 20
	config.WorldTransitionLimit = 7 << 20
	config.ChoiceTraceLimit = 9 << 20
	config.IOTranscriptLimit = 65 << 20
	config.ClockTick = record.ClockTickForward
	config.Environment = []string{"MODE=transport"}
	config.Coverage = campaignoptions.CoverageSemantic
	config.CollectExecutionEvidence = true
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20

	isolated := exploreIsolated(t, config)
	if isolated.Attempted != 1 || isolated.Succeeded != 1 || isolated.RetainedSuccesses != 1 || isolated.ExecutionEvidence == nil {
		t.Fatalf("isolated seed campaign = %#v", isolated)
	}
	wantLimits := record.ExecutionLimitsEvidence{
		ExecutionTimeoutNanos: record.Uint64String(31 * time.Second), TerminateGraceNanos: record.Uint64String(3 * time.Second),
		OutputBytes: 5 << 20, WorldTransitionBytes: 7 << 20, IOTranscriptBytes: 65 << 20, ChoiceTraceBytes: 9 << 20,
	}
	if isolated.ExecutionEvidence.Limits != wantLimits {
		t.Fatalf("isolated seed limits = %#v, want %#v", isolated.ExecutionEvidence.Limits, wantLimits)
	}
	if isolated.ChoiceTrace == nil || isolated.ChoiceTrace.Limit != 9<<20 {
		t.Fatalf("isolated seed choice trace = %#v", isolated.ChoiceTrace)
	}
	wantEnvironment := []record.Environment{
		{Name: "GOMAD3_CHOICE_PROFILE", Value: isolated.ChoiceTrace.Profile}, {Name: record.ClockTickEnvironment, Value: record.ClockTickForward},
		{Name: "GOMAD3_IO_PROFILE", Value: transcript.Deterministic}, {Name: "GOMADSEED", Value: "7"}, {Name: "MODE", Value: "transport"}, {Name: "TZ", Value: "UTC"},
	}
	if !reflect.DeepEqual(sortedEnvironment(isolated.ExecutionEvidence.Environment), sortedEnvironment(wantEnvironment)) {
		t.Fatalf("isolated seed environment = %#v, want %#v", isolated.ExecutionEvidence.Environment, wantEnvironment)
	}

	local := exploreLocal(t, config)
	isolatedEvidence, localEvidence := canonicalEvidence(t, isolated), canonicalEvidence(t, local)
	if !bytes.Equal(isolatedEvidence, localEvidence) {
		t.Fatalf("isolated and local seed evidence differ:\nisolated %s\n   local %s", isolatedEvidence, localEvidence)
	}
}

func TestIsolatedRunnerExecutesChoiceStrategyWithSuppliedLimits(t *testing.T) {
	config := isolatedCampaign(t, conformanceTarget(t, "./choice_exploration"))
	config.Strategy = StrategyChoiceExploration
	config.Parallel = 2
	config.ChoiceTraceLimit = 9 << 20
	config.MaxExecutions = 6
	config.MaxChoiceDepth = 5
	config.MaxExplorationBytes = 3 << 20

	isolated := exploreIsolated(t, config)
	if isolated.ChoiceExploration == nil {
		t.Fatalf("isolated choice campaign = %#v", isolated)
	}
	got := *isolated.ChoiceExploration
	if got.Parallel != 2 || got.MaxExecutions != 6 || got.MaxChoiceDepth != 5 || got.MaxExplorationBytes != 3<<20 || got.LogicalExecutions < 2 {
		t.Fatalf("isolated choice exploration = %#v", got)
	}
	if local := exploreLocal(t, config); local.ChoiceExploration == nil || *local.ChoiceExploration != got {
		t.Fatalf("local choice exploration = %#v, isolated %#v", local.ChoiceExploration, got)
	}
}

// The simulation fixture declares one two-way Scenario choice and the campaign
// bounds the runtime dimension to its first decision, so the whole bounded
// space fits inside the execution bound and the campaign ends bounded-complete
// rather than cut off. A second outcome and a replayed "route beta" prove the
// target consumed a Runner-issued plan forcing the Scenario alternative.
func TestIsolatedRunnerCompletesSimulationStrategyWithSuppliedLimits(t *testing.T) {
	config := isolatedSimulationCampaign(t)
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = config.MaxExecutions
	config.SuccessBytesLimit = 256 << 20
	wantBounds := SimulationExplorationSummary{
		Parallel: 2, MaxExecutions: 16, MaxForcedDecisions: 3, MaxExplorationBytes: 2 << 20, MaxResultBytes: 3 << 20, FailureBudget: 4,
		Limits: SimulationDimensionLimits{Runtime: 1, Scenario: 11, Network: 13, Storage: 17, Fault: 19, Crash: 23},
	}
	var running []SimulationExplorationSummary
	config.Progress = func(event CampaignEvent) error {
		if event.Phase == ProgressRunning && event.SimulationExploration != nil {
			running = append(running, *event.SimulationExploration)
		}
		return nil
	}
	results := make(map[string]SimulationExplorationSummary)
	for _, test := range []struct {
		name     string
		isolated bool
	}{{name: "isolated", isolated: true}, {name: "local"}} {
		running = nil
		campaign := config
		campaign.Artifacts = t.TempDir()
		if !test.isolated {
			campaign.Backend = coordinatorTestProvider(campaign, false)
		}
		summary, err := Explore(context.Background(), campaign)
		if err != nil {
			t.Fatalf("%s Explore() error = %v", test.name, err)
		}
		if summary.SimulationExploration == nil || simulationBounds(*summary.SimulationExploration) != wantBounds {
			t.Fatalf("%s simulation exploration = %#v, want bounds %#v", test.name, summary.SimulationExploration, wantBounds)
		}
		if len(running) == 0 || simulationBounds(running[0]) != wantBounds {
			t.Fatalf("%s running progress = %#v, want bounds %#v", test.name, running, wantBounds)
		}
		explored := *summary.SimulationExploration
		if !explored.BoundedComplete || explored.Pending != 0 || explored.LogicalExecutions < 2 || explored.CommittedRounds < 2 || explored.DeduplicatedOutcomes < 2 || explored.DeepestOverride == 0 {
			t.Fatalf("%s simulation exploration = %#v, want a bounded-complete campaign that committed the forced Scenario alternative", test.name, explored)
		}
		if summary.Attempted != explored.LogicalExecutions || summary.Succeeded != summary.Attempted || summary.Failures != 0 || summary.RetainedSuccesses != summary.Attempted {
			t.Fatalf("%s campaign = %#v, want every logical execution committed and retained as a success", test.name, summary)
		}
		routes := make(map[string]struct{})
		for _, path := range summary.SuccessArtifacts {
			replayed, err := Replay(context.Background(), ReplaySpec{Backend: coordinatorTestProvider(campaign, false), ArtifactPath: path, ToolchainRoot: toolchainRoot(t)})
			if err != nil {
				t.Fatalf("%s Replay(%s) error = %v", test.name, path, err)
			}
			if !replayed.Match || replayed.Divergence != "" || replayed.ChoiceReplayStatus != ChoiceReplayExact {
				t.Fatalf("%s Replay(%s) = %#v, want an exact reproduction", test.name, path, replayed)
			}
			stdout, err := os.ReadFile(filepath.Join(path, "stdout"))
			if err != nil {
				t.Fatal(err)
			}
			routes[string(stdout)] = struct{}{}
		}
		if want := map[string]struct{}{"route alpha\n": {}, "route beta\n": {}}; !reflect.DeepEqual(routes, want) {
			t.Fatalf("%s retained routes = %v, want %v", test.name, routes, want)
		}
		results[test.name] = explored
	}
	if results["isolated"] != results["local"] {
		t.Fatalf("isolated simulation exploration = %#v, local %#v", results["isolated"], results["local"])
	}
}

func simulationBounds(summary SimulationExplorationSummary) SimulationExplorationSummary {
	return SimulationExplorationSummary{
		Parallel: summary.Parallel, MaxExecutions: summary.MaxExecutions, MaxForcedDecisions: summary.MaxForcedDecisions,
		MaxExplorationBytes: summary.MaxExplorationBytes, MaxResultBytes: summary.MaxResultBytes, FailureBudget: summary.FailureBudget, Limits: summary.Limits,
	}
}

func TestCoordinatorRejectsInvalidStrategyBoundsLikeTheLocalRunner(t *testing.T) {
	simulation := func(configure func(*CampaignSpec)) CampaignSpec {
		config := isolatedSimulationCampaign(t)
		configure(&config)
		return config
	}
	choice := func(configure func(*CampaignSpec)) CampaignSpec {
		config := simulation(func(config *CampaignSpec) {
			config.Strategy = StrategyChoiceExploration
			config.MaxChoiceDepth = 5
			config.MaxForcedDecisions, config.MaxExplorationResultBytes, config.SimulationDimensionLimits = 0, 0, SimulationDimensionLimits{}
		})
		configure(&config)
		return config
	}
	tests := []struct {
		name   string
		config CampaignSpec
		omit   []string
		want   string
	}{
		{name: "zero forced decisions", config: simulation(func(config *CampaignSpec) { config.MaxForcedDecisions = 0 }), want: "simulation-exploration forced decisions must be positive"},
		{name: "missing forced decisions", config: simulation(func(*CampaignSpec) {}), omit: []string{"MaxForcedDecisions"}, want: "simulation-exploration forced decisions must be positive"},
		{name: "zero result bytes", config: simulation(func(config *CampaignSpec) { config.MaxExplorationResultBytes = 0 }), want: "simulation-exploration result bytes must be positive"},
		{name: "missing result bytes", config: simulation(func(*CampaignSpec) {}), omit: []string{"MaxExplorationResultBytes"}, want: "simulation-exploration result bytes must be positive"},
		{name: "missing dimension bounds", config: simulation(func(*CampaignSpec) {}), omit: []string{"SimulationDimensionLimits"}, want: "simulation-exploration runtime dimension bound must be positive"},
		{name: "zero executions", config: simulation(func(config *CampaignSpec) { config.MaxExecutions = 0 }), want: "simulation-exploration max executions must be positive"},
		{name: "zero exploration bytes", config: simulation(func(config *CampaignSpec) { config.MaxExplorationBytes = 0 }), want: "simulation-exploration exploration bytes must be positive"},
		{name: "choice depth with simulation strategy", config: simulation(func(config *CampaignSpec) { config.MaxChoiceDepth = 1 }), want: "choice depth requires the choice-exploration strategy"},
		{name: "forced decisions with choice strategy", config: choice(func(config *CampaignSpec) { config.MaxForcedDecisions = 3 }), want: "simulation exploration bounds require the simulation-exploration strategy"},
		{name: "result bytes with choice strategy", config: choice(func(config *CampaignSpec) { config.MaxExplorationResultBytes = 3 << 20 }), want: "simulation exploration bounds require the simulation-exploration strategy"},
		{name: "dimension bound with choice strategy", config: choice(func(config *CampaignSpec) { config.SimulationDimensionLimits.Crash = 23 }), want: "simulation exploration bounds require the simulation-exploration strategy"},
		{name: "zero choice depth", config: choice(func(config *CampaignSpec) { config.MaxChoiceDepth = 0 }), want: "choice-exploration choice depth must be positive"},
		{name: "simulation bounds with seed strategy", config: simulation(func(config *CampaignSpec) { config.Strategy = StrategySeed }), want: "exploration bounds require the choice-exploration strategy"},
		{name: "unknown strategy", config: simulation(func(config *CampaignSpec) { config.Strategy = "unknown" }), want: `unknown exploration strategy "unknown"`},
	}
	for _, dimension := range []struct {
		name string
		zero func(*SimulationDimensionLimits)
	}{
		{name: "runtime", zero: func(limits *SimulationDimensionLimits) { limits.Runtime = 0 }},
		{name: "scenario", zero: func(limits *SimulationDimensionLimits) { limits.Scenario = 0 }},
		{name: "network", zero: func(limits *SimulationDimensionLimits) { limits.Network = 0 }},
		{name: "storage", zero: func(limits *SimulationDimensionLimits) { limits.Storage = 0 }},
		{name: "fault", zero: func(limits *SimulationDimensionLimits) { limits.Fault = 0 }},
		{name: "crash", zero: func(limits *SimulationDimensionLimits) { limits.Crash = 0 }},
	} {
		tests = append(tests, struct {
			name   string
			config CampaignSpec
			omit   []string
			want   string
		}{
			name:   "zero " + dimension.name + " dimension",
			config: simulation(func(config *CampaignSpec) { dimension.zero(&config.SimulationDimensionLimits) }),
			want:   "simulation-exploration " + dimension.name + " dimension bound must be positive",
		})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := coordinatorRequest(t, test.config, test.omit...)
			stdout, stderr, err := runCoordinatorProcess(t, request)
			if err != nil {
				t.Fatalf("coordinator process: %v: %s", err, stderr)
			}
			response, err := decodeCoordinatorMessages(bytes.NewReader(stdout), nil)
			if err != nil {
				t.Fatal(err)
			}
			if want := (coordinatorResponse{ErrorReason: "coordinator_run", ErrorDetail: test.want}); !reflect.DeepEqual(response, want) {
				t.Fatalf("coordinator response = %#v, want %#v", response, want)
			}
			if len(test.omit) != 0 {
				return
			}
			local := test.config
			local.Backend = coordinatorTestProvider(local, false)
			for name, config := range map[string]CampaignSpec{"local": local, "isolated": test.config} {
				_, err := Explore(context.Background(), config)
				var hostError *HostError
				if err == nil || err.Error() != test.want || errors.As(err, &hostError) {
					t.Fatalf("%s Explore() error = %v, want %q outside a HostError", name, err, test.want)
				}
			}
		})
	}
}

func TestCoordinatorRejectsMalformedRequests(t *testing.T) {
	valid := coordinatorRequest(t, isolatedSimulationCampaign(t))
	unknownField := append([]byte(`{"Unexpected":1,`), valid[1:]...)
	for _, test := range []struct {
		name    string
		request []byte
		want    string
	}{
		{name: "malformed JSON", request: valid[:len(valid)/2], want: "decode coordinator request: unexpected EOF"},
		{name: "wrong field type", request: []byte(`{"MaxForcedDecisions":"3"}`), want: "decode coordinator request: json: cannot unmarshal string into Go struct field coordinatorConfig.MaxForcedDecisions of type uint64"},
		{name: "unknown field", request: unknownField, want: `decode coordinator request: json: unknown field "Unexpected"`},
		{name: "unknown dimension", request: []byte(`{"SimulationDimensionLimits":{"clock":1}}`), want: `decode coordinator request: json: unknown field "clock"`},
		{name: "trailing data", request: append(append([]byte(nil), valid...), []byte(`{}`)...), want: "trailing coordinator request {"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, err := runCoordinatorProcess(t, test.request)
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 3 || len(stdout) != 0 || !strings.HasPrefix(string(stderr), test.want) {
				t.Fatalf("coordinator process error = %v, stdout = %q, stderr = %q, want exit 3 with %q", err, stdout, stderr, test.want)
			}
		})
	}
}

func isolatedCampaign(t *testing.T, spec target.Spec) CampaignSpec {
	t.Helper()
	config := CampaignSpec{
		Seeds: "7", Parallel: 1, ExecutionTimeout: 30 * time.Second, OverallTimeout: 5 * time.Minute, TerminateGrace: 2 * time.Second,
		OnFailure: PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), Target: spec,
		RunnerBuild: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	config.Backend = coordinatorTestProvider(config, true)
	return config
}

func coordinatorTestProvider(config CampaignSpec, isolated bool) *native.Provider {
	provider := native.Config{ToolchainRoot: config.Target.ToolchainRoot, SupervisorCommand: []string{os.Args[0], "__supervisor"}, RunnerBuild: config.RunnerBuild}
	if isolated {
		provider.CoordinatorCommand = []string{os.Args[0], "__coordinator"}
	}
	return native.New(provider)
}

func isolatedSimulationCampaign(t *testing.T) CampaignSpec {
	t.Helper()
	config := isolatedCampaign(t, simulationTarget(t))
	config.Strategy = StrategySimulationExploration
	config.Seeds = "89"
	config.Parallel = 2
	config.OnFailure = PolicyBudget
	config.FailureBudget = 4
	config.ChoiceTraceLimit = 9 << 20
	config.MaxExecutions = 16
	config.MaxForcedDecisions = 3
	config.MaxExplorationBytes = 2 << 20
	config.MaxExplorationResultBytes = 3 << 20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 1, Scenario: 11, Network: 13, Storage: 17, Fault: 19, Crash: 23}
	return config
}

func TestSimulationCampaignTargetResolvesInGomadModule(t *testing.T) {
	spec := simulationTarget(t)
	command := exec.CommandContext(t.Context(), "go", "list", "-mod=readonly", "-tags", strings.Join(spec.BuildTags, ","), "-f", "{{.Module.Path}}|{{.ImportPath}}", spec.Source)
	command.Dir = spec.WorkingDir
	command.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("resolve simulation campaign target: %v: %s", err, output)
	}
	const expected = "go.temporal.io/server/tools/gomad|go.temporal.io/server/tools/gomad/internal/model/cluster/testdata/simulation_exploration"
	if strings.TrimSpace(string(output)) != expected {
		t.Fatalf("simulation campaign target ownership = %q, want %q", output, expected)
	}
}

func simulationTarget(t *testing.T) target.Spec {
	t.Helper()
	workingDir, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return target.Spec{
		Kind: target.KindGoRun, Source: "./internal/model/cluster/testdata/simulation_exploration", WorkingDir: workingDir,
		BuildTags: []string{"gomad3_toolchain"}, ToolchainRoot: toolchainRoot(t),
	}
}

func conformanceTarget(t *testing.T, source string) target.Spec {
	t.Helper()
	workingDir, err := filepath.Abs(filepath.Join("..", "backends", "native", "conformance", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return target.Spec{Kind: target.KindGoRun, Source: source, WorkingDir: workingDir, ToolchainRoot: toolchainRoot(t)}
}

func exploreIsolated(t *testing.T, config CampaignSpec) CampaignResult {
	t.Helper()
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatalf("isolated Explore() error = %v", err)
	}
	return summary
}

func exploreLocal(t *testing.T, config CampaignSpec) CampaignResult {
	t.Helper()
	config.Backend = coordinatorTestProvider(config, false)
	config.Artifacts = t.TempDir()
	summary, err := Explore(context.Background(), config)
	if err != nil {
		t.Fatalf("local Explore() error = %v", err)
	}
	return summary
}

func canonicalEvidence(t *testing.T, summary CampaignResult) []byte {
	t.Helper()
	if summary.ExecutionEvidence == nil {
		t.Fatalf("campaign = %#v, want execution evidence", summary)
	}
	encoded, err := json.Marshal(*summary.ExecutionEvidence)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func sortedEnvironment(environment []record.Environment) []record.Environment {
	sorted := slices.Clone(environment)
	slices.SortFunc(sorted, func(left, right record.Environment) int { return strings.Compare(left.Name, right.Name) })
	return sorted
}

// coordinatorRequest encodes the request the parent would send for config,
// optionally dropping top-level members to model a request that omits them.
func coordinatorRequest(t *testing.T, config CampaignSpec, omit ...string) []byte {
	t.Helper()
	encoded, err := json.Marshal(campaignRequestFromSpec(config).coordinatorConfig(config.OverallTimeout))
	if err != nil {
		t.Fatal(err)
	}
	if len(omit) == 0 {
		return encoded
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &members); err != nil {
		t.Fatal(err)
	}
	for _, name := range omit {
		if _, found := members[name]; !found {
			t.Fatalf("coordinator request has no member %q", name)
		}
		delete(members, name)
	}
	encoded, err = json.Marshal(members)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func runCoordinatorProcess(t *testing.T, request []byte) (stdout, stderr []byte, err error) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "__coordinator")
	command.Env = append(os.Environ(), "GOMAD3_RUNNER_COORDINATOR=1")
	command.Stdin = bytes.NewReader(request)
	var output, diagnostics bytes.Buffer
	command.Stdout, command.Stderr = &output, &diagnostics
	err = command.Run()
	return output.Bytes(), diagnostics.Bytes(), err
}

func TestIsolatedRunnerKillsAStuckCoordinatorInsideOverallDeadline(t *testing.T) {
	config, configDependencies := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.OverallTimeout = 200 * time.Millisecond
	config.Backend = native.New(native.Config{SupervisorCommand: []string{"unused"}, CoordinatorCommand: []string{os.Args[0], "-test.run=TestBlockingCoordinatorHelper"}, RunnerBuild: config.RunnerBuild})
	started := time.Now()
	_, err := exploreWith(context.Background(), config, configDependencies)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" {
		t.Fatalf("exploreWith() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 350*time.Millisecond {
		t.Fatalf("exploreWith() elapsed = %v", elapsed)
	}
}

func TestIsolatedRunnerPreservesContextReasonAfterCoordinatorExit(t *testing.T) {
	for _, test := range []struct {
		name       string
		cancel     bool
		wantReason string
	}{
		{name: "cancelled", cancel: true, wantReason: "cancelled"},
		{name: "deadline", wantReason: "overall_timeout"},
	} {
		t.Run(test.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "coordinator-exited")
			t.Setenv("GOMAD3_COORDINATOR_EXIT_MARKER", marker)
			config, configDependencies := testConfig(t, nil, nil, "1", PolicyFirst, 1)
			config.OverallTimeout = 250 * time.Millisecond
			config.Backend = native.New(native.Config{SupervisorCommand: []string{"unused"}, CoordinatorCommand: []string{os.Args[0], "-test.run=^TestExitedCoordinatorWithOpenStdoutHelper$"}, RunnerBuild: config.RunnerBuild})
			ctx := context.Background()
			if test.cancel {
				cancelCtx, cancel := context.WithCancel(ctx)
				t.Cleanup(cancel)
				ctx = cancelCtx
				go func() {
					for {
						if _, err := os.Stat(marker); err == nil {
							timer := time.NewTimer(50 * time.Millisecond)
							<-timer.C
							cancel()
							return
						}
						runtime.Gosched()
					}
				}()
			}
			_, err := exploreWith(ctx, config, configDependencies)
			var hostError *HostError
			if !errors.As(err, &hostError) || hostError.Reason != test.wantReason {
				t.Fatalf("exploreWith() error = %v", err)
			}
		})
	}
}

// TestIsolatedRunnerPreservesCoordinatorResponse pins what the isolated parent
// makes of each canned coordinator response: typed preparation and coverage
// errors, bounded evidence, the transported choice trace configuration, and
// the bound on coordinator output.
func TestIsolatedRunnerPreservesCoordinatorResponse(t *testing.T) {
	for _, test := range []struct {
		name      string
		response  string
		policy    FailurePolicy
		configure func(*CampaignSpec)
		check     func(t *testing.T, summary CampaignResult, err error)
	}{
		{
			name: "unsupported target error", response: "unsupported", policy: PolicyFirst,
			check: func(t *testing.T, _ CampaignResult, err error) {
				var unsupported *target.UnsupportedCapabilityError
				if !errors.As(err, &unsupported) || unsupported.ImportPath != "example.com/target" || unsupported.Capability != "imports os/exec" {
					t.Fatalf("exploreWith() error = %v", err)
				}
			},
		},
		{
			name: "missing semantic probes error", response: "missing-probes", policy: PolicyFirst,
			check: func(t *testing.T, _ CampaignResult, err error) {
				var missing *record.MissingSemanticProbesError
				if !errors.As(err, &missing) || len(missing.Probes) != 1 || missing.Probes[0] != "stdlib.os.openfile" {
					t.Fatalf("exploreWith() error = %v", err)
				}
			},
		},
		{
			name: "bounded execution evidence", response: "evidence", policy: PolicyAll,
			configure: func(config *CampaignSpec) {
				config.Coverage = campaignoptions.CoverageSemantic
				config.CollectExecutionEvidence = true
			},
			check: func(t *testing.T, summary CampaignResult, err error) {
				if err != nil {
					t.Fatal(err)
				}
				if summary.ExecutionEvidence == nil || summary.ExecutionEvidence.Schema != record.ExecutionEvidenceSchema || summary.ExecutionEvidence.Seed != 1 || summary.ExecutionEvidence.Target.SHA256 != "sha256:target" {
					t.Fatalf("summary = %#v", summary)
				}
			},
		},
		{
			name: "choice trace configuration", response: "choice-trace", policy: PolicyAll,
			configure: func(config *CampaignSpec) { config.ChoiceTraceLimit = backend.MinimumChoiceTraceBytes },
			check: func(t *testing.T, summary CampaignResult, err error) {
				if err != nil {
					t.Fatal(err)
				}
				if summary.ChoiceTrace == nil || summary.ChoiceTrace.Limit != backend.MinimumChoiceTraceBytes {
					t.Fatalf("summary = %#v", summary)
				}
			},
		},
		{
			name: "bounded coordinator output", response: "oversized", policy: PolicyFirst,
			check: func(t *testing.T, _ CampaignResult, err error) {
				var hostError *HostError
				if !errors.As(err, &hostError) || hostError.Reason != "coordinator_decode" {
					t.Fatalf("exploreWith() error = %v", err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			config, configDependencies := cannedCoordinatorConfig(t, test.response, test.policy)
			if test.configure != nil {
				test.configure(&config)
			}
			summary, err := exploreWith(context.Background(), config, configDependencies)
			test.check(t, summary, err)
		})
	}
}

// cannedCoordinatorConfig runs the isolated parent against
// TestCannedCoordinatorHelper answering with response.
func cannedCoordinatorConfig(t *testing.T, response string, policy FailurePolicy) (CampaignSpec, executionDependencies) {
	t.Helper()
	t.Setenv("GOMAD3_CANNED_COORDINATOR", response)
	config, configDependencies := testConfig(t, nil, nil, "1", policy, 1)
	config.Backend = native.New(native.Config{SupervisorCommand: []string{"unused"}, CoordinatorCommand: []string{os.Args[0], "-test.run=^TestCannedCoordinatorHelper$"}, RunnerBuild: config.RunnerBuild})
	return config, configDependencies
}

func TestIsolatedRunnerDrainsFastCoordinatorBeforeWaitClosesOutput(t *testing.T) {
	config, configDependencies := cannedCoordinatorConfig(t, "fast", PolicyAll)
	config.Progress = func(CampaignEvent) error {
		deadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(deadline) {
			runtime.Gosched()
		}
		return nil
	}
	summary, err := exploreWith(context.Background(), config, configDependencies)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 1 || summary.Succeeded != 1 {
		t.Fatalf("summary = %#v", summary)
	}
}

// TestCannedCoordinatorHelper is a coordinator subprocess that answers with
// the canned response GOMAD3_CANNED_COORDINATOR names.
func TestCannedCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	encoder := json.NewEncoder(os.Stdout)
	var response coordinatorResponse
	switch os.Getenv("GOMAD3_CANNED_COORDINATOR") {
	case "fast":
		progress := CampaignEvent{Phase: ProgressRunning, Attempted: 1, Running: 1}
		if err := encoder.Encode(coordinatorMessage{Type: "progress", Progress: &progress}); err != nil {
			t.Fatal(err)
		}
		response = coordinatorResponse{CampaignResult: CampaignResult{Attempted: 1, Succeeded: 1}}
	case "unsupported":
		unsupported := &target.UnsupportedCapabilityError{ImportPath: "example.com/target", Capability: "imports os/exec"}
		response = coordinatorResponse{
			ErrorReason: "target_preparation", ErrorDetail: unsupported.Error(), UnsupportedTarget: unsupported,
		}
	case "missing-probes":
		missing := &record.MissingSemanticProbesError{Probes: []string{"stdlib.os.openfile"}}
		response = coordinatorResponse{
			ErrorReason: "semantic_coverage", ErrorDetail: missing.Error(), MissingSemanticProbes: missing.Probes,
		}
	case "evidence":
		response = coordinatorResponse{CampaignResult: CampaignResult{ExecutionEvidence: &record.ExecutionEvidence{
			Schema: record.ExecutionEvidenceSchema, Seed: 1, Target: record.Target{SHA256: "sha256:target"},
		}}}
	case "choice-trace":
		var wire coordinatorConfig
		if err := json.NewDecoder(os.Stdin).Decode(&wire); err != nil {
			t.Fatal(err)
		}
		response = coordinatorResponse{CampaignResult: CampaignResult{ChoiceTrace: &record.ChoiceTraceSummary{Limit: wire.ChoiceTraceLimit}}}
	case "oversized":
		if _, err := os.Stdout.Write(make([]byte, maximumCoordinatorMessageBytes+1)); err != nil {
			t.Fatal(err)
		}
		return
	default:
		t.Fatalf("unknown canned coordinator response %q", os.Getenv("GOMAD3_CANNED_COORDINATOR"))
	}
	if err := encoder.Encode(coordinatorMessage{Type: "result", Response: &response}); err != nil {
		t.Fatal(err)
	}
	os.Exit(0) //nolint:revive // This subprocess helper must exit before the parent test harness continues.
}

func TestBlockingCoordinatorHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	for {
		runtime.Gosched()
	}
}

func TestExitedCoordinatorWithOpenStdoutHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	command := exec.Command(os.Args[0], "-test.run=^TestCoordinatorStdoutDescendantHelper$")
	command.Env = append(os.Environ(), "GOMAD3_COORDINATOR_STDOUT_DESCENDANT=1")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(os.Getenv("GOMAD3_COORDINATOR_EXIT_MARKER"), []byte("exiting"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Exit(0) //nolint:revive // This subprocess helper must exit while its descendant retains stdout.
}

func TestCoordinatorStdoutDescendantHelper(t *testing.T) {
	if os.Getenv("GOMAD3_COORDINATOR_STDOUT_DESCENDANT") != "1" {
		t.Skip("coordinator stdout descendant subprocess only")
	}
	<-time.After(10 * time.Second)
}

func TestIsolatedRunnerRemovesCoordinatorProcessGroup(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "descendant-survived")
	t.Setenv("GOMAD3_COORDINATOR_DESCENDANT_MARKER", marker)
	config, configDependencies := testConfig(t, nil, nil, "1", PolicyFirst, 1)
	config.OverallTimeout = 250 * time.Millisecond
	config.Backend = native.New(native.Config{SupervisorCommand: []string{"unused"}, CoordinatorCommand: []string{os.Args[0], "-test.run=TestCoordinatorTreeHelper"}, RunnerBuild: config.RunnerBuild})
	_, err := exploreWith(context.Background(), config, configDependencies)
	var hostError *HostError
	if !errors.As(err, &hostError) || hostError.Reason != "overall_timeout" {
		t.Fatalf("exploreWith() error = %v", err)
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, statErr := os.Stat(marker); statErr == nil {
			t.Fatal("coordinator descendant survived cleanup")
		} else if !os.IsNotExist(statErr) {
			t.Fatal(statErr)
		}
		runtime.Gosched()
	}
}

func TestCoordinatorTreeHelper(t *testing.T) {
	if os.Getenv("GOMAD3_RUNNER_COORDINATOR") != "1" {
		t.Skip("coordinator subprocess only")
	}
	command := exec.Command(os.Args[0], "-test.run=TestCoordinatorDescendantHelper")
	command.Env = append(os.Environ(), "GOMAD3_COORDINATOR_DESCENDANT=1")
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	for {
		runtime.Gosched()
	}
}

func TestCoordinatorDescendantHelper(t *testing.T) {
	if os.Getenv("GOMAD3_COORDINATOR_DESCENDANT") != "1" {
		t.Skip("coordinator descendant subprocess only")
	}
	signal.Ignore(syscall.SIGTERM)
	<-time.After(350 * time.Millisecond)
	if err := os.WriteFile(os.Getenv("GOMAD3_COORDINATOR_DESCENDANT_MARKER"), []byte("survived"), 0o600); err != nil {
		t.Fatal(err)
	}
}
