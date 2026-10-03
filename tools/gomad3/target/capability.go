package target

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/record"
	targetbuild "go.temporal.io/server/tools/gomad3/target/internal/build"
	"go.temporal.io/server/tools/gomad3/target/internal/capabilitypolicy"
	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
	"go.temporal.io/server/tools/gomad3/toolchain/installation"
)

const CapabilityClosureSchema = "gomad3.target-capability-closure/v3"
const CapabilityReviewSchema = "gomad3.target-capability-review/v4"
const maximumCapabilityReviewOutputBytes = 64 << 20
const maximumStandardPackagesBytes = 4 << 20
const maximumCapabilityReviewPackages = 100000
const maximumCapabilitySourceBytes = 16 << 20

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

func invalidCapabilityReview(err error) error {
	return &InvalidCapabilityReviewError{Err: err}
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

type CompatibilityIdentity compatibility.Identity
type CompatibilityPackEvidence compatibility.PackEvidence
type CompatibilityDisposition compatibility.Disposition
type CompatibilityRemediation compatibility.RemediationCategory

const (
	DispositionAllowedExactPack CompatibilityDisposition = CompatibilityDisposition(compatibility.DispositionAllowedExactPack)
	DispositionDenied           CompatibilityDisposition = CompatibilityDisposition(compatibility.DispositionDenied)

	RemediationAddExactPack      CompatibilityRemediation = CompatibilityRemediation(compatibility.RemediationAddExactPack)
	RemediationAddAdapter        CompatibilityRemediation = CompatibilityRemediation(compatibility.RemediationAddAdapter)
	RemediationModelOperation    CompatibilityRemediation = CompatibilityRemediation(compatibility.RemediationModelOperation)
	RemediationRemoveDependency  CompatibilityRemediation = CompatibilityRemediation(compatibility.RemediationRemoveDependency)
	RemediationRemainUnsupported CompatibilityRemediation = CompatibilityRemediation(compatibility.RemediationRemainUnsupported)
)

type CapabilityFindingKind string

const (
	FindingForbiddenImport    CapabilityFindingKind = CapabilityFindingKind(capabilitypolicy.KindForbiddenImport)
	FindingForeignSource      CapabilityFindingKind = CapabilityFindingKind(capabilitypolicy.KindForeignSource)
	FindingUnapprovedLinkname CapabilityFindingKind = CapabilityFindingKind(capabilitypolicy.KindUnapprovedLinkname)
	FindingMalformedLinkname  CapabilityFindingKind = CapabilityFindingKind(capabilitypolicy.KindMalformedLinkname)
	FindingNoReviewedGoSource CapabilityFindingKind = CapabilityFindingKind(capabilitypolicy.KindNoReviewedGoSource)
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

func ReviewCapabilityClosure(ctx context.Context, spec Spec) (CapabilityClosure, error) {
	review, err := ReviewCapabilities(ctx, spec)
	if err != nil {
		return CapabilityClosure{}, err
	}
	if len(review.Findings) != 0 {
		return CapabilityClosure{}, unsupportedFinding(review.Findings[0])
	}
	return review.Closure, nil
}

func ReviewCapabilities(ctx context.Context, spec Spec) (CapabilityReview, error) {
	if spec.Kind != KindGoRun && spec.Kind != KindGoTest {
		return CapabilityReview{}, invalidCapabilityReview(errors.New("capability review requires a go-run or go-test target"))
	}
	tags, err := targetbuild.NormalizeTags(spec.BuildTags)
	if err != nil {
		return CapabilityReview{}, invalidCapabilityReview(err)
	}
	mode, err := normalizeCapabilityMode(spec.CapabilityMode)
	if err != nil {
		return CapabilityReview{}, invalidCapabilityReview(err)
	}
	spec.CapabilityMode = mode
	if spec.Source == "" || spec.WorkingDir == "" || spec.ToolchainRoot == "" {
		return CapabilityReview{}, invalidCapabilityReview(errors.New("capability review requires source, working directory, and toolchain root"))
	}
	if strings.HasPrefix(spec.Source, "-") || strings.Contains(spec.Source, "...") || strings.IndexFunc(spec.Source, unicode.IsSpace) >= 0 || strings.IndexByte(spec.Source, 0) >= 0 {
		return CapabilityReview{}, invalidCapabilityReview(fmt.Errorf("go target package argument %q must select exactly one package", spec.Source))
	}
	layout, err := installation.At(spec.ToolchainRoot)
	if err != nil {
		return CapabilityReview{}, fmt.Errorf("resolve pinned Go command: %w", err)
	}
	goCommand := layout.GoCommand()
	buildContext, err := targetbuild.Resolve(spec.WorkingDir, spec.Source, tags)
	if err != nil {
		return CapabilityReview{}, invalidCapabilityReview(err)
	}
	review, err := reviewGoCapabilityReview(ctx, goCommand, spec, buildContext.Tags, buildContext.Directory, buildContext.Package)
	if err != nil {
		return CapabilityReview{}, err
	}
	if mode == CapabilityModeClosure {
		return review, nil
	}
	if spec.PreparationRoot == "" {
		return CapabilityReview{}, invalidCapabilityReview(errors.New("linked capability review requires a preparation root"))
	}
	identity, err := readPinnedToolchainWith(context.Background(), spec.ToolchainRoot, gocommand.Default())
	if err != nil {
		return CapabilityReview{}, err
	}
	workspace, err := os.MkdirTemp(spec.PreparationRoot, ".linked-review-")
	if err != nil {
		return CapabilityReview{}, fmt.Errorf("create linked capability review workspace: %w", err)
	}
	prepared, buildErr := buildGoTarget(ctx, spec, buildContext.Tags, identity, filepath.Join(workspace, "target"), goCommand, buildContext.Directory, buildContext.Package, review, allowUnsupported, nil)
	cleanupErr := os.RemoveAll(workspace)
	if buildErr != nil || cleanupErr != nil {
		return CapabilityReview{}, errors.Join(buildErr, cleanupErr)
	}
	return prepared.review, nil
}

func reviewGoCapabilityReview(ctx context.Context, goCommand string, spec Spec, tags []string, commandDirectory, packageArgument string) (CapabilityReview, error) {
	review, _, err := reviewGoCapabilityPackages(ctx, goCommand, spec, tags, commandDirectory, packageArgument)
	return review, err
}

// reviewGoCapabilityPackages also returns the listing the review projected,
// which preparation needs for the build inputs the closure does not review.
func reviewGoCapabilityPackages(ctx context.Context, goCommand string, spec Spec, tags []string, commandDirectory, packageArgument string) (CapabilityReview, []listedPackage, error) {
	return reviewGoCapabilityPackagesWith(ctx, goCommand, spec, tags, commandDirectory, packageArgument, gocommand.Default())
}

func reviewGoCapabilityPackagesWith(ctx context.Context, goCommand string, spec Spec, tags []string, commandDirectory, packageArgument string, runner gocommand.Runner) (CapabilityReview, []listedPackage, error) {
	packages, overlay, err := collectCapabilityListing(ctx, goCommand, spec, tags, commandDirectory, packageArgument, runner)
	if err != nil {
		return CapabilityReview{}, nil, err
	}
	review, err := projectCapabilityReview(packages, overlay, tags, spec.AdapterReplacements)
	if err != nil {
		return CapabilityReview{}, nil, err
	}
	return review, packages, nil
}

// projectCapabilityReview collects the capability evidence of a listed
// closure and the compatibility policy, then evaluates the evidence against it.
func projectCapabilityReview(packages []listedPackage, overlay map[string]string, tags []string, replacementSets ...[]AdapterReplacement) (CapabilityReview, error) {
	collected, err := collectCapabilityPackages(packages, overlay, replacementSets...)
	if err != nil {
		return CapabilityReview{}, err
	}
	policy, err := loadCompatibilityPolicy()
	if err != nil {
		return CapabilityReview{}, err
	}
	return evaluateCollectedCapabilities(policy, collected, tags)
}

// reviewRecordedClosure evaluates a closure an earlier review recorded. Its
// identity is checked before the compatibility policy is loaded.
func reviewRecordedClosure(closure CapabilityClosure, tags []string) (CapabilityReview, error) {
	if err := validateCapabilityClosureIdentity(closure); err != nil {
		return CapabilityReview{}, err
	}
	policy, err := loadCompatibilityPolicy()
	if err != nil {
		return CapabilityReview{}, err
	}
	return evaluateCapabilityClosure(policy, closure, tags)
}

func unsupportedFinding(finding CapabilityFinding) error {
	description := "requires an unsupported capability"
	switch finding.Kind {
	case FindingForbiddenImport:
		description = "imports " + strings.TrimPrefix(finding.Capability, "import:")
	case FindingForeignSource:
		description = "contains foreign or assembly source " + strings.TrimPrefix(finding.Capability, "foreign:")
	case FindingUnapprovedLinkname, FindingMalformedLinkname:
		description = "uses go:linkname in " + finding.SourceName
	case FindingNoReviewedGoSource:
		description = "has no reviewed Go source"
	case FindingDeniedBoundary:
		description = "reaches denied deterministic boundary " + finding.Capability
	default:
	}
	return &UnsupportedCapabilityError{ImportPath: finding.Package.ImportPath, Capability: description, Finding: finding}
}

func recordCompatibility(identities []CompatibilityIdentity) []record.CompatibilityPack {
	result := make([]record.CompatibilityPack, len(identities))
	for index, identity := range identities {
		result[index] = record.CompatibilityPack{ID: identity.ID, SHA256: record.SHA256(identity.SHA256)}
	}
	return result
}

func VerifyCompatibility(packs []record.CompatibilityPack) error {
	if packs == nil {
		return errors.New("compatibility pack identity is missing")
	}
	identities := make([]compatibility.Identity, len(packs))
	for index, pack := range packs {
		identities[index] = compatibility.Identity{ID: pack.ID, SHA256: string(pack.SHA256)}
	}
	return compatibility.VerifyIdentities(identities)
}
