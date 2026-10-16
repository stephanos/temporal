package compatibility

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"sort"
	"strings"
)

func (selection Selection) Evidence() []PackEvidence {
	evidence := make([]PackEvidence, 0, len(selection.packs))
	for _, selected := range selection.packs {
		evidence = append(evidence, projectPackEvidence(selected))
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].ID < evidence[j].ID })
	return evidence
}

func (selection Selection) Identities() []Identity {
	identities := make([]Identity, 0, len(selection.packs))
	for _, selected := range selection.packs {
		identities = append(identities, Identity{ID: selected.pack.ID, SHA256: selected.digest})
	}
	sort.Slice(identities, func(i, j int) bool { return identities[i].ID < identities[j].ID })
	return identities
}

func (selection Selection) AllowsCapability(pkg Package, capability string) bool {
	return selection.Evaluate(pkg, Fact{Kind: FactCapability, Capability: capability}).Allowed
}

func (selection Selection) HasPackage(pkg Package) bool {
	for _, selected := range selection.packs {
		for _, candidate := range selected.pack.Rules {
			if matchesPackRule(candidate, pkg) {
				return true
			}
		}
	}
	return false
}

func (selection Selection) AllowsLinkname(pkg Package, source, digest string, directives []string) bool {
	return selection.Evaluate(pkg, Fact{Kind: FactLinkname, Source: source, SHA256: digest, Directives: directives}).Allowed
}

func (selection Selection) Evaluate(pkg Package, fact Fact) Decision {
	for _, selected := range selection.packs {
		for _, candidate := range selected.pack.Rules {
			if !matchesPackRule(candidate, pkg) {
				continue
			}
			switch fact.Kind {
			case FactCapability:
				if slices.Contains(candidate.Capabilities, fact.Capability) {
					return Decision{Allowed: true, Disposition: DispositionAllowedExactPack, PackID: selected.pack.ID}
				}
			case FactLinkname:
				for _, allowed := range candidate.Linknames {
					if allowed.Source == fact.Source && allowed.SHA256 == fact.SHA256 && slices.Equal(allowed.Directives, fact.Directives) {
						return Decision{Allowed: true, Disposition: DispositionAllowedExactPack, PackID: selected.pack.ID}
					}
				}
			case FactMalformedLinkname, FactNoReviewedGoSource:
				continue
			}
		}
	}
	return Decision{Disposition: DispositionDenied, Remediation: remediationFor(pkg, fact)}
}

func DigestSources(sources []Source) string {
	hash := sha256.New()
	//gomad:dynamic reviewed 2026-10-12 receiver is the locally constructed SHA256 or HMAC memory hash
	_, _ = hash.Write([]byte("gomad.compatibility-source-set/v1\x00"))
	for _, source := range sources {
		//gomad:dynamic reviewed 2026-10-12 receiver is the locally constructed SHA256 or HMAC memory hash
		_, _ = hash.Write([]byte(source.Name))
		//gomad:dynamic reviewed 2026-10-12 receiver is the locally constructed SHA256 or HMAC memory hash
		_, _ = hash.Write([]byte{0})
		//gomad:dynamic reviewed 2026-10-12 receiver is the locally constructed SHA256 or HMAC memory hash
		_, _ = hash.Write([]byte(source.SHA256))
		//gomad:dynamic reviewed 2026-10-12 receiver is the locally constructed SHA256 or HMAC memory hash
		_, _ = hash.Write([]byte{0})
	}
	//gomad:dynamic reviewed 2026-10-12 receiver is the locally constructed SHA256 or HMAC memory hash
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

func remediationFor(pkg Package, fact Fact) RemediationCategory {
	switch fact.Kind {
	case FactNoReviewedGoSource:
		return RemediationRemoveDependency
	case FactMalformedLinkname:
		return RemediationRemainUnsupported
	case FactLinkname:
		if pkg.Module.Path != "" && pkg.Module.Version != "" && (pkg.Module.Sum != "" || pkg.Module.LocalReplacement) {
			return RemediationAddExactPack
		}
		return RemediationRemainUnsupported
	case FactCapability:
		if strings.HasPrefix(fact.Capability, "import:") {
			importPath := strings.TrimPrefix(fact.Capability, "import:")
			if importPath == "os/exec" || importPath == "os/signal" || importPath == "os/user" || importPath == "plugin" || importPath == "runtime/cgo" {
				return RemediationRemainUnsupported
			}
		}
		if pkg.Module.LocalReplacement {
			return RemediationAddAdapter
		}
		if pkg.Module.Path != "" && pkg.Module.Version != "" && pkg.Module.Sum != "" {
			return RemediationAddExactPack
		}
		return RemediationModelOperation
	default:
		return RemediationRemainUnsupported
	}
}

func sortedUnique(values []string) bool {
	for index, value := range values {
		if value == "" || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}
