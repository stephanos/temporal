package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

// TestMain lets the test binary serve the Runner's private modes, so an
// isolated campaign started by a test crosses the same process boundaries as
// the gomad executable: coordinator, supervisor and target bootstrap.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "__coordinator", "__target_bootstrap", "__supervisor":
			if err := DispatchPrivateMode(os.Args[1], os.Stdin, os.Stdout); err != nil {
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
	"CoordinatorCommand":   "selects the isolated path in the parent",
	"Progress":             "replaced in the child by a callback that forwards events to the parent",
	"Preparer":             "injected preparation is rejected for isolated campaigns",
	"Executor":             "injected execution is rejected for isolated campaigns",
	"Replayer":             "injected replay is rejected for isolated campaigns",
	"resumePreflight":      "the child opens its own resume preflight",
	"guidancePlan":         "selected locally after target preparation; resume restores it from the recorded plan",
	"failureArtifactLimit": "the child derives it from the campaign plan",
	"failureBytesLimit":    "the child derives it from the campaign plan",
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
			got := wire.campaignRequest()
			want := campaignRequestFromSpec(test.spec)
			want.OverallTimeout = childTimeout
			want.CoordinatorCommand, want.Progress, want.Preparer, want.Executor, want.Replayer = nil, nil, nil, nil, nil
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
	config.Coverage = CoverageSemantic
	config.CollectExecutionEvidence = true
	config.KeepSuccesses = KeepSuccessesAll
	config.SuccessArtifactLimit = 2
	config.SuccessBytesLimit = 64 << 20

	isolated := exploreIsolated(t, config)
	if isolated.Attempted != 1 || isolated.Succeeded != 1 || isolated.RetainedSuccesses != 1 || isolated.ExecutionEvidence == nil {
		t.Fatalf("isolated seed campaign = %#v", isolated)
	}
	wantLimits := ExecutionLimitsEvidence{
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
		{Name: "GOMAD3_IO_PROFILE", Value: deterministicio.Deterministic}, {Name: "GOMADSEED", Value: "7"}, {Name: "MODE", Value: "transport"}, {Name: "TZ", Value: "UTC"},
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
			campaign.CoordinatorCommand = nil
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
			replayed, err := Replay(context.Background(), ReplaySpec{ArtifactPath: path, ToolchainRoot: toolchainRoot(t), SupervisorCommand: campaign.SupervisorCommand})
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
			local.CoordinatorCommand = nil
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
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return CampaignSpec{
		Seeds: "7", Parallel: 1, ExecutionTimeout: 30 * time.Second, OverallTimeout: 5 * time.Minute, TerminateGrace: 2 * time.Second,
		OnFailure: PolicyFirst, FailureBudget: 1, OutputLimit: 1 << 20, WorldTransitionLimit: 1 << 20, Artifacts: t.TempDir(), Target: spec,
		SupervisorCommand: []string{executable, "__supervisor"}, CoordinatorCommand: []string{executable, "__coordinator"},
		RunnerBuild: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
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

// simulationTarget is the root module's Simulation fixture. It is prepared in
// the default closure capability mode from the repository root, the only
// module whose tools/gomad3sim bridges capability review admits.
func simulationTarget(t *testing.T) target.Spec {
	t.Helper()
	workingDir, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return target.Spec{
		Kind: target.KindGoRun, Source: "./tools/gomad3sim/testdata/simulation_exploration", WorkingDir: workingDir,
		BuildTags: []string{"gomad3_toolchain"}, ToolchainRoot: toolchainRoot(t),
	}
}

func conformanceTarget(t *testing.T, source string) target.Spec {
	t.Helper()
	workingDir, err := filepath.Abs(filepath.Join("..", "internal", "gomadtool", "conformance", "testdata"))
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
	config.CoordinatorCommand = nil
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
	encoded, err := canonicaljson.CanonicalJSON(*summary.ExecutionEvidence)
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
