package wasi

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	hostfs "go.temporal.io/server/tools/gomad3/internal/host/fs"
)

const warmPreparationSchema = "gomad3.wasm-warm-preparation/v1"

type warmPreparationRun struct {
	Telemetry                                   Telemetry                  `json:"telemetry"`
	Phases                                      map[string]diagnosticPhase `json:"phases"`
	WallNanos, TerminalToReapNanos              uint64
	EvidenceSHA256                              string
	Exited, Reaped                              bool
	ObservationPath, FinishPath, DiagnosticPath string
}

type warmPreparationSeries struct {
	Schema, Role, SourceCommit, SourceRoot, DriverSHA256     string
	HostBootID, HostCPUInfo, ConfigSHA256                    string
	Engine                                                   EngineIdentity
	ModuleSHA256, HelperSHA256, RuntimeIdentity, ModelSHA256 string
	Seed                                                     uint64
	ArtifactBytes, EnvelopeBytes                             uint64
	Inputs                                                   map[string]string
	HelperReceipt, ModuleReceipt, DriverPath, ArtifactPath   string
	Runs                                                     []warmPreparationRun
}

func warmPreparationValidate(series warmPreparationSeries, candidate bool) error {
	role := "baseline"
	if candidate {
		role = "candidate"
	}
	if series.Schema != warmPreparationSchema || series.Role != role || len(series.Runs) < 5 || !warmPreparationHex(series.SourceCommit, 40) || !warmPreparationHex(series.DriverSHA256, 64) || series.SourceRoot == "" || len(series.Inputs) == 0 || !warmPreparationHex(series.ModuleSHA256, 64) || !warmPreparationHex(series.HelperSHA256, 64) || !warmPreparationHex(series.ConfigSHA256, 64) || series.RuntimeIdentity == "" || series.ModelSHA256 == "" || series.ArtifactBytes == 0 || series.ArtifactBytes >= series.EnvelopeBytes {
		return fmt.Errorf("missing source, build, module, artifact or five-run bindings")
	}
	if series.HostBootID == "" || series.Engine.Name != "wasmtime" || series.Engine.Version != "47.0.3" || series.Engine.Configuration == "" || !diagnosticPerformanceHost(series.Engine.HostOS, "arm64", series.HostCPUInfo) || series.Engine.HostArch != "aarch64" {
		return fmt.Errorf("real Linux/arm64 host binding absent")
	}
	var prepare []uint64
	for i, run := range series.Runs {
		if !run.Telemetry.CacheHit || run.Telemetry.CompileNanos != 0 || run.Telemetry.PrepareNanos == 0 || run.WallNanos == 0 || !run.Exited || !run.Reaped || run.TerminalToReapNanos == 0 || candidate && run.TerminalToReapNanos > uint64(time.Second) || !warmPreparationHex(run.EvidenceSHA256, 64) || run.EvidenceSHA256 != series.Runs[0].EvidenceSHA256 {
			return fmt.Errorf("run %d is not a warm successful evidence-identical reaped run", i)
		}
		for _, phase := range strings.Fields("cache_read envelope_verify deserialize cache_identity") {
			if p, ok := run.Phases[phase]; !ok || p.Count == 0 || p.WallNanos > run.Telemetry.PrepareNanos {
				return fmt.Errorf("run %d phase %s absent or inconsistent", i, phase)
			}
		}
		prepare = append(prepare, run.Telemetry.PrepareNanos)
	}
	if candidate && variantTiming(prepare).Median > uint64(time.Second) {
		return fmt.Errorf("warm preparation median exceeds one second")
	}
	return nil
}

func warmPreparationCompare(before, after warmPreparationSeries) error {
	if err := warmPreparationValidate(before, false); err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	if err := warmPreparationValidate(after, true); err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	if before.HostBootID != after.HostBootID || before.HostCPUInfo != after.HostCPUInfo || before.ModuleSHA256 != after.ModuleSHA256 || before.Seed != after.Seed || before.ConfigSHA256 != after.ConfigSHA256 || before.RuntimeIdentity != after.RuntimeIdentity {
		return fmt.Errorf("before/after must share the real host, module, seed, modeled configuration and runtime")
	}
	if before.Runs[0].EvidenceSHA256 != after.Runs[0].EvidenceSHA256 {
		return fmt.Errorf("choice/WASI/network/stdout/stderr/terminal evidence changed")
	}
	return nil
}

func warmPreparationHex(value string, size int) bool {
	_, err := hex.DecodeString(value)
	return len(value) == size && err == nil
}

func warmPreparationFixture() warmPreparationSeries {
	series := warmPreparationSeries{Schema: warmPreparationSchema, Role: "candidate", SourceCommit: strings.Repeat("a", 40), SourceRoot: "/source", DriverSHA256: strings.Repeat("d", 64), HostBootID: "same-boot", HostCPUInfo: "Features: fp asimd\nCPU architecture: 8\n", ConfigSHA256: strings.Repeat("c", 64), Engine: PinnedEngine(), ModuleSHA256: strings.Repeat("a", 64), HelperSHA256: strings.Repeat("b", 64), RuntimeIdentity: "runtime", ModelSHA256: "model", ArtifactBytes: 100, EnvelopeBytes: 200, Inputs: map[string]string{"/source/driver.go": strings.Repeat("d", 64)}}
	for range 5 {
		series.Runs = append(series.Runs, warmPreparationRun{Telemetry: Telemetry{CacheHit: true, PrepareNanos: uint64(time.Second)}, Phases: map[string]diagnosticPhase{"cache_read": {Count: 1, WallNanos: 2}, "envelope_verify": {Count: 1, WallNanos: 3}, "deserialize": {Count: 1, WallNanos: 4}, "cache_identity": {Count: 1, WallNanos: 1}}, WallNanos: 10, TerminalToReapNanos: uint64(time.Second), EvidenceSHA256: strings.Repeat("e", 64), Exited: true, Reaped: true})
	}
	return series
}

func TestWarmPreparationAdmissionControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries)
		want   bool
	}{
		{"five-warm-at-bound", func(*warmPreparationSeries) {}, true},
		{"four-runs", func(s *warmPreparationSeries) { s.Runs = s.Runs[:4] }, false},
		{"cold-run", func(s *warmPreparationSeries) { s.Runs[2].Telemetry.CacheHit = false }, false},
		{"compiled-run", func(s *warmPreparationSeries) { s.Runs[2].Telemetry.CompileNanos = 1 }, false},
		{"prepare-over-bound", func(s *warmPreparationSeries) {
			for i := range s.Runs {
				s.Runs[i].Telemetry.PrepareNanos++
			}
		}, false},
		{"one-slow-prepare-median-still-bound", func(s *warmPreparationSeries) { s.Runs[0].Telemetry.PrepareNanos++ }, true},
		{"slow-terminal", func(s *warmPreparationSeries) { s.Runs[2].TerminalToReapNanos++ }, false},
		{"missing-terminal", func(s *warmPreparationSeries) { s.Runs[2].TerminalToReapNanos = 0 }, false},
		{"unreaped", func(s *warmPreparationSeries) { s.Runs[2].Reaped = false }, false},
		{"failed-exit", func(s *warmPreparationSeries) { s.Runs[2].Exited = false }, false},
		{"missing-phase", func(s *warmPreparationSeries) { delete(s.Runs[2].Phases, "deserialize") }, false},
		{"missing-phase-sample", func(s *warmPreparationSeries) { s.Runs[2].Phases["deserialize"] = diagnosticPhase{} }, false},
		{"unstable-evidence", func(s *warmPreparationSeries) { s.Runs[2].EvidenceSHA256 = strings.Repeat("f", 64) }, false},
		{"emulated-host", func(s *warmPreparationSeries) { s.HostCPUInfo = "vendor_id: GenuineIntel\n" }, false},
		{"missing-host-binding", func(s *warmPreparationSeries) { s.HostBootID = "" }, false},
		{"missing-engine-binding", func(s *warmPreparationSeries) { s.Engine.Configuration = "" }, false},
		{"missing-source-binding", func(s *warmPreparationSeries) { s.SourceCommit = "" }, false},
		{"missing-receipts", func(s *warmPreparationSeries) { s.Inputs = nil }, false},
		{"wrong-artifact-size", func(s *warmPreparationSeries) { s.ArtifactBytes = s.EnvelopeBytes }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			series := warmPreparationFixture()
			test.mutate(&series)
			if err := warmPreparationValidate(series, true); (err == nil) != test.want {
				t.Fatalf("admission=%v, want %v", err, test.want)
			}
		})
	}
}

func TestWarmPreparationComparisonControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*warmPreparationSeries)
		want   bool
	}{
		{"separate-engine-helper-model-bindings", func(s *warmPreparationSeries) {
			s.Engine.Configuration += "-historical"
			s.HelperSHA256 = strings.Repeat("f", 64)
			s.ModelSHA256 = "historical-model"
			s.DriverSHA256 = strings.Repeat("f", 64)
			s.SourceCommit = strings.Repeat("b", 40)
		}, true},
		{"different-host-boot", func(s *warmPreparationSeries) { s.HostBootID = "other-boot" }, false},
		{"different-module", func(s *warmPreparationSeries) { s.ModuleSHA256 = strings.Repeat("f", 64) }, false},
		{"different-seed", func(s *warmPreparationSeries) { s.Seed++ }, false},
		{"different-config", func(s *warmPreparationSeries) { s.ConfigSHA256 = strings.Repeat("f", 64) }, false},
		{"different-runtime", func(s *warmPreparationSeries) { s.RuntimeIdentity = "other-runtime" }, false},
		{"different-evidence", func(s *warmPreparationSeries) {
			for i := range s.Runs {
				s.Runs[i].EvidenceSHA256 = strings.Repeat("f", 64)
			}
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			before, after := warmPreparationFixture(), warmPreparationFixture()
			before.Role = "baseline"
			test.mutate(&before)
			if err := warmPreparationCompare(before, after); (err == nil) != test.want {
				t.Fatalf("comparison=%v, want %v", err, test.want)
			}
		})
	}
}

func warmPreparationEvidence(result Result) string {
	data, _ := json.Marshal(struct {
		Stdout, Stderr, Choices, Network []byte
		WASI                             [32]byte
		Terminal                         [16]byte
		Termination                      string
		ExitCode                         *uint32
	}{result.Stdout, result.Stderr, result.ChoiceTrace.Bytes, result.NetworkTranscript, result.TranscriptSHA256, result.runtimeTerminal, result.Termination, result.ExitCode})
	return digest(data)
}

func TestWarmPreparationEvidenceControls(t *testing.T) {
	code := uint32(0)
	baseline := Result{Stdout: []byte("stdout"), Stderr: []byte("stderr"), NetworkTranscript: []byte("network"), TranscriptSHA256: [32]byte{1}, runtimeTerminal: [16]byte{1}, Termination: "exit", ExitCode: &code}
	baseline.ChoiceTrace.Bytes = []byte("choices")
	for _, test := range []struct {
		field  string
		mutate func(*Result)
		same   bool
	}{
		{"stdout", func(r *Result) { r.Stdout = []byte("other") }, false},
		{"stderr", func(r *Result) { r.Stderr = []byte("other") }, false},
		{"network", func(r *Result) { r.NetworkTranscript = []byte("other") }, false},
		{"wasi", func(r *Result) { r.TranscriptSHA256[0]++ }, false},
		{"terminal", func(r *Result) { r.runtimeTerminal[0]++ }, false},
		{"choices", func(r *Result) { r.ChoiceTrace.Bytes = []byte("other") }, false},
		{"termination", func(r *Result) { r.Termination = "other" }, false},
		{"exit-code", func(r *Result) { other := uint32(1); r.ExitCode = &other }, false},
		{"environment-identity", func(r *Result) { r.EnvironmentIdentity = "other" }, true},
		{"started-identity", func(r *Result) { r.Started = &Started{} }, true},
	} {
		t.Run(test.field, func(t *testing.T) {
			candidate := baseline
			test.mutate(&candidate)
			if (warmPreparationEvidence(candidate) == warmPreparationEvidence(baseline)) != test.same {
				t.Fatal("behavior/identity separation lost", test.field)
			}
		})
	}
}

func TestWarmPreparationDiagnosticCargoFeatureControls(t *testing.T) {
	data := []byte(`{"reason":"compiler-artifact","target":{"name":"gomad3-wasmhost"},"profile":{"opt_level":"3","debug_assertions":false},"executable":"/release/helper","features":["diagnostic-profile"]} {"reason":"build-finished","success":true}`)
	if _, err := terminalExitCargoHelperFeatures(data, []string{"diagnostic-profile"}); err != nil {
		t.Fatal(err)
	}
	if _, err := terminalExitCargoHelper(data); err == nil {
		t.Fatal("release-only gate weakened")
	}
	for _, features := range []string{`[]`, `["guest-profile"]`, `["diagnostic-profile","fuel-off"]`} {
		if _, err := terminalExitCargoHelperFeatures(bytes.ReplaceAll(data, []byte(`["diagnostic-profile"]`), []byte(features)), []string{"diagnostic-profile"}); err == nil {
			t.Fatal("wrong feature admitted", features)
		}
	}
}

func TestBuildWarmPreparationDiagnosticHelper(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_WARM_HELPER_BUILD_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_WARM_HELPER_BUILD_ROOT to build a retained diagnostic-profile helper")
	}
	terminalExitBuildHelper(t, root, []string{"diagnostic-profile"})
}

func warmPreparationSummary(series warmPreparationSeries) map[string]variantTimingRange {
	values := map[string][]uint64{}
	for _, run := range series.Runs {
		values["prepare"] = append(values["prepare"], run.Telemetry.PrepareNanos)
		values["whole_run"] = append(values["whole_run"], run.WallNanos)
		values["terminal_to_reap"] = append(values["terminal_to_reap"], run.TerminalToReapNanos)
		for _, phase := range strings.Fields("cache_read envelope_verify deserialize cache_identity") {
			values[phase] = append(values[phase], run.Phases[phase].WallNanos)
		}
		values["cache_read_plus_deserialize"] = append(values["cache_read_plus_deserialize"], run.Phases["cache_read"].WallNanos+run.Phases["deserialize"].WallNanos)
	}
	summary := map[string]variantTimingRange{}
	for name, samples := range values {
		summary[name] = variantTiming(samples)
	}
	return summary
}

func TestWarmPreparationSummaryControls(t *testing.T) {
	series := warmPreparationFixture()
	for i, value := range []uint64{7, 2, 5, 9, 4} {
		series.Runs[i].Telemetry.PrepareNanos = value
	}
	if got := warmPreparationSummary(series)["prepare"]; got != (variantTimingRange{Median: 5, Min: 2, Max: 9}) {
		t.Fatal("median/range lost", got)
	}
	baseline := warmPreparationFixture()
	baseline.Role = "baseline"
	for i := range baseline.Runs {
		baseline.Runs[i].Telemetry.PrepareNanos++
		baseline.Runs[i].TerminalToReapNanos++
	}
	if err := warmPreparationValidate(baseline, false); err != nil {
		t.Fatal("historical timings incorrectly subject to candidate bounds", err)
	}
}

func warmPreparationRead(t *testing.T, path string) warmPreparationSeries {
	t.Helper()
	var series warmPreparationSeries
	decoder := json.NewDecoder(bytes.NewReader(mustTerminalExitRead(t, path)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&series); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		t.Fatal("trailing warm-preparation receipt", err)
	}
	temporalVerifyInputs(t, series.Inputs)
	if series.Inputs[series.DriverPath] != series.DriverSHA256 || !filepath.IsAbs(series.SourceRoot) || !strings.HasPrefix(series.DriverPath, series.SourceRoot+string(filepath.Separator)) || series.Inputs[series.HelperReceipt] == "" || series.Inputs[series.ModuleReceipt] == "" || series.Inputs[series.ArtifactPath] == "" {
		t.Fatal("historical receipt lacks actual driver/build/module/cache bindings")
	}
	var admission terminalExitHelperAdmission
	if err := json.Unmarshal(mustTerminalExitRead(t, series.HelperReceipt), &admission); err != nil {
		t.Fatal(err)
	}
	var build terminalExitHelperBuild
	if err := json.Unmarshal(mustTerminalExitRead(t, admission.BuildReceipt), &build); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(build.Crate, series.SourceRoot+string(filepath.Separator)) || admission.HelperSHA256 != series.HelperSHA256 {
		t.Fatal("historical helper does not belong to its retained source/build")
	}
	terminalExitHelperInputsFeatures(t, series.HelperReceipt, Request{HelperPath: admission.Helper, HelperSHA256: admission.HelperSHA256}, build.Crate, []string{"diagnostic-profile"})
	for _, run := range series.Runs {
		for _, rawPath := range []string{run.ObservationPath, run.FinishPath, run.DiagnosticPath} {
			if series.Inputs[rawPath] == "" {
				t.Fatal("historical run lacks bound raw observation/finish/diagnostic evidence")
			}
		}
		var observation terminalExitObservation
		if err := json.Unmarshal(mustTerminalExitRead(t, run.ObservationPath), &observation); err != nil {
			t.Fatal(err)
		}
		finish := mustTerminalExitRead(t, run.FinishPath)
		if len(finish) != len(observation.Result.runtimeTerminal) {
			t.Fatal("missing historical private terminal sidecar")
		}
		copy(observation.Result.runtimeTerminal[:], finish)
		profile, err := decodeDiagnosticReport(mustTerminalExitRead(t, run.DiagnosticPath))
		if err != nil || !reflect.DeepEqual(profile.Phases, run.Phases) || observation.Result.Telemetry == nil || *observation.Result.Telemetry != run.Telemetry || warmPreparationEvidence(observation.Result) != run.EvidenceSHA256 || observation.TerminalToReapNanos != run.TerminalToReapNanos || !observation.within(time.Duration(1<<63-1)) || !observation.Result.Reaped || observation.Result.Termination != "exit" || observation.Result.ExitCode == nil || *observation.Result.ExitCode != 0 || observation.Result.Started == nil || observation.Result.Started.Engine != series.Engine || observation.Result.Started.ModuleSHA256 != series.ModuleSHA256 {
			t.Fatal("historical raw observation does not support the declared run", err)
		}
	}
	return series
}

func warmPreparationArtifact(t *testing.T, request Request) (string, uint64, uint64) {
	t.Helper()
	entries, err := filepath.Glob(filepath.Join(request.CompileCache, "*.gomad-cache-v3"))
	if err != nil {
		t.Fatal(err)
	}
	selected := ""
	var artifactBytes, envelopeBytes uint64
	for _, entry := range entries {
		file, info, err := hostfs.OpenPath(entry)
		if err != nil {
			t.Fatal(err)
		}
		var header [24]byte
		_, err = io.ReadFull(file, header[:])
		manifestBytes := binary.LittleEndian.Uint32(header[8:12])
		artifact := binary.LittleEndian.Uint64(header[12:20])
		if err != nil || string(header[:8]) != "GOMADC03" || manifestBytes > 16<<10 || artifact == 0 || artifact > 1<<30 || uint64(info.Size()) != 24+uint64(manifestBytes)+artifact {
			_ = file.Close()
			t.Fatal("invalid current cache envelope", entry, err)
		}
		manifest := make([]byte, manifestBytes)
		_, err = io.ReadFull(file, manifest)
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			t.Fatal("read cache manifest", err, closeErr)
		}
		var identity struct {
			Version       uint64 `json:"version"`
			ModuleSHA256  string `json:"module_sha256"`
			HelperSHA256  string `json:"helper_sha256"`
			Configuration string `json:"configuration"`
			HostOS        string `json:"host_os"`
			HostArch      string `json:"host_arch"`
		}
		if err := json.Unmarshal(manifest, &identity); err != nil {
			t.Fatal(err)
		}
		if identity.ModuleSHA256 != request.ModuleSHA256 || identity.HelperSHA256 != request.HelperSHA256 {
			continue
		}
		if selected != "" || identity.Version != 3 || identity.Configuration != request.Engine.Configuration || identity.HostOS != request.Engine.HostOS || identity.HostArch != request.Engine.HostArch || filepath.Base(entry) != digest(manifest)+".gomad-cache-v3" {
			t.Fatal("ambiguous or incompatible measured cache identity")
		}
		selected, artifactBytes, envelopeBytes = entry, artifact, uint64(info.Size())
	}
	if selected == "" {
		t.Fatal("prewarmed matching cache artifact required; this driver never cold-compiles intentionally")
	}
	return selected, artifactBytes, envelopeBytes
}

func TestTemporalWarmPreparationAcceptance(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_WARM_PREPARATION_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_WARM_PREPARATION_ROOT for serial real-host five-run acceptance collection")
	}
	paths := map[string]string{"root": root, "helper": os.Getenv("GOMAD_WASM_TEST_HELPER"), "module": os.Getenv("GOMAD_WASM_WARM_MODULE"), "cache": os.Getenv("GOMAD_WASM_WARM_CACHE"), "helper_receipt": os.Getenv("GOMAD_WASM_WARM_HELPER_RECEIPT"), "module_receipt": os.Getenv("GOMAD_WASM_WARM_MODULE_RECEIPT")}
	for name, path := range paths {
		if !filepath.IsAbs(path) {
			t.Fatal("explicit absolute warm-preparation input required", name)
		}
	}
	cpuinfo, realHost := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if !realHost {
		t.Fatal("real Linux/arm64 performance host required")
	}
	boot, err := hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
	if err != nil || strings.TrimSpace(string(boot)) == "" {
		t.Fatal("host boot identity required", err)
	}
	evidence, err := diagnosticRunRoot(root, "warm-preparation-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	repo, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	request := prototypeRequest(t, paths["module"], config, 0, 30*time.Minute)
	request.CompileCache = paths["cache"]
	// Digest-only matches the unchanged timing boundary; raw evidence is retained
	// once in an untimed final run and checked against every measured run.
	request.DigestTranscript = true
	crate, err := filepath.Abs("../wasmhost")
	if err != nil {
		t.Fatal(err)
	}
	inputs := terminalExitHelperInputsFeatures(t, paths["helper_receipt"], request, crate, []string{"diagnostic-profile"})
	maps.Copy(inputs, terminalExitModuleInputs(t, paths["module_receipt"], request))
	commit := strings.TrimSpace(string(temporalCommand(t, evidence, "source-commit", repo, nil, []string{"git", "rev-parse", "HEAD"}).Stdout.RawBytes))
	status := temporalCommand(t, evidence, "source-status", repo, nil, []string{"git", "status", "--porcelain", "--", "tools/gomad3"})
	if len(status.Stdout.RawBytes) != 0 {
		t.Fatal("freeze and commit the measured tool source first")
	}
	tracked := temporalCommand(t, evidence, "source-inventory", repo, nil, []string{"git", "ls-files", "-z", "tools/gomad3"})
	for _, name := range bytes.Split(tracked.Stdout.RawBytes, []byte{0}) {
		if len(name) != 0 {
			path := filepath.Join(repo, string(name))
			inputs[path] = fileHash(t, path)
		}
	}
	driver := filepath.Join(repo, "tools/gomad3/internal/backends/wasm/wasi/warm_preparation_test.go")
	artifact, artifactBytes, envelopeBytes := warmPreparationArtifact(t, request)
	inputs[artifact] = fileHash(t, artifact) // outside all timing windows; warms its pages deliberately.
	configBytes, err := json.Marshal(request.Config)
	if err != nil {
		t.Fatal(err)
	}
	series := warmPreparationSeries{Schema: warmPreparationSchema, Role: "candidate", SourceCommit: commit, SourceRoot: repo, DriverSHA256: fileHash(t, driver), HostBootID: strings.TrimSpace(string(boot)), HostCPUInfo: cpuinfo, ConfigSHA256: digest(configBytes), Engine: request.Engine, ModuleSHA256: request.ModuleSHA256, HelperSHA256: request.HelperSHA256, RuntimeIdentity: request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, ModelSHA256: ImplementationSHA256(), Seed: request.Runtime.Seed, ArtifactBytes: artifactBytes, EnvelopeBytes: envelopeBytes, Inputs: inputs, HelperReceipt: paths["helper_receipt"], ModuleReceipt: paths["module_receipt"], DriverPath: driver, ArtifactPath: artifact}
	failed := ""
	var reference Result
	for i := range 6 {
		var observation terminalExitObservation
		started := time.Now()
		_, admissionErr := matchedPrepareAdmitted("linux", "/proc/meminfo", "1", func() (Result, error) {
			observation = measureTerminalExitContext(t, t.Context(), request, false)
			return observation.Result, nil
		})
		wall := uint64(time.Since(started).Nanoseconds())
		if admissionErr != nil {
			t.Fatal(admissionErr)
		}
		name := fmt.Sprintf("run-%d", i)
		temporalJSON(t, filepath.Join(evidence, name+"-observation.json"), observation)
		temporalWrite(t, filepath.Join(evidence, name+"-helper-stderr.txt"), observation.Host.Stderr.RawBytes)
		if observation.Error != "" || observation.Result.Telemetry == nil {
			t.Fatal("helper observation/telemetry failed", observation.Error)
		}
		if !observation.Result.Telemetry.CacheHit || observation.Result.Telemetry.CompileNanos != 0 {
			t.Fatal("every run, including the unmeasured warm-up, must hit the existing cache without compilation")
		}
		profile, err := decodeDiagnosticReport(observation.Host.Stderr.RawBytes)
		if err != nil || profile.HostOS != "linux" || profile.HostArch != "aarch64" {
			t.Fatal("diagnostic phase/host evidence required", err)
		}
		if i == 0 {
			reference = observation.Result
		} else if difference := sameEvidence(reference, observation.Result); difference != "" {
			failed = "within-series evidence changed: " + difference
		}
		if i != 0 {
			finishPath := filepath.Join(evidence, name+"-finish.bin")
			temporalWrite(t, finishPath, observation.Result.runtimeTerminal[:])
			observationPath, diagnosticPath := filepath.Join(evidence, name+"-observation.json"), filepath.Join(evidence, name+"-helper-stderr.txt")
			for _, path := range []string{observationPath, diagnosticPath, finishPath} {
				inputs[path] = fileHash(t, path)
			}
			series.Runs = append(series.Runs, warmPreparationRun{Telemetry: *observation.Result.Telemetry, Phases: profile.Phases, WallNanos: wall, TerminalToReapNanos: observation.TerminalToReapNanos, EvidenceSHA256: warmPreparationEvidence(observation.Result), Exited: observation.within(time.Second) && observation.Result.Termination == "exit" && observation.Result.ExitCode != nil && *observation.Result.ExitCode == 0, Reaped: observation.Result.Reaped, ObservationPath: observationPath, FinishPath: finishPath, DiagnosticPath: diagnosticPath})
		}
		if !strings.Contains(string(observation.Result.Stdout), "--- PASS: TestWorkflowTypeEncodingSuite") || !strings.Contains(string(observation.Result.Stdout), "TestPlainASCII/Succeeds") {
			failed = "unchanged selected workflow did not pass"
		}
	}
	request.DigestTranscript = false
	retained := measureTerminalExitContext(t, t.Context(), request, false)
	persistDiagnosticRawResult(t, evidence, "untimed-evidence", retained.Result)
	temporalJSON(t, filepath.Join(evidence, "untimed-evidence-observation.json"), retained)
	if retained.Error != "" || sameEvidence(reference, retained.Result) != "" {
		failed = "untimed canonical retention changed behavior"
	}
	temporalVerifyInputs(t, inputs)
	terminalExitHelperInputsFeatures(t, paths["helper_receipt"], request, crate, []string{"diagnostic-profile"})
	terminalExitModuleInputs(t, paths["module_receipt"], request)
	afterCommit := strings.TrimSpace(string(temporalCommand(t, evidence, "source-commit-after", repo, nil, []string{"git", "rev-parse", "HEAD"}).Stdout.RawBytes))
	if afterCommit != commit {
		failed = "source commit changed during collection"
	}
	afterBoot, err := hostfs.ReadBounded("/proc/sys/kernel/random/boot_id", 128)
	afterCPU, afterHost := diagnosticPerformanceHostEvidence("/proc/cpuinfo")
	if err != nil || !bytes.Equal(boot, afterBoot) || !afterHost || cpuinfo != afterCPU {
		failed = "host bindings changed during collection"
	}
	temporalJSON(t, filepath.Join(evidence, "series.json"), series)
	if err := warmPreparationValidate(series, true); err != nil {
		failed = err.Error()
	}
	baselinePath := os.Getenv("GOMAD_WASM_WARM_BASELINE_RECEIPT")
	var baseline *warmPreparationSeries
	var baselineSummary map[string]variantTimingRange
	if baselinePath == "" {
		failed = "baseline-material-gap: collect the historical source with its original matching engine/driver; this candidate never overrides engine identity"
	} else {
		if !filepath.IsAbs(baselinePath) {
			t.Fatal("baseline receipt must be absolute")
		}
		before := warmPreparationRead(t, baselinePath)
		baseline = &before
		baselineSummary = warmPreparationSummary(before)
		if err := warmPreparationCompare(before, series); err != nil {
			failed = err.Error()
		}
		inputs[baselinePath] = fileHash(t, baselinePath)
	}
	summary := warmPreparationSummary(series)
	temporalJSON(t, filepath.Join(evidence, "acceptance.json"), map[string]any{"schema": "gomad3.wasm-warm-preparation-acceptance/v1", "accepted": failed == "", "failure": failed, "candidate": series, "baseline": baseline, "candidate_summary_nanos": summary, "baseline_summary_nanos": baselineSummary, "artifact_bytes": artifactBytes, "envelope_bytes": envelopeBytes, "parked_mapping_owner_decision_required": summary["cache_read_plus_deserialize"].Median > uint64(time.Second), "boundary": "one unmeasured warm-up, five fresh helpers with same workflow and seed on prewarmed page cache; no artifact hashing in timed windows; full source/build/module/cache bindings verified before and after; terminal receipt through supervisor wait/capture/group reaping", "baseline_policy": "historical matching driver and PinnedEngine only; helper/engine/model/source identities separate, modeled inputs and exact behavior must remain equal"})
	if failed != "" {
		t.Fatalf("warm-preparation acceptance not met: %s; retained %s", failed, evidence)
	}
}
