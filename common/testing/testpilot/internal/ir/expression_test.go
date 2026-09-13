package ir

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	programSite  = Site{Context: ProgramContext, Path: "program.guard"}
	contractSite = Site{Context: ContractContext, Path: "contract.predicate"}
)

func literal(value *testpilotspb.Value) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: value}}
}
func slot(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_SlotId{SlotId: id}}}}
}
func present(value *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: value}}}
}
func equal(left, right *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL, Left: left, Right: right}}}
}
func negate(value *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Not{Not: &testpilotspb.NotExpression{Operand: value}}}
}
func reference(value *testpilotspb.Reference) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: value}}
}
func compare(operator testpilotspb.ComparisonOperator, left, right *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: operator, Left: left, Right: right}}}
}
func all(values ...*testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: values}}}
}
func anyOf(values ...*testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Any{Any: &testpilotspb.AnyExpression{Operands: values}}}
}

func TestExpressionsBindClosedVocabularyAndExplicitPresence(t *testing.T) {
	c := fixtureCatalog(t)
	textType := boundType(t, c, scalar(testpilotspb.SCALAR_KIND_TEXT))
	boolType := boundType(t, c, scalar(testpilotspb.SCALAR_KIND_BOOLEAN))
	intType := boundType(t, c, scalar(testpilotspb.SCALAR_KIND_INT64))
	scope := map[Reference]Binding{
		{Kind: SlotReference, ID: "s"}:        {Type: textType},
		{Kind: ObservationReference, ID: "o"}: {Type: textType, Available: true},
		{Kind: CaptureReference, ID: "c"}:     {Type: textType, Available: true},
		{Kind: OutcomeReference, Entrypoint: "main", ID: "call", Field: int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE)}: {Type: textType, Available: true},
		{Kind: EventReference, Field: int32(testpilotspb.RUN_EVENT_FIELD_SEQUENCE)}:                                                  {Type: intType, Available: true},
		{Kind: SlotReference, ID: "message"}: {Type: boundType(t, c, named("fixture.Payload", false)), Available: true},
		{Kind: SlotReference, ID: "i"}:       {Type: intType, Available: true},
	}
	for name, expression := range map[string]*testpilotspb.Expression{
		"literal":  literal(text("x")),
		"slot":     all(present(slot("s")), equal(slot("s"), literal(text("x")))),
		"outcome":  {Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: &testpilotspb.InstructionReference{EntrypointId: "main", InstructionId: "call"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE}}}}},
		"path":     {Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: slot("message"), Path: fieldPath("text")}}},
		"present":  present(slot("s")),
		"equality": equal(literal(text("x")), literal(text("y"))),
		"compare":  {Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_LESS_THAN, Left: slot("i"), Right: literal(signed("2"))}}},
		"not":      negate(literal(boolean(true))),
		"all":      all(literal(boolean(true)), literal(boolean(false))),
		"any":      anyOf(negate(present(slot("s"))), equal(slot("s"), literal(text("x")))),
	} {
		t.Run(name, func(t *testing.T) {
			compiled, err := c.BindExpression(programSite, expression, nil, scope, DefaultLimits())
			require.NoError(t, err)
			require.NotNil(t, compiled)
			require.NotNil(t, compiled.Type().Schema())
		})
	}
	_, err := c.BindExpression(programSite, slot("s"), &textType, scope, DefaultLimits())
	require.Error(t, err)
	_, err = c.BindExpression(programSite, equal(slot("s"), literal(text("x"))), &boolType, scope, DefaultLimits())
	require.Error(t, err)
	scope[Reference{Kind: SlotReference, ID: "s"}] = Binding{Type: textType, Available: true}
	compiled, err := c.BindExpression(programSite, slot("s"), &textType, scope, DefaultLimits())
	require.NoError(t, err)
	delete(scope, Reference{Kind: SlotReference, ID: "s"})
	require.True(t, textType.Equal(compiled.Type()))
}

func TestExpressionsRejectMalformedTypesAndResourceOverflow(t *testing.T) {
	c := fixtureCatalog(t)
	boolType := boundType(t, c, scalar(testpilotspb.SCALAR_KIND_BOOLEAN))
	for name, expression := range map[string]*testpilotspb.Expression{
		"nil": nil, "empty": {}, "typed nil": {Expression: (*testpilotspb.Expression_All)(nil)},
		"nil operand": negate(nil), "unknown ref": slot("missing"), "undeclared presence": present(slot("missing")),
		"crossed equality": equal(literal(text("x")), literal(boolean(true))),
		"non bool":         all(literal(text("x"))),
		"bad comparison":   {Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_UNSPECIFIED, Left: literal(signed("1")), Right: literal(signed("2"))}}},
		"unordered":        {Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_LESS_THAN, Left: literal(boolean(true)), Right: literal(boolean(false))}}},
	} {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, err := c.BindExpression(programSite, expression, nil, nil, DefaultLimits())
				require.Error(t, err)
			})
		})
	}
	source := literal(boolean(true))
	source.ProtoReflect().SetUnknown([]byte{0x78, 1})
	_, err := c.BindExpression(programSite, source, nil, nil, DefaultLimits())
	require.Error(t, err)
	limits := DefaultLimits()
	limits.Depth = 2
	_, err = c.BindExpression(programSite, negate(negate(literal(boolean(true)))), &boolType, nil, limits)
	require.Error(t, err)
	limits = DefaultLimits()
	limits.Work = 1
	_, err = c.BindExpression(programSite, literal(boolean(true)), &boolType, nil, limits)
	require.Error(t, err)
	limits = DefaultLimits()
	limits.Work = math.MaxInt64
	_, err = c.BindExpression(programSite, literal(boolean(true)), &boolType, nil, limits)
	require.Error(t, err)
	opaqueSchema := &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_OpaqueHandle{OpaqueHandle: &testpilotspb.OpaqueHandleType{}}}}}
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "capability"}: {Type: boundType(t, c, opaqueSchema), Available: true}}
	_, err = c.BindExpression(programSite, present(slot("capability")), nil, scope, DefaultLimits())
	require.Error(t, err)
}

func TestExpressionsRequireNumericSourceTypes(t *testing.T) {
	c := fixtureCatalog(t)
	for _, value := range []*testpilotspb.Value{signed("1"), unsigned("1"), {Value: &testpilotspb.Value_FloatingPointValue{FloatingPointValue: 1}}} {
		_, err := c.BindExpression(programSite, literal(value), nil, nil, DefaultLimits())
		require.Error(t, err)
		expression := &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_LESS_THAN, Left: literal(value), Right: literal(value)}}}
		_, err = c.BindExpression(programSite, expression, nil, nil, DefaultLimits())
		require.Error(t, err)
	}
	typ := boundType(t, c, scalar(testpilotspb.SCALAR_KIND_SINT32))
	_, err := c.BindExpression(programSite, literal(signed("1")), &typ, nil, DefaultLimits())
	require.NoError(t, err)
}

func TestExpressionPresenceFactsStayOnTheirSource(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "m"}: {Type: typ, Available: true}}
	projected := func(source *testpilotspb.Expression) *testpilotspb.Expression {
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: source, Path: fieldPath("child", "text")}}}
	}
	_, err := c.BindExpression(programSite, all(present(projected(slot("m"))), equal(projected(slot("m")), literal(text("x")))), nil, scope, DefaultLimits())
	require.NoError(t, err)
	_, err = c.BindExpression(programSite, anyOf(present(projected(slot("m"))), equal(projected(slot("m")), literal(text("x")))), nil, scope, DefaultLimits())
	require.Error(t, err)
	payload := func(wire []byte) *testpilotspb.Expression {
		return literal(&testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: wire}}})
	}
	_, err = c.BindExpression(programSite, all(present(projected(payload([]byte{0x12, 3, 0x0a, 1, 'x'}))), equal(projected(payload(nil)), literal(text("x")))), nil, nil, DefaultLimits())
	require.Error(t, err)
	limits := DefaultLimits()
	limits.Depth = 3
	_, err = c.BindExpression(programSite, negate(negate(literal(boolean(true)))), nil, nil, limits)
	require.NoError(t, err)
}

func TestCompiledExpressionsRemainImmutableDuringConcurrentReuse(t *testing.T) {
	c := fixtureCatalog(t)
	source := all(literal(boolean(true)), literal(boolean(false)))
	expression, err := c.BindExpression(programSite, source, nil, nil, DefaultLimits())
	require.NoError(t, err)
	source.GetAll().Operands[0] = nil
	children := expression.Children()
	children[0] = nil
	copied := expression.Children()[0].Literal()
	copied.Value = &testpilotspb.Value_BoolValue{BoolValue: false}
	for i := 0; i < 8; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			for j := 0; j < 20; j++ {
				require.True(t, expression.Children()[0].Literal().GetBoolValue())
				require.False(t, expression.Children()[1].Literal().GetBoolValue())
				_, err := c.Method("/fixture.Records/Read")
				require.NoError(t, err)
				_, err = c.BindExpression(programSite, negate(literal(boolean(true))), nil, nil, DefaultLimits())
				require.NoError(t, err)
			}
		})
	}
}

func TestGuardedExpressionUsesOnlyImpliedPresence(t *testing.T) {
	catalog, err := NewCatalog(catalogFixture())
	require.NoError(t, err)
	textType, err := catalog.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}}}}})
	require.NoError(t, err)
	slot := func(id string) *testpilotspb.Expression {
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_SlotId{SlotId: id}}}}
	}
	present := func(id string) *testpilotspb.Expression {
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: slot(id)}}}
	}
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "a"}: {Type: textType}, {Kind: SlotReference, ID: "b"}: {Type: textType}}
	for _, test := range []struct {
		name  string
		guard *testpilotspb.Expression
		good  bool
	}{
		{"present", present("a"), true},
		{"wrong source", present("b"), false},
		{"false branch", &testpilotspb.Expression{Expression: &testpilotspb.Expression_Not{Not: &testpilotspb.NotExpression{Operand: present("a")}}}, false},
		{"non implying any", &testpilotspb.Expression{Expression: &testpilotspb.Expression_Any{Any: &testpilotspb.AnyExpression{Operands: []*testpilotspb.Expression{present("a"), present("b")}}}}, false},
		{"unavailable guard", &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL, Left: slot("a"), Right: slot("b")}}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := catalog.BindGuardedExpression(Condition{Expression: test.guard}, programSite, slot("a"), &textType, scope, DefaultLimits())
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
	textType := boundType(t, c, scalar(testpilotspb.SCALAR_KIND_TEXT))
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "a"}: {Type: message, Available: true}, {Kind: SlotReference, ID: "b"}: {Type: message, Available: true}}
	project := func(id string) *testpilotspb.Expression {
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: slot(id), Path: fieldPath("optional_text")}}}
	}
	_, value, err := c.BindGuardedExpression(Condition{Expression: present(project("a"))}, programSite, project("a"), &textType, scope, DefaultLimits())
	require.NoError(t, err)
	require.False(t, value.MayBeAbsent())
	measured := DefaultLimits()
	measured.Work = value.BindingWork()
	_, _, err = c.BindGuardedExpression(Condition{Expression: present(project("a"))}, programSite, project("a"), &textType, scope, measured)
	require.NoError(t, err)
	measured.Work--
	_, _, err = c.BindGuardedExpression(Condition{Expression: present(project("a"))}, programSite, project("a"), &textType, scope, measured)
	require.Error(t, err)
	_, _, err = c.BindGuardedExpression(Condition{Expression: present(project("b"))}, programSite, project("a"), &textType, scope, DefaultLimits())
	require.Error(t, err)
	scope = map[Reference]Binding{{Kind: SlotReference, ID: "a"}: {Type: textType, Available: true}}
	limits := DefaultLimits()
	limits.Work = 10
	_, err = c.BindExpression(programSite, present(slot("a")), nil, scope, limits)
	require.NoError(t, err)
	_, err = c.BindExpression(programSite, slot("a"), nil, scope, limits)
	require.NoError(t, err)
	_, _, err = c.BindGuardedExpression(Condition{Expression: present(slot("a"))}, programSite, slot("a"), nil, scope, limits)
	require.Error(t, err)
}

// The Run Event reference range is the one guard between a Contract expression and a coordinate the
// recorder never populates, so every common coordinate must be inside it and nothing beyond.
func TestRunEventReferencesAdmitCoordinates(t *testing.T) {
	c := fixtureCatalog(t)
	textType := boundType(t, c, scalar(testpilotspb.SCALAR_KIND_TEXT))
	for _, tc := range []struct {
		field testpilotspb.RunEventField
		admit bool
	}{
		{testpilotspb.RUN_EVENT_FIELD_SEQUENCE, true},
		{testpilotspb.RUN_EVENT_FIELD_RUN_ID, true},
		{testpilotspb.RUN_EVENT_FIELD_UNSPECIFIED, false},
		{testpilotspb.RUN_EVENT_FIELD_RUN_ID + 1, false},
	} {
		t.Run(fmt.Sprint(int32(tc.field)), func(t *testing.T) {
			scope := map[Reference]Binding{{Kind: EventReference, Field: int32(tc.field)}: {Type: textType, Available: true}}
			_, err := c.BindExpression(contractSite, runEventReference(&testpilotspb.RunEventReference{Selection: &testpilotspb.RunEventReference_Field{Field: tc.field}}), nil, scope, DefaultLimits())
			if tc.admit {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}

func runEventReference(event *testpilotspb.RunEventReference) *testpilotspb.Expression {
	return reference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_RunEvent{RunEvent: event}})
}

func payloadPath(segments ...string) *testpilotspb.Expression {
	path := &testpilotspb.FieldPath{}
	for _, segment := range segments {
		path.Segments = append(path.Segments, &testpilotspb.FieldPathSegment{Field: segment})
	}
	payload := runEventReference(&testpilotspb.RunEventReference{Selection: &testpilotspb.RunEventReference_Payload{Payload: &testpilotspb.RunEventPayloadReference{}}})
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: payload, Path: path}}}
}

// A path from the Run Event payload binds through the arm its first segment names: the arm must be
// declared in scope, the rest of the path is typed by the arm's descriptor, and the payload itself is
// never an operand on its own.
func TestRunEventPayloadPathsBindThroughTheArmTheyName(t *testing.T) {
	c := fixtureCatalog(t)
	faultType, known := c.RunEventPayloadType("fault_injected")
	require.True(t, known)
	declared := map[Reference]Binding{{Kind: EventPayloadReference, ID: "fault_injected"}: {Type: faultType, Available: true}}
	unavailable := map[Reference]Binding{{Kind: EventPayloadReference, ID: "fault_injected"}: {Type: faultType}}
	queue := literal(text("queue"))
	located := "contract.predicate.compare.left.path.path"
	for _, tc := range []struct {
		name       string
		site       Site
		expression *testpilotspb.Expression
		scope      map[Reference]Binding
		want       *Error
	}{
		{name: "declared arm", site: contractSite, expression: equal(payloadPath("fault_injected", "role_id"), queue), scope: declared},
		{name: "guarded arm", site: contractSite, expression: &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: []*testpilotspb.Expression{present(payloadPath("fault_injected", "role_id")), equal(payloadPath("fault_injected", "role_id"), queue)}}}}, scope: unavailable},
		{name: "unguarded arm", site: contractSite, expression: equal(payloadPath("fault_injected", "role_id"), queue), scope: unavailable, want: &Error{Category: Unavailable, Path: "expression", Detail: "reference or projection requires an explicit presence guard"}},
		{name: "undeclared arm", site: contractSite, expression: equal(payloadPath("outcome", "detail"), queue), scope: declared, want: &Error{Category: Unknown, Path: located + ".segments[0].field", Detail: "no Run Event kind this expression evaluates can carry the payload arm"}},
		{name: "unknown arm", site: contractSite, expression: equal(payloadPath("source_id"), queue), scope: declared, want: &Error{Category: Unknown, Path: located + ".segments[0].field", Detail: "unknown Run Event payload arm"}},
		{name: "empty path", site: contractSite, expression: equal(payloadPath(), queue), scope: declared, want: &Error{Category: Malformed, Path: located, Detail: "a Run Event payload path starts with the plain name of a payload arm"}},
		{name: "bare payload", site: contractSite, expression: equal(runEventReference(&testpilotspb.RunEventReference{Selection: &testpilotspb.RunEventReference_Payload{Payload: &testpilotspb.RunEventPayloadReference{}}}), queue), scope: declared, want: &Error{Category: Malformed, Path: "contract.predicate.compare.left.reference.run_event.payload", Detail: "a Run Event payload is read only through a path that names its arm"}},
		{name: "outside the Contract context", site: programSite, expression: equal(payloadPath("fault_injected", "role_id"), queue), scope: declared, want: &Error{Category: Unknown, Path: "program.guard.compare.left.path.operand.reference.run_event", Detail: "reference is not admitted in this expression context"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := c.BindExpression(tc.site, tc.expression, nil, tc.scope, DefaultLimits())
			if tc.want == nil {
				require.NoError(t, err)
				return
			}
			require.Equal(t, tc.want, err)
		})
	}

	bound, err := c.BindExpression(contractSite, equal(payloadPath("fault_injected", "kind"), literal(&testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: int32(testpilotspb.FAULT_KIND_WORKER_STOP)}}})), nil, declared, DefaultLimits())
	require.NoError(t, err)
	event := &testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED, Payload: &testpilotspb.RunEvent_FaultInjected{FaultInjected: &testpilotspb.FaultInjected{RoleId: "queue", Kind: testpilotspb.FAULT_KIND_WORKER_STOP}}}
	matched, _, err := bound.Evaluate(t.Context(), func(reference Reference) *testpilotspb.Value {
		return RunEventPayloadValue(event, protoreflect.Name(reference.ID))
	}, DefaultLimits().Work)
	require.NoError(t, err)
	require.True(t, matched.GetBoolValue())
}

// Program and Contract share one expression language, so a reference outside its context is
// rejected at binding, before its scope is consulted, and the error names the reference arm under
// the expression's path. An admitted reference goes on to the scope, which is empty here.
func TestExpressionContextsRejectReferencesOutsideThem(t *testing.T) {
	c := fixtureCatalog(t)
	references := map[protoreflect.Name]*testpilotspb.Reference{
		"slot_id": {Reference: &testpilotspb.Reference_SlotId{SlotId: "s"}},
		"outcome": {Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{
			Instruction: &testpilotspb.InstructionReference{EntrypointId: "main", InstructionId: "call"}, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS,
		}}},
		"run":                    {Reference: &testpilotspb.Reference_Run{Run: &testpilotspb.RunReference{}}},
		"environment_binding_id": {Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: "namespace"}},
		"observation_id":         {Reference: &testpilotspb.Reference_ObservationId{ObservationId: "o"}},
		"run_event":              {Reference: &testpilotspb.Reference_RunEvent{RunEvent: &testpilotspb.RunEventReference{Selection: &testpilotspb.RunEventReference_Field{Field: testpilotspb.RUN_EVENT_FIELD_KIND}}}},
		"capture_id":             {Reference: &testpilotspb.Reference_CaptureId{CaptureId: "c"}},
		"evidence_field_id":      {Reference: &testpilotspb.Reference_EvidenceFieldId{EvidenceFieldId: "f"}},
		"correlated_capture":     {Reference: &testpilotspb.Reference_CorrelatedCapture{CorrelatedCapture: &testpilotspb.CorrelatedCaptureReference{CaptureId: "c"}}},
		"model_value":            {Reference: &testpilotspb.Reference_ModelValue{ModelValue: &testpilotspb.ModelValue{DefinitionId: "d", Value: "v"}}},
		"correlated_step":        {Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{Field: testpilotspb.CORRELATED_STEP_FIELD_ACTION, DefinitionId: "d"}}},
		"projected_value":        {Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &testpilotspb.ProjectedValueReference{}}},
	}
	require.Len(t, references, (&testpilotspb.Reference{}).ProtoReflect().Descriptor().Fields().Len(), "every Reference arm is probed")
	for _, context := range []struct {
		site     Site
		admitted []protoreflect.Name
	}{
		{programSite, []protoreflect.Name{"slot_id", "outcome", "run", "environment_binding_id"}},
		{contractSite, []protoreflect.Name{"observation_id", "run_event", "capture_id"}},
		{Site{Context: CorrelatedContext, Path: "correlated"}, []protoreflect.Name{"evidence_field_id", "correlated_capture", "correlated_step"}},
		{Site{Context: EvidenceLiftContext, Path: "lift"}, []protoreflect.Name{"projected_value"}},
	} {
		for name, value := range references {
			t.Run(context.site.Path+"/"+string(name), func(t *testing.T) {
				expression := all(literal(boolean(true)), present(reference(value)))
				_, err := c.BindExpression(context.site, expression, nil, nil, DefaultLimits())
				var diagnostic *Error
				require.ErrorAs(t, err, &diagnostic)
				admitted := AdmitReferences(context.site, expression)
				if slices.Contains(context.admitted, name) {
					require.Equal(t, "expression", diagnostic.Path)
					require.NoError(t, admitted)
					return
				}
				want := &Error{
					Category: Unknown,
					Path:     context.site.Path + ".all[1].present.reference." + string(name),
					Detail:   "reference is not admitted in this expression context",
				}
				require.Equal(t, want, diagnostic)
				require.Equal(t, want, admitted)
			})
		}
	}
	_, err := c.BindExpression(Site{Path: "unset"}, literal(boolean(true)), nil, nil, DefaultLimits())
	require.Equal(t, &Error{Category: Malformed, Path: "expression", Detail: "expression context is required"}, err)
}

// AdmitReferences locates a rejected reference at the path BindExpression reports, through every
// operator that nests an operand.
func TestAdmitReferencesLocatesLikeBinding(t *testing.T) {
	c := fixtureCatalog(t)
	path := &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: slot("s"), Path: &testpilotspb.FieldPath{}}}}
	for name, expression := range map[string]*testpilotspb.Expression{
		".present.path.operand": present(path),
		".not":                  negate(slot("s")),
		".compare.left":         equal(slot("s"), literal(boolean(true))),
		".compare.right":        equal(literal(boolean(true)), slot("s")),
		".any[0].not.not":       anyOf(negate(negate(slot("s")))),
		".all[0].present":       all(present(slot("s"))),
	} {
		t.Run(name, func(t *testing.T) {
			want := &Error{Category: Unknown, Path: contractSite.Path + name + ".reference.slot_id", Detail: "reference is not admitted in this expression context"}
			require.Equal(t, want, AdmitReferences(contractSite, expression))
			_, err := c.BindExpression(contractSite, expression, nil, nil, DefaultLimits())
			require.Equal(t, want, err)
		})
	}
}

// Equality admits any operand type, while an ordering operator on a type with no numeric order is
// rejected at the comparison it names.
func TestComparisonOperatorsAdmitTheirOperandTypes(t *testing.T) {
	c := fixtureCatalog(t)
	element := &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_INT64}}}
	types := map[string]*testpilotspb.ValueType{
		"message": named("fixture.Payload", false),
		"list":    {Shape: &testpilotspb.ValueType_Repeated{Repeated: &testpilotspb.RepeatedType{Element: element}}},
		"map":     {Shape: &testpilotspb.ValueType_Map{Map: &testpilotspb.MapType{Key: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Value: element}}},
		"bytes":   scalar(testpilotspb.SCALAR_KIND_BYTES),
	}
	for name, typ := range types {
		scope := map[Reference]Binding{{Kind: SlotReference, ID: "a"}: {Type: boundType(t, c, typ), Available: true}, {Kind: SlotReference, ID: "b"}: {Type: boundType(t, c, typ), Available: true}}
		for operator := testpilotspb.COMPARISON_OPERATOR_EQUAL; operator <= testpilotspb.COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL; operator++ {
			t.Run(name+"/"+operator.String(), func(t *testing.T) {
				_, err := c.BindExpression(programSite, negate(compare(operator, slot("a"), slot("b"))), nil, scope, DefaultLimits())
				if operator == testpilotspb.COMPARISON_OPERATOR_EQUAL || operator == testpilotspb.COMPARISON_OPERATOR_NOT_EQUAL {
					require.NoError(t, err)
					return
				}
				require.Equal(t, &Error{Category: TypeMismatch, Path: programSite.Path + ".not.compare", Detail: "comparison requires ordered numeric scalars"}, err)
			})
		}
	}
}
