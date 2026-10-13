package simulation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	wasmbackend "go.temporal.io/server/tools/gomad3/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad3/backends/wasm/internal/stockgo"
	"go.temporal.io/server/tools/gomad3/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3sim"
)

func softAssertionGuestRequest(t *testing.T) wasi.Request {
	t.Helper()
	request := guestRequest(t)
	root := stockgo.Root(t)
	runtimeRoot, err := filepath.Abs("../../../runtimesemantics")
	if err != nil {
		t.Fatal(err)
	}
	overlay, _, err := toolchain.BuildOverlay(root, runtimeRoot, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	module := filepath.Join(t.TempDir(), "soft-assertion.wasm")
	cmd := exec.Command(filepath.Join(root, "bin/go"), "test", "-tags=test_dep", "-c", "-trimpath", "-overlay", overlay, "-o", module, "./tools/gomad3sim/testdata/softassert")
	cmd.Dir = "../../../../.."
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
	return request
}

func TestSoftAssertionActualClusterGuestAndRetainedReplay(t *testing.T) {
	base := softAssertionGuestRequest(t)
	module := retainedTargetModule(t, base)
	handles := []gomad3sim.NodeHandle{{Node: "a", Incarnation: 1}, {Node: "b", Incarnation: 1}}
	for _, reportOnly := range []bool{false, true} {
		var signature string
		for _, seed := range []uint64{7, 19} {
			request := base
			runtime := *base.Runtime
			runtime.Seed = seed
			request.Runtime = &runtime
			request.Config.Args = []string{"soft-assertion", "-test.v", "-assertion-mode=late"}
			boot, err := BootConfiguration(request)
			if err != nil {
				t.Fatal(err)
			}
			spec := gomad3sim.Spec{Schema: gomad3sim.SpecSchema, Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelityHardIsolation, Seed: seed,
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
				}
				for _, handle := range handles {
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
	spec := gomad3sim.Spec{Backend: gomad3sim.BackendWASM, Fidelity: gomad3sim.FidelityHardIsolation, Limits: gomad3sim.DefaultLimits(), Nodes: []gomad3sim.NodeSpec{{ID: "a", Boot: "fixture"}}}
	p := &Provider{Boots: map[gomad3sim.BootID]wasi.Request{"fixture": {WorkloadKind: target.KindGoRun}}, Oracles: &OraclePlan{Schema: SoftAssertionObservationSchema, MaximumBytes: 4096, SoftAssertions: []SoftAssertionPolicy{{Handle: handle, ReportOnly: true}}}}
	if _, err := p.Open(t.Context(), spec); err == nil || !strings.Contains(err.Error(), "report-only mode requires") {
		t.Fatalf("unsupported workload opt-out admitted: %v", err)
	}
	p.Boots["fixture"] = wasi.Request{WorkloadKind: target.KindGoTest}
	if err := p.validateSoftAssertionWorkloads(spec); err != nil {
		t.Fatal(err)
	}
}
