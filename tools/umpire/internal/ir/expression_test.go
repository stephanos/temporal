package ir

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/types/known/anypb"
)

func literal(value *umpirespb.Value) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Literal{Literal: value}}
}
func slot(id string) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Slot{Slot: &umpirespb.SlotReference{SlotId: id}}}
}
func present(value *umpirespb.ValueExpression) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Present{Present: &umpirespb.PresentExpression{Operand: value}}}
}
func equal(left, right *umpirespb.ValueExpression) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Equals{Equals: &umpirespb.EqualsExpression{Left: left, Right: right}}}
}
func negate(value *umpirespb.ValueExpression) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Negation{Negation: &umpirespb.NotExpression{Operand: value}}}
}
func all(values ...*umpirespb.ValueExpression) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_All{All: &umpirespb.AllExpression{Operands: values}}}
}
func anyOf(values ...*umpirespb.ValueExpression) *umpirespb.ValueExpression {
	return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Any{Any: &umpirespb.AnyExpression{Operands: values}}}
}

func TestExpressionsBindClosedVocabularyAndExplicitPresence(t *testing.T) {
	c := fixtureCatalog(t)
	textType := boundType(t, c, scalar(umpirespb.SCALAR_KIND_TEXT))
	boolType := boundType(t, c, scalar(umpirespb.SCALAR_KIND_BOOLEAN))
	intType := boundType(t, c, scalar(umpirespb.SCALAR_KIND_INT64))
	scope := map[Reference]Binding{
		{Kind: SlotReference, ID: "s"}:        {Type: textType},
		{Kind: ObservationReference, ID: "o"}: {Type: textType, Available: true},
		{Kind: CaptureReference, ID: "c"}:     {Type: textType, Available: true},
		{Kind: OutcomeReference, Entrypoint: "main", ID: "call", Field: int32(umpirespb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE)}: {Type: textType, Available: true},
		{Kind: EventReference, Field: int32(umpirespb.RUN_EVENT_FIELD_SEQUENCE)}:                                                  {Type: intType, Available: true},
		{Kind: SlotReference, ID: "message"}: {Type: boundType(t, c, named("fixture.Payload", false)), Available: true},
	}
	for name, expression := range map[string]*umpirespb.ValueExpression{
		"literal":     literal(text("x")),
		"slot":        all(present(slot("s")), equal(slot("s"), literal(text("x")))),
		"observation": {Expression: &umpirespb.ValueExpression_Observation{Observation: &umpirespb.ObservationReference{ObservationId: "o"}}},
		"capture":     {Expression: &umpirespb.ValueExpression_Capture{Capture: &umpirespb.CaptureReference{CaptureId: "c"}}},
		"outcome":     {Expression: &umpirespb.ValueExpression_Outcome{Outcome: &umpirespb.InstructionOutcomeReference{Instruction: &umpirespb.InstructionReference{EntrypointId: "main", InstructionId: "call"}, Field: umpirespb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE}}},
		"event":       {Expression: &umpirespb.ValueExpression_RunEvent{RunEvent: &umpirespb.RunEventFieldReference{Field: umpirespb.RUN_EVENT_FIELD_SEQUENCE}}},
		"path":        {Expression: &umpirespb.ValueExpression_Path{Path: &umpirespb.PathExpression{Source: slot("message"), Path: fieldPath("text")}}},
		"present":     present(slot("s")),
		"equality":    equal(literal(text("x")), literal(text("y"))),
		"compare":     {Expression: &umpirespb.ValueExpression_Compare{Compare: &umpirespb.CompareExpression{Operator: umpirespb.COMPARISON_OPERATOR_LESS_THAN, Left: &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_RunEvent{RunEvent: &umpirespb.RunEventFieldReference{Field: umpirespb.RUN_EVENT_FIELD_SEQUENCE}}}, Right: literal(signed("2"))}}},
		"not":         negate(literal(boolean(true))),
		"all":         all(literal(boolean(true)), literal(boolean(false))),
		"any":         anyOf(negate(present(slot("s"))), equal(slot("s"), literal(text("x")))),
	} {
		t.Run(name, func(t *testing.T) {
			compiled, err := c.BindExpression(expression, nil, scope, DefaultLimits())
			require.NoError(t, err)
			require.NotNil(t, compiled)
			require.NotNil(t, compiled.Type().Schema())
		})
	}
	_, err := c.BindExpression(slot("s"), &textType, scope, DefaultLimits())
	require.Error(t, err)
	_, err = c.BindExpression(equal(slot("s"), literal(text("x"))), &boolType, scope, DefaultLimits())
	require.Error(t, err)
	scope[Reference{Kind: SlotReference, ID: "s"}] = Binding{Type: textType, Available: true}
	compiled, err := c.BindExpression(slot("s"), &textType, scope, DefaultLimits())
	require.NoError(t, err)
	delete(scope, Reference{Kind: SlotReference, ID: "s"})
	require.True(t, textType.Equal(compiled.Type()))
}

func TestExpressionsRejectMalformedTypesAndResourceOverflow(t *testing.T) {
	c := fixtureCatalog(t)
	boolType := boundType(t, c, scalar(umpirespb.SCALAR_KIND_BOOLEAN))
	for name, expression := range map[string]*umpirespb.ValueExpression{
		"nil": nil, "empty": {}, "typed nil": {Expression: (*umpirespb.ValueExpression_All)(nil)},
		"nil operand": negate(nil), "unknown ref": slot("missing"), "undeclared presence": present(slot("missing")),
		"crossed equality": equal(literal(text("x")), literal(boolean(true))),
		"non bool":         all(literal(text("x"))),
		"bad comparison":   {Expression: &umpirespb.ValueExpression_Compare{Compare: &umpirespb.CompareExpression{Operator: umpirespb.COMPARISON_OPERATOR_UNSPECIFIED, Left: literal(signed("1")), Right: literal(signed("2"))}}},
		"unordered":        {Expression: &umpirespb.ValueExpression_Compare{Compare: &umpirespb.CompareExpression{Operator: umpirespb.COMPARISON_OPERATOR_LESS_THAN, Left: literal(boolean(true)), Right: literal(boolean(false))}}},
	} {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _, err := c.BindExpression(expression, nil, nil, DefaultLimits()); require.Error(t, err) })
		})
	}
	source := literal(boolean(true))
	source.ProtoReflect().SetUnknown([]byte{0x78, 1})
	_, err := c.BindExpression(source, nil, nil, DefaultLimits())
	require.Error(t, err)
	limits := DefaultLimits()
	limits.Depth = 2
	_, err = c.BindExpression(negate(negate(literal(boolean(true)))), &boolType, nil, limits)
	require.Error(t, err)
	limits = DefaultLimits()
	limits.Work = 1
	_, err = c.BindExpression(literal(boolean(true)), &boolType, nil, limits)
	require.Error(t, err)
	limits = DefaultLimits()
	limits.Work = math.MaxInt64
	_, err = c.BindExpression(literal(boolean(true)), &boolType, nil, limits)
	require.Error(t, err)
	opaqueSchema := &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_OpaqueCapability{OpaqueCapability: &umpirespb.OpaqueCapabilityType{}}}}}
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "capability"}: {Type: boundType(t, c, opaqueSchema), Available: true}}
	_, err = c.BindExpression(present(slot("capability")), nil, scope, DefaultLimits())
	require.Error(t, err)
}

func TestExpressionsRequireNumericSourceTypes(t *testing.T) {
	c := fixtureCatalog(t)
	for _, value := range []*umpirespb.Value{signed("1"), unsigned("1"), {Value: &umpirespb.Value_FloatingPoint{FloatingPoint: 1}}} {
		_, err := c.BindExpression(literal(value), nil, nil, DefaultLimits())
		require.Error(t, err)
		expression := &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Compare{Compare: &umpirespb.CompareExpression{Operator: umpirespb.COMPARISON_OPERATOR_LESS_THAN, Left: literal(value), Right: literal(value)}}}
		_, err = c.BindExpression(expression, nil, nil, DefaultLimits())
		require.Error(t, err)
	}
	typ := boundType(t, c, scalar(umpirespb.SCALAR_KIND_SINT32))
	_, err := c.BindExpression(literal(signed("1")), &typ, nil, DefaultLimits())
	require.NoError(t, err)
}

func TestExpressionPresenceFactsStayOnTheirSource(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "m"}: {Type: typ, Available: true}}
	projected := func(source *umpirespb.ValueExpression) *umpirespb.ValueExpression {
		return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Path{Path: &umpirespb.PathExpression{Source: source, Path: fieldPath("child", "text")}}}
	}
	_, err := c.BindExpression(all(present(projected(slot("m"))), equal(projected(slot("m")), literal(text("x")))), nil, scope, DefaultLimits())
	require.NoError(t, err)
	_, err = c.BindExpression(anyOf(present(projected(slot("m"))), equal(projected(slot("m")), literal(text("x")))), nil, scope, DefaultLimits())
	require.Error(t, err)
	payload := func(wire []byte) *umpirespb.ValueExpression {
		return literal(&umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: wire}}})
	}
	_, err = c.BindExpression(all(present(projected(payload([]byte{0x12, 3, 0x0a, 1, 'x'}))), equal(projected(payload(nil)), literal(text("x")))), nil, nil, DefaultLimits())
	require.Error(t, err)
	limits := DefaultLimits()
	limits.Depth = 3
	_, err = c.BindExpression(negate(negate(literal(boolean(true)))), nil, nil, limits)
	require.NoError(t, err)
}

func TestCompiledExpressionsRemainImmutableDuringConcurrentReuse(t *testing.T) {
	c := fixtureCatalog(t)
	source := all(literal(boolean(true)), literal(boolean(false)))
	expression, err := c.BindExpression(source, nil, nil, DefaultLimits())
	require.NoError(t, err)
	source.GetAll().Operands[0] = nil
	children := expression.Children()
	children[0] = nil
	copied := expression.Children()[0].Literal()
	copied.Value = &umpirespb.Value_BoolValue{BoolValue: false}
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			for j := 0; j < 20; j++ {
				require.True(t, expression.Children()[0].Literal().GetBoolValue())
				require.False(t, expression.Children()[1].Literal().GetBoolValue())
				_, err := c.Method("/fixture.Records/Read")
				require.NoError(t, err)
				_, err = c.BindExpression(negate(literal(boolean(true))), nil, nil, DefaultLimits())
				require.NoError(t, err)
			}
		})
	}
}

func TestGuardedExpressionUsesOnlyImpliedPresence(t *testing.T) {
	catalog, err := NewCatalog(catalogFixture())
	require.NoError(t, err)
	textType, err := catalog.BindType(&umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Scalar{Scalar: &umpirespb.ScalarType{Kind: umpirespb.SCALAR_KIND_TEXT}}}}})
	require.NoError(t, err)
	slot := func(id string) *umpirespb.ValueExpression {
		return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Slot{Slot: &umpirespb.SlotReference{SlotId: id}}}
	}
	present := func(id string) *umpirespb.ValueExpression {
		return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Present{Present: &umpirespb.PresentExpression{Operand: slot(id)}}}
	}
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "a"}: {Type: textType}, {Kind: SlotReference, ID: "b"}: {Type: textType}}
	for _, test := range []struct {
		name  string
		guard *umpirespb.ValueExpression
		good  bool
	}{
		{"present", present("a"), true},
		{"wrong source", present("b"), false},
		{"false branch", &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Negation{Negation: &umpirespb.NotExpression{Operand: present("a")}}}, false},
		{"non implying any", &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Any{Any: &umpirespb.AnyExpression{Operands: []*umpirespb.ValueExpression{present("a"), present("b")}}}}, false},
		{"unavailable guard", &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Equals{Equals: &umpirespb.EqualsExpression{Left: slot("a"), Right: slot("b")}}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := catalog.BindGuardedExpression(test.guard, slot("a"), &textType, scope, DefaultLimits())
			if test.good {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestGuardedExpressionExactPathAndSharedBudget(t *testing.T) {
	c := fixtureCatalog(t)
	message := boundType(t, c, named("fixture.Payload", false))
	textType := boundType(t, c, scalar(umpirespb.SCALAR_KIND_TEXT))
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "a"}: {Type: message, Available: true}, {Kind: SlotReference, ID: "b"}: {Type: message, Available: true}}
	project := func(id string) *umpirespb.ValueExpression {
		return &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Path{Path: &umpirespb.PathExpression{Source: slot(id), Path: fieldPath("optional_text")}}}
	}
	_, value, err := c.BindGuardedExpression(present(project("a")), project("a"), &textType, scope, DefaultLimits())
	require.NoError(t, err)
	require.False(t, value.MayBeAbsent())
	_, _, err = c.BindGuardedExpression(present(project("b")), project("a"), &textType, scope, DefaultLimits())
	require.Error(t, err)
	scope = map[Reference]Binding{{Kind: SlotReference, ID: "a"}: {Type: textType, Available: true}}
	limits := DefaultLimits()
	limits.Work = 10
	_, err = c.BindExpression(present(slot("a")), nil, scope, limits)
	require.NoError(t, err)
	_, err = c.BindExpression(slot("a"), nil, scope, limits)
	require.NoError(t, err)
	_, _, err = c.BindGuardedExpression(present(slot("a")), slot("a"), nil, scope, limits)
	require.Error(t, err)
}
