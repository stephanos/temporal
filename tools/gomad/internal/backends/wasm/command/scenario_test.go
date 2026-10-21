package command

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/runner/inspect"
)

type typedOperator struct {
	root, binary, manifest, source, inputs, compiler, runtime string
}

func newTypedOperator(t *testing.T) typedOperator {
	t.Helper()
	compiler := os.Getenv("GOMAD_STOCK_GO")
	if compiler == "" {
		t.Skip("explicit stock Go compiler selects the installed typed scenario gate")
	}
	f := typedOperator{root: t.TempDir()}
	if retained := os.Getenv("GOMAD_WASM_OPERATOR_ROOT"); retained != "" {
		if !filepath.IsAbs(retained) {
			t.Fatal("GOMAD_WASM_OPERATOR_ROOT must be absolute")
		}
		if err := os.MkdirAll(retained, 0o700); err != nil {
			t.Fatal(err)
		}
		var err error
		f.root, err = os.MkdirTemp(retained, "typed-operator-")
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("installed typed acceptance root=%s", f.root)
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
		filepath.Join(f.source, "go.mod"):  []byte("module gomad.wasm.typedoperator\n\ngo 1.27.1\n"),
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
	var config wasi.Config
	configBytes, err := os.ReadFile("testdata/scenario/config.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(configBytes, &config); err != nil {
		t.Fatal(err)
	}
	config.Environment = append(config.Environment, "GOMAD_REQUIRE_CAPTURED_INPUT=1")
	config.CapturedInputs, err = captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: f.inputs, Target: "/inputs"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(f.root, "config.json")
	writeTypedJSON(t, configPath, config)
	helper := os.Getenv("GOMAD_WASM_TEST_HELPER")
	if helper == "" {
		helper = "../wasmhost/target/release/gomad-wasmhost"
	}
	helper, err = filepath.Abs(helper)
	if err != nil {
		t.Fatal(err)
	}
	invokeOperator(t, f.binary, 0, "install", "--output", f.manifest, "--compiler", filepath.Join(f.compiler, "bin/go"), "--helper", helper,
		"--cache", filepath.Join(f.root, "cache"), "--compile-cache", filepath.Join(f.root, "engine-cache"), "--profile", wasi.CooperativeProfile, "--runtime-root", f.runtime, "--config", configPath)
	var discovered struct {
		installation.Description
		Capabilities runnerbackend.Capabilities `json:"capabilities"`
	}
	if err := json.Unmarshal(invokeOperator(t, f.binary, 0, "--installation", f.manifest, "discover"), &discovered); err != nil {
		t.Fatal(err)
	}
	if !discovered.Capabilities.TypedScenarios || discovered.Capabilities.Simulation || discovered.Manifest.Options.Config.Profile != wasi.CooperativeProfile {
		t.Fatalf("typed scenario discovery = %+v", discovered)
	}
	for _, command := range []string{"explore", "inspect", "replay", "recover", "resume"} {
		if !slices.Contains(discovered.Capabilities.Commands, command) {
			t.Fatalf("discovery omitted %s: %+v", command, discovered.Capabilities)
		}
	}
	t.Logf("installation=%s helper=%s engine=%+v memory=%d fuel=%d config=%+v", discovered.SHA256, discovered.Manifest.Options.HelperSHA256, discovered.Manifest.Options.Engine, discovered.Manifest.Options.MemoryBytes, discovered.Manifest.Options.Fuel, config.Limits)
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

func typedScenario(t *testing.T, fixture, mode string) simulation.OperatorScenario {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata/scenario", fixture+".json"))
	if err != nil {
		t.Fatal(err)
	}
	scenario, _, err := simulation.NormalizeOperator(data)
	if err != nil {
		t.Fatal(err)
	}
	if mode != "" {
		for i := range scenario.Boots {
			if scenario.Boots[i].ID == "recover" {
				scenario.Boots[i].Args = []string{mode}
			}
		}
	}
	return scenario
}

func TestOperatorScenarioFixturesPreflight(t *testing.T) {
	for _, fixture := range []string{"durable", "network"} {
		t.Run(fixture, func(t *testing.T) {
			scenario := typedScenario(t, fixture, "")
			if len(scenario.Spec.Nodes) != 2 || len(scenario.Actions) < 6 {
				t.Fatal("operator fixture lost its multi-instance lifecycle")
			}
		})
	}
	data, err := os.ReadFile("testdata/scenario/config.json")
	if err != nil {
		t.Fatal(err)
	}
	var config wasi.Config
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if _, err := wasi.NewEnvironment(config); err != nil {
		t.Fatal(err)
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
			opened, err := artifact.OpenArtifact(filepath.Dir(path))
			if err != nil {
				t.Fatalf("invalid retained artifact %s: %v", filepath.Dir(path), err)
			}
			paths = append(paths, filepath.Dir(path))
			return opened.Close()
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return paths
}

func copyOperatorTree(t *testing.T, destination, source string) {
	t.Helper()
	if err := os.CopyFS(destination, os.DirFS(source)); err != nil {
		t.Fatal(err)
	}
	if err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		return os.Chmod(filepath.Join(destination, relative), info.Mode().Perm())
	}); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledOperatorTypedScenarioDetachedReplay(t *testing.T) {
	f := newTypedOperator(t)
	var paths, scenarios []string
	for _, mode := range []string{"recover", "canary", "network"} {
		scenario := filepath.Join(f.root, mode+".json")
		scenarios = append(scenarios, scenario)
		fixture, bootMode := "durable", mode
		if mode == "network" {
			fixture, bootMode = "network", ""
		}
		writeTypedJSON(t, scenario, typedScenario(t, fixture, bootMode))
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
		wantTransitions := 6
		if index == 2 {
			wantTransitions = 8
		}
		if err != nil || len(retained.Transitions) != wantTransitions || retained.WASM == nil {
			t.Fatalf("retained durable lifecycle: %+v %v", retained, err)
		}
		if index < 2 {
			if len(retained.Volumes.Snapshot.Volumes) != 1 || len(retained.WASM.Incarnations) != 3 {
				t.Fatalf("missing durable incarnations: %+v", retained)
			}
			for _, incarnation := range retained.WASM.Incarnations {
				crashed := incarnation.Handle == (gomadsim.NodeHandle{Node: "a", Incarnation: 1})
				if incarnation.Interrupted != crashed || len(incarnation.ChoiceTape) == 0 || incarnation.StorageSHA256 == "" {
					t.Fatalf("per-incarnation replay completeness changed: %+v", incarnation)
				}
			}
		}
		var output []byte
		for _, observation := range retained.Outputs {
			output = append(output, observation.Bytes...)
		}
		markers := []string{"committed durable-a", "recovered durable-a", "independent peer"}
		if index == 2 {
			markers = []string{"partition blocked dial", "partition blocked accept", "healed server ping", "healed client ping"}
		}
		for _, marker := range markers {
			if !bytes.Contains(output, []byte(marker)) {
				t.Fatalf("retained execution omitted %q: %s", marker, output)
			}
		}
		t.Logf("artifact=%s module=%s scenario-evidence=%s seed=%d incarnations=%d limits=%+v", path, manifest.Target.SHA256, evidence.SHA256, retained.Seed, len(retained.WASM.Incarnations), retained.Limits)
		if err := opened.Close(); err != nil {
			t.Fatal(err)
		}
		want := 0
		if index == 1 {
			want = 1
		}
		replayed := invokeOperator(t, f.binary, want, "--installation", f.manifest, "replay", path)
		if !bytes.Contains(replayed, []byte("reproduced=true")) || !bytes.Contains(replayed, []byte("choice-replay=none")) {
			t.Fatalf("detached typed replay = %s", replayed)
		}
		files := []string{manifest.Target.File}
		for _, reference := range record.BackendReferences(*manifest.Target.Backend) {
			files = append(files, reference.File)
		}
		for _, file := range files {
			for _, damage := range []string{"missing", "tampered"} {
				t.Run(filepath.Base(path)+"/"+strings.ReplaceAll(file, "/", "-")+"/"+damage, func(t *testing.T) {
					corrupt := filepath.Join(t.TempDir(), "artifact")
					copyOperatorTree(t, corrupt, path)
					copied, err := artifact.OpenArtifact(corrupt)
					if err != nil {
						t.Fatalf("copied artifact invalid before %s: %v", damage, err)
					}
					if err := copied.Close(); err != nil {
						t.Fatal(err)
					}
					payloadPath := filepath.Join(corrupt, filepath.FromSlash(file))
					if damage == "missing" {
						if err := os.Remove(payloadPath); err != nil {
							t.Fatal(err)
						}
					} else {
						data, err := os.ReadFile(payloadPath)
						if err != nil || len(data) == 0 {
							t.Fatalf("read retained payload: %v", err)
						}
						data[len(data)-1] ^= 1
						if err := os.WriteFile(payloadPath, data, 0o600); err != nil {
							t.Fatal(err)
						}
					}
					invokeOperator(t, f.binary, 2, "--installation", f.manifest, "replay", corrupt)
				})
			}
		}
		invokeOperator(t, f.binary, 3, "--installation", "/missing/incompatible-installation", "replay", path)
	}
}

// A limit exhausted inside a manual cluster run, or any other instance or
// broker failure there, ends the whole run promptly and classified instead of
// waiting for the wall-clock execution watchdog (fn-235 R1-R4). A capacity
// abort keeps each incarnation's partial streams in a retained, inspectable
// and replayable artifact. Before fn-235 both runs hung until the watchdog.
func TestInstalledOperatorAsymmetricGuestFailureEndsPromptly(t *testing.T) {
	f := newTypedOperator(t)
	scenario := typedScenario(t, "deadlock", "")
	scenario.Boots[0].Args = []string{"fail-peer"}
	scenario.Boots[1].Args = []string{"wait-peer"}
	scenario.Actions[2].Handle.Node = "b"
	scenario.Actions[3].Handle.Node = "a"
	path := filepath.Join(f.root, "asymmetric-failure.json")
	writeTypedJSON(t, path, scenario)
	arguments := f.explore(path, filepath.Join(f.root, "artifacts"), "7")
	for index, argument := range arguments {
		if argument == "--execution-timeout=30s" {
			arguments[index] = "--execution-timeout=3m"
		}
	}
	began := time.Now()
	explored := invokeOperator(t, f.binary, 1, arguments...)
	if wall := time.Since(began); wall > 10*time.Second {
		t.Fatalf("asymmetric guest failure waited %s: %s", wall, explored)
	}
	if !bytes.Contains(explored, []byte(`"classification":"target_failure"`)) || !bytes.Contains(explored, []byte("intentional asymmetric peer failure")) {
		t.Fatalf("asymmetric guest failure classification = %s", explored)
	}
}

func TestInstalledOperatorClusterFailureEndsPromptly(t *testing.T) {
	f := newTypedOperator(t)
	explore := func(fixture string) ([]byte, string) {
		scenario := filepath.Join(f.root, fixture+".json")
		writeTypedJSON(t, scenario, typedScenario(t, fixture, ""))
		artifacts := filepath.Join(f.root, fixture+"-artifacts")
		arguments := f.explore(scenario, artifacts, "7")
		for index, argument := range arguments {
			switch argument {
			case "--execution-timeout=30s":
				arguments[index] = "--execution-timeout=3m"
			case "--overall-timeout=3m":
				arguments[index] = "--overall-timeout=4m"
			}
		}
		began := time.Now()
		explored := invokeOperator(t, f.binary, 3, arguments...)
		wall := time.Since(began)
		t.Logf("%s explore wall=%s", fixture, wall)
		// Preparation compiles the guest; the hang this guards ran to the
		// 3m execution watchdog.
		if wall > 10*time.Second {
			t.Fatalf("%s cluster failure waited %s", fixture, wall)
		}
		return explored, artifacts
	}

	explored, _ := explore("deadlock")
	if !bytes.Contains(explored, []byte(`"classification":"runner_failure"`)) || !bytes.Contains(explored, []byte("backend infrastructure: ")) || !bytes.Contains(explored, []byte("WASM broker global deadlock")) {
		t.Fatalf("deadlocked cluster classification = %s", explored)
	}

	explored, artifacts := explore("capacity")
	if !bytes.Contains(explored, []byte(`"classification":"runner_failure"`)) || !bytes.Contains(explored, []byte("backend capacity: WASI capacity: payload-bytes limit=4096 used=4096 delta=64")) {
		t.Fatalf("exhausted cluster World classification = %s", explored)
	}
	retained := typedArtifactPaths(t, artifacts)
	if len(retained) != 1 {
		t.Fatalf("capacity retained artifacts = %v", retained)
	}
	path := retained[0]
	opened, err := artifact.OpenArtifact(path)
	if err != nil {
		t.Fatal(err)
	}
	manifest := opened.Manifest()
	if manifest.Target.Backend == nil || manifest.Target.Backend.Evidence == nil || manifest.Target.Backend.Evidence.Schema != simulation.RetainedSchema {
		t.Fatalf("capacity artifact lacks retained cluster evidence: %+v", manifest)
	}
	evidence := *manifest.Target.Backend.Evidence
	payload, err := opened.ReadPayload(evidence.File, 128<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	_, clusterRecord, err := simulation.VerifyRetained(target.BackendPayload{Reference: evidence, Data: payload})
	if err != nil || clusterRecord.WASM == nil || len(clusterRecord.WASM.Incarnations) != 2 || clusterRecord.Limits.ObservationBytes != 4096 || clusterRecord.WASM.VirtualElapsedNanos == 0 {
		t.Fatalf("retained capacity abort: %+v %v", clusterRecord, err)
	}
	for _, incarnation := range clusterRecord.WASM.Incarnations {
		if !incarnation.Interrupted || len(incarnation.ChoiceTape) == 0 {
			t.Fatalf("capacity abort lost its replayable prefix: %+v", incarnation)
		}
	}
	streams := map[string][]byte{}
	var stdout, stderr int
	for _, observation := range clusterRecord.Outputs {
		key := string(observation.Handle.Node) + "/" + string(observation.Stream)
		streams[key] = append(streams[key], observation.Bytes...)
		if observation.Stream == gomadsim.OutputStdout {
			stdout += len(observation.Bytes)
		} else {
			stderr += len(observation.Bytes)
		}
	}
	for key, marker := range map[string]string{"a/stderr": "ping client dialing", "a/stdout": "ping client connected", "b/stderr": "echo server listening", "b/stdout": "echo server accepted"} {
		if !bytes.Contains(streams[key], []byte(marker)) {
			t.Fatalf("capacity abort dropped %s partial stream %q: %q", key, marker, streams)
		}
	}
	t.Logf("artifact=%s virtual-elapsed=%d limits=%+v streams=%q", path, clusterRecord.WASM.VirtualElapsedNanos, clusterRecord.Limits, streams)
	var report inspect.Inspection
	if err := json.Unmarshal(invokeOperator(t, f.binary, 0, "inspect", "--json", path), &report); err != nil || report.Artifact == nil || report.Artifact.Stdout.TotalBytes != uint64(stdout) || report.Artifact.Stderr.TotalBytes != uint64(stderr) {
		t.Fatalf("capacity artifact inspection: %+v %v", report, err)
	}
	inspected := report.Artifact.Cluster
	if inspected == nil || inspected.VirtualElapsedNanos != clusterRecord.WASM.VirtualElapsedNanos || len(inspected.Instances) != 2 ||
		inspected.Exhausted == nil || *inspected.Exhausted != (inspect.ClusterLimitInspection{Dimension: "payload-bytes", Limit: 4096, Used: 4096, Delta: 64}) {
		t.Fatalf("capacity inspection omitted the cluster abort: %+v", inspected)
	}
	for _, instance := range inspected.Instances {
		key := instance.Node + "/"
		if !instance.Interrupted || instance.Stdout.Bytes != uint64(len(streams[key+"stdout"])) || instance.Stderr.Bytes != uint64(len(streams[key+"stderr"])) || !strings.HasSuffix(string(streams[key+"stdout"]), instance.Stdout.Tail) {
			t.Fatalf("capacity inspection of %s: %+v", key, instance)
		}
	}
	text := invokeOperator(t, f.binary, 0, "inspect", path)
	if !bytes.Contains(text, []byte("ping round 2")) || !bytes.Contains(text, []byte("exhausted=payload-bytes limit=4096 used=4096 delta=64")) || !bytes.Contains(text, []byte("cluster-instance: node=a incarnation=1")) {
		t.Fatalf("capacity inspection text: %s", text)
	}
	t.Logf("inspect:\n%s", text)
	replayed := invokeOperator(t, f.binary, 1, "--installation", f.manifest, "replay", path)
	if !bytes.Contains(replayed, []byte("reproduced=true")) {
		t.Fatalf("capacity abort replay = %s", replayed)
	}
}
