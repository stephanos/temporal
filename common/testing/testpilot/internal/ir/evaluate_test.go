package ir

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestRuntimeOperatorsAndExactBudget(t *testing.T) {
	c := fixtureCatalog(t)
	for _, tc := range []struct {
		name        string
		kind        testpilotspb.ScalarKind
		a, b        *testpilotspb.Value
		equal, less bool
	}{
		{"int64 precision", testpilotspb.SCALAR_KIND_INT64, signed("9007199254740992"), signed("9007199254740993"), false, true},
		{"uint64", testpilotspb.SCALAR_KIND_UINT64, unsigned("18446744073709551614"), unsigned("18446744073709551615"), false, true},
		{"natural", testpilotspb.SCALAR_KIND_NATURAL, &testpilotspb.Value{Value: &testpilotspb.Value_Natural{Natural: "99999999999999999999"}}, &testpilotspb.Value{Value: &testpilotspb.Value_Natural{Natural: "100000000000000000000"}}, false, true},
		{"negative", testpilotspb.SCALAR_KIND_INT32, signed("-2"), signed("-1"), false, true},
		{"float32 precision", testpilotspb.SCALAR_KIND_FLOAT, &testpilotspb.Value{Value: &testpilotspb.Value_FloatingPoint{FloatingPoint: 0.1}}, &testpilotspb.Value{Value: &testpilotspb.Value_FloatingPoint{FloatingPoint: float64(float32(0.1))}}, true, false},
		{"zero", testpilotspb.SCALAR_KIND_DOUBLE, &testpilotspb.Value{Value: &testpilotspb.Value_FloatingPoint{FloatingPoint: math.Copysign(0, -1)}}, &testpilotspb.Value{Value: &testpilotspb.Value_FloatingPoint{FloatingPoint: 0}}, true, false},
		{"NaN", testpilotspb.SCALAR_KIND_DOUBLE, &testpilotspb.Value{Value: &testpilotspb.Value_FloatingPoint{FloatingPoint: math.NaN()}}, &testpilotspb.Value{Value: &testpilotspb.Value_FloatingPoint{FloatingPoint: math.NaN()}}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			typ := boundType(t, c, scalar(tc.kind))
			scope := map[Reference]Binding{{Kind: SlotReference, ID: "a"}: {Type: typ, Available: true}, {Kind: SlotReference, ID: "b"}: {Type: typ, Available: true}}
			resolver := func(ref Reference) *testpilotspb.Value {
				if ref.ID == "a" {
					return tc.a
				}
				return tc.b
			}
			for _, eq := range []bool{false, true} {
				source := equal(slot("a"), slot("b"))
				want := tc.equal
				if !eq {
					source = &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Compare{Compare: &testpilotspb.ProgramCompareExpression{Left: slot("a"), Right: slot("b"), Operator: testpilotspb.COMPARISON_OPERATOR_LESS_THAN}}}
					want = tc.less
				}
				e, err := c.BindExpression(source, nil, scope, DefaultLimits())
				require.NoError(t, err)
				value, work, err := e.Evaluate(context.Background(), resolver, 10000)
				require.NoError(t, err)
				require.Equal(t, want, value.GetBoolValue())
				_, exact, err := e.Evaluate(context.Background(), resolver, work)
				require.NoError(t, err)
				require.Equal(t, work, exact)
				_, _, err = e.Evaluate(context.Background(), resolver, work-1)
				require.Error(t, err)
			}
		})
	}
	typ := boundType(t, c, scalar(testpilotspb.SCALAR_KIND_TEXT))
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "missing"}: {Type: typ}}
	for _, source := range []*testpilotspb.ProgramExpression{all(present(slot("missing")), equal(slot("missing"), literal(text("x")))), negate(anyOf(present(slot("missing")), literal(boolean(true))))} {
		e, err := c.BindExpression(source, nil, scope, DefaultLimits())
		require.NoError(t, err)
		value, _, err := e.Evaluate(context.Background(), func(Reference) *testpilotspb.Value { return nil }, 100)
		require.NoError(t, err)
		require.False(t, value.GetBoolValue())
	}
}
func TestRuntimePathsAndPresence(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	source := &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{0x12, 3, 0x0a, 1, 'x', 0x1a, 3, 0x0a, 1, 'a', 0x1a, 3, 0x0a, 1, 'b', 0x22, 7, 0x0a, 3, 'k', 'e', 'y', 0x10, 7}}}}
	require.NoError(t, c.CheckLiteral(source, typ, DefaultLimits()))
	wildcard := fieldPath("items", "text")
	wildcard.Segments[0].Selector = &testpilotspb.FieldPathSegment_Repeated{Repeated: &testpilotspb.RepeatedWildcard{}}
	lookup := fieldPath("labels")
	lookup.Segments[0].Selector = &testpilotspb.FieldPathSegment_MapKey{MapKey: &testpilotspb.MapKeySelector{Key: text("key")}}
	absence := fieldPath("child", "optional_text")
	absence.Segments[1].Selector = &testpilotspb.FieldPathSegment_Presence{Presence: &testpilotspb.PresenceSelector{}}
	for _, tc := range []struct {
		name string
		path *testpilotspb.FieldPath
		want *testpilotspb.Value
	}{
		{"nested", fieldPath("child", "text"), text("x")},
		{"wildcard", wildcard, &testpilotspb.Value{Value: &testpilotspb.Value_ListValue{ListValue: &testpilotspb.ValueList{Values: []*testpilotspb.Value{text("a"), text("b")}}}}},
		{"map", lookup, signed("7")},
		{"presence", absence, boolean(false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expr := &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Path{Path: &testpilotspb.ProgramPathExpression{Source: slot("source"), Path: tc.path}}}
			// The guard supplies the admission fact while the value itself exercises the path.
			e, err := c.BindConditionedExpression([]Condition{{Expression: present(expr), Matches: true}}, expr, nil, map[Reference]Binding{{Kind: SlotReference, ID: "source"}: {Type: typ, Available: true}}, DefaultLimits())
			require.NoError(t, err)
			value, work, err := e.Evaluate(context.Background(), func(Reference) *testpilotspb.Value { return source }, 10000)
			require.NoError(t, err)
			require.True(t, proto.Equal(tc.want, value), "%v", value)
			_, _, err = e.Evaluate(context.Background(), func(Reference) *testpilotspb.Value { return source }, work)
			require.NoError(t, err)
		})
	}
	empty := &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload"}}}
	expr := &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Path{Path: &testpilotspb.ProgramPathExpression{Source: slot("source"), Path: absence}}}
	e, err := c.BindExpression(expr, nil, map[Reference]Binding{{Kind: SlotReference, ID: "source"}: {Type: typ, Available: true}}, DefaultLimits())
	require.NoError(t, err)
	value, _, err := e.Evaluate(context.Background(), func(Reference) *testpilotspb.Value { return empty }, 1000)
	require.NoError(t, err)
	require.True(t, proto.Equal(boolean(false), value))
}

func TestRuntimeWildcardDoesNotFilterAbsentFields(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	for _, tc := range []struct {
		name     string
		wire     []byte
		present  bool
		presence []bool
	}{
		{"empty", nil, true, nil},
		{"all present", []byte{0x1a, 3, 0x52, 1, 'x', 0x1a, 3, 0x52, 1, 'y'}, true, []bool{true, true}},
		{"mixed", []byte{0x1a, 3, 0x52, 1, 'x', 0x1a, 0}, false, []bool{true, false}},
		{"all missing", []byte{0x1a, 0, 0x1a, 0}, false, []bool{false, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: tc.wire}}}
			require.NoError(t, c.CheckLiteral(source, typ, DefaultLimits()))
			path := fieldPath("items", "optional_text")
			path.Segments[0].Selector = &testpilotspb.FieldPathSegment_Repeated{Repeated: &testpilotspb.RepeatedWildcard{}}
			project := &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Path{Path: &testpilotspb.ProgramPathExpression{Source: slot("source"), Path: path}}}
			scope := map[Reference]Binding{{Kind: SlotReference, ID: "source"}: {Type: typ, Available: true}}
			e, err := c.BindExpression(present(project), nil, scope, DefaultLimits())
			require.NoError(t, err)
			value, _, err := e.Evaluate(context.Background(), func(Reference) *testpilotspb.Value { return source }, 1000)
			require.NoError(t, err)
			require.Equal(t, tc.present, value.GetBoolValue())
			path.Segments[1].Selector = &testpilotspb.FieldPathSegment_Presence{Presence: &testpilotspb.PresenceSelector{}}
			e, err = c.BindExpression(project, nil, scope, DefaultLimits())
			require.NoError(t, err)
			value, _, err = e.Evaluate(context.Background(), func(Reference) *testpilotspb.Value { return source }, 1000)
			require.NoError(t, err)
			var expected []*testpilotspb.Value
			for _, v := range tc.presence {
				expected = append(expected, boolean(v))
			}
			require.True(t, proto.Equal(&testpilotspb.Value{Value: &testpilotspb.Value_ListValue{ListValue: &testpilotspb.ValueList{Values: expected}}}, value))
		})
	}
}
