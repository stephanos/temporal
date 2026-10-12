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
	if owner == "wasm" {
		for _, path := range []string{
			"artifact", "choice", "cmd/gomad/api", "captured", "volume", "hostexec", "hostfs",
			"qualification", "qualification/workload", "record", "runner", "runner/backend",
			"simulation/network", "target", "world", "runtimesemantics",
		} {
			if imported == modulePath+"/"+path {
				return true
			}
		}
		return false
	}
	if owner == "cli" && importedOwner == "wasm" {
		return importing == modulePath+"/backends/wasm/cmd/gomad-wasm"
	}
	if owner == "cli" && importedOwner == "hostexec" {
		return importing == modulePath+"/backends/wasm/cmd/gomad-wasm" && imported == modulePath+"/hostexec"
	}
	if owner == "developer" && importedOwner == "canonicaljson" {
		return importing == modulePath+"/internal/gomadtool/architecture" && imported == modulePath+"/internal/canonicaljson"
	}
	allowed := map[string][]string{
		"cli":              {"runner", "qualification", "target", "record", "artifact", "deterministicio", "captured", "volume", "iowire", "preparation", "toolchain", "canonicaljson"},
		"developer":        {"choice", "compatibility", "qualification", "simulation", "toolchain", "upgrade", "hostexec", "hostfs", "runtimesemantics"},
		"runner":           {"target", "record", "artifact", "choice", "deterministicio", "captured", "volume", "iowire", "preparation", "world", "canonicaljson", "hostexec", "hostfs"},
		"qualification":    {"runner", "target", "record", "artifact", "choice", "deterministicio", "captured", "volume", "iowire", "preparation", "canonicaljson", "hostexec", "hostfs"},
		"target":           {"compatibility", "record", "toolchain", "canonicaljson", "hostexec", "hostfs", "sourceinventory"},
		"record":           {"canonicaljson"},
		"artifact":         {"choice", "deterministicio", "captured", "volume", "iowire", "target", "record", "hostfs"},
		"compatibility":    {"target", "record", "canonicaljson", "hostfs"},
		"deterministicio":  {"target", "record", "toolchain", "canonicaljson", "hostfs", "sourceinventory", "captured", "volume", "iowire"},
		"captured":         {"record", "canonicaljson", "iowire"},
		"volume":           {"iowire"},
		"preparation":      {"target", "deterministicio", "captured", "volume", "iowire", "record"},
		"world":            {"canonicaljson"},
		"simulation":       {"record", "canonicaljson"},
		"toolchain":        {"canonicaljson", "hostexec", "hostfs", "runtimesemantics"},
		"upgrade":          {"qualification", "toolchain", "deterministicio", "captured", "volume", "iowire", "compatibility", "canonicaljson", "hostexec", "hostfs"},
		"sourceinventory":  {"hostfs"},
		"runtimesemantics": {"hostfs"},
	}
	if !slices.Contains(allowed[owner], importedOwner) {
		return false
	}
	// Target preparation and deterministic I/O read the pinned version and the
	// validated installation layout; the builder stays out of their reach.
	if (owner == "target" || owner == "deterministicio") && importedOwner == "toolchain" {
		return imported == modulePath+"/backends/native/toolchain/version" || imported == modulePath+"/backends/native/toolchain/installation"
	}
	// Only the maintenance engines read adapters and compatibility packs.
	// The public compatibility facade also serializes its report as JSON.
	if owner == "upgrade" && slices.Contains([]string{"compatibility", "deterministicio", "captured", "volume", "iowire", "canonicaljson"}, importedOwner) {
		return importing == modulePath+"/backends/native/upgrade/pinimpact" || importing == modulePath+"/backends/native/upgrade/adapterregen" ||
			(importing == modulePath+"/backends/native/upgrade" && importedOwner == "canonicaljson")
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
		"backends/native/build":      {"backends/native/build/internal/build", "backends/native/build/internal/capabilityreview", "backends/native/build/internal/provenance", "backends/native/toolchain/installation", "internal/sourceinventory"},
		"backends/native/upgrade":    {"backends/native/qualification/set", "backends/native/toolchain/version"},
		"backends/native/toolchain":  {"backends/native/toolchain/installation"},
		"backends/native/transcript": {"backends/native/toolchain/installation", "internal/sourceinventory"},
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
			if pkg.ImportPath == module+"/backends/native/toolchain/installation" || (pkg.ImportPath == module+"/internal/sourceinventory" && imported != module+"/internal/hostfs") || (pkg.ImportPath == module+"/backends/native/toolchain" && (Within(imported, module+"/qualification") || Within(imported, module+"/backends/native/qualification") || Within(imported, module+"/backends/native/upgrade"))) {
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
