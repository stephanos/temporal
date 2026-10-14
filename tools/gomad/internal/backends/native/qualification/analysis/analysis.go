package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	nativebuild "go.temporal.io/server/tools/gomad/internal/backends/native/build"
	"go.temporal.io/server/tools/gomad/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
)

const (
	AnalysisSchema = "gomad3.capability-analysis/v1"
)

const MaximumAnalysisReportBytes = 16 << 20

type Classification string

const (
	ClassificationSupported   Classification = "supported"
	ClassificationUnsupported Classification = "unsupported"
)

type Input struct {
	Spec      target.Spec
	Review    target.CapabilityReview
	Toolchain target.ToolchainIdentity
	IOProfile transcript.Spec
	Adapters  []transcript.Adapter
}

type Spec struct {
	Target    target.Spec
	Toolchain target.ToolchainIdentity
	IOProfile transcript.Spec
	Adapters  []transcript.Adapter
}

func Analyze(ctx context.Context, spec Spec) (Report, error) {
	return analyzeWith(ctx, spec, nativebuild.ReviewCapabilities)
}

func analyzeWith(ctx context.Context, spec Spec, reviewCapabilities func(context.Context, target.Spec) (target.CapabilityReview, error)) (Report, error) {
	review, err := reviewCapabilities(ctx, spec.Target)
	if err != nil {
		return Report{}, err
	}
	return Build(Input{
		Spec: spec.Target, Review: review, Toolchain: spec.Toolchain, IOProfile: spec.IOProfile, Adapters: spec.Adapters,
	})
}

type Report struct {
	Schema             string                             `json:"schema"`
	Classification     Classification                     `json:"classification"`
	Target             Target                             `json:"target"`
	Toolchain          Toolchain                          `json:"toolchain"`
	Closure            Closure                            `json:"closure"`
	IOProfile          record.IOContract                  `json:"io_profile"`
	Packs              []target.CompatibilityPackEvidence `json:"packs"`
	Requirements       []transcript.Requirement           `json:"requirements"`
	Blockers           []Blocker                          `json:"blockers"`
	GuardedBlockers    []Blocker                          `json:"guarded_blockers"`
	EliminatedBlockers []Blocker                          `json:"eliminated_blockers"`
}

type Target struct {
	Kind               target.Kind                      `json:"kind"`
	Source             string                           `json:"source"`
	Arguments          []string                         `json:"arguments"`
	BuildTags          []string                         `json:"build_tags"`
	CapabilityMode     target.CapabilityMode            `json:"capability_mode,omitempty"`
	CapabilityManifest *record.TargetCapabilityManifest `json:"capability_manifest,omitempty"`
}

type Toolchain struct {
	GoVersion               string        `json:"go_version"`
	BuildKey                string        `json:"build_key"`
	TargetGOOS              string        `json:"target_goos"`
	TargetGOARCH            string        `json:"target_goarch"`
	BoundaryManifestVersion string        `json:"boundary_manifest_version"`
	BoundaryManifestSHA256  record.SHA256 `json:"boundary_manifest_sha256"`
}

type Closure struct {
	SHA256       record.SHA256                       `json:"sha256"`
	PackageCount record.Uint64String                 `json:"package_count"`
	Roots        []target.CapabilityPackageReference `json:"roots"`
}

type Blocker struct {
	target.CapabilityFinding
	DependencyPath []target.CapabilityPackageReference `json:"dependency_path"`
}

func Build(input Input) (Report, error) {
	if err := validateInputStrings(input); err != nil {
		return Report{}, err
	}
	if input.Spec.Kind != target.KindGoRun && input.Spec.Kind != target.KindGoTest {
		return Report{}, errors.New("capability analysis requires a go-run or go-test target")
	}
	if input.Review.Schema != target.CapabilityReviewSchema || input.Review.Closure.Schema != target.CapabilityClosureSchema || input.Review.Roots == nil || input.Review.Packs == nil || input.Review.Findings == nil || input.Review.GuardedFindings == nil || input.Review.EliminatedFindings == nil {
		return Report{}, errors.New("capability analysis review evidence is incomplete")
	}
	if len(input.Review.Roots) == 0 || len(input.Review.Closure.Packages) == 0 {
		return Report{}, errors.New("capability analysis closure has no roots or packages")
	}
	closureBytes, err := json.Marshal(input.Review.Closure)
	if err != nil {
		return Report{}, fmt.Errorf("encode target capability closure: %w", err)
	}
	requirements, err := input.IOProfile.Requirements(input.Review.Closure, input.Adapters)
	if err != nil {
		return Report{}, fmt.Errorf("project deterministic I/O requirements: %w", err)
	}
	paths, err := shortestPaths(input.Review.Closure.Packages, input.Review.Roots)
	if err != nil {
		return Report{}, err
	}
	blockers, err := projectAnalysisBlockers(input.Review.Findings, paths)
	if err != nil {
		return Report{}, err
	}
	guarded, err := projectAnalysisBlockers(input.Review.GuardedFindings, paths)
	if err != nil {
		return Report{}, err
	}
	eliminated, err := projectAnalysisBlockers(input.Review.EliminatedFindings, paths)
	if err != nil {
		return Report{}, err
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
		return Report{}, errors.New("capability analysis mode does not match the review")
	}
	capabilityTarget := record.Target{CapabilityMode: string(mode)}
	if input.Review.CapabilityManifest != nil {
		capabilityTarget.CapabilityManifest = input.Review.CapabilityManifest.Record()
	}
	if err := record.ValidateCurrentTargetCapability(capabilityTarget); err != nil {
		return Report{}, fmt.Errorf("capability analysis target: %w", err)
	}
	classification := ClassificationSupported
	if len(blockers) != 0 {
		classification = ClassificationUnsupported
	}
	boundaryVersion, boundaryDigest := transcript.BoundaryManifestIdentity()
	report := Report{
		Schema: AnalysisSchema, Classification: classification,
		Target: Target{
			Kind: input.Spec.Kind, Source: safeSource(input.Spec.Source), Arguments: safeArguments(input.Spec.Args),
			BuildTags: append([]string{}, input.Review.BuildTags...), CapabilityMode: mode,
			CapabilityManifest: capabilityTarget.CapabilityManifest,
		},
		Toolchain: Toolchain{
			GoVersion: input.Toolchain.GoVersion, BuildKey: input.Toolchain.BuildKey,
			TargetGOOS: input.Toolchain.TargetGOOS, TargetGOARCH: input.Toolchain.TargetGOARCH,
			BoundaryManifestVersion: boundaryVersion, BoundaryManifestSHA256: record.SHA256(boundaryDigest),
		},
		Closure: Closure{
			SHA256: record.HashBytes(closureBytes), PackageCount: record.Uint64String(len(input.Review.Closure.Packages)),
			Roots: append([]target.CapabilityPackageReference{}, input.Review.Roots...),
		},
		IOProfile: input.IOProfile.Identity(), Packs: append([]target.CompatibilityPackEvidence{}, input.Review.Packs...),
		Requirements: requirements, Blockers: blockers, GuardedBlockers: guarded, EliminatedBlockers: eliminated,
	}
	return cloneReport(report)
}

func validateInputStrings(input Input) error {
	values := []string{
		string(input.Spec.Kind), input.Spec.Source, string(input.Spec.CapabilityMode), input.Review.Schema, string(input.Review.CapabilityMode),
		input.Toolchain.GoVersion, input.Toolchain.BuildKey, input.Toolchain.TargetGOOS, input.Toolchain.TargetGOARCH,
	}
	values = append(values, input.Spec.Args...)
	values = append(values, input.Review.BuildTags...)
	identity := input.IOProfile.Identity()
	values = append(values, identity.Name, string(identity.ImplementationSHA256), string(identity.InventorySHA256))
	for _, adapter := range input.Adapters {
		values = append(values, adapter.Module, adapter.Sum, adapter.Version)
	}
	appendClosureStrings(&values, input.Review.Closure)
	for _, root := range input.Review.Roots {
		appendReferenceStrings(&values, root)
	}
	for _, pack := range input.Review.Packs {
		appendPackStrings(&values, pack)
	}
	for _, findings := range [][]target.CapabilityFinding{input.Review.Findings, input.Review.GuardedFindings, input.Review.EliminatedFindings} {
		for _, finding := range findings {
			appendFindingStrings(&values, finding)
		}
	}
	if manifest := input.Review.CapabilityManifest; manifest != nil {
		values = append(values, manifest.Schema, string(manifest.SHA256), manifest.ProducerImplementationSHA256, manifest.GuardImplementationSHA256, manifest.CapabilityUniverseSHA256)
	}
	for _, value := range values {
		if !utf8.ValidString(value) {
			return errors.New("capability analysis string is not valid UTF-8")
		}
	}
	return nil
}

func appendClosureStrings(values *[]string, closure target.CapabilityClosure) {
	*values = append(*values, closure.Schema)
	for _, compatibility := range closure.Compatibility {
		*values = append(*values, compatibility.ID, compatibility.SHA256)
	}
	for _, pkg := range closure.Packages {
		*values = append(*values, pkg.ImportPath, pkg.ForTest, pkg.Name)
		*values = append(*values, pkg.Imports...)
		appendModuleStrings(values, pkg.Module)
		for _, source := range pkg.Sources {
			*values = append(*values, source.Name, source.SHA256)
			*values = append(*values, source.LinknameDirectives...)
		}
		for _, source := range pkg.ForeignSources {
			*values = append(*values, source.Kind, source.Name, source.SHA256)
		}
	}
}

func appendModuleStrings(values *[]string, module *target.CapabilityModule) {
	if module == nil {
		return
	}
	*values = append(*values, module.Path, module.Version, module.Sum)
	appendModuleStrings(values, module.Replacement)
	if adapter := module.Adapter; adapter != nil {
		*values = append(*values, adapter.ProfileName, adapter.ProfileImplementationSHA256,
			adapter.Adapter.Path, adapter.Adapter.Version, adapter.Adapter.Sum,
			adapter.OriginalSourceInventorySHA256, adapter.ReplacementSourceInventorySHA256, adapter.PreparedSourceSetSHA256)
	}
}

func appendReferenceStrings(values *[]string, reference target.CapabilityPackageReference) {
	*values = append(*values, reference.ImportPath, reference.ForTest, reference.Name)
}

func appendFindingStrings(values *[]string, finding target.CapabilityFinding) {
	*values = append(*values, string(finding.Kind), finding.SourceSetSHA256, finding.SourceName, finding.SourceSHA256,
		finding.Capability, string(finding.PolicyDisposition), string(finding.Remediation), finding.PackID)
	appendReferenceStrings(values, finding.Package)
	appendModuleStrings(values, finding.Module)
	*values = append(*values, finding.Directives...)
}

func appendPackStrings(values *[]string, pack target.CompatibilityPackEvidence) {
	*values = append(*values, pack.ID, pack.SHA256, pack.RequestSHA256)
	if governance := pack.Governance; governance != nil {
		*values = append(*values, governance.Owner, governance.ReviewedAt, governance.Justification, governance.ApprovalSHA256)
		*values = append(*values, governance.Workloads...)
		*values = append(*values, governance.Platforms...)
	}
	for _, activation := range pack.Activation {
		appendCompatibilityModuleStrings(values, activation)
	}
	for _, rule := range pack.Rules {
		*values = append(*values, rule.ImportPath, rule.SourceSetSHA256)
		appendCompatibilityModuleStrings(values, rule.Module)
		*values = append(*values, rule.Capabilities...)
		for _, source := range rule.GoSources {
			*values = append(*values, source.Name, source.SHA256)
		}
		for _, source := range rule.ForeignSources {
			*values = append(*values, source.Kind, source.Name, source.SHA256)
		}
		for _, linkname := range rule.Linknames {
			*values = append(*values, linkname.Source, linkname.SHA256)
			*values = append(*values, linkname.Directives...)
		}
	}
}

func appendCompatibilityModuleStrings(values *[]string, module target.CompatibilityModuleEvidence) {
	*values = append(*values, module.Path, module.Version, module.Sum, module.Replacement)
	if adapter := module.Adapter; adapter != nil {
		*values = append(*values, adapter.ProfileName, adapter.ProfileImplementationSHA256, adapter.Module, adapter.Version, adapter.Sum,
			adapter.OriginalSourceInventorySHA256, adapter.ReplacementSourceInventorySHA256, adapter.PreparedSourceSetSHA256)
	}
}

func cloneReport(report Report) (Report, error) {
	encoded, err := json.Marshal(report)
	if err != nil {
		return Report{}, fmt.Errorf("encode capability analysis report: %w", err)
	}
	var cloned Report
	if err := strictjson.Decode(encoded, &cloned); err != nil {
		return Report{}, fmt.Errorf("clone capability analysis report: %w", err)
	}
	return cloned, nil
}

// ForArguments rebinds the report to another argument list for the same
// target: a go target's arguments reach its process, never its build, so the
// capability evidence does not depend on them.
func (report Report) ForArguments(arguments []string) Report {
	report.Target.Arguments = safeArguments(arguments)
	return report
}

func projectAnalysisBlockers(findings []target.CapabilityFinding, paths map[target.CapabilityPackageReference][]target.CapabilityPackageReference) ([]Blocker, error) {
	blockers := make([]Blocker, len(findings))
	for index, finding := range findings {
		path, found := paths[finding.Package]
		if !found {
			return nil, fmt.Errorf("capability finding package %s is not reachable from a target root", finding.Package.ImportPath)
		}
		blockers[index] = Blocker{CapabilityFinding: copyFinding(finding), DependencyPath: append([]target.CapabilityPackageReference{}, path...)}
	}
	return blockers, nil
}

func Decode(data []byte) (Report, error) {
	if len(data) == 0 || len(data) > MaximumAnalysisReportBytes {
		return Report{}, fmt.Errorf("capability analysis report must be between 1 and %d bytes", MaximumAnalysisReportBytes)
	}
	var report Report
	if err := strictjson.Decode(data, &report); err != nil {
		return Report{}, fmt.Errorf("decode capability analysis report: %w", err)
	}
	if err := validateAnalysisReport(report); err != nil {
		return Report{}, err
	}
	return report, nil
}

func validateAnalysisReport(report Report) error {
	if report.Schema != AnalysisSchema || report.Target.Kind != target.KindGoRun && report.Target.Kind != target.KindGoTest || report.Target.Source == "" {
		return errors.New("capability analysis identity is invalid")
	}
	if report.Target.Arguments == nil || report.Target.BuildTags == nil || report.Closure.Roots == nil || len(report.Closure.Roots) == 0 || report.Packs == nil || report.Requirements == nil || report.Blockers == nil || report.GuardedBlockers == nil || report.EliminatedBlockers == nil || uint64(report.Closure.PackageCount) == 0 {
		return errors.New("capability analysis evidence is incomplete")
	}
	if err := record.ValidateCurrentTargetCapability(record.Target{CapabilityMode: string(report.Target.CapabilityMode), CapabilityManifest: report.Target.CapabilityManifest}); err != nil {
		return fmt.Errorf("capability analysis target: %w", err)
	}
	for _, digest := range []record.SHA256{
		report.Closure.SHA256, report.Toolchain.BoundaryManifestSHA256,
		record.SHA256(report.IOProfile.ImplementationSHA256), record.SHA256(report.IOProfile.InventorySHA256),
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
	for index, blocker := range report.GuardedBlockers {
		if blocker.Kind == "" || blocker.Package.ImportPath == "" || len(blocker.DependencyPath) == 0 || blocker.DependencyPath[len(blocker.DependencyPath)-1] != blocker.Package {
			return fmt.Errorf("capability analysis guarded blocker %d is invalid", index)
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
			result[index] = string(record.HashBytes([]byte(argument)))
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

func FormatText(report Report) string {
	var output strings.Builder
	fmt.Fprintf(&output, "compatibility: %s\ncapability-mode: %s\npackages: %d\n", report.Classification, report.Target.CapabilityMode, uint64(report.Closure.PackageCount))
	groups := make(map[string][]Blocker)
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
	if len(report.GuardedBlockers) != 0 {
		fmt.Fprintf(&output, "guarded-blockers: %d\n", len(report.GuardedBlockers))
	}
	return output.String()
}

func blockerGroupKey(blocker Blocker) string {
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
