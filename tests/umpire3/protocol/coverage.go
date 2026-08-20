package protocol

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const CoverageDenominatorFormatVersion = "umpire3/coverage-denominator/v1"

type CoverageEdge struct {
	Identifier     string `json:"identifier"`
	FromState      string `json:"fromState"`
	Action         string `json:"action"`
	ToState        string `json:"toState"`
	RequiresFault  bool   `json:"requiresFault"`
	StandaloneOnly bool   `json:"standaloneOnly"`
}

type CoverageTarget struct {
	Identifier TargetID       `json:"identifier"`
	Property   PropertyID     `json:"property"`
	Edges      []CoverageEdge `json:"edges"`
}

type CoverageDenominator struct {
	FormatVersion string           `json:"formatVersion"`
	SemanticHash  string           `json:"semanticHash"`
	CatalogHash   string           `json:"catalogHash"`
	Targets       []CoverageTarget `json:"targets"`
}

//go:embed generated/coverage-denominator.json
var defaultCoverageDenominatorJSON []byte

func DecodeCoverageDenominator(encoded []byte) (CoverageDenominator, error) {
	var denominator CoverageDenominator
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "coverage denominator", &denominator); err != nil {
		return CoverageDenominator{}, err
	}
	if err := denominator.Validate(); err != nil {
		return CoverageDenominator{}, err
	}
	return denominator, nil
}

func DefaultCoverageDenominator() (CoverageDenominator, error) {
	return DecodeCoverageDenominator(defaultCoverageDenominatorJSON)
}

func (d CoverageDenominator) Validate() error {
	if d.FormatVersion != CoverageDenominatorFormatVersion || !validHash(d.SemanticHash) ||
		!validHash(d.CatalogHash) || len(d.Targets) == 0 {
		return errors.New("complete coverage denominator provenance and targets are required")
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return err
	}
	if d.CatalogHash != catalogHash {
		return fmt.Errorf("coverage catalog hash %q does not match semantic catalog %q", d.CatalogHash, catalogHash)
	}
	knownTargets := make(map[TargetID]struct{}, len(catalog.Targets))
	for _, target := range catalog.Targets {
		knownTargets[TargetID(target.Identifier)] = struct{}{}
	}
	knownProperties := make(map[PropertyID]struct{}, len(catalog.Properties))
	for _, property := range catalog.Properties {
		knownProperties[PropertyID(property.Identifier)] = struct{}{}
	}
	targets := make(map[TargetID]struct{}, len(d.Targets))
	identifiers := make(map[string]struct{})
	for _, target := range d.Targets {
		if _, known := knownTargets[target.Identifier]; !known {
			return fmt.Errorf("coverage denominator has unknown target %q", target.Identifier)
		}
		if _, duplicate := targets[target.Identifier]; duplicate {
			return fmt.Errorf("coverage denominator has duplicate target %q", target.Identifier)
		}
		targets[target.Identifier] = struct{}{}
		if _, known := knownProperties[target.Property]; !known {
			return fmt.Errorf("coverage denominator target %q has unknown property %q", target.Identifier, target.Property)
		}
		if len(target.Edges) == 0 {
			return fmt.Errorf("coverage denominator target %q has no edges", target.Identifier)
		}
		for _, edge := range target.Edges {
			if edge.Identifier == "" || edge.FromState == "" || edge.Action == "" || edge.ToState == "" {
				return fmt.Errorf("coverage denominator target %q has incomplete edge", target.Identifier)
			}
			if _, duplicate := identifiers[edge.Identifier]; duplicate {
				return fmt.Errorf("coverage denominator has duplicate edge %q", edge.Identifier)
			}
			identifiers[edge.Identifier] = struct{}{}
		}
	}
	return nil
}

func (d CoverageDenominator) CanonicalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	copy := d
	copy.Targets = append([]CoverageTarget(nil), d.Targets...)
	slices.SortFunc(copy.Targets, func(left, right CoverageTarget) int {
		return compareStrings(string(left.Identifier), string(right.Identifier))
	})
	for index := range copy.Targets {
		copy.Targets[index].Edges = append([]CoverageEdge(nil), copy.Targets[index].Edges...)
		slices.SortFunc(copy.Targets[index].Edges, func(left, right CoverageEdge) int {
			return compareStrings(left.Identifier, right.Identifier)
		})
	}
	return json.Marshal(copy)
}
