package protocol

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const CoverageDenominatorFormatVersion = "umpire3/coverage-denominator/v2"

type CoverageDenominatorStatus string

const (
	CoverageDenominatorDefined   CoverageDenominatorStatus = "coverage-defined"
	CoverageDenominatorUndefined CoverageDenominatorStatus = "coverage-undefined"
)

type CoverageEdge struct {
	Identifier     string `json:"identifier"`
	FromState      string `json:"fromState"`
	Action         string `json:"action"`
	ToState        string `json:"toState"`
	RequiresFault  bool   `json:"requiresFault"`
	StandaloneOnly bool   `json:"standaloneOnly"`
}

type CoverageTarget struct {
	Identifier TargetID                  `json:"identifier"`
	Property   PropertyID                `json:"property"`
	Status     CoverageDenominatorStatus `json:"status"`
	Reason     string                    `json:"reason,omitempty"`
	Edges      []CoverageEdge            `json:"edges"`
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
	type targetProperty struct {
		target   TargetID
		property PropertyID
	}
	knownTargetProperties := make(map[targetProperty]struct{})
	for _, target := range catalog.Targets {
		for _, property := range target.Properties {
			knownTargetProperties[targetProperty{
				target: TargetID(target.Identifier), property: PropertyID(property),
			}] = struct{}{}
		}
	}
	targets := make(map[targetProperty]struct{}, len(d.Targets))
	identifiers := make(map[string]struct{})
	for _, target := range d.Targets {
		key := targetProperty{target: target.Identifier, property: target.Property}
		if _, known := knownTargetProperties[key]; !known {
			return fmt.Errorf("coverage denominator has unknown target/property %q/%q", target.Identifier, target.Property)
		}
		if _, duplicate := targets[key]; duplicate {
			return fmt.Errorf("coverage denominator has duplicate target/property %q/%q", target.Identifier, target.Property)
		}
		targets[key] = struct{}{}
		switch target.Status {
		case CoverageDenominatorDefined:
			if target.Reason != "" || len(target.Edges) == 0 {
				return fmt.Errorf("defined coverage denominator target %q property %q requires edges and no reason",
					target.Identifier, target.Property)
			}
		case CoverageDenominatorUndefined:
			if target.Reason == "" || len(target.Edges) != 0 {
				return fmt.Errorf("undefined coverage denominator target %q property %q requires a reason and no edges",
					target.Identifier, target.Property)
			}
		default:
			return fmt.Errorf("coverage denominator target %q property %q has unknown status %q",
				target.Identifier, target.Property, target.Status)
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
	if len(targets) != len(knownTargetProperties) {
		return fmt.Errorf("coverage denominator classifies %d target/property pairs; catalog requires %d",
			len(targets), len(knownTargetProperties))
	}
	return nil
}

func (d CoverageDenominator) CanonicalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	canonical := d
	canonical.Targets = append([]CoverageTarget(nil), d.Targets...)
	slices.SortFunc(canonical.Targets, func(left, right CoverageTarget) int {
		if comparison := compareStrings(string(left.Identifier), string(right.Identifier)); comparison != 0 {
			return comparison
		}
		return compareStrings(string(left.Property), string(right.Property))
	})
	for index := range canonical.Targets {
		canonical.Targets[index].Edges = append([]CoverageEdge(nil), canonical.Targets[index].Edges...)
		slices.SortFunc(canonical.Targets[index].Edges, func(left, right CoverageEdge) int {
			return compareStrings(left.Identifier, right.Identifier)
		})
	}
	return json.Marshal(canonical)
}
