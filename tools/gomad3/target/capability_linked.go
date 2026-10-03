package target

import (
	"bytes"
	"sort"

	"go.temporal.io/server/tools/gomad3/target/internal/livecap"
)

// Linked projection narrows a closure review to the capabilities a linked
// target's embedded record shows live, guarded or denied at a boundary.

// readLinkedCapabilityRecord extracts the capability record a linked target
// embeds and checks it against the toolchain that built it.
func readLinkedCapabilityRecord(path string, identity ToolchainIdentity) (livecap.Record, error) {
	record, err := livecap.Read(path, livecap.Expectation{
		GoVersion: identity.GoVersion, ToolchainBuildKey: identity.BuildKey, GOOS: identity.TargetGOOS, GOARCH: identity.TargetGOARCH,
	})
	if err != nil {
		return livecap.Record{}, linkedCapabilityError(err)
	}
	return record, nil
}

func projectLinkedCapabilityReview(review CapabilityReview, record livecap.Record, mode CapabilityMode) CapabilityReview {
	packages := make([]livecap.ClosurePackage, len(review.Closure.Packages))
	for index, pkg := range review.Closure.Packages {
		packages[index] = livecap.ClosurePackage{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Root: pkg.Root, Standard: pkg.Standard}
	}
	findings := make([]livecap.ClosureFinding, len(review.Findings))
	for index, finding := range review.Findings {
		findings[index] = livecap.ClosureFinding{
			Kind: string(finding.Kind), Package: finding.Package.ImportPath, ForTest: finding.Package.ForTest, Capability: finding.Capability,
		}
	}
	projection := livecap.ProjectFindings(record.Manifest, packages, findings)
	active := make([]CapabilityFinding, 0, len(review.Findings)-len(projection.Eliminated)-len(projection.Guarded))
	guardedIndexes := make(map[int]struct{}, len(projection.Guarded))
	for _, index := range projection.Guarded {
		guardedIndexes[index] = struct{}{}
	}
	guarded := make([]CapabilityFinding, 0, len(projection.Guarded))
	eliminated := make([]CapabilityFinding, 0, len(projection.Eliminated))
	for index, finding := range review.Findings {
		finding.Directives = append([]string{}, finding.Directives...)
		if projection.Active[index] {
			active = append(active, finding)
		} else if _, protected := guardedIndexes[index]; protected && mode == CapabilityModeGuarded {
			guarded = append(guarded, finding)
		} else if protected {
			active = append(active, finding)
		} else {
			eliminated = append(eliminated, finding)
		}
	}
	active = append(active, projectDeniedBoundaryFindings(review.Closure.Packages, projection.Denied)...)
	guardedBoundaries := projectDeniedBoundaryFindings(review.Closure.Packages, projection.GuardedDenied)
	if mode == CapabilityModeGuarded {
		guarded = append(guarded, guardedBoundaries...)
	} else {
		active = append(active, guardedBoundaries...)
	}
	sort.Slice(active, func(i, j int) bool { return compareCapabilityFinding(active[i], active[j]) < 0 })
	sort.Slice(guarded, func(i, j int) bool { return compareCapabilityFinding(guarded[i], guarded[j]) < 0 })
	review.CapabilityMode = mode
	review.CapabilityManifest = capabilityManifest(record)
	review.Findings = active
	review.GuardedFindings = guarded
	review.EliminatedFindings = eliminated
	return review
}

func projectDeniedBoundaryFindings(packages []CapabilityPackage, facts []livecap.Fact) []CapabilityFinding {
	result := []CapabilityFinding{}
	seen := make(map[string]struct{})
	for _, fact := range facts {
		pkg, found := capabilityOwnerPackage(packages, fact.OwnerPackage, fact.ForTest)
		if !found {
			continue
		}
		key := pkg.ImportPath + "\x00" + pkg.ForTest + "\x00" + fact.Capability
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, CapabilityFinding{
			Kind: FindingDeniedBoundary, Package: capabilityPackageReference(pkg), Module: copyCapabilityModule(pkg.Module),
			SourceSetSHA256: capabilityCompatibilityPackage(pkg).SourceSetSHA256, Directives: []string{},
			Capability: fact.Capability, PolicyDisposition: DispositionDenied, Remediation: RemediationModelOperation,
		})
	}
	return result
}

func capabilityOwnerPackage(packages []CapabilityPackage, owner, forTest string) (CapabilityPackage, bool) {
	for _, pkg := range packages {
		if pkg.ImportPath == owner && pkg.ForTest == forTest {
			return pkg, true
		}
	}
	for _, pkg := range packages {
		if pkg.Root && !pkg.Standard {
			return pkg, true
		}
	}
	return CapabilityPackage{}, false
}

func capabilityManifest(record livecap.Record) *CapabilityManifest {
	return &CapabilityManifest{
		Schema: record.Manifest.Schema, SHA256: record.SHA256, Bytes: uint64(len(record.Payload)), Facts: uint64(len(record.Manifest.Facts)),
		ProducerImplementationSHA256: record.Manifest.ProducerImplementationSHA256,
		GuardImplementationSHA256:    record.Manifest.GuardImplementationSHA256,
		CapabilityUniverseSHA256:     record.Manifest.CapabilityUniverseSHA256,
		Payload:                      append([]byte(nil), record.Payload...),
	}
}

func cloneCapabilityManifest(manifest *CapabilityManifest) *CapabilityManifest {
	if manifest == nil {
		return nil
	}
	cloned := *manifest
	cloned.Payload = append([]byte(nil), manifest.Payload...)
	return &cloned
}

func sameCapabilityManifest(left, right *CapabilityManifest) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Schema == right.Schema && left.SHA256 == right.SHA256 && left.Bytes == right.Bytes && left.Facts == right.Facts &&
		left.ProducerImplementationSHA256 == right.ProducerImplementationSHA256 && left.GuardImplementationSHA256 == right.GuardImplementationSHA256 && left.CapabilityUniverseSHA256 == right.CapabilityUniverseSHA256 &&
		(len(left.Payload) == 0 || bytes.Equal(left.Payload, right.Payload))
}
