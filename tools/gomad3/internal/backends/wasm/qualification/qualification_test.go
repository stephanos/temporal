package qualification

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/backends/wasm/backend"
	gomadqualification "go.temporal.io/server/tools/gomad3/internal/evidence/qualification"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad3/internal/host/exec"
)

func TestResultRetainsCanonicalOwnerBytesSourceLess(t *testing.T) {
	report, err := gomadqualification.BuildQualificationFailure([]string{"qualify"}, 7, 2, nil, gomadqualification.QualificationFailure{Classification: "runner_failure", Message: "not started", Iteration: 1})
	if err != nil {
		t.Fatal(err)
	}
	path, err := gomadqualification.WriteQualificationReport(t.TempDir(), report)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result := Result{Manifest: testManifest(), Qualification: report, QualificationBytes: encoded, CostFailure: "actual preparation and compiler measurements are required"}
	retained, err := WriteResult(t.TempDir(), result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, path+".unavailable"); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenResult(retained); err != nil {
		t.Fatalf("source-less result opened original support file: %v", err)
	}
	result.Qualification.Failure.Message = "changed"
	if err := ValidateResult(result); err == nil {
		t.Fatal("changed exposed support report disagrees with retained owner bytes")
	}
}

func TestOperatorBaselineIsDetachedAndHostOnly(t *testing.T) {
	start := &hostexec.ResourceUsage{UserNanos: 7, SystemNanos: 11, PeakRSSBytes: 13}
	cloned := operatorBaseline(start)
	start.UserNanos = 99
	if cloned.UserNanos != 7 || cloned == start {
		t.Fatal("operator baseline aliases caller observation")
	}
	path := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(path, []byte(`{"OperatorStart":{"UserNanos":7}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSpec(path); err == nil {
		t.Fatal("JSON caller forged an operator observation")
	}
}

func TestRetainedClusterFreezeRejectsUnobservedPreparation(t *testing.T) {
	if _, err := FreezeRetainedCluster(Spec{Manifest: testManifest()}, Spec{}); err == nil {
		t.Fatal("unobserved preparation accepted")
	}
	if _, err := PreparedTarget(Spec{}); err == nil {
		t.Fatal("empty frozen target accepted")
	}
}

func TestSourceManifestBindsActualFilesAndKeepsSpacePaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source with spaces.go")
	data := []byte("current source")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	encoded := []byte(fmt.Sprintf("%s  %s\n", string(record.HashBytes(data))[7:], path))
	if err := validateSourceManifest(encoded, true); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("changed source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateSourceManifest(encoded, true); err == nil {
		t.Fatal("changed source accepted")
	}
	if err := validateSourceManifest(encoded, false); err != nil {
		t.Fatalf("source-less validation accessed files: %v", err)
	}
	if err := validateSourceManifest(append(encoded, encoded...), false); err == nil {
		t.Fatal("duplicate manifest path accepted")
	}
}

func TestReadSpecRejectsAmbiguousAndCompetingInputs(t *testing.T) {
	invalidUTF8 := append([]byte(`{"Manifest":{"ID":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`","Workload":"fixture","Seed":7,"Repeat":2}}`)...)
	for _, data := range [][]byte{[]byte(`{"Manifest":{},"manifest":{}}`), []byte(`{"unknown":1}`), []byte(`{} {}`), []byte(`{"Options":{}}`), invalidUTF8} {
		path := filepath.Join(t.TempDir(), "spec.json")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadSpec(path); err == nil {
			t.Fatalf("ambiguous input accepted: %s", data)
		}
	}
	path := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(path, []byte(`{"Options":{},"unknown":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSpec(path); err == nil || err.Error() != "qualification spec Options must be absent; installation owns the pinned backend" {
		t.Fatalf("Options rejection lost precedence: %v", err)
	}
	path = filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(path, []byte(`{"Manifest":{"ID":"fixture","Workload":"fixture","Seed":7,"Repeat":2}}`), 0600); err != nil {
		t.Fatal(err)
	}
	spec, err := ReadSpec(path)
	if err != nil || spec.Manifest.ID != "fixture" {
		t.Fatalf("valid draft spec: %+v %v", spec, err)
	}
}

func TestStrictDecodeRetainsQualificationDepthLimit(t *testing.T) {
	for _, test := range []struct {
		name       string
		depth      int
		emptyValue bool
		invalid    bool
		fail       bool
	}{
		{name: "edge", depth: 128},
		{name: "overflow", depth: 129, fail: true},
		{name: "malformed-overflow", depth: 129, invalid: true, fail: true},
		{name: "empty-container-edge", depth: 129, emptyValue: true},
		{name: "empty-container-overflow", depth: 130, emptyValue: true, fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			leaf := "0"
			if test.emptyValue {
				leaf = ""
			} else if test.invalid {
				leaf = "|"
			}
			data := []byte(strings.Repeat("[", test.depth) + leaf + strings.Repeat("]", test.depth))
			var value any
			err := strictDecode(data, &value)
			if test.fail {
				if err == nil || err.Error() != "qualification JSON nesting limit" {
					t.Fatalf("depth %d error = %v", test.depth, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("depth %d rejected: %v", test.depth, err)
			}
		})
	}
}

func TestRunRejectsOptionsOutsideFrozenManifest(t *testing.T) {
	_, err := Run(context.Background(), Spec{Manifest: testManifest(), Options: backend.Options{Fuel: 101, MemoryBytes: 64}})
	if err == nil {
		t.Fatal("qualification entered Runner with unmatched options")
	}
}

func TestComparisonRejectsUnmatchedLimits(t *testing.T) {
	a, b := testManifest(), testManifest()
	b.Profile = "cooperative"
	if err := Matched(a, b); err != nil {
		t.Fatal(err)
	}
	b.Fuel++
	if err := Matched(a, b); err == nil {
		t.Fatal("unmatched fuel budgets compared")
	}
}

func TestManifestRejectsUnboundOrUnadmittedProfiles(t *testing.T) {
	valid := testManifest()
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Manifest){
		func(m *Manifest) { m.IntegratedSourceSHA256 = "" },
		func(m *Manifest) { m.CompilerSHA256 = "" },
		func(m *Manifest) { m.HostArch = "" },
		func(m *Manifest) { m.Profile = "checkpoint" },
		func(m *Manifest) { m.Profile = "stock-uncontrolled" },
		func(m *Manifest) { m.Workload = "unknown" },
	} {
		candidate := valid
		mutate(&candidate)
		if err := candidate.Validate(); err == nil {
			t.Fatalf("unbound or unadmitted manifest accepted: %+v", candidate)
		}
	}
}

func TestCostRequiresRealExecutionMeasurementsAndSeparatesReplay(t *testing.T) {
	input := []Sample{{Phase: "execute", PreparationWallNanos: 7, ExecutionWallNanos: 2_000_000_000, CPUNanos: 1_000_000_000, GuestPeakMemoryBytes: 64, HostPeakRSSBytes: 128, FuelUsed: 100, TraceBytes: 32, VirtualTimeNanos: 10_000_000_000, OutcomeSHA256: record.HashBytes([]byte("failure")), ReplayableBugSHA256: record.HashBytes([]byte("bug"))}, {Phase: "replay", ExecutionWallNanos: 3_000_000_000, CPUNanos: 1_000_000_000, GuestPeakMemoryBytes: 64, HostPeakRSSBytes: 128}}
	cost, err := Summarize(input)
	if err != nil {
		t.Fatal(err)
	}
	if cost.Executions != 1 || cost.Replays != 1 || cost.DistinctOutcomes != 1 || cost.ReplayableBugs != 1 || cost.ReplayableBugsPerComputeHour != 1800 || cost.VirtualToWallRatio != 5 || cost.PreparationWallNanos != 7 {
		t.Fatalf("wrong measured cost: %+v", cost)
	}
	input[0].CPUNanos = 0
	if _, err := Summarize(input); err == nil {
		t.Fatal("unmeasured CPU admitted as qualified cost")
	}
}

func TestOperatorCostIncludesSetupGapAndSeparatesLogicalVirtualTime(t *testing.T) {
	result := Result{InitialPreparationWallNanos: 2, InitialCompilerCommandCount: 1, InitialPreparationCompiled: true, PreparationProvenanceSHA256: record.HashBytes(nil), CompilerCommands: []backend.CompilerCommandMetrics{{Arguments: []string{"build"}, WallElapsedNanos: 7, ResourceUsage: &hostexec.ResourceUsage{UserNanos: 2, SystemNanos: 3, PeakRSSBytes: 64}}}, Samples: []Sample{{Phase: "execute", ExecutionWallNanos: 5, CPUNanos: 10, GuestPeakMemoryBytes: 8, HostPeakRSSBytes: 20, VirtualTimeNanos: 11, OutcomeSHA256: record.HashBytes(nil)}}, OperatorPreparationBefore: &hostexec.ResourceUsage{UserNanos: 4, SystemNanos: 2}, OperatorPreparationAfter: &hostexec.ResourceUsage{UserNanos: 6, SystemNanos: 4, PeakRSSBytes: 32}, OperatorExecutionBefore: &hostexec.ResourceUsage{UserNanos: 9, SystemNanos: 6}, OperatorExecutionAfter: &hostexec.ResourceUsage{UserNanos: 20, SystemNanos: 10, PeakRSSBytes: 40}, Qualification: gomadqualification.QualificationReport{Executions: []gomadqualification.QualificationExecutionReport{{Evidence: &record.ExecutionEvidence{VirtualTimeElapsedNanos: 100}, WallElapsedNanos: 50}}}}
	cost, err := measuredCost(result)
	if err != nil || cost.CPUNanos != 39 || cost.OperatorCPUNanos != 24 || cost.CompilerCPUNanos != 5 || cost.HelperCPUNanos != 10 || cost.VirtualToWallRatio != 2 || cost.GuestVirtualToExecutionWallRatio != 2.2 {
		t.Fatalf("operator gap or duplicate global time: %+v %v", cost, err)
	}
	t.Run("canonical-owner-matched-evidence", func(t *testing.T) {
		result.Qualification.Evidence = result.Qualification.Executions[0].Evidence
		result.Qualification.EvidenceDigest = record.HashBytes([]byte("matched owner evidence"))
		result.Qualification.Executions[0].EvidenceDigest = result.Qualification.EvidenceDigest
		result.Qualification.Executions[0].Evidence = nil
		cost, err := measuredCost(result)
		if err != nil || cost.CPUNanos != 39 || cost.VirtualToWallRatio != 2 {
			t.Fatalf("canonical owner's shared matched evidence lost: %+v %v", cost, err)
		}
		result.Qualification.Executions[0].EvidenceDigest = record.HashBytes([]byte("divergent evidence"))
		if _, err := measuredCost(result); err == nil {
			t.Fatal("unmatched execution reused aggregate virtual time")
		}
	})
	result.InitialPreparationCompiled = false
	if _, err := measuredCost(result); err == nil {
		t.Fatal("cold module build relabeled as warm-cache preparation")
	}
}

func testManifest() Manifest {
	digest := record.HashBytes([]byte("pinned"))
	return Manifest{Provisional: true, Schema: ManifestSchema, ID: "fixture", Workload: "fixture", Profile: "controlled", IntegratedSourceSHA256: digest, WorkloadSourceSHA256: digest, CompilerSHA256: digest, RuntimeSHA256: digest, HelperSHA256: digest, ModelSHA256: digest, EngineSHA256: digest, HostConfigSHA256: digest, HostOS: "linux", HostArch: "arm64", Seed: 7, Repeat: 2, Fuel: 100, MemoryBytes: 64}
}
