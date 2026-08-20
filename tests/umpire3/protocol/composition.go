package protocol

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

const CompositionFormatVersion = "umpire3/composition/v1"

type ContractGuarantee struct {
	Identifier    string `json:"identifier"`
	StatementHash string `json:"statementHash"`
}

type ContractRequirement struct {
	ProviderModule ModuleID `json:"providerModule"`
	Guarantee      string   `json:"guarantee"`
	StatementHash  string   `json:"statementHash"`
}

type ModelObligation struct {
	Identifier string `json:"identifier"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	Detail     string `json:"detail"`
}

type ModuleContract struct {
	Identifier          ModuleID              `json:"identifier"`
	Rank                int                   `json:"rank"`
	Owns                []string              `json:"owns"`
	Provides            []ContractGuarantee   `json:"provides"`
	Requires            []ContractRequirement `json:"requires"`
	InterferenceActions []string              `json:"interferenceActions"`
	Obligations         []ModelObligation     `json:"obligations"`
}

type ProjectionOmission struct {
	Identifier string `json:"identifier"`
	Reason     string `json:"reason"`
	MaxCount   int    `json:"maxCount"`
}

type TargetProjection struct {
	Identifier      TargetID             `json:"identifier"`
	Modules         []ModuleID           `json:"modules"`
	Properties      []PropertyID         `json:"properties"`
	RetainedActions []string             `json:"retainedActions"`
	Omissions       []ProjectionOmission `json:"omissions"`
}

type Composition struct {
	FormatVersion string             `json:"formatVersion"`
	SemanticHash  string             `json:"semanticHash"`
	CatalogHash   string             `json:"catalogHash"`
	Modules       []ModuleContract   `json:"modules"`
	Targets       []TargetProjection `json:"targets"`
}

//go:embed generated/composition.json
var defaultCompositionJSON []byte

func DecodeComposition(encoded []byte) (Composition, error) {
	var composition Composition
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "composition", &composition); err != nil {
		return Composition{}, err
	}
	if err := composition.Validate(); err != nil {
		return Composition{}, err
	}
	return composition, nil
}

func DefaultComposition() (Composition, error) {
	return DecodeComposition(defaultCompositionJSON)
}

func (c Composition) Validate() error {
	if c.FormatVersion != CompositionFormatVersion || !validHash(c.SemanticHash) || len(c.Modules) == 0 || len(c.Targets) == 0 {
		return errors.New("complete composition provenance, modules, and targets are required")
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return err
	}
	if c.CatalogHash != catalogHash {
		return fmt.Errorf("composition catalog hash %q does not match semantic catalog %q", c.CatalogHash, catalogHash)
	}
	catalogModules := make(map[ModuleID]struct{}, len(catalog.Modules))
	for _, module := range catalog.Modules {
		catalogModules[ModuleID(module.Identifier)] = struct{}{}
	}
	catalogProperties := make(map[PropertyID]struct{}, len(catalog.Properties))
	for _, property := range catalog.Properties {
		catalogProperties[PropertyID(property.Identifier)] = struct{}{}
	}
	catalogTargets := make(map[TargetID]struct{}, len(catalog.Targets))
	for _, target := range catalog.Targets {
		catalogTargets[TargetID(target.Identifier)] = struct{}{}
	}

	modules := make(map[ModuleID]ModuleContract, len(c.Modules))
	owners := make(map[string]ModuleID)
	guarantees := make(map[string]ModuleID)
	for _, module := range c.Modules {
		if module.Identifier == "" || module.Rank < 0 {
			return errors.New("module identifier and non-negative rank are required")
		}
		if _, known := catalogModules[module.Identifier]; !known {
			return fmt.Errorf("unknown composition module %q", module.Identifier)
		}
		if _, duplicate := modules[module.Identifier]; duplicate {
			return fmt.Errorf("duplicate composition module %q", module.Identifier)
		}
		modules[module.Identifier] = module
		for _, owned := range module.Owns {
			if owned == "" {
				return fmt.Errorf("module %q has empty ownership", module.Identifier)
			}
			if previous, conflict := owners[owned]; conflict {
				return fmt.Errorf("conflicting owner for %q: %q and %q", owned, previous, module.Identifier)
			}
			owners[owned] = module.Identifier
		}
		for _, guarantee := range module.Provides {
			if guarantee.Identifier == "" || !validHash(guarantee.StatementHash) {
				return fmt.Errorf("module %q has incomplete guarantee", module.Identifier)
			}
			if previous, conflict := guarantees[guarantee.Identifier]; conflict {
				return fmt.Errorf("guarantee %q has conflicting providers %q and %q", guarantee.Identifier, previous, module.Identifier)
			}
			guarantees[guarantee.Identifier] = module.Identifier
		}
		for _, obligation := range module.Obligations {
			if obligation.Identifier == "" || obligation.Kind == "" || obligation.Detail == "" ||
				(obligation.Status != "complete" && obligation.Status != "pending") {
				return fmt.Errorf("module %q has incomplete obligation", module.Identifier)
			}
		}
	}
	for _, consumer := range c.Modules {
		for _, requirement := range consumer.Requires {
			provider, exists := modules[requirement.ProviderModule]
			if !exists {
				return fmt.Errorf("module %q has missing provider %q", consumer.Identifier, requirement.ProviderModule)
			}
			if provider.Rank >= consumer.Rank {
				return fmt.Errorf("dependency cycle between %q and %q", consumer.Identifier, provider.Identifier)
			}
			matched := false
			for _, guarantee := range provider.Provides {
				if guarantee.Identifier == requirement.Guarantee && guarantee.StatementHash == requirement.StatementHash {
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf("module %q has unsatisfied requirement %q", consumer.Identifier, requirement.Guarantee)
			}
		}
	}

	targets := make(map[TargetID]struct{}, len(c.Targets))
	for _, target := range c.Targets {
		if target.Identifier == "" || len(target.Modules) == 0 || len(target.Properties) == 0 || len(target.RetainedActions) == 0 {
			return fmt.Errorf("target %q is vacuous", target.Identifier)
		}
		if _, known := catalogTargets[target.Identifier]; !known {
			return fmt.Errorf("unknown composition target %q", target.Identifier)
		}
		if _, duplicate := targets[target.Identifier]; duplicate {
			return fmt.Errorf("duplicate composition target %q", target.Identifier)
		}
		targets[target.Identifier] = struct{}{}
		retained := make(map[string]struct{}, len(target.RetainedActions))
		for _, action := range target.RetainedActions {
			retained[action] = struct{}{}
		}
		for _, moduleID := range target.Modules {
			module, exists := modules[moduleID]
			if !exists {
				return fmt.Errorf("target %q references missing module %q", target.Identifier, moduleID)
			}
			for _, action := range module.InterferenceActions {
				if _, retainedAction := retained[action]; !retainedAction {
					return fmt.Errorf("target %q drops interference action %q", target.Identifier, action)
				}
			}
		}
		for _, property := range target.Properties {
			if _, known := catalogProperties[property]; !known {
				return fmt.Errorf("target %q references unknown property %q", target.Identifier, property)
			}
		}
		for _, omission := range target.Omissions {
			if omission.Identifier == "" || omission.Reason == "" || omission.MaxCount <= 0 {
				return fmt.Errorf("target %q has unbounded omission", target.Identifier)
			}
		}
	}
	return nil
}

func (c Composition) Module(identifier ModuleID) (ModuleContract, bool) {
	for _, module := range c.Modules {
		if module.Identifier == identifier {
			return module, true
		}
	}
	return ModuleContract{}, false
}

func (c Composition) PendingObligations() []ModelObligation {
	var pending []ModelObligation
	for _, module := range c.Modules {
		for _, obligation := range module.Obligations {
			if obligation.Status == "pending" {
				pending = append(pending, obligation)
			}
		}
	}
	slices.SortFunc(pending, func(left, right ModelObligation) int {
		return stringCompare(left.Identifier, right.Identifier)
	})
	return pending
}

func (c Composition) CanonicalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("encode composition: %w", err)
	}
	return encoded, nil
}

func stringCompare(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
