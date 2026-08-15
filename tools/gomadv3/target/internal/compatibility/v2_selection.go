package compatibility

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"sort"
	"strings"
)

type ValidatedPack struct {
	pack   Pack
	digest string
}

type selectedPack struct {
	ValidatedPack
}

func LoadPack(data []byte) (ValidatedPack, error) {
	decoded, err := DecodePack(data)
	if err != nil {
		return ValidatedPack{}, err
	}
	digest := sha256.Sum256(data)
	return ValidatedPack{pack: decoded, digest: fmt.Sprintf("sha256:%x", digest)}, nil
}

func loadPacksV2() ([]ValidatedPack, error) {
	entries, err := packFiles.ReadDir("packs")
	if err != nil {
		return nil, fmt.Errorf("read compatibility packs: %w", err)
	}
	if len(entries) == 0 || len(entries) > maximumPackRules {
		return nil, errors.New("compatibility pack count is invalid")
	}
	result := make([]ValidatedPack, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil, fmt.Errorf("compatibility pack entry %s is invalid", entry.Name())
		}
		contents, err := packFiles.ReadFile("packs/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read compatibility pack %s: %w", entry.Name(), err)
		}
		candidate, err := LoadPack(contents)
		if err != nil {
			return nil, fmt.Errorf("compatibility pack %s: %w", entry.Name(), err)
		}
		if entry.Name() != candidate.pack.ID+".json" {
			return nil, fmt.Errorf("compatibility pack %s does not match ID %s", entry.Name(), candidate.pack.ID)
		}
		result = append(result, candidate)
	}
	return result, nil
}

func SelectPacks(packs []ValidatedPack, packages []Package) (Selection, error) {
	return SelectPacksForPlatform(packs, packages, runtime.GOOS+"/"+runtime.GOARCH)
}

func SelectPacksForPlatform(packs []ValidatedPack, packages []Package, platform string) (Selection, error) {
	ordered := append([]ValidatedPack{}, packs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].pack.ID < ordered[j].pack.ID })
	for index, candidate := range ordered {
		if err := ValidatePack(candidate.pack); err != nil {
			return Selection{}, err
		}
		if candidate.digest == "" {
			return Selection{}, errors.New("validated compatibility pack has no digest")
		}
		if index > 0 && ordered[index-1].pack.ID == candidate.pack.ID {
			return Selection{}, fmt.Errorf("compatibility pack ID is duplicated: %s", candidate.pack.ID)
		}
	}
	selected := make([]selectedPack, 0, len(ordered))
	for _, candidate := range ordered {
		if slices.Contains(candidate.pack.Governance.Platforms, platform) && matchesPackActivation(candidate.pack.Activation, packages) {
			selected = append(selected, selectedPack{ValidatedPack: candidate})
		}
	}
	return Selection{packs: selected}, nil
}

func VerifyPackIdentities(packs []ValidatedPack, identities []Identity) error {
	available := make(map[string]string, len(packs))
	for _, candidate := range packs {
		if err := ValidatePack(candidate.pack); err != nil {
			return err
		}
		if candidate.digest == "" {
			return errors.New("validated compatibility pack has no digest")
		}
		if _, duplicate := available[candidate.pack.ID]; duplicate {
			return fmt.Errorf("compatibility pack ID is duplicated: %s", candidate.pack.ID)
		}
		available[candidate.pack.ID] = candidate.digest
	}
	for index, identity := range identities {
		if index > 0 && identities[index-1].ID >= identity.ID {
			return errors.New("compatibility pack identities are not sorted and unique")
		}
		if digest, found := available[identity.ID]; !found || digest != identity.SHA256 {
			return fmt.Errorf("compatibility pack %s is unavailable or modified", identity.ID)
		}
	}
	return nil
}

func matchesPackActivation(patterns []PackModule, packages []Package) bool {
	for _, pattern := range patterns {
		found := false
		for _, pkg := range packages {
			if matchesPackModule(pattern, pkg.Module) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func matchesPackModule(pattern PackModule, actual Module) bool {
	if pattern.Path != actual.Path || pattern.Version != actual.Version || pattern.Sum != actual.Sum {
		return false
	}
	switch pattern.Replacement.Kind {
	case ReplacementNone:
		return !actual.Replaced && !actual.LocalReplacement && actual.Adapter == nil
	case ReplacementAdapter:
		return actual.Replaced && actual.LocalReplacement && actual.Adapter != nil && matchesPackAdapter(*pattern.Replacement.Adapter, *actual.Adapter)
	default:
		return false
	}
}

func matchesPackAdapter(pattern PackAdapter, actual AdapterEvidence) bool {
	return pattern.ProfileName == actual.ProfileName &&
		pattern.ProfileImplementationSHA256 == actual.ProfileImplementationSHA256 &&
		pattern.Module == actual.Module && pattern.Version == actual.Version && pattern.Sum == actual.Sum &&
		pattern.OriginalSourceInventorySHA256 == actual.OriginalSourceInventorySHA256 &&
		pattern.ReplacementSourceInventorySHA256 == actual.ReplacementSourceInventorySHA256 &&
		pattern.PreparedSourceSetSHA256 == actual.PreparedSourceSetSHA256
}

func matchesPackRule(pattern PackRule, actual Package) bool {
	return pattern.ImportPath == actual.ImportPath && matchesPackModule(pattern.Module, actual.Module) &&
		pattern.SourceSetSHA256 == actual.SourceSetSHA256 &&
		slices.EqualFunc(pattern.GoSources, actual.GoSources, func(left PackSource, right Source) bool {
			return left.Name == right.Name && left.SHA256 == right.SHA256
		}) &&
		slices.EqualFunc(pattern.ForeignSources, actual.ForeignSources, func(left PackForeignSource, right ForeignSource) bool {
			return left.Kind == right.Kind && left.Name == right.Name && left.SHA256 == right.SHA256
		})
}

func projectPackEvidence(selected selectedPack) PackEvidence {
	governance := selected.pack.Governance
	governance.Workloads = append([]string{}, governance.Workloads...)
	governance.Platforms = append([]string{}, governance.Platforms...)
	evidence := PackEvidence{
		ID: selected.pack.ID, SHA256: selected.digest, RequestSHA256: selected.pack.RequestSHA256,
		Governance: &governance, Activation: make([]ModuleEvidence, len(selected.pack.Activation)),
		Rules: make([]PackageRuleEvidence, len(selected.pack.Rules)),
	}
	for index, module := range selected.pack.Activation {
		evidence.Activation[index] = projectV2ModuleEvidence(module)
	}
	for index, rule := range selected.pack.Rules {
		linknames := make([]LinknameEvidence, len(rule.Linknames))
		for linknameIndex, linkname := range rule.Linknames {
			linknames[linknameIndex] = LinknameEvidence{
				Source: linkname.Source, SHA256: linkname.SHA256,
				Directives: append([]string{}, linkname.Directives...),
			}
		}
		evidence.Rules[index] = PackageRuleEvidence{
			ImportPath: rule.ImportPath, Module: projectV2ModuleEvidence(rule.Module), SourceSetSHA256: rule.SourceSetSHA256,
			GoSources: append([]PackSource{}, rule.GoSources...), ForeignSources: append([]PackForeignSource{}, rule.ForeignSources...),
			Capabilities: append([]string{}, rule.Capabilities...), Linknames: linknames,
		}
	}
	return evidence
}

func projectV2ModuleEvidence(module PackModule) ModuleEvidence {
	evidence := ModuleEvidence{Path: module.Path, Version: module.Version, Sum: module.Sum, Replacement: string(module.Replacement.Kind)}
	if module.Replacement.Adapter != nil {
		adapter := *module.Replacement.Adapter
		evidence.Adapter = &adapter
	}
	return evidence
}
