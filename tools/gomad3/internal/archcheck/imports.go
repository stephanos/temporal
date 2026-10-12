package archcheck

import (
	"fmt"
	"slices"
	"strings"
)

type Package struct {
	ImportPath string
	Imports    []string
}

type Exception struct {
	From, To, RemovalTask string
}

type Config struct {
	Module, WASMModule string
	NativePackages     []string
	Exceptions         []Exception
	WASMImports        []string
}

func (config Config) CheckImports(packages []Package) []string {
	type edge struct{ from, to string }
	findings := map[string]bool{}
	add := func(format string, values ...any) { findings[fmt.Sprintf(format, values...)] = true }
	exceptions := map[edge]string{}
	used := map[edge]bool{}
	for _, exception := range config.Exceptions {
		key := edge{exception.From, exception.To}
		if !exactPackage(exception.From) || !exactPackage(exception.To) {
			add("exception %s -> %s must name exact packages", exception.From, exception.To)
			continue
		}
		if exception.RemovalTask == "" {
			add("exception %s -> %s needs a removal task", exception.From, exception.To)
			continue
		}
		if _, exists := exceptions[key]; exists {
			add("duplicate exception %s -> %s", exception.From, exception.To)
		}
		exceptions[key] = exception.RemovalTask
	}
	wasmAllowed := map[string]bool{}
	wasmUsed := map[string]bool{}
	for _, imported := range config.WASMImports {
		if !exactPackage(imported) {
			add("WASM import entry %s must name an exact package", imported)
			continue
		}
		if !within(imported, config.Module) || strings.Contains(imported, "/internal/") || strings.HasSuffix(imported, "/internal") {
			add("WASM import entry %s must name a public Gomad package", imported)
			continue
		}
		if wasmAllowed[imported] {
			add("duplicate WASM import entry %s", imported)
		}
		wasmAllowed[imported] = true
	}
	for _, pkg := range packages {
		if !within(pkg.ImportPath, config.Module) && !within(pkg.ImportPath, config.WASMModule) {
			continue
		}
		owner := config.backend(pkg.ImportPath)
		for _, imported := range pkg.Imports {
			if !within(imported, config.Module) && !within(imported, config.WASMModule) {
				continue
			}
			if within(pkg.ImportPath, config.WASMModule) && within(imported, config.Module) {
				if !wasmAllowed[imported] {
					add("WASM package %s imports %s outside its public import allowlist", pkg.ImportPath, imported)
				} else {
					wasmUsed[imported] = true
				}
			}
			importedOwner := config.backend(imported)
			if owner == "cmd" || importedOwner == "" || owner == importedOwner {
				continue
			}
			key := edge{pkg.ImportPath, imported}
			if _, allowed := exceptions[key]; allowed {
				used[key] = true
			} else {
				add("forbidden backend import %s -> %s", pkg.ImportPath, imported)
			}
		}
	}
	for key, removalTask := range exceptions {
		if !used[key] {
			add("stale exception %s -> %s (remove in %s)", key.from, key.to, removalTask)
		}
	}
	for imported := range wasmAllowed {
		if !wasmUsed[imported] {
			add("stale WASM import entry %s", imported)
		}
	}
	result := make([]string, 0, len(findings))
	for finding := range findings {
		result = append(result, finding)
	}
	slices.Sort(result)
	return result
}

func (config Config) backend(path string) string {
	if within(path, config.WASMModule) {
		if within(strings.TrimPrefix(path, config.WASMModule+"/"), "cmd") {
			return "cmd"
		}
		return "backend:wasm"
	}
	if !within(path, config.Module) {
		return ""
	}
	relative := strings.TrimPrefix(path, config.Module+"/")
	if within(relative, "cmd") {
		return "cmd"
	}
	if backend, rest, found := strings.Cut(relative, "/"); backend == "backends" && found {
		name, _, _ := strings.Cut(rest, "/")
		return "backend:" + name
	}
	for _, native := range config.NativePackages {
		if relative == native {
			return "backend:native"
		}
	}
	return ""
}

func within(path, root string) bool {
	return root != "" && (path == root || strings.HasPrefix(path, root+"/"))
}

func exactPackage(path string) bool {
	return path != "" && !strings.ContainsAny(path, "*?[]") && !strings.Contains(path, "...")
}
