package archcheck

import (
	"fmt"
	"maps"
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
	Module, WASMModule                       string
	NativePackages, WASMImports, PureImports []string
	Exceptions                               []Exception
	Owners                                   map[string][]string
	OwnerEdges                               map[string][]string
	ModuleEdges                              []ModuleEdge
	PureFiles                                func(string) bool
}

type ModuleEdge struct{ From, To string }

func (config Config) CheckImports(packages []Package) []string {
	type edge struct{ from, to string }
	findings := map[string]bool{}
	add := func(format string, values ...any) { findings[fmt.Sprintf(format, values...)] = true }
	exceptions := map[edge]string{}
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
	for _, imported := range config.WASMImports {
		if !exactPackage(imported) {
			add("WASM import entry %s must name an exact package", imported)
			continue
		}
		if !within(imported, config.Module) || strings.Contains(imported, "/internal/") || strings.HasSuffix(imported, "/internal") {
			add("WASM import entry %s must name a public Gomad package", imported)
			continue
		}
		if _, exists := wasmAllowed[imported]; exists {
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
				if _, allowed := wasmAllowed[imported]; !allowed {
					add("WASM package %s imports %s outside its public import allowlist", pkg.ImportPath, imported)
				} else {
					wasmAllowed[imported] = false
				}
			}
			importedOwner := config.backend(imported)
			if owner == "cmd" || importedOwner == "" || owner == importedOwner {
				continue
			}
			key := edge{pkg.ImportPath, imported}
			if _, allowed := exceptions[key]; allowed {
				exceptions[key] = ""
			} else {
				add("forbidden backend import %s -> %s", pkg.ImportPath, imported)
			}
		}
	}
	for key, removalTask := range exceptions {
		if removalTask != "" {
			add("stale exception %s -> %s (remove in %s)", key.from, key.to, removalTask)
		}
	}
	for imported, unused := range wasmAllowed {
		if unused {
			add("stale WASM import entry %s", imported)
		}
	}
	return slices.Sorted(maps.Keys(findings))
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
	if slices.Contains(config.NativePackages, relative) {
		return "backend:native"
	}
	return ""
}

func within(path, root string) bool {
	return root != "" && (path == root || strings.HasPrefix(path, root+"/"))
}

func exactPackage(path string) bool {
	return path != "" && !strings.ContainsAny(path, "*?[]") && !strings.Contains(path, "...")
}
