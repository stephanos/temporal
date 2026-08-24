package compatibility

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"slices"
	"sort"
	"strings"
)

//go:embed packs/*.json
var packFiles embed.FS

type Identity struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type Module struct {
	Path             string
	Version          string
	Sum              string
	Replaced         bool
	LocalReplacement bool
	Adapter          *AdapterEvidence
}

type AdapterEvidence struct {
	ProfileName                      string
	ProfileImplementationSHA256      string
	Module                           string
	Version                          string
	Sum                              string
	OriginalSourceInventorySHA256    string
	ReplacementSourceInventorySHA256 string
	PreparedSourceSetSHA256          string
}

type Package struct {
	ImportPath      string
	Module          Module
	SourceSetSHA256 string
	GoSources       []Source
	ForeignSources  []ForeignSource
}

type Source struct {
	Name   string
	SHA256 string
}

type ForeignSource struct {
	Kind   string
	Name   string
	SHA256 string
}

type Selection struct {
	packs []selectedPack
}

type FactKind string

const (
	FactCapability         FactKind = "capability"
	FactLinkname           FactKind = "linkname"
	FactMalformedLinkname  FactKind = "malformed_linkname"
	FactNoReviewedGoSource FactKind = "no_reviewed_go_source"
)

type Disposition string

const (
	DispositionAllowedExactPack Disposition = "allowed_by_exact_pack"
	DispositionDenied           Disposition = "denied"
)

type RemediationCategory string

const (
	RemediationAddExactPack      RemediationCategory = "add_exact_pack"
	RemediationAddAdapter        RemediationCategory = "add_adapter"
	RemediationModelOperation    RemediationCategory = "model_operation"
	RemediationRemoveDependency  RemediationCategory = "remove_dependency"
	RemediationRemainUnsupported RemediationCategory = "remain_unsupported"
)

type Fact struct {
	Kind       FactKind
	Capability string
	Source     string
	SHA256     string
	Directives []string
}

type Decision struct {
	Allowed     bool                `json:"allowed"`
	Disposition Disposition         `json:"disposition"`
	Remediation RemediationCategory `json:"remediation,omitempty"`
	PackID      string              `json:"pack_id,omitempty"`
}

type PackEvidence struct {
	ID            string                `json:"id"`
	SHA256        string                `json:"sha256"`
	RequestSHA256 string                `json:"request_sha256,omitempty"`
	Governance    *PackGovernance       `json:"governance,omitempty"`
	Activation    []ModuleEvidence      `json:"activation"`
	Rules         []PackageRuleEvidence `json:"rules"`
}

type ModuleEvidence struct {
	Path        string       `json:"path"`
	Version     string       `json:"version"`
	Sum         string       `json:"sum"`
	Replacement string       `json:"replacement"`
	Adapter     *PackAdapter `json:"adapter,omitempty"`
}

type PackageRuleEvidence struct {
	ImportPath      string              `json:"import_path"`
	Module          ModuleEvidence      `json:"module"`
	SourceSetSHA256 string              `json:"source_set_sha256,omitempty"`
	GoSources       []PackSource        `json:"go_sources,omitempty"`
	ForeignSources  []PackForeignSource `json:"foreign_sources,omitempty"`
	Capabilities    []string            `json:"capabilities"`
	Linknames       []LinknameEvidence  `json:"linknames"`
}

type LinknameEvidence struct {
	Source     string   `json:"source"`
	SHA256     string   `json:"sha256"`
	Directives []string `json:"directives"`
}

func Select(packages []Package) (Selection, error) {
	packs, err := loadPacksV2()
	if err != nil {
		return Selection{}, err
	}
	return SelectPacks(packs, packages)
}

func (selection Selection) Evidence() []PackEvidence {
	evidence := make([]PackEvidence, 0, len(selection.packs))
	for _, selected := range selection.packs {
		evidence = append(evidence, projectPackEvidence(selected))
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].ID < evidence[j].ID })
	return evidence
}

func (selection Selection) Identities() []Identity {
	identities := make([]Identity, 0, len(selection.packs))
	for _, selected := range selection.packs {
		identities = append(identities, Identity{ID: selected.pack.ID, SHA256: selected.digest})
	}
	sort.Slice(identities, func(i, j int) bool { return identities[i].ID < identities[j].ID })
	return identities
}

func VerifyIdentities(identities []Identity) error {
	packs, err := loadPacksV2()
	if err != nil {
		return err
	}
	return VerifyPackIdentities(packs, identities)
}

func (selection Selection) AllowsCapability(pkg Package, capability string) bool {
	return selection.Evaluate(pkg, Fact{Kind: FactCapability, Capability: capability}).Allowed
}

func (selection Selection) HasPackage(pkg Package) bool {
	for _, selected := range selection.packs {
		for _, candidate := range selected.pack.Rules {
			if matchesPackRule(candidate, pkg) {
				return true
			}
		}
	}
	return false
}

func (selection Selection) AllowsLinkname(pkg Package, source, digest string, directives []string) bool {
	return selection.Evaluate(pkg, Fact{Kind: FactLinkname, Source: source, SHA256: digest, Directives: directives}).Allowed
}

func (selection Selection) Evaluate(pkg Package, fact Fact) Decision {
	for _, selected := range selection.packs {
		for _, candidate := range selected.pack.Rules {
			if !matchesPackRule(candidate, pkg) {
				continue
			}
			switch fact.Kind {
			case FactCapability:
				if slices.Contains(candidate.Capabilities, fact.Capability) {
					return Decision{Allowed: true, Disposition: DispositionAllowedExactPack, PackID: selected.pack.ID}
				}
			case FactLinkname:
				for _, allowed := range candidate.Linknames {
					if allowed.Source == fact.Source && allowed.SHA256 == fact.SHA256 && slices.Equal(allowed.Directives, fact.Directives) {
						return Decision{Allowed: true, Disposition: DispositionAllowedExactPack, PackID: selected.pack.ID}
					}
				}
			}
		}
	}
	return Decision{Disposition: DispositionDenied, Remediation: remediationFor(pkg, fact)}
}

func DigestSources(sources []Source) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("gomad3.compatibility-source-set/v1\x00"))
	for _, source := range sources {
		_, _ = hash.Write([]byte(source.Name))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(source.SHA256))
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

func remediationFor(pkg Package, fact Fact) RemediationCategory {
	switch fact.Kind {
	case FactNoReviewedGoSource:
		return RemediationRemoveDependency
	case FactMalformedLinkname:
		return RemediationRemainUnsupported
	case FactLinkname:
		if pkg.Module.Path != "" && pkg.Module.Version != "" && (pkg.Module.Sum != "" || pkg.Module.LocalReplacement) {
			return RemediationAddExactPack
		}
		return RemediationRemainUnsupported
	case FactCapability:
		if strings.HasPrefix(fact.Capability, "import:") {
			importPath := strings.TrimPrefix(fact.Capability, "import:")
			if importPath == "os/exec" || importPath == "os/signal" || importPath == "os/user" || importPath == "plugin" || importPath == "runtime/cgo" {
				return RemediationRemainUnsupported
			}
		}
		if pkg.Module.LocalReplacement {
			return RemediationAddAdapter
		}
		if pkg.Module.Path != "" && pkg.Module.Version != "" && pkg.Module.Sum != "" {
			return RemediationAddExactPack
		}
		return RemediationModelOperation
	default:
		return RemediationRemainUnsupported
	}
}

func sortedUnique(values []string) bool {
	for index, value := range values {
		if value == "" || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}
