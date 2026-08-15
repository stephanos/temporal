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
	Active     []bool
	Eliminated []int
	Denied     []Fact
}

func ProjectFindings(manifest Manifest, packages []ClosurePackage, findings []ClosureFinding) FindingProjection {
	standard := make(map[string]struct{})
	for _, pkg := range packages {
		if pkg.Standard {
			standard[pkg.ImportPath] = struct{}{}
		}
	}
	liveCapabilities := make(map[string]struct{})
	for _, fact := range manifest.Facts {
		if fact.Kind != FactKindCapability {
			continue
		}
		if _, isStandard := standard[fact.OwnerPackage]; isStandard {
			continue
		}
		liveCapabilities[fact.Capability] = struct{}{}
	}
	projection := FindingProjection{Active: make([]bool, len(findings)), Eliminated: []int{}, Denied: []Fact{}}
	for _, fact := range manifest.Facts {
		if fact.Kind != FactKindBoundary || fact.Disposition != DispositionDenied {
			continue
		}
		if _, isStandard := standard[fact.OwnerPackage]; !isStandard {
			projection.Denied = append(projection.Denied, fact)
		}
	}
	for index, finding := range findings {
		active := true
		if finding.Kind == "forbidden_import" {
			_, active = liveCapabilities[finding.Capability]
		}
		projection.Active[index] = active
		if !active {
			projection.Eliminated = append(projection.Eliminated, index)
		}
	}
	return projection
}
