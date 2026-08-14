package capabilityanalysis

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/compatibility"
	"go.temporal.io/server/tools/gomadv3/internal/ioprofile"
	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
)

func TestBuildReportsLexicographicallyFirstShortestDependencyPath(t *testing.T) {
	review := graphReview([]target.CapabilityPackage{
		{ImportPath: "a/root", Name: "main", Root: true, Imports: []string{"a/middle", "b/middle"}, Sources: sourceEvidence("root.go")},
		{ImportPath: "a/middle", Name: "middle", Imports: []string{"dependency"}, Sources: sourceEvidence("middle.go")},
		{ImportPath: "b/middle", Name: "middle", Imports: []string{"dependency"}, Sources: sourceEvidence("middle.go")},
		{ImportPath: "dependency", Name: "dependency", Imports: []string{"a/root"}, Sources: sourceEvidence("dependency.go")},
	})
	review.Findings = []target.CapabilityFinding{{
		Kind: target.FindingForbiddenImport, Package: target.CapabilityPackageReference{ImportPath: "dependency", Name: "dependency"},
		Directives: []string{}, Capability: "import:os/exec", PolicyDisposition: compatibility.DispositionDenied,
		Remediation: compatibility.RemediationRemainUnsupported,
	}}

	report, err := Build(Input{
		Spec:      target.Spec{Kind: target.KindGoRun, Source: filepath.Join(string(filepath.Separator), "private", "work", "target"), Args: []string{}, BuildTags: []string{"gomad_fixture"}},
		Review:    review,
		Toolchain: target.ToolchainIdentity{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		IOProfile: ioprofile.Default(), Adapters: []record.TargetAdapter{},
	})
	requireTestNoError(t, err)
	requireTestEqual(t, Schema, report.Schema)
	requireTestEqual(t, ClassificationUnsupported, report.Classification)
	requireTestEqual(t, "target", report.Target.Source)
	requireTestEqual(t, []target.CapabilityPackageReference{
		{ImportPath: "a/root", Name: "main"},
		{ImportPath: "a/middle", Name: "middle"},
		{ImportPath: "dependency", Name: "dependency"},
	}, report.Blockers[0].DependencyPath)
	requireTestEqual(t, []target.CapabilityPackageReference{{ImportPath: "a/root", Name: "main"}}, report.Closure.Roots)
	if report.Closure.SHA256 == "" {
		t.Fatal("closure digest is empty")
	}

	encoded, err := json.Marshal(report)
	requireTestNoError(t, err)
	if strings.Contains(string(encoded), "/private/work") {
		t.Fatalf("report contains an absolute host path: %s", encoded)
	}
}

func TestBuildRedactsPathBearingArgumentsDeterministically(t *testing.T) {
	review := graphReview([]target.CapabilityPackage{{
		ImportPath: "example.com/target", Name: "main", Root: true, Imports: []string{}, Sources: sourceEvidence("main.go"),
	}})
	absolute := filepath.Join(string(filepath.Separator), "private", "work", "input")
	report, err := Build(Input{
		Spec: target.Spec{Kind: target.KindGoRun, Source: ".", Args: []string{absolute, "--output=" + absolute, "-test.run=TestScenario"}}, Review: review,
		Toolchain: target.ToolchainIdentity{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		IOProfile: ioprofile.Default(), Adapters: []record.TargetAdapter{},
	})
	requireTestNoError(t, err)
	if report.Target.Arguments[0] == absolute || report.Target.Arguments[1] == "--output="+absolute || report.Target.Arguments[2] != "-test.run=TestScenario" {
		t.Fatalf("projected arguments = %#v", report.Target.Arguments)
	}
	encoded, err := record.CanonicalJSON(report)
	requireTestNoError(t, err)
	if strings.Contains(string(encoded), absolute) || !strings.Contains(string(encoded), string(record.HashBytes([]byte(absolute)))) {
		t.Fatalf("report arguments expose an absolute host path: %s", encoded)
	}
}

func TestBuildUsesAllSortedRootsAndRejectsUnreachableFindings(t *testing.T) {
	review := graphReview([]target.CapabilityPackage{
		{ImportPath: "b/root", Name: "main", Root: true, Imports: []string{"dependency"}, Sources: sourceEvidence("root.go")},
		{ImportPath: "a/root", Name: "main", Root: true, Imports: []string{"dependency"}, Sources: sourceEvidence("root.go")},
		{ImportPath: "dependency", Name: "dependency", Sources: sourceEvidence("dependency.go")},
		{ImportPath: "unreachable", Name: "unreachable", Sources: sourceEvidence("unreachable.go")},
	})
	review.Findings = []target.CapabilityFinding{{
		Kind: target.FindingNoReviewedGoSource, Package: target.CapabilityPackageReference{ImportPath: "unreachable", Name: "unreachable"},
		Directives: []string{}, PolicyDisposition: compatibility.DispositionDenied, Remediation: compatibility.RemediationRemoveDependency,
	}}

	_, err := Build(Input{
		Spec: target.Spec{Kind: target.KindGoTest, Source: "./target", Args: []string{}, BuildTags: []string{}}, Review: review,
		Toolchain: target.ToolchainIdentity{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		IOProfile: ioprofile.Default(), Adapters: []record.TargetAdapter{},
	})
	if err == nil || !strings.Contains(err.Error(), "not reachable") {
		t.Fatalf("Build() error = %v", err)
	}
}

func TestBuildSupportedReportHasCanonicalNonNullEvidence(t *testing.T) {
	review := graphReview([]target.CapabilityPackage{{
		ImportPath: "example.com/target", Name: "main", Root: true, Imports: []string{}, Sources: sourceEvidence("main.go"),
	}})
	report, err := Build(Input{
		Spec: target.Spec{Kind: target.KindGoRun, Source: "example.com/target", Args: []string{}, BuildTags: []string{}}, Review: review,
		Toolchain: target.ToolchainIdentity{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		IOProfile: ioprofile.Default(), Adapters: []record.TargetAdapter{},
	})
	requireTestNoError(t, err)
	requireTestEqual(t, ClassificationSupported, report.Classification)
	if len(report.Blockers) != 0 || report.Blockers == nil || len(report.Packs) != 0 || report.Packs == nil || len(report.Requirements) != 0 || report.Requirements == nil {
		t.Fatalf("non-null empty evidence: blockers=%#v packs=%#v requirements=%#v", report.Blockers, report.Packs, report.Requirements)
	}

	encoded, err := record.CanonicalJSON(report)
	requireTestNoError(t, err)
	for _, want := range []string{`"blockers":[]`, `"packs":[]`, `"requirements":[]`} {
		if !strings.Contains(string(encoded), want) {
			t.Fatalf("canonical report = %s", encoded)
		}
	}
}

func TestDecodeValidatesCanonicalCapabilityReport(t *testing.T) {
	review := graphReview([]target.CapabilityPackage{{
		ImportPath: "example.com/target", Name: "main", Root: true, Imports: []string{}, Sources: sourceEvidence("main.go"),
	}})
	report, err := Build(Input{
		Spec: target.Spec{Kind: target.KindGoRun, Source: "example.com/target", Args: []string{}, BuildTags: []string{}}, Review: review,
		Toolchain: target.ToolchainIdentity{GoVersion: "go1.26.4", BuildKey: strings.Repeat("a", 64), TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		IOProfile: ioprofile.Default(), Adapters: []record.TargetAdapter{},
	})
	requireTestNoError(t, err)
	encoded, err := record.CanonicalJSON(report)
	requireTestNoError(t, err)
	decoded, err := Decode(encoded)
	requireTestNoError(t, err)
	requireTestEqual(t, report, decoded)

	report.Classification = ClassificationUnsupported
	encoded, err = record.CanonicalJSON(report)
	requireTestNoError(t, err)
	if _, err := Decode(encoded); err == nil {
		t.Fatal("Decode() accepted unsupported analysis without blockers")
	}
}

func TestFormatTextGroupsEquivalentBlockersWithShortestPathFirst(t *testing.T) {
	report := Report{
		Classification: ClassificationUnsupported,
		Closure:        Closure{PackageCount: 3},
		Blockers: []Blocker{
			{
				CapabilityFinding: target.CapabilityFinding{
					Kind: target.FindingForbiddenImport, Package: target.CapabilityPackageReference{ImportPath: "b/dependency", Name: "dependency"},
					Capability: "import:os/exec", Remediation: compatibility.RemediationRemainUnsupported,
				},
				DependencyPath: []target.CapabilityPackageReference{{ImportPath: "root", Name: "main"}, {ImportPath: "middle", Name: "middle"}, {ImportPath: "b/dependency", Name: "dependency"}},
			},
			{
				CapabilityFinding: target.CapabilityFinding{
					Kind: target.FindingForbiddenImport, Package: target.CapabilityPackageReference{ImportPath: "a/dependency", Name: "dependency"},
					Capability: "import:os/exec", Remediation: compatibility.RemediationRemainUnsupported,
				},
				DependencyPath: []target.CapabilityPackageReference{{ImportPath: "root", Name: "main"}, {ImportPath: "a/dependency", Name: "dependency"}},
			},
		},
	}

	formatted := FormatText(report)
	if strings.Count(formatted, "- forbidden_import import:os/exec (remain_unsupported)") != 1 {
		t.Fatalf("formatted report = %q", formatted)
	}
	if !strings.Contains(formatted, "path: root -> a/dependency") || !strings.Contains(formatted, "affected: a/dependency, b/dependency") {
		t.Fatalf("formatted report = %q", formatted)
	}
}

func graphReview(packages []target.CapabilityPackage) target.CapabilityReview {
	roots := []target.CapabilityPackageReference{}
	for _, pkg := range packages {
		if pkg.Root {
			roots = append(roots, target.CapabilityPackageReference{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: pkg.Name})
		}
	}
	return target.CapabilityReview{
		Schema: target.CapabilityReviewSchema, BuildTags: []string{}, Roots: roots,
		Closure: target.CapabilityClosure{Schema: "gomadv3.target-capability-closure/v2", Compatibility: []compatibility.Identity{}, Packages: packages},
		Packs:   []compatibility.PackEvidence{}, Findings: []target.CapabilityFinding{},
	}
}

func sourceEvidence(name string) []target.CapabilitySource {
	return []target.CapabilitySource{{Name: name, SHA256: "sha256:" + strings.Repeat("a", 64), LinknameDirectives: []string{}}}
}

func requireTestNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func requireTestEqual(t *testing.T, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
