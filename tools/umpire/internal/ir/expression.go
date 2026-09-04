package ir

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

type ReferenceKind uint8

const (
	SlotReference ReferenceKind = iota + 1
	OutcomeReference
	ObservationReference
	EventReference
	CaptureReference
)

type Reference struct {
	Kind           ReferenceKind
	Entrypoint, ID string
	Field          int32
}
type Binding struct {
	Type      Type
	Available bool
}

type Operator uint8

const (
	Literal Operator = iota + 1
	ReferenceValue
	Project
	IsPresent
	Equals
	Compare
	Not
	All
	Any
)

type Expression struct {
	operator   Operator
	typ        Type
	literal    *umpirespb.Value
	reference  Reference
	children   []*Expression
	path       *Path
	comparison umpirespb.ComparisonOperator
	absent     bool
	key        string
}

func (e *Expression) Operator() Operator                       { return e.operator }
func (e *Expression) Type() Type                               { return e.typ }
func (e *Expression) Literal() *umpirespb.Value                { return proto.CloneOf(e.literal) }
func (e *Expression) Reference() Reference                     { return e.reference }
func (e *Expression) Children() []*Expression                  { return slices.Clone(e.children) }
func (e *Expression) Path() *Path                              { return e.path }
func (e *Expression) Comparison() umpirespb.ComparisonOperator { return e.comparison }
func (e *Expression) MayBeAbsent() bool                        { return e.absent }

type compiler struct {
	catalog *Catalog
	scope   map[Reference]Binding
	budget  budget
}

func (c *Catalog) BindExpression(source *umpirespb.ValueExpression, expected *Type, scope map[Reference]Binding, limits Limits) (*Expression, error) {
	if err := limits.validate(); err != nil {
		return nil, err
	}
	if source == nil || missing(source.Expression) {
		return nil, invalid(Malformed, "expression", "expression is required")
	}
	if expected != nil && !c.owns(*expected) {
		return nil, invalid(TypeMismatch, "expression", "expected type belongs to another catalog")
	}
	binder := compiler{catalog: c, scope: scope, budget: budget{limits: limits}}
	if err := inspectSurface(source.ProtoReflect(), &binder.budget, "expression"); err != nil {
		return nil, err
	}
	return binder.bind(source, expected, nil, false, 1)
}

// BindGuardedExpression compiles an instruction input under the facts implied by its guard.
// Both expressions share the same budget; the guard must be valid before its facts are used.
func (c *Catalog) BindGuardedExpression(guard, source *umpirespb.ValueExpression, expected *Type, scope map[Reference]Binding, limits Limits) (boundGuard, boundValue *Expression, err error) {
	if err := limits.validate(); err != nil {
		return nil, nil, err
	}
	if source == nil || missing(source.Expression) {
		return nil, nil, invalid(Malformed, "expression", "expression is required")
	}
	if expected != nil && !c.owns(*expected) {
		return nil, nil, invalid(TypeMismatch, "expression", "expected type belongs to another catalog")
	}
	binder := compiler{catalog: c, scope: scope, budget: budget{limits: limits}}
	var compiledGuard *Expression
	var facts map[string]bool
	if guard != nil {
		if err := inspectSurface(guard.ProtoReflect(), &binder.budget, "guard"); err != nil {
			return nil, nil, err
		}
		boolean := c.scalarType(umpirespb.SCALAR_KIND_BOOLEAN)
		var err error
		compiledGuard, err = binder.bind(guard, &boolean, nil, false, 1)
		if err != nil {
			return nil, nil, err
		}
		facts, err = binder.presenceFacts(compiledGuard, true)
		if err != nil {
			return nil, nil, err
		}
	}
	if err := inspectSurface(source.ProtoReflect(), &binder.budget, "expression"); err != nil {
		return nil, nil, err
	}
	compiled, err := binder.bind(source, expected, facts, false, 1)
	return compiledGuard, compiled, err
}

func referenceKey(reference Reference) string {
	return fmt.Sprintf("%d:%q:%q:%d", reference.Kind, reference.Entrypoint, reference.ID, reference.Field)
}

func (b *compiler) bind(source *umpirespb.ValueExpression, expected *Type, facts map[string]bool, allowAbsent bool, depth int64) (*Expression, error) {
	if source == nil || missing(source.Expression) {
		return nil, invalid(Malformed, "expression", "missing expression node")
	}
	if err := b.budget.charge(depth, 1, 0, "expression"); err != nil {
		return nil, err
	}
	result, err := b.node(source, expected, facts, depth)
	if err != nil {
		return nil, err
	}
	c := b.catalog
	if result.reference.Kind != 0 {
		reference := result.reference
		if reference.Kind != EventReference && reference.ID == "" {
			return nil, invalid(Malformed, "expression", "reference identity is required")
		}
		binding, ok := b.scope[reference]
		if !ok {
			return nil, invalid(Unknown, "expression", "reference is not declared in this environment")
		}
		if !c.owns(binding.Type) {
			return nil, invalid(TypeMismatch, "expression", "reference type belongs to another catalog")
		}
		result.operator = ReferenceValue
		result.typ = binding.Type
		result.key = referenceKey(reference)
		result.absent = !binding.Available && !facts[result.key]
	}
	if result.typ.opaque {
		return nil, invalid(Unsupported, "expression", "capabilities cannot be inspected")
	}
	if expected != nil && !result.typ.Equal(*expected) {
		return nil, invalid(TypeMismatch, "expression", "expression type does not match expected type")
	}
	if result.absent && !allowAbsent {
		return nil, invalid(Unavailable, "expression", "reference or projection requires an explicit presence guard")
	}
	return result, nil
}

func (b *compiler) node(source *umpirespb.ValueExpression, expected *Type, facts map[string]bool, depth int64) (*Expression, error) {
	switch value := source.Expression.(type) {
	case *umpirespb.ValueExpression_Literal:
		return b.literal(value.Literal, expected, depth)
	case *umpirespb.ValueExpression_Slot, *umpirespb.ValueExpression_Observation, *umpirespb.ValueExpression_Capture, *umpirespb.ValueExpression_Outcome, *umpirespb.ValueExpression_RunEvent:
		reference, err := expressionReference(source)
		return &Expression{reference: reference}, err
	case *umpirespb.ValueExpression_Path:
		return b.project(value.Path, facts, depth)
	case *umpirespb.ValueExpression_Present:
		return b.unary(IsPresent, value.Present.GetOperand(), facts, depth)
	case *umpirespb.ValueExpression_Negation:
		return b.unary(Not, value.Negation.GetOperand(), facts, depth)
	case *umpirespb.ValueExpression_Equals:
		return b.binary(value.Equals.GetLeft(), value.Equals.GetRight(), 0, facts, depth)
	case *umpirespb.ValueExpression_Compare:
		comparison := value.Compare.GetOperator()
		if comparison < umpirespb.COMPARISON_OPERATOR_LESS_THAN || comparison > umpirespb.COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL {
			return nil, invalid(Unknown, "expression", "unknown comparison operator")
		}
		return b.binary(value.Compare.GetLeft(), value.Compare.GetRight(), comparison, facts, depth)
	case *umpirespb.ValueExpression_All:
		return b.logicalNode(All, value.All.GetOperands(), facts, depth)
	case *umpirespb.ValueExpression_Any:
		return b.logicalNode(Any, value.Any.GetOperands(), facts, depth)
	default:
		return nil, invalid(Unsupported, "expression", "unknown expression variant")
	}
}

func expressionReference(source *umpirespb.ValueExpression) (Reference, error) {
	var result Reference
	switch value := source.Expression.(type) {
	case *umpirespb.ValueExpression_Slot:
		result = Reference{Kind: SlotReference, ID: value.Slot.GetSlotId()}
	case *umpirespb.ValueExpression_Observation:
		result = Reference{Kind: ObservationReference, ID: value.Observation.GetObservationId()}
	case *umpirespb.ValueExpression_Capture:
		result = Reference{Kind: CaptureReference, ID: value.Capture.GetCaptureId()}
	case *umpirespb.ValueExpression_Outcome:
		instruction := value.Outcome.GetInstruction()
		result = Reference{Kind: OutcomeReference, Entrypoint: instruction.GetEntrypointId(), ID: instruction.GetInstructionId(), Field: int32(value.Outcome.GetField())}
		if result.Entrypoint == "" || result.Field <= 0 || result.Field > int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_VALUE) {
			return Reference{}, invalid(Malformed, "expression", "invalid outcome reference")
		}
	case *umpirespb.ValueExpression_RunEvent:
		result = Reference{Kind: EventReference, Field: int32(value.RunEvent.GetField())}
		if result.Field <= 0 || result.Field > int32(umpirespb.RUN_EVENT_FIELD_SOURCE_ID) {
			return Reference{}, invalid(Malformed, "expression", "invalid Run Event reference")
		}
	default:
		return Reference{}, invalid(Unsupported, "expression", "not a reference")
	}
	return result, nil
}

func (b *compiler) literal(value *umpirespb.Value, expected *Type, depth int64) (*Expression, error) {
	result := &Expression{operator: Literal}
	var err error
	if expected != nil {
		result.typ = *expected
	} else {
		result.typ, err = b.catalog.literalType(value)
	}
	if err != nil {
		return nil, err
	}
	if err := b.catalog.checkLiteral(value, result.typ, &b.budget, depth); err != nil {
		return nil, err
	}
	result.literal = proto.CloneOf(value)
	return result, nil
}

func (b *compiler) project(value *umpirespb.PathExpression, facts map[string]bool, depth int64) (*Expression, error) {
	operand, err := b.bind(value.GetSource(), nil, facts, true, depth+1)
	if err != nil {
		return nil, err
	}
	path, err := b.catalog.BindPath(operand.typ, value.GetPath(), b.budget.limits)
	if err != nil {
		return nil, err
	}
	if err := b.budget.charge(depth, int64(len(path.steps)), 0, "expression.path"); err != nil {
		return nil, err
	}
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(value.GetPath())
	if err != nil {
		return nil, invalid(Malformed, "expression.path", "path serialization failed")
	}
	result := &Expression{operator: Project, children: []*Expression{operand}, path: path, typ: path.typ}
	if operand.key != "" {
		result.key = operand.key + "/" + string(encoded)
	}
	result.absent = (operand.absent || path.absent) && !facts[result.key]
	if len(path.steps) > 0 && path.steps[len(path.steps)-1].Selector == Presence {
		result.absent = false
	}
	return result, nil
}

func (b *compiler) unary(operator Operator, source *umpirespb.ValueExpression, facts map[string]bool, depth int64) (*Expression, error) {
	boolean := b.catalog.scalarType(umpirespb.SCALAR_KIND_BOOLEAN)
	var expected *Type
	if operator == Not {
		expected = &boolean
	}
	operand, err := b.bind(source, expected, facts, operator == IsPresent, depth+1)
	if err != nil {
		return nil, err
	}
	return &Expression{operator: operator, children: []*Expression{operand}, typ: boolean}, nil
}

func (b *compiler) binary(left, right *umpirespb.ValueExpression, comparison umpirespb.ComparisonOperator, facts map[string]bool, depth int64) (*Expression, error) {
	operands, err := b.pair(left, right, facts, depth)
	if err != nil {
		return nil, err
	}
	operator := Equals
	if comparison != 0 {
		operator = Compare
		if !ordered(operands[0].typ) {
			return nil, invalid(TypeMismatch, "expression", "comparison requires ordered numeric scalars")
		}
	}
	return &Expression{operator: operator, children: operands, comparison: comparison, typ: b.catalog.scalarType(umpirespb.SCALAR_KIND_BOOLEAN)}, nil
}

func (b *compiler) logicalNode(operator Operator, operands []*umpirespb.ValueExpression, facts map[string]bool, depth int64) (*Expression, error) {
	children, err := b.logical(operands, facts, operator == All, depth)
	if err != nil {
		return nil, err
	}
	return &Expression{operator: operator, children: children, typ: b.catalog.scalarType(umpirespb.SCALAR_KIND_BOOLEAN)}, nil
}

func (b *compiler) pair(left, right *umpirespb.ValueExpression, facts map[string]bool, depth int64) ([]*Expression, error) {
	first, second := left, right
	reversed := left.GetLiteral() != nil && right.GetLiteral() == nil
	if reversed {
		first, second = right, left
	}
	a, err := b.bind(first, nil, facts, false, depth+1)
	if err != nil {
		return nil, err
	}
	other, err := b.bind(second, &a.typ, facts, false, depth+1)
	if err != nil {
		return nil, err
	}
	if reversed {
		return []*Expression{other, a}, nil
	}
	return []*Expression{a, other}, nil
}

func (b *compiler) logical(operands []*umpirespb.ValueExpression, facts map[string]bool, continuing bool, depth int64) ([]*Expression, error) {
	if err := b.budget.charge(depth, int64(len(facts)), 0, "expression.presence"); err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(facts))
	maps.Copy(known, facts)
	result := make([]*Expression, 0, len(operands))
	boolean := b.catalog.scalarType(umpirespb.SCALAR_KIND_BOOLEAN)
	for _, operand := range operands {
		item, err := b.bind(operand, &boolean, known, false, depth+1)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
		learned, err := b.presenceFacts(item, continuing)
		if err != nil {
			return nil, err
		}
		if err := b.budget.charge(depth, int64(len(learned)), 0, "expression.presence"); err != nil {
			return nil, err
		}
		maps.Copy(known, learned)
	}
	return result, nil
}

func (b *compiler) presenceFacts(e *Expression, truth bool) (map[string]bool, error) {
	if err := b.budget.charge(1, 1, 0, "expression.presence"); err != nil {
		return nil, err
	}
	switch e.operator {
	case Literal, ReferenceValue, Project, Equals, Compare:
	case IsPresent:
		if e.children[0].key != "" {
			return map[string]bool{e.children[0].key: truth}, nil
		}
	case Not:
		return b.presenceFacts(e.children[0], !truth)
	case All, Any:
		result := map[string]bool{}
		merge := (e.operator == All && truth) || (e.operator == Any && !truth)
		for i, child := range e.children {
			facts, err := b.presenceFacts(child, truth)
			if err != nil {
				return nil, err
			}
			if err := b.budget.charge(1, int64(len(result))+int64(len(facts)), 0, "expression.presence"); err != nil {
				return nil, err
			}
			if merge || i == 0 {
				maps.Copy(result, facts)
			} else {
				intersectFacts(result, facts)
			}
		}
		return result, nil
	default:
		return nil, nil
	}
	return nil, nil
}

func intersectFacts(result, facts map[string]bool) {
	for key, value := range result {
		if other, ok := facts[key]; !ok || other != value {
			delete(result, key)
		}
	}
}

func (c *Catalog) literalType(value *umpirespb.Value) (Type, error) {
	if value == nil || missing(value.Value) {
		return Type{}, invalid(Malformed, "literal", "missing literal")
	}
	switch literal := value.Value.(type) {
	case *umpirespb.Value_Text:
		return c.scalarType(umpirespb.SCALAR_KIND_TEXT), nil
	case *umpirespb.Value_Natural:
		return c.scalarType(umpirespb.SCALAR_KIND_NATURAL), nil
	case *umpirespb.Value_BoolValue:
		return c.scalarType(umpirespb.SCALAR_KIND_BOOLEAN), nil
	case *umpirespb.Value_BytesValue:
		return c.scalarType(umpirespb.SCALAR_KIND_BYTES), nil
	case *umpirespb.Value_MessageValue:
		url := literal.MessageValue.GetTypeUrl()
		name := url[strings.LastIndexByte(url, '/')+1:]
		return c.BindType(&umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Message{Message: &umpirespb.NamedType{ProtobufType: name}}}}})
	default:
		return Type{}, invalid(TypeMismatch, "literal", "numeric, enum, and collection literals require a contextual source type")
	}
}

func ordered(typ Type) bool {
	return typ.cardinality == Singular && (typ.scalar == umpirespb.SCALAR_KIND_NATURAL || typ.scalar >= umpirespb.SCALAR_KIND_INT32 && typ.scalar <= umpirespb.SCALAR_KIND_DOUBLE)
}
