package simulation

import (
	"encoding/json"
	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"reflect"
	"testing"

	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func TestOperatorArgumentsSurviveNormalization(t *testing.T) {
	for _, args := range [][]string{nil, {}, {"override"}} {
		s := operatorFixture()
		s.Boots[0].Args = args
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		_, canonical, err := NormalizeOperator(data)
		if err != nil {
			t.Fatal(err)
		}
		sealed, _, err := NormalizeOperator(canonical)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(sealed.Boots[0].Args, args) {
			t.Fatalf("arguments changed: got %#v want %#v", sealed.Boots[0].Args, args)
		}
		base := operatorBase()
		_, provider, err := sealed.Bind(base)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"guest", "inherited"}
		if args != nil {
			want = append([]string{"guest"}, args...)
		}
		if !reflect.DeepEqual(provider.Boots["main"].Config.Args, want) {
			t.Fatalf("boot arguments %q want %q", provider.Boots["main"].Config.Args, want)
		}
	}
}

func operatorBase() wasi.Request {
	return wasi.Request{Config: wasi.Config{Profile: wasi.CooperativeProfile, Args: []string{"guest", "inherited"}, Environment: []string{"PWD=/workspace"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace"}, Limits: wasi.Limits{OutputBytes: 1024, TranscriptBytes: 1 << 20, Calls: 10000, Descriptors: 64, Files: 100, FilesystemBytes: 1 << 20, PendingEvents: 4096}}, Runtime: &wasi.RuntimeControl{Choice: &backends.ChoiceRequest{}}}
}

func TestOperatorAcceptsDeclaredStorageCrash(t *testing.T) {
	s := operatorFixture()
	s.Spec.Nodes = s.Spec.Nodes[:1]
	s.Actions = []OperatorAction{{Action: "start", Handle: model.NodeHandle{Node: "a", Incarnation: 1}}, {Action: "crash", Handle: model.NodeHandle{Node: "a", Incarnation: 1}}}
	s.Spec.Volumes = []model.VolumeSpec{{ID: "data", CapacityBytes: 1024}}
	s.Spec.Nodes[0].Volumes = []model.VolumeMount{{Volume: "data", Path: "/workspace"}}
	s.Boots[0].Storage = &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: "data", Path: "/workspace", CapacityBytes: 1024}}, Limits: volume.VolumeLimits{PendingOperations: 100, Transitions: 1000}, Faults: []wasi.StorageFault{{Name: "crash", Resource: "/workspace/data", Operation: "sync", Occurrence: 1, Action: "crash-before"}}}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	sealed, _, err := NormalizeOperator(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := sealed.Bind(operatorBase()); err != nil {
		t.Fatal(err)
	}
	s.Boots[0].Storage.Faults[0].Action = "io-error"
	data, _ = json.Marshal(s)
	if _, _, err := NormalizeOperator(data); err == nil {
		t.Fatal("unbound crash accepted")
	}
}

func operatorFixture() OperatorScenario {
	return OperatorScenario{Schema: OperatorSchema, Spec: model.Spec{Schema: model.SpecSchema, Backend: model.BackendWASM, Fidelity: model.FidelitySimulationModel, Limits: model.DefaultLimits(), Nodes: []model.NodeSpec{{ID: "a", Boot: "main", Address: "10.0.0.1"}, {ID: "b", Boot: "main", Address: "10.0.0.2"}}}, Boots: []OperatorBoot{{ID: "main"}}, Actions: []OperatorAction{{Action: "start", Handle: model.NodeHandle{Node: "a", Incarnation: 1}}, {Action: "start", Handle: model.NodeHandle{Node: "b", Incarnation: 1}}, {Action: "wait", Handle: model.NodeHandle{Node: "a", Incarnation: 1}}, {Action: "wait", Handle: model.NodeHandle{Node: "b", Incarnation: 1}}}}
}

func TestOperatorStrictPreflight(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*OperatorScenario)
	}{
		{"valid", func(*OperatorScenario) {}},
		{"version", func(s *OperatorScenario) { s.Schema = "unknown" }},
		{"duplicate-node", func(s *OperatorScenario) { s.Spec.Nodes[1].ID = "a" }},
		{"missing-boot", func(s *OperatorScenario) { s.Boots = nil }},
		{"incarnation", func(s *OperatorScenario) { s.Actions[0].Handle.Incarnation = 2 }},
		{"ordering", func(s *OperatorScenario) { s.Actions[0].Action = "restart" }},
		{"unbound-crash", func(s *OperatorScenario) { s.Actions[2].Action = "crash" }},
		{"capacity", func(s *OperatorScenario) { s.Spec.Limits.Nodes = 1 }},
		{"fault-capacity", func(s *OperatorScenario) {
			s.Spec.Limits.FaultActions = 1
			s.NetworkFaults = []wasi.BrokerNetworkFault{{Name: "one"}, {Name: "two"}}
		}},
		{"callback", func(s *OperatorScenario) { s.Actions[0].Action = "callback" }},
		{"missing-reference", func(s *OperatorScenario) { s.Actions[0].Handle.Node = "missing" }},
		{"unknown-restart-boot", func(s *OperatorScenario) {
			s.IncarnationBoots = []IncarnationBoot{{Handle: model.NodeHandle{Node: "a", Incarnation: 2}, Boot: "missing"}}
		}},
		{"duplicate-boot", func(s *OperatorScenario) { s.Boots = append(s.Boots, s.Boots[0]) }},
		{"active-restart", func(s *OperatorScenario) {
			s.Actions[2] = OperatorAction{Action: "restart", Handle: model.NodeHandle{Node: "a", Incarnation: 2}}
		}},
		{"incomplete", func(s *OperatorScenario) { s.Actions = s.Actions[:2] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := operatorFixture()
			tc.mutate(&s)
			data, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = NormalizeOperator(data)
			if (err == nil) != (tc.name == "valid") {
				t.Fatalf("preflight error=%v", err)
			}
		})
	}
	data, _ := json.Marshal(operatorFixture())
	for _, invalid := range [][]byte{append(data, []byte(" {}")...), append([]byte(`{"unknown":true,`), data[1:]...)} {
		if _, _, err := NormalizeOperator(invalid); err == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
}
