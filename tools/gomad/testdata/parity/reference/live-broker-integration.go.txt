package wasi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/sqlitebusy"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	netmodel "go.temporal.io/server/tools/gomad/internal/model/network"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

func setChoiceReplay(t *testing.T, request *runnerbackend.ChoiceRequest, mode choice.Mode, source []byte, plan choice.ReplayPlan) {
	t.Helper()
	capability := &runnerbackend.ChoiceCapability{Mode: mode, ExecutionIdentity: request.ExecutionIdentity, Limit: request.Limit,
		Trace: source, TraceSHA256: sha256.Sum256(source), TapeSHA256: plan.SHA256, Decisions: uint64(len(plan.Decisions))}
	if mode == choice.ModePrefix {
		capability.ReplayPlan = &plan
	}
	converted, err := runnerbackend.ChoiceRequestForCapability(capability)
	if err != nil {
		t.Fatal(err)
	}
	*request = *converted
}

// The determinism prototype gates (fn-151 tasks 5-8) run only when
// GOMAD_WASM_PROTOTYPE_ROOT names an absolute evidence directory. Every guest
// uses the cooperative profile with a frozen clock (zero read-step), so time
// moves only when the guest is idle.

// prototypeDeterminism runs seeds x instances fresh guests, requires identical
// evidence per seed, and replays the first seed's tape exactly in another fresh
// guest. Recorded guests run concurrently after a large module's compiled code
// is cached. It returns the per-seed baselines.
func prototypeDeterminism(t *testing.T, report *prototypeReport, guest, variant string, request Request, seeds []uint64, instances int, codes ...uint32) map[uint64]Result {
	t.Helper()
	withSeed := func(seed uint64, tape *choice.ReplayPlan, trace []byte) Request {
		next := request
		runtime := *request.Runtime
		choiceRequest := *runtime.Choice
		if tape != nil {
			setChoiceReplay(t, &choiceRequest, choice.ModeReplay, trace, *tape)
		}
		runtime.Choice, runtime.Seed = &choiceRequest, seed
		next.Runtime = &runtime
		return next
	}
	// Compiling a whole-program module takes several GiB, so cold first runs of
	// large modules never overlap.
	if info, err := os.Stat(request.ModulePath); err == nil && info.Size() > 64<<20 {
		// A run that selects no tests exits right after startup and fills the cache.
		warm := withSeed(seeds[0], nil, nil)
		warm.Config.Args = []string{"temporal.test", "-test.run=^$"}
		warm.Config.NetworkV2 = nil
		prototypeWarm.Lock()
		result, err := prototypeRunOne(report, guest, variant+"-warm", 0, warm)
		prototypeWarm.Unlock()
		if err != nil || result.Termination == "infrastructure" {
			t.Fatalf("warm compile cache: %v %s %s", err, result.Termination, result.Message)
		}
	}
	type job struct {
		seed     uint64
		instance int
		result   Result
		err      error
	}
	var jobs []*job
	for _, seed := range seeds {
		for instance := range instances {
			jobs = append(jobs, &job{seed: seed, instance: instance})
		}
	}
	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Go(func() {
			j.result, j.err = prototypeRunOne(report, guest, variant, j.instance, withSeed(j.seed, nil, nil))
		})
	}
	wg.Wait()
	baselines := map[uint64]Result{}
	for _, j := range jobs {
		if j.err != nil {
			t.Fatal(j.err)
		}
		requireExit(t, fmt.Sprintf("%s/%s seed %d instance %d", guest, variant, j.seed, j.instance), j.result, codes...)
		if j.instance == 0 {
			baselines[j.seed] = j.result
		}
	}
	for _, j := range jobs {
		if j.instance > 0 {
			if stream := sameEvidence(baselines[j.seed], j.result); stream != "" {
				t.Fatalf("%s/%s seed %d fresh instance %d diverged in %s", guest, variant, j.seed, j.instance, stream)
			}
		}
	}
	// The replay is another fresh guest forced through the first seed's tape.
	first := baselines[seeds[0]]
	tape, err := choice.ProjectReplayPlan(first.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	replayed := prototypeExecute(t, report, guest, variant+"-replay", 0, withSeed(seeds[0], &tape, first.ChoiceTrace.Bytes))
	if stream := sameEvidence(first, replayed); stream != "" {
		t.Fatalf("%s/%s exact replay diverged in %s: %s %s", guest, variant, stream, replayed.Termination, replayed.Message)
	}
	return baselines
}

func prototypeFixture(t *testing.T, name string) string {
	t.Helper()
	module := filepath.Join(prototypeRoot(t), "modules", name+".wasm")
	if err := os.MkdirAll(filepath.Dir(module), 0700); err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if name == "choices" {
		dir, err = filepath.Abs(filepath.Join("..", "toolchain", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
	}
	module, _ = prototypeBuild(t, dir, "build", ".", module)
	return module
}

// TestPrototypeSchedulingDeterminism is the fn-151.5 gate: identical evidence
// for 4 seeds x 10 fresh instances, schedule diversity across seeds, exact
// replay, and rejection of tampered tapes.
func TestPrototypeSchedulingDeterminism(t *testing.T) {
	t.Parallel()
	report := prototypeReportFor(t, "scheduling")
	seeds := []uint64{0, 1000003, 2000006, 3000009}
	for _, fixture := range []string{"choices", "runtime_probes"} {
		t.Run(fixture, func(t *testing.T) {
			module := prototypeFixture(t, fixture)
			config := prototypeFixtureConfig()
			if fixture == "runtime_probes" {
				config.Environment = append(config.Environment, "EXTRA_ENTROPY=no_")
			}
			request := prototypeRequest(t, module, config, 0, 2*time.Minute)
			baselines := prototypeDeterminism(t, report, fixture, "strict", request, seeds, 10, 0, 17)
			distinct := distinctChoiceTraces(baselines)
			report.guest(fixture, map[string]any{"module_sha256": request.ModuleSHA256, "distinct_schedules": distinct, "seeds": seeds})
			// The small choices fixture checks tape rejection; runtime_probes must show schedule diversity.
			if fixture != "choices" {
				if distinct < 2 {
					t.Fatalf("%s: %d distinct schedules across %d seeds", fixture, distinct, len(seeds))
				}
				return
			}
			tape, err := choice.ProjectReplayPlan(baselines[0].ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
			if err != nil {
				t.Fatal(err)
			}
			for _, mutation := range []struct {
				name   string
				reason choice.DivergenceReason
			}{{"missing", choice.DivergenceTapeExhausted}, {"extra", choice.DivergenceTapeUnconsumed}, {"altered", choice.DivergenceAlternativeSet}} {
				plan := tape
				var source []byte
				if mutation.name == "missing" {
					plan, err = choice.ProjectInterruptedReplayPlan(source, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
				} else {
					records := slices.Clone(baselines[0].ChoiceTrace.Records)
					target := slices.IndexFunc(records, func(record choice.Record) bool { return record.Kind == choice.KindRunnable })
					if mutation.name == "extra" {
						extra := records[target]
						extra.Ordinal = uint64(len(records))
						records = append(records, extra)
					} else {
						records[target].AlternativeSetDigest = sha256.Sum256([]byte("tampered"))
					}
					altered, buildErr := choice.BuildTrace(records, choice.TerminalComplete)
					if buildErr != nil {
						t.Fatal(buildErr)
					}
					source = altered.Bytes
					plan, err = choice.ProjectReplayPlan(altered, request.Runtime.Choice.ExecutionIdentity)
				}
				if err != nil {
					t.Fatal(err)
				}
				tampered := request
				runtime := *request.Runtime
				choiceRequest := *runtime.Choice
				setChoiceReplay(t, &choiceRequest, choice.ModeReplay, source, plan)
				runtime.Choice = &choiceRequest
				tampered.Runtime = &runtime
				result := prototypeExecute(t, report, fixture, "tamper-"+mutation.name, 0, tampered)
				if result.Termination != "divergence" || result.ChoiceDivergence == nil || result.ChoiceDivergence.Reason != mutation.reason {
					t.Fatalf("tampered tape %s: %s %s %+v", mutation.name, result.Termination, result.Message, result.ChoiceDivergence)
				}
			}
		})
	}
}

// TestPrototypeVirtualTime is the fn-151.6 fixture gate: hours of virtual
// time pass with exact deadlines while wall time stays small.
func TestPrototypeVirtualTime(t *testing.T) {
	t.Parallel()
	report := prototypeReportFor(t, "virtual-time")
	module := prototypeFixture(t, "time_probes")
	request := prototypeRequest(t, module, prototypeFixtureConfig(), 0, 2*time.Minute)
	baselines := prototypeDeterminism(t, report, "time_probes", "strict", request, []uint64{0, 1000003, 2000006}, 5, 0)
	for _, result := range baselines {
		for _, line := range []string{"runnable work keeps time frozen", "sleep 2h woke at 2h0m0s", "timer tie fired at 2h30m0s", "36 ticks took 6h0m0s", "cancellation after 5s"} {
			if !bytes.Contains(result.Stdout, []byte(line)) {
				t.Fatalf("missing %q in:\n%s", line, result.Stdout)
			}
		}
	}
	report.guest("time_probes", map[string]any{"module_sha256": request.ModuleSHA256})
}

var prototypeFaults = []NetworkFaultV2{
	{Name: "workflow-reset", AtNanos: uint64(40 * time.Second), Action: "reset"},
	{Name: "workflow-partition", AtNanos: uint64(100 * time.Second), Action: "partition"},
	{Name: "workflow-heal", AtNanos: uint64(110 * time.Second), Action: "heal"},
}

type prototypeResetEvidence struct {
	Connections, StreamErrors, Reconnections uint64
}

func prototypeNetworkResetWitness(data []byte, fault NetworkFaultV2) (prototypeResetEvidence, error) {
	var transcript struct {
		Operations []struct {
			At      uint64           `json:"at"`
			Fault   *NetworkFaultV2  `json:"fault"`
			Request netmodel.Request `json:"request"`
			Result  netmodel.Result  `json:"result"`
		} `json:"operations"`
		Native netmodel.Record `json:"native"`
	}
	if err := json.Unmarshal(data, &transcript); err != nil {
		return prototypeResetEvidence{}, err
	}
	var evidence prototypeResetEvidence
	for _, transition := range transcript.Native.Transitions {
		if transition.Kind == "crash" && transition.Outcome == "reset" {
			evidence.Connections += transition.Count
		}
	}
	named := false
	handles := map[uint64]bool{}
	pending := map[uint64]uint64{}
	for _, operation := range transcript.Operations {
		if operation.Fault != nil && *operation.Fault == fault {
			named = true
		}
		if operation.Request.Op == "dial" && operation.Result.State == "completed" && operation.Result.Error == "" && operation.Result.Handle != 0 {
			if operation.At < fault.AtNanos {
				handles[operation.Result.Handle] = true
			} else {
				evidence.Reconnections++
			}
		}
		if operation.Request.Op == "read" || operation.Request.Op == "write" {
			pending[operation.Request.RequestID] = operation.Request.Handle
		}
		handle := operation.Request.Handle
		if operation.Request.Op == "poll" {
			handle = pending[operation.Request.RequestID]
		}
		if operation.At >= fault.AtNanos && handles[handle] && (operation.Request.Op == "read" || operation.Request.Op == "write" || operation.Request.Op == "poll") && (operation.Result.Error == "use of closed network connection" || operation.Result.Error == "connection reset by peer") {
			evidence.StreamErrors++
		}
	}
	if !named || evidence.Connections == 0 || evidence.StreamErrors == 0 || evidence.Reconnections == 0 {
		return evidence, fmt.Errorf("named reset lacks connection reset and recovery: named=%v evidence=%+v", named, evidence)
	}
	return evidence, nil
}

func TestPrototypeNetworkResetWitnessRequiresAffectedConnection(t *testing.T) {
	fault := NetworkFaultV2{Name: "workflow-reset", Action: "reset", AtNanos: 10}
	t.Run("no connections", func(t *testing.T) {
		e := streamEnvironment(t, streamConfig())
		e.networkV2.config.Faults = []NetworkFaultV2{fault}
		if err := e.networkV2Advance(fault.AtNanos); err != nil {
			t.Fatal(err)
		}
		data := e.NetworkV2Transcript()
		if !bytes.Contains(data, []byte(`"name":"workflow-reset"`)) {
			t.Fatal("negative control did not record named reset")
		}
		if evidence, err := prototypeNetworkResetWitness(data, fault); err == nil || evidence.Connections != 0 {
			t.Fatalf("reset without connections passed witness: %+v, %v", evidence, err)
		}
	})
	t.Run("reset and reconnect", func(t *testing.T) {
		e := streamEnvironment(t, streamConfig())
		e.networkV2.config.Faults = []NetworkFaultV2{fault}
		listener := streamCall(t, e, streamFrame(1, 1, 0, 0, 0, 0, nil))
		port := binary.BigEndian.Uint32(listener[48:])
		client := streamCall(t, e, streamFrame(2, 2, 0, 0, port, 0, nil))
		streamCall(t, e, streamFrame(3, 3, binary.BigEndian.Uint64(listener[16:]), 0, 0, 0, nil))
		e.now = fault.AtNanos
		streamCall(t, e, streamFrame(5, 4, binary.BigEndian.Uint64(client[16:]), 0, 0, 1, []byte{1}))
		streamCall(t, e, streamFrame(2, 5, 0, 0, port, 0, nil))
		evidence, err := prototypeNetworkResetWitness(e.NetworkV2Transcript(), fault)
		if err != nil || evidence.Connections != 1 || evidence.StreamErrors != 1 || evidence.Reconnections != 1 {
			t.Fatalf("reset and reconnect witness: %+v, %v", evidence, err)
		}
	})
}

// TestPrototypeTemporal runs the fn-151.6/.7/.8 one-box gates: a workflow with
// timers, retries and network faults, and an unchanged functional suite.
func TestPrototypeTemporal(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	instances := prototypeLimit("GOMAD_WASM_PROTOTYPE_INSTANCES", 1)
	only := os.Getenv("GOMAD_WASM_PROTOTYPE_ONLY")
	selected := func(name string) bool { return only == "" || strings.Contains(","+only+",", ","+name+",") }
	tags := []string{"disable_grpc_modules", "test_dep", "sqlite3_dotlk"}
	persistence := []string{"-persistenceType=sql", "-persistenceDriver=sqlite"}
	modules := filepath.Join(prototypeRoot(t), "modules")
	if err := os.MkdirAll(modules, 0700); err != nil {
		t.Fatal(err)
	}
	seeds := []uint64{0, 1000003}

	t.Run("workflow", func(t *testing.T) {
		if !selected("workflow") {
			t.Skip("not selected")
		}
		t.Parallel()
		report := prototypeReportFor(t, "temporal-workflow")
		staged := filepath.Join(".tmp", "wasm-backend-temporal", "prototype-temporal")
		if err := os.RemoveAll(filepath.Join(root, staged)); err != nil {
			t.Fatal(err)
		}
		if err := temporalStageFixture(root, filepath.Join("tools", "gomad", "internal", "backends", "wasm", "testdata", "prototype_temporal"), root, staged); err != nil {
			t.Fatal(err)
		}
		module, seconds := prototypeBuild(t, root, "test", "./"+staged, filepath.Join(modules, "prototype-temporal.wasm"), tags...)
		config := prototypeTemporalConfig(t, root, "/workspace/prototype", append([]string{"-test.run=^TestPrototypeWorkflow$"}, persistence...)...)
		config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}, Faults: prototypeFaults}
		request := prototypeRequest(t, module, config, 0, 30*time.Minute)
		baselines := prototypeDeterminism(t, report, "temporal", "workflow", request, seeds, instances, 0)
		resetEvidence := map[string]prototypeResetEvidence{}
		for seed, result := range baselines {
			if !bytes.Contains(result.Stdout, []byte("--- PASS: TestPrototypeWorkflow")) || !bytes.Contains(result.Stdout, []byte("workflow completed [step 0 attempt 2 step 1 attempt 2]")) {
				t.Fatalf("seed %d: workflow did not pass with both activity retries", seed)
			}
			witness, err := prototypeNetworkResetWitness(result.NetworkTranscript, prototypeFaults[0])
			if err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
			resetEvidence[strconv.FormatUint(seed, 10)] = witness
		}
		result := baselines[seeds[0]]
		partitions := bytes.Count(result.NetworkTranscript, []byte(`"name":"workflow-partition"`))
		info, _ := os.Stat(module)
		report.guest("prototype-temporal", map[string]any{"module_sha256": request.ModuleSHA256, "module_bytes": info.Size(), "build_seconds": seconds, "model": config.NetworkV2, "reset_evidence": resetEvidence, "activity_retries": 2, "partitions": partitions, "distinct_schedules": distinctChoiceTraces(baselines)})
		if distinctChoiceTraces(baselines) < 2 {
			t.Fatal("seeds produced identical schedules")
		}
	})

	// The default unchanged suite is the small gomadfunctional package. The
	// ./tests suites (smoke-workflow, smoke-activity, smoke-child, smoke-update,
	// smoke-timers) compile a 338 MB module that needs about 6 GiB of free host
	// memory once, so they run only when GOMAD_WASM_PROTOTYPE_ONLY selects them.
	type suite struct{ pkg, cwd, run string }
	smoke := map[string]suite{
		"frontend": {"./tests/gomadfunctional", "/workspace/tests/gomadfunctional", "^TestFrontendSystemInfo$"},
		"workflow": {"./tests", "/workspace/tests", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"},
		"activity": {"./tests", "/workspace/tests", "^TestActivityTestSuite$"},
		"child":    {"./tests", "/workspace/tests", "^TestChildWorkflowSuite$"},
		"update":   {"./tests", "/workspace/tests", "^TestWorkflowUpdateSuite$"},
		"timers":   {"./tests", "/workspace/tests", "^TestUserTimersTestSuite$"},
	}
	var suites []string
	for name := range smoke {
		if only == "" && name == "frontend" || only != "" && (selected("smoke") || selected("smoke-"+name)) {
			suites = append(suites, name)
		}
	}
	slices.Sort(suites)
	if len(suites) == 0 {
		return
	}
	t.Run("smoke", func(t *testing.T) {
		t.Parallel()
		report := prototypeReportFor(t, "temporal-smoke")
		for _, name := range suites {
			t.Run(name, func(t *testing.T) {
				selected := smoke[name]
				module, seconds := prototypeBuild(t, root, "test", selected.pkg, filepath.Join(modules, strings.ReplaceAll(strings.TrimPrefix(selected.pkg, "./"), "/", "-")+".wasm"), tags...)
				info, _ := os.Stat(module)
				report.guest(name, map[string]any{"package": selected.pkg, "module_sha256": fileHash(t, module), "module_bytes": info.Size(), "build_seconds": seconds})
				config := prototypeTemporalConfig(t, root, selected.cwd, append([]string{"-test.run=" + selected.run}, persistence...)...)
				request := prototypeRequest(t, module, config, 0, 30*time.Minute)
				baselines := prototypeDeterminism(t, report, "tests", "smoke-"+name, request, seeds, instances, 0)
				for seed, result := range baselines {
					if !bytes.Contains(result.Stdout, []byte("\nPASS\n")) {
						t.Fatalf("seed %d: suite %s did not pass", seed, name)
					}
				}
				if distinctChoiceTraces(baselines) < 2 {
					t.Fatalf("suite %s seeds produced identical schedules", name)
				}
			})
		}
	})
}

func TestCooperativeGoGuestForcesExactReplay(t *testing.T) {
	request := cooperativeRequest(t, "../toolchain/testdata/choices")
	observed, err := Run(context.Background(), request)
	if err != nil || observed.Termination != "exit" || observed.ExitCode == nil || *observed.ExitCode != 0 || !observed.Reaped {
		t.Fatalf("actual guest: %s %s err=%v stderr=%s", observed.Termination, observed.Message, err, observed.Stderr)
	}
	if observed.ChoiceTrace.Summary.Runnable == 0 || observed.ChoiceTrace.Summary.SelectPoll == 0 || len(observed.DiagnosticTrace.Records) != len(observed.ChoiceTrace.Records) || !bytes.Contains(observed.Stdout, []byte("timer")) {
		t.Fatalf("actual hook coverage: summary=%+v diagnostics=%d stdout=%s", observed.ChoiceTrace.Summary, len(observed.DiagnosticTrace.Records), observed.Stdout)
	}
	tape, err := choice.ProjectReplayPlan(observed.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	setChoiceReplay(t, request.Runtime.Choice, choice.ModeReplay, observed.ChoiceTrace.Bytes, tape)
	request.Runtime.Diagnostics = false
	replayed, err := Run(context.Background(), request)
	if err != nil || replayed.Termination != "exit" || !replayed.Reaped || !bytes.Equal(observed.ChoiceTrace.Bytes, replayed.ChoiceTrace.Bytes) || !bytes.Equal(observed.Stdout, replayed.Stdout) || !bytes.Equal(observed.Transcript, replayed.Transcript) {
		t.Fatalf("actual exact replay: %s %s err=%v stderr=%s trace=%x/%x evidence=%x/%x", replayed.Termination, replayed.Message, err, replayed.Stderr, observed.ChoiceTrace.SHA256, replayed.ChoiceTrace.SHA256, sha256.Sum256(observed.Transcript), sha256.Sum256(replayed.Transcript))
	}
	t.Logf("actual cooperative guest record/exact replay: module=%s helper=%s runtime=%s choices=%+v diagnostics=%d", request.ModuleSHA256, request.HelperSHA256, request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, observed.ChoiceTrace.Summary, len(observed.DiagnosticTrace.Records))
	for _, mutation := range []struct {
		name   string
		reason choice.DivergenceReason
	}{{"missing", choice.DivergenceTapeExhausted}, {"extra", choice.DivergenceTapeUnconsumed}, {"kind", choice.DivergenceKind}, {"site", choice.DivergenceSite}, {"enabled", choice.DivergenceAlternativeSet}} {
		t.Run(mutation.name, func(t *testing.T) {
			plan := tape
			var source []byte
			if mutation.name == "missing" {
				plan, err = choice.ProjectInterruptedReplayPlan(source, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
			} else {
				records := slices.Clone(observed.ChoiceTrace.Records)
				target := slices.IndexFunc(records, func(record choice.Record) bool { return record.Kind == choice.KindRunnable })
				if target < 0 {
					t.Fatal("fixture has no runnable record to alter")
				}
				switch mutation.name {
				case "extra":
					extra := records[target]
					extra.Ordinal = uint64(len(records))
					records = append(records, extra)
				case "kind":
					// A select poll carries no creation metadata.
					records, target = withoutCreationMetadata(records, target)
					records[target].Kind = choice.KindSelectPoll
				case "site":
					records[target].Flags &^= choice.FlagSiteMissing
					records[target].SiteOffset++
				case "enabled":
					// Creation metadata is bound to the recorded alternative set.
					records, target = withoutCreationMetadata(records, target)
					records[target].AlternativeSetDigest = sha256.Sum256([]byte("inapplicable alternatives"))
				}
				altered, buildErr := choice.BuildTrace(records, choice.TerminalComplete)
				if buildErr != nil {
					t.Fatal(buildErr)
				}
				source = altered.Bytes
				plan, err = choice.ProjectReplayPlan(altered, request.Runtime.Choice.ExecutionIdentity)
			}
			if err != nil {
				t.Fatal(err)
			}
			setChoiceReplay(t, request.Runtime.Choice, choice.ModeReplay, source, plan)
			result, runErr := Run(t.Context(), request)
			if runErr != nil || result.Termination != "divergence" || result.ChoiceDivergence == nil || result.ChoiceDivergence.Reason != mutation.reason || !result.Reaped {
				t.Fatalf("actual tape %s: %s %s divergence=%+v err=%v", mutation.name, result.Termination, result.Message, result.ChoiceDivergence, runErr)
			}
		})
	}
}

func TestCooperativeRuntimeProbes(t *testing.T) {
	request := cooperativeRequest(t, "../testdata/runtime_probes")
	request.Config.Environment = append(request.Config.Environment, "EXTRA_ENTROPY=no_")
	observed, err := Run(context.Background(), request)
	if err != nil || observed.Termination != "exit" || observed.ExitCode == nil || (*observed.ExitCode != 0 && *observed.ExitCode != 17) || !observed.Reaped {
		t.Fatalf("actual runtime probes: %s %s err=%v stdout=%s stderr=%s", observed.Termination, observed.Message, err, observed.Stdout, observed.Stderr)
	}
	for _, witness := range []string{"entropy ", "map ", "runq ", "timer callbacks ", "timer parking", "note negative park wake", "collector automatic finalizer cleanup 1 1", "canary "} {
		if !bytes.Contains(observed.Stdout, []byte(witness)) {
			t.Fatalf("missing runtime witness %q: %s", witness, observed.Stdout)
		}
	}
	t.Logf("runtime probes stdout=%s summary=%+v diagnostics=%d", observed.Stdout, observed.ChoiceTrace.Summary, len(observed.DiagnosticTrace.Records))
	base := observed
	t.Run("diagnostic-divergence", func(t *testing.T) {
		records := slices.Clone(base.DiagnosticTrace.Records)
		for index := range records {
			records[index].RuntimeRandDraws++
		}
		altered, err := choice.BuildDiagnosticTrace(records, base.DiagnosticTrace.Capacity)
		if err != nil {
			t.Fatal(err)
		}
		difference, err := choice.DiffDiagnostics(base.DiagnosticTrace.Bytes, altered.Bytes)
		if err != nil || difference == nil || difference.Ordinal != 0 || !slices.Contains(difference.Fields, "runtime_rand_draws") {
			t.Fatalf("actual guest diagnostic mutation undetected: %+v %v", difference, err)
		}
	})
	t.Run("frozen-clock-parking", func(t *testing.T) {
		frozen := request
		frozen.Config.Clock.ReadStepNanos = 0
		result, err := Run(t.Context(), frozen)
		if err != nil || result.Termination != "exit" || !result.Reaped || !bytes.Contains(result.Stdout, []byte("timer parking")) || !bytes.Contains(result.Stdout, []byte("note negative park wake")) || !bytes.Contains(result.Stdout, []byte("collector automatic finalizer cleanup 1 1")) {
			t.Fatalf("frozen-clock idle handshake: %s %s err=%v stdout=%s stderr=%s", result.Termination, result.Message, err, result.Stdout, result.Stderr)
		}
		t.Logf("read-step=0 timer deadlines and negative note parking progressed: %s", result.Stdout)
	})
	for _, perturbation := range []string{"application-draws", "application-key", "runtime-seed"} {
		t.Run(perturbation, func(t *testing.T) {
			changed := request
			changed.Config.Environment = append([]string(nil), request.Config.Environment...)
			control := *request.Runtime
			changed.Runtime = &control
			switch perturbation {
			case "application-draws":
				changed.Config.Environment[len(changed.Config.Environment)-1] = "EXTRA_ENTROPY=yes"
			case "application-key":
				changed.Config.EntropyKey[31] = 99
			case "runtime-seed":
				changed.Runtime.Seed++
			}
			result, err := Run(context.Background(), changed)
			if err != nil || result.Termination != "exit" || !result.Reaped {
				t.Fatalf("entropy perturbation: %s %s %v %s", result.Termination, result.Message, err, result.Stderr)
			}
			first, rest, ok := bytes.Cut(result.Stdout, []byte{'\n'})
			baseFirst, baseRest, baseOK := bytes.Cut(base.Stdout, []byte{'\n'})
			if !ok || !baseOK {
				t.Fatal("missing entropy observation")
			}
			if perturbation == "runtime-seed" {
				if !bytes.Equal(first, baseFirst) || bytes.Equal(rest, baseRest) {
					t.Fatal("runtime seed changed application entropy or failed to vary runtime behavior")
				}
			} else {
				if perturbation == "application-key" && bytes.Equal(first, baseFirst) {
					t.Fatal("application entropy key had no effect")
				}
				if !bytes.Equal(rest, baseRest) || !bytes.Equal(result.ChoiceTrace.Bytes, base.ChoiceTrace.Bytes) || len(result.DiagnosticTrace.Records) != len(base.DiagnosticTrace.Records) {
					t.Fatal("application entropy perturbed runtime behavior")
				}
				for i, record := range result.DiagnosticTrace.Records {
					previous := base.DiagnosticTrace.Records[i]
					if record.RunqDraws != previous.RunqDraws || record.SchedulerDraws != previous.SchedulerDraws || record.SelectDraws != previous.SelectDraws || record.RuntimeRandDraws != previous.RuntimeRandDraws || record.RuntimeCheapRandDraws != previous.RuntimeCheapRandDraws || record.TimerDraws != previous.TimerDraws {
						t.Fatalf("application entropy perturbed seeded draw counters at %d", i)
					}
				}
			}
		})
	}
	tape, err := choice.ProjectReplayPlan(base.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	last := uint64(len(tape.Decisions) - 1)
	if tape.Decisions[last].Kind != choice.KindSelectPoll || tape.Decisions[last].Alternatives != 2 {
		t.Fatal("canary is not the final two-way branch")
	}
	prefix, err := choice.BuildRankPrefix(tape, last, 1-tape.Decisions[last].Selected)
	if err != nil {
		t.Fatal(err)
	}
	setChoiceReplay(t, request.Runtime.Choice, choice.ModePrefix, base.ChoiceTrace.Bytes, prefix)
	alternate, err := Run(context.Background(), request)
	if err != nil || alternate.Termination != "exit" || alternate.ExitCode == nil || *alternate.ExitCode == *base.ExitCode || !alternate.Reaped {
		t.Fatalf("alternate forced canary: %s %s %v stdout=%s stderr=%s", alternate.Termination, alternate.Message, err, alternate.Stdout, alternate.Stderr)
	}
	alternateTrace, err := choice.DecodeStoredTrace(choice.Profile, alternate.ChoiceTrace.Bytes, choice.TerminalMetadata{
		State: alternate.ChoiceTrace.Summary.Terminal, Limit: request.Runtime.Choice.Limit,
		Records: alternate.ChoiceTrace.Summary.Records, SHA256: alternate.ChoiceTrace.SHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	alternateTape, err := choice.ProjectReplayPlan(alternateTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	setChoiceReplay(t, request.Runtime.Choice, choice.ModeReplay, alternate.ChoiceTrace.Bytes, alternateTape)
	request.Runtime.Diagnostics = false
	exact, err := Run(context.Background(), request)
	if err != nil || exact.Termination != "exit" || exact.ExitCode == nil || *exact.ExitCode != *alternate.ExitCode || !bytes.Equal(exact.Stdout, alternate.Stdout) || !bytes.Equal(exact.ChoiceTrace.Bytes, alternate.ChoiceTrace.Bytes) || !bytes.Equal(exact.Transcript, alternate.Transcript) {
		t.Fatalf("alternate exact replay: %s %s %v stdout=%s stderr=%s", exact.Termination, exact.Message, err, exact.Stdout, exact.Stderr)
	}
	t.Logf("forced alternate canary exit=%d then full-tape exact replay; application/runtime entropy independence checked", *alternate.ExitCode)
}

func TestCooperativePreflightRejectsIncompleteRuntimeABI(t *testing.T) {
	request := fixtureRequest(t, `(module (memory (export "memory") 1) (func (export "_start")))`, nil)
	request.Config.Profile = CooperativeProfile
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	request.Runtime = &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}}
	if _, err := Run(context.Background(), request); err == nil {
		t.Fatal("cooperative profile authorized guest without runtime ABI")
	}
}

type runtimeQualification struct {
	SeedIndex, Seed, Instances, ExactReplays                               uint64
	Module, Helper, Runtime, Model                                         string
	SourceSHA256, RuntimeImplementationSHA256, CompilerSHA256, GoVersion   string
	HostGOOS, HostGOARCH, Collector, Profile                               string
	Engine                                                                 EngineIdentity
	Clock                                                                  ClockPolicy
	Limits                                                                 Limits
	ChoiceSHA256, DiagnosticSHA256, StdoutSHA256, StderrSHA256, WASISHA256 string
	TerminalSHA256                                                         string
	ExitCode                                                               uint32
	Summary                                                                choice.Summary
	LastDiagnostic                                                         choice.DiagnosticRecord
}

// Owner gate budget (2026-10-12): fresh instances recorded per qualification seed.
const qualificationInstances = 25

func TestCooperativeRuntimeQualification(t *testing.T) {
	indexValue := os.Getenv("GOMAD_WASM_QUALIFICATION_INDEX")
	if indexValue == "" {
		t.Skip("run qualify-runtime for the mandatory scheduling qualification gate")
	}
	index, err := strconv.ParseUint(indexValue, 10, 64)
	if err != nil || index >= 32 {
		t.Fatal("qualification index must be 0..31")
	}
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if !filepath.IsAbs(root) {
		t.Fatal("qualification requires an absolute retained evidence root")
	}
	request := cooperativeRequest(t, "../testdata/runtime_probes")
	sourceSHA256 := qualificationSourceSHA256(t)
	request.Runtime.Seed = index * 1000003
	request.Config.Environment = append(request.Config.Environment, "EXTRA_ENTROPY=no_")
	var baseline Result
	for instance := range qualificationInstances {
		observed, err := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) {
			return Run(t.Context(), request)
		})
		if err != nil || observed.Termination != "exit" || observed.ExitCode == nil || (*observed.ExitCode != 0 && *observed.ExitCode != 17) || !observed.Reaped || observed.Cancelled || observed.WatchdogTimeout {
			t.Fatalf("seed %d fresh instance %d: %s %s err=%v stderr=%s", request.Runtime.Seed, instance, observed.Termination, observed.Message, err, observed.Stderr)
		}
		for _, witness := range []string{"entropy ", "map ", "runq ", "timer callbacks ", "timer parking", "note negative park wake", "collector automatic finalizer cleanup 1 1", "canary "} {
			if !bytes.Contains(observed.Stdout, []byte(witness)) {
				t.Fatalf("seed %d instance %d missing %q", request.Runtime.Seed, instance, witness)
			}
		}
		if observed.ChoiceTrace.Summary.Runnable == 0 || observed.ChoiceTrace.Summary.SelectPoll == 0 || observed.ChoiceTrace.Summary.SelectResult == 0 || observed.ChoiceTrace.Summary.Terminal != choice.TerminalComplete || len(observed.DiagnosticTrace.Records) != len(observed.ChoiceTrace.Records) {
			t.Fatalf("seed %d instance %d incomplete hook witnesses: %+v", request.Runtime.Seed, instance, observed.ChoiceTrace.Summary)
		}
		last := observed.DiagnosticTrace.Records[len(observed.DiagnosticTrace.Records)-1]
		if last.Allocations == 0 || last.GCCycle == 0 || last.RunqDraws == 0 || last.SelectDraws == 0 || last.RuntimeRandDraws == 0 || last.TimerDraws == 0 {
			t.Fatalf("seed %d instance %d missing collector/seeded-stream evidence: %+v", request.Runtime.Seed, instance, last)
		}
		if instance == 0 {
			baseline = observed
		} else if *observed.ExitCode != *baseline.ExitCode || !bytes.Equal(observed.Stdout, baseline.Stdout) || !bytes.Equal(observed.Stderr, baseline.Stderr) || !bytes.Equal(observed.Transcript, baseline.Transcript) || !bytes.Equal(observed.ChoiceTrace.Bytes, baseline.ChoiceTrace.Bytes) || !bytes.Equal(observed.DiagnosticTrace.Bytes, baseline.DiagnosticTrace.Bytes) || observed.ChoiceTrace.Summary != baseline.ChoiceTrace.Summary || observed.runtimeTerminal != baseline.runtimeTerminal {
			t.Fatalf("seed %d fresh instance %d diverged: stdout=%x/%x wasi=%x/%x choices=%x/%x diagnostics=%x/%x", request.Runtime.Seed, instance, sha256.Sum256(observed.Stdout), sha256.Sum256(baseline.Stdout), sha256.Sum256(observed.Transcript), sha256.Sum256(baseline.Transcript), observed.ChoiceTrace.SHA256, baseline.ChoiceTrace.SHA256, observed.DiagnosticTrace.SHA256, baseline.DiagnosticTrace.SHA256)
		}
		if difference, err := choice.DiffDiagnostics(baseline.DiagnosticTrace.Bytes, observed.DiagnosticTrace.Bytes); err != nil || difference != nil {
			t.Fatalf("seed %d instance %d diagnostic divergence: %+v %v", request.Runtime.Seed, instance, difference, err)
		}
	}
	tape, err := choice.ProjectReplayPlan(baseline.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	setChoiceReplay(t, request.Runtime.Choice, choice.ModeReplay, baseline.ChoiceTrace.Bytes, tape)
	request.Runtime.Diagnostics = false
	exact, err := matchedPrepareAdmitted(runtime.GOOS, "/proc/meminfo", os.Getenv("GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD"), func() (Result, error) {
		return Run(t.Context(), request)
	})
	if err != nil || exact.Termination != "exit" || exact.ExitCode == nil || *exact.ExitCode != *baseline.ExitCode || !exact.Reaped || !bytes.Equal(exact.Stdout, baseline.Stdout) || !bytes.Equal(exact.Stderr, baseline.Stderr) || !bytes.Equal(exact.Transcript, baseline.Transcript) || !bytes.Equal(exact.ChoiceTrace.Bytes, baseline.ChoiceTrace.Bytes) {
		t.Fatalf("seed %d forced full-tape replay: %s %s err=%v stderr=%s", request.Runtime.Seed, exact.Termination, exact.Message, err, exact.Stderr)
	}
	report := runtimeQualification{SeedIndex: index, Seed: request.Runtime.Seed, Instances: qualificationInstances, ExactReplays: 1, Module: request.ModuleSHA256, Helper: request.HelperSHA256, Runtime: request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, Model: ImplementationSHA256(), ChoiceSHA256: fmt.Sprintf("%x", baseline.ChoiceTrace.SHA256), DiagnosticSHA256: fmt.Sprintf("%x", baseline.DiagnosticTrace.SHA256), StdoutSHA256: fmt.Sprintf("%x", sha256.Sum256(baseline.Stdout)), StderrSHA256: fmt.Sprintf("%x", sha256.Sum256(baseline.Stderr)), WASISHA256: fmt.Sprintf("%x", sha256.Sum256(baseline.Transcript)), ExitCode: *baseline.ExitCode, Summary: baseline.ChoiceTrace.Summary, LastDiagnostic: baseline.DiagnosticTrace.Records[len(baseline.DiagnosticTrace.Records)-1]}
	version, err := exec.Command(os.Getenv("GOMAD_STOCK_GO"), "version").Output()
	if err != nil || !strings.Contains(string(version), "go1.27.1 ") {
		t.Fatalf("qualification compiler version: %q %v", version, err)
	}
	report.SourceSHA256 = sourceSHA256
	report.RuntimeImplementationSHA256 = toolchain.ImplementationSHA256()
	report.CompilerSHA256 = fileHash(t, os.Getenv("GOMAD_STOCK_GO"))
	report.GoVersion = strings.TrimSpace(string(version))
	report.HostGOOS, report.HostGOARCH = runtime.GOOS, runtime.GOARCH
	report.Engine, report.Clock, report.Limits = request.Engine, request.Config.Clock, request.Config.Limits
	report.Collector, report.Profile = "nogreenteagc", request.Config.Profile
	report.TerminalSHA256 = fmt.Sprintf("%x", sha256.Sum256(baseline.runtimeTerminal[:]))
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, fmt.Sprintf("seed-%02d", index))
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	for name, payload := range map[string][]byte{"report.json": data, "stdout": baseline.Stdout, "stderr": baseline.Stderr, "choices.bin": baseline.ChoiceTrace.Bytes, "diagnostics.bin": baseline.DiagnosticTrace.Bytes, "wasi.bin": baseline.Transcript, "finish.bin": baseline.runtimeTerminal[:]} {
		if err := os.WriteFile(filepath.Join(directory, name), payload, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("qualification %s", data)
}

func TestCooperativeRuntimeQualificationAggregate(t *testing.T) {
	root := os.Getenv("GOMAD_WASM_QUALIFICATION_ROOT")
	if root == "" {
		t.Skip("run qualify-runtime for the retained aggregate gate")
	}
	if !filepath.IsAbs(root) {
		t.Fatal("qualification requires an absolute retained evidence root")
	}
	sourceSHA256 := qualificationSourceSHA256(t)
	choices, outputs, outcomes := map[string]bool{}, map[string]bool{}, map[uint32]bool{}
	domains := map[string]map[string]bool{"map ": {}, "runq ": {}, "timer callbacks ": {}, "canary ": {}}
	var first runtimeQualification
	for index := range 32 {
		data, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("seed-%02d/report.json", index)))
		if err != nil {
			t.Fatal(err)
		}
		var report runtimeQualification
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}
		if report.SeedIndex != uint64(index) || report.Seed != uint64(index)*1000003 || report.Instances != qualificationInstances || report.ExactReplays != 1 || report.Model != ImplementationSHA256() || report.SourceSHA256 != sourceSHA256 || report.RuntimeImplementationSHA256 != toolchain.ImplementationSHA256() || report.Helper != fileHash(t, helperPath(t)) || report.CompilerSHA256 != fileHash(t, os.Getenv("GOMAD_STOCK_GO")) || report.Engine != PinnedEngine() || report.HostGOOS != runtime.GOOS || report.HostGOARCH != runtime.GOARCH || report.Profile != CooperativeProfile || report.Collector != "nogreenteagc" || !strings.Contains(report.GoVersion, "go1.27.1 ") {
			t.Fatalf("missing or stale qualification shard %d", index)
		}
		if index == 0 {
			first = report
		} else if report.Module != first.Module || report.Helper != first.Helper || report.Runtime != first.Runtime || report.GoVersion != first.GoVersion || report.Clock != first.Clock || report.Limits != first.Limits {
			t.Fatal("qualification shards are not one frozen candidate")
		}
		choices[report.ChoiceSHA256], outputs[report.StdoutSHA256], outcomes[report.ExitCode] = true, true, true
		payloads := map[string][]byte{}
		for name, expected := range map[string]string{"stdout": report.StdoutSHA256, "stderr": report.StderrSHA256, "choices.bin": report.ChoiceSHA256, "diagnostics.bin": report.DiagnosticSHA256, "wasi.bin": report.WASISHA256, "finish.bin": report.TerminalSHA256} {
			payload, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("seed-%02d", index), name))
			if err != nil || fmt.Sprintf("%x", sha256.Sum256(payload)) != expected {
				t.Fatalf("qualification shard %d payload %s changed: %v", index, name, err)
			}
			payloads[name] = payload
		}
		if err := validateQualificationPayloads(report, payloads); err != nil {
			t.Fatalf("qualification shard %d payload malformed: %v", index, err)
		}
		for prefix, observations := range domains {
			found := false
			for _, line := range strings.Split(string(payloads["stdout"]), "\n") {
				if strings.HasPrefix(line, prefix) {
					observations[line] = true
					found = true
				}
			}
			if !found {
				t.Fatalf("qualification shard %d lacks %s", index, prefix)
			}
		}
	}
	if len(choices) < 2 || len(outputs) < 2 || !outcomes[0] || !outcomes[17] {
		t.Fatal("representative seeds did not reach distinct choices, outputs and canary outcomes")
	}
	for name, observations := range domains {
		if len(observations) < 2 {
			t.Fatalf("representative seeds did not vary the %s domain", name)
		}
	}
	t.Logf("32 seeds x %d fresh instances: %d complete record observations and 32 fresh forced exact replays; choices=%d outputs=%d outcomes=%d module=%s helper=%s runtime=%s model=%s", qualificationInstances, 32*qualificationInstances, len(choices), len(outputs), len(outcomes), first.Module, first.Helper, first.Runtime, first.Model)
}

func qualificationSourceSHA256(t *testing.T) string {
	t.Helper()
	manifest := os.Getenv("GOMAD_WASM_QUALIFICATION_MANIFEST")
	if !filepath.IsAbs(manifest) {
		t.Fatal("qualification requires an absolute frozen source manifest")
	}
	data, err := os.ReadFile(manifest)
	if err != nil || len(data) == 0 {
		t.Fatalf("read frozen source manifest: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		digest, path, found := strings.Cut(line, "  ")
		if !found || !filepath.IsAbs(path) || digest != fileHash(t, path) {
			t.Fatalf("qualification source changed: %q", line)
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func TestQualificationPayloadPreservesTerminalPeak(t *testing.T) {
	a, b := sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b"))
	decision, err := choice.CanonicalDecision(0, choice.KindRunnable, 20, false, [][32]byte{a, b}, a, 0)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := choice.BuildTrace([]choice.Record{decision.Record()}, choice.TerminalComplete)
	if err != nil {
		t.Fatal(err)
	}
	diagnostic, err := choice.BuildDiagnosticTrace([]choice.DiagnosticRecord{{Ordinal: 0}}, 8192)
	if err != nil {
		t.Fatal(err)
	}
	trace.Summary.PeakGoroutines = 5
	finish := make([]byte, 16)
	binary.BigEndian.PutUint32(finish, 1)
	binary.BigEndian.PutUint32(finish[4:], 5)
	report := runtimeQualification{Summary: trace.Summary, LastDiagnostic: diagnostic.Records[0]}
	payloads := map[string][]byte{"choices.bin": trace.Bytes, "diagnostics.bin": diagnostic.Bytes, "finish.bin": finish}
	if err := validateQualificationPayloads(report, payloads); err != nil {
		t.Fatalf("valid terminal peak rejected: %v", err)
	}
	binary.BigEndian.PutUint32(finish[4:], 4)
	if err := validateQualificationPayloads(report, payloads); err == nil {
		t.Fatal("altered retained terminal peak accepted")
	}
}

func validateQualificationPayloads(report runtimeQualification, payloads map[string][]byte) error {
	trace, err := choice.DecodeStoredTrace(choice.Profile, payloads["choices.bin"], choice.TerminalMetadata{State: report.Summary.Terminal, Limit: 1 << 20, Records: report.Summary.Records, SHA256: sha256.Sum256(payloads["choices.bin"])})
	if err != nil {
		return err
	}
	finish := payloads["finish.bin"]
	if len(finish) != 16 || binary.BigEndian.Uint32(finish) != 1 || binary.BigEndian.Uint32(finish[4:]) == 0 {
		return fmt.Errorf("terminal frame malformed")
	}
	trace.Summary.PeakGoroutines = binary.BigEndian.Uint32(finish[4:])
	if err != nil || len(trace.Records) == 0 || trace.Summary != report.Summary || trace.Summary.Terminal != choice.TerminalComplete {
		return fmt.Errorf("choices malformed: %v", err)
	}
	diagnostic, err := choice.DecodeDiagnosticTrace(payloads["diagnostics.bin"])
	if err != nil || len(diagnostic.Records) == 0 || len(diagnostic.Records) != len(trace.Records) || diagnostic.Records[len(diagnostic.Records)-1] != report.LastDiagnostic {
		return fmt.Errorf("diagnostics malformed: %v", err)
	}
	return nil
}

func TestStockGoGuestRepeatsIn100FreshInstances(t *testing.T) {
	helper := helperPath(t)
	goCommand := os.Getenv("GOMAD_STOCK_GO")
	if goCommand == "" {
		goCommand = "go"
	}
	compiler, err := exec.Command(goCommand, "version").Output()
	if err != nil || !strings.Contains(string(compiler), "go1.27.1 ") {
		t.Fatalf("stock compiler identity: %q, %v", compiler, err)
	}
	module := filepath.Join(t.TempDir(), "environment.wasm")
	command := exec.Command(goCommand, "build", "-trimpath", "-o", module, "../testdata/environment")
	command.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOEXPERIMENT=nogreenteagc", "GOTOOLCHAIN=local")
	if data, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build real guest: %v\n%s", err, data)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	config := testConfig()
	config.Clock.ReadStepNanos = 1000000
	environment := newTestEnvironment(t, config)
	request := Request{HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config, Timeout: 60 * time.Second}
	var stdout, transcript []byte
	for i := 0; i < 100; i++ {
		result, err := Run(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || !result.Reaped || result.EnvironmentIdentity != environment.Identity() {
			t.Fatalf("guest %d: %s: %s reaped=%v\nstdout=%s stderr=%s transcript bytes=%d hash=%x", i, result.Termination, result.Message, result.Reaped, string(result.Stdout), string(result.Stderr), len(result.Transcript), sha256.Sum256(result.Transcript))
		}
		if i == 0 {
			stdout = result.Stdout
			transcript = result.Transcript
			for _, marker := range []string{"map ", "select ", "random ", "timer ", "fakeTCP echo", "cwd /workspace env captured arg literal $arg", "file scratch"} {
				if !strings.Contains(string(stdout), marker) {
					t.Fatalf("missing guest probe %q: %s", marker, stdout)
				}
			}
		} else if !bytes.Equal(stdout, result.Stdout) || !bytes.Equal(transcript, result.Transcript) {
			t.Fatalf("fresh guest %d diverged: stdout %q != %q; transcript %x != %x", i, result.Stdout, stdout, sha256.Sum256(result.Transcript), sha256.Sum256(transcript))
		}
	}
	t.Logf("100 fresh guests, 100 output and transcript equalities: compiler=%s experiment=nogreenteagc environment=%s module=%s helper=%s stdout=%x transcript=%x imports=%d memory-pages=%d", strings.TrimSpace(string(compiler)), environment.Identity(), request.ModuleSHA256, request.HelperSHA256, sha256.Sum256(stdout), sha256.Sum256(transcript), len(imports), pages)
}

func TestBrokerCrashRecoveryFreshGuest(t *testing.T) {
	base := cooperativeRequest(t, "../cluster/testdata/service")
	for _, readStep := range []uint64{0, 1000} {
		t.Run(fmt.Sprintf("read-step-%d", readStep), func(t *testing.T) { brokerCrashRecoveryParity(t, base, readStep) })
	}
}

// brokerCrashRecoveryParity runs the storage crash/recover broker fixture
// under the Go-owned path and under split ownership (fn-176.6). Split keeps
// clock reads and polls in the helper, so the choice traces and transcript
// digests, including clock-run folding, must equal the Go-owned path's.
func brokerCrashRecoveryParity(t *testing.T, request Request, readStep uint64) {
	request.Config.Clock.ReadStepNanos = readStep
	request.Config.Storage = &StorageModel{Volumes: []volume.VolumeConfig{{ID: "a-store", Path: "/workspace", CapacityBytes: 8 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1024, Transitions: 10000}}
	run := func(remoteModel bool) ([]Result, world.Snapshot) {
		b, err := NewBroker(brokerTestConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err = b.Session("a").CrashOnOutput([]byte("committed durable-a")); err != nil {
			t.Fatal(err)
		}
		peer := request
		peer.Config = request.Config
		peer.Config.Storage = nil
		peer.Config.Args = []string{"service", "park"}
		peer.Broker = b.Session("b")
		peerDone := make(chan Result, 1)
		peerErr := make(chan error, 1)
		go func() { result, err := Run(t.Context(), peer); peerDone <- result; peerErr <- err }()
		first := request
		first.Config = request.Config
		first.Config.Args = []string{"service", "durable"}
		first.Broker = b.Session("a")
		first.remoteModel = remoteModel
		crashed, err := Run(t.Context(), first)
		if err != nil || crashed.Termination != "crash" || crashed.StorageImage == nil || !crashed.Reaped {
			t.Fatalf("crash: %s %s err=%v stdout=%s stderr=%s", crashed.Termination, crashed.Message, err, crashed.Stdout, crashed.Stderr)
		}
		session, err := b.Restart("a")
		if err != nil {
			t.Fatal(err)
		}
		restart := request
		restart.Config = request.Config
		restart.Config.Args = []string{"service", "recover"}
		restart.Broker = session
		restart.StorageImage = crashed.StorageImage
		restart.remoteModel = remoteModel
		recovered, err := Run(t.Context(), restart)
		if err != nil || recovered.Termination != "exit" || !recovered.Reaped || !bytes.Contains(recovered.Stdout, []byte("fresh heap 1\nrecovered durable-a")) {
			t.Fatalf("recover: %s %s %v stdout=%s stderr=%s", recovered.Termination, recovered.Message, err, recovered.Stdout, recovered.Stderr)
		}
		other := <-peerDone
		if err = <-peerErr; err != nil || other.Termination != "exit" || !other.Reaped {
			t.Fatalf("peer: %s %s %v", other.Termination, other.Message, err)
		}
		return []Result{crashed, recovered, other}, b.Snapshot()
	}
	left, leftOrder := run(true)
	right, rightOrder := run(false)
	for i := range left {
		if !bytes.Equal(left[i].Stdout, right[i].Stdout) || !bytes.Equal(left[i].Transcript, right[i].Transcript) || !bytes.Equal(left[i].StorageTranscript, right[i].StorageTranscript) {
			t.Fatalf("crash/recover node incarnation %d differs: %s %q / %s %q stdout=%q/%q", i, left[i].Termination, left[i].Message, right[i].Termination, right[i].Message, left[i].Stdout, right[i].Stdout)
		}
		diff := ""
		if i < 2 {
			diff = sameStorageEvidence(left[i], right[i])
		}
		if diff != "" || !bytes.Equal(left[i].ChoiceTrace.Bytes, right[i].ChoiceTrace.Bytes) || left[i].TranscriptSHA256 != right[i].TranscriptSHA256 {
			t.Fatalf("Go-owned vs split incarnation %d differs in %s", i, diff)
		}
	}
	for i, split := range right[:2] {
		if left[i].GoFilesystemCalls["path_open"] == 0 {
			t.Fatalf("Go-owned incarnation %d did not serve its storage: %v", i, left[i].GoFilesystemCalls)
		}
		requireSplitServedOnlyOutput(t, fmt.Sprintf("split incarnation %d", i), split)
	}
	if !reflect.DeepEqual(leftOrder, rightOrder) {
		t.Fatal("crash recovery global event order differs")
	}
}

func TestClusterOutputBoundaryParity(t *testing.T) {
	base := cooperativeRequest(t, "../cluster/testdata/service")
	base.Config.Clock.ReadStepNanos = 0
	for _, test := range []struct{ fixture, kind string }{
		{"boundary-stderr", "crash"}, {"boundary-stderr", "stop"},
		{"boundary-split", "crash"}, {"boundary-split", "stop"},
	} {
		t.Run(test.fixture+"/"+test.kind, func(t *testing.T) {
			fixture := test.fixture
			request := base
			control, choices := *base.Runtime, *base.Runtime.Choice
			control.Choice, request.Runtime = &choices, &control
			request.Config.Args = []string{"service", fixture}
			configuration := brokerTestConfig()
			configuration.Nodes = configuration.Nodes[:1]
			broker, err := NewBroker(configuration)
			if err != nil {
				t.Fatal(err)
			}
			marker := []byte("output boundary")
			boundary := broker.Session("a").CrashOnOutput
			if test.kind == "stop" {
				boundary = broker.Session("a").StopOnOutput
			}
			if err := boundary(marker); err != nil {
				t.Fatal(err)
			}
			referenceRequest := request
			referenceRequest.Broker = broker.Session("a")
			reference, err := Run(t.Context(), referenceRequest)
			if err != nil {
				t.Fatal(err)
			}
			if fixture == "boundary-stderr" {
				requireExit(t, "stderr-only reference", reference, 0)
			} else if reference.Termination != test.kind || !reference.RuntimeInterrupted {
				t.Fatalf("split stdout marker did not revoke the Go guest: %s %s", reference.Termination, reference.Message)
			}
			wantWorld, err := broker.EncodeWorldEvidence()
			if err != nil {
				t.Fatal(err)
			}
			for _, replay := range []bool{false, true} {
				if replay {
					var tape choice.ReplayPlan
					if reference.RuntimeInterrupted {
						tape, err = choice.ProjectInterruptedReplayPlan(reference.InterruptedChoiceBytes, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
						request.Runtime.Choice.Mode = choice.ModePrefix
					} else {
						tape, err = choice.ProjectReplayPlan(reference.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
						request.Runtime.Choice.Mode = choice.ModeReplay
					}
					if err != nil {
						t.Fatal(err)
					}
					source := reference.ChoiceTrace.Bytes
					if reference.RuntimeInterrupted {
						source = reference.InterruptedChoiceBytes
					}
					setChoiceReplay(t, request.Runtime.Choice, request.Runtime.Choice.Mode, source, tape)
				}
				cluster := ClusterRequest{Broker: configuration, Instances: []ClusterInstance{{ID: "a", Request: request}}}
				boundaries := []ClusterBoundary{{Node: "a", Incarnation: 1, Marker: marker}}
				if test.kind == "stop" {
					cluster.OutputStops = boundaries
				} else {
					cluster.OutputCrashes = boundaries
				}
				session, err := OpenCluster(t.Context(), cluster)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					_ = session.Shutdown(cleanup)
				})
				if err := session.Start(t.Context(), "a", 1, "a", request.Runtime); err != nil {
					t.Fatal(err)
				}
				actual, err := session.Wait(t.Context(), "a", 1)
				if err != nil {
					t.Fatal(err)
				}
				if diff := sameEvidence(reference, actual); diff != "" || reference.RuntimeInterrupted != actual.RuntimeInterrupted || !bytes.Equal(reference.InterruptedChoiceBytes, actual.InterruptedChoiceBytes) {
					t.Fatalf("replay=%t output boundary differs in %s: %s %s", replay, diff, actual.Termination, actual.Message)
				}
				// fn-178.8 revocation-after-reply analogue without a durable write:
				// the revoking stdout call is the instance's last effect.
				if bytes.Contains(actual.Stdout, []byte("continuation executed")) || bytes.Contains(actual.Stderr, []byte("continuation executed")) {
					t.Fatalf("replay=%t revoked guest performed a further effect", replay)
				}
				finished, err := session.Finish(t.Context())
				if err != nil || !bytes.Equal(wantWorld, finished.World) || finished.Frames.GuestCallbacks != 0 {
					t.Fatalf("replay=%t output boundary World/callback parity: %v", replay, err)
				}
			}
		})
	}
}

func TestClusterBrokerDurableRestartParity(t *testing.T) {
	base := cooperativeRequest(t, "../cluster/testdata/service")
	base.Config.Clock.ReadStepNanos = 0
	base.Config.Storage = &StorageModel{Volumes: []volume.VolumeConfig{{ID: "a-store", Path: "/workspace", CapacityBytes: 8 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1024, Transitions: 10000}}
	for _, fixture := range []string{"crash", "stop", "open-streams", "storage-crash"} {
		for _, seed := range []uint64{7, 19, 42} {
			action := fixture
			if fixture == "open-streams" {
				action = "crash"
			}
			if fixture == "storage-crash" {
				action = "storage_crash"
			}
			t.Run(fmt.Sprintf("%s/seed-%d", fixture, seed), func(t *testing.T) {
				configuration := brokerTestConfig()
				configuration.Seed = seed
				if fixture == "open-streams" || fixture == "storage-crash" {
					configuration.Network = &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 10000}}
					configuration.Addresses = map[string]string{"a": "10.0.0.1", "b": "10.0.0.2"}
				}
				broker, err := NewBroker(configuration)
				if err != nil {
					t.Fatal(err)
				}
				marker := []byte("committed durable-a")
				if fixture == "open-streams" {
					marker = []byte("open stream ready")
					err = broker.CrashAfterOutputs([]string{"a", "b"}, marker)
				} else if action == "crash" {
					err = broker.Session("a").CrashOnOutput(marker)
				} else if action == "stop" {
					err = broker.Session("a").StopOnOutput(marker)
				}
				if err != nil {
					t.Fatal(err)
				}
				requests := make([]Request, 3)
				for i := range requests {
					requests[i] = base
					control, choices := *base.Runtime, *base.Runtime.Choice
					control.Choice = &choices
					control.Seed = seed
					requests[i].Runtime = &control
					requests[i].Config.NetworkV2 = configuration.Network
				}
				requests[0].Config.Args = []string{"service", "durable"}
				requests[1].Config.Args = []string{"service", "recover"}
				requests[2].Config.Args = []string{"service", "park"}
				requests[2].Config.Storage = nil
				if fixture == "storage-crash" {
					model := *base.Config.Storage
					model.Faults = []StorageFault{{Name: "open-stream-storage-crash", Resource: "/workspace/crash-trigger", Operation: "open", Occurrence: 1, Action: "crash-before"}}
					requests[0].Config.Storage, requests[1].Config.Storage = &model, &model
					requests[0].Config.Args = []string{"service", "server-storage-crash"}
					requests[0].StorageCrashLimits = &volume.CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}
					requests[2].Config.Args = []string{"service", "client", "10.0.0.1"}
				}
				if fixture == "open-streams" {
					requests[0].Config.Args = []string{"service", "server-open"}
					requests[2].Config.Args = []string{"service", "client-open", "10.0.0.1"}
				}
				peer := requests[2]
				peer.Broker = broker.Session("b")
				peerDone := make(chan Result, 1)
				peerError := make(chan error, 1)
				go func() { result, err := Run(t.Context(), peer); peerDone <- result; peerError <- err }()
				first := requests[0]
				first.Broker = broker.Session("a")
				crashed, err := Run(t.Context(), first)
				if err != nil || crashed.Termination != action || !crashed.RuntimeInterrupted || crashed.StorageImage == nil {
					t.Fatalf("Go %s: %s %s %v stdout=%s stderr=%s storage=%s", action, crashed.Termination, crashed.Message, err, crashed.Stdout, crashed.Stderr, crashed.StorageTranscript)
				}
				if fixture == "storage-crash" {
					if len(crashed.StorageCrashContext) == 0 {
						t.Fatal("Go storage crash omitted prior pending context")
					}
					requests[0].StorageCrashContext = crashed.StorageCrashContext
				}
				var other Result
				if fixture == "open-streams" || fixture == "storage-crash" {
					other = <-peerDone
					if err := <-peerError; err != nil {
						t.Fatal(err)
					}
				}
				restart, err := broker.Restart("a")
				if err != nil {
					t.Fatal(err)
				}
				recoveredRequest := requests[1]
				recoveredRequest.Broker = restart
				recoveredRequest.StorageImage = crashed.StorageImage
				recovered, err := Run(t.Context(), recoveredRequest)
				if err != nil {
					t.Fatal(err)
				}
				requireExit(t, "Go recovered", recovered, 0)
				if fixture != "open-streams" && fixture != "storage-crash" {
					other = <-peerDone
					if err := <-peerError; err != nil {
						t.Fatal(err)
					}
				}
				if fixture == "open-streams" {
					if other.Termination != "crash" || !other.RuntimeInterrupted {
						t.Fatalf("Go peer did not crash with an open stream: %s %s", other.Termination, other.Message)
					}
				} else {
					requireExit(t, "Go peer", other, 0)
				}
				reference := []Result{crashed, recovered, other}
				wantWorld, err := broker.EncodeWorldEvidence()
				if err != nil {
					t.Fatal(err)
				}
				var wantNetwork []byte
				if configuration.Network != nil {
					record, err := broker.NetworkRecord()
					if err != nil {
						t.Fatal(err)
					}
					wantNetwork, err = json.Marshal(record)
					if err != nil {
						t.Fatal(err)
					}
				}
				for i, result := range reference {
					t.Logf("Go golden %s incarnation[%d]: termination=%s wasi=%x storage=%x choices=%x", action, i, result.Termination, result.TranscriptSHA256, sha256.Sum256(result.StorageTranscript), sha256.Sum256(append(bytes.Clone(result.ChoiceTrace.Bytes), result.InterruptedChoiceBytes...)))
				}
				for _, replay := range []bool{false, true} {
					if replay {
						for i := range requests {
							var tape choice.ReplayPlan
							var err error
							mode := choice.ModeReplay
							if reference[i].RuntimeInterrupted {
								mode = choice.ModePrefix
								tape, err = choice.ProjectInterruptedReplayPlan(reference[i].InterruptedChoiceBytes, requests[i].Runtime.Choice.Limit, requests[i].Runtime.Choice.ExecutionIdentity)
							} else {
								tape, err = choice.ProjectReplayPlan(reference[i].ChoiceTrace, requests[i].Runtime.Choice.ExecutionIdentity)
							}
							if err != nil {
								t.Fatal(err)
							}
							source := reference[i].ChoiceTrace.Bytes
							if reference[i].RuntimeInterrupted {
								source = reference[i].InterruptedChoiceBytes
							}
							setChoiceReplay(t, requests[i].Runtime.Choice, mode, source, tape)
						}
					}
					cluster := ClusterRequest{Broker: configuration, Instances: []ClusterInstance{{ID: "a", Request: requests[0]}, {ID: "b", Request: requests[2]}}, Boots: map[string]Request{"recover": requests[1]}}
					boundary := ClusterBoundary{Node: "a", Incarnation: 1, Marker: marker}
					if fixture == "open-streams" {
						cluster.OutputCrashGroup = &ClusterCrashGroup{Nodes: []string{"a", "b"}, Marker: marker}
					} else if action == "crash" {
						cluster.OutputCrashes = []ClusterBoundary{boundary}
					} else if action == "stop" {
						cluster.OutputStops = []ClusterBoundary{boundary}
					}
					session, err := OpenCluster(t.Context(), cluster)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()
						_ = session.Shutdown(ctx)
					})
					for _, start := range []struct {
						node  string
						index int
					}{{"a", 0}, {"b", 2}} {
						if err := session.Start(t.Context(), start.node, 1, start.node, requests[start.index].Runtime); err != nil {
							t.Fatal(err)
						}
					}
					actual := make([]Result, 3)
					actual[0], err = session.Wait(t.Context(), "a", 1)
					if err != nil {
						t.Fatal(err)
					}
					if fixture == "open-streams" || fixture == "storage-crash" {
						actual[2], err = session.Wait(t.Context(), "b", 1)
						if err != nil {
							t.Fatal(err)
						}
					}
					if err := session.Start(t.Context(), "a", 2, "recover", requests[1].Runtime); err != nil {
						t.Fatal(err)
					}
					actual[1], err = session.Wait(t.Context(), "a", 2)
					if err != nil {
						t.Fatal(err)
					}
					if fixture != "open-streams" && fixture != "storage-crash" {
						actual[2], err = session.Wait(t.Context(), "b", 1)
						if err != nil {
							t.Fatal(err)
						}
					}
					for i := range actual {
						if diff := sameEvidence(reference[i], actual[i]); diff != "" {
							if diff == "network" {
								t.Logf("Go reference network=%s\nRust candidate network=%s", reference[i].NetworkTranscript, actual[i].NetworkTranscript)
								t.Logf("Go reference peer network=%s\nRust candidate peer network=%s", reference[2].NetworkTranscript, actual[2].NetworkTranscript)
								finished, finishErr := session.Finish(t.Context())
								t.Logf("terminal World equality=%t Go=%x Rust=%x finish=%v; Go final network=%s; Rust final network=%+v", bytes.Equal(wantWorld, finished.World), sha256.Sum256(wantWorld), sha256.Sum256(finished.World), finishErr, wantNetwork, finished.Network)
							}
							t.Fatalf("%s replay=%t incarnation[%d] differs in %s: %s %s", action, replay, i, diff, actual[i].Termination, actual[i].Message)
						}
						if reference[i].RuntimeInterrupted != actual[i].RuntimeInterrupted || !bytes.Equal(reference[i].InterruptedChoiceBytes, actual[i].InterruptedChoiceBytes) {
							t.Fatalf("%s replay=%t incarnation[%d] interrupted choices differ", fixture, replay, i)
						}
						// fn-178.8 revocation-after-reply analogue with an intervening
						// durable write: the crash/stop decided by the marker write lands
						// before the guest's next call (sleep, then panic).
						if i == 0 && (fixture == "crash" || fixture == "stop") && (!bytes.HasSuffix(actual[i].Stdout, marker) && !bytes.HasSuffix(actual[i].Stdout, append(bytes.Clone(marker), '\n')) || bytes.Contains(actual[i].Stderr, []byte("continuation executed"))) {
							t.Fatalf("%s replay=%t revoked guest continued after its durable write: stdout=%q", fixture, replay, actual[i].Stdout)
						}
						if i < 2 {
							if diff := sameStorageEvidence(reference[i], actual[i]); diff != "" {
								t.Fatalf("%s replay=%t incarnation[%d] storage differs in %s; messages %q / %q", action, replay, i, diff, reference[i].Message, actual[i].Message)
							}
						}
						if actual[i].StorageImage != nil && len(actual[i].StorageImage.data) != 0 {
							t.Fatal("cluster recovery image crossed to Go")
						}
					}
					finished, err := session.Finish(t.Context())
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(wantWorld, finished.World) {
						t.Fatalf("%s replay=%t World differs", action, replay)
					}
					if configuration.Network != nil {
						actualNetwork, err := json.Marshal(finished.Network)
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(wantNetwork, actualNetwork) {
							t.Fatalf("%s replay=%t final network Record differs", fixture, replay)
						}
					}
				}
				if fixture == "storage-crash" {
					// The sealed cluster boot carries the prior pending context to
					// Rust, which refuses a forged inventory like the Go owner does.
					forged := requests[0]
					prior := forged.StorageCrashContext["a-store"]
					prior.PendingSHA256 = "sha256:" + strings.Repeat("0", 64)
					forged.StorageCrashContext = map[string]StorageCrashContext{"a-store": prior}
					session, err := OpenCluster(t.Context(), ClusterRequest{Broker: configuration, Instances: []ClusterInstance{{ID: "a", Request: forged}, {ID: "b", Request: requests[2]}}})
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()
						_ = session.Shutdown(ctx)
					})
					for _, start := range []struct {
						node  string
						index int
					}{{"a", 0}, {"b", 2}} {
						if err := session.Start(t.Context(), start.node, 1, start.node, requests[start.index].Runtime); err != nil {
							t.Fatal(err)
						}
					}
					diverged, err := session.Wait(t.Context(), "a", 1)
					if err != nil || diverged.Termination != "invalid" || diverged.Message != "storage crash context diverged from prior observation" || diverged.StorageImage != nil {
						t.Fatalf("forged cluster crash context: %s %q image=%v err=%v", diverged.Termination, diverged.Message, diverged.StorageImage, err)
					}
				}
			})
		}
	}
}

func TestClusterShutdownLiveNodesEvidenceOrder(t *testing.T) {
	request := cooperativeRequest(t, "../cluster/testdata/service")
	request.Config.Clock.ReadStepNanos = 0
	request.Config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 10000}}
	configuration := brokerTestConfig()
	configuration.Network = request.Config.NetworkV2
	var wantWorld, wantNetwork []byte
	for iteration := range 16 {
		session, err := OpenCluster(t.Context(), ClusterRequest{Broker: configuration, Instances: []ClusterInstance{{ID: "a", Request: request}, {ID: "b", Request: request}}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := session.Shutdown(ctx); err != nil {
				t.Error(err)
			}
		})
		for _, id := range []string{"a", "b"} {
			if err := session.Start(t.Context(), id, 1, id, request.Runtime); err != nil {
				t.Fatal(err)
			}
		}
		session.mu.Lock()
		states := session.states
		session.states = make(map[BrokerNode]*clusterState)
		ids := []string{"a", "b"}
		if iteration%2 != 0 {
			ids = []string{"b", "a"}
		}
		for _, id := range ids {
			handle := BrokerNode{ID: id, Incarnation: 1}
			select {
			case <-states[handle].done:
				session.mu.Unlock()
				t.Fatalf("node %s finished before shutdown", id)
			default:
			}
			session.states[handle] = states[handle]
		}
		session.mu.Unlock()
		if err := session.Shutdown(t.Context()); err != nil {
			t.Fatal(err)
		}
		result, err := session.Finish(t.Context())
		if err != nil || len(result.World) == 0 {
			t.Fatalf("shutdown did not retain World evidence: %v", err)
		}
		network, err := json.Marshal(result.Network)
		if err != nil {
			t.Fatal(err)
		}
		if iteration == 0 {
			wantWorld, wantNetwork = bytes.Clone(result.World), network
		} else if !bytes.Equal(result.World, wantWorld) || !bytes.Equal(network, wantNetwork) {
			t.Fatalf("iteration %d shutdown changed World or network evidence", iteration)
		}
	}
}

func TestBrokerIndependentGuestStreamsReplay(t *testing.T) {
	request := cooperativeRequest(t, "../cluster/testdata/service")
	baselines := map[string]map[uint64]Result{}
	var samples []map[string]any
	scheduleWitness := regexp.MustCompile(`(?m)^runnable order(?: [0-3]){4}$`)
	request.Config.Clock.ReadStepNanos = 0
	request.Config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 10000}}
	request.Config.Limits.Calls, request.Config.Limits.TranscriptBytes = 100000, 32<<20
	request.Runtime.Choice.Limit = 8 << 20
	for _, seed := range []uint64{7, 19, 42} {
		for _, fixture := range []string{"partition-heal-delay", "stall", "live-link"} {
			t.Run(fmt.Sprintf("seed-%d/%s", seed, fixture), func(t *testing.T) {
				configuration := brokerTestConfig()
				configuration.Seed, configuration.Network = seed, request.Config.NetworkV2
				configuration.Addresses = map[string]string{"a": "10.0.0.1", "b": "10.0.0.2"}
				configuration.Faults = []BrokerNetworkFault{{Name: "partition", Action: "partition", From: "a", To: "b", AtNanos: 1, Symmetric: true}, {Name: "heal", Action: "heal", From: "a", To: "b", AtNanos: 1_000_000_001, Symmetric: true}, {Name: "delay", Action: "delay", From: "a", To: "b", AtNanos: 1_000_000_002, DelayNanos: 1000, Symmetric: true}}
				if fixture == "stall" {
					configuration.Faults = configuration.Faults[:2]
					configuration.Faults[1].AtNanos = 20_000_000_001
				}
				requests := make([]Request, 2)
				for i := range requests {
					requests[i] = request
					control, choices := *request.Runtime, *request.Runtime.Choice
					control.Seed, control.Choice = seed, &choices
					requests[i].Runtime = &control
					requests[i].Config.Args = []string{"service", "server"}
					if i == 1 {
						requests[i].Config.Args = []string{"service", "client", "10.0.0.1"}
					}
					if fixture == "stall" {
						if i == 0 {
							requests[i].Config.Args = []string{"service", "server-stall"}
						} else {
							requests[i].Config.Args = []string{"service", "client-stall", "10.0.0.1"}
						}
					}
					requests[i].Config.Args = append(requests[i].Config.Args, "schedule")
				}
				runReference := func() ([]Result, []byte, []byte) {
					broker, err := NewBroker(configuration)
					if err != nil {
						t.Fatal(err)
					}
					if fixture == "live-link" {
						if err := broker.ChangeLink("delay", "a", "b", 1000, true); err != nil {
							t.Fatal(err)
						}
					}
					reference, failures := make([]Result, 2), make([]error, 2)
					var wait sync.WaitGroup
					for i, id := range []string{"a", "b"} {
						r := requests[i]
						r.Broker = broker.Session(id)
						wait.Go(func() { reference[i], failures[i] = Run(t.Context(), r) })
					}
					wait.Wait()
					for i, result := range reference {
						if failures[i] != nil {
							t.Fatal(failures[i])
						}
						requireExit(t, "Go broker", result, 0)
						witness := []byte("ordered bytes ordered-byte")
						if fixture == "stall" {
							witness = []byte("stall expired")
						}
						if !bytes.Contains(result.Stdout, []byte("fresh heap 1")) || !bytes.Contains(result.Stdout, witness) {
							t.Fatalf("node %d missing independent stream witness: %s", i, result.Stdout)
						}
					}
					worldBytes, err := broker.EncodeWorldEvidence()
					if err != nil {
						t.Fatal(err)
					}
					network, err := broker.NetworkRecord()
					if err != nil {
						t.Fatal(err)
					}
					wantNetwork, err := json.Marshal(network)
					if err != nil {
						t.Fatal(err)
					}
					return reference, worldBytes, wantNetwork
				}
				reference, worldBytes, wantNetwork := runReference()
				for _, replay := range []bool{false, true} {
					if replay {
						for i := range requests {
							tape, err := choice.ProjectReplayPlan(reference[i].ChoiceTrace, requests[i].Runtime.Choice.ExecutionIdentity)
							if err != nil {
								t.Fatal(err)
							}
							setChoiceReplay(t, requests[i].Runtime.Choice, choice.ModeReplay, reference[i].ChoiceTrace.Bytes, tape)
						}
						goReplay, goWorld, goNetwork := runReference()
						for i := range reference {
							if diff := sameEvidence(reference[i], goReplay[i]); diff != "" || !bytes.Equal(reference[i].Transcript, goReplay[i].Transcript) {
								t.Fatalf("Go reference replay node %d changed evidence: %s", i, diff)
							}
						}
						if !bytes.Equal(worldBytes, goWorld) || !bytes.Equal(wantNetwork, goNetwork) {
							t.Fatal("Go reference replay changed global event order or network Record")
						}
					}
					session, err := OpenCluster(t.Context(), ClusterRequest{Broker: configuration, Instances: []ClusterInstance{{ID: "a", Request: requests[0]}, {ID: "b", Request: requests[1]}}})
					if err != nil {
						t.Fatal(err)
					}
					if fixture == "live-link" {
						if err := session.Link(t.Context(), "delay", "a", "b", 1000, true); err != nil {
							t.Fatal(err)
						}
					}
					t.Cleanup(func() {
						cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()
						_ = session.Shutdown(cleanup)
					})
					for i, id := range []string{"a", "b"} {
						if err := session.Start(t.Context(), id, 1, id, requests[i].Runtime); err != nil {
							t.Fatal(err)
						}
					}
					for i, id := range []string{"a", "b"} {
						actual, err := session.Wait(t.Context(), id, 1)
						if err != nil {
							t.Fatal(err)
						}
						if mismatch := sameEvidence(reference[i], actual); mismatch != "" || !bytes.Equal(reference[i].Transcript, actual.Transcript) {
							t.Fatalf("node %s replay=%t differs in %s: %s %s", id, replay, mismatch, actual.Termination, actual.Message)
						}
						if i == 0 {
							if !replay {
								if baselines[fixture] == nil {
									baselines[fixture] = map[uint64]Result{}
								}
								baselines[fixture][seed] = actual
							}
							samples = append(samples, map[string]any{"seed": seed, "fixture": fixture, "replay": replay, "choice_sha256": fmt.Sprintf("%x", actual.ChoiceTrace.SHA256), "choice_records": actual.ChoiceTrace.Summary.Records, "runnable_order": string(scheduleWitness.Find(actual.Stdout))})
						}
					}
					actual, err := session.Finish(t.Context())
					if err != nil {
						t.Fatal(err)
					}
					actualNetwork, err := json.Marshal(actual.Network)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(worldBytes, actual.World) || !bytes.Equal(wantNetwork, actualNetwork) {
						t.Fatalf("cluster replay=%t changed World or shared network Record", replay)
					}
					if frames := session.FrameCounts(); frames.GuestCallbacks != 0 || frames.Results != 2 {
						t.Fatalf("cluster callback/result frames: %+v", frames)
					}
				}
			})
		}
	}
	distinct := map[string]int{}
	distinctOrders := map[string]int{}
	for fixture, results := range baselines {
		distinct[fixture] = distinctChoiceTraces(results)
		orders := map[string]bool{}
		for _, result := range results {
			witness := string(scheduleWitness.Find(result.Stdout))
			if witness == "" {
				t.Errorf("%s omitted its runnable-order witness", fixture)
			}
			orders[witness] = true
		}
		distinctOrders[fixture] = len(orders)
	}
	report := map[string]any{"network_owner": "rust-cluster", "network_owner_implementation_sha256": request.HelperSHA256, "reference_network_owner": "go-broker-reference", "reference_network_owner_implementation_sha256": netmodel.ImplementationSHA256(), "samples": samples, "distinct_choice_traces": distinct, "distinct_runnable_orders": distinctOrders}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("broker network gate report: %s", encoded)
	if root := os.Getenv("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT"); root != "" {
		if !filepath.IsAbs(root) {
			t.Fatal("GOMAD_WASM_TEMPORAL_RECOVERY_ROOT must be absolute")
		}
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		temporalJSON(t, filepath.Join(root, "broker-network-streams-report.json"), report)
	}
	for fixture, count := range distinct {
		if count < 2 {
			t.Errorf("%s explored %d distinct choice traces across three seeds", fixture, count)
		}
		if distinctOrders[fixture] < 2 {
			t.Errorf("%s realized %d distinct runnable orders across three seeds", fixture, distinctOrders[fixture])
		}
	}
}

func TestHelperModelPreservesGoOwnerEvidence(t *testing.T) {
	request := cooperativeRequest(t, "../toolchain/testdata/choices")
	requireSmallHelperParityModule(t, request)
	for _, digestOnly := range []bool{false, true} {
		request.DigestTranscript = digestOnly
		for _, frozen := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/digest-%t", map[bool]string{false: "read-step", true: "frozen"}[frozen], digestOnly), func(t *testing.T) {
				request.Config.Clock.ReadStepNanos = 1000
				if frozen {
					request.Config.Clock.ReadStepNanos = 0
				}
				remote := request
				remote.remoteModel = true
				reference, err := Run(t.Context(), remote)
				if err != nil {
					t.Fatal(err)
				}
				requireExit(t, "Go reference", reference, 0)
				local, err := Run(t.Context(), request)
				if err != nil {
					t.Fatal(err)
				}
				if local.GoFilesystemCalls == nil || len(local.GoFilesystemCalls) != 0 {
					t.Fatalf("helper namespace leaked to Go: %v", local.GoFilesystemCalls)
				}
				if diff := sameEvidence(reference, local); diff != "" || !bytes.Equal(reference.DiagnosticTrace.Bytes, local.DiagnosticTrace.Bytes) || reference.EnvironmentIdentity != local.EnvironmentIdentity || reference.Calls != local.Calls || reference.TimeAdvances != local.TimeAdvances {
					t.Fatalf("helper evidence differs in %s: %s %s diagnostics=%v calls=%d/%d advances=%d/%d", diff, local.Termination, local.Message, bytes.Equal(reference.DiagnosticTrace.Bytes, local.DiagnosticTrace.Bytes), reference.Calls, local.Calls, reference.TimeAdvances, local.TimeAdvances)
				}
				if local.Telemetry == nil || local.Telemetry.PrepareNanos == 0 || local.Telemetry.InstantiateNanos == 0 || local.Telemetry.ExecuteNanos == 0 || local.HelperResourceUsage == nil || reference.AuthorizationToTerminalNanos == 0 || local.AuthorizationToTerminalNanos == 0 {
					t.Fatalf("measured helper phase/resource telemetry missing: %+v %+v", local.Telemetry, local.HelperResourceUsage)
				}
				t.Logf("measured authorized execution Go-owner=%dns helper-owner=%dns (includes fresh instantiation and evidence flush)", reference.AuthorizationToTerminalNanos, local.AuthorizationToTerminalNanos)
				tape, err := choice.ProjectReplayPlan(reference.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
				if err != nil {
					t.Fatal(err)
				}
				for _, mode := range []choice.Mode{choice.ModeReplay, choice.ModePrefix} {
					forced := request
					control := *request.Runtime
					choices := *control.Choice
					setChoiceReplay(t, &choices, mode, reference.ChoiceTrace.Bytes, tape)
					control.Choice = &choices
					forced.Runtime = &control
					replayed, err := Run(context.Background(), forced)
					if err != nil {
						t.Fatal(err)
					}
					if diff := sameEvidence(reference, replayed); diff != "" {
						t.Fatalf("forced mode %d differs in %s: %s %s", mode, diff, replayed.Termination, replayed.Message)
					}
				}
			})
		}
	}
}

func TestReplayPrefixControlsPreserveGoOracle(t *testing.T) {
	base := cooperativeRequest(t, "../toolchain/testdata/choices")
	base.Config.Clock.ReadStepNanos = 0
	for _, surface := range []string{"standalone", "cluster"} {
		for _, variant := range []string{"short-prefix", "forced-rank"} {
			t.Run(surface+"/"+variant, func(t *testing.T) {
				count := 1
				if surface == "cluster" {
					count = 2
				}
				requests := make([]Request, count)
				for index := range requests {
					requests[index] = base
					control, choices := *base.Runtime, *base.Runtime.Choice
					control.Choice = &choices
					requests[index].Runtime = &control
				}
				configuration := brokerTestConfig()
				runReference := func() ([]Result, []byte) {
					var broker *Broker
					if surface == "cluster" {
						var err error
						broker, err = NewBroker(configuration)
						if err != nil {
							t.Fatal(err)
						}
					}
					results, failures := make([]Result, count), make([]error, count)
					var wait sync.WaitGroup
					for index, request := range requests {
						request.remoteModel = true
						if broker != nil {
							request.Broker = broker.Session([]string{"a", "b"}[index])
						}
						wait.Go(func() { results[index], failures[index] = Run(t.Context(), request) })
					}
					wait.Wait()
					for index, result := range results {
						if failures[index] != nil {
							t.Fatal(failures[index])
						}
						requireExit(t, "Go oracle", result, 0)
					}
					if broker == nil {
						return results, nil
					}
					world, err := broker.EncodeWorldEvidence()
					if err != nil {
						t.Fatal(err)
					}
					return results, world
				}
				source, _ := runReference()
				for index, result := range source {
					plan, err := choice.ProjectReplayPlan(result.ChoiceTrace, requests[index].Runtime.Choice.ExecutionIdentity)
					if err != nil || len(plan.Decisions) < 2 {
						t.Fatalf("oracle fixture lacks a shortened branching prefix: %v", err)
					}
					if variant == "forced-rank" {
						plan, err = choice.BuildRankPrefix(plan, 0, (plan.Decisions[0].Selected+1)%plan.Decisions[0].Alternatives)
					} else {
						plan, err = plan.Prefix(1)
					}
					if err != nil {
						t.Fatal(err)
					}
					setChoiceReplay(t, requests[index].Runtime.Choice, choice.ModePrefix, result.ChoiceTrace.Bytes, plan)
				}
				want, world := runReference()
				actual := make([]Result, count)
				if surface == "standalone" {
					var err error
					actual[0], err = Run(t.Context(), requests[0])
					if err != nil {
						t.Fatal(err)
					}
				} else {
					session, err := OpenCluster(t.Context(), ClusterRequest{Broker: configuration, Instances: []ClusterInstance{{ID: "a", Request: requests[0]}, {ID: "b", Request: requests[1]}}})
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()
						if err := session.Shutdown(ctx); err != nil {
							t.Error(err)
						}
					})
					for index, id := range []string{"a", "b"} {
						if err := session.Start(t.Context(), id, 1, id, requests[index].Runtime); err != nil {
							t.Fatal(err)
						}
					}
					for index, id := range []string{"a", "b"} {
						actual[index], err = session.Wait(t.Context(), id, 1)
						if err != nil {
							t.Fatal(err)
						}
					}
					finished, err := session.Finish(t.Context())
					if err != nil || !bytes.Equal(world, finished.World) {
						t.Fatalf("controlled cluster changed shared World evidence: %v", err)
					}
				}
				for index, result := range actual {
					requireExit(t, "helper", result, 0)
					if difference := sameEvidence(want[index], result); difference != "" || !bytes.Equal(want[index].Transcript, result.Transcript) || !bytes.Equal(want[index].DiagnosticTrace.Bytes, result.DiagnosticTrace.Bytes) {
						t.Fatalf("node %d controlled replay differs from Go oracle: %s", index, difference)
					}
				}
			})
		}
	}
}

func TestNamespaceOwnerFilesystemParity(t *testing.T) {
	request := cooperativeRequest(t, "./testdata/namespace_owner")
	requireSmallHelperParityModule(t, request)
	for _, digest := range []bool{false, true} {
		request.DigestTranscript = digest
		referenceRequest := request
		referenceRequest.remoteModel = true
		reference, err := Run(t.Context(), referenceRequest)
		if err != nil {
			t.Fatal(err)
		}
		requireExit(t, "Go namespace", reference, 0)
		if reference.GoFilesystemCalls["path_open"] == 0 || reference.GoFilesystemCalls["fd_pwrite"] == 0 || reference.GoFilesystemCalls["fd_readdir"] == 0 || reference.GoFilesystemCalls["path_rename"] == 0 {
			t.Fatal("filesystem parity fixture did not exercise required Go operations", reference.GoFilesystemCalls)
		}
		var mirror bytes.Buffer
		request.StdoutMirror = &mirror
		local, err := Run(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}
		requireExit(t, "helper namespace", local, 0)
		if local.GoFilesystemCalls == nil || len(local.GoFilesystemCalls) != 0 {
			t.Fatal("filesystem operation escaped helper owner", local.GoFilesystemCalls)
		}
		if difference := sameEvidence(reference, local); difference != "" || !bytes.Equal(reference.DiagnosticTrace.Bytes, local.DiagnosticTrace.Bytes) || reference.EnvironmentIdentity != local.EnvironmentIdentity || reference.Calls != local.Calls || reference.TimeAdvances != local.TimeAdvances {
			t.Fatal("filesystem owner changed evidence", difference)
		}
		if !bytes.Equal(mirror.Bytes(), local.Stdout) || !bytes.Contains(local.Stdout, []byte("namespace owner parity")) || string(local.Stderr) != "namespace stderr\n" {
			t.Fatal("namespace stdout/stderr/mirror changed")
		}
		plan, err := choice.ProjectReplayPlan(reference.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		control, choices := *request.Runtime, *request.Runtime.Choice
		setChoiceReplay(t, &choices, choice.ModeReplay, reference.ChoiceTrace.Bytes, plan)
		control.Choice = &choices
		forced := request
		forced.Runtime = &control
		forced.StdoutMirror = nil
		replayed, err := Run(t.Context(), forced)
		if err != nil {
			t.Fatal(err)
		}
		if difference := sameEvidence(reference, replayed); difference != "" || replayed.GoFilesystemCalls == nil || len(replayed.GoFilesystemCalls) != 0 {
			t.Fatal("forced namespace replay changed evidence", difference, replayed.GoFilesystemCalls)
		}
	}
}

// TestNamespaceOwnerSQLiteParity runs the file-backed SQLite guest without a
// storage model, so the helper owns its namespace, and requires the Go
// namespace reference to produce identical evidence.
func TestNamespaceOwnerSQLiteParity(t *testing.T) {
	base := storageSQLiteRequest(t)
	base.Config.Storage = nil
	for _, journal := range []string{"WAL", "DELETE"} {
		t.Run(journal, func(t *testing.T) {
			request := storageGuestOperation(base, "setup", journal, nil)
			reference := request
			reference.remoteModel = true
			expected, err := Run(t.Context(), reference)
			if err != nil {
				t.Fatal(err)
			}
			requireExit(t, "Go namespace SQLite", expected, 0)
			if string(expected.Stdout) != "ACK 1\n" || expected.GoFilesystemCalls["fd_pwrite"] == 0 || expected.GoFilesystemCalls["fd_pread"] == 0 {
				t.Fatalf("SQLite reference did not exercise positional I/O: %q %v", expected.Stdout, expected.GoFilesystemCalls)
			}
			local, err := Run(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			requireExit(t, "helper namespace SQLite", local, 0)
			if local.GoFilesystemCalls == nil || len(local.GoFilesystemCalls) != 0 {
				t.Fatal("SQLite filesystem operation escaped helper owner", local.GoFilesystemCalls)
			}
			if difference := sameEvidence(expected, local); difference != "" || expected.EnvironmentIdentity != local.EnvironmentIdentity || expected.Calls != local.Calls || expected.TimeAdvances != local.TimeAdvances {
				t.Fatalf("SQLite namespace owner changed evidence: %s calls=%d/%d advances=%d/%d", difference, expected.Calls, local.Calls, expected.TimeAdvances, local.TimeAdvances)
			}
			t.Logf("SQLite %s: calls=%d go-served=%v authorized Go-owner=%dns helper-owner=%dns", journal, local.Calls, expected.GoFilesystemCalls, expected.AuthorizationToTerminalNanos, local.AuthorizationToTerminalNanos)
		})
	}
}

func TestBrokerCrashRetainsInterruptedRuntimePrefix(t *testing.T) {
	request := cooperativeRequest(t, "../toolchain/testdata/choices")
	requireSmallHelperParityModule(t, request)
	request.Config.Clock.ReadStepNanos = 0
	config := brokerTestConfig()
	config.Nodes = config.Nodes[:1]
	broker, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	request.Broker = broker.Session("a")
	if err := request.Broker.CrashOnOutput([]byte("timer")); err != nil {
		t.Fatal(err)
	}
	crashed, err := Run(t.Context(), request)
	if err != nil || crashed.Termination != "crash" || !crashed.Reaped || !crashed.RuntimeInterrupted || len(crashed.InterruptedChoiceBytes) == 0 || len(crashed.ChoiceTrace.Bytes) != 0 {
		t.Fatalf("modeled broker crash prefix missing or claimed complete: termination=%s reaped=%v interrupted=%v bytes=%d complete=%d err=%v message=%s", crashed.Termination, crashed.Reaped, crashed.RuntimeInterrupted, len(crashed.InterruptedChoiceBytes), len(crashed.ChoiceTrace.Bytes), err, crashed.Message)
	}
	tape, err := choice.ProjectInterruptedReplayPlan(crashed.InterruptedChoiceBytes, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if crashed.Telemetry == nil || !crashed.Telemetry.ResourceUsageAvailable || crashed.Telemetry.ExecuteNanos == 0 || crashed.AuthorizationToTerminalNanos == 0 || crashed.PeakMemoryBytes == 0 || crashed.FuelRemaining == request.Fuel || crashed.HelperResourceUsage == nil {
		t.Fatalf("modeled broker interruption lost actual helper costs: metrics=%+v wait=%+v execution=%d peak=%d fuel=%d", crashed.Telemetry, crashed.HelperResourceUsage, crashed.AuthorizationToTerminalNanos, crashed.PeakMemoryBytes, crashed.FuelRemaining)
	}
	replayBroker, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	request.Broker = replayBroker.Session("a")
	if err := request.Broker.CrashOnOutput([]byte("timer")); err != nil {
		t.Fatal(err)
	}
	control, choices := *request.Runtime, *request.Runtime.Choice
	setChoiceReplay(t, &choices, choice.ModePrefix, crashed.InterruptedChoiceBytes, tape)
	control.Choice, request.Runtime = &choices, &control
	replayed, err := Run(t.Context(), request)
	if err != nil || replayed.Termination != "crash" || !replayed.Reaped || !replayed.RuntimeInterrupted || len(replayed.ChoiceTrace.Bytes) != 0 || replayed.Message != crashed.Message || !bytes.Equal(replayed.InterruptedChoiceBytes, crashed.InterruptedChoiceBytes) || !bytes.Equal(replayed.InterruptedDiagnosticBytes, crashed.InterruptedDiagnosticBytes) || !bytes.Equal(replayed.Transcript, crashed.Transcript) || !bytes.Equal(replayed.NetworkTranscript, crashed.NetworkTranscript) || !bytes.Equal(replayed.Stdout, crashed.Stdout) || !bytes.Equal(replayed.Stderr, crashed.Stderr) || replayed.EnvironmentIdentity != crashed.EnvironmentIdentity {
		t.Fatalf("fresh forced interrupted prefix differs: termination=%s interrupted=%v err=%v message=%s", replayed.Termination, replayed.RuntimeInterrupted, err, replayed.Message)
	}
	t.Logf("actual fresh interrupted-prefix replay: choices=%d transcript=%x diagnostics=%x", len(crashed.InterruptedChoiceBytes), sha256.Sum256(crashed.Transcript), sha256.Sum256(crashed.InterruptedDiagnosticBytes))
	// With a storage model the broker run is split-owned (fn-176.6); its
	// interrupted prefix, transcript and storage evidence equal the Go owner's.
	stored := request
	stored.Config.Storage = &StorageModel{Volumes: []volume.VolumeConfig{{ID: "a-store", Path: "/workspace", CapacityBytes: 8 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1024, Transitions: 10000}}
	stored.Runtime = &control
	choices = runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: choices.ExecutionIdentity, Limit: choices.Limit}
	var owners [2]Result
	for i, remote := range []bool{true, false} {
		b, err := NewBroker(config)
		if err != nil {
			t.Fatal(err)
		}
		stored.Broker, stored.remoteModel = b.Session("a"), remote
		if err := stored.Broker.CrashOnOutput([]byte("timer")); err != nil {
			t.Fatal(err)
		}
		if owners[i], err = Run(t.Context(), stored); err != nil || owners[i].Termination != "crash" || !owners[i].RuntimeInterrupted {
			t.Fatalf("storage broker crash (Go owner=%v): %s %s %v", remote, owners[i].Termination, owners[i].Message, err)
		}
	}
	if diff := sameStorageEvidence(owners[0], owners[1]); diff != "" || !bytes.Equal(owners[0].InterruptedDiagnosticBytes, owners[1].InterruptedDiagnosticBytes) || owners[0].TranscriptSHA256 != owners[1].TranscriptSHA256 {
		t.Fatalf("Go-owned vs split interrupted prefix differs in %s", diff)
	}
	requireSplitServedOnlyOutput(t, "split interrupted prefix", owners[1])
}

func requireSmallHelperParityModule(t *testing.T, request Request) {
	t.Helper()
	info, err := os.Stat(request.ModulePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() > 8<<20 {
		t.Fatalf("short parity gate unexpectedly prepared large module: %d bytes", info.Size())
	}
	t.Logf("small parity module=%s bytes=%d helper=%s runtime=%s model=%s", request.ModuleSHA256, info.Size(), request.HelperSHA256, request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, ImplementationSHA256())
}

func TestNetworkStreamsGuestRecordsAndReplays(t *testing.T) {
	root := prototypeRoot(t)
	module := prototypeFixture(t, "network_streams")
	config := prototypeFixtureConfig()
	config.Args = []string{"network_streams"}
	config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile,
		Limits: netmodel.Limits{Listeners: 16, Connections: 128, Bytes: 8 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 16384},
		Faults: []NetworkFaultV2{
			{Name: "stream-delay", Action: "delay", DelayNanos: uint64(5 * time.Millisecond)},
			{Name: "dial-partition", Action: "partition", AtNanos: uint64(50 * time.Millisecond)},
			{Name: "dial-heal", Action: "heal", AtNanos: uint64(90 * time.Millisecond)},
			{Name: "connection-reset", Action: "reset", AtNanos: uint64(120 * time.Millisecond), Port: 9000},
		}}
	request := prototypeRequest(t, module, config, 7, 2*time.Minute)
	request.DigestTranscript = false
	request.MemoryBytes = 256 << 20
	request.Fuel = 1 << 36
	observed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "network stream recording", observed, 0)
	for _, want := range []string{"chunks 196745 sha256", "half-close drained", "read and accept deadlines", "partition dial deadline and cancellation", "named reset; new connection survived"} {
		if !bytes.Contains(observed.Stdout, []byte(want)) {
			t.Fatalf("missing %q in guest stdout: %s", want, observed.Stdout)
		}
	}
	for _, want := range []string{"stream-delay", "dial-partition", "dial-heal", "connection-reset", `"kind":"write"`, `"kind":"network"`, `"op":"cancel"`, `"op":"poll"`, `"op":"close_read"`, `"op":"close_write"`} {
		if !bytes.Contains(observed.NetworkTranscript, []byte(want)) {
			t.Fatalf("missing %q in network evidence", want)
		}
	}
	tape, err := choice.ProjectReplayPlan(observed.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	setChoiceReplay(t, request.Runtime.Choice, choice.ModeReplay, observed.ChoiceTrace.Bytes, tape)
	replayed, err := Run(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "network stream replay", replayed, 0)
	if mismatch := sameEvidence(observed, replayed); mismatch != "" {
		t.Fatalf("network stream replay differs: %s", mismatch)
	}
	if !bytes.Equal(observed.Transcript, replayed.Transcript) || len(observed.NetworkTranscript) == 0 {
		t.Fatal("network stream replay transcript differs or network evidence is empty")
	}
	report, err := json.MarshalIndent(struct {
		Schema, ModuleSHA256, HelperSHA256, RuntimeSHA256 string
		NetworkSHA256, WASISHA256, ChoiceSHA256           string
		ModelImplementationSHA256, ProjectionSHA256       string
		Seed                                              uint64
		ChoiceRecords                                     int
		ExactReplay                                       bool
		Model                                             *NetworkModelV2
	}{"gomad.wasm-network-streams-execution/v1", request.ModuleSHA256, request.HelperSHA256,
		request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey, digest(observed.NetworkTranscript),
		digest(observed.Transcript), digest(observed.ChoiceTrace.Bytes), ImplementationSHA256(), netmodel.ImplementationSHA256(), request.Runtime.Seed,
		len(observed.ChoiceTrace.Records), true, config.NetworkV2}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{
		"network-streams.report.json":  report,
		"network-streams.stdout":       observed.Stdout,
		"network-streams.wasi.bin":     observed.Transcript,
		"network-streams.network.json": observed.NetworkTranscript,
		"network-streams.choices":      observed.ChoiceTrace.Bytes,
	} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("actual stream bytes, parked readiness and exact replay: network=%s wasi=%s choices=%d", digest(observed.NetworkTranscript), digest(observed.Transcript), len(observed.ChoiceTrace.Records))
}

func storageSQLiteRequest(t *testing.T) Request {
	t.Helper()
	helper := helperPath(t)
	version, err := exec.Command(stockgo.Command(), "version").Output()
	if err != nil || !strings.Contains(string(version), "go1.27.1 ") {
		t.Fatalf("storage guest requires stock Go 1.27.1: %q %v", version, err)
	}
	root := stockgo.Root(t)
	native, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	overlay, identity, err := toolchain.BuildOverlay(root, native, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "storage.wasm")
	fixture, err := filepath.Abs("../testdata/storage_sqlite/main.go")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(filepath.Join(root, "bin/go"), "build", "-trimpath", "-tags", "sqlite3_dotlk", "-overlay", overlay, "-o", module, fixture)
	command.Dir = "../../../../../.."
	command.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=-mod=readonly", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build file-backed SQLite guest: %v\n%s", err, output)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	config := testConfig()
	config.Profile = CooperativeProfile
	config.WritableDirectories = append(config.WritableDirectories, "/volume")
	config.Limits.Calls = 100000
	config.Limits.TranscriptBytes = 32 << 20
	config.Storage = &StorageModel{Volumes: []volume.VolumeConfig{{ID: "sqlite", Path: "/volume", CapacityBytes: 4 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 10000, Transitions: 100000}}
	execution := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256(data), ToolchainBuildKey: identity, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte(identity))}
	t.Logf("actual SQLite guest identities: module=%s helper=%s runtime=%s model=%s", fileHash(t, module), fileHash(t, helper), identity, ImplementationSHA256())
	cache := t.TempDir()
	if err := os.Chmod(cache, 0700); err != nil {
		t.Fatal(err)
	}
	return Request{CompileCache: cache, HelperPath: helper, HelperSHA256: fileHash(t, helper), ModulePath: module, ModuleSHA256: fileHash(t, module), Engine: PinnedEngine(), Imports: imports, InitialMemoryPages: pages, MemoryBytes: 256 << 20, Fuel: 4000000000, Config: config, Timeout: time.Minute, Runtime: &RuntimeControl{Seed: 7, Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: execution, Limit: 1 << 20}}}
}

func storageGuestOperation(request Request, operation, journal string, image *StorageImage) Request {
	request.Config.Args = []string{"storage", operation, journal}
	request.StorageImage = image
	control := *request.Runtime
	choices := *control.Choice
	control.Choice = &choices
	request.Runtime = &control
	return request
}

func runStorageGuest(t *testing.T, request Request) Result {
	t.Helper()
	// Every storage guest result may seed a later recovery.
	request.ReturnStorageImage = true
	result, err := Run(t.Context(), request)
	if result.StorageImage != nil {
		t.Logf("storage recovery image: bytes=%d filesystem_limit=%d", result.StorageImage.Bytes(), request.Config.Limits.FilesystemBytes)
	}
	if err != nil || !result.Reaped || result.StorageImage == nil {
		t.Fatalf("storage guest: %s %s reaped=%v image=%v err=%v stdout=%s stderr=%s", result.Termination, result.Message, result.Reaped, result.StorageImage != nil, err, result.Stdout, result.Stderr)
	}
	return result
}

func requireStorageGuestExit(t *testing.T, result Result) {
	t.Helper()
	if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 {
		t.Fatalf("storage guest failed: %s %s exit=%v stdout=%s stderr=%s", result.Termination, result.Message, result.ExitCode, result.Stdout, result.Stderr)
	}
}

func replayStorageGuest(t *testing.T, request Request, recorded Result) Result {
	t.Helper()
	if recorded.Termination == "exit" || recorded.Termination == "storage_crash" {
		var tape choice.ReplayPlan
		var err error
		mode := choice.ModeReplay
		if recorded.Termination == "storage_crash" {
			if !recorded.RuntimeInterrupted || len(recorded.ChoiceTrace.Bytes) != 0 {
				t.Fatal("storage crash was not retained as interrupted runtime evidence")
			}
			tape, err = choice.ProjectInterruptedReplayPlan(recorded.InterruptedChoiceBytes, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
			mode = choice.ModePrefix
		} else {
			tape, err = choice.ProjectReplayPlan(recorded.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
		}
		if err != nil {
			t.Fatal(err)
		}
		control := *request.Runtime
		choices := *control.Choice
		source := recorded.ChoiceTrace.Bytes
		if recorded.RuntimeInterrupted {
			source = recorded.InterruptedChoiceBytes
		}
		setChoiceReplay(t, &choices, mode, source, tape)
		control.Choice = &choices
		request.Runtime = &control
	}
	replayed := runStorageGuest(t, request)
	sameExit := replayed.ExitCode == nil && recorded.ExitCode == nil || replayed.ExitCode != nil && recorded.ExitCode != nil && *replayed.ExitCode == *recorded.ExitCode
	if replayed.Termination != recorded.Termination || !sameExit || replayed.Message != recorded.Message || !bytes.Equal(replayed.Stdout, recorded.Stdout) || !bytes.Equal(replayed.Stderr, recorded.Stderr) || !bytes.Equal(replayed.Transcript, recorded.Transcript) || !bytes.Equal(replayed.StorageTranscript, recorded.StorageTranscript) || !bytes.Equal(replayed.ChoiceTrace.Bytes, recorded.ChoiceTrace.Bytes) || !bytes.Equal(replayed.InterruptedChoiceBytes, recorded.InterruptedChoiceBytes) || !bytes.Equal(replayed.InterruptedDiagnosticBytes, recorded.InterruptedDiagnosticBytes) || replayed.RuntimeInterrupted != recorded.RuntimeInterrupted || replayed.StorageImage.Identity() != recorded.StorageImage.Identity() || replayed.EnvironmentIdentity != recorded.EnvironmentIdentity {
		t.Fatalf("fresh storage replay differs: %s/%s message=%q/%q stdout=%s/%s storage=%x/%x choice=%x/%x image=%s/%s", recorded.Termination, replayed.Termination, recorded.Message, replayed.Message, recorded.Stdout, replayed.Stdout, sha256.Sum256(recorded.StorageTranscript), sha256.Sum256(replayed.StorageTranscript), recorded.ChoiceTrace.SHA256, replayed.ChoiceTrace.SHA256, recorded.StorageImage.Identity(), replayed.StorageImage.Identity())
	}
	return replayed
}

func storageGuestEvents(t *testing.T, result Result) []StorageEvent {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(result.StorageTranscript))
	var events []StorageEvent
	for decoder.More() {
		var event StorageEvent
		if err := decoder.Decode(&event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	return events
}

func requireStorageHistory(t *testing.T, request Request, image *StorageImage, history string) Result {
	t.Helper()
	request.StorageImage = image
	result := runStorageGuest(t, request)
	requireStorageGuestExit(t, result)
	// Exact ordered rows cover both lost and duplicate effects; integrity_check
	// independently checks the recovered database's structural invariant.
	want := "INTEGRITY ok\nHISTORY " + history + "\n"
	if string(result.Stdout) != want {
		t.Fatalf("recovered history: %q, want %q", result.Stdout, want)
	}
	replayStorageGuest(t, request, result)
	// A second incarnation must reach the same state within two fresh opens.
	request.StorageImage = result.StorageImage
	converged := runStorageGuest(t, request)
	requireStorageGuestExit(t, converged)
	if !bytes.Equal(result.Stdout, converged.Stdout) {
		t.Fatalf("recovered history failed bounded convergence: %s/%s", result.Stdout, converged.Stdout)
	}
	return result
}

func TestSQLiteStorageGuestRecovery(t *testing.T) {
	base := storageSQLiteRequest(t)
	t.Run("raw-durable-lock", func(t *testing.T) {
		request := storageGuestOperation(base, "raw-lock", "WAL", nil)
		locked := runStorageGuest(t, request)
		requireStorageGuestExit(t, locked)
		if string(locked.Stdout) != "RAW LOCK durable\n" {
			t.Fatalf("raw lock not acknowledged: %s", locked.Stdout)
		}
		replayStorageGuest(t, request, locked)
		probe := storageGuestOperation(base, "probe-lock", "WAL", locked.StorageImage)
		blocked := runStorageGuest(t, probe)
		requireStorageGuestExit(t, blocked)
		var code, extended int
		if _, err := fmt.Sscanf(string(blocked.Stdout), "LOCKED code=%d extended=%d\n", &code, &extended); err != nil || code != 5 && code != 6 || extended == 0 {
			t.Fatalf("durable lock did not block fresh SQLite with typed BUSY/LOCKED: %s %v", blocked.Stdout, err)
		}
		replayStorageGuest(t, probe, blocked)
		if !slices.ContainsFunc(blocked.StorageSnapshots[0].Persisted, func(entry volume.SnapshotEntry) bool { return entry.Path == "/db.lock" }) {
			t.Fatal("raw persisted lock was deleted during recovery")
		}
		t.Logf("raw durable lock blocks reopened SQLite: code=%d extended=%d", code, extended)
	})
	t.Run("acknowledged-WAL-crash", func(t *testing.T) {
		request := base
		model := *base.Config.Storage
		model.SQLiteLockDatabases = []string{"/volume/db"}
		model.Faults = []StorageFault{{Name: "acknowledged-crash", Resource: "/volume/crash-trigger", Operation: "open", Occurrence: 1, Action: "crash-before"}}
		request.Config.Storage = &model
		setup := runStorageGuest(t, storageGuestOperation(request, "setup-live", "WAL", nil))
		requireStorageGuestExit(t, setup)
		crashRequest := storageGuestOperation(request, "append-crash", "WAL", setup.StorageImage)
		crashRequest.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}
		crashed := runStorageGuest(t, crashRequest)
		if crashed.Termination != "storage_crash" || string(crashed.Stdout) != "ACK 2\n" {
			t.Fatalf("acknowledged WAL was not crashed with SQLite open: %s %s %s", crashed.Termination, crashed.Message, crashed.Stdout)
		}
		replayed := replayStorageGuest(t, crashRequest, crashed)
		recoverRequest := storageGuestOperation(request, "recover", "WAL", nil)
		requireStorageHistory(t, recoverRequest, crashed.StorageImage, "1:committed,2:committed")
		requireStorageHistory(t, recoverRequest, replayed.StorageImage, "1:committed,2:committed")
		enumeration := crashed.StorageCrashEnumeration["sqlite"]
		if !enumeration.Complete || enumeration.Capacity != "" || len(enumeration.States) <= 1 || crashed.StorageSnapshots[0].PendingOperations == 0 {
			t.Fatalf("live SQLite crash envelope incomplete or trivial: %+v", enumeration)
		}
		markerStates := map[string]bool{}
		for index, state := range enumeration.States {
			t.Run(fmt.Sprintf("selection-%d", index), func(t *testing.T) {
				selectedRequest := crashRequest
				selectedRequest.StorageCrashSelection = map[string][]uint64{"sqlite": state.SelectedOperations}
				selectedRequest.StorageCrashContext = crashed.StorageCrashContext
				selected := runStorageGuest(t, selectedRequest)
				if selected.Termination != "storage_crash" || string(selected.Stdout) != "ACK 2\n" {
					t.Fatalf("selected acknowledged crash: %s %s %s", selected.Termination, selected.Message, selected.Stdout)
				}
				replayStorageGuest(t, selectedRequest, selected)
				recovered := requireStorageHistory(t, recoverRequest, selected.StorageImage, "1:committed,2:committed")
				predicted := slices.IndexFunc(state.Entries, func(entry volume.CrashEntry) bool { return entry.Path == "/marker" })
				actual := slices.IndexFunc(recovered.StorageSnapshots[0].Persisted, func(entry volume.SnapshotEntry) bool { return entry.Path == "/marker" })
				if predicted < 0 {
					if actual >= 0 {
						t.Fatal("unselected marker creation survived the crash")
					}
					markerStates["absent"] = true
					t.Logf("selected operations=%v: marker absent", state.SelectedOperations)
					return
				}
				data := state.Entries[predicted].Data
				if actual < 0 || recovered.StorageSnapshots[0].Persisted[actual].Size != uint64(len(data)) || recovered.StorageSnapshots[0].Persisted[actual].DataSHA256 != fmt.Sprintf("sha256:%x", sha256.Sum256(data)) {
					t.Fatal("actual recovered marker differs from the reviewed native crash state")
				}
				markerStates[string(data)] = true
				t.Logf("selected operations=%v: marker bytes=%d sha256=%s", state.SelectedOperations, len(data), recovered.StorageSnapshots[0].Persisted[actual].DataSHA256)
			})
		}
		if len(markerStates) < 3 {
			t.Fatalf("crash selections did not realize absent, empty and written marker states: %v", markerStates)
		}
		t.Logf("all %d bounded dependency-valid SQLite crash states preserved ACK2; marker states=%d pending operations=%d", len(enumeration.States), len(markerStates), crashed.StorageSnapshots[0].PendingOperations)
	})
	t.Run("live-WAL-commit", func(t *testing.T) {
		request := base
		model := *base.Config.Storage
		model.SQLiteLockDatabases = []string{"/volume/db"}
		request.Config.Storage = &model
		setupRequest := storageGuestOperation(request, "setup-live", "WAL", nil)
		setup := runStorageGuest(t, setupRequest)
		requireStorageGuestExit(t, setup)
		replayStorageGuest(t, setupRequest, setup)
		appendRequest := storageGuestOperation(request, "append-live", "WAL", setup.StorageImage)
		appended := runStorageGuest(t, appendRequest)
		requireStorageGuestExit(t, appended)
		if string(appended.Stdout) != "ACK 2\n" {
			t.Fatalf("live WAL transaction not acknowledged: %s", appended.Stdout)
		}
		replayStorageGuest(t, appendRequest, appended)
		if !slices.ContainsFunc(storageGuestEvents(t, appended), func(event StorageEvent) bool { return event.Resource == "/volume/db-wal" && event.Operation == "sync" }) {
			t.Fatal("live WAL commit did not sync its log")
		}
		if !slices.ContainsFunc(appended.StorageSnapshots[0].Persisted, func(entry volume.SnapshotEntry) bool { return entry.Path == "/db-wal" }) {
			t.Fatal("live WAL disappeared before crash image capture")
		}
		requireStorageHistory(t, storageGuestOperation(request, "recover", "WAL", nil), appended.StorageImage, "1:committed,2:committed")
	})
	for _, journal := range []string{"WAL", "DELETE"} {
		t.Run(journal, func(t *testing.T) {
			setupRequest := storageGuestOperation(base, "setup", journal, nil)
			setup := runStorageGuest(t, setupRequest)
			requireStorageGuestExit(t, setup)
			if len(setup.ChoiceTrace.Records) == 0 {
				t.Fatal("SQLite guest produced no cooperative decisions")
			}
			replayStorageGuest(t, setupRequest, setup)
			appendRequest := storageGuestOperation(base, "append", journal, setup.StorageImage)
			appended := runStorageGuest(t, appendRequest)
			requireStorageGuestExit(t, appended)
			if string(appended.Stdout) != "ACK 2\n" {
				t.Fatalf("successful commit not acknowledged: %s", appended.Stdout)
			}
			replayStorageGuest(t, appendRequest, appended)
			recoverRequest := storageGuestOperation(base, "recover", journal, nil)
			requireStorageHistory(t, recoverRequest, appended.StorageImage, "1:committed,2:committed")
			events := storageGuestEvents(t, appended)
			for _, operation := range []string{"read", "write", "sync"} {
				if !slices.ContainsFunc(events, func(event StorageEvent) bool { return event.Operation == operation }) {
					t.Fatalf("SQLite bypassed storage %s: %+v", operation, events)
				}
			}
			records, err := DecodeTranscript(appended.Transcript)
			if err != nil {
				t.Fatal(err)
			}
			for _, operation := range []string{"fd_pread", "fd_pwrite", "fd_sync"} {
				if !slices.ContainsFunc(records, func(record TranscriptRecord) bool { return record.Op == operation }) {
					t.Fatalf("SQLite did not reach WASI %s", operation)
				}
			}
			resource := "/volume/db-journal"
			if journal == "WAL" {
				resource = "/volume/db-wal"
			}
			for _, action := range []string{"short-write", "io-error", "no-space", "sync-error", "torn-write", "crash-before"} {
				t.Run(action, func(t *testing.T) {
					operation := "write"
					if action == "sync-error" || action == "crash-before" {
						operation = "sync"
					}
					index := slices.IndexFunc(events, func(event StorageEvent) bool { return event.Resource == resource && event.Operation == operation })
					if index < 0 {
						t.Fatalf("missing SQLite %s %s", resource, operation)
					}
					fault := StorageFault{Name: "sqlite-" + action, Resource: resource, Operation: operation, Occurrence: events[index].Occurrence, Action: action}
					if action == "short-write" || action == "torn-write" {
						fault.Bytes = 1
					}
					request := base
					model := *base.Config.Storage
					model.Faults = []StorageFault{fault}
					request.Config.Storage = &model
					initialized := runStorageGuest(t, storageGuestOperation(request, "setup", journal, nil))
					requireStorageGuestExit(t, initialized)
					faultRequest := storageGuestOperation(request, "append", journal, initialized.StorageImage)
					failed := runStorageGuest(t, faultRequest)
					switch action {
					case "crash-before":
						if failed.Termination != "storage_crash" {
							t.Fatalf("fault did not kill helper: %s %s", failed.Termination, failed.Message)
						}
					case "short-write":
						requireStorageGuestExit(t, failed)
						if string(failed.Stdout) != "ACK 2\n" {
							t.Fatalf("Go WriteAt did not complete the short write: %s", failed.Stdout)
						}
					default:
						requireStorageGuestExit(t, failed)
						var code, extended int
						if _, err := fmt.Sscanf(string(failed.Stdout), "REJECTED 2 code=%d extended=%d\n", &code, &extended); err != nil || extended == 0 || code != 10 && (action != "no-space" || code != 13) {
							t.Fatalf("faulty commit did not produce SQLite IOERR/FULL: %s (%v)", failed.Stdout, err)
						}
						t.Logf("actual SQLite storage fault %s: code=%d extended=%d", action, code, extended)
					}
					if !slices.ContainsFunc(storageGuestEvents(t, failed), func(event StorageEvent) bool { return event.Fault == fault.Name && event.Action == action }) {
						t.Fatal("storage fault not realized by actual SQLite guest")
					}
					faultReplay := replayStorageGuest(t, faultRequest, failed)
					recoveredRequest := storageGuestOperation(request, "recover", journal, nil)
					history := "1:committed"
					if action == "short-write" {
						history += ",2:committed"
					}
					recovered := requireStorageHistory(t, recoveredRequest, failed.StorageImage, history)
					replayedRecovery := requireStorageHistory(t, recoveredRequest, faultReplay.StorageImage, history)
					if !bytes.Equal(recovered.Stdout, replayedRecovery.Stdout) || !bytes.Equal(recovered.StorageTranscript, replayedRecovery.StorageTranscript) {
						t.Fatal("replayed fault recovered a different history")
					}
				})
			}
		})
	}
}

func TestSQLiteStorageGuestUnsyncedAcknowledgementCanary(t *testing.T) {
	request := storageSQLiteRequest(t)
	setup := runStorageGuest(t, storageGuestOperation(request, "setup", "WAL", nil))
	requireStorageGuestExit(t, setup)
	unsyncedRequest := storageGuestOperation(request, "unsynced", "WAL", setup.StorageImage)
	unsynced := runStorageGuest(t, unsyncedRequest)
	requireStorageGuestExit(t, unsynced)
	if string(unsynced.Stdout) != "ACK 2\n" {
		t.Fatalf("negative canary did not acknowledge: %s", unsynced.Stdout)
	}
	replayStorageGuest(t, unsyncedRequest, unsynced)
	recovered := requireStorageHistory(t, storageGuestOperation(request, "recover", "WAL", nil), unsynced.StorageImage, "1:committed")
	if bytes.Contains(recovered.Stdout, []byte("2:committed")) {
		t.Fatal("missing-sync durability violation escaped the history oracle")
	}
}

func temporalRecoveryRequest(t *testing.T, fixture string) (Request, string) {
	t.Helper()
	return temporalRecoveryRequestWithAdmission(t, fixture, nil)
}

func TestClusterRetentionRefusesGoServedCalls(t *testing.T) {
	if err := clusterServedOnlyByHelper(Result{}, 0); err != nil {
		t.Fatalf("helper-only incarnation refused: %v", err)
	}
	for _, calls := range []map[string]uint64{{}, {"fd_write": 1}, {"path_open": 1}} {
		if err := clusterServedOnlyByHelper(Result{GoFilesystemCalls: calls}, 0); err == nil {
			t.Fatalf("Go filesystem %v accepted", calls)
		}
	}
	if err := clusterServedOnlyByHelper(Result{}, 1); err == nil {
		t.Fatal("relayed guest callback accepted")
	}
}

func temporalStorageBrokerConfiguration() BrokerConfig {
	return BrokerConfig{Nodes: []BrokerNode{{ID: "temporal", Incarnation: 1}}, Seed: 7,
		Limits: world.Limits{MaxRequests: 1000000, MaxEvents: 1000000, MaxQueuedEvents: 65536, MaxTransitions: 4000000, MaxPayloadBytes: 64 << 20, MaxStringBytes: 1024}}
}

func TestTemporalStorageBrokerAdmission(t *testing.T) {
	config := temporalStorageBrokerConfiguration()
	if _, err := NewBroker(config); err != nil {
		t.Fatal(err)
	}
	if config.Limits.MaxRequests > config.Limits.MaxPayloadBytes/(2*sha256.Size) {
		t.Fatal("storage broker can exhaust payload bytes before its request bound")
	}
	config.Limits.MaxPayloadBytes++
	if _, err := NewBroker(config); err == nil {
		t.Fatal("storage broker admitted payload bytes beyond the compressed World profile")
	}
}

func TestTemporalStorageBrokerPayloadAccounting(t *testing.T) {
	for _, test := range []struct {
		name      string
		bytes     uint64
		accepted  uint64
		dimension string
	}{{"request-first", 192, 3, "requests"}, {"payload-double-accounting", 191, 2, "payload-bytes"}} {
		t.Run(test.name, func(t *testing.T) {
			config := temporalStorageBrokerConfiguration()
			config.Limits.MaxRequests, config.Limits.MaxPayloadBytes = 3, test.bytes
			broker, err := NewBroker(config)
			if err != nil {
				t.Fatal(err)
			}
			session := broker.Session("temporal")
			payload := bytes.Repeat([]byte("storage"), 128)
			for range test.accepted {
				if err := session.register("storage", payload); err != nil {
					t.Fatal(err)
				}
				if err := session.ready("storage"); err != nil {
					t.Fatal(err)
				}
			}
			before := broker.world.DiagnosticCardinality()
			if before.Requests != test.accepted || before.Events != test.accepted || before.PayloadBytes != test.accepted*64 {
				t.Fatalf("broker payload accounting = %+v", before)
			}
			err = session.register("storage", payload)
			var capacity *world.CapacityError
			if !errors.As(err, &capacity) || capacity.Dimension != test.dimension {
				t.Fatalf("broker boundary = %v, want %s capacity", err, test.dimension)
			}
			if after := broker.world.DiagnosticCardinality(); after != before {
				t.Fatalf("rejected request changed World: before=%+v after=%+v", before, after)
			}
		})
	}
}

func TestTemporalCommittedWorkflowFreshGuestCrashReopen(t *testing.T) {
	request, evidence := temporalRecoveryRequest(t, "temporal_storage")
	request.Config.Args = []string{"recovery.test", "-test.v", "-test.count=1", "-test.timeout=1h", "-test.run=^TestCommittedWorkflowCrashRecovery$", "-persistenceType=sql", "-persistenceDriver=sqlite"}
	request.Config.Environment = append(request.Config.Environment, "GOMAD_TEMPORAL_PHASE=commit")
	request.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 128, Operations: 256, Depth: 256, Bytes: 256 << 20, Work: 1 << 20}
	temporalRetainRequest(t, evidence, "temporal-commit", request)
	newBroker := func() *Broker {
		b, err := NewBroker(temporalStorageBrokerConfiguration())
		if err != nil {
			t.Fatal(err)
		}
		if err := b.Session("temporal").CrashOnOutput([]byte(temporalObservationPrefix)); err != nil {
			t.Fatal(err)
		}
		return b
	}
	b := newBroker()
	request.Broker = b.Session("temporal")
	committed, err := Run(t.Context(), request)
	temporalRetainRecoveryRun(t, evidence, "temporal-committed-crash", committed)
	if err != nil || committed.Termination != "crash" || !committed.Reaped || committed.StorageImage == nil {
		t.Fatalf("abrupt committed guest crash: termination=%s message=%s err=%v stdout=%s stderr=%s", committed.Termination, committed.Message, err, committed.Stdout, committed.Stderr)
	}
	expected := temporalRecoveryValue(t, committed.Stdout)
	crashRequest := request
	if !committed.RuntimeInterrupted {
		t.Fatal("Temporal crash lost interrupted runtime evidence")
	}
	tape, err := choice.ProjectInterruptedReplayPlan(committed.InterruptedChoiceBytes, request.Runtime.Choice.Limit, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	control := *request.Runtime
	choices := *control.Choice
	setChoiceReplay(t, &choices, choice.ModePrefix, committed.InterruptedChoiceBytes, tape)
	control.Choice = &choices
	crashRequest.Runtime = &control
	enumeration := committed.StorageCrashEnumeration["temporal"]
	if !enumeration.Complete || len(enumeration.States) == 0 {
		t.Fatalf("Temporal supported crash-state enumeration remains incomplete: capacity=%s states=%d", enumeration.Capacity, len(enumeration.States))
	}
	for i, state := range enumeration.States {
		b := newBroker()
		crashRequest.Broker = b.Session("temporal")
		crashRequest.StorageCrashSelection = map[string][]uint64{"temporal": state.SelectedOperations}
		crashRequest.StorageCrashContext = committed.StorageCrashContext
		crashed, err := Run(t.Context(), crashRequest)
		temporalRetainRecoveryRun(t, evidence, fmt.Sprintf("temporal-state-%d-crash", i), crashed)
		if err != nil || crashed.Termination != "crash" || !crashed.Reaped || crashed.StorageImage == nil || !bytes.Equal(committed.Stdout, crashed.Stdout) || !bytes.Equal(committed.StorageTranscript, crashed.StorageTranscript) || !bytes.Equal(committed.InterruptedChoiceBytes, crashed.InterruptedChoiceBytes) {
			t.Fatalf("reviewed Temporal crash state %d did not replay its exact committed boundary: termination=%s message=%s err=%v", i, crashed.Termination, crashed.Message, err)
		}
		reopen := request
		reopen.StorageCrashLimits = nil
		reopen.StorageImage = crashed.StorageImage
		reopen.Config.Environment = append([]string(nil), request.Config.Environment...)
		reopen.Config.Environment[len(reopen.Config.Environment)-1] = "GOMAD_TEMPORAL_PHASE=reopen"
		reopen.Broker, err = b.Restart("temporal")
		if err != nil {
			t.Fatal(err)
		}
		temporalRetainRequest(t, evidence, fmt.Sprintf("temporal-state-%d-recovered", i), reopen)
		recovered, err := Run(t.Context(), reopen)
		temporalRetainRecoveryRun(t, evidence, fmt.Sprintf("temporal-state-%d-recovered", i), recovered)
		if err != nil || recovered.Termination != "exit" || recovered.ExitCode == nil || *recovered.ExitCode != 0 || !recovered.Reaped || !reflect.DeepEqual(expected, temporalRecoveryValue(t, recovered.Stdout)) {
			t.Fatalf("reviewed Temporal crash state %d lost acknowledged history/result: termination=%s message=%s err=%v", i, recovered.Termination, recovered.Message, err)
		}
		temporalHistoryOracles(t, evidence, fmt.Sprintf("temporal.recovery.state-%d", i), expected, temporalRecoveryValue(t, recovered.Stdout))
		temporalJSON(t, filepath.Join(evidence, fmt.Sprintf("temporal-state-%d-selection.json", i)), struct {
			Identity   string
			Operations []uint64
		}{state.Identity, state.SelectedOperations})
	}
	request.StdoutMirror = nil
	request.StorageCrashLimits = nil
	request.StorageImage = committed.StorageImage
	request.Config.Environment[len(request.Config.Environment)-1] = "GOMAD_TEMPORAL_PHASE=reopen"
	for incarnation := range 2 {
		request.Broker, err = b.Restart("temporal")
		if err != nil {
			t.Fatal(err)
		}
		reopened, err := Run(t.Context(), request)
		temporalRetainRecoveryRun(t, evidence, fmt.Sprintf("temporal-reopened-%d", incarnation), reopened)
		if err != nil || reopened.Termination != "exit" || reopened.ExitCode == nil || *reopened.ExitCode != 0 || !reopened.Reaped {
			t.Fatalf("fresh recovered guest: termination=%s message=%s err=%v stdout=%s stderr=%s", reopened.Termination, reopened.Message, err, reopened.Stdout, reopened.Stderr)
		}
		actual := temporalRecoveryValue(t, reopened.Stdout)
		temporalHistoryOracles(t, evidence, fmt.Sprintf("temporal.recovery.incarnation-%d", incarnation), expected, actual)
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("detached committed history or metadata identity changed after crash: expected=%+v actual=%+v", expected, actual)
		}
	}
	temporalJSON(t, filepath.Join(evidence, "temporal-history-oracle.json"), expected)
}

func TestTemporalGuestAdmission(t *testing.T) {
	target := os.Getenv("GOMAD_WASM_TEMPORAL_TARGET")
	if target == "" {
		t.Skip("set GOMAD_WASM_TEMPORAL_TARGET to run the explicit Temporal guest gate")
	}
	clockSelection := os.Getenv("GOMAD_WASM_TEMPORAL_CLOCK")
	clock, err := temporalClockPolicy(clockSelection)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Join(root, ".tmp", "wasm-backend-temporal")
	adapterMode := os.Getenv("GOMAD_WASM_SQLITE_ADAPTER")
	if adapterMode != "" && adapterMode != "cooperative-v1" {
		t.Fatalf("unknown SQLite adapter %q", adapterMode)
	}
	if target == "grpc-progress-diagnostic" && adapterMode != "" {
		t.Fatal("standalone gRPC diagnostic requires the original dependency graph without SQLite adapter")
	}
	projectionControl := os.Getenv("GOMAD_WASM_SQLITE_PROJECTION_CONTROL") == "1"
	if projectionControl && (adapterMode == "" || target != "contention") {
		t.Fatal("projection-only control requires explicit contention adapter selection")
	}
	if adapterMode != "" {
		evidence = filepath.Join(evidence, "cooperative-sqlite-v1")
		if projectionControl {
			evidence = filepath.Join(evidence, "projection-control")
		}
	}
	if clockSelection != "" {
		evidence = filepath.Join(evidence, clockSelection)
	}
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal(err)
	}
	packageName, version, cwd := "", "go1.27.0", "/workspace/tests"
	selections := map[string]string{}
	tags := "disable_grpc_modules,test_dep,sqlite3_dotlk"
	switch target {
	case "sqlite":
		packageName, cwd = "./common/persistence/sql/sqlplugin/sqlite", "/workspace/common/persistence/sql/sqlplugin/sqlite"
		selections["sqlite"] = "."
	case "persistence":
		packageName, cwd = "./common/persistence/tests", "/workspace/common/persistence/tests"
		selections["cancellation"] = "^TestSQLiteTransactionContextCancellation$"
		selections["visibility"] = "^TestSQLiteVisibilitySuite$"
		selections["visibility-persistence"] = "^TestSQLiteVisibilityPersistenceSuite$"
	case "frontend":
		packageName, cwd = "./tests/gomadfunctional", "/workspace/tests/gomadfunctional"
		selections["frontend"] = "^TestFrontendSystemInfo$"
	case "namespace-diagnostic":
		packageName, cwd = "./.tmp/wasm-backend-temporal/frontend-namespace", "/workspace/namespace-diagnostic"
		selections["namespace-diagnostic"] = "^TestNamespaceRPCCause$"
	case "namespace-stack-diagnostic":
		if adapterMode != "cooperative-v1" {
			t.Fatal("namespace stack diagnostic requires the explicitly admitted cooperative SQLite adapter")
		}
		packageName, cwd = "./.tmp/wasm-backend-temporal/frontend-namespace-stack", "/workspace/namespace-diagnostic"
		selections["namespace-stack-diagnostic"] = "^TestNamespacePendingStack$"
	case "grpc-progress-diagnostic":
		packageName, cwd = "./.tmp/wasm-backend-temporal/grpc-progress", "/workspace/grpc-progress-diagnostic"
		selections["grpc-progress-diagnostic"] = "^TestGRPCUnaryProgress$"
	case "contention":
		packageName, cwd = "./.tmp/wasm-backend-temporal/sqlite-contention", "/workspace/contention"
		selections["contention"] = "^TestBusyTimeoutAllowsQueuedLockHolder$"
	case "contention-controls":
		packageName, cwd = "./.tmp/wasm-backend-temporal/sqlite-contention", "/workspace/contention"
		selections["default-timeout"] = "^TestUnreleasedLockPreservesDefaultTimeout$"
		selections["queued-cancellation"] = "^TestQueuedCancellationInterruptsBusyRetry$"
		selections["zero-timeout"] = "^TestZeroBusyTimeoutPreserved$"
		selections["custom-timeout"] = "^TestCustomBusyTimeoutPreserved$"
	case "functional":
		packageName = "./tests"
		selections["workflow"] = "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$"
		selections["workflow-activity-cleanup"] = "^TestActivityApiPause_AttributesToActivityInContextMetadata$"
		selections["activity"] = "^TestActivityTestSuite$"
		selections["child"] = "^TestChildWorkflowSuite$"
		selections["update"] = "^TestWorkflowUpdateSuite$"
		selections["timers"] = "^TestUserTimersTestSuite$"
		selections["application-failure"] = "^TestActivityClientTestSuite$/^TestActivity_AttemptsExceeded$"
	case "failure":
		root = filepath.Join(root, "tools", "gomad", "internal", "backends", "wasm", "testdata", "io_failure_module")
		packageName, version, cwd = "./io_failure", "go1.27.1", "/workspace/failure"
		tags = "test_dep,gomad_fixture"
		selections["failure"] = "^TestDeterministicIOFailure$"
	default:
		t.Fatalf("unknown Temporal gate %q", target)
	}
	if strings.HasPrefix(target, "contention") || target == "namespace-diagnostic" || target == "namespace-stack-diagnostic" || target == "grpc-progress-diagnostic" {
		owner, staged := "sqlite_contention", "sqlite-contention"
		if target == "namespace-diagnostic" {
			owner, staged = "frontend_namespace", "frontend-namespace"
		}
		if target == "namespace-stack-diagnostic" {
			owner, staged = "frontend_namespace", "frontend-namespace-stack"
		}
		if target == "grpc-progress-diagnostic" {
			owner, staged = "environment", "grpc-progress"
		}
		if err := temporalStageFixture(root, filepath.Join("tools", "gomad", "internal", "backends", "wasm", "testdata", owner), root, filepath.Join(".tmp", "wasm-backend-temporal", staged)); err != nil {
			t.Fatal(err)
		}
	}
	compiler := os.Getenv("GOMAD_WASM_TEMPORAL_GO")
	if !filepath.IsAbs(compiler) {
		t.Fatal("GOMAD_WASM_TEMPORAL_GO must name the absolute stock compiler")
	}
	buildEnv := temporalBuildEnvironment(compiler)
	compilerResult := temporalCommand(t, evidence, target+"-compiler", root, buildEnv, []string{compiler, "version"})
	if !strings.Contains(string(compilerResult.Stdout.RawBytes), version+" ") {
		t.Fatalf("wrong stock compiler: %s", compilerResult.Stdout.RawBytes)
	}
	listCommand := []string{compiler, "list", "-mod=readonly", "-tags", tags, "-deps", "-test", "-json", packageName}
	var projection *temporalSQLiteProjection
	var overlayArgs []string
	if adapterMode != "" {
		discovery := temporalCommand(t, evidence, target+"-discovery", root, buildEnv, listCommand)
		projection = temporalSQLiteAdapter(t, evidence, root, discovery.Stdout.RawBytes)
		originalGraph := temporalCommand(t, evidence, target+"-original-modules", root, buildEnv, []string{compiler, "list", "-mod=readonly", "-m", "-json", "all"})
		effectiveGraph := temporalCommand(t, evidence, target+"-projected-modules", root, buildEnv, []string{compiler, "list", "-modfile", projection.ModFile, "-mod=readonly", "-m", "-json", "all"})
		if err := temporalCompareModuleGraphs(originalGraph.Stdout.RawBytes, effectiveGraph.Stdout.RawBytes, filepath.Dir(projection.Original), projection.PrivateRoot, projection.ModFile); err != nil {
			t.Fatal(err)
		}
		overlayArgs = []string{"-modfile", projection.ModFile}
		if !projectionControl {
			overlayArgs = append(overlayArgs, "-overlay", projection.Overlay)
		}
		listCommand = append([]string{compiler, "list"}, append(overlayArgs, listCommand[2:]...)...)
	}
	listed := temporalCommand(t, evidence, target+"-inputs", root, buildEnv, listCommand)
	inputs := temporalSourceInputs(t, listed.Stdout.RawBytes)
	if projection != nil {
		inputs[projection.Original] = projection.OriginalSHA256
		inputs[projection.Replacement] = projection.ReplacementSHA256
		inputs[projection.Overlay] = projection.OverlaySHA256
		inputs[projection.ModFile] = fileHash(t, projection.ModFile)
		inputs[projection.SumFile] = fileHash(t, projection.SumFile)
		for _, name := range []string{target + "-original-modules.stdout", target + "-projected-modules.stdout"} {
			inputs[filepath.Join(evidence, name)] = fileHash(t, filepath.Join(evidence, name))
		}
		inputs[filepath.Join(projection.PrivateRoot, "conn.go")] = projection.OriginalSHA256
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		file := filepath.Join(root, name)
		if _, err := os.Stat(file); err == nil {
			inputs[file] = fileHash(t, file)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { temporalVerifyInputs(t, inputs) })
	module := filepath.Join(evidence, target+".test.wasm")
	command := []string{compiler, "test", "-mod=readonly", "-tags", tags, "-c", "-o", module, packageName}
	if projection != nil {
		command = append([]string{compiler, "test"}, append(overlayArgs, command[2:]...)...)
	}
	temporalCommand(t, evidence, target+"-build", root, buildEnv, command)
	temporalVerifyInputs(t, inputs)
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages := binaryInventory(t, data)
	helper := helperPath(t)
	provenance := struct {
		Compiler, CompilerSHA256, ModuleSHA256, HelperSHA256 string
		BuildCommand, BuildEnvironment                       []string
		Inputs                                               map[string]string
		Imports                                              []Import
		InitialMemoryPages                                   uint64
		Engine                                               EngineIdentity
		SQLiteAdapter                                        *temporalSQLiteProjection
		DiagnosticOnly                                       bool
		ClockSelection, ClockClassification                  string
		Clock                                                ClockPolicy
		ConfigIdentities                                     map[string]string
	}{string(compilerResult.Stdout.RawBytes), fileHash(t, compiler), fileHash(t, module), fileHash(t, helper), command, buildEnv, inputs, imports, pages, PinnedEngine(), projection, strings.HasPrefix(target, "contention") || target == "namespace-diagnostic" || target == "namespace-stack-diagnostic" || target == "grpc-progress-diagnostic", clockSelection, "exploratory; no strict-time or production clock-policy qualification", clock, map[string]string{}}
	temporalJSON(t, filepath.Join(evidence, target+"-provenance.json"), provenance)
	config := Config{
		Profile: StockProfile, WorkingDirectory: cwd,
		Environment:         []string{"PWD=" + cwd, "TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TZ=UTC"},
		WritableDirectories: []string{"/workspace", "/tmp", cwd},
		EntropyKey:          sha256.Sum256([]byte("gomad-wasm-temporal-admission/v1")),
		Clock:               clock,
		Limits:              Limits{OutputBytes: 8 << 20, TranscriptBytes: 128 << 20, Calls: 250000, Descriptors: 1024, Files: 10000, FilesystemBytes: 128 << 20, PendingEvents: 4096},
	}
	if target != "failure" {
		captured, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: filepath.Join(root, "schema", "sqlite"), Target: "/workspace/schema/sqlite"}}, captured.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		config.CapturedInputs = captured
		temporalJSON(t, filepath.Join(evidence, target+"-captured.json"), captured)
	}
	names := make([]string, 0, len(selections))
	for name := range selections {
		names = append(names, name)
	}
	slices.Sort(names)
	if target == "functional" {
		names = []string{"workflow", "workflow-activity-cleanup", "activity", "child", "update", "timers", "application-failure"}
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			config.Args = []string{"temporal.test", "-test.v", "-test.count=1", "-test.timeout=20m", "-test.run=" + selections[name]}
			if target == "frontend" || target == "functional" || target == "namespace-diagnostic" || target == "namespace-stack-diagnostic" {
				config.Args = append(config.Args, "-persistenceType=sql", "-persistenceDriver=sqlite")
			}
			environment, err := NewEnvironment(config)
			if err != nil {
				t.Fatal(err)
			}
			provenance.ConfigIdentities[name] = environment.Identity()
			temporalJSON(t, filepath.Join(evidence, target+"-provenance.json"), provenance)
			request := Request{HelperPath: helper, HelperSHA256: provenance.HelperSHA256, ModulePath: module, ModuleSHA256: provenance.ModuleSHA256, Engine: provenance.Engine, Imports: imports, InitialMemoryPages: pages, MemoryBytes: 2 << 30, Fuel: 2000000000000, Config: config, Timeout: 15 * time.Minute}
			temporalJSON(t, filepath.Join(evidence, name+"-request.json"), request)
			var first Result
			repeat := 1
			if target == "failure" {
				repeat = 2
			}
			for iteration := range repeat {
				started := time.Now()
				result, err := Run(t.Context(), request)
				prefix := filepath.Join(evidence, fmt.Sprintf("%s-%d", name, iteration))
				temporalWrite(t, prefix+".stdout", result.Stdout)
				temporalWrite(t, prefix+".stderr", result.Stderr)
				temporalWrite(t, prefix+".transcript.bin", result.Transcript)
				metadata := result
				metadata.Stdout, metadata.Stderr, metadata.Transcript = nil, nil, nil
				temporalJSON(t, prefix+".json", struct {
					Result                                       Result
					ElapsedNanos                                 int64
					StdoutSHA256, StderrSHA256, TranscriptSHA256 string
					Error                                        string
				}{metadata, time.Since(started).Nanoseconds(), temporalHash(result.Stdout), temporalHash(result.Stderr), temporalHash(result.Transcript), fmt.Sprint(err)})
				if err != nil {
					t.Fatal(err)
				}
				if result.EnvironmentIdentity != provenance.ConfigIdentities[name] {
					t.Fatal("guest configuration identity differs from admitted provenance")
				}
				wantExit := uint32(0)
				if target == "failure" || projectionControl {
					wantExit = 1
				}
				if (target == "namespace-diagnostic" || target == "namespace-stack-diagnostic") && result.ExitCode != nil && *result.ExitCode <= 1 {
					wantExit = *result.ExitCode
				}
				if result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != wantExit || !result.Reaped {
					t.Fatalf("guest %s: termination=%s exit=%v reaped=%v message=%s; complete output retained in %s", name, result.Termination, result.ExitCode, result.Reaped, result.Message, prefix)
				}
				if err := temporalCheckOutput(result.Stdout, selections[name], target == "failure" || projectionControl || (target == "namespace-diagnostic" || target == "namespace-stack-diagnostic") && wantExit == 1); err != nil {
					t.Fatal(err)
				}
				if target == "failure" && !bytes.Contains(result.Stdout, []byte(`deterministic failure after reading "pending"`)) {
					t.Fatal("deliberate application failure was not reached")
				}
				if iteration > 0 && (!bytes.Equal(first.Stdout, result.Stdout) || !bytes.Equal(first.Stderr, result.Stderr) || !bytes.Equal(first.Transcript, result.Transcript)) {
					t.Fatal("fresh deliberate failure output or transcript diverged")
				}
				first = result
				t.Logf("target=%s guest=%s iteration=%d exit=%d environment=%s module=%s transcript=%s memory=%d fuel-remaining=%d", target, name, iteration, wantExit, result.EnvironmentIdentity, request.ModuleSHA256, temporalHash(result.Transcript), result.PeakMemoryBytes, result.FuelRemaining)
			}
		})
	}
}

func temporalClockPolicy(selection string) (ClockPolicy, error) {
	step := uint64(1000000)
	switch selection {
	case "":
	case "read-step-1us-v1":
		step = 1000
	default:
		return ClockPolicy{}, fmt.Errorf("unknown Temporal exploratory clock policy %q", selection)
	}
	return ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: step}, nil
}

func TestTemporalClockPolicy(t *testing.T) {
	for _, test := range []struct {
		selection string
		step      uint64
	}{
		{"", 1000000},
		{"read-step-1us-v1", 1000},
		{"read-step-1ms-v1", 0},
		{"1000", 0},
		{"read-step-1us-v2", 0},
		{"read-step-1us-v1 ", 0},
		{"\xff", 0},
	} {
		t.Run(test.selection, func(t *testing.T) {
			clock, err := temporalClockPolicy(test.selection)
			if test.step == 0 {
				if err == nil || clock != (ClockPolicy{}) {
					t.Fatal("accepted an arbitrary clock policy")
				}
				return
			}
			if err != nil || clock != (ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: test.step}) {
				t.Fatalf("clock policy = %+v, error = %v", clock, err)
			}
		})
	}
}

func TestTemporalClockConfigurationIdentity(t *testing.T) {
	var identities []string
	for _, selection := range []string{"", "read-step-1us-v1"} {
		clock, err := temporalClockPolicy(selection)
		if err != nil {
			t.Fatal(err)
		}
		config := testConfig()
		config.Clock = clock
		environment, err := NewEnvironment(config)
		if err != nil {
			t.Fatal(err)
		}
		identities = append(identities, environment.Identity())
	}
	if identities[0] == identities[1] {
		t.Fatal("clock selection was omitted from the configuration identity")
	}
}

func TestTemporalGuestOutput(t *testing.T) {
	for _, test := range []struct {
		name, selection, output string
		failure, valid          bool
	}{
		{"empty", "^TestFrontendSystemInfo$", "", false, false},
		{"zero-match", "^TestFrontendSystemInfo$", "testing: warning: no tests to run\nPASS\n", false, false},
		{"skip", "^TestFrontendSystemInfo$", "--- SKIP: TestFrontendSystemInfo (0.00s)\nPASS\n", false, false},
		{"different", "^TestFrontendSystemInfo$", "--- PASS: TestFrontendSystemInfoOther (0.00s)\nPASS\n", false, false},
		{"pass", "^TestFrontendSystemInfo$", "--- PASS: TestFrontendSystemInfo (0.00s)\nPASS\n", false, true},
		{"grpc-zero-match", "^TestGRPCUnaryProgress$", "testing: warning: no tests to run\nPASS\n", false, false},
		{"grpc-skip", "^TestGRPCUnaryProgress$", "--- SKIP: TestGRPCUnaryProgress (0.00s)\nPASS\n", false, false},
		{"grpc-pass", "^TestGRPCUnaryProgress$", "--- PASS: TestGRPCUnaryProgress (0.00s)\nPASS\n", false, true},
		{"stack-zero-match", "^TestNamespacePendingStack$", "testing: warning: no tests to run\nPASS\n", false, false},
		{"stack-skip", "^TestNamespacePendingStack$", "--- SKIP: TestNamespacePendingStack (0.00s)\nPASS\n", false, false},
		{"stack-pass", "^TestNamespacePendingStack$", "--- PASS: TestNamespacePendingStack (0.00s)\nPASS\n", false, true},
		{"all-suite-skipped", "^TestActivityTestSuite$", "--- PASS: TestActivityTestSuite (0.00s)\n    --- SKIP: TestActivityTestSuite/TestActivity (0.00s)\nPASS\n", false, false},
		{"suite", "^TestActivityTestSuite$", "--- PASS: TestActivityTestSuite (0.00s)\n    --- PASS: TestActivityTestSuite/TestActivity (0.00s)\nPASS\n", false, true},
		{"failure", "^TestDeterministicIOFailure$", "--- FAIL: TestDeterministicIOFailure (0.00s)\nFAIL\n", true, true},
		{"wrong-failure", "^TestDeterministicIOFailure$", "--- FAIL: TestOther (0.00s)\nFAIL\n", true, false},
		{"nested", "^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$", "    --- PASS: TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds (0.00s)\nPASS\n", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := temporalCheckOutput([]byte(test.output), test.selection, test.failure); (err == nil) != test.valid {
				t.Fatalf("output validity: expected %v, got %v", test.valid, err)
			}
		})
	}
}

func temporalCheckOutput(output []byte, selection string, failure bool) error {
	names := []string{strings.NewReplacer("^", "", "$", "").Replace(selection)}
	if selection == "." {
		names = []string{"TestDriverErrors", "TestDriverTimestampCompatibility", "TestDriverMemorySchema", "TestDriverTimestampPrecision", "TestQueryConverter_GetCoalesceCloseTimeExpr", "TestQueryConverter_ConvertKeywordListComparisonExpr", "TestQueryConverter_ConvertTextComparisonExpr", "TestQueryConverter_BuildSelectStmt", "TestQueryConverter_BuildCountStmt", "TestRewriteSchemaStatements_EnforcesVarcharLengthChecks"}
	}
	status := "PASS"
	if failure {
		status = "FAIL"
	}
	for _, name := range names {
		found, child := false, false
		for line := range strings.SplitSeq(string(output), "\n") {
			line = strings.TrimSpace(line)
			found = found || strings.HasPrefix(line, "--- "+status+": "+name+" (")
			child = child || strings.HasPrefix(line, "--- "+status+": "+name+"/")
		}
		if !found || (strings.HasSuffix(name, "Suite") && !child) {
			return fmt.Errorf("guest did not execute named %s outcome for %s", status, name)
		}
	}
	return nil
}

type temporalSQLiteProjection struct {
	Policy, Module, Version                                  string
	Original, OriginalSHA256, Replacement, ReplacementSHA256 string
	Overlay, OverlaySHA256                                   string
	PrivateRoot, ModFile, SumFile                            string
	OriginalInventory, PrivateInventory                      captured.CapturedInputsManifest
}

func temporalSQLiteAdapter(t *testing.T, evidence, root string, listed []byte) *temporalSQLiteProjection {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(listed))
	var projection *temporalSQLiteProjection
	for {
		var pkg struct {
			ImportPath, Dir string
			GoFiles         []string
			Module          *struct {
				Path, Version, GoMod string
				Replace              json.RawMessage
			}
		}
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if pkg.ImportPath != "github.com/ncruces/go-sqlite3" {
			continue
		}
		if projection != nil || pkg.Module == nil || !filepath.IsAbs(pkg.Dir) || !slices.Contains(pkg.GoFiles, "conn.go") {
			t.Fatal("SQLite adapter requires one selected pinned conn.go source and its module identity")
		}
		original := filepath.Join(pkg.Dir, "conn.go")
		source, err := os.ReadFile(original)
		if err != nil {
			t.Fatal(err)
		}
		dependency := sqlitebusy.Dependency{Path: pkg.Module.Path, Version: pkg.Module.Version, Replaced: len(pkg.Module.Replace) > 0 && string(pkg.Module.Replace) != "null"}
		adapter, err := sqlitebusy.Prepare(dependency, source)
		if err != nil {
			t.Fatal(err)
		}
		cachedMod, err := os.ReadFile(pkg.Module.GoMod)
		if err != nil {
			t.Fatal(err)
		}
		moduleMod, err := os.ReadFile(filepath.Join(pkg.Dir, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(cachedMod, moduleMod) {
			t.Fatal("cached SQLite module metadata differs from captured module go.mod")
		}
		t.Cleanup(func() {
			if fileHash(t, pkg.Module.GoMod) != temporalHash(cachedMod) {
				t.Fatal("original SQLite cached module metadata changed")
			}
		})
		captured := temporalCaptureModule(t, pkg.Dir)
		privateParent := filepath.Join(evidence, "private-modules")
		if err := os.MkdirAll(privateParent, 0700); err != nil {
			t.Fatal(err)
		}
		privateRoot := filepath.Join(privateParent, strings.TrimPrefix(string(captured.Manifest.SHA256), "sha256:"))
		if _, err := os.Stat(privateRoot); os.IsNotExist(err) {
			if err := temporalMaterializeModule(privateRoot, captured); err != nil {
				t.Fatal(err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
		privateCaptured := temporalCaptureModule(t, privateRoot)
		if !bytes.Equal(captured.Descriptor, privateCaptured.Descriptor) {
			t.Fatal("private SQLite module does not match original complete captured inventory")
		}
		verify := func() {
			for _, location := range []string{pkg.Dir, privateRoot} {
				if current := temporalCaptureModule(t, location); !bytes.Equal(captured.Descriptor, current.Descriptor) {
					t.Fatalf("SQLite module inventory changed during admission: %s", location)
				}
			}
		}
		t.Cleanup(verify)
		for _, name := range []string{"go.mod", "go.sum"} {
			contents, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			if name == "go.mod" {
				contents = append(contents, []byte("\nreplace github.com/ncruces/go-sqlite3 v0.35.6 => "+strconv.Quote(privateRoot)+"\n")...)
			}
			output := "temporal.mod"
			if name == "go.sum" {
				output = "temporal.sum"
			}
			temporalWrite(t, filepath.Join(evidence, output), contents)
		}
		temporalJSON(t, filepath.Join(evidence, "sqlite-original-capture.json"), captured)
		if fileHash(t, filepath.Join(privateRoot, "conn.go")) != adapter.OriginalSHA256 {
			t.Fatal("private SQLite source differs from admitted source")
		}
		replacement := filepath.Join(evidence, "sqlite-conn-cooperative.go")
		temporalWrite(t, replacement, adapter.Source)
		overlay := filepath.Join(evidence, "sqlite-overlay.json")
		temporalJSON(t, overlay, struct{ Replace map[string]string }{map[string]string{filepath.Join(privateRoot, "conn.go"): replacement}})
		projection = &temporalSQLiteProjection{Policy: adapter.Policy, Module: dependency.Path, Version: dependency.Version, Original: original, OriginalSHA256: adapter.OriginalSHA256, Replacement: replacement, ReplacementSHA256: adapter.ReplacementSHA256, Overlay: overlay, OverlaySHA256: fileHash(t, overlay), PrivateRoot: privateRoot, ModFile: filepath.Join(evidence, "temporal.mod"), SumFile: filepath.Join(evidence, "temporal.sum"), OriginalInventory: captured.Manifest, PrivateInventory: privateCaptured.Manifest}
	}
	if projection == nil {
		t.Fatal("SQLite adapter selected without the pinned SQLite dependency")
	}
	return projection
}

func temporalCaptureModule(t *testing.T, source string) captured.CapturedInputs {
	t.Helper()
	captured, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: source, Target: "/module"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	return captured
}

func temporalHash(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func TestTemporalPrivateModule(t *testing.T) {
	source := t.TempDir()
	if err := os.Mkdir(filepath.Join(source, "sub"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "sub", "source.go"), []byte("package fixture\n"), 0440); err != nil {
		t.Fatal(err)
	}
	inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: source, Target: "/module"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "module")
	if err := temporalMaterializeModule(destination, inputs); err != nil {
		t.Fatal(err)
	}
	copy, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: destination, Target: "/module"}}, captured.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(inputs.Descriptor, copy.Descriptor) {
		t.Fatal("private module changed complete bytes or modes")
	}
	if err := temporalMaterializeModule(destination, inputs); err == nil {
		t.Fatal("materializer must reject an existing root")
	}
	inputs.Descriptor = append(inputs.Descriptor, '\n')
	if err := temporalMaterializeModule(filepath.Join(t.TempDir(), "module"), inputs); err == nil {
		t.Fatal("materializer accepted unvalidated snapshot")
	}
}

func TestTemporalFixtureStaging(t *testing.T) {
	source := filepath.Join("..", "testdata", "sqlite_contention")
	original, err := os.ReadFile(filepath.Join(source, "contention_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if temporalHash(original) != "a9bfaf87df31ad4eb18abfccb4920ae0911c77851f34be02ab7f2ea4683a52ae" {
		t.Fatal("original exclusive fixture bytes changed")
	}
	destinationAnchor := t.TempDir()
	destination := filepath.Join(destinationAnchor, "parent", "fixture")
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"contention_test.go", "controls_test.go"} {
		expected, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(filepath.Join(destination, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Fatal("staged fixture body changed")
		}
	}
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(destination, "contention_test.go")
	if err := os.WriteFile(target, append(bytes.Clone(original), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err == nil {
		t.Fatal("staging accepted changed fixture hash")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(original, data) {
		t.Fatal("staging silently replaced changed source")
	}
	if err := os.WriteFile(target, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "extra_test.go"), []byte("package contention\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destinationAnchor, filepath.Join("parent", "fixture")); err == nil {
		t.Fatal("staging accepted an extra test body")
	}
}

func TestTemporalGRPCDiagnosticFixtureStaging(t *testing.T) {
	source := filepath.Join("..", "testdata", "environment")
	destination := t.TempDir()
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destination, "grpc-progress"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.go", "grpc_progress_test.go"} {
		expected, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "main.go" && temporalHash(expected) != "fde93d6955cafae1d1696d405eaa46987534c05cfef5dc2fe2a14483bacc98f3" {
			t.Fatal("original environment fixture bytes changed")
		}
		actual, err := os.ReadFile(filepath.Join(destination, "grpc-progress", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Fatalf("staged gRPC diagnostic changed %s", name)
		}
	}
}

func TestTemporalNamespaceStackFixtureStaging(t *testing.T) {
	source := filepath.Join("..", "testdata", "frontend_namespace")
	destination := t.TempDir()
	if err := temporalStageFixture(filepath.Dir(source), filepath.Base(source), destination, "namespace-stack"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"namespace_test.go", "namespace_stack_test.go"} {
		expected, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(filepath.Join(destination, "namespace-stack", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(expected, actual) {
			t.Fatalf("staged namespace stack diagnostic changed %s", name)
		}
	}
}

func TestTemporalFixtureStagingRejectsLinks(t *testing.T) {
	for _, name := range []string{"source-root", "source-parent", "destination-root", "destination-parent"} {
		t.Run(name, func(t *testing.T) {
			anchor, outside := t.TempDir(), t.TempDir()
			source, destination := filepath.Join(anchor, "source"), filepath.Join(anchor, "destination")
			owner := filepath.Join("..", "testdata", "sqlite_contention")
			if err := temporalStageFixture(filepath.Dir(owner), filepath.Base(owner), anchor, "source"); err != nil {
				t.Fatal(err)
			}
			outsideFixture := filepath.Join(outside, "fixture")
			if err := temporalStageFixture(filepath.Dir(owner), filepath.Base(owner), outside, "fixture"); err != nil {
				t.Fatal(err)
			}
			before, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: outside, Target: "/outside"}}, captured.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "source-root":
				source = filepath.Join(anchor, "source-link")
				if err := os.Symlink(outsideFixture, source); err != nil {
					t.Fatal(err)
				}
			case "source-parent":
				parent := filepath.Join(anchor, "source-parent")
				if err := os.Symlink(outside, parent); err != nil {
					t.Fatal(err)
				}
				source = filepath.Join(parent, "fixture")
			case "destination-root":
				if err := os.Symlink(outsideFixture, destination); err != nil {
					t.Fatal(err)
				}
			case "destination-parent":
				parent := filepath.Join(anchor, "destination-parent")
				if err := os.Symlink(outside, parent); err != nil {
					t.Fatal(err)
				}
				destination = filepath.Join(parent, "new-fixture")
			}
			sourceRelative, err := filepath.Rel(anchor, source)
			if err != nil {
				t.Fatal(err)
			}
			destinationRelative, err := filepath.Rel(anchor, destination)
			if err != nil {
				t.Fatal(err)
			}
			stageErr := temporalStageFixture(anchor, sourceRelative, anchor, destinationRelative)
			after, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: outside, Target: "/outside"}}, captured.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before.Descriptor, after.Descriptor) {
				t.Error("staging changed bytes or entries outside the trusted anchor")
			}
			if stageErr == nil {
				t.Error("staging accepted a linked fixture path")
			}
		})
	}
}

func TestTemporalFixtureStagingPolicy(t *testing.T) {
	owner := filepath.Join("..", "testdata")
	for _, relative := range []string{"../escape", "/absolute", "."} {
		t.Run(relative, func(t *testing.T) {
			anchor := t.TempDir()
			if err := temporalStageFixture(owner, "sqlite_contention", anchor, relative); err == nil {
				t.Fatal("accepted invalid destination")
			}
			if err := temporalStageFixture(owner, relative, anchor, "fixture"); err == nil {
				t.Fatal("accepted invalid source")
			}
			entries, err := os.ReadDir(anchor)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatal("invalid path admission changed the destination anchor")
			}
		})
	}
	anchor := t.TempDir()
	if err := os.MkdirAll(filepath.Join(anchor, "source", "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := temporalStageFixture(anchor, "source", anchor, "destination"); err == nil {
		t.Fatal("accepted a nonflat fixture owner")
	}
	if _, err := os.Lstat(filepath.Join(anchor, "destination")); !os.IsNotExist(err) {
		t.Fatal("rejected source created a destination")
	}
}

func TestTemporalModuleGraph(t *testing.T) {
	original := []byte(`{"Path":"main","Main":true,"GoMod":"/root/go.mod"}
{"Path":"github.com/ncruces/go-sqlite3","Version":"v0.35.6","Dir":"/cache/sqlite","GoMod":"/cache/sqlite/go.mod","Sum":"pinned","GoModSum":"pinned-mod"}
{"Path":"unrelated","Version":"v1.2.3","Sum":"unchanged"}`)
	effective := []byte(`{"Path":"main","Main":true,"GoMod":"/stage/temporal.mod"}
{"Path":"github.com/ncruces/go-sqlite3","Version":"v0.35.6","Dir":"/private/sqlite","GoMod":"/private/sqlite/go.mod","Replace":{"Path":"/private/sqlite","Dir":"/private/sqlite","GoMod":"/private/sqlite/go.mod"}}
{"Path":"unrelated","Version":"v1.2.3","Sum":"unchanged"}`)
	for _, test := range []struct {
		name                string
		original, effective []byte
		valid               bool
	}{
		{"owned", original, effective, true},
		{"extra-module", original, append(bytes.Clone(effective), []byte(`{"Path":"extra","Version":"v1"}`)...), false},
		{"version", original, bytes.Replace(effective, []byte("v1.2.3"), []byte("v1.2.4"), 1), false},
		{"preexisting", bytes.Replace(original, []byte(`"Sum":"pinned"`), []byte(`"Sum":"pinned","Replace":{"Path":"/user"}`), 1), effective, false},
		{"wrong-copy", original, bytes.ReplaceAll(effective, []byte("/private/sqlite"), []byte("/elsewhere")), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := temporalCompareModuleGraphs(test.original, test.effective, "/cache/sqlite", "/private/sqlite", "/stage/temporal.mod"); (err == nil) != test.valid {
				t.Fatalf("expected admission %v: %v", test.valid, err)
			}
		})
	}
}

func temporalMaterializeModule(destination string, captured captured.CapturedInputs) (retErr error) {
	snapshot, err := temporalModuleSnapshot(captured)
	if err != nil {
		return err
	}
	if err := os.Mkdir(destination, 0700); err != nil {
		return err
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	return temporalWriteModule(root, snapshot)
}

func temporalCompareModuleGraphs(original, effective []byte, originalRoot, privateRoot, modfile string) error {
	decode := func(data []byte) (map[string]map[string]any, error) {
		modules := map[string]map[string]any{}
		decoder := json.NewDecoder(bytes.NewReader(data))
		for {
			var module map[string]any
			if err := decoder.Decode(&module); err == io.EOF {
				return modules, nil
			} else if err != nil {
				return nil, err
			}
			name, valid := module["Path"].(string)
			if !valid || modules[name] != nil {
				return nil, fmt.Errorf("invalid or duplicate module graph identity")
			}
			modules[name] = module
		}
	}
	before, err := decode(original)
	if err != nil {
		return err
	}
	after, err := decode(effective)
	if err != nil {
		return err
	}
	if len(before) != len(after) {
		return fmt.Errorf("owned projection changed module graph size")
	}
	for name, module := range before {
		projected := after[name]
		if projected == nil {
			return fmt.Errorf("owned projection removed module %s", name)
		}
		if module["Main"] == true {
			if projected["GoMod"] != modfile {
				return fmt.Errorf("owned projection has wrong main modfile")
			}
			projected["GoMod"] = module["GoMod"]
		}
		if name == "github.com/ncruces/go-sqlite3" {
			if module["Replace"] != nil || module["Version"] != "v0.35.6" || module["Dir"] != originalRoot || !filepath.IsAbs(fmt.Sprint(module["GoMod"])) {
				return fmt.Errorf("owned projection requires original unreplaced pinned SQLite graph")
			}
			replacement, valid := projected["Replace"].(map[string]any)
			expected := map[string]any{"Path": privateRoot, "Dir": privateRoot, "GoMod": filepath.Join(privateRoot, "go.mod")}
			if module["GoVersion"] != nil {
				expected["GoVersion"] = module["GoVersion"]
			}
			if !valid || !reflect.DeepEqual(replacement, expected) || projected["Dir"] != privateRoot || projected["GoMod"] != filepath.Join(privateRoot, "go.mod") {
				return fmt.Errorf("SQLite replacement is not the exact owned projection")
			}
			delete(projected, "Replace")
			for _, key := range []string{"Dir", "GoMod", "Sum", "GoModSum", "Time"} {
				if key == "Sum" || key == "GoModSum" || key == "Time" {
					if projected[key] != nil {
						return fmt.Errorf("unexpected projected SQLite checksum metadata")
					}
				}
				if module[key] != nil {
					projected[key] = module[key]
				} else {
					delete(projected, key)
				}
			}
		}
		if !reflect.DeepEqual(module, projected) {
			return fmt.Errorf("owned projection changed unrelated graph metadata for %s: original=%v effective=%v", name, module, projected)
		}
	}
	if before["github.com/ncruces/go-sqlite3"] == nil {
		return fmt.Errorf("owned projection missing SQLite module")
	}
	return nil
}

const networkTemporalObservationPrefix = "GOMAD_NETWORK_TEMPORAL_OBSERVATION "

type networkTemporalValue struct {
	WorkflowID string   `json:"workflow_id"`
	RunID      string   `json:"run_id"`
	Result     string   `json:"result"`
	History    []byte   `json:"history"`
	EventIDs   []int64  `json:"event_ids"`
	Types      []string `json:"types"`
	Attempts   []struct {
		StartedNanos uint64 `json:"started_nanos"`
		EndedNanos   uint64 `json:"ended_nanos"`
		Code         string `json:"code"`
	} `json:"attempts"`
	Connections []struct {
		OpenedNanos uint64 `json:"opened_nanos"`
		LocalPort   int    `json:"local_port"`
		RemotePort  int    `json:"remote_port"`
	} `json:"connections"`
}

func networkTemporalValueFrom(t *testing.T, stdout []byte) networkTemporalValue {
	t.Helper()
	var value networkTemporalValue
	count := 0
	for _, line := range bytes.Split(stdout, []byte{'\n'}) {
		if !bytes.HasPrefix(line, []byte(networkTemporalObservationPrefix)) {
			continue
		}
		if err := json.Unmarshal(line[len(networkTemporalObservationPrefix):], &value); err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 1 || value.WorkflowID == "" || value.RunID == "" || len(value.History) == 0 || len(value.EventIDs) == 0 || len(value.EventIDs) != len(value.Types) || value.Result != "durable timer survived rpc-reset" {
		t.Fatalf("invalid independent Temporal history observation: count=%d value=%+v", count, value)
	}
	for index, id := range value.EventIDs {
		if id != int64(index+1) {
			t.Fatalf("duplicate or lost Temporal history: %v", value.EventIDs)
		}
	}
	return value
}

func networkTemporalResetWitness(t *testing.T, result Result, observation networkTemporalValue) {
	t.Helper()
	var transcript struct {
		Operations []struct {
			At      uint64           `json:"at"`
			Fault   *NetworkFaultV2  `json:"fault"`
			Request netmodel.Request `json:"request"`
			Result  netmodel.Result  `json:"result"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(result.NetworkTranscript, &transcript); err != nil {
		t.Fatal(err)
	}
	const resetAt = uint64(40 * time.Second)
	// Guest elapsed durations exclude the cooperative runtime's 1 ns clock origin.
	const clockOrigin = uint64(1)
	fault := false
	ports := map[int]bool{}
	for _, connection := range observation.Connections {
		if connection.OpenedNanos+clockOrigin < resetAt {
			ports[connection.LocalPort] = true
		}
	}
	if len(ports) == 0 {
		t.Fatal("no SDK connection existed before the named reset")
	}
	handles := map[uint64]bool{}
	pending := map[uint64]uint64{}
	resetError, reconnected := false, false
	for _, operation := range transcript.Operations {
		if operation.Fault != nil && operation.Fault.Name == "rpc-reset" && operation.Fault.Action == "reset" && operation.Fault.AtNanos == resetAt {
			fault = true
		}
		if operation.Request.Op == "dial" && operation.Result.State == "completed" && operation.Result.Error == "" && operation.Result.Handle != 0 {
			if operation.At < resetAt && ports[operation.Result.Local.Port] {
				handles[operation.Result.Handle] = true
			}
			for _, connection := range observation.Connections {
				if operation.At >= resetAt && connection.OpenedNanos+clockOrigin >= resetAt && operation.Result.Local.Port == connection.LocalPort {
					reconnected = true
				}
			}
		}
		if operation.Request.Op == "read" || operation.Request.Op == "write" {
			pending[operation.Request.RequestID] = operation.Request.Handle
		}
		handle := operation.Request.Handle
		if operation.Request.Op == "poll" {
			handle = pending[operation.Request.RequestID]
		}
		if operation.At >= resetAt && handles[handle] && (operation.Request.Op == "read" || operation.Request.Op == "write" || operation.Request.Op == "poll") && (operation.Result.Error == "use of closed network connection" || operation.Result.Error == "connection reset by peer") {
			resetError = true
		}
	}
	if !fault || !resetError || !reconnected {
		t.Fatalf("named reset lacks an established SDK stream error and fresh SDK connection: fault=%v stream_error=%v reconnected=%v", fault, resetError, reconnected)
	}
	failed := false
	retried := false
	for _, attempt := range observation.Attempts {
		if attempt.StartedNanos+clockOrigin < resetAt && attempt.EndedNanos+clockOrigin >= resetAt && attempt.Code == "Unavailable" {
			failed = true
		}
		if failed && attempt.StartedNanos+clockOrigin >= resetAt && attempt.Code == "OK" {
			retried = true
		}
	}
	if !retried {
		t.Fatalf("no in-flight history RPC failed across rpc-reset and then recovered: %+v", observation.Attempts)
	}
}

func TestTemporalNetworkNamedResetRetriesAndExactReplay(t *testing.T) {
	evidence := os.Getenv("GOMAD_WASM_NETWORK_TEMPORAL_ROOT")
	if evidence == "" {
		t.Skip("set GOMAD_WASM_NETWORK_TEMPORAL_ROOT to run the actual Temporal network retry gate")
	}
	if !filepath.IsAbs(evidence) {
		t.Fatal("GOMAD_WASM_NETWORK_TEMPORAL_ROOT must be absolute")
	}
	t.Setenv("GOMAD_WASM_PROTOTYPE_ROOT", evidence)
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	fixtureSource, err := os.ReadFile(filepath.Join(root, "tools", "gomad", "internal", "backends", "wasm", "testdata", "network_streams", "temporal_retry_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(".tmp", "wasm-backend-temporal", "network-streams-retry-"+digest(fixtureSource)[:12])
	if err := temporalStageFixture(root, filepath.Join("tools", "gomad", "internal", "backends", "wasm", "testdata", "network_streams"), root, staged); err != nil {
		t.Fatal(err)
	}
	module, _ := prototypeBuild(t, root, "test", "./"+filepath.ToSlash(staged), filepath.Join(prototypeRoot(t), "network-temporal-retry.wasm"), "test_dep", "disable_grpc_modules", "sqlite3_dotlk")
	config := prototypeTemporalConfig(t, root, "/workspace/network-temporal", "-test.run=^TestTemporalWorkflowRecoversNamedNetworkReset$", "-persistenceType=sql", "-persistenceDriver=sqlite")
	config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile,
		Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20},
		Faults: []NetworkFaultV2{{Name: "rpc-reset", Action: "reset", AtNanos: uint64(40 * time.Second)}},
	}
	request := prototypeRequest(t, module, config, 7, 15*time.Minute)
	request.DigestTranscript = false
	observed, err := Run(t.Context(), request)
	retain := func(label string, result Result) {
		for suffix, data := range map[string][]byte{"stdout": result.Stdout, "stderr": result.Stderr, "wasi.bin": result.Transcript, "network.json": result.NetworkTranscript, "choices": result.ChoiceTrace.Bytes} {
			temporalWrite(t, filepath.Join(evidence, "network-temporal-"+label+"."+suffix), data)
		}
		temporalJSON(t, filepath.Join(evidence, "network-temporal-"+label+".terminal.json"), map[string]any{
			"termination": result.Termination, "message": result.Message, "exit_code": result.ExitCode, "reaped": result.Reaped,
			"environment_sha256": result.EnvironmentIdentity, "model_sha256": ImplementationSHA256(), "network_model_sha256": netmodel.ImplementationSHA256(),
			"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_sha256": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey,
			"clock": request.Config.Clock, "network": request.Config.NetworkV2,
		})
	}
	retain("record", observed)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "Temporal named network reset recording", observed, 0)
	value := networkTemporalValueFrom(t, observed.Stdout)
	networkTemporalResetWitness(t, observed, value)
	tape, err := choice.ProjectReplayPlan(observed.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	setChoiceReplay(t, request.Runtime.Choice, choice.ModeReplay, observed.ChoiceTrace.Bytes, tape)
	replayed, err := Run(t.Context(), request)
	retain("replay", replayed)
	if err != nil {
		t.Fatal(err)
	}
	requireExit(t, "Temporal named network reset replay", replayed, 0)
	if mismatch := sameEvidence(observed, replayed); mismatch != "" {
		t.Fatalf("Temporal named-reset replay differs: %s", mismatch)
	}
	if !bytes.Equal(observed.Transcript, replayed.Transcript) || !reflect.DeepEqual(value, networkTemporalValueFrom(t, replayed.Stdout)) {
		t.Fatal("exact replay changed I/O or independent workflow history")
	}
	temporalJSON(t, filepath.Join(evidence, "network-temporal-history-oracle.json"), value)
	temporalJSON(t, filepath.Join(evidence, "network-temporal-retry-report.json"), map[string]any{
		"schema": "gomad.wasm-temporal-network-retry/v1", "exact_replay": true,
		"model_sha256": ImplementationSHA256(), "network_model_sha256": netmodel.ImplementationSHA256(),
		"module_sha256": request.ModuleSHA256, "helper_sha256": request.HelperSHA256, "runtime_sha256": request.Runtime.Choice.ExecutionIdentity.ToolchainBuildKey,
		"environment_sha256": observed.EnvironmentIdentity, "seed": request.Runtime.Seed, "clock": request.Config.Clock, "network": request.Config.NetworkV2,
		"wasi_sha256": digest(observed.Transcript), "choices_sha256": digest(observed.ChoiceTrace.Bytes), "network_sha256": digest(observed.NetworkTranscript),
		"history_sha256": digest(value.History), "history": value, "termination": observed.Termination, "exit_code": observed.ExitCode,
	})
	t.Logf("Temporal history RPC retried after named reset; events=%d attempts=%d connections=%d choices=%d", len(value.EventIDs), len(value.Attempts), len(value.Connections), len(observed.ChoiceTrace.Records))
}

func TestTemporalHostDiagnosticInputs(t *testing.T) {
	for _, invalid := range []string{"not-a-duration", "0s", "-1s", "16m"} {
		if _, err := temporalPairBudget(invalid); err == nil {
			t.Fatalf("accepted invalid diagnostic budget %q", invalid)
		}
	}
	for _, value := range []struct {
		input string
		want  time.Duration
	}{{"", 15 * time.Minute}, {"3m", 3 * time.Minute}} {
		actual, err := temporalPairBudget(value.input)
		if err != nil || actual != value.want {
			t.Fatalf("budget %q: got %v, %v", value.input, actual, err)
		}
	}
	root := t.TempDir()
	if actual, err := temporalCompileCachePath(root, ""); err != nil || actual != filepath.Join(root, "compile-cache") {
		t.Fatalf("default cache changed: %s, %v", actual, err)
	}
	for _, invalid := range []string{"relative/cache", string(filepath.Separator), root + "/../"} {
		if _, err := temporalCompileCachePath(root, invalid); err == nil {
			t.Fatalf("accepted unsafe cache path %q", invalid)
		}
	}
	cache := filepath.Join(root, "owned-cache")
	if actual, err := temporalCompileCachePath(root, cache); err != nil || actual != cache {
		t.Fatalf("explicit cache changed: %s, %v", actual, err)
	}
}

func TestTemporalBrokerBoundedWorldAdmission(t *testing.T) {
	config := temporalBrokerConfiguration(&NetworkModelV2{Profile: NetworkStreamProfile,
		Limits: netmodel.Limits{Listeners: 128, Connections: 4096, Bytes: 128 << 20, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 1 << 20}})
	if config.WorldProfile != world.CompressedSnapshotSchema || config.Limits != world.CompressedSnapshotProfile().Limits {
		t.Fatal("Temporal producer lost its explicit bounded World profile")
	}
	if _, err := NewBroker(config); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*BrokerConfig){
		func(c *BrokerConfig) { c.Limits.MaxRequests++ },
		func(c *BrokerConfig) { c.Limits.MaxEvents++ },
		func(c *BrokerConfig) { c.Limits.MaxQueuedEvents++ },
		func(c *BrokerConfig) { c.Limits.MaxTransitions++ },
		func(c *BrokerConfig) { c.Limits.MaxPayloadBytes++ },
		func(c *BrokerConfig) { c.Limits.MaxStringBytes++ },
		func(c *BrokerConfig) { c.Limits.MaxRequests = 0 },
		func(c *BrokerConfig) { c.WorldProfile = "unknown" },
	} {
		invalid := config
		mutate(&invalid)
		if _, err := NewBroker(invalid); err == nil {
			t.Fatal("Temporal producer admitted an oversized, implicit or unknown World profile")
		}
	}
}

func TestTemporalBrokerReadinessSource(t *testing.T) {
	compiler := os.Getenv("GOMAD_STOCK_GO")
	if !filepath.IsAbs(compiler) {
		t.Skip("set absolute GOMAD_STOCK_GO for the ordinary fixture-source readiness unit test")
	}
	root, err := filepath.Abs("../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	var body []byte
	for _, name := range []string{"temporal_storage/recovery_test.go", "temporal_broker/replication_test.go"} {
		data, err := os.ReadFile(filepath.Join(root, "tools/gomad/internal/backends/wasm/testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		body = append(body, data...)
	}
	staged := filepath.Join(".tmp", "wasm-backend-temporal", "temporal_broker-"+digest(body)[:16])
	temporalStageBrokerFixture(t, root, staged)
	environment := append(os.Environ(), "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0", "GOENV=off", "GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local", "GOROOT="+filepath.Dir(filepath.Dir(compiler)))
	temporalCommand(t, t.TempDir(), "readiness-source", root, environment, []string{compiler, "test", "-mod=readonly", "-tags", "test_dep,disable_grpc_modules,sqlite3_dotlk", "-run", "^(TestReplicationReadinessRequiresRemoteAcknowledgment|TestReplicatedNamespaceReadiness)$", "-count=1", "./" + filepath.ToSlash(staged)})
}

func TestTemporalIndependentClustersPartitionHealWholeClusterCrash(t *testing.T) {
	request, evidence := temporalRecoveryRequest(t, "temporal_broker")
	temporalIndependentClustersPartitionHealWholeClusterCrash(t, request, evidence)
}

func TestHelperBuildPublishesConfiguredTarget(t *testing.T) {
	root := t.TempDir()
	makefile, err := os.ReadFile("../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), makefile, 0600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	target := filepath.Join(root, "cargo-target")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "release"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "cargo"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	compiled := []byte("current configured helper")
	if err := os.WriteFile(filepath.Join(target, "release/gomad-wasmhost"), compiled, 0700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("make", "-C", root, "helper")
	command.Env = append(os.Environ(), "CARGO_TARGET_DIR="+target, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("helper build: %v\n%s", err, output)
	}
	installed, err := os.ReadFile(filepath.Join(root, "wasmhost/target/release/gomad-wasmhost"))
	if err != nil || !bytes.Equal(installed, compiled) {
		t.Fatalf("configured Cargo target was not published to guest execution path: %q %v", installed, err)
	}
}

// runStorageOwners runs one storage request through the Go-owner reference
// and the helper-owned volume and requires identical evidence.
func runStorageOwners(t *testing.T, label string, request Request, referenceImage ...*StorageImage) (Result, Result) {
	t.Helper()
	remote := request
	remote.remoteModel = true
	if len(referenceImage) != 0 {
		// The Go owner recovers only its own (Go volume) images.
		remote.StorageImage = referenceImage[0]
	}
	reference := runStorageGuest(t, remote)
	local := runStorageGuest(t, request)
	if local.GoFilesystemCalls == nil || len(local.GoFilesystemCalls) != 0 {
		t.Fatalf("%s: helper storage run called Go filesystem: %v", label, local.GoFilesystemCalls)
	}
	if reference.GoFilesystemCalls["fd_pwrite"] == 0 {
		t.Fatalf("%s: Go reference did not serve storage operations: %v", label, reference.GoFilesystemCalls)
	}
	if diff := sameStorageEvidence(reference, local); diff != "" {
		t.Fatalf("%s: helper storage evidence differs in %s: Go=%s %q Rust=%s %q", label, diff, reference.Termination, reference.Message, local.Termination, local.Message)
	}
	return reference, local
}

func TestStorageOwnerParity(t *testing.T) {
	base := storageSQLiteRequest(t)
	model := *base.Config.Storage
	model.SQLiteLockDatabases = []string{"/volume/db"}
	model.Faults = []StorageFault{{Name: "acknowledged-crash", Resource: "/volume/crash-trigger", Operation: "open", Occurrence: 1, Action: "crash-before"}}
	crashing := base
	crashing.Config.Storage = &model
	setupReference, setup := runStorageOwners(t, "setup", storageGuestOperation(crashing, "setup-live", "WAL", nil))
	requireStorageGuestExit(t, setup)
	crash := storageGuestOperation(crashing, "append-crash", "WAL", setup.StorageImage)
	crash.StorageCrashLimits = &volume.CrashEnumerationLimits{States: 32, Operations: 8, Depth: 8, Bytes: 4 << 20, Work: 1 << 16}
	crashReference, crashed := runStorageOwners(t, "crash-before", crash, setupReference.StorageImage)
	if crashed.Termination != "storage_crash" || !crashed.RuntimeInterrupted || string(crashed.Stdout) != "ACK 2\n" {
		t.Fatalf("helper-originated storage crash: %s %s %s", crashed.Termination, crashed.Message, crashed.Stdout)
	}
	// The Go reference recovers the Go image; the helper recovers the blob.
	recoverRequest := storageGuestOperation(crashing, "recover", "WAL", nil)
	referenceRecovery := recoverRequest
	referenceRecovery.remoteModel = true
	referenceRecovery.StorageImage = crashReference.StorageImage
	reopenedReference := runStorageGuest(t, referenceRecovery)
	localRecovery := recoverRequest
	localRecovery.StorageImage = crashed.StorageImage
	reopened := runStorageGuest(t, localRecovery)
	requireStorageGuestExit(t, reopened)
	if reopened.GoFilesystemCalls == nil || len(reopened.GoFilesystemCalls) != 0 {
		t.Fatalf("helper recovery called Go filesystem: %v", reopened.GoFilesystemCalls)
	}
	if diff := sameStorageEvidence(reopenedReference, reopened); diff != "" || string(reopened.Stdout) != "INTEGRITY ok\nHISTORY 1:committed,2:committed\n" {
		t.Fatalf("reopen from helper image differs in %s: %s", diff, reopened.Stdout)
	}
	// A Go-owner image carries the same blob for a helper-owned recovery.
	crossRecovery := recoverRequest
	crossRecovery.StorageImage = crashReference.StorageImage
	if crossed := runStorageGuest(t, crossRecovery); sameStorageEvidence(reopenedReference, crossed) != "" {
		t.Fatal("helper recovery from the Go-owner image differs")
	}
	if setupReference.StorageImage.Identity() != setup.StorageImage.Identity() {
		t.Fatal("setup image identity differs")
	}
	// A broker CrashOnOutput boundary on the acknowledgement: the Go-owned
	// broker path and split ownership crash at the same durable state, and
	// both images reopen to the same history (fn-176.6).
	brokerConfig := brokerTestConfig()
	brokerConfig.Nodes = brokerConfig.Nodes[:1]
	var brokerCrashes [2]Result
	for i, image := range []*StorageImage{setupReference.StorageImage, setup.StorageImage} {
		b, err := NewBroker(brokerConfig)
		if err != nil {
			t.Fatal(err)
		}
		appendRequest := storageGuestOperation(crashing, "append-live", "WAL", image)
		appendRequest.remoteModel = i == 0
		appendRequest.Broker = b.Session("a")
		if err := appendRequest.Broker.CrashOnOutput([]byte("ACK 2")); err != nil {
			t.Fatal(err)
		}
		brokerCrashes[i] = runStorageGuest(t, appendRequest)
		if brokerCrashes[i].Termination != "crash" || !brokerCrashes[i].RuntimeInterrupted {
			t.Fatalf("broker output crash (Go owner=%v): %s %s %s", i == 0, brokerCrashes[i].Termination, brokerCrashes[i].Message, brokerCrashes[i].Stdout)
		}
	}
	if diff := sameStorageEvidence(brokerCrashes[0], brokerCrashes[1]); diff != "" {
		t.Fatalf("Go-owned vs split SQLite output crash differs in %s", diff)
	}
	requireSplitServedOnlyOutput(t, "split SQLite output crash", brokerCrashes[1])
	referenceReopen := recoverRequest
	referenceReopen.remoteModel = true
	referenceReopen.StorageImage = brokerCrashes[0].StorageImage
	splitReopen := recoverRequest
	splitReopen.StorageImage = brokerCrashes[1].StorageImage
	if left, right := runStorageGuest(t, referenceReopen), runStorageGuest(t, splitReopen); sameStorageEvidence(left, right) != "" || string(right.Stdout) != "INTEGRITY ok\nHISTORY 1:committed,2:committed\n" {
		t.Fatalf("SQLite output-crash reopen differs: %s / %s", left.Stdout, right.Stdout)
	}
	// Under a broker, a crash-before storage fault reached first is a
	// helper-originated storage_crash in split ownership and a Go interrupt in
	// the Go-owned path; a broker revocation reached first (an output boundary
	// before the faulted open) wins in both. Each order keeps owner parity.
	for _, revocationFirst := range []bool{false, true} {
		var ordered [2]Result
		for i, image := range []*StorageImage{setupReference.StorageImage, setup.StorageImage} {
			b, err := NewBroker(brokerConfig)
			if err != nil {
				t.Fatal(err)
			}
			crashRun := storageGuestOperation(crashing, "append-crash", "WAL", image)
			crashRun.remoteModel = i == 0
			crashRun.Broker = b.Session("a")
			if revocationFirst {
				if err := crashRun.Broker.CrashOnOutput([]byte("ACK 2")); err != nil {
					t.Fatal(err)
				}
			}
			ordered[i] = runStorageGuest(t, crashRun)
		}
		want := "storage_crash"
		if revocationFirst {
			want = "crash"
		}
		if ordered[0].Termination != want || ordered[1].Termination != want || !ordered[1].RuntimeInterrupted {
			t.Fatalf("revocation first=%v: Go=%s %q split=%s %q", revocationFirst, ordered[0].Termination, ordered[0].Message, ordered[1].Termination, ordered[1].Message)
		}
		if diff := sameStorageEvidence(ordered[0], ordered[1]); diff != "" {
			t.Fatalf("revocation first=%v: Go-owned vs split differs in %s", revocationFirst, diff)
		}
		requireSplitServedOnlyOutput(t, fmt.Sprintf("split revocation first=%v", revocationFirst), ordered[1])
	}
	// Faults land on the append run's occurrences, which continue from setup.
	plainSetup := runStorageGuest(t, storageGuestOperation(base, "setup", "WAL", nil))
	requireStorageGuestExit(t, plainSetup)
	plainAppend := runStorageGuest(t, storageGuestOperation(base, "append", "WAL", plainSetup.StorageImage))
	requireStorageGuestExit(t, plainAppend)
	events := storageGuestEvents(t, plainAppend)
	for _, action := range []string{"io-error", "short-write", "sync-error"} {
		t.Run(action, func(t *testing.T) {
			operation := "write"
			if action == "sync-error" {
				operation = "sync"
			}
			index := slices.IndexFunc(events, func(event StorageEvent) bool {
				return event.Resource == "/volume/db-wal" && event.Operation == operation
			})
			if index < 0 {
				t.Fatalf("append run has no WAL %s", operation)
			}
			fault := StorageFault{Name: "parity-" + action, Resource: "/volume/db-wal", Operation: operation, Occurrence: events[index].Occurrence, Action: action}
			if action == "short-write" {
				fault.Bytes = 1
			}
			faulty := base
			faultModel := *base.Config.Storage
			faultModel.Faults = []StorageFault{fault}
			faulty.Config.Storage = &faultModel
			setup := storageGuestOperation(faulty, "setup", "WAL", nil)
			reference := setup
			reference.remoteModel = true
			referenceInitialized := runStorageGuest(t, reference)
			initialized := runStorageGuest(t, setup)
			requireStorageGuestExit(t, initialized)
			failedReference, failed := runStorageOwners(t, action, storageGuestOperation(faulty, "append", "WAL", initialized.StorageImage), referenceInitialized.StorageImage)
			if !bytes.Contains(failed.StorageTranscript, []byte(`"fault":"parity-`+action+`"`)) {
				t.Fatalf("fault %s not realized", action)
			}
			history := "1:committed"
			if action == "short-write" {
				history += ",2:committed"
			}
			recovery := storageGuestOperation(faulty, "recover", "WAL", nil)
			recovered := requireStorageHistory(t, recovery, failed.StorageImage, history)
			recovery.remoteModel = true
			recoveredReference := requireStorageHistory(t, recovery, failedReference.StorageImage, history)
			if diff := sameStorageEvidence(recoveredReference, recovered); diff != "" {
				t.Fatalf("%s recovery differs in %s", action, diff)
			}
			if recovered.GoFilesystemCalls == nil || len(recovered.GoFilesystemCalls) != 0 {
				t.Fatalf("%s recovery called Go filesystem: %v", action, recovered.GoFilesystemCalls)
			}
		})
	}
}

func TestStorageHelperRefusesInvalidRecoveryBeforeStart(t *testing.T) {
	base := storageSQLiteRequest(t)
	setup := runStorageGuest(t, storageGuestOperation(base, "setup", "WAL", nil))
	requireStorageGuestExit(t, setup)
	tampered := *setup.StorageImage
	tampered.fs = nil
	tampered.data = append([]byte(nil), setup.StorageImage.data...)
	tampered.data[len(tampered.data)-1] ^= 1
	request := storageGuestOperation(base, "recover", "WAL", &tampered)
	result, err := Run(t.Context(), request)
	if err != nil || result.Termination != "invalid" || len(result.Stdout) != 0 || result.Calls != 0 {
		t.Fatalf("tampered recovery image: %s %q calls=%d err=%v", result.Termination, result.Message, result.Calls, err)
	}
	other := base
	otherModel := *base.Config.Storage
	otherModel.Volumes = append([]volume.VolumeConfig(nil), otherModel.Volumes...)
	otherModel.Volumes[0].CapacityBytes--
	other.Config.Storage = &otherModel
	if _, err := Run(t.Context(), storageGuestOperation(other, "recover", "WAL", setup.StorageImage)); err == nil {
		t.Fatal("recovery image of another storage model admitted")
	}
}

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
	if string(result.HelperStderr()) != "diagnostic helper bytes\n" {
		t.Fatalf("helper diagnostic capture = %q", result.HelperStderr())
	}
	if len(result.Stdout) != 0 || len(result.Stderr) != 0 || len(result.Transcript) != 0 || result.Calls != 0 {
		t.Fatalf("helper diagnostics entered guest evidence: stdout=%q stderr=%q transcript=%q calls=%d", result.Stdout, result.Stderr, result.Transcript, result.Calls)
	}
}

// withoutCreationMetadata drops the v4 creation records of the decision at
// target and renumbers the trace, returning the decision's new index.
func withoutCreationMetadata(records []choice.Record, target int) ([]choice.Record, int) {
	start := target
	for start > 0 && records[start-1].Kind == choice.KindGoroutineCreation {
		start--
	}
	removed := uint64(target - start)
	records = slices.Delete(records, start, target)
	for index := range records {
		records[index].Ordinal = uint64(index)
		if records[index].Kind == choice.KindSelectResult && records[index].Origin >= uint64(target) {
			records[index].Origin -= removed
		}
	}
	return records, start
}
