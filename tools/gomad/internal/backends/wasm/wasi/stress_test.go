package wasi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestStressHostConditionsStayOutsideGuestInputs(t *testing.T) {
	t.Parallel()
	request := Request{Engine: PinnedEngine(), Config: Config{Environment: []string{"DECLARED=unchanged"}}}
	before, err := json.Marshal(request.Config)
	if err != nil {
		t.Fatal(err)
	}
	request.SetStressHostConditions(strings.Repeat("x", 4096), true)
	after, err := json.Marshal(request.Config)
	if err != nil || string(before) != string(after) {
		t.Fatalf("host perturbation changed declared inputs: %v", err)
	}
	if len(request.stressHostEnvironment) != 3 || request.DiagnosticConfigurationRevision() != PinnedEngine().Configuration+"-stress-jitter" {
		t.Fatal("stress helper not admitted")
	}
	request.SetStressHostConditions("", false)
	if len(request.stressHostEnvironment) != 2 {
		t.Fatal("jitter switch leaked into reference")
	}
}

func TestStressJitterHelperRefusedForQualification(t *testing.T) {
	t.Parallel()
	request := Request{Engine: PinnedEngine()}
	request.Engine.Configuration += "-stress-jitter"
	_, err := Run(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "requires test opt-in") {
		t.Fatalf("qualification admitted jitter helper: %v", err)
	}
	if _, err := prepareClusterBoot("guest", request); err == nil {
		t.Fatal("cluster qualification admitted jitter helper")
	}
}

func TestDeterminismFixtureConditionsStayOutsideDeclaredInputs(t *testing.T) {
	t.Parallel()
	request := Request{Engine: PinnedEngine(), Config: testConfig()}
	before, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"", "draw", "clock", "handoff", "output-clock"} {
		if err := request.SetDeterminismFixtureConditions(fault); err != nil {
			t.Fatal(err)
		}
		after, err := json.Marshal(request)
		if err != nil || string(before) != string(after) {
			t.Fatalf("fault changed declared inputs: %v", err)
		}
		if len(request.stressHostEnvironment) != 2 || request.stressHostEnvironment[0] != "GOMAD_WASM_DETERMINISM_FAULT="+fault || request.stressHostEnvironment[1] != "GOMAD_WASM_EVENT_ORDER=1" {
			t.Fatal("fixture fault not transported privately")
		}
	}
	if err := request.SetDeterminismFixtureConditions("unknown"); err == nil {
		t.Fatal("unknown fixture fault accepted")
	}
	request.Engine.Configuration += "-determinism-fixture"
	if _, err := Run(t.Context(), request); err == nil {
		t.Fatal("fixture helper admitted without a pinned declared engine")
	}
	request = Request{Engine: request.Engine}
	if _, err := Run(t.Context(), request); err == nil || !strings.Contains(err.Error(), "requires test opt-in") {
		t.Fatalf("fixture helper admitted for qualification: %v", err)
	}
}

func TestDeterminismFixtureClusterStartedUsesPreparedEngine(t *testing.T) {
	input, err := newHelperInput()
	if err != nil {
		t.Fatal(err)
	}
	input.start(nil)
	t.Cleanup(func() {
		if err := input.finish(); err != nil {
			t.Error(err)
		}
	})
	declared := Request{Engine: PinnedEngine(), ModuleSHA256: "fixture", InitialMemoryPages: 1}
	if err := declared.SetDeterminismFixtureConditions(""); err != nil {
		t.Fatal(err)
	}
	prepared := declared
	prepared.Engine.Configuration = prepared.DiagnosticConfigurationRevision()
	started := Started{Schema: "gomad.wasm-host/v1", Engine: prepared.Engine, ModuleSHA256: "fixture", InitialMemoryPages: 1, Imports: []Import{}, Exports: []Export{{Name: "memory", Kind: "memory"}, {Name: "_start", Kind: "func"}}}
	for _, mismatch := range []struct {
		name   string
		mutate func(*Started)
	}{
		{"engine", func(value *Started) { value.Engine = PinnedEngine() }},
		{"module", func(value *Started) { value.ModuleSHA256 = "different" }},
		{"imports", func(value *Started) { value.Imports = []Import{{Module: "future", Name: "operation"}} }},
		{"memory", func(value *Started) { value.InitialMemoryPages++ }},
	} {
		changed := started
		mismatch.mutate(&changed)
		if err := validateClusterStarted(changed, prepared); err == nil {
			t.Fatalf("prepared %s mismatch admitted", mismatch.name)
		}
	}
	frame, err := json.Marshal(map[string]any{"schema": started.Schema, "type": "started", "engine": started.Engine, "module_sha256": started.ModuleSHA256, "imports": started.Imports, "exports": started.Exports, "initial_memory_pages": started.InitialMemoryPages})
	if err != nil {
		t.Fatal(err)
	}
	session := &ClusterSession{config: ClusterRequest{Instances: []ClusterInstance{{ID: "a", Request: declared}}}, boots: map[string]*clusterBoot{"a": {request: prepared, started: &started}}, ready: make(chan struct{}), input: input}
	if err := (&clusterStream{session: session}).frame(frame); err != nil {
		t.Fatal(err)
	}
}
