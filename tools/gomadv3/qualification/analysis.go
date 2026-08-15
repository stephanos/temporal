package qualification

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/target"
)

const (
	AnalysisSchema      = "gomadv3.capability-analysis/v3"
	PriorAnalysisSchema = "gomadv3.capability-analysis/v2"
)

const MaximumAnalysisReportBytes = 16 << 20

type AnalysisClassification string

const (
	ClassificationSupported   AnalysisClassification = "supported"
	ClassificationUnsupported AnalysisClassification = "unsupported"
)

type AnalysisInput struct {
	Spec      target.Spec
	Review    target.CapabilityReview
	Toolchain target.ToolchainIdentity
	IOProfile deterministicio.Spec
	Adapters  []deterministicio.Adapter
}

type AnalysisSpec struct {
	Target    target.Spec
	Toolchain target.ToolchainIdentity
	IOProfile deterministicio.Spec
	Adapters  []deterministicio.Adapter
}

func Analyze(ctx context.Context, spec AnalysisSpec) (AnalysisReport, error) {
	return analyzeWith(ctx, spec, target.ReviewCapabilities)
}

func analyzeWith(ctx context.Context, spec AnalysisSpec, reviewCapabilities func(context.Context, target.Spec) (target.CapabilityReview, error)) (AnalysisReport, error) {
	review, err := reviewCapabilities(ctx, spec.Target)
	if err != nil {
		return AnalysisReport{}, err
	}
	return BuildAnalysis(AnalysisInput{
		Spec: spec.Target, Review: review, Toolchain: spec.Toolchain, IOProfile: spec.IOProfile, Adapters: spec.Adapters,
	})
}

type AnalysisReport struct {
	Schema             string                             `json:"schema"`
	Classification     AnalysisClassification             `json:"classification"`
	Target             AnalysisTarget                     `json:"target"`
	Toolchain          AnalysisToolchain                  `json:"toolchain"`
	Closure            AnalysisClosure                    `json:"closure"`
	IOProfile          deterministicio.Contract           `json:"io_profile"`
	Packs              []target.CompatibilityPackEvidence `json:"packs"`
	Requirements       []deterministicio.Requirement      `json:"requirements"`
	Blockers           []AnalysisBlocker                  `json:"blockers"`
	EliminatedBlockers []AnalysisBlocker                  `json:"eliminated_blockers,omitempty"`
}

type AnalysisTarget struct {
	Kind               target.Kind                        `json:"kind"`
	Source             string                             `json:"source"`
	Arguments          []string                           `json:"arguments"`
	BuildTags          []string                           `json:"build_tags"`
	CapabilityMode     target.CapabilityMode              `json:"capability_mode,omitempty"`
	CapabilityManifest *evidence.TargetCapabilityManifest `json:"capability_manifest,omitempty"`
}

type AnalysisToolchain struct {
	GoVersion               string          `json:"go_version"`
	BuildKey                string          `json:"build_key"`
	TargetGOOS              string          `json:"target_goos"`
	TargetGOARCH            string          `json:"target_goarch"`
	BoundaryManifestVersion string          `json:"boundary_manifest_version"`
	BoundaryManifestSHA256  evidence.SHA256 `json:"boundary_manifest_sha256"`
}

type AnalysisClosure struct {
	SHA256       evidence.SHA256                     `json:"sha256"`
	PackageCount evidence.Uint64String               `json:"package_count"`
	Roots        []target.CapabilityPackageReference `json:"roots"`
}

type AnalysisBlocker struct {
	target.CapabilityFinding
	DependencyPath []target.CapabilityPackageReference `json:"dependency_path"`
}

func BuildAnalysis(input AnalysisInput) (AnalysisReport, error) {
	if input.Spec.Kind != target.KindGoRun && input.Spec.Kind != target.KindGoTest {
		return AnalysisReport{}, errors.New("capability analysis requires a go-run or go-test target")
	}
	if input.Review.Schema != target.CapabilityReviewSchema || input.Review.Closure.Schema != target.CapabilityClosureSchema || input.Review.Roots == nil || input.Review.Packs == nil || input.Review.Findings == nil || input.Review.EliminatedFindings == nil {
		return AnalysisReport{}, errors.New("capability analysis review evidence is incomplete")
	}
	if len(input.Review.Roots) == 0 || len(input.Review.Closure.Packages) == 0 {
		return AnalysisReport{}, errors.New("capability analysis closure has no roots or packages")
	}
	closureBytes, err := evidence.CanonicalJSON(input.Review.Closure)
	if err != nil {
		return AnalysisReport{}, fmt.Errorf("encode target capability closure: %w", err)
	}
	requirements, err := input.IOProfile.Requirements(input.Review.Closure, input.Adapters)
	if err != nil {
		return AnalysisReport{}, fmt.Errorf("project deterministic I/O requirements: %w", err)
	}
	paths, err := shortestPaths(input.Review.Closure.Packages, input.Review.Roots)
	if err != nil {
		return AnalysisReport{}, err
	}
	blockers, err := projectAnalysisBlockers(input.Review.Findings, paths)
	if err != nil {
		return AnalysisReport{}, err
	}
	eliminated, err := projectAnalysisBlockers(input.Review.EliminatedFindings, paths)
	if err != nil {
		return AnalysisReport{}, err
	}
	mode := input.Review.CapabilityMode
	if mode == "" {
		mode = target.CapabilityModeClosure
	}
	specMode := input.Spec.CapabilityMode
	if specMode == "" {
		specMode = target.CapabilityModeClosure
	}
	if mode != specMode {
		return AnalysisReport{}, errors.New("capability analysis mode does not match the review")
	}
	capabilityTarget := evidence.Target{CapabilityMode: string(mode)}
	if input.Review.CapabilityManifest != nil {
		capabilityTarget.CapabilityManifest = input.Review.CapabilityManifest.Record()
	}
	if err := evidence.ValidateCurrentTargetCapability(capabilityTarget); err != nil {
		return AnalysisReport{}, fmt.Errorf("capability analysis target: %w", err)
	}
	classification := ClassificationSupported
	if len(blockers) != 0 {
		classification = ClassificationUnsupported
	}
	boundaryVersion, boundaryDigest := deterministicio.BoundaryManifestIdentity()
	return AnalysisReport{
		Schema: AnalysisSchema, Classification: classification,
		Target: AnalysisTarget{
			Kind: input.Spec.Kind, Source: safeSource(input.Spec.Source), Arguments: safeArguments(input.Spec.Args),
			BuildTags: append([]string{}, input.Review.BuildTags...), CapabilityMode: mode,
			CapabilityManifest: capabilityTarget.CapabilityManifest,
		},
		Toolchain: AnalysisToolchain{
			GoVersion: input.Toolchain.GoVersion, BuildKey: input.Toolchain.BuildKey,
			TargetGOOS: input.Toolchain.TargetGOOS, TargetGOARCH: input.Toolchain.TargetGOARCH,
			BoundaryManifestVersion: boundaryVersion, BoundaryManifestSHA256: evidence.SHA256(boundaryDigest),
		},
		Closure: AnalysisClosure{
			SHA256: evidence.HashBytes(closureBytes), PackageCount: evidence.Uint64String(len(input.Review.Closure.Packages)),
			Roots: append([]target.CapabilityPackageReference{}, input.Review.Roots...),
		},
		IOProfile: input.IOProfile.Identity(), Packs: append([]target.CompatibilityPackEvidence{}, input.Review.Packs...),
		Requirements: requirements, Blockers: blockers, EliminatedBlockers: eliminated,
	}, nil
}

func projectAnalysisBlockers(findings []target.CapabilityFinding, paths map[target.CapabilityPackageReference][]target.CapabilityPackageReference) ([]AnalysisBlocker, error) {
	blockers := make([]AnalysisBlocker, len(findings))
	for index, finding := range findings {
		path, found := paths[finding.Package]
		if !found {
			return nil, fmt.Errorf("capability finding package %s is not reachable from a target root", finding.Package.ImportPath)
		}
		blockers[index] = AnalysisBlocker{CapabilityFinding: copyFinding(finding), DependencyPath: append([]target.CapabilityPackageReference{}, path...)}
	}
	return blockers, nil
}

func DecodeAnalysisReport(data []byte) (AnalysisReport, error) {
	if len(data) == 0 || len(data) > MaximumAnalysisReportBytes {
		return AnalysisReport{}, fmt.Errorf("capability analysis report must be between 1 and %d bytes", MaximumAnalysisReportBytes)
	}
	var report AnalysisReport
	if err := evidence.DecodeCanonicalJSON(data, &report); err != nil {
		return AnalysisReport{}, fmt.Errorf("decode capability analysis report: %w", err)
	}
	if err := validateAnalysisReport(report); err != nil {
		return AnalysisReport{}, err
	}
	if report.Schema == PriorAnalysisSchema {
		report.Schema = AnalysisSchema
		report.Target.CapabilityMode = target.CapabilityModeClosure
	}
	if report.EliminatedBlockers == nil {
		report.EliminatedBlockers = []AnalysisBlocker{}
	}
	return report, nil
}

func validateAnalysisReport(report AnalysisReport) error {
	if report.Schema != AnalysisSchema && report.Schema != PriorAnalysisSchema || report.Target.Kind != target.KindGoRun && report.Target.Kind != target.KindGoTest || report.Target.Source == "" {
		return errors.New("capability analysis identity is invalid")
	}
	if report.Target.Arguments == nil || report.Target.BuildTags == nil || report.Closure.Roots == nil || len(report.Closure.Roots) == 0 || report.Packs == nil || report.Requirements == nil || report.Blockers == nil || uint64(report.Closure.PackageCount) == 0 {
		return errors.New("capability analysis evidence is incomplete")
	}
	if report.Schema == PriorAnalysisSchema {
		if report.Target.CapabilityMode != "" || report.Target.CapabilityManifest != nil || report.EliminatedBlockers != nil {
			return errors.New("historical capability analysis contains linked capability evidence")
		}
	} else {
		if err := evidence.ValidateCurrentTargetCapability(evidence.Target{CapabilityMode: string(report.Target.CapabilityMode), CapabilityManifest: report.Target.CapabilityManifest}); err != nil {
			return fmt.Errorf("capability analysis target: %w", err)
		}
	}
	for _, digest := range []evidence.SHA256{
		report.Closure.SHA256, report.Toolchain.BoundaryManifestSHA256,
		evidence.SHA256(report.IOProfile.ImplementationSHA256), evidence.SHA256(report.IOProfile.InventorySHA256),
	} {
		if _, err := digest.Bytes(); err != nil {
			return fmt.Errorf("capability analysis digest is invalid: %w", err)
		}
	}
	if report.Toolchain.GoVersion == "" || report.Toolchain.BuildKey == "" || report.Toolchain.TargetGOOS == "" || report.Toolchain.TargetGOARCH == "" || report.Toolchain.BoundaryManifestVersion == "" || report.IOProfile.Name == "" {
		return errors.New("capability analysis implementation identity is incomplete")
	}
	switch report.Classification {
	case ClassificationSupported:
		if len(report.Blockers) != 0 {
			return errors.New("supported capability analysis contains blockers")
		}
	case ClassificationUnsupported:
		if len(report.Blockers) == 0 {
			return errors.New("unsupported capability analysis has no blockers")
		}
	default:
		return fmt.Errorf("unknown capability analysis classification %q", report.Classification)
	}
	for index, blocker := range report.Blockers {
		if blocker.Kind == "" || blocker.Package.ImportPath == "" || len(blocker.DependencyPath) == 0 || blocker.DependencyPath[len(blocker.DependencyPath)-1] != blocker.Package {
			return fmt.Errorf("capability analysis blocker %d is invalid", index)
		}
	}
	for index, blocker := range report.EliminatedBlockers {
		if blocker.Kind == "" || blocker.Package.ImportPath == "" || len(blocker.DependencyPath) == 0 || blocker.DependencyPath[len(blocker.DependencyPath)-1] != blocker.Package {
			return fmt.Errorf("capability analysis eliminated blocker %d is invalid", index)
		}
	}
	return nil
}

func shortestPaths(packages []target.CapabilityPackage, roots []target.CapabilityPackageReference) (map[target.CapabilityPackageReference][]target.CapabilityPackageReference, error) {
	nodes := make(map[target.CapabilityPackageReference]target.CapabilityPackage, len(packages))
	byImportPath := make(map[string][]target.CapabilityPackageReference, len(packages))
	for _, pkg := range packages {
		reference := packageReference(pkg)
		if _, duplicate := nodes[reference]; duplicate {
			return nil, fmt.Errorf("capability closure package is duplicated: %s", reference.ImportPath)
		}
		nodes[reference] = pkg
		byImportPath[pkg.ImportPath] = append(byImportPath[pkg.ImportPath], reference)
	}
	for importPath := range byImportPath {
		sort.Slice(byImportPath[importPath], func(i, j int) bool { return lessReference(byImportPath[importPath][i], byImportPath[importPath][j]) })
	}
	orderedRoots := append([]target.CapabilityPackageReference{}, roots...)
	sort.Slice(orderedRoots, func(i, j int) bool { return lessReference(orderedRoots[i], orderedRoots[j]) })
	paths := make(map[target.CapabilityPackageReference][]target.CapabilityPackageReference, len(packages))
	queue := make([]target.CapabilityPackageReference, 0, len(packages))
	for _, root := range orderedRoots {
		if _, found := nodes[root]; !found {
			return nil, fmt.Errorf("capability closure root is missing: %s", root.ImportPath)
		}
		if _, duplicate := paths[root]; duplicate {
			continue
		}
		paths[root] = []target.CapabilityPackageReference{root}
		queue = append(queue, root)
	}
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		neighbors := []target.CapabilityPackageReference{}
		for _, imported := range nodes[current].Imports {
			neighbors = append(neighbors, byImportPath[imported]...)
		}
		sort.Slice(neighbors, func(i, j int) bool { return lessReference(neighbors[i], neighbors[j]) })
		for _, neighbor := range neighbors {
			if _, visited := paths[neighbor]; visited {
				continue
			}
			paths[neighbor] = append(append([]target.CapabilityPackageReference{}, paths[current]...), neighbor)
			queue = append(queue, neighbor)
		}
	}
	return paths, nil
}

func packageReference(pkg target.CapabilityPackage) target.CapabilityPackageReference {
	return target.CapabilityPackageReference{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: pkg.Name}
}

func lessReference(left, right target.CapabilityPackageReference) bool {
	if left.ImportPath != right.ImportPath {
		return left.ImportPath < right.ImportPath
	}
	if left.ForTest != right.ForTest {
		return left.ForTest < right.ForTest
	}
	return left.Name < right.Name
}

func safeSource(source string) string {
	if filepath.IsAbs(source) {
		return filepath.Base(filepath.Clean(source))
	}
	return filepath.ToSlash(filepath.Clean(source))
}

func safeArguments(arguments []string) []string {
	result := make([]string, len(arguments))
	for index, argument := range arguments {
		if strings.ContainsAny(argument, `/\`) {
			result[index] = string(evidence.HashBytes([]byte(argument)))
		} else {
			result[index] = argument
		}
	}
	return result
}

func copyFinding(finding target.CapabilityFinding) target.CapabilityFinding {
	result := finding
	result.Directives = append([]string{}, finding.Directives...)
	if finding.Module != nil {
		module := *finding.Module
		module.Replacement = nil
		if finding.Module.Replacement != nil {
			replacement := *finding.Module.Replacement
			module.Replacement = &replacement
		}
		result.Module = &module
	}
	return result
}

func FormatAnalysisText(report AnalysisReport) string {
	var output strings.Builder
	fmt.Fprintf(&output, "compatibility: %s\ncapability-mode: %s\npackages: %d\n", report.Classification, report.Target.CapabilityMode, uint64(report.Closure.PackageCount))
	groups := make(map[string][]AnalysisBlocker)
	keys := []string{}
	for _, blocker := range report.Blockers {
		key := blockerGroupKey(blocker)
		if _, found := groups[key]; !found {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], blocker)
	}
	sort.Strings(keys)
	for _, key := range keys {
		blockers := groups[key]
		sort.Slice(blockers, func(i, j int) bool { return lessPath(blockers[i].DependencyPath, blockers[j].DependencyPath) })
		first := blockers[0]
		fmt.Fprintf(&output, "- %s %s (%s)\n  path: ", first.Kind, first.Capability, first.Remediation)
		for index, pkg := range first.DependencyPath {
			if index != 0 {
				output.WriteString(" -> ")
			}
			output.WriteString(pkg.ImportPath)
		}
		output.WriteByte('\n')
		affected := make([]target.CapabilityPackageReference, len(blockers))
		for index, blocker := range blockers {
			affected[index] = blocker.Package
		}
		sort.Slice(affected, func(i, j int) bool { return lessReference(affected[i], affected[j]) })
		output.WriteString("  affected: ")
		for index, pkg := range affected {
			if index != 0 {
				output.WriteString(", ")
			}
			output.WriteString(pkg.ImportPath)
		}
		output.WriteByte('\n')
	}
	if len(report.EliminatedBlockers) != 0 {
		fmt.Fprintf(&output, "eliminated-blockers: %d\n", len(report.EliminatedBlockers))
	}
	return output.String()
}

func blockerGroupKey(blocker AnalysisBlocker) string {
	return strings.Join([]string{
		string(blocker.Kind), blocker.Capability, blocker.SourceName, blocker.SourceSHA256,
		strings.Join(blocker.Directives, "\x00"), string(blocker.PolicyDisposition), string(blocker.Remediation), blocker.PackID,
	}, "\x01")
}

func lessPath(left, right []target.CapabilityPackageReference) bool {
	if len(left) != len(right) {
		return len(left) < len(right)
	}
	for index := range left {
		if left[index] != right[index] {
			return lessReference(left[index], right[index])
		}
	}
	return false
}
