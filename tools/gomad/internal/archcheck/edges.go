package archcheck

import (
	"slices"
	"strings"
)

func Owner(module, importPath string) string {
	if !Within(importPath, module) {
		return ""
	}
	relative := strings.TrimPrefix(importPath, module+"/")
	if Within(relative, "internal/backends/testfixture") || Within(relative, "internal/evidence/record/testfixture") {
		return "testfixture"
	}
	if relative == "internal/backends" {
		return "runner"
	}
	if relative == "internal/backends/native" || Within(relative, "internal/backends/native/execution") {
		return "native"
	}
	if relative == "cmd/gomad" {
		return "cli"
	}
	if Within(relative, "internal/evidence/artifact/capture") {
		return "captured"
	}
	for path, owner := range map[string]string{
		"internal/backends/wasm": "wasm", "cmd/gomad/internal/cli": "cli", "internal/backends/native/cmd/gomadtool": "developer", "internal/archcheck": "developer",
		"internal/backends/native/toolchain": "toolchain", "internal/backends/native/build": "target", "internal/backends/native/transcript": "deterministicio", "internal/backends/native/upgrade": "upgrade",
		"internal/backends/native/conformance": "developer", "internal/backends/native/generation": "developer",
		"internal/backends/native/qualification": "qualification", "internal/backends/native/preparation": "preparation",
		"internal/evidence/artifact/capture": "captured", "internal/model/volume": "volume", "internal/model/iowire": "iowire",
		"internal/backends/native/compatibilitypack": "compatibility", "internal/model/strictjson": "strictjson", "internal/host/exec": "hostexec", "internal/host/fs": "hostfs", "internal/backends/native/sourceinventory": "sourceinventory",
		"internal/runner": "runner", "internal/evidence/qualification": "qualification", "internal/backends/target": "target", "internal/evidence/record": "record", "internal/evidence/artifact": "artifact", "internal/model/choice": "choice", "internal/model/world": "world", "internal/model/clock": "simulation", "internal/model/network": "simulation", "internal/model/schema": "developer", "internal/model/cluster": "simulation", "internal/model/runtimesource": "runtimesemantics",
	} {
		if Within(relative, path) {
			return owner
		}
	}
	return ""
}
func OwnerMayImport(modulePath, owner, importedOwner, importing, imported string) bool {
	if owner == "testfixture" {
		return slices.Contains([]string{
			modulePath + "/internal/backends", modulePath + "/internal/backends/target", modulePath + "/internal/evidence/artifact", modulePath + "/internal/evidence/record", modulePath + "/internal/evidence/record/testfixture", modulePath + "/internal/model/choice", modulePath + "/internal/model/iowire", modulePath + "/internal/backends/native", modulePath + "/internal/backends/native/build", modulePath + "/internal/backends/native/transcript",
		}, imported)
	}
	if owner == importedOwner {
		return true
	}
	if owner == "wasm" {
		if importing == modulePath+"/internal/backends/wasm/backend" && imported == modulePath+"/internal/host/fs" {
			return true
		}
		return slices.ContainsFunc([]string{
			"internal/evidence/artifact", "internal/model/choice", "internal/evidence/artifact/capture", "internal/model/volume", "internal/host/exec", "internal/host/fs",
			"internal/evidence/qualification", "internal/evidence/record", "internal/evidence/record/softassert", "internal/runner", "internal/runner/inspect", "internal/runner/options", "internal/backends",
			"internal/model/clock", "internal/model/network", "internal/model/cluster", "internal/backends/target", "internal/model/world", "internal/model/runtimesource", "internal/model/strictjson",
		}, func(path string) bool { return imported == modulePath+"/"+path })
	}
	if owner == "cli" && slices.Contains([]string{"wasm", "native", "hostexec", "hostfs"}, importedOwner) {
		return importing == modulePath+"/cmd/gomad/internal/cli" && slices.Contains([]string{modulePath + "/internal/backends/wasm/backend", modulePath + "/internal/backends/wasm/command", modulePath + "/internal/backends/wasm/installation", modulePath + "/internal/backends/wasm/qualification", modulePath + "/internal/backends/native", modulePath + "/internal/host/exec", modulePath + "/internal/host/fs"}, imported)
	}
	if owner == "runner" && imported == modulePath+"/internal/model/clock" {
		return true
	}
	if owner == "captured" && importedOwner == "hostfs" {
		return imported == modulePath+"/internal/host/fs"
	}
	allowed := map[string][]string{
		"native": {"runner", "target", "record", "artifact", "choice", "deterministicio", "captured", "volume", "iowire", "preparation", "world", "strictjson", "hostexec", "hostfs", "simulation"},
		"cli":    {"runner", "qualification", "target", "record", "artifact", "deterministicio", "captured", "volume", "iowire", "preparation", "toolchain", "strictjson"}, "developer": {"choice", "compatibility", "qualification", "simulation", "toolchain", "upgrade", "hostexec", "hostfs", "runtimesemantics"},
		"runner": {"target", "record", "artifact", "choice", "deterministicio", "captured", "volume", "iowire", "preparation", "world", "strictjson", "hostexec", "hostfs"}, "qualification": {"runner", "target", "record", "artifact", "choice", "deterministicio", "captured", "volume", "iowire", "preparation", "strictjson", "hostexec", "hostfs"},
		"target": {"compatibility", "record", "toolchain", "strictjson", "hostexec", "hostfs", "sourceinventory"}, "record": {"strictjson", "choice"}, "world": {"strictjson"}, "simulation": {"record", "strictjson"},
		"artifact": {"choice", "deterministicio", "captured", "volume", "iowire", "target", "record", "hostfs"}, "compatibility": {"target", "record", "strictjson", "hostfs"},
		"deterministicio": {"target", "record", "toolchain", "strictjson", "hostfs", "sourceinventory", "captured", "volume", "iowire"}, "captured": {"record", "strictjson", "iowire"}, "volume": {"iowire"},
		"preparation": {"target", "deterministicio", "captured", "volume", "iowire", "record"}, "toolchain": {"strictjson", "hostexec", "hostfs", "runtimesemantics"},
		"upgrade": {"qualification", "toolchain", "deterministicio", "captured", "volume", "iowire", "compatibility", "strictjson", "hostexec", "hostfs"}, "sourceinventory": {"hostfs"}, "runtimesemantics": {"hostfs"},
	}
	if !slices.Contains(allowed[owner], importedOwner) {
		return false
	}
	if (owner == "target" || owner == "deterministicio") && importedOwner == "toolchain" {
		return imported == modulePath+"/internal/backends/native/toolchain/version" || imported == modulePath+"/internal/backends/native/toolchain/installation"
	}
	if owner == "upgrade" && slices.Contains([]string{"compatibility", "deterministicio", "captured", "volume", "iowire"}, importedOwner) {
		return importing == modulePath+"/internal/backends/native/upgrade/pinimpact" || importing == modulePath+"/internal/backends/native/upgrade/adapterregen"
	}
	return true
}

func ModuleMayImport(modulePath, importing, imported string) bool {
	if Owner(modulePath, imported) == "testfixture" && Owner(modulePath, importing) != "testfixture" {
		return false
	}

	forbidden := map[string][]string{
		"internal/evidence/artifact": {"internal/runner"}, "internal/evidence/record": {"internal/runner"},
		"internal/runner/internal/campaign":    {"internal/runner/internal/corpus", "internal/runner/internal/minimizer"},
		"internal/runner/internal/corpus":      {"internal/runner/internal/campaign", "internal/runner/internal/exploration"},
		"internal/runner/internal/exploration": {"internal/runner/internal/logstore"},
	}
	// The shared log owner depends only on the standard library and hostfs, so
	// campaign and corpus can both build on it without importing each other.
	if Within(importing, modulePath+"/internal/runner/internal/logstore") {
		return imported == modulePath+"/internal/host/fs" || Within(imported, modulePath+"/internal/runner/internal/logstore")
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
		"internal/backends/native/build": {"internal/backends/native/build/internal/build", "internal/backends/native/build/internal/capabilityreview", "internal/backends/native/build/internal/provenance", "internal/backends/native/toolchain/installation", "internal/backends/native/sourceinventory"}, "internal/backends/native/upgrade": {"internal/backends/native/qualification/set", "internal/backends/native/toolchain/version"},
		"internal/backends/native/toolchain": {"internal/backends/native/toolchain/installation"}, "internal/backends/native/transcript": {"internal/backends/native/toolchain/installation", "internal/backends/native/sourceinventory"},
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
			if pkg.ImportPath == module+"/internal/backends/native/toolchain/installation" || (pkg.ImportPath == module+"/internal/backends/native/sourceinventory" && imported != module+"/internal/host/fs") || (pkg.ImportPath == module+"/internal/backends/native/toolchain" && (Within(imported, module+"/internal/evidence/qualification") || Within(imported, module+"/internal/backends/native/qualification") || Within(imported, module+"/internal/backends/native/upgrade"))) {
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
