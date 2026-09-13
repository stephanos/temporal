package ir

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ReferenceKind uint8

const (
	SlotReference ReferenceKind = iota + 1
	OutcomeReference
	ObservationReference
	EventReference
	CaptureReference
	// ProjectedValueReference is the value an evidence lift is projecting; it carries no identity.
	ProjectedValueReference
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

// Context is where an expression appears in a Case. Program and Contract share one expression
// language, so which references an expression may use is decided by its context at binding.
type Context uint8

const (
	// ProgramContext is an instruction input or guard.
	ProgramContext Context = iota + 1
	// ContractContext is a Contract transition predicate.
	ContractContext
	// CorrelatedContext is a correlated rule's trigger, response or correlation. The correlated
	// capability admits and evaluates these itself rather than through BindExpression.
	CorrelatedContext
	// EvidenceLiftContext is an evidence-lift rule's guard over the value being projected.
	EvidenceLiftContext
)

// admittedReferences names the Reference arms each context admits.
var admittedReferences = map[Context]map[protoreflect.Name]bool{
	ProgramContext:      {"slot_id": true, "outcome": true, "run": true, "environment_binding_id": true},
	ContractContext:     {"observation_id": true, "run_event": true, "capture_id": true},
	CorrelatedContext:   {"evidence_field_id": true, "correlated_capture": true, "correlated_step": true},
	EvidenceLiftContext: {"projected_value": true},
}

// admitReference rejects a Reference arm its context does not admit, located at path, the path of
// the Reference within the Case.
func admitReference(context Context, path string, arm protoreflect.Name) error {
	if !admittedReferences[context][arm] {
		return invalid(Unknown, path+"."+string(arm), "reference is not admitted in this expression context")
	}
	return nil
}

// AdmitReferences rejects the first reference in source, in evaluation order, that the site's
// context does not admit, at the path BindExpression would report. It is the context check for
// expressions a caller evaluates without binding them.
func AdmitReferences(site Site, source *testpilotspb.Expression) error {
	switch v := source.GetExpression().(type) {
	case *testpilotspb.Expression_Reference:
		message := v.Reference.ProtoReflect()
		if arm := message.WhichOneof(message.Descriptor().Oneofs().ByName("reference")); arm != nil {
			return admitReference(site.Context, site.Path+".reference", arm.Name())
		}
	case *testpilotspb.Expression_Path:
		return AdmitReferences(Site{Context: site.Context, Path: site.Path + ".path.operand"}, v.Path.GetOperand())
	case *testpilotspb.Expression_Present:
		return AdmitReferences(Site{Context: site.Context, Path: site.Path + ".present"}, v.Present.GetOperand())
	case *testpilotspb.Expression_Not:
		return AdmitReferences(Site{Context: site.Context, Path: site.Path + ".not"}, v.Not.GetOperand())
	case *testpilotspb.Expression_Compare:
		if err := AdmitReferences(Site{Context: site.Context, Path: site.Path + ".compare.left"}, v.Compare.GetLeft()); err != nil {
			return err
		}
		return AdmitReferences(Site{Context: site.Context, Path: site.Path + ".compare.right"}, v.Compare.GetRight())
	case *testpilotspb.Expression_All:
		return admitOperands(site, ".all", v.All.GetOperands())
	case *testpilotspb.Expression_Any:
		return admitOperands(site, ".any", v.Any.GetOperands())
	default:
		// A literal reads no reference.
	}
	return nil
}

func admitOperands(site Site, group string, operands []*testpilotspb.Expression) error {
	for index, operand := range operands {
		if err := AdmitReferences(Site{Context: site.Context, Path: fmt.Sprintf("%s%s[%d]", site.Path, group, index)}, operand); err != nil {
			return err
		}
	}
	return nil
}

// Site locates one authored expression: the context it is bound in and its path in the Case, which
// a rejected reference extends to name itself.
type Site struct {
	Context Context
	Path    string
}

type Operator uint8

const (
	Literal Operator = iota + 1
	ReferenceValue
	Project
	IsPresent
	Compare
	Not
	All
	Any
)

type Expression struct {
	bindingWork int64
	operator    Operator
	typ         Type
	literal     *testpilotspb.Value
	reference   Reference
	children    []*Expression
	path        *Path
	comparison  testpilotspb.ComparisonOperator
	absent      bool
	key         string
}

func (e *Expression) BindingWork() int64                          { return e.bindingWork }
func (e *Expression) Operator() Operator                          { return e.operator }
func (e *Expression) Type() Type                                  { return e.typ }
func (e *Expression) Literal() *testpilotspb.Value                { return proto.CloneOf(e.literal) }
func (e *Expression) Reference() Reference                        { return e.reference }
func (e *Expression) Children() []*Expression                     { return slices.Clone(e.children) }
func (e *Expression) Path() *Path                                 { return e.path }
func (e *Expression) Comparison() testpilotspb.ComparisonOperator { return e.comparison }
func (e *Expression) MayBeAbsent() bool                           { return e.absent }

type compiler struct {
	catalog *Catalog
	context Context
	scope   map[Reference]Binding
	budget  budget
}

func (c *Catalog) BindExpression(site Site, source *testpilotspb.Expression, expected *Type, scope map[Reference]Binding, limits Limits) (*Expression, error) {
	if err := c.checkBinding(site, source, expected, limits); err != nil {
		return nil, err
	}
	binder := compiler{catalog: c, context: site.Context, scope: scope, budget: budget{limits: limits}}
	if err := inspectSurface(source.ProtoReflect(), &binder.budget, "expression"); err != nil {
		return nil, err
	}
	result, err := binder.bind(source, site.Path, expected, nil, false, 1)
	if err == nil {
		result.bindingWork = binder.budget.work
	}
	return result, err
}

// BindGuardedExpression compiles an instruction input under the facts implied by its guard.
// Both expressions share the same budget; the guard must be valid before its facts are used.
func (c *Catalog) BindGuardedExpression(guard Condition, site Site, source *testpilotspb.Expression, expected *Type, scope map[Reference]Binding, limits Limits) (boundGuard, boundValue *Expression, err error) {
	var conditions []Condition
	if !isNilMessage(guard.Expression) {
		conditions = []Condition{{Expression: guard.Expression, Path: guard.Path, Matches: true}}
	}
	return c.bindConditionedExpression(conditions, site, source, expected, scope, limits)
}

// Condition is an expression bound in the same context as the value it constrains, and whether it
// held; Path locates it in the Case.
type Condition struct {
	Expression *testpilotspb.Expression
	Path       string
	Matches    bool
}

// BindConditionedExpression preserves each authored expression's depth while sharing guard facts and work.
func (c *Catalog) BindConditionedExpression(conditions []Condition, site Site, source *testpilotspb.Expression, expected *Type, scope map[Reference]Binding, limits Limits) (*Expression, error) {
	_, value, err := c.bindConditionedExpression(conditions, site, source, expected, scope, limits)
	return value, err
}

func (c *Catalog) checkBinding(site Site, source *testpilotspb.Expression, expected *Type, limits Limits) error {
	if err := limits.validate(); err != nil {
		return err
	}
	if admittedReferences[site.Context] == nil {
		return invalid(Malformed, "expression", "expression context is required")
	}
	if !hasExpression(source) {
		return invalid(Malformed, "expression", "expression is required")
	}
	if expected != nil && !c.owns(*expected) {
		return invalid(TypeMismatch, "expression", "expected type belongs to another catalog")
	}
	return nil
}

func (c *Catalog) bindConditionedExpression(conditions []Condition, site Site, source *testpilotspb.Expression, expected *Type, scope map[Reference]Binding, limits Limits) (boundGuard, boundValue *Expression, err error) {
	if err := c.checkBinding(site, source, expected, limits); err != nil {
		return nil, nil, err
	}
	binder := compiler{catalog: c, context: site.Context, scope: scope, budget: budget{limits: limits}}
	var compiledGuard *Expression
	facts := map[string]bool{}
	for _, condition := range conditions {
		guard := condition.Expression
		if err := inspectSurface(guard.ProtoReflect(), &binder.budget, "guard"); err != nil {
			return nil, nil, err
		}
		boolean := c.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN)
		var err error
		compiledGuard, err = binder.bind(guard, condition.Path, &boolean, facts, false, 1)
		if err != nil {
			return nil, nil, err
		}
		learned, err := binder.presenceFacts(compiledGuard, condition.Matches)
		if err != nil {
			return nil, nil, err
		}
		if err := binder.budget.charge(1, int64(len(learned)), 0, "expression.presence"); err != nil {
			return nil, nil, err
		}
		maps.Copy(facts, learned)
	}
	if err := inspectSurface(source.ProtoReflect(), &binder.budget, "expression"); err != nil {
		return nil, nil, err
	}
	compiled, err := binder.bind(source, site.Path, expected, facts, false, 1)
	if err == nil {
		compiled.bindingWork = binder.budget.work
	}
	return compiledGuard, compiled, err
}

func referenceKey(reference Reference) string {
	return fmt.Sprintf("%d:%q:%q:%d", reference.Kind, reference.Entrypoint, reference.ID, reference.Field)
}

func hasExpression(source proto.Message) bool {
	if isNilMessage(source) {
		return false
	}
	message := source.ProtoReflect()
	oneof := message.Descriptor().Oneofs().ByName("expression")
	return oneof != nil && message.WhichOneof(oneof) != nil
}

func isNilMessage(source proto.Message) bool {
	return source == nil || reflect.ValueOf(source).Kind() == reflect.Pointer && reflect.ValueOf(source).IsNil()
}

func expressionVariant(source proto.Message) (protoreflect.Name, protoreflect.Value) {
	message := source.ProtoReflect()
	field := message.WhichOneof(message.Descriptor().Oneofs().ByName("expression"))
	if field == nil {
		return "", protoreflect.Value{}
	}
	return field.Name(), message.Get(field)
}

func messageField(message protoreflect.Message, name protoreflect.Name) protoreflect.Message {
	return message.Get(message.Descriptor().Fields().ByName(name)).Message()
}

func (b *compiler) bind(source proto.Message, path string, expected *Type, facts map[string]bool, allowAbsent bool, depth int64) (*Expression, error) {
	if !hasExpression(source) {
		return nil, invalid(Malformed, "expression", "missing expression node")
	}
	if err := b.budget.charge(depth, 1, 0, "expression"); err != nil {
		return nil, err
	}
	result, err := b.node(source, path, expected, facts, depth)
	if err != nil {
		return nil, err
	}
	c := b.catalog
	if result.reference.Kind != 0 {
		reference := result.reference
		if reference.Kind != EventReference && reference.Kind != ProjectedValueReference && reference.ID == "" {
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

// node binds one Expression located at path. Child paths name the child by its operator, with the
// operand index of all and any, so a rejection stays short enough to locate within 256 bytes.
func (b *compiler) node(source proto.Message, path string, expected *Type, facts map[string]bool, depth int64) (*Expression, error) {
	kind, value := expressionVariant(source)
	switch kind {
	case "literal":
		return b.literal(value.Message().Interface().(*testpilotspb.Value), expected, depth)
	case "reference":
		reference, err := b.reference(value.Message(), path+".reference")
		return &Expression{reference: reference}, err
	case "path":
		return b.project(value.Message(), path+".path", facts, depth)
	case "present":
		return b.unary(IsPresent, messageField(value.Message(), "operand").Interface(), path+".present", facts, depth)
	case "not":
		return b.unary(Not, messageField(value.Message(), "operand").Interface(), path+".not", facts, depth)
	case "compare":
		comparison := testpilotspb.ComparisonOperator(value.Message().Get(value.Message().Descriptor().Fields().ByName("operator")).Enum())
		if comparison < testpilotspb.COMPARISON_OPERATOR_EQUAL || comparison > testpilotspb.COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL {
			return nil, invalid(Unknown, "expression", "unknown comparison operator")
		}
		return b.binary(messageField(value.Message(), "left").Interface(), messageField(value.Message(), "right").Interface(), comparison, path+".compare", facts, depth)
	case "all":
		return b.logicalNode(All, expressionOperands(value.Message()), path+".all", facts, depth)
	case "any":
		return b.logicalNode(Any, expressionOperands(value.Message()), path+".any", facts, depth)
	default:
		return nil, invalid(Unsupported, "expression", "unknown expression variant")
	}
}

// reference resolves a Reference its context admits. A reference outside the context is a static
// rejection located at the reference arm.
func (b *compiler) reference(message protoreflect.Message, path string) (Reference, error) {
	var result Reference
	selected := message.WhichOneof(message.Descriptor().Oneofs().ByName("reference"))
	if selected == nil {
		return Reference{}, invalid(Malformed, "expression", "reference is required")
	}
	if err := admitReference(b.context, path, selected.Name()); err != nil {
		return Reference{}, err
	}
	value := message.Get(selected)
	switch selected.Name() {
	case "slot_id":
		result = Reference{Kind: SlotReference, ID: value.String()}
	case "observation_id":
		result = Reference{Kind: ObservationReference, ID: value.String()}
	case "capture_id":
		result = Reference{Kind: CaptureReference, ID: value.String()}
	case "outcome":
		message := value.Message()
		instruction := messageField(message, "instruction")
		result = Reference{Kind: OutcomeReference, Entrypoint: instruction.Get(instruction.Descriptor().Fields().ByName("entrypoint_id")).String(), ID: instruction.Get(instruction.Descriptor().Fields().ByName("instruction_id")).String(), Field: int32(message.Get(message.Descriptor().Fields().ByName("field")).Enum())}
		if result.Entrypoint == "" || result.Field <= 0 || result.Field > int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE) {
			return Reference{}, invalid(Malformed, "expression", "invalid outcome reference")
		}
	case "run_event":
		message := value.Message()
		result = Reference{Kind: EventReference, Field: int32(message.Get(message.Descriptor().Fields().ByName("field")).Enum())}
		if result.Field <= 0 || result.Field > int32(testpilotspb.RUN_EVENT_FIELD_FAULT_KIND) {
			return Reference{}, invalid(Malformed, "expression", "invalid Run Event reference")
		}
	case "run":
		result = Reference{Kind: EventReference, Field: int32(testpilotspb.RUN_EVENT_FIELD_RUN_ID)}
	case "projected_value":
		result = Reference{Kind: ProjectedValueReference}
	default:
		// An environment binding is admitted only as a whole request assignment value, which
		// Program admission resolves to a literal before binding.
		return Reference{}, invalid(Unsupported, "expression", "unknown expression variant")
	}
	return result, nil
}

func (b *compiler) literal(value *testpilotspb.Value, expected *Type, depth int64) (*Expression, error) {
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

func (b *compiler) project(value protoreflect.Message, location string, facts map[string]bool, depth int64) (*Expression, error) {
	operand, err := b.bind(messageField(value, "operand").Interface(), location+".operand", nil, facts, true, depth+1)
	if err != nil {
		return nil, err
	}
	pathValue := messageField(value, "path").Interface().(*testpilotspb.FieldPath)
	path, err := b.catalog.BindPath(operand.typ, pathValue, b.budget.limits)
	if err != nil {
		return nil, err
	}
	if err := b.budget.charge(depth, int64(len(path.steps)), 0, "expression.path"); err != nil {
		return nil, err
	}
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(pathValue)
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

func (b *compiler) unary(operator Operator, source proto.Message, path string, facts map[string]bool, depth int64) (*Expression, error) {
	boolean := b.catalog.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN)
	var expected *Type
	if operator == Not {
		expected = &boolean
	}
	operand, err := b.bind(source, path, expected, facts, operator == IsPresent, depth+1)
	if err != nil {
		return nil, err
	}
	return &Expression{operator: operator, children: []*Expression{operand}, typ: boolean}, nil
}

// binary binds a comparison. Equality admits operands of any one type; an ordering operator admits
// only ordered numeric scalars.
func (b *compiler) binary(left, right proto.Message, comparison testpilotspb.ComparisonOperator, path string, facts map[string]bool, depth int64) (*Expression, error) {
	operands, err := b.pair(left, path+".left", right, path+".right", facts, depth)
	if err != nil {
		return nil, err
	}
	equality := comparison == testpilotspb.COMPARISON_OPERATOR_EQUAL || comparison == testpilotspb.COMPARISON_OPERATOR_NOT_EQUAL
	if !equality && !ordered(operands[0].typ) {
		return nil, invalid(TypeMismatch, path, "comparison requires ordered numeric scalars")
	}
	return &Expression{operator: Compare, children: operands, comparison: comparison, typ: b.catalog.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN)}, nil
}

func (b *compiler) logicalNode(operator Operator, operands []proto.Message, path string, facts map[string]bool, depth int64) (*Expression, error) {
	children, err := b.logical(operands, path, facts, operator == All, depth)
	if err != nil {
		return nil, err
	}
	return &Expression{operator: operator, children: children, typ: b.catalog.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN)}, nil
}

func (b *compiler) pair(left proto.Message, leftPath string, right proto.Message, rightPath string, facts map[string]bool, depth int64) ([]*Expression, error) {
	first, second := left, right
	firstPath, secondPath := leftPath, rightPath
	leftKind, _ := expressionVariant(left)
	rightKind, _ := expressionVariant(right)
	reversed := leftKind == "literal" && rightKind != "literal"
	if reversed {
		first, second = right, left
		firstPath, secondPath = rightPath, leftPath
	}
	a, err := b.bind(first, firstPath, nil, facts, false, depth+1)
	if err != nil {
		return nil, err
	}
	other, err := b.bind(second, secondPath, &a.typ, facts, false, depth+1)
	if err != nil {
		return nil, err
	}
	if reversed {
		return []*Expression{other, a}, nil
	}
	return []*Expression{a, other}, nil
}

func (b *compiler) logical(operands []proto.Message, path string, facts map[string]bool, continuing bool, depth int64) ([]*Expression, error) {
	if err := b.budget.charge(depth, int64(len(facts)), 0, "expression.presence"); err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(facts))
	maps.Copy(known, facts)
	result := make([]*Expression, 0, len(operands))
	boolean := b.catalog.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN)
	for index, operand := range operands {
		item, err := b.bind(operand, fmt.Sprintf("%s[%d]", path, index), &boolean, known, false, depth+1)
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

func expressionOperands(message protoreflect.Message) []proto.Message {
	field := message.Descriptor().Fields().ByName("operands")
	values := message.Get(field).List()
	result := make([]proto.Message, values.Len())
	for i := 0; i < values.Len(); i++ {
		result[i] = values.Get(i).Message().Interface()
	}
	return result
}

func (b *compiler) presenceFacts(e *Expression, truth bool) (map[string]bool, error) {
	if err := b.budget.charge(1, 1, 0, "expression.presence"); err != nil {
		return nil, err
	}
	switch e.operator {
	case Literal, ReferenceValue, Project, Compare:
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

func (c *Catalog) literalType(value *testpilotspb.Value) (Type, error) {
	if value == nil || missing(value.Value) {
		return Type{}, invalid(Malformed, "literal", "missing literal")
	}
	switch literal := value.Value.(type) {
	case *testpilotspb.Value_TextValue:
		return c.scalarType(testpilotspb.SCALAR_KIND_TEXT), nil
	case *testpilotspb.Value_NaturalValue:
		return c.scalarType(testpilotspb.SCALAR_KIND_NATURAL), nil
	case *testpilotspb.Value_BoolValue:
		return c.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN), nil
	case *testpilotspb.Value_BytesValue:
		return c.scalarType(testpilotspb.SCALAR_KIND_BYTES), nil
	case *testpilotspb.Value_MessageValue:
		url := literal.MessageValue.GetTypeUrl()
		name := url[strings.LastIndexByte(url, '/')+1:]
		return c.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: name}}}}})
	default:
		return Type{}, invalid(TypeMismatch, "literal", "numeric, enum, and collection literals require a contextual source type")
	}
}

func ordered(typ Type) bool {
	return typ.cardinality == Singular && (typ.scalar == testpilotspb.SCALAR_KIND_NATURAL || typ.scalar >= testpilotspb.SCALAR_KIND_INT32 && typ.scalar <= testpilotspb.SCALAR_KIND_DOUBLE)
}
