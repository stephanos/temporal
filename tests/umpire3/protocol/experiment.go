package protocol

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const DefaultDecodeLimit int64 = 1 << 20

type Model struct {
	Modules        []string `json:"modules"`
	SourceRevision string   `json:"sourceRevision"`
	SemanticHash   string   `json:"semanticHash"`
	CatalogHash    string   `json:"catalogHash"`
	LeanVersion    string   `json:"leanVersion"`
}

type Property struct {
	Identifier    string `json:"identifier"`
	StatementHash string `json:"statementHash"`
	Claim         string `json:"claim"`
}

type Bounds struct {
	MaxDepth   int `json:"maxDepth"`
	MaxResults int `json:"maxResults"`
}

type Assumption struct {
	Identifier    string `json:"identifier"`
	StatementHash string `json:"statementHash"`
}

type Scope struct {
	Bounds      Bounds       `json:"bounds"`
	Assumptions []Assumption `json:"assumptions"`
	Strategy    string       `json:"strategy"`
	Seed        int64        `json:"seed"`
}

type Resource struct {
	Identifier string `json:"identifier"`
	Kind       string `json:"kind"`
}

type ResponseMode string

const (
	ResponseSynchronous  ResponseMode = "synchronous"
	ResponseAsynchronous ResponseMode = "asynchronous"
	ResponseDeferred     ResponseMode = "deferred"
	ResponseBlocking     ResponseMode = "blocking"
	ResponseFailure      ResponseMode = "failure"
)

type Action struct {
	Identifier           string       `json:"identifier"`
	Kind                 string       `json:"kind"`
	Arguments            []NamedValue `json:"arguments,omitempty"`
	Bindings             []Binding    `json:"bindings,omitempty"`
	RequiredCapabilities []string     `json:"requiredCapabilities"`
	PreCheckpoint        string       `json:"preCheckpoint,omitempty"`
	PostCheckpoint       string       `json:"postCheckpoint,omitempty"`
	ResponseMode         ResponseMode `json:"responseMode,omitempty"`
	MaxBlockNanos        int64        `json:"maxBlockNanos,omitempty"`
}

func (a Action) EffectiveResponseMode() ResponseMode {
	if a.ResponseMode == "" {
		return ResponseSynchronous
	}
	return a.ResponseMode
}

type Policy struct {
	Identifier string       `json:"identifier"`
	Kind       string       `json:"kind"`
	Scope      []string     `json:"scope"`
	Arguments  []NamedValue `json:"arguments,omitempty"`
}

type Fault struct {
	Identifier           string          `json:"identifier"`
	Kind                 string          `json:"kind"`
	Policy               string          `json:"policy,omitempty"`
	SafetyClass          string          `json:"safetyClass"`
	Scope                FaultScope      `json:"scope"`
	Occurrence           FaultOccurrence `json:"occurrence"`
	Interval             FaultInterval   `json:"interval"`
	Arguments            []NamedValue    `json:"arguments,omitempty"`
	RequiredCapabilities []string        `json:"requiredCapabilities"`
}

type FaultScope struct {
	Resources    []string `json:"resources"`
	Endpoints    []string `json:"endpoints"`
	TaskQueues   []string `json:"taskQueues"`
	Services     []string `json:"services"`
	Routes       []string `json:"routes"`
	Participants []string `json:"participants"`
	Attempts     []int    `json:"attempts"`
}

type FaultOccurrence struct {
	First int `json:"first"`
	Count int `json:"count"`
}

type FaultInterval struct {
	StartAction string `json:"startAction"`
	StopAction  string `json:"stopAction"`
}

type OrderRelation string

const (
	OrderUser          OrderRelation = "user"
	OrderSemantic      OrderRelation = "semantic"
	OrderSameSource    OrderRelation = "same-source"
	OrderRuntimeCausal OrderRelation = "runtime-causal"
)

type OrderConstraint struct {
	Before   string        `json:"before"`
	After    string        `json:"after"`
	Relation OrderRelation `json:"relation"`
}

type Checkpoint struct {
	Identifier     string `json:"identifier"`
	Observation    string `json:"observation"`
	Ordering       string `json:"ordering"`
	OmissionPolicy string `json:"omissionPolicy"`
}

type Provenance struct {
	Kind          string `json:"kind"`
	ProofManifest string `json:"proofManifest"`
}

type Retention struct {
	RedactionClass   string `json:"redactionClass"`
	MaxArtifactBytes int64  `json:"maxArtifactBytes"`
}

type Experiment struct {
	FormatVersion string            `json:"formatVersion"`
	ExperimentID  string            `json:"experimentID"`
	Model         Model             `json:"model"`
	Property      Property          `json:"property"`
	Scope         Scope             `json:"scope"`
	Resources     []Resource        `json:"resources"`
	Actions       []Action          `json:"actions"`
	Policies      []Policy          `json:"policies"`
	Faults        []Fault           `json:"faults"`
	Order         []OrderConstraint `json:"order"`
	Checkpoints   []Checkpoint      `json:"checkpoints"`
	Provenance    Provenance        `json:"provenance"`
	Retention     Retention         `json:"retention"`
}

func DecodeExperiment(reader io.Reader, limit int64) (Experiment, error) {
	var experiment Experiment
	if err := decodeStrictJSON(reader, limit, "experiment", &experiment); err != nil {
		return Experiment{}, err
	}
	if err := experiment.Validate(); err != nil {
		return Experiment{}, err
	}
	return experiment, nil
}

func (e Experiment) Validate() error {
	catalog, err := DefaultCatalog()
	if err != nil {
		return fmt.Errorf("load semantic catalog: %w", err)
	}
	catalogHash, err := catalog.Digest()
	if err != nil {
		return fmt.Errorf("digest semantic catalog: %w", err)
	}
	if e.FormatVersion != FormatVersion {
		return fmt.Errorf("unsupported experiment format version %q", e.FormatVersion)
	}
	if e.ExperimentID == "" {
		return errors.New("experiment ID is required")
	}
	if len(e.Model.Modules) == 0 || e.Model.SourceRevision == "" || e.Model.LeanVersion == "" {
		return errors.New("complete model provenance is required")
	}
	if !validHash(e.Model.SemanticHash) {
		return errors.New("model semantic hash must be a sha256 digest")
	}
	if e.Model.CatalogHash != catalogHash {
		return fmt.Errorf("model catalog hash %q does not match authoritative catalog %q", e.Model.CatalogHash, catalogHash)
	}
	moduleIDs := make(map[string]struct{}, len(catalog.Modules))
	for _, module := range catalog.Modules {
		moduleIDs[module.Identifier] = struct{}{}
	}
	for _, module := range e.Model.Modules {
		if _, known := moduleIDs[module]; !known {
			return fmt.Errorf("model references unknown module %q", module)
		}
	}
	if e.Property.Identifier == "" || !validHash(e.Property.StatementHash) {
		return errors.New("complete property provenance is required")
	}
	propertyIDs := make(map[string]PropertyDeclaration, len(catalog.Properties))
	for _, property := range catalog.Properties {
		propertyIDs[property.Identifier] = property
	}
	propertyDeclaration, known := propertyIDs[e.Property.Identifier]
	if !known {
		return fmt.Errorf("unknown property %q", e.Property.Identifier)
	}
	if e.Property.StatementHash != propertyDeclaration.StatementHash {
		return fmt.Errorf("property %q statement hash does not match semantic catalog", e.Property.Identifier)
	}
	if e.Property.Claim != "implementation-conformance" {
		return fmt.Errorf("unknown requested claim %q", e.Property.Claim)
	}
	if e.Scope.Bounds.MaxDepth <= 0 || e.Scope.Bounds.MaxResults <= 0 || e.Scope.Strategy == "" {
		return errors.New("positive exploration bounds and strategy are required")
	}
	for _, assumption := range e.Scope.Assumptions {
		if assumption.Identifier == "" || !validHash(assumption.StatementHash) {
			return errors.New("every assumption requires an identifier and sha256 statement hash")
		}
	}
	if len(e.Resources) == 0 || len(e.Actions) == 0 || len(e.Checkpoints) == 0 {
		return errors.New("resources, actions, and checkpoints are required")
	}
	entityKinds := make(map[string]struct{}, len(catalog.Entities))
	for _, entity := range catalog.Entities {
		entityKinds[entity.Identifier] = struct{}{}
	}
	resourceIDs := make(map[string]struct{}, len(e.Resources))
	for _, resource := range e.Resources {
		if resource.Identifier == "" || resource.Kind == "" {
			return errors.New("resource identifier and kind are required")
		}
		if _, duplicate := resourceIDs[resource.Identifier]; duplicate {
			return fmt.Errorf("duplicate resource %q", resource.Identifier)
		}
		resourceIDs[resource.Identifier] = struct{}{}
		if _, known := entityKinds[resource.Kind]; !known {
			return fmt.Errorf("unknown resource kind %q", resource.Kind)
		}
	}

	observationIDs := make(map[string]struct{}, len(catalog.Observations))
	for _, observation := range catalog.Observations {
		observationIDs[observation.Identifier] = struct{}{}
	}
	checkpointIDs := make(map[string]struct{}, len(e.Checkpoints))
	for _, checkpoint := range e.Checkpoints {
		if checkpoint.Identifier == "" || checkpoint.Observation == "" {
			return errors.New("checkpoint identifier and observation are required")
		}
		if _, known := observationIDs[checkpoint.Observation]; !known {
			return fmt.Errorf("checkpoint %q references unknown observation %q", checkpoint.Identifier, checkpoint.Observation)
		}
		if checkpoint.Ordering != "causal" && checkpoint.Ordering != "source-sequence" && checkpoint.Ordering != "none" {
			return fmt.Errorf("unknown checkpoint ordering %q", checkpoint.Ordering)
		}
		if checkpoint.OmissionPolicy != "required" && checkpoint.OmissionPolicy != "optional" {
			return fmt.Errorf("unknown omission policy %q", checkpoint.OmissionPolicy)
		}
		if _, duplicate := checkpointIDs[checkpoint.Identifier]; duplicate {
			return fmt.Errorf("duplicate checkpoint %q", checkpoint.Identifier)
		}
		checkpointIDs[checkpoint.Identifier] = struct{}{}
	}

	typeIDs := make(map[string]struct{}, len(catalog.Types))
	for _, declaration := range catalog.Types {
		typeIDs[declaration.Identifier] = struct{}{}
	}
	actionIDs := make(map[string]struct{}, len(e.Actions))
	bindingSymbols := make(map[string]struct{})
	for _, action := range e.Actions {
		if action.Identifier == "" {
			return errors.New("action identifier is required")
		}
		if _, duplicate := actionIDs[action.Identifier]; duplicate {
			return fmt.Errorf("duplicate action %q", action.Identifier)
		}
		actionIDs[action.Identifier] = struct{}{}
		declaration, known := catalog.Action(action.Kind)
		if !known {
			return fmt.Errorf("unknown action kind %q", action.Kind)
		}
		requiredCapabilities := make(map[string]struct{}, len(declaration.RequiredCapabilities))
		for _, capability := range declaration.RequiredCapabilities {
			requiredCapabilities[string(capability)] = struct{}{}
		}
		seenCapabilities := make(map[string]struct{}, len(action.RequiredCapabilities))
		for _, capability := range action.RequiredCapabilities {
			if !catalog.HasCapability(CapabilityID(capability)) {
				return fmt.Errorf("unknown capability %q", capability)
			}
			if _, duplicate := seenCapabilities[capability]; duplicate {
				return fmt.Errorf("action %q has duplicate capability %q", action.Identifier, capability)
			}
			seenCapabilities[capability] = struct{}{}
			if _, declared := requiredCapabilities[capability]; !declared {
				return fmt.Errorf("action %q uses undeclared capability %q", action.Identifier, capability)
			}
			delete(requiredCapabilities, capability)
		}
		if len(requiredCapabilities) != 0 {
			return fmt.Errorf("action %q omits required capabilities", action.Identifier)
		}
		if err := validateActionArguments(action, declaration); err != nil {
			return fmt.Errorf("action %q: %w", action.Identifier, err)
		}
		if err := validateActionResponse(action); err != nil {
			return fmt.Errorf("action %q: %w", action.Identifier, err)
		}
		projections := make(map[string]ProjectionDeclaration, len(declaration.Projections))
		for _, projection := range declaration.Projections {
			projections[projection.Name] = projection
		}
		for _, binding := range action.Bindings {
			if binding.Symbol == "" || binding.Type == "" || binding.Projection == "" {
				return fmt.Errorf("action %q has an incomplete binding", action.Identifier)
			}
			if _, known := typeIDs[binding.Type]; !known {
				return fmt.Errorf("action %q binding %q references unknown type %q", action.Identifier, binding.Symbol, binding.Type)
			}
			projection, known := projections[binding.Projection]
			if !known {
				return fmt.Errorf("action %q binding %q references unknown projection %q", action.Identifier, binding.Symbol, binding.Projection)
			}
			if binding.Type != projection.Type {
				return fmt.Errorf("action %q binding %q has type %q, expected %q", action.Identifier, binding.Symbol, binding.Type, projection.Type)
			}
			if _, duplicate := bindingSymbols[binding.Symbol]; duplicate {
				return fmt.Errorf("duplicate binding symbol %q", binding.Symbol)
			}
			bindingSymbols[binding.Symbol] = struct{}{}
		}
		for _, checkpoint := range []string{action.PreCheckpoint, action.PostCheckpoint} {
			if checkpoint != "" {
				if _, exists := checkpointIDs[checkpoint]; !exists {
					return fmt.Errorf("action %q references unknown checkpoint %q", action.Identifier, checkpoint)
				}
			}
		}
	}
	if err := validatePoliciesAndFaults(e, catalog, actionIDs, resourceIDs); err != nil {
		return err
	}
	if err := validateBoundSymbols(e, bindingSymbols); err != nil {
		return err
	}
	if err := validateOrder(e.Order, actionIDs); err != nil {
		return err
	}
	if e.Provenance.Kind != "proof" && e.Provenance.Kind != "bounded-exploration" &&
		e.Provenance.Kind != "counterexample" && e.Provenance.Kind != "curated-trace" {
		return fmt.Errorf("unknown provenance kind %q", e.Provenance.Kind)
	}
	if e.Provenance.ProofManifest == "" {
		return errors.New("proof manifest is required")
	}
	if e.Retention.RedactionClass != "semantic-only" || e.Retention.MaxArtifactBytes <= 0 {
		return errors.New("bounded semantic-only retention is required")
	}
	return nil
}

func validateActionResponse(action Action) error {
	switch action.EffectiveResponseMode() {
	case ResponseSynchronous, ResponseAsynchronous, ResponseDeferred, ResponseFailure:
		if action.MaxBlockNanos != 0 {
			return errors.New("only blocking responses accept maxBlockNanos")
		}
	case ResponseBlocking:
		if action.MaxBlockNanos <= 0 {
			return errors.New("blocking response requires a positive maxBlockNanos")
		}
	default:
		return fmt.Errorf("unknown response mode %q", action.ResponseMode)
	}
	return nil
}

func validateActionArguments(action Action, declaration ActionDeclaration) error {
	if err := validateNamedValues(action.Arguments, 0); err != nil {
		return err
	}
	parameters := make(map[string]ParameterDeclaration, len(declaration.Parameters))
	for _, parameter := range declaration.Parameters {
		parameters[parameter.Name] = parameter
	}
	for _, argument := range action.Arguments {
		parameter, known := parameters[argument.Name]
		if !known {
			return fmt.Errorf("unknown argument %q", argument.Name)
		}
		if argument.Value.semanticType() != parameter.Type {
			return fmt.Errorf("argument %q has type %q, expected %q", argument.Name, argument.Value.Type, parameter.Type)
		}
		delete(parameters, argument.Name)
	}
	for _, parameter := range parameters {
		if parameter.Required {
			return fmt.Errorf("required argument %q is missing", parameter.Name)
		}
	}
	return nil
}

func validatePoliciesAndFaults(
	e Experiment,
	catalog Catalog,
	actionIDs map[string]struct{},
	resourceIDs map[string]struct{},
) error {
	policyKinds := make(map[string]struct{}, len(catalog.Policies))
	for _, policy := range catalog.Policies {
		policyKinds[policy.Identifier] = struct{}{}
	}
	policyIDs := make(map[string]struct{}, len(e.Policies))
	for _, policy := range e.Policies {
		if policy.Identifier == "" {
			return errors.New("policy identifier is required")
		}
		if _, duplicate := policyIDs[policy.Identifier]; duplicate {
			return fmt.Errorf("duplicate policy %q", policy.Identifier)
		}
		policyIDs[policy.Identifier] = struct{}{}
		if _, known := policyKinds[policy.Kind]; !known {
			return fmt.Errorf("unknown policy kind %q", policy.Kind)
		}
		if len(policy.Scope) == 0 {
			return fmt.Errorf("policy %q requires an action scope", policy.Identifier)
		}
		scopedActions := make(map[string]struct{}, len(policy.Scope))
		for _, action := range policy.Scope {
			if _, known := actionIDs[action]; !known {
				return fmt.Errorf("policy %q references unknown action %q", policy.Identifier, action)
			}
			if _, duplicate := scopedActions[action]; duplicate {
				return fmt.Errorf("policy %q has duplicate scoped action %q", policy.Identifier, action)
			}
			scopedActions[action] = struct{}{}
		}
		if err := validateNamedValues(policy.Arguments, 0); err != nil {
			return fmt.Errorf("policy %q: %w", policy.Identifier, err)
		}
	}
	faultKinds := make(map[string]FaultDeclaration, len(catalog.Faults))
	for _, fault := range catalog.Faults {
		faultKinds[fault.Identifier] = fault
	}
	faultIDs := make(map[string]struct{}, len(e.Faults))
	for _, fault := range e.Faults {
		if fault.Identifier == "" {
			return errors.New("fault identifier is required")
		}
		if _, duplicate := faultIDs[fault.Identifier]; duplicate {
			return fmt.Errorf("duplicate fault %q", fault.Identifier)
		}
		faultIDs[fault.Identifier] = struct{}{}
		declaration, known := faultKinds[fault.Kind]
		if !known {
			return fmt.Errorf("unknown fault kind %q", fault.Kind)
		}
		if fault.Policy != "" {
			if _, known := policyIDs[fault.Policy]; !known {
				return fmt.Errorf("fault %q references unknown policy %q", fault.Identifier, fault.Policy)
			}
		}
		if fault.SafetyClass != declaration.SafetyClass {
			return fmt.Errorf("fault %q safety class does not match catalog", fault.Identifier)
		}
		if len(fault.Scope.Resources) == 0 {
			return fmt.Errorf("fault %q requires an isolation resource scope", fault.Identifier)
		}
		seenResources := make(map[string]struct{}, len(fault.Scope.Resources))
		for _, resource := range fault.Scope.Resources {
			if _, known := resourceIDs[resource]; !known {
				return fmt.Errorf("fault %q references unknown resource %q", fault.Identifier, resource)
			}
			if _, duplicate := seenResources[resource]; duplicate {
				return fmt.Errorf("fault %q has duplicate resource %q", fault.Identifier, resource)
			}
			seenResources[resource] = struct{}{}
		}
		if fault.Occurrence.First <= 0 || fault.Occurrence.Count <= 0 {
			return fmt.Errorf("fault %q requires a positive bounded occurrence", fault.Identifier)
		}
		if _, known := actionIDs[fault.Interval.StartAction]; !known {
			return fmt.Errorf("fault %q interval starts at unknown action %q", fault.Identifier, fault.Interval.StartAction)
		}
		if _, known := actionIDs[fault.Interval.StopAction]; !known {
			return fmt.Errorf("fault %q interval stops at unknown action %q", fault.Identifier, fault.Interval.StopAction)
		}
		if err := validateNamedValues(fault.Arguments, 0); err != nil {
			return fmt.Errorf("fault %q: %w", fault.Identifier, err)
		}
		required := make(map[string]struct{}, len(declaration.RequiredCapabilities))
		for _, capability := range declaration.RequiredCapabilities {
			required[string(capability)] = struct{}{}
		}
		seenCapabilities := make(map[string]struct{}, len(fault.RequiredCapabilities))
		for _, capability := range fault.RequiredCapabilities {
			if !catalog.HasCapability(CapabilityID(capability)) {
				return fmt.Errorf("fault %q references unknown capability %q", fault.Identifier, capability)
			}
			if _, duplicate := seenCapabilities[capability]; duplicate {
				return fmt.Errorf("fault %q has duplicate capability %q", fault.Identifier, capability)
			}
			seenCapabilities[capability] = struct{}{}
			if _, declared := required[capability]; !declared {
				return fmt.Errorf("fault %q uses undeclared capability %q", fault.Identifier, capability)
			}
			delete(required, capability)
		}
		if len(required) != 0 {
			return fmt.Errorf("fault %q omits required capabilities", fault.Identifier)
		}
	}
	return nil
}

func validateBoundSymbols(e Experiment, bindings map[string]struct{}) error {
	var values [][]NamedValue
	for _, action := range e.Actions {
		values = append(values, action.Arguments)
	}
	for _, policy := range e.Policies {
		values = append(values, policy.Arguments)
	}
	for _, fault := range e.Faults {
		values = append(values, fault.Arguments)
	}
	for _, group := range values {
		for _, symbol := range referencedSymbols(group) {
			if _, bound := bindings[symbol]; !bound {
				return fmt.Errorf("unbound symbol %q", symbol)
			}
		}
	}
	return nil
}

func validateOrder(order []OrderConstraint, actionIDs map[string]struct{}) error {
	edges := make(map[string][]string, len(actionIDs))
	seen := make(map[OrderConstraint]struct{}, len(order))
	for _, constraint := range order {
		if _, known := actionIDs[constraint.Before]; !known {
			return fmt.Errorf("order references unknown action %q", constraint.Before)
		}
		if _, known := actionIDs[constraint.After]; !known {
			return fmt.Errorf("order references unknown action %q", constraint.After)
		}
		if constraint.Before == constraint.After {
			return fmt.Errorf("order action %q cannot precede itself", constraint.Before)
		}
		switch constraint.Relation {
		case OrderUser, OrderSemantic, OrderSameSource, OrderRuntimeCausal:
		default:
			return fmt.Errorf("unknown order relation %q", constraint.Relation)
		}
		if _, duplicate := seen[constraint]; duplicate {
			return fmt.Errorf("duplicate order constraint %q before %q", constraint.Before, constraint.After)
		}
		seen[constraint] = struct{}{}
		edges[constraint.Before] = append(edges[constraint.Before], constraint.After)
	}
	visiting := make(map[string]bool, len(actionIDs))
	visited := make(map[string]bool, len(actionIDs))
	var visit func(string) bool
	visit = func(action string) bool {
		if visiting[action] {
			return false
		}
		if visited[action] {
			return true
		}
		visiting[action] = true
		for _, next := range edges[action] {
			if !visit(next) {
				return false
			}
		}
		visiting[action] = false
		visited[action] = true
		return true
	}
	for action := range actionIDs {
		if !visit(action) {
			return errors.New("order cycle detected")
		}
	}
	return nil
}

func sensitiveField(field string) bool {
	normalized := strings.ToLower(field)
	for _, fragment := range []string{"authorization", "credential", "header", "password", "payload", "secret", "token"} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func validHash(value string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, prefix))
	return err == nil
}

func (e Experiment) CanonicalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("encode canonical experiment: %w", err)
	}
	return encoded, nil
}

func (e Experiment) Digest() (string, error) {
	encoded, err := e.CanonicalJSON()
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
