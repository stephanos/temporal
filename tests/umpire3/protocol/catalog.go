package protocol

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

const CatalogFormatVersion = "umpire3/catalog/v1"

type CapabilityID string

type TypeDeclaration struct {
	Identifier  string `json:"identifier"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

type CapabilityDeclaration struct {
	Identifier  CapabilityID `json:"identifier"`
	Description string       `json:"description"`
}

type ParameterDeclaration struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Values   []string `json:"values"`
}

type ProjectionDeclaration struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type FootprintDeclaration struct {
	Protocol string `json:"protocol"`
	Service  string `json:"service"`
	Route    string `json:"route"`
}

type ActionDeclaration struct {
	Identifier           string                  `json:"identifier"`
	Description          string                  `json:"description"`
	Parameters           []ParameterDeclaration  `json:"parameters"`
	Dependencies         []string                `json:"dependencies"`
	Projections          []ProjectionDeclaration `json:"projections"`
	Footprint            []FootprintDeclaration  `json:"footprint"`
	RequiredCapabilities []CapabilityID          `json:"requiredCapabilities"`
}

type EntityDeclaration struct {
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
}

type RelationDeclaration struct {
	Identifier  string `json:"identifier"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	Description string `json:"description"`
}

type ObservationDeclaration struct {
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
}

type EvidenceDeclaration struct {
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
}

type PropertyDeclaration struct {
	Identifier    string     `json:"identifier"`
	Description   string     `json:"description"`
	StatementHash string     `json:"statementHash"`
	Evidence      []string   `json:"evidence"`
	Theorem       string     `json:"theorem"`
	Statement     string     `json:"statement"`
	Axioms        []string   `json:"axioms"`
	TrustBadge    TrustBadge `json:"trustBadge"`
}

type PolicyDeclaration struct {
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
}

type FaultDeclaration struct {
	Identifier           string         `json:"identifier"`
	Description          string         `json:"description"`
	SafetyClass          string         `json:"safetyClass"`
	ScopeDimensions      []string       `json:"scopeDimensions"`
	RequiredCapabilities []CapabilityID `json:"requiredCapabilities"`
}

type ModuleDeclaration struct {
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
}

type TargetDeclaration struct {
	Identifier string   `json:"identifier"`
	Modules    []string `json:"modules"`
	Properties []string `json:"properties"`
}

type Catalog struct {
	FormatVersion  string                   `json:"formatVersion"`
	CatalogVersion string                   `json:"catalogVersion"`
	LeanVersion    string                   `json:"leanVersion"`
	SemanticHash   string                   `json:"semanticHash"`
	Types          []TypeDeclaration        `json:"types"`
	Capabilities   []CapabilityDeclaration  `json:"capabilities"`
	Actions        []ActionDeclaration      `json:"actions"`
	Entities       []EntityDeclaration      `json:"entities"`
	Relations      []RelationDeclaration    `json:"relations"`
	Observations   []ObservationDeclaration `json:"observations"`
	Evidence       []EvidenceDeclaration    `json:"evidence"`
	Properties     []PropertyDeclaration    `json:"properties"`
	Policies       []PolicyDeclaration      `json:"policies"`
	Faults         []FaultDeclaration       `json:"faults"`
	Modules        []ModuleDeclaration      `json:"modules"`
	Targets        []TargetDeclaration      `json:"targets"`
}

//go:embed generated/catalog.json
var defaultCatalogJSON []byte

func DecodeCatalog(reader io.Reader, limit int64) (Catalog, error) {
	var catalog Catalog
	if err := decodeStrictJSON(reader, limit, "catalog", &catalog); err != nil {
		return Catalog{}, err
	}
	catalog.derivePropertyProofs()
	if err := catalog.Validate(); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

func DefaultCatalog() (Catalog, error) {
	return DecodeCatalog(bytes.NewReader(defaultCatalogJSON), DefaultDecodeLimit)
}

func (c Catalog) Validate() error {
	if c.FormatVersion != CatalogFormatVersion {
		return fmt.Errorf("unsupported catalog format version %q", c.FormatVersion)
	}
	if c.CatalogVersion == "" || c.LeanVersion == "" || !validHash(c.SemanticHash) {
		return errors.New("complete catalog provenance is required")
	}
	if len(c.Types) == 0 || len(c.Capabilities) == 0 || len(c.Actions) == 0 ||
		len(c.Entities) == 0 || len(c.Relations) == 0 || len(c.Observations) == 0 ||
		len(c.Evidence) == 0 || len(c.Properties) == 0 || len(c.Policies) == 0 ||
		len(c.Faults) == 0 || len(c.Modules) == 0 || len(c.Targets) == 0 {
		return errors.New("catalog declarations are incomplete")
	}

	types, err := declarationSet("type", len(c.Types), func(index int) (string, string) {
		return c.Types[index].Identifier, c.Types[index].Description
	})
	if err != nil {
		return err
	}
	capabilities, err := declarationSet("capability", len(c.Capabilities), func(index int) (string, string) {
		return string(c.Capabilities[index].Identifier), c.Capabilities[index].Description
	})
	if err != nil {
		return err
	}
	actions, err := declarationSet("action", len(c.Actions), func(index int) (string, string) {
		return c.Actions[index].Identifier, c.Actions[index].Description
	})
	if err != nil {
		return err
	}
	for _, action := range c.Actions {
		seen := make(map[string]struct{}, len(action.Footprint))
		for _, call := range action.Footprint {
			if call.Protocol == "" || call.Service == "" || call.Route == "" {
				return fmt.Errorf("action %q has an incomplete footprint declaration", action.Identifier)
			}
			identity := call.Protocol + "\x00" + call.Service + "\x00" + call.Route
			if _, duplicate := seen[identity]; duplicate {
				return fmt.Errorf("action %q has duplicate footprint call %q", action.Identifier, call.Route)
			}
			seen[identity] = struct{}{}
		}
	}
	entities, err := declarationSet("entity", len(c.Entities), func(index int) (string, string) {
		return c.Entities[index].Identifier, c.Entities[index].Description
	})
	if err != nil {
		return err
	}
	if _, err := declarationSet("relation", len(c.Relations), func(index int) (string, string) {
		return c.Relations[index].Identifier, c.Relations[index].Description
	}); err != nil {
		return err
	}
	if _, err := declarationSet("observation", len(c.Observations), func(index int) (string, string) {
		return c.Observations[index].Identifier, c.Observations[index].Description
	}); err != nil {
		return err
	}
	evidence, err := declarationSet("evidence", len(c.Evidence), func(index int) (string, string) {
		return c.Evidence[index].Identifier, c.Evidence[index].Description
	})
	if err != nil {
		return err
	}
	properties, err := declarationSet("property", len(c.Properties), func(index int) (string, string) {
		return c.Properties[index].Identifier, c.Properties[index].Description
	})
	if err != nil {
		return err
	}
	if _, err := declarationSet("policy", len(c.Policies), func(index int) (string, string) {
		return c.Policies[index].Identifier, c.Policies[index].Description
	}); err != nil {
		return err
	}
	if _, err := declarationSet("fault", len(c.Faults), func(index int) (string, string) {
		return c.Faults[index].Identifier, c.Faults[index].Description
	}); err != nil {
		return err
	}
	modules, err := declarationSet("module", len(c.Modules), func(index int) (string, string) {
		return c.Modules[index].Identifier, c.Modules[index].Description
	})
	if err != nil {
		return err
	}
	if _, err := declarationSet("target", len(c.Targets), func(index int) (string, string) {
		return c.Targets[index].Identifier, c.Targets[index].Identifier
	}); err != nil {
		return err
	}

	for _, action := range c.Actions {
		parameterNames := make(map[string]struct{}, len(action.Parameters))
		for _, parameter := range action.Parameters {
			if parameter.Name == "" {
				return fmt.Errorf("action %q has an empty parameter name", action.Identifier)
			}
			if _, duplicate := parameterNames[parameter.Name]; duplicate {
				return fmt.Errorf("action %q has duplicate parameter %q", action.Identifier, parameter.Name)
			}
			parameterNames[parameter.Name] = struct{}{}
			if _, known := types[parameter.Type]; !known {
				return fmt.Errorf("action %q parameter %q references unknown type %q", action.Identifier, parameter.Name, parameter.Type)
			}
			if len(parameter.Values) != 0 && parameter.Type != "string" {
				return fmt.Errorf("action %q parameter %q has values for non-string type %q",
					action.Identifier, parameter.Name, parameter.Type)
			}
			values := make(map[string]struct{}, len(parameter.Values))
			for _, value := range parameter.Values {
				if value == "" {
					return fmt.Errorf("action %q parameter %q has an empty value", action.Identifier, parameter.Name)
				}
				if _, duplicate := values[value]; duplicate {
					return fmt.Errorf("action %q parameter %q has duplicate value %q",
						action.Identifier, parameter.Name, value)
				}
				values[value] = struct{}{}
			}
		}
		for _, capability := range action.RequiredCapabilities {
			if _, known := capabilities[string(capability)]; !known {
				return fmt.Errorf("action %q references unknown capability %q", action.Identifier, capability)
			}
		}
		dependencies := make(map[string]struct{}, len(action.Dependencies))
		for _, dependency := range action.Dependencies {
			if _, known := actions[dependency]; !known {
				return fmt.Errorf("action %q depends on unknown action %q", action.Identifier, dependency)
			}
			if _, duplicate := dependencies[dependency]; duplicate {
				return fmt.Errorf("action %q has duplicate dependency %q", action.Identifier, dependency)
			}
			dependencies[dependency] = struct{}{}
		}
		projections := make(map[string]struct{}, len(action.Projections))
		for _, projection := range action.Projections {
			if projection.Name == "" {
				return fmt.Errorf("action %q has empty projection", action.Identifier)
			}
			if _, known := types[projection.Type]; !known {
				return fmt.Errorf("action %q projection %q references unknown type %q", action.Identifier, projection.Name, projection.Type)
			}
			if _, duplicate := projections[projection.Name]; duplicate {
				return fmt.Errorf("action %q has duplicate projection %q", action.Identifier, projection.Name)
			}
			projections[projection.Name] = struct{}{}
		}
	}
	if err := validateActionDependencies(c.Actions); err != nil {
		return err
	}
	for _, relation := range c.Relations {
		if _, known := entities[relation.Source]; !known {
			return fmt.Errorf("relation %q references unknown source entity %q", relation.Identifier, relation.Source)
		}
		if _, known := entities[relation.Target]; !known {
			return fmt.Errorf("relation %q references unknown target entity %q", relation.Identifier, relation.Target)
		}
	}
	for _, property := range c.Properties {
		if !validHash(property.StatementHash) {
			return fmt.Errorf("property %q has invalid statement hash", property.Identifier)
		}
		if property.Theorem == "" || property.Statement == "" {
			return fmt.Errorf("property %q has no resolved Lean theorem", property.Identifier)
		}
		if property.StatementHash != statementDigest(property.Statement) {
			return fmt.Errorf("property %q derived statement hash does not match its resolved theorem", property.Identifier)
		}
		if !slices.IsSorted(property.Axioms) || len(slices.Compact(append([]string(nil), property.Axioms...))) != len(property.Axioms) {
			return fmt.Errorf("property %q axioms are not sorted and unique", property.Identifier)
		}
		for _, axiom := range property.Axioms {
			if axiom == "" || strings.Contains(axiom, "sorryAx") || axiom == "Lean.ofReduceBool" {
				return fmt.Errorf("property %q has invalid axiom %q", property.Identifier, axiom)
			}
		}
		expectedTrust := TrustBadgeKernel
		if len(property.Axioms) != 0 {
			expectedTrust = TrustBadgeKernelWithDeclaredAxioms
		}
		if property.TrustBadge != expectedTrust {
			return fmt.Errorf("property %q trust badge does not match its resolved axioms", property.Identifier)
		}
		for _, requirement := range property.Evidence {
			if _, known := evidence[requirement]; !known {
				return fmt.Errorf("property %q references unknown evidence %q", property.Identifier, requirement)
			}
		}
	}
	for _, fault := range c.Faults {
		if fault.SafetyClass != "controlled" && fault.SafetyClass != "restricted" {
			return fmt.Errorf("fault %q has unknown safety class %q", fault.Identifier, fault.SafetyClass)
		}
		if len(fault.ScopeDimensions) == 0 {
			return fmt.Errorf("fault %q has no isolation scope", fault.Identifier)
		}
		dimensions := make(map[string]struct{}, len(fault.ScopeDimensions))
		for _, dimension := range fault.ScopeDimensions {
			if dimension == "" {
				return fmt.Errorf("fault %q has an empty scope dimension", fault.Identifier)
			}
			if _, duplicate := dimensions[dimension]; duplicate {
				return fmt.Errorf("fault %q has duplicate scope dimension %q", fault.Identifier, dimension)
			}
			dimensions[dimension] = struct{}{}
		}
		for _, capability := range fault.RequiredCapabilities {
			if _, known := capabilities[string(capability)]; !known {
				return fmt.Errorf("fault %q references unknown capability %q", fault.Identifier, capability)
			}
		}
	}
	for _, target := range c.Targets {
		if len(target.Modules) == 0 || len(target.Properties) == 0 {
			return fmt.Errorf("target %q requires modules and properties", target.Identifier)
		}
		for _, module := range target.Modules {
			if _, known := modules[module]; !known {
				return fmt.Errorf("target %q references unknown module %q", target.Identifier, module)
			}
		}
		for _, property := range target.Properties {
			if _, known := properties[property]; !known {
				return fmt.Errorf("target %q references unknown property %q", target.Identifier, property)
			}
		}
	}
	return nil
}

func (c *Catalog) derivePropertyProofs() {
	for index := range c.Properties {
		if c.Properties[index].StatementHash == "derived" {
			c.Properties[index].StatementHash = statementDigest(c.Properties[index].Statement)
		}
	}
}

func statementDigest(statement string) string {
	digest := sha256.Sum256([]byte(statement))
	return "sha256:" + hex.EncodeToString(digest[:])
}

func validateActionDependencies(actions []ActionDeclaration) error {
	edges := make(map[string][]string, len(actions))
	for _, action := range actions {
		edges[action.Identifier] = append([]string(nil), action.Dependencies...)
	}
	visiting := make(map[string]bool, len(actions))
	visited := make(map[string]bool, len(actions))
	var visit func(string) bool
	visit = func(action string) bool {
		if visiting[action] {
			return false
		}
		if visited[action] {
			return true
		}
		visiting[action] = true
		for _, dependency := range edges[action] {
			if !visit(dependency) {
				return false
			}
		}
		visiting[action] = false
		visited[action] = true
		return true
	}
	for action := range edges {
		if !visit(action) {
			return errors.New("action dependency cycle detected")
		}
	}
	return nil
}

func declarationSet(kind string, count int, declaration func(int) (string, string)) (map[string]struct{}, error) {
	result := make(map[string]struct{}, count)
	for index := range count {
		identifier, description := declaration(index)
		if identifier == "" || description == "" {
			return nil, fmt.Errorf("%s identifier and description are required", kind)
		}
		if _, duplicate := result[identifier]; duplicate {
			return nil, fmt.Errorf("duplicate %s %q", kind, identifier)
		}
		result[identifier] = struct{}{}
	}
	return result, nil
}

func (c Catalog) Action(identifier string) (ActionDeclaration, bool) {
	for _, action := range c.Actions {
		if action.Identifier == identifier {
			return action, true
		}
	}
	return ActionDeclaration{}, false
}

func (c Catalog) Property(identifier string) (PropertyDeclaration, bool) {
	for _, property := range c.Properties {
		if property.Identifier == identifier {
			return property, true
		}
	}
	return PropertyDeclaration{}, false
}

func (c Catalog) Fault(identifier string) (FaultDeclaration, bool) {
	for _, declaration := range c.Faults {
		if declaration.Identifier == identifier {
			return declaration, true
		}
	}
	return FaultDeclaration{}, false
}

func (c Catalog) HasCapability(identifier CapabilityID) bool {
	for _, capability := range c.Capabilities {
		if capability.Identifier == identifier {
			return true
		}
	}
	return false
}

func (c Catalog) CanonicalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("encode canonical catalog: %w", err)
	}
	return encoded, nil
}

func (c Catalog) Digest() (string, error) {
	encoded, err := c.CanonicalJSON()
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
