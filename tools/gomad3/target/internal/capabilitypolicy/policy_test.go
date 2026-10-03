package capabilitypolicy

import (
	"reflect"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
)

func TestEvaluateOrdersFindingsByPackageAndConcern(t *testing.T) {
	malformed := Source{Name: "malformed.go", SHA256: "sha256:" + strings.Repeat("1", 64), MalformedLinkname: true}
	linked := Source{Name: "linked.go", SHA256: "sha256:" + strings.Repeat("2", 64), LinknameDirectives: []string{"local remote.Symbol"}}
	packages := []Package{
		{ImportPath: "os", Standard: true, Imports: []string{"syscall"}},
		{
			ImportPath: "example.com/target", Imports: []string{"fmt", "os/exec", "golang.org/x/sys/unix"},
			Sources: []Source{malformed, linked},
			Policy: compatibility.Package{ImportPath: "example.com/target", ForeignSources: []compatibility.ForeignSource{
				{Kind: "c", Name: "escape.c", SHA256: "sha256:" + strings.Repeat("3", 64)},
				{Kind: "header", Name: "api.h", SHA256: "sha256:" + strings.Repeat("4", 64)},
			}},
		},
		{ImportPath: "example.com/empty"},
		{ImportPath: "example.com/target.test", GeneratedTestMain: true, Imports: []string{"testing"}},
	}
	evaluation, err := Evaluate(Policy{}, packages)
	if err != nil {
		t.Fatal(err)
	}
	type summary struct {
		Package    int
		Kind       Kind
		Capability string
		Source     string
	}
	got := []summary{}
	for _, finding := range evaluation.Findings {
		if finding.Decision.Allowed || finding.Decision.Disposition != compatibility.DispositionDenied {
			t.Fatalf("finding decision = %#v", finding.Decision)
		}
		got = append(got, summary{Package: finding.Package, Kind: finding.Kind, Capability: finding.Fact.Capability, Source: finding.SourceName})
	}
	want := []summary{
		{Package: 1, Kind: KindForbiddenImport, Capability: "import:os/exec"},
		{Package: 1, Kind: KindForbiddenImport, Capability: "import:golang.org/x/sys/unix"},
		{Package: 1, Kind: KindForeignSource, Capability: "foreign:c:escape.c", Source: "escape.c"},
		{Package: 1, Kind: KindMalformedLinkname, Capability: "linkname:malformed", Source: "malformed.go"},
		{Package: 1, Kind: KindUnapprovedLinkname, Capability: "linkname:linked.go", Source: "linked.go"},
		{Package: 2, Kind: KindNoReviewedGoSource, Capability: "source:go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("findings = %#v, want %#v", got, want)
	}
	if directives := evaluation.Findings[4].Fact.Directives; !reflect.DeepEqual(directives, linked.LinknameDirectives) {
		t.Fatalf("linkname directives = %#v", directives)
	}
}

// Evaluation reads only the policy it is given: the environment that names
// external packs is consulted when the caller loads the policy, not here.
func TestEvaluateUsesOnlyTheSuppliedPolicy(t *testing.T) {
	packs, err := compatibility.LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(compatibility.ExternalPacksEnvironment, "relative/not-a-pack-directory")
	if _, err := Evaluate(Policy{Packs: packs, Platform: "darwin/arm64"}, []Package{{ImportPath: "example.com/target"}}); err != nil {
		t.Fatalf("Evaluate() consulted the environment: %v", err)
	}
	if _, err := Evaluate(Policy{Packs: append(packs, packs[0]), Platform: "darwin/arm64"}, nil); err == nil || !strings.Contains(err.Error(), "compatibility pack ID is duplicated") {
		t.Fatalf("Evaluate() with a duplicated pack error = %v", err)
	}
}

func TestBuiltInSimulationLinknamesRequireExactFirstPartySource(t *testing.T) {
	want := builtInSimulationLinknames["runtime_domain.go"]
	pkg := Package{ImportPath: "go.temporal.io/server/tools/gomad3sim", MainModule: true, Policy: compatibility.Package{Module: compatibility.Module{Path: "go.temporal.io/server"}}}
	if !AllowsSimulationBridge(pkg, want) {
		t.Fatal("exact built-in simulation linkname source was rejected")
	}
	testVariant := pkg
	testVariant.ImportPath, testVariant.ForTest = "go.temporal.io/server/tools/gomad3sim [go.temporal.io/server/tools/gomad3sim.test]", "go.temporal.io/server/tools/gomad3sim"
	if !AllowsSimulationBridge(testVariant, want) {
		t.Fatal("exact built-in simulation linkname source was rejected in the test variant")
	}
	changed := want
	changed.SHA256 = "sha256:" + strings.Repeat("0", 64)
	if AllowsSimulationBridge(pkg, changed) {
		t.Fatal("changed built-in simulation linkname source was accepted")
	}
	for name, mutate := range map[string]func(*Package){
		"lookalike module": func(pkg *Package) { pkg.Policy.Module.Path = "example.com/lookalike" },
		"dependency":       func(pkg *Package) { pkg.MainModule = false },
		"replaced module":  func(pkg *Package) { pkg.Policy.Module.Replaced = true },
	} {
		candidate := pkg
		mutate(&candidate)
		if AllowsSimulationBridge(candidate, want) {
			t.Fatalf("%s simulation package was accepted", name)
		}
	}
}
