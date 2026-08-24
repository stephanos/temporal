package comparison

import (
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	capabilityanalysis "go.temporal.io/server/tools/gomad3/qualification/analysis"
	qualificationset "go.temporal.io/server/tools/gomad3/qualification/set"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestCompareClassifiesSupportTransitions(t *testing.T) {
	for _, test := range []struct {
		name           string
		baseline       string
		candidate      string
		classification Classification
	}{
		{name: "clean", baseline: "qualified", candidate: "qualified", classification: Clean},
		{name: "improved", baseline: "unsupported_target", candidate: "qualified", classification: Improved},
		{name: "regressed", baseline: "qualified", candidate: "unsupported_target", classification: Regressed},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := Compare(Input{Baseline: comparisonReport(test.baseline), Candidate: comparisonReport(test.candidate)})
			if err != nil {
				t.Fatal(err)
			}
			if result.Classification != test.classification || len(result.Workloads) != 1 || result.Workloads[0].BaselineClassification != test.baseline || result.Workloads[0].CandidateClassification != test.candidate {
				t.Fatalf("comparison = %#v", result)
			}
		})
	}
}

func TestCompareRequiresExactBoundaryApproval(t *testing.T) {
	baseline := comparisonReport("qualified")
	candidate := comparisonReport("qualified")
	candidate.Toolchain.BoundaryManifestSHA256 = record.HashBytes([]byte("candidate-boundary"))
	result, err := Compare(Input{Baseline: baseline, Candidate: candidate})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReviewRequired || !result.Boundary.Changed || result.Boundary.DiffSHA256 == "" || result.Boundary.Approved {
		t.Fatalf("comparison = %#v", result)
	}
	approved, err := Compare(Input{Baseline: baseline, Candidate: candidate, ApprovedBoundaryDiff: result.Boundary.DiffSHA256})
	if err != nil {
		t.Fatal(err)
	}
	if approved.ReviewRequired || !approved.Boundary.Approved {
		t.Fatalf("approved comparison = %#v", approved)
	}
	wrong, err := Compare(Input{Baseline: baseline, Candidate: candidate, ApprovedBoundaryDiff: record.HashBytes([]byte("different"))})
	if err != nil {
		t.Fatal(err)
	}
	if !wrong.ReviewRequired || wrong.Boundary.Approved {
		t.Fatalf("wrong approval comparison = %#v", wrong)
	}
}

func TestCompareMarksLegacyEvidenceIncomparable(t *testing.T) {
	baseline := comparisonReport("qualified")
	baseline.Dimensions.PortableV3 = false
	result, err := Compare(Input{Baseline: baseline, Candidate: comparisonReport("qualified")})
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != Incomparable || !strings.Contains(FormatText(result), "portable_v3") {
		t.Fatalf("comparison = %#v", result)
	}
}

func TestCompareRequiresReviewForAnyChangedUnsupportedBlocker(t *testing.T) {
	first := comparisonBlocker("example.com/first", "import:os/exec")
	second := comparisonBlocker("example.com/second", "import:syscall")
	baseline := comparisonReport("unsupported_target")
	baseline.Workloads[0].Blockers = []capabilityanalysis.Blocker{first, second}
	candidate := comparisonReport("unsupported_target")
	candidate.Workloads[0].Blockers = []capabilityanalysis.Blocker{second}

	result, err := Compare(Input{Baseline: baseline, Candidate: candidate})
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != Regressed || !result.ReviewRequired || len(result.Workloads[0].BlockersRemoved) != 1 || len(result.Workloads[0].BlockersAdded) != 0 {
		t.Fatalf("changed blocker comparison = %#v", result)
	}
}

func TestCompareMarksWorkloadMembershipChangesIncomparable(t *testing.T) {
	populated := comparisonReport("qualified")
	empty := comparisonReport("qualified")
	empty.Selected = 0
	empty.AnalysisCompleted = 0
	empty.Completed = 0
	empty.Supported = 0
	empty.Workloads = []qualificationset.WorkloadReport{}
	for _, input := range []Input{
		{Baseline: populated, Candidate: empty},
		{Baseline: empty, Candidate: populated},
	} {
		result, err := Compare(input)
		if err != nil {
			t.Fatal(err)
		}
		if result.Classification != Incomparable {
			t.Fatalf("membership comparison = %#v", result)
		}
	}
}

func TestCompareReportsUpgradeIdentitiesWithoutLosingSupportComparison(t *testing.T) {
	baseline := comparisonReport("qualified")
	candidate := comparisonReport("qualified")
	candidate.Module.GoModSHA256 = record.HashBytes([]byte("changed go.mod"))
	candidate.Platform.GOARCH = "amd64"
	candidate.Toolchain.GoVersion = "go1.27.0"
	candidate.Toolchain.BuildKey = strings.Repeat("b", 64)
	candidate.IOProfile.ImplementationSHA256 = deterministicio.Digest(record.HashBytes([]byte("changed io")))
	result, err := Compare(Input{Baseline: baseline, Candidate: candidate})
	if err != nil {
		t.Fatal(err)
	}
	if result.Classification != Clean || result.ReviewRequired || len(result.IdentityDifferences) != 5 {
		t.Fatalf("upgrade comparison = %#v", result)
	}
}

func comparisonBlocker(importPath, capability string) capabilityanalysis.Blocker {
	return capabilityanalysis.Blocker{CapabilityFinding: target.CapabilityFinding{
		Kind: target.FindingForbiddenImport, Package: target.CapabilityPackageReference{ImportPath: importPath, Name: "dependency"},
		Capability: capability, Directives: []string{}, PolicyDisposition: target.DispositionDenied,
		Remediation: target.RemediationRemainUnsupported,
	}}
}

func comparisonReport(classification string) qualificationset.Report {
	boundary := record.HashBytes([]byte("boundary"))
	report := qualificationset.Report{
		Schema: qualificationset.ReportSchema, Name: "test-set", Description: "comparison fixture",
		ManifestSHA256: record.HashBytes([]byte("manifest")),
		Module:         qualificationset.ModuleIdentity{Path: "example.com/target", GoModSHA256: record.HashBytes([]byte("go.mod"))},
		Platform:       qualificationset.PlatformIdentity{GOOS: "darwin", GOARCH: "arm64"},
		Toolchain: capabilityanalysis.Toolchain{
			GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64",
			BoundaryManifestVersion: "boundary-v1", BoundaryManifestSHA256: boundary,
		},
		IOProfile:       deterministicio.Contract{Name: "io", ImplementationSHA256: deterministicio.Digest(record.HashBytes([]byte("io"))), InventorySHA256: deterministicio.Digest(record.HashBytes([]byte("inventory")))},
		Dimensions:      qualificationset.EvidenceDimensions{PortableV3: true, Analysis: true, Replay: true, Choice: true},
		ExpectationsMet: classification == "qualified", Selected: 1, AnalysisCompleted: 1, Completed: 1,
		Workloads: []qualificationset.WorkloadReport{{
			ID: "fixture-case", Name: "Fixture", Tier: 1, Invariant: "fixture invariant",
			Expected: qualificationset.WorkloadExpectation{Classification: "qualified"}, ExpectationMet: classification == "qualified",
			Classification: classification, Seeds: []qualificationset.SeedReport{}, Blockers: []capabilityanalysis.Blocker{},
			Choice: qualificationset.ChoiceCoverage{Features: []choice.Feature{}},
		}},
	}
	switch classification {
	case "qualified":
		report.Supported = 1
	case "unsupported_target":
		report.Unsupported = 1
	default:
		report.Failed = 1
	}
	return report
}
