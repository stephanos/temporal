package protocol

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

const CoverageDenominatorFormatVersion = "umpire3/coverage-denominator/v3"

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

type CoverageDimension string

const (
	CoverageTransition  CoverageDimension = "transition"
	CoverageRelation    CoverageDimension = "relation"
	CoverageProperty    CoverageDimension = "property"
	CoverageFault       CoverageDimension = "fault"
	CoverageObservation CoverageDimension = "observation"
	CoverageRefinement  CoverageDimension = "refinement"
)

type ModelCoveragePoint struct {
	Dimension  CoverageDimension `json:"dimension"`
	Identifier string            `json:"identifier"`
	Source     string            `json:"source"`
}

type CoverageTarget struct {
	Identifier TargetID                  `json:"identifier"`
	Property   PropertyID                `json:"property"`
	Status     CoverageDenominatorStatus `json:"status"`
	Reason     string                    `json:"reason,omitempty"`
	Points     []ModelCoveragePoint      `json:"points"`
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
			if target.Reason != "" || len(target.Points) == 0 {
				return fmt.Errorf("defined coverage denominator target %q property %q requires points and no reason",
					target.Identifier, target.Property)
			}
		case CoverageDenominatorUndefined:
			if target.Reason == "" || len(target.Points) != 0 || len(target.Edges) != 0 {
				return fmt.Errorf("undefined coverage denominator target %q property %q requires a reason and no coverage",
					target.Identifier, target.Property)
			}
		default:
			return fmt.Errorf("coverage denominator target %q property %q has unknown status %q",
				target.Identifier, target.Property, target.Status)
		}
		pointIdentifiers := make(map[string]struct{}, len(target.Points))
		dimensions := make(map[CoverageDimension]struct{})
		for _, point := range target.Points {
			if !point.Dimension.valid() || point.Identifier == "" || point.Source == "" {
				return fmt.Errorf("coverage denominator target %q has incomplete model-derived point", target.Identifier)
			}
			if _, duplicate := pointIdentifiers[point.Identifier]; duplicate {
				return fmt.Errorf("coverage denominator target %q has duplicate point %q", target.Identifier, point.Identifier)
			}
			pointIdentifiers[point.Identifier] = struct{}{}
			dimensions[point.Dimension] = struct{}{}
		}
		if target.Status == CoverageDenominatorDefined {
			for _, required := range []CoverageDimension{CoverageTransition, CoverageProperty, CoverageObservation} {
				if _, exists := dimensions[required]; !exists {
					return fmt.Errorf("coverage denominator target %q property %q has no %s points",
						target.Identifier, target.Property, required)
				}
			}
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
		canonical.Targets[index].Points = append([]ModelCoveragePoint(nil), canonical.Targets[index].Points...)
		slices.SortFunc(canonical.Targets[index].Points, func(left, right ModelCoveragePoint) int {
			if comparison := compareStrings(string(left.Dimension), string(right.Dimension)); comparison != 0 {
				return comparison
			}
			return compareStrings(left.Identifier, right.Identifier)
		})
		canonical.Targets[index].Edges = append([]CoverageEdge(nil), canonical.Targets[index].Edges...)
		slices.SortFunc(canonical.Targets[index].Edges, func(left, right CoverageEdge) int {
			return compareStrings(left.Identifier, right.Identifier)
		})
	}
	return json.Marshal(canonical)
}

func (d CoverageDenominator) PointsForExperiment(experiment Experiment) ([]ModelCoveragePoint, error) {
	if err := experiment.Validate(); err != nil {
		return nil, fmt.Errorf("validate experiment for model coverage: %w", err)
	}
	target, found := d.targetForExperiment(experiment)
	if !found {
		return nil, errors.New("experiment does not resolve to one model coverage target")
	}
	actions := make(map[string]struct{}, len(experiment.Actions))
	for _, action := range experiment.Actions {
		actions[action.Kind] = struct{}{}
	}
	faults := make(map[string]struct{}, len(experiment.Faults))
	for _, fault := range experiment.Faults {
		faults[fault.Kind] = struct{}{}
	}
	observations := make(map[string]struct{}, len(experiment.Checkpoints))
	for _, checkpoint := range experiment.Checkpoints {
		observations[checkpoint.Observation] = struct{}{}
	}
	modules := make(map[string]struct{}, len(experiment.Model.Modules))
	for _, module := range experiment.Model.Modules {
		modules[module] = struct{}{}
	}
	points := make([]ModelCoveragePoint, 0, len(target.Points))
	for _, point := range target.Points {
		covered := false
		suffix := strings.TrimPrefix(point.Identifier, string(target.Identifier)+"/")
		switch point.Dimension {
		case CoverageTransition:
			_, covered = actions[suffix]
		case CoverageProperty:
			covered = suffix == experiment.Property.Identifier
		case CoverageFault:
			_, covered = faults[suffix]
		case CoverageObservation:
			_, covered = observations[strings.TrimPrefix(suffix, "observation.")]
		case CoverageRelation, CoverageRefinement:
			_, covered = modules[point.Source]
		}
		if covered {
			points = append(points, point)
		}
	}
	slices.SortFunc(points, func(left, right ModelCoveragePoint) int {
		if comparison := compareStrings(string(left.Dimension), string(right.Dimension)); comparison != 0 {
			return comparison
		}
		return compareStrings(left.Identifier, right.Identifier)
	})
	return points, nil
}

func (d CoverageDenominator) targetForExperiment(experiment Experiment) (CoverageTarget, bool) {
	composition, err := DefaultComposition()
	if err != nil {
		return CoverageTarget{}, false
	}
	var candidates []CoverageTarget
	boundTarget := TargetID("")
	if strings.HasPrefix(experiment.Provenance.ProofManifest, "composition:") {
		boundTarget = TargetID(strings.TrimPrefix(experiment.Provenance.ProofManifest, "composition:"))
	}
	for _, target := range d.Targets {
		if string(target.Property) != experiment.Property.Identifier {
			continue
		}
		if boundTarget != "" && target.Identifier != boundTarget {
			continue
		}
		for _, projection := range composition.Targets {
			if projection.Identifier != target.Identifier {
				continue
			}
			available := make(map[string]struct{}, len(projection.Modules))
			for _, module := range projection.Modules {
				available[string(module)] = struct{}{}
			}
			matches := true
			for _, module := range experiment.Model.Modules {
				if _, exists := available[module]; !exists {
					matches = false
					break
				}
			}
			if matches {
				candidates = append(candidates, target)
			}
		}
	}
	if len(candidates) != 1 {
		return CoverageTarget{}, false
	}
	return candidates[0], true
}

func (d CoverageDimension) valid() bool {
	switch d {
	case CoverageTransition, CoverageRelation, CoverageProperty, CoverageFault,
		CoverageObservation, CoverageRefinement:
		return true
	default:
		return false
	}
}
