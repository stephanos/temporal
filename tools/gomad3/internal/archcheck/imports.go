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

type Config struct {
	Module      string
	PureImports []string
	Owners      map[string][]string
	OwnerEdges  map[string][]string
	ModuleEdges []ModuleEdge
	PureFiles   func(string) bool
}

type ModuleEdge struct{ From, To string }

func (config Config) CheckImports(packages []Package) []string {
	findings := map[string]bool{}
	for _, pkg := range packages {
		owner := config.backend(pkg.ImportPath)
		for _, imported := range pkg.Imports {
			importedOwner := config.backend(imported)
			if !within(pkg.ImportPath, config.Module) || owner == "cmd" || importedOwner == "" || owner == importedOwner {
				continue
			}
			findings[fmt.Sprintf("forbidden backend import %s -> %s", pkg.ImportPath, imported)] = true
		}
	}
	return slices.Sorted(maps.Keys(findings))
}

func (config Config) backend(path string) string {
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
	return ""
}

func within(path, root string) bool {
	return root != "" && (path == root || strings.HasPrefix(path, root+"/"))
}
