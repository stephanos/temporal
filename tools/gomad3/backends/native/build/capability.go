package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	targetbuild "go.temporal.io/server/tools/gomad3/backends/native/build/internal/build"
	"go.temporal.io/server/tools/gomad3/backends/native/build/internal/gocommand"
	compatibility "go.temporal.io/server/tools/gomad3/backends/native/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/backends/native/toolchain/installation"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
)

const maximumCapabilityReviewOutputBytes = 64 << 20

const maximumStandardPackagesBytes = 4 << 20

const maximumCapabilityReviewPackages = 100000

const maximumCapabilitySourceBytes = 16 << 20

func invalidCapabilityReview(err error) error {
	return &target.InvalidCapabilityReviewError{Err: err}
}

func ReviewCapabilityClosure(ctx context.Context, spec target.Spec) (target.CapabilityClosure, error) {
	review, err := ReviewCapabilities(ctx, spec)
	if err != nil {
		return target.CapabilityClosure{}, err
	}
	if len(review.Findings) != 0 {
		return target.CapabilityClosure{}, unsupportedFinding(review.Findings[0])
	}
	return review.Closure, nil
}

func ReviewCapabilities(ctx context.Context, spec target.Spec) (target.CapabilityReview, error) {
	return reviewCapabilitiesWith(ctx, spec, gocommand.Default())
}

func reviewCapabilitiesWith(ctx context.Context, spec target.Spec, runner gocommand.Runner) (target.CapabilityReview, error) {
	if spec.Kind != target.KindGoRun && spec.Kind != target.KindGoTest {
		return target.CapabilityReview{}, invalidCapabilityReview(errors.New("capability review requires a go-run or go-test target"))
	}
	tags, err := targetbuild.NormalizeTags(spec.BuildTags)
	if err != nil {
		return target.CapabilityReview{}, invalidCapabilityReview(err)
	}
	mode, err := normalizeCapabilityMode(spec.CapabilityMode)
	if err != nil {
		return target.CapabilityReview{}, invalidCapabilityReview(err)
	}
	spec.CapabilityMode = mode
	if spec.Source == "" || spec.WorkingDir == "" || spec.ToolchainRoot == "" {
		return target.CapabilityReview{}, invalidCapabilityReview(errors.New("capability review requires source, working directory, and toolchain root"))
	}
	if strings.HasPrefix(spec.Source, "-") || strings.Contains(spec.Source, "...") || strings.IndexFunc(spec.Source, unicode.IsSpace) >= 0 || strings.IndexByte(spec.Source, 0) >= 0 {
		return target.CapabilityReview{}, invalidCapabilityReview(fmt.Errorf("go target package argument %q must select exactly one package", spec.Source))
	}
	layout, err := installation.At(spec.ToolchainRoot)
	if err != nil {
		return target.CapabilityReview{}, fmt.Errorf("resolve pinned Go command: %w", err)
	}
	goCommand := layout.GoCommand()
	buildContext, err := targetbuild.Resolve(spec.WorkingDir, spec.Source, tags)
	if err != nil {
		return target.CapabilityReview{}, invalidCapabilityReview(err)
	}
	review, _, err := reviewGoCapabilityPackagesWith(ctx, goCommand, spec, buildContext.Tags, buildContext.Directory, buildContext.Package, runner)
	if err != nil {
		return target.CapabilityReview{}, err
	}
	if mode == target.CapabilityModeClosure {
		return review, nil
	}
	if spec.PreparationRoot == "" {
		return target.CapabilityReview{}, invalidCapabilityReview(errors.New("linked capability review requires a preparation root"))
	}
	identity, err := readPinnedToolchainWith(context.Background(), spec.ToolchainRoot, runner)
	if err != nil {
		return target.CapabilityReview{}, err
	}
	workspace, err := os.MkdirTemp(spec.PreparationRoot, ".linked-review-")
	if err != nil {
		return target.CapabilityReview{}, fmt.Errorf("create linked capability review workspace: %w", err)
	}
	prepared, buildErr := buildGoTargetWith(ctx, spec, buildContext.Tags, identity, filepath.Join(workspace, "target"), goCommand, buildContext.Directory, buildContext.Package, review, allowUnsupported, nil, runner)
	cleanupErr := os.RemoveAll(workspace)
	if buildErr != nil || cleanupErr != nil {
		return target.CapabilityReview{}, errors.Join(buildErr, cleanupErr)
	}
	return prepared.review, nil
}

func reviewGoCapabilityReview(ctx context.Context, goCommand string, spec target.Spec, tags []string, commandDirectory, packageArgument string) (target.CapabilityReview, error) {
	review, _, err := reviewGoCapabilityPackages(ctx, goCommand, spec, tags, commandDirectory, packageArgument)
	return review, err
}

// reviewGoCapabilityPackages also returns the listing the review projected,
// which preparation needs for the build inputs the closure does not review.
func reviewGoCapabilityPackages(ctx context.Context, goCommand string, spec target.Spec, tags []string, commandDirectory, packageArgument string) (target.CapabilityReview, []listedPackage, error) {
	return reviewGoCapabilityPackagesWith(ctx, goCommand, spec, tags, commandDirectory, packageArgument, gocommand.Default())
}

func reviewGoCapabilityPackagesWith(ctx context.Context, goCommand string, spec target.Spec, tags []string, commandDirectory, packageArgument string, runner gocommand.Runner) (target.CapabilityReview, []listedPackage, error) {
	packages, overlay, err := collectCapabilityListing(ctx, goCommand, spec, tags, commandDirectory, packageArgument, runner)
	if err != nil {
		return target.CapabilityReview{}, nil, err
	}
	review, err := projectCapabilityReview(packages, overlay, tags, spec.AdapterReplacements)
	if err != nil {
		return target.CapabilityReview{}, nil, err
	}
	return review, packages, nil
}

// projectCapabilityReview collects the capability evidence of a listed
// closure and the compatibility policy, then evaluates the evidence against it.
func projectCapabilityReview(packages []listedPackage, overlay map[string]string, tags []string, replacementSets ...[]target.AdapterReplacement) (target.CapabilityReview, error) {
	collected, err := collectCapabilityPackages(packages, overlay, replacementSets...)
	if err != nil {
		return target.CapabilityReview{}, err
	}
	policy, err := loadCompatibilityPolicy()
	if err != nil {
		return target.CapabilityReview{}, err
	}
	return evaluateCollectedCapabilities(policy, collected, tags)
}

// reviewRecordedClosure evaluates a closure an earlier review recorded. Its
// identity is checked before the compatibility policy is loaded.
func reviewRecordedClosure(closure target.CapabilityClosure, tags []string) (target.CapabilityReview, error) {
	if err := validateCapabilityClosureIdentity(closure); err != nil {
		return target.CapabilityReview{}, err
	}
	policy, err := loadCompatibilityPolicy()
	if err != nil {
		return target.CapabilityReview{}, err
	}
	return evaluateCapabilityClosure(policy, closure, tags)
}

func unsupportedFinding(finding target.CapabilityFinding) error {
	description := "requires an unsupported capability"
	switch finding.Kind {
	case target.FindingForbiddenImport:
		description = "imports " + strings.TrimPrefix(finding.Capability, "import:")
	case target.FindingForeignSource:
		description = "contains foreign or assembly source " + strings.TrimPrefix(finding.Capability, "foreign:")
	case target.FindingUnapprovedLinkname, target.FindingMalformedLinkname:
		description = "uses go:linkname in " + finding.SourceName
	case target.FindingNoReviewedGoSource:
		description = "has no reviewed Go source"
	case target.FindingDeniedBoundary:
		description = "reaches denied deterministic boundary " + finding.Capability
	default:
	}
	return &target.UnsupportedCapabilityError{ImportPath: finding.Package.ImportPath, Capability: description, Finding: finding}
}

func recordCompatibility(identities []target.CompatibilityIdentity) []record.CompatibilityPack {
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
