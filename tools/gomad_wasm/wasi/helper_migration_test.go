package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
)

func stringSeed(seed uint64) string    { return strconv.FormatUint(seed, 10) }
func hexDigest(digest [32]byte) string { return hex.EncodeToString(digest[:]) }

// This gate executes the retained baseline modules, avoiding compiler/source
// drift and excluding cold engine compilation from the paired cost comparison.
func TestHelperMigrationQualification(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_MIGRATION_BASELINE")
	if root == "" {
		t.Skip("set GOMAD_WASM_MIGRATION_BASELINE to retained pre-migration prototype evidence")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("migration baseline must be absolute")
	}
	baseline := map[string]prototypeRun{}
	modules := map[string]string{}
	for _, name := range []string{"scheduling", "virtual-time", "network", "temporal-workflow", "temporal-smoke"} {
		data, err := os.ReadFile(filepath.Join(root, name+"-report.json"))
		if err != nil {
			t.Fatal(err)
		}
		var report prototypeReport
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}
		for _, run := range report.Runs {
			if run.Instance == 0 && !strings.Contains(run.Variant, "warm") && !strings.Contains(run.Variant, "replay") && !strings.HasPrefix(run.Variant, "tamper-") && run.Termination == "exit" {
				baseline[run.Guest+"/"+run.Variant+"/"+stringSeed(run.Seed)] = run
			}
		}
		for name, guest := range report.Guests {
			if hash, ok := guest["module_sha256"].(string); ok {
				modules[name] = hash
			}
		}
	}
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	report := prototypeReportFor(t, "helper-migration")
	for _, guest := range []struct{ name, variant, module, identity, cwd, run string }{
		{"choices", "strict", "choices.wasm", "choices", "", ""},
		{"runtime_probes", "strict", "runtime_probes.wasm", "runtime_probes", "", ""},
		{"time_probes", "strict", "time_probes.wasm", "time_probes", "", ""},
		{"network_probes", "faults", "network_probes.wasm", "network_probes", "", ""},
		{"temporal", "workflow", "prototype-temporal.wasm", "prototype-temporal", "/workspace/prototype", "^TestPrototypeWorkflow$"},
		{"tests", "smoke-frontend", "tests-gomadfunctional.wasm", "frontend", "/workspace/tests/gomadfunctional", "^TestFrontendSystemInfo$"},
		{"tests", "smoke-workflow", "tests.wasm", "workflow", "/workspace/tests", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"tests", "smoke-activity", "tests.wasm", "activity", "/workspace/tests", "^TestActivityTestSuite$"},
		{"tests", "smoke-child", "tests.wasm", "child", "/workspace/tests", "^TestChildWorkflowSuite$"},
		{"tests", "smoke-update", "tests.wasm", "update", "/workspace/tests", "^TestWorkflowUpdateSuite$"},
		{"tests", "smoke-timers", "tests.wasm", "timers", "/workspace/tests", "^TestUserTimersTestSuite$"},
	} {
		t.Run(guest.name+"/"+guest.variant, func(t *testing.T) {
			module := filepath.Join(root, "modules", guest.module)
			if want := modules[guest.identity]; want == "" || fileHash(t, module) != want {
				t.Fatal("baseline guest module identity unavailable or changed")
			}
			config := prototypeFixtureConfig()
			if guest.name == "runtime_probes" {
				config.Environment = append(config.Environment, "EXTRA_ENTROPY=no_")
			}
			if guest.name == "network_probes" {
				config.Args = []string{"network_probes", "expect-faults"}
				config.Network = &NetworkModel{Seed: 11, MaxDelayNanos: uint64(5 * time.Millisecond), Faults: []NetworkFault{{AtNanos: uint64(3 * time.Second), Action: "reset", Port: 9000}, {AtNanos: uint64(6 * time.Second), Action: "partition"}, {AtNanos: uint64(8500 * time.Millisecond), Action: "heal"}}}
			}
			if guest.cwd != "" {
				config = prototypeTemporalConfig(t, repo, guest.cwd, "-test.run="+guest.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
			}
			if guest.name == "temporal" {
				config.Network = &NetworkModel{Seed: 21, MaxDelayNanos: uint64(2 * time.Millisecond), Faults: prototypeFaults}
			}
			request := prototypeRequest(t, module, config, 0, 30*time.Minute)
			request.CompileCache = filepath.Join(root, "compile-cache")
			if cache := os.Getenv("GOMAD_WASM_MIGRATION_CACHE"); cache != "" {
				if !filepath.IsAbs(cache) {
					t.Fatal("migration compiled cache must be absolute")
				}
				request.CompileCache = cache
			}
			warm := request
			warm.Config.Args = append([]string(nil), warm.Config.Args...)
			for index, arg := range warm.Config.Args {
				if strings.HasPrefix(arg, "-test.run=") {
					warm.Config.Args[index] = "-test.run=^$"
				}
			}
			prepared := prototypeExecute(t, report, guest.name, guest.variant+"-prepare", 0, warm)
			requireExit(t, "cache prepare", prepared, 0, 17)
			var first Result
			var beforeNanos, afterNanos int64
			for _, seed := range []uint64{0, 1000003} {
				request.Runtime.Seed = seed
				old, ok := baseline[guest.name+"/"+guest.variant+"/"+stringSeed(seed)]
				if !ok {
					t.Fatal("missing baseline seed")
				}
				started := time.Now()
				result := prototypeExecute(t, report, guest.name, guest.variant, 0, request)
				elapsed := time.Since(started).Nanoseconds()
				requireExit(t, "migrated", result, 0, 17)
				if digest(result.Stdout) != old.StdoutSHA256 || digest(result.ChoiceTrace.Bytes) != old.ChoiceSHA256 || digest(result.NetworkTranscript) != old.NetworkSHA256 || hexDigest(result.TranscriptSHA256) != old.WASISHA256 || int64FromTerminal(result.runtimeTerminal) != int64(old.VirtualNanos) || int(result.Calls) != old.Calls || result.TimeAdvances != old.TimeAdvances {
					t.Fatalf("baseline evidence changed: seed=%d stdout=%v choice=%v network=%v wasi=%v calls=%d/%d virtual=%d/%d", seed, digest(result.Stdout) == old.StdoutSHA256, digest(result.ChoiceTrace.Bytes) == old.ChoiceSHA256, digest(result.NetworkTranscript) == old.NetworkSHA256, hexDigest(result.TranscriptSHA256) == old.WASISHA256, result.Calls, old.Calls, int64FromTerminal(result.runtimeTerminal), old.VirtualNanos)
				}
				beforeNanos += old.WallNanos
				afterNanos += elapsed
				if seed == 0 {
					first = result
				}
			}
			tape, err := choice.ProjectReplayPlan(first.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
			if err != nil {
				t.Fatal(err)
			}
			request.Runtime.Seed = 0
			request.Runtime.Choice.Mode, request.Runtime.Choice.Tape = choice.ModeReplay, &tape
			replayed := prototypeExecute(t, report, guest.name, guest.variant+"-replay", 0, request)
			if difference := sameEvidence(first, replayed); difference != "" {
				t.Fatalf("migrated replay differs in %s: %s %s", difference, replayed.Termination, replayed.Message)
			}
			speedup := float64(beforeNanos) / float64(afterNanos)
			t.Logf("same module/seeds: baseline=%s migrated=%s speedup=%.3fx", time.Duration(beforeNanos), time.Duration(afterNanos), speedup)
			report.guest(guest.variant, map[string]any{"module_sha256": request.ModuleSHA256, "baseline_wall_nanos": beforeNanos, "migrated_wall_nanos": afterNanos, "speedup": speedup, "byte_identical": true})
			if guest.cwd != "" && speedup < 5 {
				t.Fatalf("Temporal speedup %.3fx is below required 5x", speedup)
			}
			if guest.name == "network_probes" && !bytes.Contains(first.Stdout, []byte("dial refused")) {
				t.Fatal("network fault witness missing")
			}
		})
	}
}

// TestHelperMatchedSuiteQualification compares both execution owners against a
// single immutable compiled suite. No historical report can substitute for the
// reference execution, and compilation is excluded by explicit warm runs.
func TestHelperMatchedSuiteQualification(t *testing.T) {
	module := os.Getenv("GOMAD_WASM_MATCHED_MODULE")
	baselineHelper := os.Getenv("GOMAD_WASM_MATCHED_BASELINE_HELPER")
	toolchainIdentity := os.Getenv("GOMAD_WASM_MATCHED_TOOLCHAIN_ID")
	if module == "" || baselineHelper == "" || toolchainIdentity == "" {
		t.Skip("set matched immutable module, original helper and runtime identity")
	}
	if !filepath.IsAbs(module) || !filepath.IsAbs(baselineHelper) {
		t.Fatal("matched inputs must be absolute")
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	helper := helperPath(t)
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	report := prototypeReportFor(t, "helper-matched-suite")
	cache := os.Getenv("GOMAD_WASM_MATCHED_CACHE")
	if cache == "" {
		cache = filepath.Join(prototypeRoot(t), "compile-cache")
	}
	if !filepath.IsAbs(cache) {
		t.Fatal("matched compiled cache must be absolute")
	}
	for _, suite := range []struct{ name, cwd, run string }{
		{"workflow", "/workspace/tests", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		{"activity", "/workspace/tests", "^TestActivityTestSuite$"},
		{"child", "/workspace/tests", "^TestChildWorkflowSuite$"},
		{"update", "/workspace/tests", "^TestWorkflowUpdateSuite$"},
		{"timers", "/workspace/tests", "^TestUserTimersTestSuite$"},
		{"frontend", "/workspace/tests/gomadfunctional", "^TestFrontendSystemInfo$"},
	} {
		only := os.Getenv("GOMAD_WASM_MATCHED_SUITE")
		if only != "" && only != suite.name || only == "" && suite.name == "frontend" {
			continue
		}
		t.Run(suite.name, func(t *testing.T) {
			config := prototypeTemporalConfig(t, repo, suite.cwd, "-test.run="+suite.run, "-persistenceType=sql", "-persistenceDriver=sqlite")
			config.Profile, config.Clock.ReadStepNanos = CooperativeProfile, 0
			identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: toolchainIdentity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(toolchainIdentity))}
			request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 4 << 30, Fuel: 1 << 50, Config: config, Timeout: 30 * time.Minute, CompileCache: cache, DigestTranscript: true, Runtime: &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: choice.MaximumTraceBytes}}}
			baseline := request
			baseline.HelperPath, baseline.HelperSHA256, baseline.remoteModel = baselineHelper, fileHash(t, baselineHelper), true
			if baselineCache := os.Getenv("GOMAD_WASM_MATCHED_BASELINE_CACHE"); baselineCache != "" {
				if !filepath.IsAbs(baselineCache) {
					t.Fatal("baseline compiled cache must be absolute")
				}
				baseline.CompileCache = baselineCache
			}
			for index, warm := range []Request{baseline, request} {
				warm.Config.Args = append([]string(nil), warm.Config.Args...)
				for index, arg := range warm.Config.Args {
					if strings.HasPrefix(arg, "-test.run=") {
						warm.Config.Args[index] = "-test.run=^$"
					}
				}
				started := time.Now()
				result, err := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) {
					return Run(context.Background(), warm)
				})
				if err != nil {
					t.Fatal(err)
				}
				requireExit(t, "cache warmup", result, 0)
				report.guest(suite.name+"-prepare-"+strconv.Itoa(index), map[string]any{"helper_sha256": warm.HelperSHA256, "module_sha256": warm.ModuleSHA256, "wall_nanos": time.Since(started).Nanoseconds(), "phase_metrics": result.Telemetry, "wait_resource_usage": result.HelperResourceUsage})
			}
			var beforeNanos, afterNanos int64
			var beforeExecutionNanos, afterExecutionNanos uint64
			var first Result
			for _, seed := range []uint64{0, 1000003} {
				request.Runtime.Seed = seed
				started := time.Now()
				before := prototypeExecute(t, report, suite.name, "baseline", 0, baseline)
				beforeNanos += time.Since(started).Nanoseconds()
				started = time.Now()
				after := prototypeExecute(t, report, suite.name, "candidate", 0, request)
				afterNanos += time.Since(started).Nanoseconds()
				if before.AuthorizationToTerminalNanos == 0 || after.AuthorizationToTerminalNanos == 0 {
					t.Fatal("validated authorization-to-terminal measurements unavailable")
				}
				beforeExecutionNanos += before.AuthorizationToTerminalNanos
				afterExecutionNanos += after.AuthorizationToTerminalNanos
				requireExit(t, "baseline", before, 0)
				requireExit(t, "candidate", after, 0)
				witness := []byte("=== RUN   " + strings.Trim(strings.Split(suite.run, "/")[0], "^$"))
				if !bytes.Contains(before.Stdout, witness) || !bytes.Contains(after.Stdout, witness) {
					t.Fatalf("selected suite did not execute: missing %q", witness)
				}
				if difference := sameEvidence(before, after); difference != "" || before.EnvironmentIdentity != after.EnvironmentIdentity || before.Calls != after.Calls || before.TimeAdvances != after.TimeAdvances {
					t.Fatalf("same-module paired evidence differs: %s seed=%d", difference, seed)
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
			replayed := prototypeExecute(t, report, suite.name, "candidate-replay", 0, request)
			if difference := sameEvidence(first, replayed); difference != "" {
				t.Fatalf("original-helper tape differs in candidate: %s", difference)
			}
			speedup := float64(beforeNanos) / float64(afterNanos)
			executionSpeedup := float64(beforeExecutionNanos) / float64(afterExecutionNanos)
			report.guest(suite.name, map[string]any{"compatibility_profile": "prototype-legacy-v1", "module_sha256": request.ModuleSHA256, "runtime_identity": identity.ToolchainBuildKey, "model_sha256": ImplementationSHA256(), "reference_owner": "current_go_model", "host_os": runtime.GOOS, "host_arch": runtime.GOARCH, "engine": request.Engine, "environment_identity": first.EnvironmentIdentity, "baseline_helper_sha256": baseline.HelperSHA256, "candidate_helper_sha256": request.HelperSHA256, "baseline_wall_nanos": beforeNanos, "candidate_wall_nanos": afterNanos, "speedup": speedup, "byte_identical": true})
			report.guest(suite.name+"-execution-boundary", map[string]any{"boundary": "authorize-send-to-validated-terminal-includes-instantiation-and-evidence-flush", "baseline_nanos": beforeExecutionNanos, "candidate_nanos": afterExecutionNanos, "speedup": executionSpeedup, "full_wall_speedup": speedup, "full_wall_meets_5x": speedup >= 5})
			t.Logf("same module/seeds/tape: full wall baseline=%s candidate=%s speedup=%.3fx; authorized execution baseline=%s candidate=%s speedup=%.3fx", time.Duration(beforeNanos), time.Duration(afterNanos), speedup, time.Duration(beforeExecutionNanos), time.Duration(afterExecutionNanos), executionSpeedup)
			if executionSpeedup < 5 {
				t.Fatalf("unchanged suite authorized execution speedup %.3fx below required 5x (full wall %.3fx)", executionSpeedup, speedup)
			}
		})
	}
}
