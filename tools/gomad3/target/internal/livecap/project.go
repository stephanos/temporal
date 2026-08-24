package livecap

type ClosurePackage struct {
	ImportPath string
	ForTest    string
	Root       bool
	Standard   bool
}

type ClosureFinding struct {
	Kind       string
	Package    string
	ForTest    string
	Capability string
}

type FindingProjection struct {
	Active        []bool
	Eliminated    []int
	Guarded       []int
	Denied        []Fact
	GuardedDenied []Fact
}

func ProjectFindings(manifest Manifest, packages []ClosurePackage, findings []ClosureFinding) FindingProjection {
	type ownerIdentity struct {
		ImportPath string
		ForTest    string
	}
	known := make(map[ownerIdentity]ClosurePackage, len(packages))
	var root *ClosurePackage
	multipleRoots := false
	for _, pkg := range packages {
		known[ownerIdentity{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest}] = pkg
		if pkg.Root && !pkg.Standard {
			if root == nil && !multipleRoots {
				candidate := pkg
				root = &candidate
			} else {
				root = nil
				multipleRoots = true
			}
		}
	}
	resolveOwner := func(fact Fact) (ClosurePackage, bool) {
		if pkg, ok := known[ownerIdentity{ImportPath: fact.OwnerPackage, ForTest: fact.ForTest}]; ok {
			return pkg, true
		}
		if fact.OwnerPackage == "main" && fact.ForTest == "" && root != nil {
			return *root, true
		}
		return ClosurePackage{}, false
	}
	findingKey := func(importPath, forTest, capability string) string {
		return importPath + "\x00" + forTest + "\x00" + capability
	}
	liveCapabilities := make(map[string]struct{})
	guardedCapabilities := make(map[string]struct{})
	unknownLiveCapabilities := make(map[string]struct{})
	for _, fact := range manifest.Facts {
		if fact.Kind != FactKindCapability && fact.Kind != FactKindGuard {
			continue
		}
		owner, found := resolveOwner(fact)
		if found && owner.Standard {
			continue
		}
		if !found {
			if fact.Kind == FactKindCapability {
				unknownLiveCapabilities[fact.Capability] = struct{}{}
			}
			continue
		}
		key := findingKey(owner.ImportPath, owner.ForTest, fact.Capability)
		if fact.Kind == FactKindGuard {
			guardedCapabilities[key] = struct{}{}
		} else {
			liveCapabilities[key] = struct{}{}
		}
	}
	projection := FindingProjection{Active: make([]bool, len(findings)), Eliminated: []int{}, Guarded: []int{}, Denied: []Fact{}, GuardedDenied: []Fact{}}
	for _, fact := range manifest.Facts {
		if fact.Kind != FactKindBoundary && fact.Kind != FactKindGuard {
			continue
		}
		owner, found := resolveOwner(fact)
		if !found || !owner.Standard {
			if fact.Kind == FactKindBoundary && fact.Disposition == DispositionDenied {
				projection.Denied = append(projection.Denied, fact)
			} else if fact.Kind == FactKindGuard && !isForbiddenImportGuard(fact) {
				projection.GuardedDenied = append(projection.GuardedDenied, fact)
			}
		}
	}
	for index, finding := range findings {
		active := true
		guarded := false
		if finding.Kind == "forbidden_import" {
			key := findingKey(finding.Package, finding.ForTest, finding.Capability)
			_, active = liveCapabilities[key]
			if _, unknown := unknownLiveCapabilities[finding.Capability]; unknown {
				active = true
			}
			_, guarded = guardedCapabilities[key]
		}
		projection.Active[index] = active
		if active {
			continue
		}
		if guarded {
			projection.Guarded = append(projection.Guarded, index)
		} else {
			projection.Eliminated = append(projection.Eliminated, index)
		}
	}
	return projection
}

func isForbiddenImportGuard(fact Fact) bool {
	const prefix = "import:"
	return len(fact.Capability) > len(prefix) && fact.Capability[:len(prefix)] == prefix
}
