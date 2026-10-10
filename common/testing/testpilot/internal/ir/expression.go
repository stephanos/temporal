package ir

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	engine "cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/env"
	"cel.dev/cel-go/common/types"
	celpb "cel.dev/expr"
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
	ProjectedValueReference
	EventPayloadReference
	InstanceValueReference
	EvidenceFieldReference
	CorrelatedCaptureReference
	CorrelatedStepReference
)

type Reference struct {
	Kind           ReferenceKind
	Entrypoint, ID string
	Field          int32
	Ordinal        int64
}
type Binding struct {
	Type      Type
	Available bool
}
type Context uint8

const (
	ProgramContext Context = iota + 1
	ContractContext
	CorrelatedContext
	EvidenceLiftContext
)

var admittedReferences = map[Context]map[protoreflect.Name]bool{
	ProgramContext:      {"slot_id": true, "outcome": true, "run": true, "environment_binding_id": true},
	ContractContext:     {"observation_id": true, "run_event": true, "capture_id": true, "instance_value_id": true},
	CorrelatedContext:   {"evidence_field_id": true, "correlated_capture": true, "correlated_step": true},
	EvidenceLiftContext: {"projected_value": true},
}

func admitReference(context Context, path string, arm protoreflect.Name) error {
	if !admittedReferences[context][arm] {
		return Invalid(Unknown, path+"."+string(arm), "reference is not admitted in this expression context")
	}
	return nil
}
func AdmitReferences(site Site, source *testpilotspb.Expression) error {
	return WalkReferences(site.Path, source, func(path string, reference *testpilotspb.Reference) error {
		if reference == nil {
			return Invalid(Malformed, path, "reference is required")
		}
		message := reference.ProtoReflect()
		arm := message.WhichOneof(message.Descriptor().Oneofs().ByName("reference"))
		if arm == nil {
			return Invalid(Malformed, path, "reference is required")
		}
		return admitReference(site.Context, path+".reference", arm.Name())
	})
}
func WalkReferences(path string, source *testpilotspb.Expression, visit func(string, *testpilotspb.Reference) error) error {
	for i, binding := range source.GetBindings() {
		if reference := binding.GetReference(); reference != nil {
			if err := visit(fmt.Sprintf("%s.bindings[%d]", path, i), reference); err != nil {
				return err
			}
		}
	}
	return nil
}

type Site struct {
	Context Context
	Path    string
}
type Operator uint8

const (
	Literal Operator = iota + 1
	ReferenceValue
	ReadPath
	IsPresent
	Compare
	Not
	All
	Any
	Size
	Conditional
	ListExpression
	MapExpression
	OptionalExpression
	IndexExpression
)

type Expression struct {
	bindingWork        int64
	operator           Operator
	typ                Type
	literal            *celpb.Value
	reference          Reference
	children           []*Expression
	path               *Path
	comparison         string
	absent             bool
	key                string
	location           string
	optional           bool
	instanceValueReads []string
	variables          map[string]*Expression
	environment        *engine.Env
	checked            *engine.Ast
	registry           *types.Registry
	site               Site
	limits             Limits
}

func (e *Expression) BindingWork() int64           { return e.bindingWork }
func (e *Expression) Operator() Operator           { return e.operator }
func (e *Expression) Type() Type                   { return e.typ }
func (e *Expression) Literal() *celpb.Value        { return proto.CloneOf(e.literal) }
func (e *Expression) Reference() Reference         { return e.reference }
func (e *Expression) Children() []*Expression      { return slices.Clone(e.children) }
func (e *Expression) Path() *Path                  { return e.path }
func (e *Expression) Comparison() string           { return e.comparison }
func (e *Expression) MayBeAbsent() bool            { return e.absent }
func (e *Expression) InstanceValueReads() []string { return slices.Clone(e.instanceValueReads) }

type Condition struct {
	Expression *testpilotspb.Expression
	Path       string
	Matches    bool
}

func (c *Catalog) BindExpression(site Site, source *testpilotspb.Expression, expected *Type, scope map[Reference]Binding, limits Limits) (*Expression, error) {
	return c.BindConditionedExpression(nil, site, source, expected, scope, limits)
}
func (c *Catalog) BindGuardedExpression(guard Condition, site Site, source *testpilotspb.Expression, expected *Type, scope map[Reference]Binding, limits Limits) (*Expression, *Expression, error) {
	var conditions []Condition
	if guard.Expression != nil {
		guard.Matches = true
		conditions = []Condition{guard}
	}
	return c.bindConditioned(conditions, site, source, expected, scope, limits)
}
func (c *Catalog) BindConditionedExpression(conditions []Condition, site Site, source *testpilotspb.Expression, expected *Type, scope map[Reference]Binding, limits Limits) (*Expression, error) {
	_, value, err := c.bindConditioned(conditions, site, source, expected, scope, limits)
	return value, err
}
func (c *Catalog) bindConditioned(conditions []Condition, site Site, source *testpilotspb.Expression, expected *Type, scope map[Reference]Binding, limits Limits) (*Expression, *Expression, error) {
	if err := limits.validate(); err != nil {
		return nil, nil, err
	}
	b := &budget{limits: limits}
	facts := map[string]bool{}
	var guard *Expression
	boolean := c.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN)
	var reads []string
	for _, condition := range conditions {
		var err error
		guard, err = c.compile(Site{Context: site.Context, Path: condition.Path}, condition.Expression, &boolean, scope, b, facts)
		if err != nil {
			return nil, nil, err
		}
		maps.Copy(facts, presenceFacts(guard, condition.Matches))
		reads = append(reads, guard.instanceValueReads...)
	}
	bound, err := c.compile(site, source, expected, scope, b, facts)
	if err != nil {
		return nil, nil, err
	}
	bound.bindingWork = b.work
	bound.instanceValueReads = append(reads, bound.instanceValueReads...)
	return guard, bound, nil
}
func referenceKey(reference Reference) string {
	return fmt.Sprintf("%d:%q:%q:%d:%d", reference.Kind, reference.Entrypoint, reference.ID, reference.Field, reference.Ordinal)
}
func (c *Catalog) compile(site Site, source *testpilotspb.Expression, expected *Type, scope map[Reference]Binding, b *budget, facts map[string]bool) (*Expression, error) {
	if admittedReferences[site.Context] == nil {
		return nil, Invalid(Malformed, site.Path, "expression context is required")
	}
	if source == nil || source.GetCel().GetExpr() == nil {
		return nil, Invalid(Malformed, site.Path, "CEL expression is required")
	}
	if expected != nil && !c.owns(*expected) {
		return nil, Invalid(TypeMismatch, site.Path, "expected type belongs to another catalog")
	}
	surface := *b
	surface.work = b.surfaceWork
	surface.limits.Work = DefaultLimits().Work
	surface.limits.Depth = DefaultLimits().Depth
	if err := inspectSurface(source.ProtoReflect(), &surface, site.Path); err != nil {
		return nil, err
	}
	b.bytes = surface.bytes
	b.surfaceWork = surface.work
	canonical, err := CanonicalCEL(source.Cel)
	if err != nil {
		return nil, err
	}
	registry, err := c.celRegistry()
	if err != nil {
		return nil, err
	}
	subset := env.NewLibrarySubset().SetDisableMacros(true)
	for _, name := range []string{"_==_", "_!=_", "_<_", "_<=_", "_>_", "_>=_", "_&&_", "_||_", "!_", "_?_:_", "@in", "size", "_[_]"} {
		subset.AddIncludedFunctions(env.NewFunction(name))
	}
	options := []engine.EnvOption{engine.StdLib(engine.StdLibSubset(subset)), engine.OptionalTypes(), engine.ClearMacros(), engine.CustomTypeProvider(registry), engine.CustomTypeAdapter(registry)}
	variables := map[string]*Expression{}
	for pass := 0; pass < 2; pass++ {
		for i, input := range source.Bindings {
			if input != nil && (input.GetLiteral() != nil) != (pass == 1) {
				continue
			}
			location := fmt.Sprintf("%s.bindings[%d]", site.Path, i)
			if input == nil || input.Variable == "" || strings.HasPrefix(input.Variable, "__") || !validVariable(input.Variable) {
				return nil, Invalid(Malformed, location, "invalid CEL variable")
			}
			if variables[input.Variable] != nil {
				return nil, Invalid(Malformed, location, "duplicate CEL variable")
			}
			want := contextualType(canonical.Expr, input.Variable, variables, expected)
			bound, err := c.bindInput(site.Context, input, location, scope, b, facts, want)
			if err != nil {
				return nil, err
			}
			variables[input.Variable] = bound
			bound.optional = input.GetReference() != nil || input.Path != "" || optionalBindingUsed(canonical.Expr, input.Variable)
			variableType := celType(bound.typ)
			if bound.optional {
				variableType = engine.OptionalType(variableType)
			}
			options = append(options, engine.Variable(input.Variable, variableType))
		}
	}
	bound, err := c.bindNode(canonical.Expr, expected, variables, b, site.Path, 1)
	if err != nil {
		return nil, err
	}
	if err := checkCaptureAvailability(bound, facts); err != nil {
		return nil, err
	}
	if expected != nil && !bound.typ.Equal(*expected) {
		return nil, Invalid(TypeMismatch, site.Path, "expression type does not match expected type")
	}
	if bound.absent && !facts[bound.key] && site.Context != EvidenceLiftContext {
		return nil, Invalid(Unavailable, site.Path, "reference or path read requires an explicit presence guard")
	}
	legacy, err := bridgeCEL(canonical, b.limits)
	if err != nil {
		return nil, err
	}
	environment, err := engine.NewCustomEnv(options...)
	if err != nil {
		return nil, err
	}
	checked, issues := environment.Check(engine.ParsedExprToAst(legacy))
	if issues.Err() != nil {
		return nil, Invalid(TypeMismatch, site.Path, issues.String())
	}
	bound.variables, bound.environment, bound.checked, bound.registry, bound.site, bound.limits = variables, environment, checked, registry, site, b.limits
	for _, input := range source.Bindings {
		if id := input.GetReference().GetInstanceValueId(); id != "" {
			bound.instanceValueReads = append(bound.instanceValueReads, id)
		}
	}
	return bound, nil
}
func validVariable(name string) bool {
	word, end := scanName(name, 0)
	return word != "" && end == len(name)
}
func contextualType(node *celpb.Expr, name string, variables map[string]*Expression, expected *Type) *Type {
	if bindingIdentifier(node) == name {
		return expected
	}
	if call := node.GetCallExpr(); call != nil {
		if len(call.Args) == 2 && (call.Function == "_==_" || call.Function == "_!=_" || call.Function == "_<_" || call.Function == "_<=_" || call.Function == "_>_" || call.Function == "_>=_") {
			for i, arg := range call.Args {
				if bindingIdentifier(arg) == name {
					if other := bindingIdentifier(call.Args[1-i]); other != "" {
						if bound := variables[other]; bound != nil {
							return &bound.typ
						}
					}
				}
			}
		}
		for _, arg := range call.Args {
			if result := contextualType(arg, name, variables, nil); result != nil {
				return result
			}
		}
		if call.Target != nil {
			return contextualType(call.Target, name, variables, nil)
		}
	}
	return nil
}
func bindingIdentifier(node *celpb.Expr) string {
	if call := node.GetCallExpr(); call != nil && call.Function == "value" && len(call.Args) == 0 {
		node = call.Target
	}
	return node.GetIdentExpr().GetName()
}
func optionalBindingUsed(node *celpb.Expr, name string) bool {
	if call := node.GetCallExpr(); call != nil {
		if call.Target.GetIdentExpr().GetName() == name && (call.Function == "value" || call.Function == "hasValue") {
			return true
		}
		if optionalBindingUsed(call.Target, name) {
			return true
		}
		for _, arg := range call.Args {
			if optionalBindingUsed(arg, name) {
				return true
			}
		}
	}
	if selection := node.GetSelectExpr(); selection != nil {
		return optionalBindingUsed(selection.Operand, name)
	}
	if list := node.GetListExpr(); list != nil {
		for _, element := range list.Elements {
			if optionalBindingUsed(element, name) {
				return true
			}
		}
	}
	if mapping := node.GetStructExpr(); mapping != nil {
		for _, entry := range mapping.Entries {
			if optionalBindingUsed(entry.GetMapKey(), name) || optionalBindingUsed(entry.Value, name) {
				return true
			}
		}
	}
	return false
}
func (c *Catalog) bindInput(context Context, input *testpilotspb.ExpressionBinding, location string, scope map[Reference]Binding, b *budget, facts map[string]bool, expected *Type) (*Expression, error) {
	var operand *Expression
	if literal := input.GetLiteral(); literal != nil {
		var typ Type
		var err error
		if expected != nil && input.Path == "" {
			typ = *expected
		} else {
			typ, err = c.literalType(literal)
		}
		if err != nil {
			return nil, err
		}
		if err = c.checkLiteral(literal, typ, b, 1); err != nil {
			return nil, err
		}
		operand = &Expression{operator: Literal, typ: typ, literal: proto.CloneOf(literal)}
	} else {
		reference, err := parseReference(context, input.GetReference(), location+".reference")
		if err != nil {
			return nil, err
		}
		text := input.Path
		if input.GetReference().GetRunEvent().GetPayload() != nil {
			segments, err := parsePath(text)
			if err != nil || len(segments) == 0 || segments[0].selector != Field {
				return nil, Invalid(Malformed, location+".path", "payload path must name its arm")
			}
			reference = Reference{Kind: EventPayloadReference, ID: segments[0].field}
			text = strings.TrimPrefix(strings.TrimPrefix(text, segments[0].field), ".")
			if _, ok := c.RunEventPayloadType(protoreflect.Name(reference.ID)); !ok {
				return nil, Invalid(Unknown, location+".path", "unknown Run Event payload arm")
			}
		}
		binding, ok := scope[reference]
		if !ok {
			return nil, Invalid(Unknown, location, "reference is not declared in this environment")
		}
		if !c.owns(binding.Type) {
			return nil, Invalid(TypeMismatch, location, "reference type belongs to another catalog")
		}
		if binding.Type.opaque {
			return nil, Invalid(Unsupported, location, "opaque handles cannot be inspected")
		}
		key := referenceKey(reference)
		operand = &Expression{operator: ReferenceValue, typ: binding.Type, reference: reference, key: key, absent: !binding.Available && !facts[key]}
		if input.Path == "" {
			return operand, nil
		}
		path, err := c.BindPath(binding.Type, location+".path", text, b.limits)
		if err != nil {
			return nil, err
		}
		return bindPathMetadata(operand, path, b, facts)
	}
	if input.Path == "" {
		return operand, nil
	}
	path, err := c.BindPath(operand.typ, location+".path", input.Path, b.limits)
	if err != nil {
		return nil, err
	}
	return bindPathMetadata(operand, path, b, facts)
}
func bindPathMetadata(operand *Expression, path *Path, b *budget, facts map[string]bool) (*Expression, error) {
	if err := b.charge(1, int64(len(path.steps)), 0, "expression.path"); err != nil {
		return nil, err
	}
	key := ""
	if operand.key != "" {
		key = operand.key + "/" + path.text
	}
	absent := operand.absent
	segments, _ := parsePath(path.text)
	for i, step := range path.steps {
		if step.Selector == MapKey || step.Field.HasPresence() && step.Selector != Presence {
			prefix := operand.key + "/" + formatPath(segments[:i+1])
			absent = absent || operand.key == "" || !facts[prefix]
		}
	}
	bound := &Expression{operator: ReadPath, children: []*Expression{operand}, path: path, typ: path.typ, key: key, absent: absent && !facts[key]}
	if len(path.steps) > 0 && path.steps[len(path.steps)-1].Selector == Presence {
		bound.absent = false
	}
	return bound, nil
}
func parseReference(context Context, source *testpilotspb.Reference, path string) (Reference, error) {
	if source == nil {
		return Reference{}, Invalid(Malformed, path, "reference is required")
	}
	message := source.ProtoReflect()
	arm := message.WhichOneof(message.Descriptor().Oneofs().ByName("reference"))
	if arm == nil {
		return Reference{}, Invalid(Malformed, path, "reference is required")
	}
	if err := admitReference(context, path, arm.Name()); err != nil {
		return Reference{}, err
	}
	var result Reference
	switch v := source.Reference.(type) {
	case *testpilotspb.Reference_SlotId:
		result = Reference{Kind: SlotReference, ID: v.SlotId}
	case *testpilotspb.Reference_ObservationId:
		result = Reference{Kind: ObservationReference, ID: v.ObservationId}
	case *testpilotspb.Reference_CaptureId:
		result = Reference{Kind: CaptureReference, ID: v.CaptureId}
	case *testpilotspb.Reference_InstanceValueId:
		result = Reference{Kind: InstanceValueReference, ID: v.InstanceValueId}
	case *testpilotspb.Reference_ProjectedValue:
		result = Reference{Kind: ProjectedValueReference}
	case *testpilotspb.Reference_Run:
		result = Reference{Kind: EventReference, Field: int32(testpilotspb.RUN_EVENT_FIELD_RUN_ID)}
	case *testpilotspb.Reference_Outcome:
		result = Reference{Kind: OutcomeReference, Entrypoint: v.Outcome.GetInstruction().GetEntrypointId(), ID: v.Outcome.GetInstruction().GetInstructionId(), Field: int32(v.Outcome.GetField())}
		if result.Entrypoint == "" || result.Field <= 0 || result.Field > int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE) {
			return Reference{}, Invalid(Malformed, path, "invalid outcome reference")
		}
	case *testpilotspb.Reference_RunEvent:
		if v.RunEvent.GetPayload() != nil {
			return Reference{Kind: EventPayloadReference}, nil
		}
		result = Reference{Kind: EventReference, Field: int32(v.RunEvent.GetField())}
		if result.Field <= 0 || result.Field > int32(testpilotspb.RUN_EVENT_FIELD_RUN_ID) {
			return Reference{}, Invalid(Malformed, path, "invalid Run Event reference")
		}
	case *testpilotspb.Reference_EvidenceFieldId:
		result = Reference{Kind: EvidenceFieldReference, ID: v.EvidenceFieldId}
	case *testpilotspb.Reference_CorrelatedCapture:
		result = Reference{Kind: CorrelatedCaptureReference, ID: v.CorrelatedCapture.GetCaptureId(), Ordinal: v.CorrelatedCapture.GetOrdinal()}
		if result.Ordinal < 0 {
			return Reference{}, Invalid(Malformed, path, "negative capture ordinal")
		}
	case *testpilotspb.Reference_CorrelatedStep:
		result = Reference{Kind: CorrelatedStepReference, ID: v.CorrelatedStep.GetDefinitionId(), Field: int32(v.CorrelatedStep.GetField())}
		if result.Field < 1 || result.Field > 4 {
			return Reference{}, Invalid(Malformed, path, "invalid correlated step field")
		}
	default:
		return Reference{}, Invalid(Unsupported, path, "reference requires domain substitution before CEL binding")
	}
	if result.Kind != ProjectedValueReference && result.Kind != EventReference && result.ID == "" {
		return Reference{}, Invalid(Malformed, path, "reference identity is required")
	}
	return result, nil
}
func (c *Catalog) bindNode(node *celpb.Expr, expected *Type, variables map[string]*Expression, b *budget, path string, depth int64) (*Expression, error) {
	location := fmt.Sprintf("%s.cel.expr[%d]", path, node.GetId())
	work := int64(1)
	if call := node.GetCallExpr(); call != nil && call.Function == "optional.of" && call.Target == nil {
		work = 0
	}
	if err := b.charge(depth, work, 0, location); err != nil {
		return nil, err
	}
	if constant := node.GetConstExpr(); constant != nil {
		value, err := constantValue(constant)
		if err != nil {
			return nil, err
		}
		typ, err := c.literalType(value)
		if err != nil {
			return nil, err
		}
		if expected != nil {
			typ = *expected
			if typ.enumeration != nil {
				if integer, ok := value.Kind.(*celpb.Value_Int64Value); ok {
					if integer.Int64Value < -2147483648 || integer.Int64Value > 2147483647 {
						return nil, literalMismatch()
					}
					value = EnumValue(typ.enumeration, protoreflect.EnumNumber(integer.Int64Value))
				}
			}
		}
		if err := c.checkLiteral(value, typ, b, depth); err != nil {
			return nil, err
		}
		return &Expression{operator: Literal, literal: value, typ: typ}, nil
	}
	if identifier := node.GetIdentExpr(); identifier != nil {
		bound := variables[identifier.Name]
		if bound == nil {
			return nil, Invalid(Unknown, location, "unknown CEL variable "+identifier.Name)
		}
		if bound.optional {
			return nil, Invalid(Unsupported, location, "optional binding requires value() or hasValue()")
		}
		copy := *bound
		copy.location = location
		return &copy, nil
	}
	if receiver := node.GetCallExpr(); receiver != nil && receiver.Target != nil {
		if len(receiver.Args) != 0 || receiver.Function != "value" && receiver.Function != "hasValue" {
			return nil, Invalid(Unsupported, location, "unsupported CEL receiver function")
		}
		var target *Expression
		if name := receiver.Target.GetIdentExpr().GetName(); name != "" {
			input := variables[name]
			if input != nil {
				copy := *input
				target = &copy
			}
		} else {
			var err error
			target, err = c.bindNode(receiver.Target, expected, variables, b, path, depth+1)
			if err != nil {
				return nil, err
			}
		}
		if target == nil || !target.optional {
			return nil, Invalid(TypeMismatch, location, "optional receiver is required")
		}
		targetWork := int64(0)
		if receiver.Function == "hasValue" && receiver.Target.GetIdentExpr() != nil {
			targetWork = 1
		}
		if err := b.charge(depth+1, targetWork, 0, location); err != nil {
			return nil, err
		}
		target.location = location
		if receiver.Function == "hasValue" {
			if target.operator == OptionalExpression {
				target = target.children[0]
			}
			return &Expression{operator: IsPresent, typ: c.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN), children: []*Expression{target}, location: location}, nil
		}
		if target.operator == OptionalExpression {
			target = target.children[0]
		}
		target.optional = false
		return target, nil
	}
	if selection := node.GetSelectExpr(); selection != nil {
		return nil, Invalid(Unsupported, location, "message projections require an admitted binding path")
	}
	if list := node.GetListExpr(); list != nil {
		bound := &Expression{operator: ListExpression, location: location}
		var element *Type
		if expected != nil && expected.cardinality == Repeated {
			typ := expected.Element()
			element = &typ
		}
		for _, item := range list.Elements {
			child, err := c.bindNode(item, element, variables, b, path, depth+1)
			if err != nil {
				return nil, err
			}
			if child.typ.cardinality != Singular || element != nil && !element.Equal(child.typ) {
				return nil, Invalid(TypeMismatch, location, "list elements require one descriptor-exact singular type")
			}
			if element == nil {
				typ := child.typ
				element = &typ
			}
			bound.children = append(bound.children, child)
		}
		if element == nil {
			typ := c.scalarType(testpilotspb.SCALAR_KIND_TEXT)
			element = &typ
		}
		bound.typ = *element
		bound.typ.cardinality = Repeated
		bound.typ.schema = &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Repeated{Repeated: &testpilotspb.RepeatedType{Element: proto.CloneOf(element.schema.GetSingular())}}}
		return bound, nil
	}
	if mapping := node.GetStructExpr(); mapping != nil {
		bound := &Expression{operator: MapExpression, location: location}
		keyType := c.scalarType(testpilotspb.SCALAR_KIND_TEXT)
		var element *Type
		if expected != nil && expected.cardinality == Map {
			keyType = c.scalarType(expected.key)
			typ := expected.Element()
			element = &typ
		}
		for index, entry := range mapping.Entries {
			var wantKey *Type
			if expected != nil || index > 0 {
				wantKey = &keyType
			}
			key, err := c.bindNode(entry.GetMapKey(), wantKey, variables, b, path, depth+1)
			if err != nil {
				return nil, err
			}
			if index == 0 && expected == nil {
				keyType = key.typ
			}
			if !mapKeyKind(key.typ.scalar) || !key.typ.Equal(keyType) {
				return nil, Invalid(TypeMismatch, location, "map keys require one supported scalar type")
			}
			value, err := c.bindNode(entry.Value, element, variables, b, path, depth+1)
			if err != nil {
				return nil, err
			}
			if value.typ.cardinality != Singular || element != nil && !element.Equal(value.typ) {
				return nil, Invalid(TypeMismatch, location, "map values require one descriptor-exact singular type")
			}
			if element == nil {
				typ := value.typ
				element = &typ
			}
			bound.children = append(bound.children, key, value)
		}
		if element == nil {
			typ := c.scalarType(testpilotspb.SCALAR_KIND_TEXT)
			element = &typ
		}
		bound.typ = *element
		bound.typ.cardinality, bound.typ.key = Map, keyType.scalar
		bound.typ.schema = &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Map{Map: &testpilotspb.MapType{Key: &testpilotspb.ScalarType{Kind: keyType.scalar}, Value: proto.CloneOf(element.schema.GetSingular())}}}
		return bound, nil
	}
	call := node.GetCallExpr()
	if call == nil || call.Target != nil {
		return nil, Invalid(Unsupported, location, "unsupported CEL node or receiver call")
	}
	arity := map[string]int{"_==_": 2, "_!=_": 2, "_<_": 2, "_<=_": 2, "_>_": 2, "_>=_": 2, "_&&_": 2, "_||_": 2, "!_": 1, "_?_:_": 3, "@in": 2, "size": 1, "optional.of": 1, "_[_]": 2}
	count, ok := arity[call.Function]
	if !ok || len(call.Args) != count {
		return nil, Invalid(Unsupported, location, "unsupported CEL function or arity: "+call.Function)
	}
	boolean := c.scalarType(testpilotspb.SCALAR_KIND_BOOLEAN)
	bound := &Expression{typ: boolean, comparison: call.Function, location: location}
	for i, arg := range call.Args {
		var want *Type
		if call.Function == "!_" || call.Function == "_&&_" || call.Function == "_||_" || call.Function == "_?_:_" && i == 0 {
			want = &boolean
		}
		if i == 1 && call.Function != "@in" && call.Function != "_[_]" && len(bound.children) > 0 {
			if arg.GetConstExpr() != nil {
				want = &bound.children[0].typ
			}
		}
		child, err := c.bindNode(arg, want, variables, b, path, depth+1)
		if err != nil {
			return nil, err
		}
		bound.children = append(bound.children, child)
	}
	switch call.Function {
	case "optional.of":
		bound.operator, bound.typ, bound.optional = OptionalExpression, bound.children[0].typ, true
	case "!_":
		bound.operator = Not
	case "_&&_":
		bound.operator = All
	case "_||_":
		bound.operator = Any
	case "size":
		bound.operator = Size
		bound.typ = c.scalarType(testpilotspb.SCALAR_KIND_INT64)
	case "_?_:_":
		if !bound.children[1].typ.Equal(bound.children[2].typ) {
			return nil, Invalid(TypeMismatch, location, "conditional branches require the same descriptor-exact type")
		}
		bound.operator = Conditional
		bound.typ = bound.children[1].typ
	case "_[_]":
		bound.operator = IndexExpression
		bound.typ = bound.children[0].typ.Element()
	default:
		bound.operator = Compare
	}
	if bound.operator == Compare && len(bound.children) == 2 {
		left, right := bound.children[0].typ, bound.children[1].typ
		if call.Function == "@in" {
			if right.cardinality == Map {
				right = c.scalarType(right.key)
			} else if right.cardinality == Repeated {
				right = right.Element()
			}
		}
		if left.enumeration != nil || right.enumeration != nil {
			if !left.Equal(right) {
				return nil, Invalid(TypeMismatch, location, "enum comparison requires the same descriptor type")
			}
		}
	}
	if expected != nil && bound.operator == Conditional {
		bound.typ = *expected
	}
	return bound, nil
}
func checkCaptureAvailability(expression *Expression, facts map[string]bool) error {
	if expression.operator == IsPresent {
		return nil
	}
	if expression.operator == ReferenceValue && expression.reference.Kind == CaptureReference && expression.absent && !facts[expression.key] {
		return Invalid(Unavailable, expression.location, "capture read requires an explicit presence guard")
	}
	current := maps.Clone(facts)
	for i, child := range expression.children {
		branch := current
		if expression.operator == Conditional && i > 0 {
			branch = maps.Clone(facts)
			maps.Copy(branch, presenceFacts(expression.children[0], i == 1))
		}
		if err := checkCaptureAvailability(child, branch); err != nil {
			return err
		}
		if expression.operator == All || expression.operator == Any {
			maps.Copy(current, presenceFacts(child, expression.operator == All))
		}
	}
	return nil
}
func presenceFacts(e *Expression, truth bool) map[string]bool {
	switch e.operator {
	case IsPresent:
		if e.children[0].key != "" {
			return map[string]bool{e.children[0].key: truth}
		}
	case Not:
		return presenceFacts(e.children[0], !truth)
	case All, Any:
		result := map[string]bool{}
		merge := e.operator == All && truth || e.operator == Any && !truth
		for i, child := range e.children {
			facts := presenceFacts(child, truth)
			if merge || i == 0 {
				maps.Copy(result, facts)
			} else {
				for key, value := range result {
					if other, ok := facts[key]; !ok || other != value {
						delete(result, key)
					}
				}
			}
		}
		return result
	}
	return nil
}
func (c *Catalog) InstanceValueWork(id string, value *celpb.Value, typ Type) (int64, error) {
	b := &budget{limits: DefaultLimits()}
	if err := c.checkLiteral(value, typ, b, 1); err != nil {
		return 0, err
	}
	return b.work, nil
}
