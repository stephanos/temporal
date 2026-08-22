package checker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

const (
	FirstOrderViewFormatVersion     = "umpire3/first-order-view/v2"
	MaxFirstOrderSymbolicDepth      = 1024
	MaxFirstOrderConcreteStateLimit = 65536
)

type FirstOrderSortKind string

const (
	FirstOrderSortEnum          FirstOrderSortKind = "enum"
	FirstOrderSortUninterpreted FirstOrderSortKind = "uninterpreted"
)

type FirstOrderTermKind string

const (
	FirstOrderTermField FirstOrderTermKind = "field"
	FirstOrderTermValue FirstOrderTermKind = "value"
)

type FirstOrderFormulaKind string

const (
	FirstOrderFormulaTrue  FirstOrderFormulaKind = "true"
	FirstOrderFormulaEqual FirstOrderFormulaKind = "equal"
	FirstOrderFormulaNot   FirstOrderFormulaKind = "not"
	FirstOrderFormulaAll   FirstOrderFormulaKind = "all"
	FirstOrderFormulaAny   FirstOrderFormulaKind = "any"
)

type FirstOrderSort struct {
	Identifier  string             `json:"identifier"`
	Kind        FirstOrderSortKind `json:"kind"`
	Values      []string           `json:"values"`
	Cardinality int                `json:"cardinality,omitempty"`
}

type FirstOrderField struct {
	Identifier string `json:"identifier"`
	Sort       string `json:"sort"`
}

type FirstOrderTerm struct {
	Kind  FirstOrderTermKind `json:"kind"`
	Field string             `json:"field,omitempty"`
	Sort  string             `json:"sort,omitempty"`
	Value string             `json:"value,omitempty"`
}

type FirstOrderFormula struct {
	Kind     FirstOrderFormulaKind `json:"kind"`
	Left     *FirstOrderTerm       `json:"left,omitempty"`
	Right    *FirstOrderTerm       `json:"right,omitempty"`
	Operand  *FirstOrderFormula    `json:"operand,omitempty"`
	Operands []FirstOrderFormula   `json:"operands,omitempty"`
}

type FirstOrderUpdate struct {
	Field string         `json:"field"`
	Value FirstOrderTerm `json:"value"`
}

type FirstOrderAction struct {
	Identifier string             `json:"identifier"`
	Guard      FirstOrderFormula  `json:"guard"`
	Updates    []FirstOrderUpdate `json:"updates"`
}

type FirstOrderBinding struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type FirstOrderState struct {
	Fields []FirstOrderBinding `json:"fields"`
}

type FirstOrderRelation struct {
	Declaration string     `json:"declaration"`
	Axioms      []string   `json:"axioms"`
	TrustBadge  TrustBadge `json:"trustBadge"`
}

type FirstOrderBounds struct {
	SymbolicDepth      int `json:"symbolicDepth"`
	ConcreteStateLimit int `json:"concreteStateLimit"`
}

type FirstOrderResource struct {
	Identifier string     `json:"identifier"`
	Kind       EntityKind `json:"kind"`
}

type FirstOrderOracle struct {
	ResultClass ResultClass       `json:"resultClass"`
	TrustBadge  TrustBadge        `json:"trustBadge"`
	States      []FirstOrderState `json:"states"`
}

type FirstOrderView struct {
	FormatVersion    string               `json:"formatVersion"`
	Target           TargetID             `json:"target"`
	Property         PropertyID           `json:"property"`
	World            string               `json:"world"`
	Variant          string               `json:"variant"`
	SemanticHash     string               `json:"semanticHash"`
	CanonicalModel   string               `json:"canonicalModel"`
	Resources        []FirstOrderResource `json:"resources"`
	LiveOnlyActions  []ActionKind         `json:"liveOnlyActions"`
	ActivatingFaults []FaultKind          `json:"activatingFaults"`
	Relation         FirstOrderRelation   `json:"relation"`
	Bounds           FirstOrderBounds     `json:"bounds"`
	Sorts            []FirstOrderSort     `json:"sorts"`
	StateFields      []FirstOrderField    `json:"stateFields"`
	Initial          FirstOrderFormula    `json:"initial"`
	Actions          []FirstOrderAction   `json:"actions"`
	Invariant        FirstOrderFormula    `json:"invariant"`
	Oracle           FirstOrderOracle     `json:"oracle"`
}

func DecodeFirstOrderView(reader io.Reader, limit int64) (FirstOrderView, error) {
	var view FirstOrderView
	if err := decodeStrictJSON(reader, limit, "first-order view", &view); err != nil {
		return FirstOrderView{}, err
	}
	if err := view.Validate(); err != nil {
		return FirstOrderView{}, err
	}
	return view, nil
}

func (v FirstOrderView) Validate() error {
	if v.FormatVersion != FirstOrderViewFormatVersion || v.Target == "" || v.Property == "" ||
		v.World == "" || v.Variant == "" || !validHash(v.SemanticHash) || v.CanonicalModel == "" ||
		len(v.Resources) == 0 || v.LiveOnlyActions == nil || v.ActivatingFaults == nil {
		return errors.New("complete first-order view identity and provenance are required")
	}
	if err := validateFirstOrderTarget(v.Target, v.Property); err != nil {
		return err
	}
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	entities := make(map[EntityKind]struct{}, len(catalog.Entities))
	for _, entity := range catalog.Entities {
		entities[EntityKind(entity.Identifier)] = struct{}{}
	}
	resourceIdentifiers := make(map[string]struct{}, len(v.Resources))
	for _, resource := range v.Resources {
		if resource.Identifier == "" {
			return errors.New("first-order resource identifier is required")
		}
		if _, known := entities[resource.Kind]; !known {
			return fmt.Errorf("first-order resource %q has unknown entity kind %q", resource.Identifier, resource.Kind)
		}
		if _, duplicate := resourceIdentifiers[resource.Identifier]; duplicate {
			return fmt.Errorf("duplicate first-order resource %q", resource.Identifier)
		}
		resourceIdentifiers[resource.Identifier] = struct{}{}
	}
	if err := v.Relation.validate(); err != nil {
		return err
	}
	if v.Bounds.SymbolicDepth <= 0 || v.Bounds.SymbolicDepth > MaxFirstOrderSymbolicDepth ||
		v.Bounds.ConcreteStateLimit <= 0 ||
		v.Bounds.ConcreteStateLimit > MaxFirstOrderConcreteStateLimit {
		return errors.New("bounded first-order symbolic depth and concrete state limit are required")
	}

	sorts := make(map[string]FirstOrderSort, len(v.Sorts))
	for _, sort := range v.Sorts {
		if err := sort.validate(); err != nil {
			return err
		}
		if _, duplicate := sorts[sort.Identifier]; duplicate {
			return fmt.Errorf("duplicate first-order sort %q", sort.Identifier)
		}
		sorts[sort.Identifier] = sort
	}
	if len(sorts) == 0 {
		return errors.New("first-order view requires at least one sort")
	}

	fields := make(map[string]FirstOrderField, len(v.StateFields))
	for _, field := range v.StateFields {
		if field.Identifier == "" || field.Sort == "" {
			return errors.New("complete first-order state field is required")
		}
		if _, known := sorts[field.Sort]; !known {
			return fmt.Errorf("state field %q references unknown sort %q", field.Identifier, field.Sort)
		}
		if _, duplicate := fields[field.Identifier]; duplicate {
			return fmt.Errorf("duplicate first-order state field %q", field.Identifier)
		}
		fields[field.Identifier] = field
	}
	if len(fields) == 0 {
		return errors.New("first-order view requires at least one state field")
	}
	domainSize := 1
	for _, field := range v.StateFields {
		sort := sorts[field.Sort]
		size := len(sort.Values)
		if sort.Kind == FirstOrderSortUninterpreted {
			size = sort.Cardinality
		}
		if domainSize > v.Bounds.ConcreteStateLimit/size {
			return fmt.Errorf("first-order state domain exceeds concrete state limit %d",
				v.Bounds.ConcreteStateLimit)
		}
		domainSize *= size
	}

	if err := validateFirstOrderFormula(v.Initial, fields, sorts); err != nil {
		return fmt.Errorf("validate first-order initial formula: %w", err)
	}
	if err := validateFirstOrderFormula(v.Invariant, fields, sorts); err != nil {
		return fmt.Errorf("validate first-order invariant formula: %w", err)
	}
	actions := make(map[string]struct{}, len(v.Actions))
	for _, action := range v.Actions {
		if action.Identifier == "" {
			return errors.New("first-order action identifier is required")
		}
		if _, duplicate := actions[action.Identifier]; duplicate {
			return fmt.Errorf("duplicate first-order action %q", action.Identifier)
		}
		actions[action.Identifier] = struct{}{}
		if err := validateFirstOrderFormula(action.Guard, fields, sorts); err != nil {
			return fmt.Errorf("validate first-order action %q guard: %w", action.Identifier, err)
		}
		updated := make(map[string]struct{}, len(action.Updates))
		for _, update := range action.Updates {
			field, known := fields[update.Field]
			if !known {
				return fmt.Errorf("action %q updates unknown state field %q", action.Identifier, update.Field)
			}
			if _, duplicate := updated[update.Field]; duplicate {
				return fmt.Errorf("action %q updates state field %q more than once", action.Identifier, update.Field)
			}
			updated[update.Field] = struct{}{}
			sort, err := validateFirstOrderTerm(update.Value, fields, sorts)
			if err != nil {
				return fmt.Errorf("validate first-order action %q update %q: %w", action.Identifier, update.Field, err)
			}
			if sort != field.Sort {
				return fmt.Errorf("action %q assigns sort %q to field %q of sort %q",
					action.Identifier, sort, update.Field, field.Sort)
			}
		}
	}
	if len(actions) == 0 {
		return errors.New("first-order view requires at least one action")
	}
	liveOnly := make(map[ActionKind]struct{}, len(v.LiveOnlyActions))
	for _, action := range v.LiveOnlyActions {
		if _, known := catalog.Action(string(action)); !known {
			return fmt.Errorf("unknown live-only first-order action %q", action)
		}
		if _, modeled := actions[string(action)]; modeled {
			return fmt.Errorf("first-order action %q cannot also be live-only", action)
		}
		if _, duplicate := liveOnly[action]; duplicate {
			return fmt.Errorf("duplicate live-only first-order action %q", action)
		}
		liveOnly[action] = struct{}{}
	}
	faults := make(map[FaultKind]struct{}, len(catalog.Faults))
	for _, fault := range catalog.Faults {
		faults[FaultKind(fault.Identifier)] = struct{}{}
	}
	seenFaults := make(map[FaultKind]struct{}, len(v.ActivatingFaults))
	for _, fault := range v.ActivatingFaults {
		if _, known := faults[fault]; !known {
			return fmt.Errorf("unknown first-order activating fault %q", fault)
		}
		if _, duplicate := seenFaults[fault]; duplicate {
			return fmt.Errorf("duplicate first-order activating fault %q", fault)
		}
		seenFaults[fault] = struct{}{}
	}
	return v.Oracle.validate(v.StateFields, sorts)
}

func (v FirstOrderView) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func (s FirstOrderSort) validate() error {
	if s.Identifier == "" {
		return errors.New("first-order sort identifier is required")
	}
	values := make(map[string]struct{}, len(s.Values))
	for _, value := range s.Values {
		if value == "" {
			return fmt.Errorf("first-order sort %q has an empty value", s.Identifier)
		}
		if _, duplicate := values[value]; duplicate {
			return fmt.Errorf("first-order sort %q has duplicate value %q", s.Identifier, value)
		}
		values[value] = struct{}{}
	}
	switch s.Kind {
	case FirstOrderSortEnum:
		if len(s.Values) == 0 || s.Cardinality != 0 {
			return fmt.Errorf("enum sort %q requires values and no cardinality", s.Identifier)
		}
	case FirstOrderSortUninterpreted:
		if len(s.Values) != 0 || s.Cardinality <= 0 {
			return fmt.Errorf("uninterpreted sort %q requires a positive concrete cardinality and no values", s.Identifier)
		}
	default:
		return fmt.Errorf("first-order sort %q has unknown kind %q", s.Identifier, s.Kind)
	}
	return nil
}

func (r FirstOrderRelation) validate() error {
	if r.Declaration == "" {
		return errors.New("first-order relation declaration is required")
	}
	seen := make(map[string]struct{}, len(r.Axioms))
	for _, axiom := range r.Axioms {
		if axiom == "" {
			return errors.New("first-order relation axiom cannot be empty")
		}
		if _, duplicate := seen[axiom]; duplicate {
			return fmt.Errorf("first-order relation has duplicate axiom %q", axiom)
		}
		seen[axiom] = struct{}{}
	}
	if len(r.Axioms) == 0 {
		if r.TrustBadge != TrustBadgeKernel {
			return errors.New("axiom-free first-order relation requires kernel trust")
		}
		return nil
	}
	if r.TrustBadge != TrustBadgeKernelWithDeclaredAxioms {
		return errors.New("relation with axioms requires kernel-with-declared-axioms trust")
	}
	return nil
}

func (o FirstOrderOracle) validate(fields []FirstOrderField, sorts map[string]FirstOrderSort) error {
	if o.ResultClass != ResultClassFiniteExhaustive {
		return errors.New("oracle result class must be finite-exhaustive")
	}
	if o.TrustBadge != TrustBadgeCheckedCertificate {
		return errors.New("first-order oracle requires checked-certificate trust")
	}
	if len(o.States) == 0 {
		return errors.New("first-order oracle requires reachable states")
	}
	states := make(map[string]struct{}, len(o.States))
	for index, state := range o.States {
		bindings := make(map[string]string, len(state.Fields))
		for _, binding := range state.Fields {
			field, known := findFirstOrderField(fields, binding.Field)
			if !known {
				return fmt.Errorf("oracle state %d references unknown state field %q", index, binding.Field)
			}
			if _, duplicate := bindings[binding.Field]; duplicate {
				return fmt.Errorf("oracle state %d has duplicate state field %q", index, binding.Field)
			}
			sort := sorts[field.Sort]
			if !firstOrderSortContains(sort, binding.Value) {
				return fmt.Errorf("oracle state %d has unknown value %q for sort %q", index, binding.Value, field.Sort)
			}
			bindings[binding.Field] = binding.Value
		}
		if len(bindings) != len(fields) {
			return fmt.Errorf("oracle state %d binds %d fields; view requires %d", index, len(bindings), len(fields))
		}
		keyParts := make([]string, 0, len(fields))
		for _, field := range fields {
			keyParts = append(keyParts, field.Identifier+"="+bindings[field.Identifier])
		}
		key := strings.Join(keyParts, "\x00")
		if _, duplicate := states[key]; duplicate {
			return fmt.Errorf("oracle has duplicate state %d", index)
		}
		states[key] = struct{}{}
	}
	return nil
}

func validateFirstOrderTarget(target TargetID, property PropertyID) error {
	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}
	for _, candidate := range catalog.Targets {
		if TargetID(candidate.Identifier) == target && slices.Contains(candidate.Properties, string(property)) {
			return nil
		}
	}
	return fmt.Errorf("unknown first-order target/property %q/%q", target, property)
}

func validateFirstOrderFormula(
	formula FirstOrderFormula,
	fields map[string]FirstOrderField,
	sorts map[string]FirstOrderSort,
) error {
	switch formula.Kind {
	case FirstOrderFormulaTrue:
		if formula.Left != nil || formula.Right != nil || formula.Operand != nil || len(formula.Operands) != 0 {
			return errors.New("true formula cannot have operands")
		}
	case FirstOrderFormulaEqual:
		if formula.Left == nil || formula.Right == nil || formula.Operand != nil || len(formula.Operands) != 0 {
			return errors.New("equal formula requires exactly left and right terms")
		}
		leftSort, err := validateFirstOrderTerm(*formula.Left, fields, sorts)
		if err != nil {
			return err
		}
		rightSort, err := validateFirstOrderTerm(*formula.Right, fields, sorts)
		if err != nil {
			return err
		}
		if leftSort != rightSort {
			return fmt.Errorf("cannot compare first-order sorts %q and %q", leftSort, rightSort)
		}
	case FirstOrderFormulaNot:
		if formula.Left != nil || formula.Right != nil || formula.Operand == nil || len(formula.Operands) != 0 {
			return errors.New("not formula requires exactly one operand")
		}
		return validateFirstOrderFormula(*formula.Operand, fields, sorts)
	case FirstOrderFormulaAll, FirstOrderFormulaAny:
		if formula.Left != nil || formula.Right != nil || formula.Operand != nil || len(formula.Operands) == 0 {
			return fmt.Errorf("%s formula requires an operand list", formula.Kind)
		}
		for _, operand := range formula.Operands {
			if err := validateFirstOrderFormula(operand, fields, sorts); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown first-order formula kind %q", formula.Kind)
	}
	return nil
}

func validateFirstOrderTerm(
	term FirstOrderTerm,
	fields map[string]FirstOrderField,
	sorts map[string]FirstOrderSort,
) (string, error) {
	switch term.Kind {
	case FirstOrderTermField:
		if term.Field == "" || term.Sort != "" || term.Value != "" {
			return "", errors.New("first-order field term requires only a field")
		}
		field, known := fields[term.Field]
		if !known {
			return "", fmt.Errorf("unknown state field %q", term.Field)
		}
		return field.Sort, nil
	case FirstOrderTermValue:
		if term.Field != "" || term.Sort == "" || term.Value == "" {
			return "", errors.New("first-order value term requires only a sort and value")
		}
		sort, known := sorts[term.Sort]
		if !known {
			return "", fmt.Errorf("unknown first-order sort %q", term.Sort)
		}
		if !firstOrderSortContains(sort, term.Value) {
			return "", fmt.Errorf("unknown value %q for sort %q", term.Value, term.Sort)
		}
		return term.Sort, nil
	default:
		return "", fmt.Errorf("unknown first-order term kind %q", term.Kind)
	}
}

func firstOrderSortContains(sort FirstOrderSort, value string) bool {
	if sort.Kind == FirstOrderSortEnum {
		return slices.Contains(sort.Values, value)
	}
	if sort.Kind != FirstOrderSortUninterpreted || !strings.HasPrefix(value, "member-") {
		return false
	}
	index, err := strconv.Atoi(strings.TrimPrefix(value, "member-"))
	return err == nil && index >= 0 && index < sort.Cardinality && value == fmt.Sprintf("member-%d", index)
}

func findFirstOrderField(fields []FirstOrderField, identifier string) (FirstOrderField, bool) {
	for _, field := range fields {
		if field.Identifier == identifier {
			return field, true
		}
	}
	return FirstOrderField{}, false
}
