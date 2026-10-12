package target

import (
	"errors"
	"fmt"
)

const CapabilityClosureSchema = "gomad3.target-capability-closure/v3"

const CapabilityReviewSchema = "gomad3.target-capability-review/v4"

type AdapterCapacityError struct {
	Resource string
	Limit    uint64
}

func (err *AdapterCapacityError) Error() string {
	return fmt.Sprintf("adapter module exceeds %s limit %d", err.Resource, err.Limit)
}

type InvalidCapabilityReviewError struct {
	Err error
}

func (err *InvalidCapabilityReviewError) Error() string {
	return err.Err.Error()
}

func (err *InvalidCapabilityReviewError) Unwrap() error {
	return err.Err
}

func IsInvalidCapabilityReview(err error) bool {
	var invalid *InvalidCapabilityReviewError
	return errors.As(err, &invalid)
}

type CapabilityClosure struct {
	Schema        string                  `json:"schema"`
	Compatibility []CompatibilityIdentity `json:"compatibility"`
	Packages      []CapabilityPackage     `json:"packages"`
}

type CapabilityPackage struct {
	ImportPath        string                    `json:"import_path"`
	ForTest           string                    `json:"for_test,omitempty"`
	Name              string                    `json:"name"`
	Root              bool                      `json:"root,omitempty"`
	Standard          bool                      `json:"standard"`
	Imports           []string                  `json:"imports"`
	Module            *CapabilityModule         `json:"module,omitempty"`
	Sources           []CapabilitySource        `json:"sources"`
	ForeignSources    []CapabilityForeignSource `json:"foreign_sources"`
	GeneratedTestMain bool                      `json:"generated_test_main,omitempty"`
}

type CapabilityModule struct {
	Path        string                        `json:"path"`
	Version     string                        `json:"version"`
	Sum         string                        `json:"sum"`
	Main        bool                          `json:"main"`
	Local       bool                          `json:"local"`
	Replacement *CapabilityModule             `json:"replacement,omitempty"`
	Adapter     *CapabilityAdapterReplacement `json:"adapter,omitempty"`
}

type CapabilityAdapterReplacement struct {
	ProfileName                      string         `json:"profile_name"`
	ProfileImplementationSHA256      string         `json:"profile_implementation_sha256"`
	Adapter                          ModuleIdentity `json:"adapter"`
	OriginalSourceInventorySHA256    string         `json:"original_source_inventory_sha256"`
	ReplacementSourceInventorySHA256 string         `json:"replacement_source_inventory_sha256"`
	PreparedSourceSetSHA256          string         `json:"prepared_source_set_sha256"`
}

type CapabilitySource struct {
	Name               string   `json:"name"`
	SHA256             string   `json:"sha256"`
	LinknameDirectives []string `json:"linkname_directives,omitempty"`
	MalformedLinkname  bool     `json:"malformed_linkname,omitempty"`
}

type CapabilityForeignSource struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type CapabilityPackageReference struct {
	ImportPath string `json:"import_path"`
	ForTest    string `json:"for_test,omitempty"`
	Name       string `json:"name"`
}

type CompatibilityIdentity struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

// CompatibilityPackEvidence is the complete detached evidence selected by policy.
type CompatibilityPackEvidence struct {
	ID            string                             `json:"id"`
	SHA256        string                             `json:"sha256"`
	RequestSHA256 string                             `json:"request_sha256,omitempty"`
	Governance    *CompatibilityPackGovernance       `json:"governance,omitempty"`
	Activation    []CompatibilityModuleEvidence      `json:"activation"`
	Rules         []CompatibilityPackageRuleEvidence `json:"rules"`
}

type CompatibilityPackGovernance struct {
	Owner          string   `json:"owner"`
	ReviewedAt     string   `json:"reviewed_at"`
	Justification  string   `json:"justification"`
	Workloads      []string `json:"workloads"`
	Platforms      []string `json:"platforms"`
	ApprovalSHA256 string   `json:"approval_sha256"`
}

type CompatibilityModuleEvidence struct {
	Path        string                    `json:"path"`
	Version     string                    `json:"version"`
	Sum         string                    `json:"sum"`
	Replacement string                    `json:"replacement"`
	Adapter     *CompatibilityPackAdapter `json:"adapter,omitempty"`
}

type CompatibilityPackAdapter struct {
	ProfileName                      string `json:"profile_name"`
	ProfileImplementationSHA256      string `json:"profile_implementation_sha256"`
	Module                           string `json:"module"`
	Version                          string `json:"version"`
	Sum                              string `json:"sum"`
	OriginalSourceInventorySHA256    string `json:"original_source_inventory_sha256"`
	ReplacementSourceInventorySHA256 string `json:"replacement_source_inventory_sha256"`
	PreparedSourceSetSHA256          string `json:"prepared_source_set_sha256"`
}

type CompatibilityPackageRuleEvidence struct {
	ImportPath      string                           `json:"import_path"`
	Module          CompatibilityModuleEvidence      `json:"module"`
	SourceSetSHA256 string                           `json:"source_set_sha256,omitempty"`
	GoSources       []CompatibilityPackSource        `json:"go_sources,omitempty"`
	ForeignSources  []CompatibilityPackForeignSource `json:"foreign_sources,omitempty"`
	Capabilities    []string                         `json:"capabilities"`
	Linknames       []CompatibilityLinknameEvidence  `json:"linknames"`
}

type CompatibilityPackSource struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type CompatibilityPackForeignSource struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type CompatibilityLinknameEvidence struct {
	Source     string   `json:"source"`
	SHA256     string   `json:"sha256"`
	Directives []string `json:"directives"`
}

type CompatibilityDisposition string

type CompatibilityRemediation string

const (
	DispositionAllowedExactPack CompatibilityDisposition = CompatibilityDisposition("allowed_by_exact_pack")
	DispositionDenied           CompatibilityDisposition = CompatibilityDisposition("denied")

	RemediationAddExactPack      CompatibilityRemediation = CompatibilityRemediation("add_exact_pack")
	RemediationAddAdapter        CompatibilityRemediation = CompatibilityRemediation("add_adapter")
	RemediationModelOperation    CompatibilityRemediation = CompatibilityRemediation("model_operation")
	RemediationRemoveDependency  CompatibilityRemediation = CompatibilityRemediation("remove_dependency")
	RemediationRemainUnsupported CompatibilityRemediation = CompatibilityRemediation("remain_unsupported")
)

type CapabilityFindingKind string

const (
	FindingForbiddenImport    CapabilityFindingKind = CapabilityFindingKind("forbidden_import")
	FindingForeignSource      CapabilityFindingKind = CapabilityFindingKind("foreign_source")
	FindingUnapprovedLinkname CapabilityFindingKind = CapabilityFindingKind("unapproved_linkname")
	FindingMalformedLinkname  CapabilityFindingKind = CapabilityFindingKind("malformed_linkname")
	FindingNoReviewedGoSource CapabilityFindingKind = CapabilityFindingKind("no_reviewed_go_source")
	FindingDeniedBoundary     CapabilityFindingKind = "denied_boundary"
)

type CapabilityFinding struct {
	Kind              CapabilityFindingKind      `json:"kind"`
	Package           CapabilityPackageReference `json:"package"`
	Module            *CapabilityModule          `json:"module,omitempty"`
	SourceSetSHA256   string                     `json:"source_set_sha256"`
	SourceName        string                     `json:"source_name,omitempty"`
	SourceSHA256      string                     `json:"source_sha256,omitempty"`
	Directives        []string                   `json:"directives"`
	Capability        string                     `json:"capability"`
	PolicyDisposition CompatibilityDisposition   `json:"policy_disposition"`
	Remediation       CompatibilityRemediation   `json:"remediation"`
	PackID            string                     `json:"pack_id,omitempty"`
}

type CapabilityReview struct {
	Schema             string                       `json:"schema"`
	BuildTags          []string                     `json:"build_tags"`
	Roots              []CapabilityPackageReference `json:"roots"`
	Closure            CapabilityClosure            `json:"closure"`
	Packs              []CompatibilityPackEvidence  `json:"packs"`
	CapabilityMode     CapabilityMode               `json:"capability_mode"`
	CapabilityManifest *CapabilityManifest          `json:"capability_manifest,omitempty"`
	Findings           []CapabilityFinding          `json:"findings"`
	GuardedFindings    []CapabilityFinding          `json:"guarded_findings"`
	EliminatedFindings []CapabilityFinding          `json:"eliminated_findings"`
}

type UnsupportedCapabilityError struct {
	ImportPath string
	Capability string
	Finding    CapabilityFinding
}

func (err *UnsupportedCapabilityError) Error() string {
	return fmt.Sprintf("unsupported target capability: package %s %s", err.ImportPath, err.Capability)
}
