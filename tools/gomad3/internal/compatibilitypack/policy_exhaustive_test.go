package compatibility

import (
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
)

const (
	policyAllowedJSON     = `{"allowed":true,"disposition":"allowed_by_exact_pack","pack_id":"example-pack"}`
	policyExactPackJSON   = `{"allowed":false,"disposition":"denied","remediation":"add_exact_pack"}`
	policyUnsupportedJSON = `{"allowed":false,"disposition":"denied","remediation":"remain_unsupported"}`
	policyRemoveJSON      = `{"allowed":false,"disposition":"denied","remediation":"remove_dependency"}`
	policyLinknames       = `"linknames": [{"source": "runtime.go", "sha256": "sha256:4444444444444444444444444444444444444444444444444444444444444444", "directives": ["local runtime.first", "other runtime.second"]}]`
)

func TestEvaluateMatchingPackDecisions(t *testing.T) {
	selection, _, pkg := policyMatchingSelection(t)
	tests := []struct {
		name     string
		change   func(*Package, *Fact)
		want     Decision
		wantJSON string
	}{
		{"capability", nil, Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "example-pack"}, policyAllowedJSON},
		{"linkname", func(_ *Package, fact *Fact) { fact.Kind = FactLinkname }, Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "example-pack"}, policyAllowedJSON},
		{"malformed-linkname", func(_ *Package, fact *Fact) { fact.Kind = FactMalformedLinkname }, Decision{Disposition: "denied", Remediation: "remain_unsupported"}, policyUnsupportedJSON},
		{"no-reviewed-source", func(_ *Package, fact *Fact) { fact.Kind = FactNoReviewedGoSource }, Decision{Disposition: "denied", Remediation: "remove_dependency"}, policyRemoveJSON},
		{"unknown-kind", func(_ *Package, fact *Fact) { fact.Kind = "future-kind" }, Decision{Disposition: "denied", Remediation: "remain_unsupported"}, policyUnsupportedJSON},
		{"empty-kind", func(_ *Package, fact *Fact) { fact.Kind = "" }, Decision{Disposition: "denied", Remediation: "remain_unsupported"}, policyUnsupportedJSON},
		{"capability-drift", func(_ *Package, fact *Fact) { fact.Capability = "import:unsafe" }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"linkname-source-drift", func(_ *Package, fact *Fact) { fact.Kind = FactLinkname; fact.Source = "other.go" }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"linkname-hash-drift", func(_ *Package, fact *Fact) {
			fact.Kind = FactLinkname
			fact.SHA256 = "sha256:5555555555555555555555555555555555555555555555555555555555555555"
		}, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"linkname-directive-drift", func(_ *Package, fact *Fact) { fact.Kind = FactLinkname; fact.Directives[0] = "changed runtime.first" }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"linkname-order-drift", func(_ *Package, fact *Fact) { fact.Kind = FactLinkname; slices.Reverse(fact.Directives) }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"linkname-nil-directives", func(_ *Package, fact *Fact) { fact.Kind = FactLinkname; fact.Directives = nil }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"linkname-empty-directives", func(_ *Package, fact *Fact) { fact.Kind = FactLinkname; fact.Directives = []string{} }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"package-path-drift", func(pkg *Package, _ *Fact) { pkg.ImportPath = "example.com/dependency/other" }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"module-path-drift", func(pkg *Package, _ *Fact) { pkg.Module.Path = "example.com/other" }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"module-version-drift", func(pkg *Package, _ *Fact) { pkg.Module.Version = "v1.2.4" }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"module-sum-drift", func(pkg *Package, _ *Fact) { pkg.Module.Sum = "h1:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=" }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"replacement-drift", func(pkg *Package, _ *Fact) { pkg.Module.Replaced = true }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"local-replacement", func(pkg *Package, _ *Fact) { pkg.Module.LocalReplacement = true }, Decision{Disposition: "denied", Remediation: "add_adapter"}, `{"allowed":false,"disposition":"denied","remediation":"add_adapter"}`},
		{"missing-module", func(pkg *Package, _ *Fact) { pkg.Module = Module{} }, Decision{Disposition: "denied", Remediation: "model_operation"}, `{"allowed":false,"disposition":"denied","remediation":"model_operation"}`},
		{"linkname-missing-module", func(pkg *Package, fact *Fact) { pkg.Module = Module{}; fact.Kind = FactLinkname }, Decision{Disposition: "denied", Remediation: "remain_unsupported"}, policyUnsupportedJSON},
		{"source-set-drift", func(pkg *Package, _ *Fact) {
			pkg.SourceSetSHA256 = "sha256:5555555555555555555555555555555555555555555555555555555555555555"
		}, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"source-name-drift", func(pkg *Package, _ *Fact) { pkg.GoSources[0].Name = "other.go" }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"source-hash-drift", func(pkg *Package, _ *Fact) {
			pkg.GoSources[0].SHA256 = "sha256:5555555555555555555555555555555555555555555555555555555555555555"
		}, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"source-nil-inventory", func(pkg *Package, _ *Fact) { pkg.GoSources = nil }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"source-empty-inventory", func(pkg *Package, _ *Fact) { pkg.GoSources = []Source{} }, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"source-extra-inventory", func(pkg *Package, _ *Fact) {
			pkg.GoSources = append(pkg.GoSources, Source{Name: "z.go", SHA256: pkg.GoSources[0].SHA256})
		}, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"foreign-source-drift", func(pkg *Package, _ *Fact) {
			pkg.ForeignSources = []ForeignSource{{Kind: "assembly", Name: "native.s", SHA256: pkg.GoSources[0].SHA256}}
		}, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{"nil-empty-foreign-equivalence", func(pkg *Package, _ *Fact) { pkg.ForeignSources = nil }, Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "example-pack"}, policyAllowedJSON},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualPackage := clonePolicyPackage(pkg)
			fact := policyMatchingFact()
			if test.change != nil {
				test.change(&actualPackage, &fact)
			}
			requirePolicyDecision(t, selection.Evaluate(actualPackage, fact), test.want, test.wantJSON)
		})
	}
}

func TestEvaluateHostImportDecisions(t *testing.T) {
	selection, _, pkg := policyMatchingSelection(t)
	tests := []struct{ capability, wantError string }{
		{"import:os/exec", "compatibility pack rule 0: capability import:os/exec is never admitted"},
		{"import:os/signal", "compatibility pack rule 0: capability import:os/signal is never admitted"},
		{"import:os/user", "compatibility pack rule 0: capability import:os/user is never admitted"},
		{"import:plugin", ""},
		{"import:runtime/cgo", ""},
	}
	for _, test := range tests {
		t.Run(test.capability, func(t *testing.T) {
			fact := policyMatchingFact()
			fact.Capability = test.capability
			requirePolicyDecision(t, selection.Evaluate(pkg, fact), Decision{Disposition: "denied", Remediation: "remain_unsupported"}, policyUnsupportedJSON)
			encoded := strings.Replace(validPackV2, `"import:syscall"`, `"`+test.capability+`"`, 1)
			loaded, err := LoadPack([]byte(encoded))
			if test.wantError == "" {
				requireTestNoError(t, err)
				t.Log("loader_error=<nil>")
				selected, err := SelectPacksForPlatform([]ValidatedPack{loaded}, []Package{pkg}, "darwin/arm64")
				requireTestNoError(t, err)
				if !selected.HasPackage(pkg) {
					t.Fatal("accepted fixture did not match its package")
				}
				requirePolicyDecision(t, selected.Evaluate(pkg, fact), Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "example-pack"}, policyAllowedJSON)
				return
			}
			if err == nil {
				t.Fatal("LoadPack admitted a forbidden host import")
			}
			requireTestEqual(t, test.wantError, err.Error())
		})
	}
}

func TestEvaluateNilAndEmptySelections(t *testing.T) {
	_, _, pkg := policyMatchingSelection(t)
	tests := []struct {
		kind     FactKind
		want     Decision
		wantJSON string
	}{
		{FactCapability, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{FactLinkname, Decision{Disposition: "denied", Remediation: "add_exact_pack"}, policyExactPackJSON},
		{FactMalformedLinkname, Decision{Disposition: "denied", Remediation: "remain_unsupported"}, policyUnsupportedJSON},
		{FactNoReviewedGoSource, Decision{Disposition: "denied", Remediation: "remove_dependency"}, policyRemoveJSON},
		{"future-kind", Decision{Disposition: "denied", Remediation: "remain_unsupported"}, policyUnsupportedJSON},
		{"", Decision{Disposition: "denied", Remediation: "remain_unsupported"}, policyUnsupportedJSON},
	}
	for _, empty := range []struct {
		name      string
		selection Selection
	}{{"nil", Selection{}}, {"empty", Selection{packs: []selectedPack{}}}} {
		t.Run(empty.name, func(t *testing.T) {
			selection := empty.selection
			requireTestEqual(t, []Identity{}, selection.Identities())
			if selection.HasPackage(pkg) {
				t.Fatal("empty selection matched a package")
			}
			for _, test := range tests {
				t.Run(string(test.kind), func(t *testing.T) {
					fact := policyMatchingFact()
					fact.Kind = test.kind
					requirePolicyDecision(t, selection.Evaluate(pkg, fact), test.want, test.wantJSON)
				})
			}
		})
	}
}

func TestEvaluateTraversesRulesAndPacks(t *testing.T) {
	_, validated, pkg := policyMatchingSelection(t)
	t.Run("later-matching-rule", func(t *testing.T) {
		pack := validated.Pack()
		first := validated.Pack().Rules[0]
		first.ImportPath = "example.com/dependency/aaa"
		pack.Rules = append([]PackRule{first}, pack.Rules...)
		selection, err := SelectPacksForPlatform([]ValidatedPack{loadPolicyTestPack(t, pack)}, []Package{pkg}, "darwin/arm64")
		requireTestNoError(t, err)
		if !selection.HasPackage(pkg) {
			t.Fatal("later rule did not match")
		}
		for _, kind := range []FactKind{FactCapability, FactLinkname} {
			fact := policyMatchingFact()
			fact.Kind = kind
			requirePolicyDecision(t, selection.Evaluate(pkg, fact), Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "example-pack"}, policyAllowedJSON)
		}
	})
	tests := []struct {
		name      string
		firstPath string
		firstCap  string
	}{
		{"later-matching-pack", "example.com/dependency/aaa", "import:syscall"},
		{"later-granting-pack", "example.com/dependency/internal/runtime", "import:unsafe"},
		{"first-grant-precedence", "example.com/dependency/internal/runtime", "import:syscall"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first, last := validated.Pack(), validated.Pack()
			first.ID, last.ID = "a-pack", "z-pack"
			first.Rules[0].ImportPath = test.firstPath
			first.Rules[0].Capabilities = []string{test.firstCap}
			if test.name != "first-grant-precedence" {
				first.Rules[0].Linknames = []PackLinkname{}
			}
			packages := append(generatedExactPackages(first), clonePolicyPackage(pkg))
			selection, err := SelectPacksForPlatform([]ValidatedPack{loadPolicyTestPack(t, last), loadPolicyTestPack(t, first)}, packages, "darwin/arm64")
			requireTestNoError(t, err)
			identities := selection.Identities()
			requireTestEqual(t, 2, len(identities))
			requireTestEqual(t, []string{"a-pack", "z-pack"}, []string{identities[0].ID, identities[1].ID})
			if !selection.HasPackage(pkg) {
				t.Fatal("selected packs did not match")
			}
			for _, kind := range []FactKind{FactCapability, FactLinkname} {
				fact := policyMatchingFact()
				fact.Kind = kind
				if test.name == "first-grant-precedence" {
					requirePolicyDecision(t, selection.Evaluate(pkg, fact), Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "a-pack"}, `{"allowed":true,"disposition":"allowed_by_exact_pack","pack_id":"a-pack"}`)
				} else {
					requirePolicyDecision(t, selection.Evaluate(pkg, fact), Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "z-pack"}, `{"allowed":true,"disposition":"allowed_by_exact_pack","pack_id":"z-pack"}`)
				}
			}
		})
	}
}

func TestEvaluatePreservesInputsAndDetachedCopies(t *testing.T) {
	selection, validated, pkg := policyMatchingSelection(t)
	fact := policyMatchingFact()
	beforePackage, beforeFact := clonePolicyPackage(pkg), policyMatchingFact()
	for _, kind := range []FactKind{FactCapability, FactLinkname} {
		fact.Kind, beforeFact.Kind = kind, kind
		requirePolicyDecision(t, selection.Evaluate(pkg, fact), Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "example-pack"}, policyAllowedJSON)
		requireTestEqual(t, beforePackage, pkg)
		requireTestEqual(t, beforeFact, fact)
	}
	identities := selection.Identities()
	identities[0].ID, identities[0].SHA256 = "changed", "changed"
	evidence := selection.Evidence()
	evidence[0].ID = "changed"
	evidence[0].Governance.Platforms[0] = "linux/amd64"
	evidence[0].Governance.Workloads[0] = "changed"
	evidence[0].Activation[0].Version = "v9.0.0"
	evidence[0].Rules[0].Module.Sum = "changed"
	evidence[0].Rules[0].GoSources[0].SHA256 = "changed"
	evidence[0].Rules[0].Capabilities[0] = "import:unsafe"
	evidence[0].Rules[0].Linknames[0].Directives[0] = "changed runtime.first"
	packCopy := validated.Pack()
	packCopy.Rules[0].GoSources[0].SHA256 = "changed"
	packCopy.Rules[0].Capabilities[0] = "import:unsafe"
	packCopy.Rules[0].Linknames[0].Directives[0] = "changed runtime.first"
	requireTestEqual(t, []Identity{{ID: "example-pack", SHA256: "sha256:535a48101ccd58f1e158b5ad89c1cb2182aa302ba8799bf80bf539181b1a9df2"}}, selection.Identities())
	for _, kind := range []FactKind{FactCapability, FactLinkname} {
		fact.Kind = kind
		requirePolicyDecision(t, selection.Evaluate(pkg, fact), Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "example-pack"}, policyAllowedJSON)
	}
	packages := []Package{clonePolicyPackage(pkg)}
	packs := []ValidatedPack{validated}
	selected, err := SelectPacksForPlatform(packs, packages, "darwin/arm64")
	requireTestNoError(t, err)
	packs[0] = ValidatedPack{}
	packages[0].GoSources[0].SHA256 = "changed"
	fact.Directives[0] = "changed runtime.first"
	requirePolicyDecision(t, selected.Evaluate(beforePackage, policyMatchingFact()), Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "example-pack"}, policyAllowedJSON)
	linkname := policyMatchingFact()
	linkname.Kind = FactLinkname
	requirePolicyDecision(t, selected.Evaluate(beforePackage, linkname), Decision{Allowed: true, Disposition: "allowed_by_exact_pack", PackID: "example-pack"}, policyAllowedJSON)
}

func policyMatchingSelection(t *testing.T) (Selection, ValidatedPack, Package) {
	t.Helper()
	encoded := strings.Replace(validPackV2, `"linknames": []`, policyLinknames, 1)
	validated, err := LoadPack([]byte(encoded))
	requireTestNoError(t, err)
	requireTestEqual(t, "sha256:535a48101ccd58f1e158b5ad89c1cb2182aa302ba8799bf80bf539181b1a9df2", validated.SHA256())
	requireTestEqual(t, []string{"darwin/arm64"}, validated.Pack().Governance.Platforms)
	packages := generatedExactPackages(validated.Pack())
	pkg := generatedPackageForTest(t, packages, "example.com/dependency/internal/runtime")
	selection, err := SelectPacksForPlatform([]ValidatedPack{validated}, packages, "darwin/arm64")
	requireTestNoError(t, err)
	requireTestEqual(t, []Identity{{ID: "example-pack", SHA256: "sha256:535a48101ccd58f1e158b5ad89c1cb2182aa302ba8799bf80bf539181b1a9df2"}}, selection.Identities())
	if !selection.HasPackage(pkg) || !selection.AllowsCapability(pkg, "import:syscall") || !selection.AllowsLinkname(pkg, "runtime.go", "sha256:4444444444444444444444444444444444444444444444444444444444444444", []string{"local runtime.first", "other runtime.second"}) {
		t.Fatal("validated selected fixture did not grant its exact facts")
	}
	wrongPlatform, err := SelectPacksForPlatform([]ValidatedPack{validated}, packages, "linux/amd64")
	requireTestNoError(t, err)
	requireTestEqual(t, []Identity{}, wrongPlatform.Identities())
	if wrongPlatform.HasPackage(pkg) {
		t.Fatal("unapproved platform matched fixture")
	}
	return selection, validated, pkg
}

func policyMatchingFact() Fact {
	return Fact{Kind: FactCapability, Capability: "import:syscall", Source: "runtime.go", SHA256: "sha256:4444444444444444444444444444444444444444444444444444444444444444", Directives: []string{"local runtime.first", "other runtime.second"}}
}

func clonePolicyPackage(pkg Package) Package {
	pkg.GoSources = slices.Clone(pkg.GoSources)
	pkg.ForeignSources = slices.Clone(pkg.ForeignSources)
	if pkg.Module.Adapter != nil {
		adapter := *pkg.Module.Adapter
		pkg.Module.Adapter = &adapter
	}
	return pkg
}

func loadPolicyTestPack(t *testing.T, pack Pack) ValidatedPack {
	t.Helper()
	encoded, err := canonicaljson.CanonicalJSON(pack)
	requireTestNoError(t, err)
	validated, err := LoadPack(encoded)
	requireTestNoError(t, err)
	return validated
}

func requirePolicyDecision(t *testing.T, got, want Decision, wantJSON string) {
	t.Helper()
	requireTestEqual(t, want, got)
	encoded, err := canonicaljson.CanonicalJSON(got)
	requireTestNoError(t, err)
	requireTestEqual(t, wantJSON, string(encoded))
	t.Logf("decision=%#v bytes=%q", got, encoded)
}
