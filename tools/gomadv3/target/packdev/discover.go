package packdev

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomadv3/target"
	"go.temporal.io/server/tools/gomadv3/target/internal/compatibility"
	"golang.org/x/mod/module"
)

func Discover(draft Request, review target.CapabilityReview) (Request, string, error) {
	if err := validateDraft(draft); err != nil {
		return Request{}, "", err
	}
	if review.Schema != target.CapabilityReviewSchema || review.Closure.Schema != target.CapabilityClosureSchema || review.Closure.Packages == nil {
		return Request{}, "", errors.New("compatibility-pack discovery review evidence is incomplete")
	}
	if !slices.Equal(draft.Target.BuildTags, review.BuildTags) {
		return Request{}, "", errors.New("compatibility-pack discovery build tags do not match the target review")
	}
	if !reviewContainsMainModule(review, draft.Target.ExpectedModule) {
		return Request{}, "", errors.New("compatibility-pack discovery target module does not match the request")
	}

	result := draft
	result.ApprovalSHA256 = ""
	result.Activation = make([]Activation, len(draft.Activation))
	for index, selected := range draft.Activation {
		capabilityModule, err := findReviewedModule(review.Closure.Packages, selected.Path)
		if err != nil {
			return Request{}, "", err
		}
		packModule, err := projectReviewedModule(capabilityModule)
		if err != nil {
			return Request{}, "", err
		}
		result.Activation[index] = Activation{Path: selected.Path, Evidence: packModule}
	}
	result.Packages = make([]Package, len(draft.Packages))
	for index, selected := range draft.Packages {
		pkg, err := findReviewedPackage(review.Closure.Packages, selected.ImportPath)
		if err != nil {
			return Request{}, "", err
		}
		evidence, err := projectReviewedPackage(pkg)
		if err != nil {
			return Request{}, "", err
		}
		facts, err := discoverFacts(selected.Facts, pkg, review.Findings)
		if err != nil {
			return Request{}, "", fmt.Errorf("discover compatibility-pack facts for %s: %w", selected.ImportPath, err)
		}
		result.Packages[index] = Package{ImportPath: selected.ImportPath, Facts: facts, Evidence: evidence}
	}
	if err := ValidateRequest(result); err != nil {
		return Request{}, "", err
	}
	digest, err := ApprovalSHA256(result)
	if err != nil {
		return Request{}, "", err
	}
	return result, digest, nil
}

func validateDraft(draft Request) error {
	if draft.Schema != RequestSchema || draft.ID == "" {
		return errors.New("compatibility-pack discovery request identity is invalid")
	}
	if draft.Target.Kind != target.KindGoRun && draft.Target.Kind != target.KindGoTest {
		return errors.New("compatibility-pack discovery target kind is invalid")
	}
	if draft.Target.Package == "" || len(draft.Target.Package) > maximumRequestStringBytes || filepath.IsAbs(draft.Target.Package) ||
		draft.Target.ExpectedModule == "" || draft.Target.TestArguments == nil || len(draft.Target.TestArguments) > maximumRequestArguments ||
		draft.Target.BuildTags == nil || len(draft.Target.BuildTags) > maximumRequestBuildTags || !sortedUnique(draft.Target.BuildTags) {
		return errors.New("compatibility-pack discovery target is incomplete")
	}
	for _, argument := range draft.Target.TestArguments {
		if !validPortableArgument(argument) {
			return errors.New("compatibility-pack discovery target argument is invalid")
		}
	}
	if err := module.CheckPath(draft.Target.ExpectedModule); err != nil {
		return fmt.Errorf("compatibility-pack discovery expected module is invalid: %w", err)
	}
	if len(draft.Activation) == 0 || len(draft.Activation) > maximumRequestActivations || len(draft.Packages) == 0 || len(draft.Packages) > maximumRequestPackages {
		return errors.New("compatibility-pack discovery selectors are empty")
	}
	factCount := 0
	for index, activation := range draft.Activation {
		if err := module.CheckPath(activation.Path); err != nil || index > 0 && draft.Activation[index-1].Path >= activation.Path {
			return errors.New("compatibility-pack discovery activations are not canonical")
		}
	}
	for index, pkg := range draft.Packages {
		factCount += len(pkg.Facts)
		if factCount > maximumRequestFacts {
			return errors.New("compatibility-pack discovery fact count exceeds its bound")
		}
		if err := module.CheckImportPath(pkg.ImportPath); err != nil || index > 0 && draft.Packages[index-1].ImportPath >= pkg.ImportPath || len(pkg.Facts) == 0 {
			return errors.New("compatibility-pack discovery packages are not canonical")
		}
		for factIndex, fact := range pkg.Facts {
			if factIndex > 0 && compareFact(pkg.Facts[factIndex-1], fact) >= 0 || fact.Disposition != DispositionAllow && fact.Disposition != DispositionDeny {
				return errors.New("compatibility-pack discovery facts are not canonical")
			}
			switch fact.Kind {
			case FactCapability:
				if fact.Capability == "" {
					return errors.New("compatibility-pack discovery capability is empty")
				}
			case FactLinkname:
				if fact.Source == "" {
					return errors.New("compatibility-pack discovery linkname source is empty")
				}
			default:
				return errors.New("compatibility-pack discovery fact kind is invalid")
			}
		}
	}
	return nil
}

func reviewContainsMainModule(review target.CapabilityReview, expected string) bool {
	for _, pkg := range review.Closure.Packages {
		if pkg.Module != nil && pkg.Module.Main && pkg.Module.Path == expected {
			return true
		}
	}
	return false
}

func findReviewedModule(packages []target.CapabilityPackage, path string) (*target.CapabilityModule, error) {
	var found *target.CapabilityModule
	for _, pkg := range packages {
		if pkg.Module == nil || pkg.Module.Path != path {
			continue
		}
		if found != nil && capabilityModuleKey(found) != capabilityModuleKey(pkg.Module) {
			return nil, fmt.Errorf("compatibility-pack activation module %s is ambiguous", path)
		}
		found = pkg.Module
	}
	if found == nil {
		return nil, fmt.Errorf("compatibility-pack activation module %s is absent from the target review", path)
	}
	return found, nil
}

func findReviewedPackage(packages []target.CapabilityPackage, importPath string) (target.CapabilityPackage, error) {
	candidates := []target.CapabilityPackage{}
	for _, pkg := range packages {
		if pkg.ImportPath == importPath && pkg.ForTest == "" {
			candidates = append(candidates, pkg)
		}
	}
	if len(candidates) != 1 {
		return target.CapabilityPackage{}, fmt.Errorf("compatibility-pack package %s has %d canonical target-review matches", importPath, len(candidates))
	}
	return candidates[0], nil
}

func projectReviewedModule(moduleEvidence *target.CapabilityModule) (compatibility.PackModule, error) {
	if moduleEvidence == nil || moduleEvidence.Path == "" || moduleEvidence.Version == "" || moduleEvidence.Sum == "" {
		return compatibility.PackModule{}, errors.New("compatibility-pack reviewed module identity is incomplete")
	}
	result := compatibility.PackModule{
		Path: moduleEvidence.Path, Version: moduleEvidence.Version, Sum: moduleEvidence.Sum,
		Replacement: compatibility.PackReplacement{Kind: compatibility.ReplacementNone},
	}
	if moduleEvidence.Replacement == nil {
		return result, nil
	}
	if !moduleEvidence.Replacement.Local || moduleEvidence.Adapter == nil {
		return compatibility.PackModule{}, fmt.Errorf("compatibility-pack module %s has an unregistered replacement", moduleEvidence.Path)
	}
	adapter := moduleEvidence.Adapter
	result.Replacement = compatibility.PackReplacement{
		Kind: compatibility.ReplacementAdapter,
		Adapter: &compatibility.PackAdapter{
			ProfileName: adapter.ProfileName, ProfileImplementationSHA256: adapter.ProfileImplementationSHA256,
			Module: adapter.Adapter.Path, Version: adapter.Adapter.Version, Sum: adapter.Adapter.Sum,
			OriginalSourceInventorySHA256:    adapter.OriginalSourceInventorySHA256,
			ReplacementSourceInventorySHA256: adapter.ReplacementSourceInventorySHA256,
			PreparedSourceSetSHA256:          adapter.PreparedSourceSetSHA256,
		},
	}
	return result, nil
}

func projectReviewedPackage(pkg target.CapabilityPackage) (compatibility.PackRule, error) {
	moduleEvidence, err := projectReviewedModule(pkg.Module)
	if err != nil {
		return compatibility.PackRule{}, err
	}
	goSources := make([]compatibility.PackSource, len(pkg.Sources))
	digestSources := make([]compatibility.Source, 0, len(pkg.Sources)+len(pkg.ForeignSources))
	for index, source := range pkg.Sources {
		goSources[index] = compatibility.PackSource{Name: source.Name, SHA256: source.SHA256}
		digestSources = append(digestSources, compatibility.Source{Name: source.Name, SHA256: source.SHA256})
	}
	foreignSources := make([]compatibility.PackForeignSource, len(pkg.ForeignSources))
	for index, source := range pkg.ForeignSources {
		foreignSources[index] = compatibility.PackForeignSource{Kind: source.Kind, Name: source.Name, SHA256: source.SHA256}
		digestSources = append(digestSources, compatibility.Source{Name: source.Kind + ":" + source.Name, SHA256: source.SHA256})
	}
	return compatibility.PackRule{
		ImportPath: pkg.ImportPath, Module: moduleEvidence,
		SourceSetSHA256: compatibility.DigestSources(digestSources),
		GoSources:       goSources, ForeignSources: foreignSources,
		Capabilities: []string{}, Linknames: []compatibility.PackLinkname{},
	}, nil
}

func discoverFacts(requested []Fact, pkg target.CapabilityPackage, findings []target.CapabilityFinding) ([]Fact, error) {
	result := make([]Fact, len(requested))
	for index, fact := range requested {
		result[index] = fact
		result[index].Directives = []string{}
		switch fact.Kind {
		case FactCapability:
			if !reviewedCapabilityPresent(pkg, fact.Capability) {
				return nil, fmt.Errorf("requested capability %s is absent", fact.Capability)
			}
			result[index].Source = ""
			result[index].SHA256 = ""
		case FactLinkname:
			found := false
			for _, source := range pkg.Sources {
				if source.Name != fact.Source || len(source.LinknameDirectives) == 0 || source.MalformedLinkname {
					continue
				}
				result[index].SHA256 = source.SHA256
				result[index].Directives = append([]string{}, source.LinknameDirectives...)
				result[index].Capability = ""
				found = true
				break
			}
			if !found {
				return nil, fmt.Errorf("requested linkname source %s is absent or malformed", fact.Source)
			}
		}
	}
	for _, finding := range findings {
		if finding.Package.ImportPath != pkg.ImportPath || finding.Package.ForTest != pkg.ForTest {
			continue
		}
		fact, include := factFromFinding(finding)
		if !include {
			continue
		}
		found := false
		for _, existing := range result {
			if compareFact(existing, fact) == 0 {
				found = true
				break
			}
		}
		if !found {
			result = append(result, fact)
		}
	}
	slices.SortFunc(result, compareFact)
	return result, nil
}

func factFromFinding(finding target.CapabilityFinding) (Fact, bool) {
	switch finding.Kind {
	case target.FindingForbiddenImport, target.FindingForeignSource:
		return Fact{
			Kind: FactCapability, Capability: finding.Capability,
			Directives: []string{}, Disposition: DispositionDeny,
		}, true
	case target.FindingUnapprovedLinkname:
		return Fact{
			Kind: FactLinkname, Source: finding.SourceName, SHA256: finding.SourceSHA256,
			Directives: append([]string{}, finding.Directives...), Disposition: DispositionDeny,
		}, true
	default:
		return Fact{}, false
	}
}

func reviewedCapabilityPresent(pkg target.CapabilityPackage, capability string) bool {
	prefix, value, found := strings.Cut(capability, ":")
	if !found {
		return false
	}
	switch prefix {
	case "import":
		return slices.Contains(pkg.Imports, value)
	case "foreign":
		kind, name, found := strings.Cut(value, ":")
		if !found {
			return false
		}
		for _, source := range pkg.ForeignSources {
			if source.Kind == kind && source.Name == name {
				return true
			}
		}
	}
	return false
}

func capabilityModuleKey(module *target.CapabilityModule) string {
	if module == nil {
		return ""
	}
	return module.Path + "\x00" + module.Version + "\x00" + module.Sum
}
