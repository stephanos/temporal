package protocol

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

const FamilyDependencyGraphFormatVersion = "umpire3/family-dependencies/v1"

type FamilyDependency struct {
	Target       TargetID `json:"target"`
	Modules      []string `json:"modules"`
	BuildModules []string `json:"buildModules"`
	Sources      []string `json:"sources"`
	LeanTests    []string `json:"leanTests"`
	Checkers     []string `json:"checkers"`
}

type FamilyDependencyGraph struct {
	FormatVersion string             `json:"formatVersion"`
	CatalogHash   string             `json:"catalogHash"`
	Families      []FamilyDependency `json:"families"`
}

func (g FamilyDependencyGraph) Validate(catalog Catalog) error {
	if g.FormatVersion != FamilyDependencyGraphFormatVersion || !validHash(g.CatalogHash) {
		return errors.New("family dependency graph identity is invalid")
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return err
	}
	if g.CatalogHash != catalogHash {
		return errors.New("family dependency graph catalog hash mismatch")
	}
	if len(g.Families) != len(catalog.Targets) {
		return errors.New("family dependency graph must cover every catalog target")
	}
	if !slices.IsSortedFunc(g.Families, func(left, right FamilyDependency) int {
		return strings.Compare(string(left.Target), string(right.Target))
	}) {
		return errors.New("family dependencies must be sorted by target")
	}
	seen := make(map[TargetID]struct{}, len(g.Families))
	for _, family := range g.Families {
		if _, duplicate := seen[family.Target]; duplicate {
			return fmt.Errorf("duplicate family dependency target %q", family.Target)
		}
		seen[family.Target] = struct{}{}
		var target TargetDeclaration
		found := false
		for _, candidate := range catalog.Targets {
			if candidate.Identifier == string(family.Target) {
				target = candidate
				found = true
				break
			}
		}
		if !found || !slices.Equal(family.Modules, target.Modules) {
			return fmt.Errorf("family dependency target %q modules do not match the catalog", family.Target)
		}
		if err := validateSortedStrings("sources", family.Sources, true); err != nil {
			return fmt.Errorf("family dependency target %q: %w", family.Target, err)
		}
		if err := validateSortedStrings("build modules", family.BuildModules, true); err != nil {
			return fmt.Errorf("family dependency target %q: %w", family.Target, err)
		}
		if err := validateSortedStrings("Lean tests", family.LeanTests, false); err != nil {
			return fmt.Errorf("family dependency target %q: %w", family.Target, err)
		}
		if err := validateSortedStrings("checkers", family.Checkers, true); err != nil {
			return fmt.Errorf("family dependency target %q: %w", family.Target, err)
		}
		if !slices.Contains(family.Checkers, "exact") {
			return fmt.Errorf("family dependency target %q requires the exact checker", family.Target)
		}
	}
	return nil
}

func validateSortedStrings(kind string, values []string, required bool) error {
	if required && len(values) == 0 {
		return fmt.Errorf("%s are required", kind)
	}
	if !slices.IsSorted(values) || len(slices.Compact(slices.Clone(values))) != len(values) {
		return fmt.Errorf("%s must be sorted and unique", kind)
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s contain an empty value", kind)
		}
	}
	return nil
}

func (g FamilyDependencyGraph) Family(target TargetID) (FamilyDependency, bool) {
	for _, family := range g.Families {
		if family.Target == target {
			return family, true
		}
	}
	return FamilyDependency{}, false
}

func (g FamilyDependencyGraph) AffectedTargets(paths []string) []TargetID {
	changed := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		path = strings.TrimPrefix(strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./"),
			"tests/umpire3/model/")
		changed[path] = struct{}{}
	}
	var result []TargetID
	for _, family := range g.Families {
		affected := false
		for _, source := range append(slices.Clone(family.Sources), family.LeanTests...) {
			if _, found := changed[source]; found {
				affected = true
				break
			}
		}
		if affected {
			result = append(result, family.Target)
		}
	}
	return result
}

func (g FamilyDependencyGraph) CanonicalJSON(catalog Catalog) ([]byte, error) {
	if err := g.Validate(catalog); err != nil {
		return nil, err
	}
	return json.Marshal(g)
}

func DecodeFamilyDependencyGraph(encoded []byte, catalog Catalog) (FamilyDependencyGraph, error) {
	if int64(len(encoded)) > DefaultDecodeLimit {
		return FamilyDependencyGraph{}, fmt.Errorf("family dependency graph exceeds %d bytes", DefaultDecodeLimit)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var graph FamilyDependencyGraph
	if err := decoder.Decode(&graph); err != nil {
		return FamilyDependencyGraph{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return FamilyDependencyGraph{}, errors.New("family dependency graph must contain exactly one JSON value")
	}
	if err := graph.Validate(catalog); err != nil {
		return FamilyDependencyGraph{}, err
	}
	return graph, nil
}

//go:embed generated/family-dependencies.json
var defaultFamilyDependencyGraphJSON []byte

func DefaultFamilyDependencyGraph() (FamilyDependencyGraph, error) {
	catalog, err := DefaultCatalog()
	if err != nil {
		return FamilyDependencyGraph{}, err
	}
	return DecodeFamilyDependencyGraph(defaultFamilyDependencyGraphJSON, catalog)
}
