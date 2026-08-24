package monitor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tools/umpire3/protocol/internal/generated"
)

const MonitorFormatVersion = "umpire3/monitor-programs/v1"

type MonitorOperation string

const (
	MonitorObservation MonitorOperation = "observation"
	MonitorAll         MonitorOperation = "all"
	MonitorAny         MonitorOperation = "any"
	MonitorNot         MonitorOperation = "not"
	MonitorImplies     MonitorOperation = "implies"
)

type MonitorExpression struct {
	Operation   MonitorOperation    `json:"operation"`
	Observation ObservationID       `json:"observation,omitempty"`
	Expected    *bool               `json:"expected,omitempty"`
	Children    []MonitorExpression `json:"children,omitempty"`
}

type MonitorProgram struct {
	Identifier string            `json:"identifier"`
	Property   PropertyID        `json:"property"`
	Evidence   []EvidenceID      `json:"evidence"`
	Coverage   []string          `json:"coverage"`
	Expression MonitorExpression `json:"expression"`
}

type MonitorCatalog struct {
	FormatVersion string           `json:"formatVersion"`
	SemanticHash  string           `json:"semanticHash"`
	CatalogHash   string           `json:"catalogHash"`
	Programs      []MonitorProgram `json:"programs"`
}

type ObservedFact struct {
	Observation ObservationID
	Value       bool
}

type MonitorEvaluation struct {
	Complete       bool
	Satisfied      bool
	Missing        []ObservationID
	Contradictions []ObservationID
}

var defaultMonitorCatalogJSON = generated.Read(generated.MonitorPrograms)

func DecodeMonitorCatalog(encoded []byte) (MonitorCatalog, error) {
	var catalog MonitorCatalog
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "monitor catalog", &catalog); err != nil {
		return MonitorCatalog{}, err
	}
	if err := catalog.Validate(); err != nil {
		return MonitorCatalog{}, err
	}
	return catalog, nil
}

func DefaultMonitorCatalog() (MonitorCatalog, error) {
	return DecodeMonitorCatalog(defaultMonitorCatalogJSON)
}

func (c MonitorCatalog) CanonicalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("encode monitor catalog: %w", err)
	}
	return encoded, nil
}

func (c MonitorCatalog) Validate() error {
	if c.FormatVersion != MonitorFormatVersion || !validHash(c.SemanticHash) || len(c.Programs) == 0 {
		return errors.New("complete monitor catalog is required")
	}
	semanticCatalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	digest, err := semanticCatalog.Digest()
	if err != nil {
		return err
	}
	if c.CatalogHash != digest {
		return fmt.Errorf("monitor catalog hash %q does not match semantic catalog %q", c.CatalogHash, digest)
	}
	properties := make(map[PropertyID]PropertyDeclaration, len(semanticCatalog.Properties))
	for _, property := range semanticCatalog.Properties {
		properties[PropertyID(property.Identifier)] = property
	}
	observations := make(map[ObservationID]struct{}, len(semanticCatalog.Observations))
	for _, observation := range semanticCatalog.Observations {
		observations[ObservationID(observation.Identifier)] = struct{}{}
	}
	identifiers := make(map[string]struct{}, len(c.Programs))
	programmedProperties := make(map[PropertyID]struct{}, len(c.Programs))
	for _, program := range c.Programs {
		if program.Identifier == "" || len(program.Coverage) == 0 {
			return errors.New("monitor identifier and coverage are required")
		}
		if _, duplicate := identifiers[program.Identifier]; duplicate {
			return fmt.Errorf("duplicate monitor %q", program.Identifier)
		}
		identifiers[program.Identifier] = struct{}{}
		property, known := properties[program.Property]
		if !known {
			return fmt.Errorf("monitor %q references unknown property %q", program.Identifier, program.Property)
		}
		if _, duplicate := programmedProperties[program.Property]; duplicate {
			return fmt.Errorf("duplicate monitor for property %q", program.Property)
		}
		programmedProperties[program.Property] = struct{}{}
		expectedEvidence := append([]string(nil), property.Evidence...)
		actualEvidence := make([]string, len(program.Evidence))
		for index, evidence := range program.Evidence {
			actualEvidence[index] = string(evidence)
		}
		slices.Sort(expectedEvidence)
		slices.Sort(actualEvidence)
		if !slices.Equal(expectedEvidence, actualEvidence) {
			return fmt.Errorf("monitor %q evidence does not match property requirements", program.Identifier)
		}
		if err := program.Expression.validate(observations, 0); err != nil {
			return fmt.Errorf("monitor %q: %w", program.Identifier, err)
		}
	}
	return nil
}

func (c MonitorCatalog) Program(property PropertyID) (MonitorProgram, bool) {
	for _, program := range c.Programs {
		if program.Property == property {
			return program, true
		}
	}
	return MonitorProgram{}, false
}

func (e MonitorExpression) validate(observations map[ObservationID]struct{}, depth int) error {
	if depth > 32 {
		return errors.New("monitor expression exceeds 32 levels")
	}
	switch e.Operation {
	case MonitorObservation:
		if e.Observation == "" || e.Expected == nil || len(e.Children) != 0 {
			return errors.New("observation expression is incomplete")
		}
		if _, known := observations[e.Observation]; !known {
			return fmt.Errorf("unknown observation %q", e.Observation)
		}
	case MonitorAll, MonitorAny:
		if e.Observation != "" || e.Expected != nil || len(e.Children) == 0 {
			return fmt.Errorf("%s expression requires only children", e.Operation)
		}
	case MonitorNot:
		if e.Observation != "" || e.Expected != nil || len(e.Children) != 1 {
			return errors.New("not expression requires one child")
		}
	case MonitorImplies:
		if e.Observation != "" || e.Expected != nil || len(e.Children) != 2 {
			return errors.New("implies expression requires two children")
		}
	default:
		return fmt.Errorf("unknown monitor operation %q", e.Operation)
	}
	for _, child := range e.Children {
		if err := child.validate(observations, depth+1); err != nil {
			return err
		}
	}
	return nil
}
