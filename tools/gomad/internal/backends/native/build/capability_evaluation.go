package build

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends/native/build/internal/capabilitypolicy"
	compatibility "go.temporal.io/server/tools/gomad/internal/backends/native/compatibilitypack"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

// Capability evaluation is pure: it judges collected or recorded evidence
// against a loaded compatibility policy and reads no file, environment,
// process or clock. capabilitypolicy owns the policy decisions; this file
// validates the evidence shape and projects the decisions into the review.

// evaluateCollectedCapabilities records the packs policy selects for freshly
// collected packages and reviews the resulting closure.
func evaluateCollectedCapabilities(policy capabilitypolicy.Policy, packages []target.CapabilityPackage, tags []string) (target.CapabilityReview, error) {
	evaluation, err := evaluateCapabilityPolicy(policy, packages)
	if err != nil {
		return target.CapabilityReview{}, err
	}
	closure := target.CapabilityClosure{
		Schema:        target.CapabilityClosureSchema,
		Compatibility: projectCompatibilityIdentities(evaluation.Selection.Identities()),
		Packages:      packages,
	}
	if err := validateCapabilityClosureIdentity(closure); err != nil {
		return target.CapabilityReview{}, err
	}
	return reviewEvaluatedClosure(closure, tags, evaluation)
}

// evaluateCapabilityClosure validates a recorded closure against policy and
// reviews it.
func evaluateCapabilityClosure(policy capabilitypolicy.Policy, closure target.CapabilityClosure, tags []string) (target.CapabilityReview, error) {
	if err := validateCapabilityClosureIdentity(closure); err != nil {
		return target.CapabilityReview{}, err
	}
	evaluation, err := evaluateCapabilityPolicy(policy, closure.Packages)
	if err != nil {
		return target.CapabilityReview{}, err
	}
	return reviewEvaluatedClosure(closure, tags, evaluation)
}

// reviewEvaluatedClosure checks that the closure records the packs its
// evaluation selected and that its packages are canonical, then projects the
// evaluation into the review.
func reviewEvaluatedClosure(closure target.CapabilityClosure, tags []string, evaluation capabilitypolicy.Evaluation) (target.CapabilityReview, error) {
	if !slices.Equal(evaluation.Selection.Identities(), internalCompatibilityIdentities(closure.Compatibility)) {
		return target.CapabilityReview{}, errors.New("target capability closure compatibility pack identity does not match its package closure")
	}
	mainPackage := false
	for index, pkg := range closure.Packages {
		if err := validateCapabilityPackageStructure(closure.Packages, index); err != nil {
			return target.CapabilityReview{}, err
		}
		if pkg.Name == "main" {
			mainPackage = true
		}
	}
	if !mainPackage {
		return target.CapabilityReview{}, errors.New("target capability closure has no main package")
	}
	return capabilityReviewFromEvaluation(closure, tags, evaluation), nil
}

func validateCapabilityClosureIdentity(closure target.CapabilityClosure) error {
	if closure.Schema != target.CapabilityClosureSchema || closure.Compatibility == nil || len(closure.Packages) == 0 {
		return errors.New("unsupported or empty target capability closure")
	}
	if !sortedUniqueCompatibility(closure.Compatibility) {
		return errors.New("target capability closure compatibility packs are not canonical")
	}
	return nil
}

func evaluateCapabilityPolicy(policy capabilitypolicy.Policy, packages []target.CapabilityPackage) (capabilitypolicy.Evaluation, error) {
	policyPackages := make([]capabilitypolicy.Package, len(packages))
	for index, pkg := range packages {
		policyPackages[index] = capabilityPolicyPackage(pkg)
	}
	evaluation, err := capabilitypolicy.Evaluate(policy, policyPackages)
	if err != nil {
		//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
		return capabilitypolicy.Evaluation{}, fmt.Errorf("select target compatibility packs: %w", err)
	}
	return evaluation, nil
}

func capabilityPolicyPackage(pkg target.CapabilityPackage) capabilitypolicy.Package {
	sources := make([]capabilitypolicy.Source, len(pkg.Sources))
	for index, source := range pkg.Sources {
		sources[index] = capabilitypolicy.Source{
			Name: source.Name, SHA256: source.SHA256, LinknameDirectives: source.LinknameDirectives, MalformedLinkname: source.MalformedLinkname,
		}
	}
	return capabilitypolicy.Package{
		ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Standard: pkg.Standard, GeneratedTestMain: pkg.GeneratedTestMain,
		MainModule: pkg.Module != nil && pkg.Module.Main, Imports: pkg.Imports, Sources: sources, Policy: capabilityCompatibilityPackage(pkg),
	}
}

func capabilityReviewFromEvaluation(closure target.CapabilityClosure, tags []string, evaluation capabilitypolicy.Evaluation) target.CapabilityReview {
	roots := []target.CapabilityPackageReference{}
	for _, pkg := range closure.Packages {
		if pkg.Root {
			roots = append(roots, capabilityPackageReference(pkg))
		}
	}
	findings := make([]target.CapabilityFinding, 0, len(evaluation.Findings))
	for _, finding := range evaluation.Findings {
		findings = append(findings, capabilityFinding(closure.Packages[finding.Package], finding))
	}
	sort.Slice(findings, func(i, j int) bool { return compareCapabilityFinding(findings[i], findings[j]) < 0 })
	return target.CapabilityReview{
		Schema: target.CapabilityReviewSchema, BuildTags: append([]string{}, tags...), Roots: roots, Closure: closure,
		Packs: projectCompatibilityPackEvidence(evaluation.Selection.Evidence()), CapabilityMode: target.CapabilityModeClosure,
		Findings: findings, GuardedFindings: []target.CapabilityFinding{}, EliminatedFindings: []target.CapabilityFinding{},
	}
}

func capabilityFinding(pkg target.CapabilityPackage, finding capabilitypolicy.Finding) target.CapabilityFinding {
	return target.CapabilityFinding{
		Kind: target.CapabilityFindingKind(finding.Kind), Package: capabilityPackageReference(pkg), Module: copyCapabilityModule(pkg.Module),
		SourceSetSHA256: capabilityCompatibilityPackage(pkg).SourceSetSHA256,
		SourceName:      finding.SourceName, SourceSHA256: finding.SourceSHA256, Directives: append([]string{}, finding.Fact.Directives...),
		Capability: finding.Fact.Capability, PolicyDisposition: target.CompatibilityDisposition(finding.Decision.Disposition), Remediation: target.CompatibilityRemediation(finding.Decision.Remediation), PackID: finding.Decision.PackID,
	}
}

func validateCapabilityPackageStructure(packages []target.CapabilityPackage, index int) error {
	pkg := packages[index]
	if pkg.ImportPath == "" || pkg.Name == "" {
		return errors.New("target capability closure has an empty package identity")
	}
	if pkg.Imports == nil || pkg.Sources == nil || pkg.ForeignSources == nil {
		return fmt.Errorf("target capability closure package %s has non-canonical null fields", pkg.ImportPath)
	}
	if index > 0 && compareCapabilityPackage(packages[index-1], pkg) >= 0 {
		return errors.New("target capability closure packages are not sorted and unique")
	}
	if !sortedUnique(pkg.Imports) || !sortedUniqueForeignSources(pkg.ForeignSources) || !sortedUniqueSources(pkg.Sources) {
		return fmt.Errorf("target capability closure package %s is not canonical", pkg.ImportPath)
	}
	if err := validateCapabilityModule(pkg.Module); err != nil {
		//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
		return fmt.Errorf("target capability closure package %s: %w", pkg.ImportPath, err)
	}
	for _, source := range pkg.Sources {
		if err := validateCapabilitySource(source); err != nil {
			//gomad:dynamic reviewed 2026-10-12 reviewed codec and validator errors have memory-only methods
			return fmt.Errorf("target capability closure package %s: %w", pkg.ImportPath, err)
		}
	}
	for _, source := range pkg.ForeignSources {
		if source.Kind == "" || filepath.Base(source.Name) != source.Name || source.Name == "" {
			return fmt.Errorf("target capability closure package %s has invalid foreign source evidence", pkg.ImportPath)
		}
		if _, err := record.ParseSHA256(source.SHA256); err != nil {
			return fmt.Errorf("target capability closure package %s has invalid foreign source evidence", pkg.ImportPath)
		}
	}
	if pkg.GeneratedTestMain && (pkg.Name != "main" || !strings.HasSuffix(pkg.ImportPath, ".test") || pkg.Standard || pkg.Module != nil && !pkg.Module.Main || len(pkg.Sources) != 0 || len(pkg.ForeignSources) != 0) {
		return fmt.Errorf("target capability closure package %s has invalid generated test-main evidence", pkg.ImportPath)
	}
	return nil
}

func validateCapabilitySource(source target.CapabilitySource) error {
	_, digestErr := record.ParseSHA256(source.SHA256)
	if filepath.Base(source.Name) != source.Name || source.Name == "" || digestErr != nil {
		return errors.New("has invalid source evidence")
	}
	if source.MalformedLinkname && len(source.LinknameDirectives) != 0 {
		return errors.New("has invalid linkname evidence")
	}
	return nil
}

func validateCapabilityModule(module *target.CapabilityModule) error {
	if module == nil {
		return nil
	}
	if module.Path == "" && !module.Local {
		return errors.New("module identity is empty")
	}
	if module.Main && module.Local {
		return errors.New("main module cannot be a local replacement")
	}
	if module.Replacement != nil {
		if module.Replacement.Main || module.Replacement.Replacement != nil {
			return errors.New("module replacement is malformed")
		}
		if err := validateCapabilityModule(module.Replacement); err != nil {
			return err
		}
	}
	if module.Adapter != nil {
		if module.Replacement == nil || !module.Replacement.Local {
			return errors.New("adapter evidence requires a local replacement")
		}
		if err := validateCapabilityAdapter(*module.Adapter); err != nil {
			return err
		}
	}
	return nil
}

func validateCapabilityAdapter(adapter target.CapabilityAdapterReplacement) error {
	if adapter.ProfileName == "" || adapter.Adapter.Path == "" || adapter.Adapter.Version == "" || adapter.Adapter.Sum == "" {
		return errors.New("adapter replacement identity is incomplete")
	}
	for _, digest := range []string{
		adapter.ProfileImplementationSHA256, adapter.OriginalSourceInventorySHA256,
		adapter.ReplacementSourceInventorySHA256, adapter.PreparedSourceSetSHA256,
	} {
		if _, err := record.ParseSHA256(digest); err != nil {
			return errors.New("adapter replacement digest is invalid")
		}
	}
	return nil
}

func capabilityCompatibilityPackage(pkg target.CapabilityPackage) compatibility.Package {
	goSources := make([]compatibility.Source, len(pkg.Sources))
	sources := make([]compatibility.Source, 0, len(pkg.Sources)+len(pkg.ForeignSources))
	for index, source := range pkg.Sources {
		goSources[index] = compatibility.Source{Name: source.Name, SHA256: source.SHA256}
		sources = append(sources, goSources[index])
	}
	foreignSources := make([]compatibility.ForeignSource, len(pkg.ForeignSources))
	for index, source := range pkg.ForeignSources {
		foreignSources[index] = compatibility.ForeignSource{Kind: source.Kind, Name: source.Name, SHA256: source.SHA256}
		sources = append(sources, compatibility.Source{Name: source.Kind + ":" + source.Name, SHA256: source.SHA256})
	}
	return compatibility.Package{
		ImportPath: pkg.ImportPath, Module: capabilityCompatibilityModule(pkg.Module), SourceSetSHA256: compatibility.DigestSources(sources),
		GoSources: goSources, ForeignSources: foreignSources,
	}
}

func capabilityCompatibilityModule(module *target.CapabilityModule) compatibility.Module {
	if module == nil {
		return compatibility.Module{}
	}
	projected := compatibility.Module{
		Path:             module.Path,
		Version:          module.Version,
		Sum:              module.Sum,
		Replaced:         module.Replacement != nil,
		LocalReplacement: module.Replacement != nil && module.Replacement.Local,
	}
	if module.Adapter != nil {
		projected.Adapter = &compatibility.AdapterEvidence{
			ProfileName: module.Adapter.ProfileName, ProfileImplementationSHA256: module.Adapter.ProfileImplementationSHA256,
			Module: module.Adapter.Adapter.Path, Version: module.Adapter.Adapter.Version, Sum: module.Adapter.Adapter.Sum,
			OriginalSourceInventorySHA256:    module.Adapter.OriginalSourceInventorySHA256,
			ReplacementSourceInventorySHA256: module.Adapter.ReplacementSourceInventorySHA256,
			PreparedSourceSetSHA256:          module.Adapter.PreparedSourceSetSHA256,
		}
	}
	return projected
}

func capabilityPackageReference(pkg target.CapabilityPackage) target.CapabilityPackageReference {
	return target.CapabilityPackageReference{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: pkg.Name}
}

func compareCapabilityPackage(left, right target.CapabilityPackage) int {
	return compareCapabilityPackageReference(capabilityPackageReference(left), capabilityPackageReference(right))
}

func compareCapabilityPackageReference(left, right target.CapabilityPackageReference) int {
	if comparison := strings.Compare(left.ImportPath, right.ImportPath); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(left.ForTest, right.ForTest); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.Name, right.Name)
}

func compareCapabilityFinding(left, right target.CapabilityFinding) int {
	if comparison := compareCapabilityPackageReference(left.Package, right.Package); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(string(left.Kind), string(right.Kind)); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(left.Capability, right.Capability); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.SourceName, right.SourceName)
}

func copyCapabilityModule(module *target.CapabilityModule) *target.CapabilityModule {
	if module == nil {
		return nil
	}
	result := *module
	result.Replacement = copyCapabilityModule(module.Replacement)
	if module.Adapter != nil {
		adapter := *module.Adapter
		result.Adapter = &adapter
	}
	return &result
}

func sortedUnique(values []string) bool {
	for index, value := range values {
		if value == "" || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func sortedUniqueSources(sources []target.CapabilitySource) bool {
	for index, source := range sources {
		if index > 0 && sources[index-1].Name >= source.Name {
			return false
		}
	}
	return true
}

func sortedUniqueForeignSources(sources []target.CapabilityForeignSource) bool {
	for index, source := range sources {
		if index == 0 {
			continue
		}
		previous := sources[index-1]
		if previous.Kind > source.Kind || previous.Kind == source.Kind && previous.Name >= source.Name {
			return false
		}
	}
	return true
}

func sortedUniqueCompatibility(identities []target.CompatibilityIdentity) bool {
	for index, identity := range identities {
		_, digestErr := record.ParseSHA256(identity.SHA256)
		if identity.ID == "" || digestErr != nil || index > 0 && identities[index-1].ID >= identity.ID {
			return false
		}
	}
	return true
}

func projectCompatibilityIdentities(values []compatibility.Identity) []target.CompatibilityIdentity {
	result := make([]target.CompatibilityIdentity, len(values))
	for index, value := range values {
		result[index] = target.CompatibilityIdentity(value)
	}
	return result
}

func internalCompatibilityIdentities(values []target.CompatibilityIdentity) []compatibility.Identity {
	result := make([]compatibility.Identity, len(values))
	for index, value := range values {
		result[index] = compatibility.Identity(value)
	}
	return result
}

func projectCompatibilityPackEvidence(values []compatibility.PackEvidence) []target.CompatibilityPackEvidence {
	result := make([]target.CompatibilityPackEvidence, len(values))
	for index, value := range values {
		projected := target.CompatibilityPackEvidence{
			ID:            value.ID,
			SHA256:        value.SHA256,
			RequestSHA256: value.RequestSHA256,
		}
		if value.Governance != nil {
			g := value.Governance
			projected.Governance = &target.CompatibilityPackGovernance{
				Owner:          g.Owner,
				ReviewedAt:     g.ReviewedAt,
				Justification:  g.Justification,
				Workloads:      slices.Clone(g.Workloads),
				Platforms:      slices.Clone(g.Platforms),
				ApprovalSHA256: g.ApprovalSHA256,
			}
		}
		if value.Activation != nil {
			projected.Activation = make([]target.CompatibilityModuleEvidence, len(value.Activation))
			for i, module := range value.Activation {
				projected.Activation[i] = projectCompatibilityModuleEvidence(module)
			}
		}
		if value.Rules != nil {
			projected.Rules = make([]target.CompatibilityPackageRuleEvidence, len(value.Rules))
			for i, rule := range value.Rules {
				r := target.CompatibilityPackageRuleEvidence{
					ImportPath:      rule.ImportPath,
					Module:          projectCompatibilityModuleEvidence(rule.Module),
					SourceSetSHA256: rule.SourceSetSHA256,
					Capabilities:    slices.Clone(rule.Capabilities),
				}
				if rule.GoSources != nil {
					r.GoSources = make([]target.CompatibilityPackSource, len(rule.GoSources))
					for j, s := range rule.GoSources {
						r.GoSources[j] = target.CompatibilityPackSource{Name: s.Name, SHA256: s.SHA256}
					}
				}
				if rule.ForeignSources != nil {
					r.ForeignSources = make([]target.CompatibilityPackForeignSource, len(rule.ForeignSources))
					for j, s := range rule.ForeignSources {
						r.ForeignSources[j] = target.CompatibilityPackForeignSource{Kind: s.Kind, Name: s.Name, SHA256: s.SHA256}
					}
				}
				if rule.Linknames != nil {
					r.Linknames = make([]target.CompatibilityLinknameEvidence, len(rule.Linknames))
					for j, l := range rule.Linknames {
						r.Linknames[j] = target.CompatibilityLinknameEvidence{Source: l.Source, SHA256: l.SHA256, Directives: slices.Clone(l.Directives)}
					}
				}
				projected.Rules[i] = r
			}
		}
		result[index] = projected
	}
	return result
}

func projectCompatibilityModuleEvidence(value compatibility.ModuleEvidence) target.CompatibilityModuleEvidence {
	result := target.CompatibilityModuleEvidence{
		Path:        value.Path,
		Version:     value.Version,
		Sum:         value.Sum,
		Replacement: value.Replacement,
	}
	if value.Adapter != nil {
		a := value.Adapter
		result.Adapter = &target.CompatibilityPackAdapter{
			ProfileName:                      a.ProfileName,
			ProfileImplementationSHA256:      a.ProfileImplementationSHA256,
			Module:                           a.Module,
			Version:                          a.Version,
			Sum:                              a.Sum,
			OriginalSourceInventorySHA256:    a.OriginalSourceInventorySHA256,
			ReplacementSourceInventorySHA256: a.ReplacementSourceInventorySHA256,
			PreparedSourceSetSHA256:          a.PreparedSourceSetSHA256,
		}
	}
	return result
}
