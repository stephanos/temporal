package compatibility

import (
	"embed"
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

func VerifyIdentities(identities []Identity) error {
	packs, err := loadPacksV2()
	if err != nil {
		return err
	}
	return VerifyPackIdentities(packs, identities)
}
