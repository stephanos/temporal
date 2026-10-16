package command

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/runner/inspect"
)

type typedOperator struct {
	root, binary, manifest, source, inputs, compiler, runtime string
}

func newTypedOperator(t *testing.T) typedOperator {
	t.Helper()
	compiler := os.Getenv("GOMAD3_STOCK_GO")
	if compiler == "" {
		t.Skip("explicit stock Go compiler selects the installed typed scenario gate")
	}
	f := typedOperator{root: t.TempDir()}
	f.binary, f.manifest = filepath.Join(f.root, "gomad"), filepath.Join(f.root, "installation.json")
	build := exec.CommandContext(t.Context(), compiler, "build", "-p=1", "-trimpath", "-o", f.binary, "../../../../cmd/gomad")
	build.Env = operatorEnvironment()
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build installed operator: %v\n%s", err, output)
	}
	f.compiler = stockgo.Clone(t, filepath.Join(f.root, "go"))
	f.source, f.inputs, f.runtime = filepath.Join(f.root, "source"), filepath.Join(f.root, "inputs"), filepath.Join(f.root, "runtime")
	for _, path := range []string{f.source, f.inputs, f.runtime} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	fixture, err := os.ReadFile("testdata/scenario/main.go")
	if err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string][]byte{
		filepath.Join(f.source, "go.mod"): []byte("module gomad.wasm.typedoperator\n\ngo 1.27.1\n"),
		filepath.Join(f.source, "main.go"): fixture,
		filepath.Join(f.inputs, "control"): []byte("retained typed input"),
	} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"gomad_semantics.go.txt", "gomad_choicewire_generated.go.txt"} {
		data, err := os.ReadFile(filepath.Join("../../../model/runtimesource", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.runtime, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	config := installation.DefaultConfig(wasi.CooperativeProfile)
	config.CapturedInputs, err = captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: f.inputs, Target: "/inputs"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(f.root, "config.json")
	writeTypedJSON(t, configPath, config)
	helper := os.Getenv("GOMAD_WASM_TEST_HELPER")
	if helper == "" {
		helper = "../wasmhost/target/release/gomad3-wasmhost"
	}
	helper, err = filepath.Abs(helper)
	if err != nil {
		t.Fatal(err)
	}
	invokeOperator(t, f.binary, 0, "install", "--output", f.manifest, "--compiler", filepath.Join(f.compiler, "bin/go"), "--helper", helper,
		"--cache", filepath.Join(f.root, "cache"), "--compile-cache", filepath.Join(f.root, "engine-cache"), "--profile", wasi.CooperativeProfile, "--runtime-root", f.runtime, "--config", configPath)
	return f
}

func writeTypedJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func typedStorageScenario(mode string) map[string]any {
	a := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
	a2 := gomad3sim.NodeHandle{Node: "a", Incarnation: 2}
	b := gomad3sim.NodeHandle{Node: "b", Incarnation: 1}
	storage := &wasi.StorageModel{Volumes: []volume.VolumeConfig{{ID: "a-store", Path: "/workspace", CapacityBytes: 8 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1024, Transitions: 10000}}
	return map[string]any{
		"schema": "gomad-wasm.scenario/v1",
		"spec": gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelitySimulationModel, Limits: gomad3sim.DefaultLimits(),
			Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "durable", Address: "10.0.0.1", Volumes: []gomad3sim.VolumeMount{{Volume: "a-store", Path: "/workspace"}}}, {ID: "b", Boot: "park", Address: "10.0.0.2"}},
			Volumes: []gomad3sim.VolumeSpec{{ID: "a-store", CapacityBytes: 8 << 20}}},
		"boots": []map[string]any{{"id": "durable", "args": []string{"scenario", "durable"}, "storage": storage}, {"id": "recover", "args": []string{"scenario", mode}, "storage": storage}, {"id": "park", "args": []string{"scenario", "park"}}},
		"incarnation_boots": []simulation.IncarnationBoot{{Handle: a2, Boot: "recover"}},
		"output_crashes": []simulation.OutputCrash{{Handle: a, Marker: []byte("committed durable-a")}},
		"actions": []map[string]any{{"action": "start", "handle": a}, {"action": "start", "handle": b}, {"action": "crash", "handle": a}, {"action": "restart", "handle": a2}, {"action": "wait", "handle": a2}, {"action": "wait", "handle": b}},
	}
}

func (f typedOperator) explore(scenario, artifacts, seeds string) []string {
	return []string{"--installation", f.manifest, "explore", "--json", "--scenario", scenario, "--seeds=" + seeds, "--parallel=1", "--on-failure=all", "--keep-successes=all", "--success-limit=32", "--success-bytes=536870912", "--execution-timeout=30s", "--overall-timeout=3m", "--terminate-grace=100ms", "--artifacts", artifacts, "--working-dir", f.source, "go-run", "."}
}

func (f typedOperator) detach(t *testing.T, scenarios ...string) {
	t.Helper()
	paths := append([]string{f.source, f.inputs, f.compiler, f.runtime, filepath.Join(f.root, "config.json"), filepath.Join(f.root, "cache"), filepath.Join(f.root, "engine-cache")}, scenarios...)
	for _, path := range paths {
		if err := os.Rename(path, path+"-unavailable"); err != nil {
			t.Fatal(err)
		}
	}
}

func typedArtifactPaths(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == "manifest.json" {
			if opened, err := artifact.OpenArtifact(filepath.Dir(path)); err == nil {
				paths = append(paths, filepath.Dir(path))
				return opened.Close()
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return paths
}

func TestInstalledOperatorTypedScenarioDetachedReplay(t *testing.T) {
	f := newTypedOperator(t)
	var paths, scenarios []string
	for _, mode := range []string{"recover", "canary"} {
		scenario := filepath.Join(f.root, mode+".json")
		scenarios = append(scenarios, scenario)
		writeTypedJSON(t, scenario, typedStorageScenario(mode))
		want := 0
		if mode == "canary" {
			want = 1
		}
		artifacts := filepath.Join(f.root, mode+"-artifacts")
		invokeOperator(t, f.binary, want, f.explore(scenario, artifacts, "7")...)
		retained := typedArtifactPaths(t, artifacts)
		if len(retained) != 1 {
			t.Fatalf("%s retained artifacts = %v", mode, retained)
		}
		paths = append(paths, retained...)
	}
	f.detach(t, scenarios...)
	invokeOperator(t, f.binary, 0, "--installation", f.manifest, "discover")
	for index, path := range paths {
		var report inspect.Inspection
		if err := json.Unmarshal(invokeOperator(t, f.binary, 0, "inspect", "--json", path), &report); err != nil || report.Artifact == nil {
			t.Fatalf("detached inspection: %+v %v", report, err)
		}
		opened, err := artifact.OpenArtifact(path)
		if err != nil {
			t.Fatal(err)
		}
		manifest := opened.Manifest()
		if manifest.Target.Backend == nil || manifest.Target.Backend.Evidence == nil || manifest.Target.Backend.Evidence.Schema != simulation.RetainedSchema || manifest.ChoiceProfile != nil {
			t.Fatalf("cluster artifact mislabeled as a global exact tape: %+v", manifest)
		}
		evidence := *manifest.Target.Backend.Evidence
		payload, err := opened.ReadPayload(evidence.File, 128<<20)
		if err != nil {
			t.Fatal(err)
		}
		_, retained, err := simulation.VerifyRetained(target.BackendPayload{Reference: evidence, Data: payload})
		if err != nil || len(retained.Transitions) != 6 || len(retained.Volumes.Snapshot.Volumes) != 1 {
			t.Fatalf("retained durable lifecycle: %+v %v", retained, err)
		}
		if !bytes.Contains(payload, []byte("storage")) {
			t.Fatal("retained scenario lost storage evidence")
		}
		if err := opened.Close(); err != nil {
			t.Fatal(err)
		}
		replayed := invokeOperator(t, f.binary, index, "--installation", f.manifest, "replay", path)
		if !bytes.Contains(replayed, []byte("reproduced=true")) || !bytes.Contains(replayed, []byte("choice-replay=none")) {
			t.Fatalf("detached typed replay = %s", replayed)
		}
		for _, reference := range record.BackendReferences(*manifest.Target.Backend) {
			t.Run(filepath.Base(path)+"/"+strings.ReplaceAll(reference.File, "/", "-"), func(t *testing.T) {
				corrupt := filepath.Join(t.TempDir(), "artifact")
				if err := os.CopyFS(corrupt, os.DirFS(path)); err != nil {
					t.Fatal(err)
				}
				payloadPath := filepath.Join(corrupt, filepath.FromSlash(reference.File))
				if err := os.Remove(payloadPath); err != nil {
					t.Fatal(err)
				}
				invokeOperator(t, f.binary, 3, "--installation", f.manifest, "replay", corrupt)
			})
		}
		invokeOperator(t, f.binary, 3, "--installation", "/missing/incompatible-installation", "replay", path)
	}
}
