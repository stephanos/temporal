package runner

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestNewCampaignRunNormalizesTheRequest(t *testing.T) {
	spec := CampaignSpec{
		Environment: []string{"A=1"}, IOROMounts: []string{}, RequiredSemanticProbes: []string{"probe"},
		SupervisorCommand: []string{"supervisor"}, CoordinatorCommand: []string{"coordinator"},
	}
	run := newCampaignRun(spec)
	if run.Strategy != StrategySeed {
		t.Fatalf("empty strategy normalized to %q, want %q", run.Strategy, StrategySeed)
	}
	if run.Shard != (CampaignShard{}) {
		t.Fatalf("zero shard normalized to %#v, want the unsharded zero shard", run.Shard)
	}
	if run.IOROMounts != nil {
		t.Fatalf("empty mount list normalized to %#v, want an absent list", run.IOROMounts)
	}
	spec.Environment[0], spec.RequiredSemanticProbes[0], spec.SupervisorCommand[0], spec.CoordinatorCommand[0] = "B=2", "changed", "changed", "changed"
	want := []string{"A=1", "probe", "supervisor", "coordinator"}
	if got := []string{run.Environment[0], run.RequiredSemanticProbes[0], run.SupervisorCommand[0], run.CoordinatorCommand[0]}; !reflect.DeepEqual(got, want) {
		t.Fatalf("run lists = %q after the request changed, want the run to own %q", got, want)
	}
	for _, strategy := range []Strategy{StrategySeed, StrategyChoiceExploration, StrategySimulationExploration, "unknown"} {
		if got := newCampaignRun(CampaignSpec{Strategy: strategy}).Strategy; got != strategy {
			t.Fatalf("strategy %q normalized to %q", strategy, got)
		}
	}
}

// A request that reaches the coordinator without a defaulted option means
// what the same request means to a local run.
func TestCoordinatorRequestNormalizesDecodedOptions(t *testing.T) {
	decoder := json.NewDecoder(bytes.NewReader([]byte(`{"Options":{"Search":{"Seeds":"7"}},"RunnerBuild":"build"}`)))
	decoder.DisallowUnknownFields()
	var request coordinatorRequest
	if err := decoder.Decode(&request); err != nil {
		t.Fatal(err)
	}
	got := request.campaignRun()
	want := newCampaignRun(CampaignSpec{Seeds: "7", RunnerBuild: "build"})
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("coordinator campaign = %#v, want the local campaign %#v", got, want)
	}
}

// The isolated parent hands its coordinator exactly the options a local run
// would use, except the deadline it leaves to the child.
func TestCoordinatorRequestCarriesTheNormalizedOptions(t *testing.T) {
	run := newCampaignRun(distinctCampaignSpec(t))
	request := newCoordinatorRequest(run, time.Second)
	if run.OverallTimeout == time.Second {
		t.Fatal("distinct campaign already has the child deadline")
	}
	want := run.campaignOptions
	want.OverallTimeout = time.Second
	if !reflect.DeepEqual(request.Options, want) {
		t.Fatalf("coordinator options = %#v, want %#v", request.Options, want)
	}
	if !reflect.DeepEqual(request.SupervisorCommand, run.SupervisorCommand) || request.RunnerBuild != run.RunnerBuild {
		t.Fatalf("coordinator wiring = %q %q, want %q %q", request.SupervisorCommand, request.RunnerBuild, run.SupervisorCommand, run.RunnerBuild)
	}
	if run.OverallTimeout == want.OverallTimeout {
		t.Fatal("building the coordinator request changed the local deadline")
	}
}
