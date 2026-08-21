package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tests/umpire3/protocol"
)

func exportFamilyDependencies(modelRoot string, writer io.Writer) error {
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return err
	}
	coverage, err := protocol.DefaultCheckerCoverage()
	if err != nil {
		return err
	}
	tests, err := familyLeanTests(modelRoot)
	if err != nil {
		return err
	}
	temporal, temporalFound, err := protocol.DefaultTemporalView("sound")
	if err != nil {
		return err
	}

	graph := protocol.FamilyDependencyGraph{
		FormatVersion: protocol.FamilyDependencyGraphFormatVersion,
		CatalogHash:   catalogHash,
		Families:      make([]protocol.FamilyDependency, 0, len(catalog.Targets)),
	}
	for _, target := range catalog.Targets {
		directSources := make(map[string]struct{}, len(target.Modules))
		buildModules := make(map[string]struct{}, len(target.Modules))
		sources := map[string]struct{}{
			"Temporal/Catalog.lean":     {},
			"Temporal/Composition.lean": {},
		}
		for _, module := range target.Modules {
			source, err := familyModuleSource(modelRoot, module)
			if err != nil {
				return fmt.Errorf("resolve family %q module %q: %w", target.Identifier, module, err)
			}
			directSources[source] = struct{}{}
			buildModules[strings.TrimSuffix(strings.ReplaceAll(source, "/", "."), ".lean")] = struct{}{}
			dependencies, err := resolveSourceDependencies(modelRoot, source, nil)
			if err != nil {
				return fmt.Errorf("resolve family %q module %q: %w", target.Identifier, module, err)
			}
			for _, dependency := range dependencies {
				sources[dependency.Path] = struct{}{}
			}
		}
		leanTests := make([]string, 0)
		for _, test := range tests {
			if intersectsSources(test.dependencies, directSources) {
				leanTests = append(leanTests, test.module)
			}
		}
		checkers := []string{"exact"}
		for _, entry := range coverage.Entries {
			if entry.Target != protocol.TargetID(target.Identifier) || entry.Status != protocol.CheckerCoverageChecked {
				continue
			}
			switch entry.Checker {
			case protocol.CheckerNative, protocol.CheckerVeil:
				checkers = append(checkers, string(entry.Checker))
			}
		}
		if temporalFound && temporal.Target == protocol.TargetID(target.Identifier) {
			checkers = append(checkers, "lean-temporal")
		}
		slices.Sort(checkers)
		checkers = slices.Compact(checkers)
		graph.Families = append(graph.Families, protocol.FamilyDependency{
			Target: protocol.TargetID(target.Identifier), Modules: slices.Clone(target.Modules),
			BuildModules: mapKeysSorted(buildModules), Sources: mapKeysSorted(sources),
			LeanTests: leanTests, Checkers: checkers,
		})
	}
	slices.SortFunc(graph.Families, func(left, right protocol.FamilyDependency) int {
		return strings.Compare(string(left.Target), string(right.Target))
	})
	encoded, err := graph.CanonicalJSON(catalog)
	if err != nil {
		return err
	}
	if _, err := writer.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("write family dependency graph: %w", err)
	}
	return nil
}

func familyModuleSource(modelRoot, module string) (string, error) {
	parts := strings.Split(module, ".")
	for len(parts) > 0 {
		path := strings.Join(parts, "/") + ".lean"
		info, err := os.Stat(filepath.Join(modelRoot, filepath.FromSlash(path)))
		if err == nil && !info.IsDir() {
			return path, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		parts = parts[:len(parts)-1]
	}
	return "", fmt.Errorf("no Lean source declares module %q", module)
}

type familyLeanTest struct {
	module       string
	dependencies map[string]struct{}
}

func familyLeanTests(modelRoot string) ([]familyLeanTest, error) {
	paths, err := filepath.Glob(filepath.Join(modelRoot, "Umpire3Tests", "*.lean"))
	if err != nil {
		return nil, err
	}
	result := make([]familyLeanTest, 0, len(paths))
	for _, path := range paths {
		relative, err := filepath.Rel(modelRoot, path)
		if err != nil {
			return nil, err
		}
		relative = filepath.ToSlash(relative)
		dependencies, err := resolveSourceDependencies(modelRoot, relative, nil)
		if err != nil {
			return nil, err
		}
		dependencySet := make(map[string]struct{}, len(dependencies))
		for _, dependency := range dependencies {
			dependencySet[dependency.Path] = struct{}{}
		}
		result = append(result, familyLeanTest{
			module:       strings.TrimSuffix(strings.ReplaceAll(relative, "/", "."), ".lean"),
			dependencies: dependencySet,
		})
	}
	slices.SortFunc(result, func(left, right familyLeanTest) int {
		return strings.Compare(left.module, right.module)
	})
	return result, nil
}

func intersectsSources(left, right map[string]struct{}) bool {
	for source := range left {
		if _, found := right[source]; found {
			return true
		}
	}
	return false
}

func mapKeysSorted(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}
