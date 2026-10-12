package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/runner"
	runnerbackend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/simulation/network"
	"go.temporal.io/server/tools/gomad3/volume"
	"go.temporal.io/server/tools/gomad3/world"
)

func TestDiagnosticHelperStderrIsSeparateFromGuestEvidence(t *testing.T) {
	request := syntheticHelper(t, "read request\n")
	script := "#!/bin/sh\nread request\nprintf '%s\\n' '" + startedJSON(t, request) + "'\nread authorization\nprintf '%s\\n' 'diagnostic helper bytes' >&2\nprintf '%s\\n' '{\"type\":\"result\",\"termination\":\"exit\",\"exit_code\":0,\"message\":\"\",\"fuel_remaining\":100000000,\"peak_memory_bytes\":65536,\"output_bytes\":0}'\n"
	if err := os.WriteFile(request.HelperPath, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	request.HelperSHA256 = fileHash(t, request.HelperPath)
	result, err := Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "diagnostic transport", result, 0)
	if string(result.helperStderr) != "diagnostic helper bytes\n" {
		t.Fatalf("helper diagnostic capture = %q", result.helperStderr)
	}
	if len(result.Stdout) != 0 || len(result.Stderr) != 0 || len(result.Transcript) != 0 || result.Calls != 0 {
		t.Fatalf("helper diagnostics entered guest evidence: stdout=%q stderr=%q transcript=%q calls=%d", result.Stdout, result.Stderr, result.Transcript, result.Calls)
	}
}

func TestHelperLegacyWorkflowProfile(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_WORKFLOW_PROFILE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_WORKFLOW_PROFILE_ROOT for diagnostic-only retained workflow profiling")
	}
	module, cache, runtimeIdentity := os.Getenv("GOMAD_WASM_MATCHED_MODULE"), os.Getenv("GOMAD_WASM_MATCHED_CACHE"), os.Getenv("GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	if !filepath.IsAbs(root) || !filepath.IsAbs(module) || !filepath.IsAbs(cache) || runtimeIdentity == "" {
		t.Fatal("profile requires absolute evidence/module/existing cache paths and explicit legacy runtime identity")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	repo, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run=^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
	helper := helperPath(t)
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: runtimeIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(runtimeIdentity))}
	request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 3 * time.Minute, CompileCache: cache, DigestTranscript: true, Runtime: &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
	var first Result
	for _, seed := range []uint64{0, 1000003} {
		request.Runtime.Seed = seed
		profile, err := os.Create(filepath.Join(root, "seed"+stringSeed(seed)+"-parent.pprof"))
		if err != nil {
			t.Fatal(err)
		}
		if err := pprof.StartCPUProfile(profile); err != nil {
			_ = profile.Close()
			t.Fatal(err)
		}
		started := time.Now()
		result, runErr := Run(t.Context(), request)
		wall := time.Since(started).Nanoseconds()
		pprof.StopCPUProfile()
		if err := profile.Close(); err != nil {
			t.Fatal(err)
		}
		if runErr != nil {
			t.Fatal(runErr)
		}
		requireExit(t, "profile", result, 0)
		if result.Telemetry == nil || !result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0 || !bytes.Contains(result.Stdout, []byte("=== RUN   TestWorkflowTypeEncodingSuite")) {
			t.Fatal("profile did not execute the unchanged selected workflow using existing compiled code")
		}
		temporalJSON(t, filepath.Join(root, "seed"+stringSeed(seed)+"-metrics.json"), map[string]any{"diagnostic_only": true, "profile": "prototype-legacy-v1", "module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_identity": runtimeIdentity, "model_sha256": ImplementationSHA256(), "wall_nanos": wall, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "choices": result.ChoiceTrace.Summary})
		if seed == 0 {
			first = result
		}
	}
	// Retain the otherwise digest-only canonical bytes in a separate diagnostic
	// run, never using its timing as the unchanged-suite performance measurement.
	request.Runtime.Seed, request.DigestTranscript = 0, false
	retained, err := Run(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "transcript retention", retained, 0)
	if difference := sameEvidence(first, retained); difference != "" {
		t.Fatalf("retaining canonical transcript changed evidence: %s", difference)
	}
	operations := map[string]uint64{}
	for _, line := range bytes.Split(retained.Transcript, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var frame struct {
			Type string `json:"type"`
			Op   string `json:"op"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			t.Fatal(err)
		}
		if frame.Type == "call" {
			operations[frame.Op]++
		}
	}
	temporalJSON(t, filepath.Join(root, "canonical-operation-counts.json"), operations)
	temporalWrite(t, filepath.Join(root, "seed0-choice.bin"), first.ChoiceTrace.Bytes)
}

func TestHelperFailingSuiteDiagnosticProfile(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_DIAGNOSTIC_PROFILE_ROOT")
	if root == "" {
		t.Skip("set diagnostic profile inputs after the exclusive guest window is scheduled")
	}
	module, cache := os.Getenv("GOMAD_WASM_MATCHED_MODULE"), os.Getenv("GOMAD_WASM_MATCHED_CACHE")
	referenceHelper, referenceCache := os.Getenv("GOMAD_WASM_PROFILE_REFERENCE_HELPER"), os.Getenv("GOMAD_WASM_PROFILE_REFERENCE_CACHE")
	runtimeIdentity := os.Getenv("GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	for _, name := range []string{root, module, cache, referenceHelper, referenceCache} {
		if !filepath.IsAbs(name) {
			t.Fatal("diagnostic profile requires absolute paths")
		}
	}
	if runtimeIdentity == "" || fileHash(t, referenceHelper) == fileHash(t, helperPath(t)) {
		t.Fatal("diagnostic helper must have a distinct identity and explicit legacy runtime")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	repo, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: runtimeIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(runtimeIdentity))}
	preparations, err := os.OpenFile(filepath.Join(root, "matched-preparations.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := preparations.Close(); err != nil {
			t.Error(err)
		}
	})
	for index, suite := range []struct{ name, run string }{
		{"workflow", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"child", "^TestChildWorkflowSuite$"},
		{"timers", "^TestUserTimersTestSuite$"},
	} {
		if !t.Run(suite.name, func(t *testing.T) {
			config := prototypeTemporalConfig(t, repo, "/workspace/tests", "-test.run="+suite.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
			config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
			helper := helperPath(t)
			request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 5 * time.Minute, CompileCache: cache, DigestTranscript: true, Runtime: &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
			reference := request
			reference.HelperPath, reference.HelperSHA256, reference.CompileCache = referenceHelper, fileHash(t, referenceHelper), referenceCache
			if index == 0 {
				for epoch, owner := range []struct {
					name    string
					request Request
				}{{"reference", reference}, {"diagnostic", request}} {
					warm := owner.request
					warm.Config.Args = append([]string(nil), warm.Config.Args...)
					for index, arg := range warm.Config.Args {
						if strings.HasPrefix(arg, "-test.run=") {
							warm.Config.Args[index] = "-test.run=^$"
						}
					}
					entries, err := os.ReadDir(warm.CompileCache)
					if err != nil {
						t.Fatal(err)
					}
					cold := len(entries) == 0
					profilePreparationEvent(t, preparations, "start", epoch, warm, cold)
					runDiagnosticProfile(t, root, owner.name+"-prepare", warm, owner.name == "diagnostic", !cold)
					profilePreparationEvent(t, preparations, "end", epoch, warm, cold)
				}
			}
			var first Result
			for _, seed := range []uint64{0, 1000003} {
				request.Runtime.Seed = seed
				before := runDiagnosticProfile(t, root, suite.name+"-reference-seed"+stringSeed(seed), reference, false, true)
				after := runDiagnosticProfile(t, root, suite.name+"-diagnostic-seed"+stringSeed(seed), request, true, true)
				requireDiagnosticEvidence(t, before, after)
				if !bytes.Contains(before.Stdout, []byte("=== RUN   "+strings.Trim(strings.Split(suite.run, "/")[0], "^$"))) {
					t.Fatal("diagnostic selected suite did not run")
				}
				if seed == 0 {
					first = before
				}
			}
			plan, err := choice.ProjectReplayPlan(first.ChoiceTrace, identity)
			if err != nil {
				t.Fatal(err)
			}
			request.Runtime.Seed = 0
			request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &plan
			replayed := runDiagnosticProfile(t, root, suite.name+"-diagnostic-replay", request, true, true)
			requireDiagnosticEvidence(t, first, replayed)
		}) {
			return
		}
	}
}

func TestDiagnosticCompiledIdentityOwners(t *testing.T) {
	owners := map[string]string{"wasi": ImplementationSHA256(), "volume": volume.ImplementationSHA256(), "network": network.ImplementationSHA256(), "interrupted_replay": choice.InterruptedReplayImplementationSHA256(), "runner_clock": runner.SimulationClockImplementationSHA256(), "world_including_canonicaljson": world.ImplementationSHA256()}
	for name, identity := range owners {
		if !strings.HasPrefix(identity, "sha256:") || len(identity) != len("sha256:")+64 {
			t.Fatalf("missing compiled owner identity: %s=%q", name, identity)
		}
	}
	root := os.Getenv("GOMAD_WASM_DIAGNOSTIC_IDENTITY_ROOT")
	if root != "" {
		if !filepath.IsAbs(root) {
			t.Fatal("identity capture requires absolute root")
		}
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		temporalJSON(t, filepath.Join(root, "compiled-owner-identities.json"), map[string]any{"compiled_go_version": runtime.Version(), "owners": owners})
	}
	t.Logf("compiled identity owners: %v", owners)
}

func requireDiagnosticEvidence(t *testing.T, before, after Result) {
	t.Helper()
	if difference := sameEvidence(before, after); difference != "" || before.Calls != after.Calls || before.TimeAdvances != after.TimeAdvances || before.EnvironmentIdentity != after.EnvironmentIdentity || before.FuelRemaining != after.FuelRemaining {
		t.Fatalf("profiling changed canonical execution: %s", difference)
	}
}

func profilePreparationEvent(t *testing.T, channel *os.File, event string, epoch int, request Request, cold bool) {
	t.Helper()
	data, err := json.Marshal(struct {
		Event        string `json:"event"`
		Epoch        int    `json:"epoch"`
		Helper       string `json:"helper"`
		HelperSHA256 string `json:"helper_sha256"`
		Cold         bool   `json:"cold"`
	}{event, epoch, request.HelperPath, request.HelperSHA256, cold})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := channel.Write(append(data, '\n')); err != nil {
		t.Fatal(err)
	}
}

func runDiagnosticProfile(t *testing.T, root, name string, request Request, instrumented, warm bool) Result {
	t.Helper()
	profile, err := os.Create(filepath.Join(root, name+"-parent.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.StartCPUProfile(profile); err != nil {
		_ = profile.Close()
		t.Fatal(err)
	}
	var memoryBefore, memoryAfter runtime.MemStats
	runtime.ReadMemStats(&memoryBefore)
	started := time.Now()
	result, runErr := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) {
		return Run(t.Context(), request)
	})
	wall := time.Since(started).Nanoseconds()
	runtime.ReadMemStats(&memoryAfter)
	pprof.StopCPUProfile()
	if err := profile.Close(); err != nil {
		t.Fatal(err)
	}
	heap, err := os.Create(filepath.Join(root, name+"-parent-heap.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pprof.WriteHeapProfile(heap); err != nil {
		_ = heap.Close()
		t.Fatal(err)
	}
	if err := heap.Close(); err != nil {
		t.Fatal(err)
	}
	temporalJSON(t, filepath.Join(root, name+"-parent-memory.json"), map[string]any{"diagnostic_only": true, "before": memoryBefore, "after": memoryAfter})
	temporalWrite(t, filepath.Join(root, name+"-helper-stderr.txt"), result.helperStderr)
	temporalWrite(t, filepath.Join(root, name+"-guest-stdout.txt"), result.Stdout)
	temporalWrite(t, filepath.Join(root, name+"-guest-stderr.txt"), result.Stderr)
	var diagnostic *diagnosticReport
	var diagnosticErr error
	if instrumented {
		var decoded diagnosticReport
		decoded, diagnosticErr = decodeDiagnosticReport(result.helperStderr)
		if diagnosticErr == nil {
			diagnostic = &decoded
		}
	}
	temporalJSON(t, filepath.Join(root, name+"-metrics.json"), map[string]any{"diagnostic_only": true, "compatibility_profile": "prototype-legacy-v1", "module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_identity": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, "model_sha256": ImplementationSHA256(), "instrumented": instrumented, "run_wall_nanos": wall, "authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos, "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage, "helper_profile": diagnostic, "scope_labels": map[string]string{"run_wall": "Go Run boundary; excludes diagnostics publication and pprof finalization", "phase_cpu": "helper RUSAGE_SELF samples within named phase; no parent CPU", "wait_cpu": "entire helper process including preparation", "nested_wall": "overlaps ABI and phase spans; never sum as disjoint work", "residual_wall": "unattributed Wasmtime/trampoline wall; never guest-only CPU", "peak_rss": "cumulative process high-water, not phase-local allocation"}})
	persistDiagnosticRawResult(t, root, name, result)
	if runErr != nil {
		t.Fatal(runErr)
	}
	requireExit(t, name, result, 0)
	if diagnosticErr != nil {
		t.Fatal(diagnosticErr)
	}
	if !instrumented && len(result.helperStderr) != 0 {
		t.Fatal("release-default helper emitted diagnostic bytes")
	}
	if result.Telemetry == nil || warm && (!result.Telemetry.CacheHit || result.Telemetry.CompileNanos != 0) {
		t.Fatal("profile phase metrics or warm cache contract")
	}
	return result
}

// This diagnostic is not a replacement for the unchanged-suite performance gate.
func TestHelperRuntimeCostProfile(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_PROFILE_ROOT")
	if root == "" {
		t.Skip("set GOMAD_WASM_PROFILE_ROOT for actual runtime-loop profiling")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("profile evidence root must be absolute")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	request := cooperativeRequest(t, "./testdata/runtime_cost")
	request.Config.Clock.ReadStepNanos = 0
	request.Config.Limits.Calls = 1 << 20
	request.Config.Limits.TranscriptBytes = 64 << 20
	request.Runtime.Diagnostics = false
	request.Runtime.Choice.Limit = 64 << 20
	request.Fuel = 1 << 40
	request.Timeout = 3 * time.Minute
	module, err := os.ReadFile(request.ModulePath)
	if err != nil {
		t.Fatal(err)
	}
	temporalWrite(t, filepath.Join(root, "runtime-cost.wasm"), module)
	var reference Result
	for _, owner := range []string{"go", "helper"} {
		request.remoteModel = owner == "go"
		profile, err := os.Create(filepath.Join(root, owner+"-parent.pprof"))
		if err != nil {
			t.Fatal(err)
		}
		if err := pprof.StartCPUProfile(profile); err != nil {
			_ = profile.Close()
			t.Fatal(err)
		}
		started := time.Now()
		result, runErr := Run(t.Context(), request)
		wall := time.Since(started).Nanoseconds()
		pprof.StopCPUProfile()
		if err := profile.Close(); err != nil {
			t.Fatal(err)
		}
		if runErr != nil {
			t.Fatal(runErr)
		}
		requireExit(t, owner, result, 0)
		if !bytes.Contains(result.Stdout, []byte("runtime-cost 100000")) || result.AuthorizationToTerminalNanos == 0 || result.ChoiceTrace.Summary.SelectPoll < 100000 {
			t.Fatalf("runtime loop witness missing: stdout=%q summary=%+v boundary=%d", result.Stdout, result.ChoiceTrace.Summary, result.AuthorizationToTerminalNanos)
		}
		if owner == "go" {
			reference = result
		} else if difference := sameEvidence(reference, result); difference != "" {
			t.Fatalf("runtime-cost profiling evidence differs: %s", difference)
		}
		temporalWrite(t, filepath.Join(root, owner+"-choice.bin"), result.ChoiceTrace.Bytes)
		temporalJSON(t, filepath.Join(root, owner+"-metrics.json"), map[string]any{
			"diagnostic_only": true, "profile": "current-runtime-go-owner-reference",
			"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256,
			"runtime_identity": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey,
			"model_sha256":     ImplementationSHA256(), "wall_nanos": wall,
			"authorization_to_terminal_nanos": result.AuthorizationToTerminalNanos,
			"phase_metrics":                   result.Telemetry, "wait_resource_usage": result.HelperResourceUsage,
			"choices": result.ChoiceTrace.Summary,
		})
	}
}
