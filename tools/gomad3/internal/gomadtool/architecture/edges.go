package architecture

import (
	"fmt"
	"slices"
	"strings"
)

func OwnerMayImport(modulePath, owner, importedOwner, importing, imported string) bool {
	if owner == importedOwner {
		return true
	}
	allowed := map[string][]string{
		"cli":             {"runner", "qualification", "target", "record", "artifact", "deterministicio", "preparation", "toolchain", "canonicaljson"},
		"developer":       {"choice", "compatibility", "qualification", "simulation", "toolchain", "upgrade", "hostexec", "hostfs"},
		"runner":          {"target", "record", "artifact", "choice", "deterministicio", "preparation", "world", "canonicaljson", "hostexec", "hostfs"},
		"qualification":   {"runner", "target", "record", "artifact", "choice", "deterministicio", "preparation", "canonicaljson", "hostexec", "hostfs"},
		"target":          {"compatibility", "record", "toolchain", "canonicaljson", "hostexec", "hostfs", "sourceinventory"},
		"record":          {"canonicaljson"},
		"artifact":        {"choice", "deterministicio", "target", "record", "hostfs"},
		"compatibility":   {"target", "record", "canonicaljson", "hostfs"},
		"deterministicio": {"target", "record", "toolchain", "canonicaljson", "hostfs", "sourceinventory"},
		"preparation":     {"target", "deterministicio", "record"},
		"world":           {"canonicaljson"},
		"simulation":      {"record", "canonicaljson"},
		"toolchain":       {"canonicaljson", "hostexec", "hostfs"},
		"upgrade":         {"qualification", "toolchain", "deterministicio", "compatibility", "canonicaljson", "hostexec", "hostfs"},
		"sourceinventory": {"hostfs"},
	}
	if !slices.Contains(allowed[owner], importedOwner) {
		return false
	}
	// Target preparation and deterministic I/O read the pinned version and the
	// validated installation layout; the builder stays out of their reach.
	if (owner == "target" || owner == "deterministicio") && importedOwner == "toolchain" {
		return imported == modulePath+"/toolchain/version" || imported == modulePath+"/toolchain/installation"
	}
	// Only the maintenance engines read adapters and compatibility packs.
	// The public compatibility facade also serializes its report as JSON.
	if owner == "upgrade" && (importedOwner == "compatibility" || importedOwner == "deterministicio" || importedOwner == "canonicaljson") {
		return importing == modulePath+"/upgrade/pinimpact" || importing == modulePath+"/upgrade/adapterregen" ||
			(importing == modulePath+"/upgrade" && importedOwner == "canonicaljson")
	}
	return true
}

func ModuleMayImport(modulePath, importing, imported string) bool {
	if !strings.HasPrefix(imported, modulePath+"/") {
		return true
	}
	forbidden := map[string][]string{
		modulePath + "/artifact": {modulePath + "/runner"},
		modulePath + "/record":   {modulePath + "/runner"},
		modulePath + "/runner/internal/campaign": {
			modulePath + "/runner/internal/execution", modulePath + "/runner/internal/corpus", modulePath + "/runner/internal/minimizer",
		},
		modulePath + "/runner/internal/execution": {
			modulePath + "/runner/internal/campaign", modulePath + "/runner/internal/corpus", modulePath + "/runner/internal/exploration",
		},
		modulePath + "/runner/internal/corpus": {
			modulePath + "/runner/internal/campaign", modulePath + "/runner/internal/execution", modulePath + "/runner/internal/exploration",
		},
	}
	for module, denied := range forbidden {
		if importing == module || strings.HasPrefix(importing, module+"/") {
			for _, prefix := range denied {
				if imported == prefix || strings.HasPrefix(imported, prefix+"/") {
					return false
				}
			}
		}
	}
	return true
}

func PackageEdges(module string, packages []Package) []Finding {
	var findings []Finding
	required := map[string][]string{
		"target":          {"target/internal/build", "target/internal/capabilityreview", "target/internal/provenance", "toolchain/installation", "internal/sourceinventory"},
		"upgrade":         {"qualification/set", "toolchain/version"},
		"toolchain":       {"toolchain/installation"},
		"deterministicio": {"toolchain/installation", "internal/sourceinventory"},
	}
	for _, pkg := range packages {
		owner := Owner(module, pkg.ImportPath)
		for _, dependency := range required[strings.TrimPrefix(pkg.ImportPath, module+"/")] {
			if !slices.Contains(pkg.Imports, module+"/"+dependency) {
				findings = append(findings, Finding{Category: "required-edge", Path: pkg.ImportPath, Detail: module + "/" + dependency})
			}
		}
		for _, imported := range pkg.Imports {
			if !Within(imported, module) {
				continue
			}
			if pkg.ImportPath == module+"/toolchain/installation" || (pkg.ImportPath == module+"/internal/sourceinventory" && imported != module+"/internal/hostfs") || (pkg.ImportPath == module+"/toolchain" && (imported == module+"/qualification" || imported == module+"/upgrade")) {
				findings = append(findings, Finding{Category: "module-edge", Path: pkg.ImportPath, Detail: imported})
			}
			importedOwner := Owner(module, imported)
			if importedOwner == "" {
				findings = append(findings, Finding{Category: "ownerless", Path: pkg.ImportPath, Detail: imported})
				continue
			}
			if !OwnerMayImport(module, owner, importedOwner, pkg.ImportPath, imported) {
				findings = append(findings, Finding{Category: "owner-edge", Path: pkg.ImportPath, Detail: fmt.Sprintf("%s imports %s", owner, importedOwner)})
			}
			if !ModuleMayImport(module, pkg.ImportPath, imported) {
				findings = append(findings, Finding{Category: "module-edge", Path: pkg.ImportPath, Detail: imported})
			}
		}
	}
	return findings
}
