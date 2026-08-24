package generate

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tools/umpire3/protocol/checker"
)

func exportFamilyDependencies(modelRoot string, writer io.Writer) error {
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return err
	}
	coverage, err := protocolchecker.DefaultCheckerCoverage()
	if err != nil {
		return err
	}
	ownership, err := readFamilyOwnership(modelRoot, catalog)
	if err != nil {
		return err
	}
	moduleSources, err := readFamilyModuleSources(modelRoot, catalog)
	if err != nil {
		return err
	}
	temporal, temporalFound, err := protocolchecker.DefaultTemporalView("sound")
	if err != nil {
		return err
	}

	graph := protocolcatalog.FamilyDependencyGraph{
		FormatVersion: protocolcatalog.FamilyDependencyGraphFormatVersion,
		CatalogHash:   catalogHash,
		Families:      make([]protocolcatalog.FamilyDependency, 0, len(catalog.Targets)),
	}
	for _, target := range catalog.Targets {
		buildModules := make(map[string]struct{}, len(target.Modules))
		sources := map[string]struct{}{
			"Temporal/Catalog.lean":           {},
			"Temporal/Composition.lean":       {},
			"Temporal/Families/modules.tsv":   {},
			"Temporal/Families/ownership.tsv": {},
		}
		for _, module := range target.Modules {
			source := moduleSources[module]
			buildModules[strings.TrimSuffix(strings.ReplaceAll(source, "/", "."), ".lean")] = struct{}{}
			dependencies, err := resolveSourceDependencies(modelRoot, source, nil)
			if err != nil {
				return fmt.Errorf("resolve family %q module %q: %w", target.Identifier, module, err)
			}
			for _, dependency := range dependencies {
				sources[dependency.Path] = struct{}{}
			}
		}
		leanTests := slices.Clone(ownership[protocolcatalog.TargetID(target.Identifier)])
		checkers := []string{"exact"}
		for _, entry := range coverage.Entries {
			if entry.Target != protocolcatalog.TargetID(target.Identifier) || entry.Status != protocolchecker.CheckerCoverageChecked {
				continue
			}
			switch entry.Checker {
			case protocolchecker.CheckerNative, protocolchecker.CheckerVeil:
				checkers = append(checkers, string(entry.Checker))
			default:
			}
		}
		if temporalFound && temporal.Target == protocolcatalog.TargetID(target.Identifier) {
			checkers = append(checkers, "lean-temporal")
		}
		slices.Sort(checkers)
		checkers = slices.Compact(checkers)
		graph.Families = append(graph.Families, protocolcatalog.FamilyDependency{
			Target: protocolcatalog.TargetID(target.Identifier), Modules: slices.Clone(target.Modules),
			BuildModules: mapKeysSorted(buildModules), Sources: mapKeysSorted(sources),
			LeanTests: leanTests, Checkers: checkers,
		})
	}
	slices.SortFunc(graph.Families, func(left, right protocolcatalog.FamilyDependency) int {
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

const familyOwnershipPath = "Temporal/Families/ownership.tsv"
const familyModuleOwnershipPath = "Temporal/Families/modules.tsv"

func readFamilyModuleSources(modelRoot string, catalog protocolcatalog.Catalog) (map[string]string, error) {
	file, err := os.Open(filepath.Join(modelRoot, filepath.FromSlash(familyModuleOwnershipPath)))
	if err != nil {
		return nil, fmt.Errorf("open family module manifest: %w", err)
	}
	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = 2
	records, err := reader.ReadAll()
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		return nil, fmt.Errorf("decode family module manifest: %w", errors.Join(err, closeErr))
	}
	if len(records) == 0 || !slices.Equal(records[0], []string{"module", "source"}) {
		return nil, errors.New("family module manifest requires module and source columns")
	}
	required := make(map[string]struct{})
	for _, target := range catalog.Targets {
		for _, module := range target.Modules {
			required[module] = struct{}{}
		}
	}
	result := make(map[string]string, len(required))
	previous := ""
	for _, record := range records[1:] {
		module, source := record[0], filepath.ToSlash(record[1])
		if previous != "" && strings.Compare(previous, module) >= 0 {
			return nil, errors.New("family modules must be sorted and unique")
		}
		previous = module
		if _, exists := required[module]; !exists {
			return nil, fmt.Errorf("family module %q is not used by a catalog target", module)
		}
		if strings.TrimSpace(source) != source || filepath.IsAbs(source) || filepath.Ext(source) != ".lean" {
			return nil, fmt.Errorf("family module %q has invalid source %q", module, source)
		}
		info, statErr := os.Stat(filepath.Join(modelRoot, filepath.FromSlash(source)))
		if statErr != nil {
			return nil, fmt.Errorf("family module %q source %q: %w", module, source, statErr)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("family module %q source %q is not a file", module, source)
		}
		result[module] = source
	}
	if len(result) != len(required) {
		return nil, errors.New("family module manifest must cover every catalog target module")
	}
	return result, nil
}

func readFamilyOwnership(modelRoot string, catalog protocolcatalog.Catalog) (map[protocolcatalog.TargetID][]string, error) {
	file, err := os.Open(filepath.Join(modelRoot, filepath.FromSlash(familyOwnershipPath)))
	if err != nil {
		return nil, fmt.Errorf("open family ownership manifest: %w", err)
	}

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = 2
	records, err := reader.ReadAll()
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return nil, fmt.Errorf("decode family ownership manifest: %v; close: %w", err, closeErr)
		}
		return nil, fmt.Errorf("decode family ownership manifest: %w", err)
	}
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("close family ownership manifest: %w", err)
	}
	if len(records) == 0 || !slices.Equal(records[0], []string{"target", "lean-tests"}) {
		return nil, errors.New("family ownership manifest requires target and lean-tests columns")
	}

	catalogTargets := make(map[protocolcatalog.TargetID]struct{}, len(catalog.Targets))
	for _, target := range catalog.Targets {
		catalogTargets[protocolcatalog.TargetID(target.Identifier)] = struct{}{}
	}
	ownership := make(map[protocolcatalog.TargetID][]string, len(catalog.Targets))
	referencedTests := make(map[string]struct{})
	previousTarget := ""
	for _, record := range records[1:] {
		target := protocolcatalog.TargetID(record[0])
		if _, found := catalogTargets[target]; !found {
			return nil, fmt.Errorf("family ownership target %q is not in the catalog", target)
		}
		if _, duplicate := ownership[target]; duplicate {
			return nil, fmt.Errorf("family ownership target %q is duplicated", target)
		}
		if previousTarget != "" && strings.Compare(previousTarget, string(target)) >= 0 {
			return nil, errors.New("family ownership targets must be sorted and unique")
		}
		previousTarget = string(target)
		leanTests := strings.Split(record[1], ",")
		if err := validateOwnedLeanTests(modelRoot, leanTests); err != nil {
			return nil, fmt.Errorf("family ownership target %q: %w", target, err)
		}
		ownership[target] = leanTests
		for _, module := range leanTests {
			referencedTests[module] = struct{}{}
		}
	}
	if len(ownership) != len(catalogTargets) {
		return nil, errors.New("family ownership manifest must cover every catalog target")
	}
	if err := validateAllFamilyTestsOwned(modelRoot, referencedTests); err != nil {
		return nil, err
	}
	return ownership, nil
}

func validateOwnedLeanTests(modelRoot string, modules []string) error {
	if len(modules) == 0 || !slices.IsSorted(modules) ||
		len(slices.Compact(slices.Clone(modules))) != len(modules) {
		return errors.New("lean tests must be non-empty, sorted, and unique")
	}
	for _, module := range modules {
		if strings.TrimSpace(module) != module || !strings.HasPrefix(module, "Umpire3Tests.Families.") {
			return fmt.Errorf("lean test %q is outside Umpire3Tests.Families", module)
		}
		path := filepath.Join(modelRoot, filepath.FromSlash(strings.ReplaceAll(module, ".", "/")+".lean"))
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("lean test %q does not exist", module)
			}
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("lean test %q is not a file", module)
		}
	}
	return nil
}

func validateAllFamilyTestsOwned(modelRoot string, referenced map[string]struct{}) error {
	root := filepath.Join(modelRoot, "Umpire3Tests", "Families")
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".lean" {
			return nil
		}
		relative, err := filepath.Rel(modelRoot, path)
		if err != nil {
			return err
		}
		module := strings.TrimSuffix(strings.ReplaceAll(filepath.ToSlash(relative), "/", "."), ".lean")
		if _, found := referenced[module]; !found {
			return fmt.Errorf("family Lean test %q has no declared owner", module)
		}
		return nil
	})
}

func mapKeysSorted(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	slices.Sort(result)
	return result
}
