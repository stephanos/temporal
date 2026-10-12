package archcheck

import (
	"slices"
	"strings"
)

func Owner(module, importPath string) string {
	if importPath == module {
		return "test-harness"
	}
	if !Within(importPath, module) {
		return ""
	}
	relative := strings.TrimPrefix(importPath, module+"/")
	if relative == "backends" {
		return "runner"
	}
	if relative == "backends/native" || Within(relative, "backends/native/execution") {
		return "native"
	}
	if relative == "cmd/gomad" {
		return "cli"
	}
	for path, owner := range map[string]string{
		"backends/wasm": "wasm", "cmd/gomad/internal/cli": "cli", "cmd/gomadtool": "developer", "internal/gomadtool": "developer", "internal/archcheck": "developer", "internal/puritycheck": "developer",
		"backends/native/toolchain": "toolchain", "backends/native/build": "target", "backends/native/transcript": "deterministicio", "backends/native/upgrade": "upgrade",
		"backends/native/conformance": "developer", "backends/native/generation": "developer",
		"backends/native/qualification": "qualification", "backends/native/preparation": "preparation",
		"captured": "captured", "volume": "volume", "internal/iowire": "iowire",
		"internal/compatibilitypack": "compatibility", "internal/canonicaljson": "canonicaljson", "internal/hostexec": "hostexec", "hostexec": "hostexec", "internal/hostfs": "hostfs", "hostfs": "hostfs", "internal/preparation": "preparation", "internal/sourceinventory": "sourceinventory",
	} {
		if Within(relative, path) {
			return owner
		}
	}
	owner, _, _ := strings.Cut(relative, "/")
	if slices.Contains([]string{"runner", "qualification", "target", "record", "artifact", "choice", "world", "simulation", "runtimesemantics"}, owner) {
		return owner
	}
	return ""
}
func OwnerMayImport(modulePath, owner, importedOwner, importing, imported string) bool {
	if owner == importedOwner {
		return true
	}
	if owner == "wasm" {
		return slices.ContainsFunc([]string{
			"artifact", "choice", "captured", "volume", "hostexec", "hostfs",
			"qualification", "qualification/workload", "record", "runner", "backends",
			"simulation/network", "target", "world", "runtimesemantics",
		}, func(path string) bool { return imported == modulePath+"/"+path })
	}
	if owner == "cli" && slices.Contains([]string{"wasm", "native", "hostexec", "hostfs"}, importedOwner) {
		return importing == modulePath+"/cmd/gomad/internal/cli" && slices.Contains([]string{modulePath + "/backends/wasm/backend", modulePath + "/backends/wasm/command", modulePath + "/backends/wasm/installation", modulePath + "/backends/native", modulePath + "/hostexec", modulePath + "/hostfs"}, imported)
	}
	if owner == "runner" && imported == modulePath+"/simulation/clock" {
		return true
	}
	allowed := map[string][]string{
		"native": {"runner", "target", "record", "artifact", "choice", "deterministicio", "captured", "volume", "iowire", "preparation", "world", "canonicaljson", "hostexec", "hostfs", "simulation"},
		"cli":    {"runner", "qualification", "target", "record", "artifact", "deterministicio", "captured", "volume", "iowire", "preparation", "toolchain", "canonicaljson"}, "developer": {"choice", "compatibility", "qualification", "simulation", "toolchain", "upgrade", "hostexec", "hostfs", "runtimesemantics"},
		"runner": {"target", "record", "artifact", "choice", "deterministicio", "captured", "volume", "iowire", "preparation", "world", "canonicaljson", "hostexec", "hostfs"}, "qualification": {"runner", "target", "record", "artifact", "choice", "deterministicio", "captured", "volume", "iowire", "preparation", "canonicaljson", "hostexec", "hostfs"},
		"target": {"compatibility", "record", "toolchain", "canonicaljson", "hostexec", "hostfs", "sourceinventory"}, "record": {"canonicaljson"}, "world": {"canonicaljson"}, "simulation": {"record", "canonicaljson"},
		"artifact": {"choice", "deterministicio", "captured", "volume", "iowire", "target", "record", "hostfs"}, "compatibility": {"target", "record", "canonicaljson", "hostfs"},
		"deterministicio": {"target", "record", "toolchain", "canonicaljson", "hostfs", "sourceinventory", "captured", "volume", "iowire"}, "captured": {"record", "canonicaljson", "iowire"}, "volume": {"iowire"},
		"preparation": {"target", "deterministicio", "captured", "volume", "iowire", "record"}, "toolchain": {"canonicaljson", "hostexec", "hostfs", "runtimesemantics"},
		"upgrade": {"qualification", "toolchain", "deterministicio", "captured", "volume", "iowire", "compatibility", "canonicaljson", "hostexec", "hostfs"}, "sourceinventory": {"hostfs"}, "runtimesemantics": {"hostfs"},
	}
	if !slices.Contains(allowed[owner], importedOwner) {
		return false
	}
	if (owner == "target" || owner == "deterministicio") && importedOwner == "toolchain" {
		return imported == modulePath+"/backends/native/toolchain/version" || imported == modulePath+"/backends/native/toolchain/installation"
	}
	if owner == "upgrade" && slices.Contains([]string{"compatibility", "deterministicio", "captured", "volume", "iowire", "canonicaljson"}, importedOwner) {
		return importing == modulePath+"/backends/native/upgrade/pinimpact" || importing == modulePath+"/backends/native/upgrade/adapterregen" ||
			(importing == modulePath+"/backends/native/upgrade" && importedOwner == "canonicaljson")
	}
	return true
}

func ModuleMayImport(modulePath, importing, imported string) bool {
	forbidden := map[string][]string{
		"artifact": {"runner"}, "record": {"runner"},
		"runner/internal/campaign":  {"runner/internal/execution", "runner/internal/corpus", "runner/internal/minimizer"},
		"runner/internal/execution": {"runner/internal/campaign", "runner/internal/corpus", "runner/internal/exploration"},
		"runner/internal/corpus":    {"runner/internal/campaign", "runner/internal/execution", "runner/internal/exploration"},
	}
	for module, denied := range forbidden {
		if Within(importing, modulePath+"/"+module) && slices.ContainsFunc(denied, func(prefix string) bool { return Within(imported, modulePath+"/"+prefix) }) {
			return false
		}
	}
	return true
}

func PackageEdges(module string, packages []HostPackage) []Finding {
	var findings []Finding
	add := func(category, path, detail string) { findings = append(findings, Finding{category, path, detail}) }
	required := map[string][]string{
		"backends/native/build": {"backends/native/build/internal/build", "backends/native/build/internal/capabilityreview", "backends/native/build/internal/provenance", "backends/native/toolchain/installation", "internal/sourceinventory"}, "backends/native/upgrade": {"backends/native/qualification/set", "backends/native/toolchain/version"},
		"backends/native/toolchain": {"backends/native/toolchain/installation"}, "backends/native/transcript": {"backends/native/toolchain/installation", "internal/sourceinventory"},
	}
	for _, pkg := range packages {
		owner := Owner(module, pkg.ImportPath)
		if owner == "" {
			add("ownerless", pkg.ImportPath, pkg.ImportPath)
		}
		for _, dependency := range required[strings.TrimPrefix(pkg.ImportPath, module+"/")] {
			if !slices.Contains(pkg.Imports, module+"/"+dependency) {
				add("required-edge", pkg.ImportPath, module+"/"+dependency)
			}
		}
		for _, imported := range pkg.Imports {
			if !Within(imported, module) {
				continue
			}
			if pkg.ImportPath == module+"/backends/native/toolchain/installation" || (pkg.ImportPath == module+"/internal/sourceinventory" && imported != module+"/internal/hostfs") || (pkg.ImportPath == module+"/backends/native/toolchain" && (Within(imported, module+"/qualification") || Within(imported, module+"/backends/native/qualification") || Within(imported, module+"/backends/native/upgrade"))) {
				add("module-edge", pkg.ImportPath, imported)
			}
			importedOwner := Owner(module, imported)
			if importedOwner == "" {
				add("ownerless", pkg.ImportPath, imported)
				continue
			}
			if !OwnerMayImport(module, owner, importedOwner, pkg.ImportPath, imported) {
				add("owner-edge", pkg.ImportPath, imported)
			}
			if !ModuleMayImport(module, pkg.ImportPath, imported) {
				add("module-edge", pkg.ImportPath, imported)
			}
		}
	}
	return findings
}
