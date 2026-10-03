// Package capabilitypolicy evaluates collected target capability evidence
// against the compatibility policy. Evaluation is a pure function of its
// inputs: the caller collects source evidence and loads compatibility packs,
// and this package reads no file or environment, starts no process and
// observes no clock.
package capabilitypolicy

import (
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
)

type Kind string

const (
	KindForbiddenImport    Kind = "forbidden_import"
	KindForeignSource      Kind = "foreign_source"
	KindUnapprovedLinkname Kind = "unapproved_linkname"
	KindMalformedLinkname  Kind = "malformed_linkname"
	KindNoReviewedGoSource Kind = "no_reviewed_go_source"
)

// Policy is the loaded compatibility policy: the validated packs a build
// selects from and the platform that scopes them.
type Policy struct {
	Packs    []compatibility.ValidatedPack
	Platform string
}

// Package is the collected evidence for one closure package. Policy carries
// the package's identity, module and source set as the compatibility policy
// matches them.
type Package struct {
	ImportPath        string
	ForTest           string
	Standard          bool
	GeneratedTestMain bool
	// MainModule reports that the package belongs to the main module.
	MainModule bool
	Imports    []string
	Sources    []Source
	Policy     compatibility.Package
}

type Source struct {
	Name               string
	SHA256             string
	LinknameDirectives []string
	MalformedLinkname  bool
}

// Finding is one capability the policy denies. Package indexes the evaluated
// packages; findings follow package order and, within a package, imports,
// foreign sources, linknames and missing Go source.
type Finding struct {
	Package      int
	Kind         Kind
	Fact         compatibility.Fact
	SourceName   string
	SourceSHA256 string
	Decision     compatibility.Decision
}

type Evaluation struct {
	Selection compatibility.Selection
	Findings  []Finding
}

// Evaluate selects the compatibility packs that apply to packages and
// evaluates every package against that selection.
func Evaluate(policy Policy, packages []Package) (Evaluation, error) {
	policyPackages := make([]compatibility.Package, len(packages))
	for index, pkg := range packages {
		policyPackages[index] = pkg.Policy
	}
	selection, err := compatibility.SelectPacksForPlatform(policy.Packs, policyPackages, policy.Platform)
	if err != nil {
		return Evaluation{}, err
	}
	findings := []Finding{}
	for index, pkg := range packages {
		if pkg.Standard {
			continue
		}
		findings = append(findings, packageFindings(index, pkg, selection)...)
	}
	return Evaluation{Selection: selection, Findings: findings}, nil
}

func packageFindings(index int, pkg Package, selection compatibility.Selection) []Finding {
	findings := []Finding{}
	for _, imported := range pkg.Imports {
		if !forbiddenImport(imported) {
			continue
		}
		fact := compatibility.Fact{Kind: compatibility.FactCapability, Capability: "import:" + imported}
		if decision := selection.Evaluate(pkg.Policy, fact); !decision.Allowed {
			findings = append(findings, Finding{Package: index, Kind: KindForbiddenImport, Fact: fact, Decision: decision})
		}
	}
	for _, source := range pkg.Policy.ForeignSources {
		// Headers remain source-set evidence, but cannot execute without a separately reviewed compiled foreign input.
		if source.Kind == "header" {
			continue
		}
		fact := compatibility.Fact{Kind: compatibility.FactCapability, Capability: "foreign:" + source.Kind + ":" + source.Name}
		if decision := selection.Evaluate(pkg.Policy, fact); !decision.Allowed {
			findings = append(findings, Finding{Package: index, Kind: KindForeignSource, Fact: fact, SourceName: source.Name, SourceSHA256: source.SHA256, Decision: decision})
		}
	}
	for _, source := range pkg.Sources {
		fact, kind, present := linknameFact(source)
		if !present {
			continue
		}
		if AllowsSimulationBridge(pkg, source) {
			continue
		}
		if decision := selection.Evaluate(pkg.Policy, fact); !decision.Allowed {
			findings = append(findings, Finding{Package: index, Kind: kind, Fact: fact, SourceName: source.Name, SourceSHA256: source.SHA256, Decision: decision})
		}
	}
	if !pkg.GeneratedTestMain && len(pkg.Sources) == 0 {
		fact := compatibility.Fact{Kind: compatibility.FactNoReviewedGoSource, Capability: "source:go"}
		findings = append(findings, Finding{Package: index, Kind: KindNoReviewedGoSource, Fact: fact, Decision: selection.Evaluate(pkg.Policy, fact)})
	}
	return findings
}

func forbiddenImport(importPath string) bool {
	return importPath == "syscall" || importPath == "os/exec" || importPath == "os/signal" || importPath == "os/user" || importPath == "plugin" || importPath == "runtime/cgo" || strings.HasPrefix(importPath, "golang.org/x/sys/")
}

func linknameFact(source Source) (compatibility.Fact, Kind, bool) {
	if source.MalformedLinkname {
		return compatibility.Fact{Kind: compatibility.FactMalformedLinkname, Capability: "linkname:malformed", Source: source.Name, SHA256: source.SHA256, Directives: []string{}}, KindMalformedLinkname, true
	}
	if len(source.LinknameDirectives) == 0 {
		return compatibility.Fact{}, "", false
	}
	return compatibility.Fact{
		Kind: compatibility.FactLinkname, Capability: "linkname:" + source.Name, Source: source.Name,
		SHA256: source.SHA256, Directives: source.LinknameDirectives,
	}, KindUnapprovedLinkname, true
}
