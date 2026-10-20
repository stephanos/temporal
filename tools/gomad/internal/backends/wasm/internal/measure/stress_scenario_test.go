package measure

import (
	"encoding/json"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func TestStressClusterScenarioPreservesFourIncarnationsAndGroupCrash(t *testing.T) {
	t.Parallel()
	config := standaloneConfig()
	config.Profile = wasi.CooperativeProfile
	config.WritableDirectories = append(config.WritableDirectories, "/volume")
	config.Clock.ReadStepNanos = 0
	config.NetworkV2 = stressClusterNetwork()
	storage := &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: "temporal", Path: "/volume", CapacityBytes: 1 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1000, Transitions: 10000}}
	scenario, err := stressClusterScenario(storage)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenario.Spec.Nodes) != 2 || len(scenario.Boots) != 4 || len(scenario.Actions) != 8 || len(scenario.IncarnationBoots) != 2 {
		t.Fatal("incomplete two-cluster lifecycle")
	}
	base := wasi.Request{Config: config, Runtime: &wasi.RuntimeControl{Choice: &backends.ChoiceRequest{}}}
	spec, provider, err := scenario.Bind(base)
	if err != nil {
		t.Fatal(err)
	}
	spec, err = stressBindCluster(base, spec, provider)
	if err != nil {
		t.Fatal(err)
	}
	if len(provider.OutputCrashes) != 0 || len(provider.OutputCrashGroups) != 1 || len(provider.OutputCrashGroups[0].Handles) != 2 {
		t.Fatal("crashes are not synchronized after both convergence markers")
	}
	for id, request := range provider.Boots {
		name, phase, _ := strings.Cut(string(id), "-")
		found := false
		for _, entry := range request.Config.Environment {
			found = found || entry == "GOMAD_TEMPORAL_PHASE="+phase
		}
		if !found {
			t.Fatalf("boot %s lacks declared phase", id)
		}
		if request.Config.Storage.Volumes[0].ID != name {
			t.Fatalf("boot %s shares another cluster's durable state", id)
		}
	}
	if len(base.Config.Environment) != len(config.Environment) || storage.Volumes[0].ID != "temporal" {
		t.Fatal("binding mutated its template")
	}
	for _, node := range spec.Nodes {
		var declared map[string]any
		if err := json.Unmarshal(node.Config, &declared); err != nil || len(declared) == 0 {
			t.Fatalf("missing authenticated boot config: %v", err)
		}
	}
}
