package authoring

import (
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target"
)

func TestDiscoverProjectsCompleteCurrentEvidenceAndClearsApproval(t *testing.T) {
	draft := validRequest()
	draft.ApprovalSHA256 = "sha256:" + strings.Repeat("f", 64)
	draft.Activation[0].Evidence = compatibility.PackModule{}
	draft.Packages[0].Evidence = compatibility.PackRule{}
	review := discoveryReview("sha256:" + strings.Repeat("4", 64))

	discovered, digest, err := Discover(draft, review)
	if err != nil {
		t.Fatal(err)
	}
	if discovered.ApprovalSHA256 != "" || discovered.Activation[0].Evidence.Version != "v1.2.3" || discovered.Packages[0].Evidence.GoSources[0].Name != "runtime.go" {
		t.Fatalf("discovered request = %#v", discovered)
	}
	if digest == "" {
		t.Fatal("discovery returned no review digest")
	}

	review.Closure.Packages[0].Sources[0].SHA256 = "sha256:" + strings.Repeat("e", 64)
	_, changedDigest, err := Discover(draft, review)
	if err != nil {
		t.Fatal(err)
	}
	if changedDigest == digest {
		t.Fatalf("changed source retained review digest %q", digest)
	}
}

func TestDiscoverAddsNewlyObservedFactsAsDenied(t *testing.T) {
	draft := validRequest()
	draft.Activation[0].Evidence = compatibility.PackModule{}
	draft.Packages[0].Evidence = compatibility.PackRule{}
	review := discoveryReview("sha256:" + strings.Repeat("4", 64))
	review.Closure.Packages[0].Imports = []string{"os/exec", "syscall"}
	review.Findings = []target.CapabilityFinding{{
		Kind: target.FindingForbiddenImport,
		Package: target.CapabilityPackageReference{
			ImportPath: "example.com/dependency/internal/runtime",
			Name:       "runtime",
		},
		Capability: "import:os/exec",
	}}

	discovered, _, err := Discover(draft, review)
	if err != nil {
		t.Fatal(err)
	}
	want := []Fact{
		{Kind: FactCapability, Capability: "import:os/exec", Directives: []string{}, Disposition: DispositionDeny},
		{Kind: FactCapability, Capability: "import:syscall", Directives: []string{}, Disposition: DispositionAllow},
	}
	if !slices.EqualFunc(discovered.Packages[0].Facts, want, func(left, right Fact) bool {
		return left.Kind == right.Kind && left.Capability == right.Capability && left.Source == right.Source &&
			left.SHA256 == right.SHA256 && slices.Equal(left.Directives, right.Directives) && left.Disposition == right.Disposition
	}) {
		t.Fatalf("discovered facts = %#v, want %#v", discovered.Packages[0].Facts, want)
	}
}

func discoveryReview(sourceSHA256 string) target.CapabilityReview {
	module := &target.CapabilityModule{Path: "example.com/dependency", Version: "v1.2.3", Sum: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}
	return target.CapabilityReview{
		Schema: target.CapabilityReviewSchema, BuildTags: []string{"test_dep"},
		Roots: []target.CapabilityPackageReference{{ImportPath: "example.com/target", Name: "main"}},
		Closure: target.CapabilityClosure{
			Schema: target.CapabilityClosureSchema, Compatibility: []target.CompatibilityIdentity{},
			Packages: []target.CapabilityPackage{{
				ImportPath: "example.com/dependency/internal/runtime", Name: "runtime", Imports: []string{"syscall"}, Module: module,
				Sources: []target.CapabilitySource{{Name: "runtime.go", SHA256: sourceSHA256}}, ForeignSources: []target.CapabilityForeignSource{},
			}, {
				ImportPath: "example.com/target", Name: "main", Root: true, Imports: []string{"example.com/dependency/internal/runtime"},
				Module: &target.CapabilityModule{Path: "example.com/main", Main: true}, Sources: []target.CapabilitySource{}, ForeignSources: []target.CapabilityForeignSource{},
			}},
		},
		Packs: []target.CompatibilityPackEvidence{}, Findings: []target.CapabilityFinding{},
	}
}
