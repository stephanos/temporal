package simulation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	wasmbackend "go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/model/world"
)

func softAssertionGuestRequest(t *testing.T) wasi.Request {
	t.Helper()
	request := guestRequest(t)
	root := stockgo.Root(t)
	runtimeRoot, err := filepath.Abs("../../../model/runtimesource")
	if err != nil {
		t.Fatal(err)
	}
	overlay, _, err := toolchain.BuildOverlay(root, runtimeRoot, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "soft-assertion.wasm")
	cmd := exec.Command(filepath.Join(root, "bin/go"), "test", "-tags=test_dep", "-c", "-trimpath", "-overlay", overlay, "-o", module, "./tests/gomadfunctional/testdata/softassert")
	cmd.Dir = "../../../../../.."
	cmd.Env = append(os.Environ(), "GOROOT="+root, "GOOS=wasip1", "GOARCH=wasm", "GOEXPERIMENT=nogreenteagc", "GOWORK=off", "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile assertion guest: %v %s", err, output)
	}
	data, err := os.ReadFile(module)
	if err != nil {
		t.Fatal(err)
	}
	imports, pages, err := wasmbackend.InspectModule(data)
	if err != nil {
		t.Fatal(err)
	}
	request.ModulePath, request.ModuleSHA256 = module, fmt.Sprintf("%x", sha256.Sum256(data))
	request.Imports, request.InitialMemoryPages = imports, pages
	request.Runtime.Choice.ExecutionIdentity.TargetSHA256 = sha256.Sum256(data)
	request.WorkloadKind = target.KindGoTest
	request.Timeout = 5 * time.Minute
	request.CompileCache = t.TempDir()
	if err := os.Chmod(request.CompileCache, 0700); err != nil {
		t.Fatal(err)
	}
	return request
}

func TestSoftAssertionActualClusterGuestAndRetainedReplay(t *testing.T) {
	base := softAssertionGuestRequest(t)
	module := retainedTargetModule(t, base)
	handles := []gomad3sim.NodeHandle{{Node: "a", Incarnation: 1}, {Node: "b", Incarnation: 1}}
	for _, reportOnly := range []bool{false, true} {
		var signature string
		seeds := []uint64{7, 19}
		if reportOnly {
			seeds = seeds[:1]
		}
		for _, seed := range seeds {
			request := base
			runtime := *base.Runtime
			runtime.Seed = seed
			request.Runtime = &runtime
			request.Config.Args = []string{"soft-assertion", "-test.v", "-assertion-mode=late"}
			boot, err := BootConfiguration(request)
			if err != nil {
				t.Fatal(err)
			}
			spec := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelitySimulationModel, Seed: seed,
				Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "fixture", Address: "10.0.0.1", Config: boot}, {ID: "b", Boot: "fixture", Address: "10.0.0.2", Config: boot}}}
			p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"fixture": request}}
			if reportOnly {
				p.Oracles = &OraclePlan{Schema: SoftAssertionObservationSchema, MaximumBytes: 4096, SoftAssertions: []SoftAssertionPolicy{{Handle: handles[0], ReportOnly: true}, {Handle: handles[1], ReportOnly: true}}}
			}
			result, err := gomad3sim.RunWASM(t.Context(), spec, func(ctx context.Context, c gomad3sim.Cluster) error {
				for _, handle := range handles {
					if _, err := c.Start(ctx, handle.Node); err != nil {
						return err
					}
					if _, err := c.Wait(ctx, handle); err != nil {
						return err
					}
				}
				return nil
			}, p)
			if err != nil {
				for _, completed := range p.CompletedResults() {
					t.Logf("guest termination=%s message=%s stdout=%s stderr=%s", completed.Result.Termination, completed.Result.Message, completed.Result.Stdout, completed.Result.Stderr)
				}
				t.Fatal(err)
			}
			want := gomad3sim.OutcomeOracleFailed
			if reportOnly {
				want = gomad3sim.OutcomeCompleted
			}
			if result.Outcome != want || len(result.Oracles) != 2 {
				t.Fatalf("assertion cluster outcome: %+v", result)
			}
			for _, oracle := range result.Oracles {
				if !oracle.SoftAssertion || oracle.ReportOnly != reportOnly || oracle.Passed != reportOnly || !bytes.Equal(oracle.Evidence[1].Value, []byte(`["invalid fixture state"]`)) {
					t.Fatalf("assertion oracle: %+v", oracle)
				}
			}
			if !reportOnly {
				if result.Reason != "soft_assertion" || signature != "" && result.FailureIdentity != signature {
					t.Fatalf("seed changed static assertion signature: %s/%s", signature, result.FailureIdentity)
				}
				signature = result.FailureIdentity
			}
			for _, observed := range p.CompletedResults() {
				if observed.Result.ExitCode == nil || *observed.Result.ExitCode != 0 || !strings.Contains(string(observed.Result.Stdout), "PASS") {
					t.Fatalf("fixture did not pass: %+v", observed)
				}
			}
			payload, err := p.RetentionPayload(spec, result.Record)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := VerifyRetained(payload); err != nil {
				t.Fatal(err)
			}
			if seed != 7 {
				continue
			}
			replayed, err := ReplayRetained(t.Context(), payload, module, t.TempDir(), installedHelper(base))
			if err != nil {
				t.Fatal(err)
			}
			left, err := gomad3sim.EncodeClusterRecord(result.Record)
			if err != nil {
				t.Fatal(err)
			}
			right, err := gomad3sim.EncodeClusterRecord(replayed.Record)
			if err != nil || !bytes.Equal(left, right) {
				t.Logf("original outcome=%s reason=%s failure=%s replay outcome=%s reason=%s failure=%s divergence=%+v", result.Outcome, result.Reason, result.FailureIdentity, replayed.Outcome, replayed.Reason, replayed.FailureIdentity, replayed.Divergence)
				t.Fatalf("assertion replay changed static signature or evidence: %v", err)
			}
			t.Logf("seed=%d report_only=%t failure=%s retained=%s replay=%s", seed, reportOnly, result.FailureIdentity, payload.Reference.SHA256, replayed.FailureIdentity)
		}
	}
}

func TestSoftAssertionIncompleteClusterEvidenceIsInconclusive(t *testing.T) {
	handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
	for _, reportOnly := range []bool{false, true} {
		for _, result := range []wasi.Result{{Reaped: true, Stdout: []byte("PASS\n"), OfferedOutputBytes: 100}, {Reaped: true, Stdout: []byte{0xff}, OfferedOutputBytes: 1}} {
			e := &execution{provider: Provider{Oracles: &OraclePlan{MaximumBytes: 4096, SoftAssertions: []SoftAssertionPolicy{{Handle: handle, ReportOnly: reportOnly}}}}, limits: gomad3sim.DefaultLimits(), results: map[gomad3sim.NodeHandle]wasi.Result{handle: result}}
			oracles, err := e.softAssertionOracles()
			if err != nil || len(oracles) != 1 || !oracles[0].Inconclusive || oracles[0].Passed {
				t.Fatalf("incomplete evidence passed: %+v %v", oracles, err)
			}
		}
	}
}

func TestSoftAssertionClusterOptOutRequiresRecordedTestWorkload(t *testing.T) {
	handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
	spec := gomad3sim.Spec{Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelitySimulationModel, Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "fixture"}}}
	p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"fixture": {WorkloadKind: target.KindGoRun}}, Oracles: &OraclePlan{Schema: SoftAssertionObservationSchema, MaximumBytes: 4096, SoftAssertions: []SoftAssertionPolicy{{Handle: handle, ReportOnly: true}}}}
	if _, err := p.Open(t.Context(), spec); err == nil || !strings.Contains(err.Error(), "report-only mode requires") {
		t.Fatalf("unsupported workload opt-out admitted: %v", err)
	}
	p.Boots["fixture"] = wasi.Request{WorkloadKind: target.KindGoTest}
	if err := p.validateSoftAssertionWorkloads(spec); err != nil {
		t.Fatal(err)
	}
}

func TestSoftAssertionRetainedEvidenceRequiresActualOutput(t *testing.T) {
	handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
	data := []byte("PASS\nfailed assertion: invalid state\n")
	output := gomad3sim.OutputObservation{Handle: handle, Stream: gomad3sim.OutputStderr, Bytes: data, TotalBytes: uint64(len(data)), RetainedBytes: uint64(len(data)), FullSHA256: string(record.HashBytes(data))}
	oracle, err := gomad3sim.SoftAssertion("a/1", []string{"invalid state"}, false, false, 4096)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"valid", "forged-message", "forged-instance", "missing-output", "forged-digest", "generic", "mixed", "truncated-pass"} {
		t.Run(change, func(t *testing.T) {
			plan := gomad3sim.ReplayPlan{WASM: &gomad3sim.WASMExecutionEvidence{Incarnations: []gomad3sim.WASMIncarnationEvidence{{Handle: handle}}}, Outputs: []gomad3sim.OutputObservation{output}, Oracles: []gomad3sim.OracleResult{oracle}}
			switch change {
			case "forged-message":
				plan.Oracles[0], err = gomad3sim.SoftAssertion("a/1", []string{"forged invariant"}, false, false, 4096)
			case "forged-instance":
				plan.Oracles[0], err = gomad3sim.SoftAssertion("b/1", []string{"invalid state"}, false, false, 4096)
			case "missing-output":
				plan.Outputs = nil
			case "forged-digest":
				plan.Outputs[0].FullSHA256 = string(record.HashBytes([]byte("forged output")))
			case "generic", "mixed":
				generic, genericErr := gomad3sim.StateInvariant("generic", false, nil, 4096)
				if genericErr != nil {
					t.Fatal(genericErr)
				}
				plan.Oracles = []gomad3sim.OracleResult{generic}
				if change == "mixed" {
					plan.Oracles = append(plan.Oracles, oracle)
				}
			case "truncated-pass":
				plan.Outputs[0].Truncated = true
				plan.Outputs[0].TotalBytes++
			}
			if err != nil {
				t.Fatal(err)
			}
			err := validateRetainedLifecycle(gomad3sim.Spec{Limits: gomad3sim.DefaultLimits(), Replay: &plan}, &Provider{})
			if change == "valid" && err != nil || change != "valid" && err == nil {
				t.Fatalf("retained observation admission %s: %v", change, err)
			}
		})
	}
}

func TestSoftAssertionRetainedTruncatedOutputIsInconclusive(t *testing.T) {
	handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
	oracle, err := gomad3sim.SoftAssertion("a/1", nil, true, false, 4096)
	if err != nil {
		t.Fatal(err)
	}
	plan := gomad3sim.ReplayPlan{WASM: &gomad3sim.WASMExecutionEvidence{Incarnations: []gomad3sim.WASMIncarnationEvidence{{Handle: handle}}},
		Outputs: []gomad3sim.OutputObservation{{Handle: handle, Stream: gomad3sim.OutputStderr, TotalBytes: 100, Truncated: true}}, Oracles: []gomad3sim.OracleResult{oracle}}
	if err := validateRetainedLifecycle(gomad3sim.Spec{Limits: gomad3sim.DefaultLimits(), Replay: &plan}, &Provider{}); err != nil {
		t.Fatal(err)
	}
	if oracle.Passed || !oracle.Inconclusive {
		t.Fatal("truncated retained evidence passed")
	}
}

func TestSoftAssertionRetainedLiteralBracesRederiveCompleteMessage(t *testing.T) {
	handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
	for _, fixture := range []struct{ output, message string }{
		{`failed assertion: invalid {"left":1}`, `invalid {"left":1}`},
		{"2000-01-01\terror\tfailed assertion: invalid {state}\t{\"failed-assertion\":true}", "invalid {state}"},
	} {
		data := []byte(fixture.output)
		output := gomad3sim.OutputObservation{Handle: handle, Stream: gomad3sim.OutputStderr, Bytes: data, TotalBytes: uint64(len(data)), RetainedBytes: uint64(len(data)), FullSHA256: string(record.HashBytes(data))}
		for _, message := range []string{fixture.message, "invalid"} {
			oracle, err := gomad3sim.SoftAssertion("a/1", []string{message}, false, false, 4096)
			if err != nil {
				t.Fatal(err)
			}
			plan := gomad3sim.ReplayPlan{WASM: &gomad3sim.WASMExecutionEvidence{Incarnations: []gomad3sim.WASMIncarnationEvidence{{Handle: handle}}}, Outputs: []gomad3sim.OutputObservation{output}, Oracles: []gomad3sim.OracleResult{oracle}}
			err = validateRetainedLifecycle(gomad3sim.Spec{Limits: gomad3sim.DefaultLimits(), Replay: &plan}, &Provider{})
			if message == fixture.message && err != nil || message != fixture.message && err == nil {
				t.Fatalf("literal %q retained as %q: %v", fixture.output, message, err)
			}
		}
	}
}

func TestSoftAssertionActualClusterOutputCapacityRemainsInfrastructureFailure(t *testing.T) {
	base := guestRequestWithTestBinary(t, true)
	base.WorkloadKind = target.KindGoTest
	base.Config.Args = []string{"capacity", "-test.run=TestOutputCapacity"}
	base.Config.Limits.OutputBytes = 4
	base.CompileCache = t.TempDir()
	if err := os.Chmod(base.CompileCache, 0700); err != nil {
		t.Fatal(err)
	}
	boot, err := BootConfiguration(base)
	if err != nil {
		t.Fatal(err)
	}
	handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
	spec := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelitySimulationModel, Seed: 7,
		Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "fixture", Address: "10.0.0.1", Config: boot}}}
	for _, policy := range []struct {
		reportOnly   bool
		maximumBytes uint64
	}{{false, 4096}, {true, 4096}, {true, 1}} {
		reportOnly := policy.reportOnly
		t.Run(fmt.Sprintf("report_only=%t/maximum=%d", reportOnly, policy.maximumBytes), func(t *testing.T) {
			provider := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"fixture": base}}
			if reportOnly {
				provider.Oracles = &OraclePlan{Schema: SoftAssertionObservationSchema, MaximumBytes: policy.maximumBytes, SoftAssertions: []SoftAssertionPolicy{{Handle: handle, ReportOnly: true}}}
			}
			result, runErr := gomad3sim.RunWASM(t.Context(), spec, func(ctx context.Context, cluster gomad3sim.Cluster) error {
				if _, err := cluster.Start(ctx, handle.Node); err != nil {
					return err
				}
				_, err := cluster.Wait(ctx, handle)
				return err
			}, provider)
			completed := provider.CompletedResults()
			if len(completed) != 1 {
				t.Fatalf("capacity guest did not terminate: %+v %v", completed, runErr)
			}
			observed := completed[0].Result
			t.Logf("actual capacity report_only=%t termination=%s message=%q stdout=%q offered=%d choice_bytes=%d run_error=%v", reportOnly, observed.Termination, observed.Message, observed.Stdout, observed.OfferedOutputBytes, len(observed.ChoiceTrace.Bytes), runErr)
			if observed.Termination != "capacity" || !observed.Reaped || string(observed.Stdout) != "PASS" || observed.OfferedOutputBytes != 4 || len(observed.ChoiceTrace.Bytes) != 0 {
				t.Fatalf("capacity control did not exercise matching accepted-output counters and missing choice evidence: %+v", observed)
			}
			var failure *backends.FailureError
			if !errors.As(runErr, &failure) || failure.Termination != backends.Capacity || failure.Message != "output capacity" || failure.SoftAssertions == nil || !failure.SoftAssertions.Inconclusive || failure.SoftAssertions.ReportOnly != reportOnly {
				t.Fatalf("capacity/oracle truth lost: %v", runErr)
			}
			if policy.maximumBytes == 1 {
				if !strings.Contains(runErr.Error(), "resource=oracle_evidence_bytes") {
					t.Fatalf("capacity error hid oracle evidence budget exhaustion: %v", runErr)
				}
			} else if !strings.Contains(runErr.Error(), "choice trace is not complete") {
				t.Fatalf("capacity error hid incomplete choice evidence: %v", runErr)
			}
			if result.Record.Schema != "" || result.WASM != nil {
				t.Fatalf("incomplete capacity execution acquired replayable evidence: %+v", result)
			}
			if _, err := provider.RetentionPayload(spec, result.Record); err == nil {
				t.Fatal("capacity execution published retained replay evidence")
			}
			oracles, err := provider.latest.softAssertionOracles()
			if policy.maximumBytes == 1 {
				if err == nil || !strings.Contains(err.Error(), "resource=oracle_evidence_bytes") {
					t.Fatalf("tiny oracle budget admitted evidence: %+v %v", oracles, err)
				}
			} else if err != nil || len(oracles) != 1 || oracles[0].Passed || !oracles[0].Inconclusive || oracles[0].ReportOnly != reportOnly {
				t.Fatalf("capacity output treated as complete: %+v %v", oracles, err)
			}
		})
	}
}

func TestSoftAssertionClusterCapacityCountersNeverProveCompleteness(t *testing.T) {
	handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
	for _, reportOnly := range []bool{false, true} {
		execution := execution{provider: Provider{}, limits: gomad3sim.DefaultLimits(), results: map[gomad3sim.NodeHandle]wasi.Result{
			handle: {Reaped: true, Termination: "capacity", Message: "output capacity", OfferedOutputBytes: 4, Stdout: []byte("PASS")},
		}}
		if reportOnly {
			execution.provider.Oracles = &OraclePlan{MaximumBytes: 4096, SoftAssertions: []SoftAssertionPolicy{{Handle: handle, ReportOnly: true}}}
		}
		oracles, err := execution.softAssertionOracles()
		if err != nil || len(oracles) != 1 || oracles[0].Passed || !oracles[0].Inconclusive || oracles[0].ReportOnly != reportOnly {
			t.Errorf("report_only=%t capacity passed oracle: %+v %v", reportOnly, oracles, err)
		}
	}
}

func TestSoftAssertionOutputCapacityAnnotationDoesNotBroadenOtherFailures(t *testing.T) {
	handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
	execution := execution{provider: Provider{Oracles: &OraclePlan{SoftAssertions: []SoftAssertionPolicy{{Handle: handle, ReportOnly: true}}}}}
	for _, result := range []wasi.Result{{Termination: "infrastructure", Message: "output capacity"}, {Termination: "capacity", Message: "transcript capacity"}, {Termination: "exit"}} {
		if err := execution.outputCapacityFailure(handle, result); err != nil {
			t.Fatalf("unrelated failure mapped to output capacity: %+v %v", result, err)
		}
	}
	result := wasi.Result{Termination: "capacity", Message: "output capacity", Stdout: []byte(`failed assertion: invalid {state}`)}
	var failure *backends.FailureError
	if err := execution.outputCapacityFailure(handle, result); !errors.As(err, &failure) || failure.SoftAssertions == nil || !failure.SoftAssertions.Inconclusive || !failure.SoftAssertions.ReportOnly || len(failure.SoftAssertions.Messages) != 1 || failure.SoftAssertions.Messages[0] != "invalid {state}" {
		t.Fatalf("capacity annotation lost observed static assertion or policy: %+v %v", failure, err)
	}
}

func TestSoftAssertionCapacityPrecedesFinalizationErrors(t *testing.T) {
	for _, mode := range []string{"default-budget", "report-only-budget", "capacity-first", "capacity-later"} {
		t.Run(mode, func(t *testing.T) {
			handle := gomad3sim.NodeHandle{Node: "a", Incarnation: 1}
			limits := gomad3sim.DefaultLimits()
			var oraclePlan *OraclePlan
			if mode == "default-budget" {
				limits.ScenarioEvidenceBytes = 1
			}
			if mode == "capacity-later" {
				handle.Node = "b"
			}
			if mode == "report-only-budget" {
				oraclePlan = &OraclePlan{Schema: SoftAssertionObservationSchema, MaximumBytes: 1, SoftAssertions: []SoftAssertionPolicy{{Handle: handle, ReportOnly: true}}}
			}
			broker, err := wasi.NewBroker(wasi.BrokerConfig{Nodes: []wasi.BrokerNode{{ID: "a", Incarnation: 1}, {ID: "b", Incarnation: 1}}, Limits: world.CompressedSnapshotProfile().Limits})
			if err != nil {
				t.Fatal(err)
			}
			request := wasi.Request{Runtime: &wasi.RuntimeControl{Choice: &backends.ChoiceRequest{ExecutionIdentity: choice.ExecutionIdentity{}}}}
			execution := execution{provider: Provider{Oracles: oraclePlan}, limits: limits, broker: broker,
				results:  map[gomad3sim.NodeHandle]wasi.Result{handle: {Reaped: true, Termination: "capacity", Message: "output capacity", OfferedOutputBytes: 4, Stdout: []byte("PASS")}},
				requests: map[gomad3sim.NodeHandle]wasi.Request{handle: request}}
			if strings.HasPrefix(mode, "capacity-") {
				other := gomad3sim.NodeHandle{Node: "b", Incarnation: 1}
				if handle.Node == "b" {
					other.Node = "a"
				}
				execution.results[other] = wasi.Result{Reaped: true, Termination: "exit"}
				execution.requests[other] = request
			}
			evidence, err := execution.Finish()
			var failure *backends.FailureError
			if !errors.As(err, &failure) || failure.Termination != backends.Capacity || failure.SoftAssertions == nil || !failure.SoftAssertions.Inconclusive || failure.SoftAssertions.ReportOnly != (mode == "report-only-budget") {
				t.Fatalf("%s masked authoritative capacity: %v", mode, err)
			}
			if strings.HasSuffix(mode, "budget") {
				if !strings.Contains(err.Error(), "resource=oracle_evidence_bytes") || len(evidence.Oracles) != 0 {
					t.Fatalf("oracle budget was weakened: %+v %v", evidence, err)
				}
			} else if !strings.Contains(err.Error(), "retain WASM a/1 choice control") {
				t.Fatalf("first-handle strict choice failure was lost: %v", err)
			}
			if evidence.Execution != nil && len(evidence.Execution.Incarnations) != 0 {
				t.Fatal("incomplete evidence acquired retained incarnations")
			}
		})
	}
}
