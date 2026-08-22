package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/protocol/internal/generated"
)

const CompositionFormatVersion = "umpire3/composition/v4"

type ContractGuarantee struct {
	Identifier    string     `json:"identifier"`
	StatementHash string     `json:"statementHash"`
	Theorem       string     `json:"theorem"`
	Statement     string     `json:"statement"`
	Axioms        []string   `json:"axioms"`
	TrustBadge    TrustBadge `json:"trustBadge"`
}

type ContractRequirement struct {
	ProviderModule ModuleID   `json:"providerModule"`
	Guarantee      string     `json:"guarantee"`
	StatementHash  string     `json:"statementHash"`
	Theorem        string     `json:"theorem"`
	Statement      string     `json:"statement"`
	Axioms         []string   `json:"axioms"`
	TrustBadge     TrustBadge `json:"trustBadge"`
}

type ModelObligation struct {
	Identifier string         `json:"identifier"`
	Kind       string         `json:"kind"`
	Status     MetadataStatus `json:"status"`
	Detail     string         `json:"detail"`
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
	FormatVersion    string              `json:"formatVersion"`
	ResultClass      ResultClass         `json:"resultClass"`
	TrustBadge       TrustBadge          `json:"trustBadge"`
	SemanticHash     string              `json:"semanticHash"`
	SourceDigest     string              `json:"sourceDigest"`
	DependencyDigest string              `json:"dependencyDigest"`
	ArtifactDigest   string              `json:"artifactDigest"`
	CatalogHash      string              `json:"catalogHash"`
	Proof            ResolvedDeclaration `json:"proof"`
	Modules          []ModuleContract    `json:"modules"`
	Targets          []TargetProjection  `json:"targets"`
}

var defaultCompositionJSON = generated.Read(generated.Composition)

func DecodeComposition(encoded []byte) (Composition, error) {
	var composition Composition
	if err := decodeStrictJSON(bytes.NewReader(encoded), DefaultDecodeLimit, "composition", &composition); err != nil {
		return Composition{}, err
	}
	composition.deriveContractProofs()
	composition.Proof.derive()
	composition.deriveArtifactDigest()
	if err := composition.Validate(); err != nil {
		return Composition{}, err
	}
	return composition, nil
}

func DefaultComposition() (Composition, error) {
	return DecodeComposition(defaultCompositionJSON)
}

func (c Composition) Validate() error {
	if c.FormatVersion != CompositionFormatVersion || c.ResultClass != ResultClassCompositionProved ||
		!validHash(c.SemanticHash) || c.SourceDigest != c.SemanticHash || !validHash(c.DependencyDigest) ||
		len(c.Modules) == 0 || len(c.Targets) == 0 {
		return errors.New("proof-backed composition provenance, modules, and targets are required")
	}
	if err := c.Proof.Validate(); err != nil {
		return fmt.Errorf("composition proof: %w", err)
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
	catalogTargets := make(map[TargetID]TargetDeclaration, len(catalog.Targets))
	for _, target := range catalog.Targets {
		catalogTargets[TargetID(target.Identifier)] = target
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
			if guarantee.Identifier == "" {
				return fmt.Errorf("module %q has incomplete guarantee", module.Identifier)
			}
			if err := validateContractProof(guarantee.StatementHash, guarantee.Theorem,
				guarantee.Statement, guarantee.Axioms, guarantee.TrustBadge); err != nil {
				return fmt.Errorf("module %q guarantee %q: %w", module.Identifier, guarantee.Identifier, err)
			}
			if previous, conflict := guarantees[guarantee.Identifier]; conflict {
				return fmt.Errorf("guarantee %q has conflicting providers %q and %q", guarantee.Identifier, previous, module.Identifier)
			}
			guarantees[guarantee.Identifier] = module.Identifier
		}
		for _, obligation := range module.Obligations {
			if obligation.Identifier == "" || obligation.Kind == "" || obligation.Detail == "" ||
				(obligation.Status != MetadataPresent && obligation.Status != MetadataMissing) {
				return fmt.Errorf("module %q has invalid obligation metadata", module.Identifier)
			}
		}
	}
	declarations := []ResolvedDeclaration{c.Proof}
	for _, module := range c.Modules {
		for _, guarantee := range module.Provides {
			declarations = append(declarations, ResolvedDeclaration{Axioms: guarantee.Axioms})
		}
		for _, requirement := range module.Requires {
			declarations = append(declarations, ResolvedDeclaration{Axioms: requirement.Axioms})
		}
	}
	if c.TrustBadge != aggregateTrustBadge(declarations...) {
		return errors.New("composition trust badge does not match its resolved axiom inventories")
	}
	for _, consumer := range c.Modules {
		for _, requirement := range consumer.Requires {
			if err := validateContractProof(requirement.StatementHash, requirement.Theorem,
				requirement.Statement, requirement.Axioms, requirement.TrustBadge); err != nil {
				return fmt.Errorf("module %q requirement %q: %w", consumer.Identifier, requirement.Guarantee, err)
			}
			provider, exists := modules[requirement.ProviderModule]
			if !exists {
				return fmt.Errorf("module %q has missing provider %q", consumer.Identifier, requirement.ProviderModule)
			}
			if provider.Rank >= consumer.Rank {
				return fmt.Errorf("dependency cycle between %q and %q", consumer.Identifier, provider.Identifier)
			}
			matched := false
			for _, guarantee := range provider.Provides {
				if guarantee.Identifier == requirement.Guarantee &&
					guarantee.StatementHash == requirement.StatementHash &&
					guarantee.Theorem == requirement.Theorem && guarantee.Statement == requirement.Statement &&
					slices.Equal(guarantee.Axioms, requirement.Axioms) &&
					guarantee.TrustBadge == requirement.TrustBadge {
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
		catalogTarget, known := catalogTargets[target.Identifier]
		if !known {
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
		targetModules := make(map[ModuleID]struct{}, len(target.Modules))
		for _, moduleID := range target.Modules {
			module, exists := modules[moduleID]
			if !exists {
				return fmt.Errorf("target %q references missing module %q", target.Identifier, moduleID)
			}
			targetModules[moduleID] = struct{}{}
			for _, action := range module.InterferenceActions {
				if _, retainedAction := retained[action]; !retainedAction {
					return fmt.Errorf("target %q drops interference action %q", target.Identifier, action)
				}
			}
		}
		for _, moduleID := range catalogTarget.Modules {
			if _, present := targetModules[ModuleID(moduleID)]; !present {
				return fmt.Errorf("target %q omits catalog module %q", target.Identifier, moduleID)
			}
		}
		targetProperties := make(map[PropertyID]struct{}, len(target.Properties))
		for _, property := range target.Properties {
			if _, known := catalogProperties[property]; !known {
				return fmt.Errorf("target %q references unknown property %q", target.Identifier, property)
			}
			targetProperties[property] = struct{}{}
		}
		for _, property := range catalogTarget.Properties {
			if _, present := targetProperties[PropertyID(property)]; !present {
				return fmt.Errorf("target %q omits catalog property %q", target.Identifier, property)
			}
		}
		for _, omission := range target.Omissions {
			if omission.Identifier == "" || omission.Reason == "" || omission.MaxCount <= 0 {
				return fmt.Errorf("target %q has unbounded omission", target.Identifier)
			}
		}
	}
	expectedArtifactDigest, err := c.computedArtifactDigest()
	if err != nil {
		return err
	}
	if c.ArtifactDigest != expectedArtifactDigest {
		return errors.New("composition artifact digest does not match its canonical contents")
	}
	return nil
}

func (c *Composition) deriveContractProofs() {
	for moduleIndex := range c.Modules {
		for guaranteeIndex := range c.Modules[moduleIndex].Provides {
			guarantee := &c.Modules[moduleIndex].Provides[guaranteeIndex]
			slices.Sort(guarantee.Axioms)
			if guarantee.StatementHash == "derived" {
				guarantee.StatementHash = statementDigest(guarantee.Statement)
			}
		}
		for requirementIndex := range c.Modules[moduleIndex].Requires {
			requirement := &c.Modules[moduleIndex].Requires[requirementIndex]
			slices.Sort(requirement.Axioms)
			if requirement.StatementHash == "derived" {
				requirement.StatementHash = statementDigest(requirement.Statement)
			}
		}
	}
}

func (c *Composition) deriveArtifactDigest() {
	if c.ArtifactDigest != "derived" {
		return
	}
	digest, err := c.computedArtifactDigest()
	if err == nil {
		c.ArtifactDigest = digest
	}
}

func (c Composition) computedArtifactDigest() (string, error) {
	canonical := c
	canonical.ArtifactDigest = ""
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("encode composition digest payload: %w", err)
	}
	return digestBytes(encoded), nil
}

func validateContractProof(
	statementHash string,
	theorem string,
	statement string,
	axioms []string,
	trustBadge TrustBadge,
) error {
	if theorem == "" || statement == "" || statementHash != statementDigest(statement) {
		return errors.New("resolved theorem and derived statement hash are required")
	}
	if !slices.IsSorted(axioms) || len(slices.Compact(append([]string(nil), axioms...))) != len(axioms) {
		return errors.New("axioms must be sorted and unique")
	}
	for _, axiom := range axioms {
		if axiom == "" || axiom == "sorryAx" || axiom == "Lean.ofReduceBool" {
			return fmt.Errorf("invalid axiom %q", axiom)
		}
	}
	expectedTrust := TrustBadgeKernel
	if len(axioms) != 0 {
		expectedTrust = TrustBadgeKernelWithDeclaredAxioms
	}
	if trustBadge != expectedTrust {
		return errors.New("trust badge does not match resolved axioms")
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

func (c Composition) MissingMetadata() []ModelObligation {
	var missing []ModelObligation
	for _, module := range c.Modules {
		for _, obligation := range module.Obligations {
			if obligation.Status == MetadataMissing {
				missing = append(missing, obligation)
			}
		}
	}
	slices.SortFunc(missing, func(left, right ModelObligation) int {
		return stringCompare(left.Identifier, right.Identifier)
	})
	return missing
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
