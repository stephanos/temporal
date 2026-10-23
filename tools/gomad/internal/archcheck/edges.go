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
	if relative == "cmd/gomad" {
		return "cli"
	}
	if Within(relative, "internal/model/capture") {
		return "captured"
	}
	for path, owner := range map[string]string{
		"internal/backends/wasm": "wasm", "cmd/gomad/internal/cli": "cli", "internal/archcheck": "developer",
		"internal/model/wasi": "wasireference", "internal/model/softassert": "record", "internal/support/temporalinput": "developer",
		"internal/model/generation": "developer", "internal/model/cmd": "developer",
		"internal/model/capture": "captured", "internal/model/volume": "volume", "internal/model/iowire": "iowire",
		"internal/model/strictjson": "strictjson", "internal/host/exec": "hostexec", "internal/support/fs": "hostfs",
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
			modulePath + "/internal/backends", modulePath + "/internal/backends/target", modulePath + "/internal/evidence/artifact", modulePath + "/internal/evidence/record", modulePath + "/internal/evidence/record/testfixture", modulePath + "/internal/model/choice", modulePath + "/internal/model/iowire",
		}, imported)
	}
	if owner == importedOwner {
		return true
	}
	if owner == "wasm" {
		if importing == modulePath+"/internal/backends/wasm/backend" && imported == modulePath+"/internal/support/fs" {
			return true
		}
		return slices.ContainsFunc([]string{
			"internal/evidence/artifact", "internal/model/choice", "internal/model/capture", "internal/model/volume", "internal/host/exec", "internal/support/fs",
			"internal/evidence/qualification", "internal/evidence/record", "internal/model/softassert", "internal/runner", "internal/runner/inspect", "internal/runner/options", "internal/backends",
			"internal/model/clock", "internal/model/network", "internal/model/cluster", "internal/backends/target", "internal/model/world", "internal/model/runtimesource", "internal/model/strictjson",
		}, func(path string) bool { return imported == modulePath+"/"+path })
	}
	if owner == "cli" && slices.Contains([]string{"wasm", "hostexec", "hostfs"}, importedOwner) {
		return importing == modulePath+"/cmd/gomad/internal/cli" && slices.Contains([]string{modulePath + "/internal/backends/wasm/backend", modulePath + "/internal/backends/wasm/command", modulePath + "/internal/backends/wasm/installation", modulePath + "/internal/backends/wasm/qualification", modulePath + "/internal/host/exec", modulePath + "/internal/support/fs"}, imported)
	}
	if owner == "runner" && imported == modulePath+"/internal/model/clock" {
		return true
	}
	if importing == modulePath+"/internal/model/cmd/sqlite-native" && imported == modulePath+"/internal/model/strictjson" {
		return true
	}
	if owner == "captured" && importedOwner == "hostfs" {
		return imported == modulePath+"/internal/support/fs"
	}
	allowed := map[string][]string{
		"cli": {"runner", "qualification", "target", "record", "artifact", "captured", "volume", "iowire", "strictjson"}, "wasireference": {"captured", "choice", "simulation", "record", "strictjson", "volume", "world", "developer"},
		"developer": {"wasireference", "captured", "choice", "qualification", "simulation", "hostexec", "hostfs", "runtimesemantics"},
		"runner":    {"target", "record", "artifact", "choice", "captured", "volume", "iowire", "world", "strictjson", "hostexec", "hostfs"}, "qualification": {"runner", "target", "record", "artifact", "choice", "captured", "volume", "iowire", "strictjson", "hostexec", "hostfs"},
		"target": {"record", "strictjson", "hostexec", "hostfs"}, "record": {"strictjson", "choice"}, "world": {"strictjson"}, "simulation": {"record", "strictjson"},
		"artifact": {"choice", "captured", "volume", "iowire", "target", "record", "hostfs"}, "captured": {"record", "strictjson", "iowire"}, "volume": {"iowire"},
		"runtimesemantics": {"hostfs"},
	}
	if !slices.Contains(allowed[owner], importedOwner) {
		return false
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
		return imported == modulePath+"/internal/support/fs" || Within(imported, modulePath+"/internal/runner/internal/logstore")
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
	for _, pkg := range packages {
		owner := Owner(module, pkg.ImportPath)
		if owner == "" {
			add("ownerless", pkg.ImportPath, pkg.ImportPath)
		}
		for _, imported := range pkg.Imports {
			if !Within(imported, module) {
				continue
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
