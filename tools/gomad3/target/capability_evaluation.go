package target

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target/internal/capabilitypolicy"
)

// Capability evaluation is pure: it judges collected or recorded evidence
// against a loaded compatibility policy and reads no file, environment,
// process or clock. capabilitypolicy owns the policy decisions; this file
// validates the evidence shape and projects the decisions into the review.

// evaluateCollectedCapabilities records the packs policy selects for freshly
// collected packages and reviews the resulting closure.
func evaluateCollectedCapabilities(policy capabilitypolicy.Policy, packages []CapabilityPackage, tags []string) (CapabilityReview, error) {
	evaluation, err := evaluateCapabilityPolicy(policy, packages)
	if err != nil {
		return CapabilityReview{}, err
	}
	closure := CapabilityClosure{
		Schema:        CapabilityClosureSchema,
		Compatibility: projectCompatibilityIdentities(evaluation.Selection.Identities()),
		Packages:      packages,
	}
	if err := validateCapabilityClosureIdentity(closure); err != nil {
		return CapabilityReview{}, err
	}
	return reviewEvaluatedClosure(closure, tags, evaluation)
}

// evaluateCapabilityClosure validates a recorded closure against policy and
// reviews it.
func evaluateCapabilityClosure(policy capabilitypolicy.Policy, closure CapabilityClosure, tags []string) (CapabilityReview, error) {
	if err := validateCapabilityClosureIdentity(closure); err != nil {
		return CapabilityReview{}, err
	}
	evaluation, err := evaluateCapabilityPolicy(policy, closure.Packages)
	if err != nil {
		return CapabilityReview{}, err
	}
	return reviewEvaluatedClosure(closure, tags, evaluation)
}

// reviewEvaluatedClosure checks that the closure records the packs its
// evaluation selected and that its packages are canonical, then projects the
// evaluation into the review.
func reviewEvaluatedClosure(closure CapabilityClosure, tags []string, evaluation capabilitypolicy.Evaluation) (CapabilityReview, error) {
	if !slices.Equal(evaluation.Selection.Identities(), internalCompatibilityIdentities(closure.Compatibility)) {
		return CapabilityReview{}, errors.New("target capability closure compatibility pack identity does not match its package closure")
	}
	mainPackage := false
	for index, pkg := range closure.Packages {
		if err := validateCapabilityPackageStructure(closure.Packages, index); err != nil {
			return CapabilityReview{}, err
		}
		if pkg.Name == "main" {
			mainPackage = true
		}
	}
	if !mainPackage {
		return CapabilityReview{}, errors.New("target capability closure has no main package")
	}
	return capabilityReviewFromEvaluation(closure, tags, evaluation), nil
}

func validateCapabilityClosureIdentity(closure CapabilityClosure) error {
	if closure.Schema != CapabilityClosureSchema || closure.Compatibility == nil || len(closure.Packages) == 0 {
		return errors.New("unsupported or empty target capability closure")
	}
	if !sortedUniqueCompatibility(closure.Compatibility) {
		return errors.New("target capability closure compatibility packs are not canonical")
	}
	return nil
}

func evaluateCapabilityPolicy(policy capabilitypolicy.Policy, packages []CapabilityPackage) (capabilitypolicy.Evaluation, error) {
	policyPackages := make([]capabilitypolicy.Package, len(packages))
	for index, pkg := range packages {
		policyPackages[index] = capabilityPolicyPackage(pkg)
	}
	evaluation, err := capabilitypolicy.Evaluate(policy, policyPackages)
	if err != nil {
		return capabilitypolicy.Evaluation{}, fmt.Errorf("select target compatibility packs: %w", err)
	}
	return evaluation, nil
}

func capabilityPolicyPackage(pkg CapabilityPackage) capabilitypolicy.Package {
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

func capabilityReviewFromEvaluation(closure CapabilityClosure, tags []string, evaluation capabilitypolicy.Evaluation) CapabilityReview {
	roots := []CapabilityPackageReference{}
	for _, pkg := range closure.Packages {
		if pkg.Root {
			roots = append(roots, capabilityPackageReference(pkg))
		}
	}
	findings := make([]CapabilityFinding, 0, len(evaluation.Findings))
	for _, finding := range evaluation.Findings {
		findings = append(findings, capabilityFinding(closure.Packages[finding.Package], finding))
	}
	sort.Slice(findings, func(i, j int) bool { return compareCapabilityFinding(findings[i], findings[j]) < 0 })
	return CapabilityReview{
		Schema: CapabilityReviewSchema, BuildTags: append([]string{}, tags...), Roots: roots, Closure: closure,
		Packs: projectCompatibilityPackEvidence(evaluation.Selection.Evidence()), CapabilityMode: CapabilityModeClosure,
		Findings: findings, GuardedFindings: []CapabilityFinding{}, EliminatedFindings: []CapabilityFinding{},
	}
}

func capabilityFinding(pkg CapabilityPackage, finding capabilitypolicy.Finding) CapabilityFinding {
	return CapabilityFinding{
		Kind: CapabilityFindingKind(finding.Kind), Package: capabilityPackageReference(pkg), Module: copyCapabilityModule(pkg.Module),
		SourceSetSHA256: capabilityCompatibilityPackage(pkg).SourceSetSHA256,
		SourceName:      finding.SourceName, SourceSHA256: finding.SourceSHA256, Directives: append([]string{}, finding.Fact.Directives...),
		Capability: finding.Fact.Capability, PolicyDisposition: CompatibilityDisposition(finding.Decision.Disposition), Remediation: CompatibilityRemediation(finding.Decision.Remediation), PackID: finding.Decision.PackID,
	}
}

func validateCapabilityPackageStructure(packages []CapabilityPackage, index int) error {
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
		return fmt.Errorf("target capability closure package %s: %w", pkg.ImportPath, err)
	}
	for _, source := range pkg.Sources {
		if err := validateCapabilitySource(source); err != nil {
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

func validateCapabilitySource(source CapabilitySource) error {
	_, digestErr := record.ParseSHA256(source.SHA256)
	if filepath.Base(source.Name) != source.Name || source.Name == "" || digestErr != nil {
		return errors.New("has invalid source evidence")
	}
	if source.MalformedLinkname && len(source.LinknameDirectives) != 0 {
		return errors.New("has invalid linkname evidence")
	}
	return nil
}

func validateCapabilityModule(module *CapabilityModule) error {
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

func validateCapabilityAdapter(adapter CapabilityAdapterReplacement) error {
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

func capabilityCompatibilityPackage(pkg CapabilityPackage) compatibility.Package {
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

func capabilityCompatibilityModule(module *CapabilityModule) compatibility.Module {
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

func capabilityPackageReference(pkg CapabilityPackage) CapabilityPackageReference {
	return CapabilityPackageReference{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: pkg.Name}
}

func compareCapabilityPackage(left, right CapabilityPackage) int {
	return compareCapabilityPackageReference(capabilityPackageReference(left), capabilityPackageReference(right))
}

func compareCapabilityPackageReference(left, right CapabilityPackageReference) int {
	if comparison := strings.Compare(left.ImportPath, right.ImportPath); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(left.ForTest, right.ForTest); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.Name, right.Name)
}

func compareCapabilityFinding(left, right CapabilityFinding) int {
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

func copyCapabilityModule(module *CapabilityModule) *CapabilityModule {
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

func sortedUniqueSources(sources []CapabilitySource) bool {
	for index, source := range sources {
		if index > 0 && sources[index-1].Name >= source.Name {
			return false
		}
	}
	return true
}

func sortedUniqueForeignSources(sources []CapabilityForeignSource) bool {
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

func sortedUniqueCompatibility(identities []CompatibilityIdentity) bool {
	for index, identity := range identities {
		_, digestErr := record.ParseSHA256(identity.SHA256)
		if identity.ID == "" || digestErr != nil || index > 0 && identities[index-1].ID >= identity.ID {
			return false
		}
	}
	return true
}

func projectCompatibilityIdentities(values []compatibility.Identity) []CompatibilityIdentity {
	result := make([]CompatibilityIdentity, len(values))
	for index, value := range values {
		result[index] = CompatibilityIdentity(value)
	}
	return result
}

func internalCompatibilityIdentities(values []CompatibilityIdentity) []compatibility.Identity {
	result := make([]compatibility.Identity, len(values))
	for index, value := range values {
		result[index] = compatibility.Identity(value)
	}
	return result
}

func projectCompatibilityPackEvidence(values []compatibility.PackEvidence) []CompatibilityPackEvidence {
	result := make([]CompatibilityPackEvidence, len(values))
	for index, value := range values {
		projected := CompatibilityPackEvidence{
			ID:            value.ID,
			SHA256:        value.SHA256,
			RequestSHA256: value.RequestSHA256,
		}
		if value.Governance != nil {
			g := value.Governance
			projected.Governance = &CompatibilityPackGovernance{
				Owner:          g.Owner,
				ReviewedAt:     g.ReviewedAt,
				Justification:  g.Justification,
				Workloads:      slices.Clone(g.Workloads),
				Platforms:      slices.Clone(g.Platforms),
				ApprovalSHA256: g.ApprovalSHA256,
			}
		}
		if value.Activation != nil {
			projected.Activation = make([]CompatibilityModuleEvidence, len(value.Activation))
			for i, module := range value.Activation {
				projected.Activation[i] = projectCompatibilityModuleEvidence(module)
			}
		}
		if value.Rules != nil {
			projected.Rules = make([]CompatibilityPackageRuleEvidence, len(value.Rules))
			for i, rule := range value.Rules {
				r := CompatibilityPackageRuleEvidence{
					ImportPath:      rule.ImportPath,
					Module:          projectCompatibilityModuleEvidence(rule.Module),
					SourceSetSHA256: rule.SourceSetSHA256,
					Capabilities:    slices.Clone(rule.Capabilities),
				}
				if rule.GoSources != nil {
					r.GoSources = make([]CompatibilityPackSource, len(rule.GoSources))
					for j, s := range rule.GoSources {
						r.GoSources[j] = CompatibilityPackSource{Name: s.Name, SHA256: s.SHA256}
					}
				}
				if rule.ForeignSources != nil {
					r.ForeignSources = make([]CompatibilityPackForeignSource, len(rule.ForeignSources))
					for j, s := range rule.ForeignSources {
						r.ForeignSources[j] = CompatibilityPackForeignSource{Kind: s.Kind, Name: s.Name, SHA256: s.SHA256}
					}
				}
				if rule.Linknames != nil {
					r.Linknames = make([]CompatibilityLinknameEvidence, len(rule.Linknames))
					for j, l := range rule.Linknames {
						r.Linknames[j] = CompatibilityLinknameEvidence{Source: l.Source, SHA256: l.SHA256, Directives: slices.Clone(l.Directives)}
					}
				}
				projected.Rules[i] = r
			}
		}
		result[index] = projected
	}
	return result
}

func projectCompatibilityModuleEvidence(value compatibility.ModuleEvidence) CompatibilityModuleEvidence {
	result := CompatibilityModuleEvidence{
		Path:        value.Path,
		Version:     value.Version,
		Sum:         value.Sum,
		Replacement: value.Replacement,
	}
	if value.Adapter != nil {
		a := value.Adapter
		result.Adapter = &CompatibilityPackAdapter{
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
