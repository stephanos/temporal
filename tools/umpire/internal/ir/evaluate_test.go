package ir

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestRuntimeOperatorsAndExactBudget(t *testing.T) {
	c := fixtureCatalog(t)
	for _, tc := range []struct {
		name        string
		kind        umpirespb.ScalarKind
		a, b        *umpirespb.Value
		equal, less bool
	}{
		{"int64 precision", umpirespb.SCALAR_KIND_INT64, signed("9007199254740992"), signed("9007199254740993"), false, true},
		{"uint64", umpirespb.SCALAR_KIND_UINT64, unsigned("18446744073709551614"), unsigned("18446744073709551615"), false, true},
		{"natural", umpirespb.SCALAR_KIND_NATURAL, &umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: "99999999999999999999"}}, &umpirespb.Value{Value: &umpirespb.Value_Natural{Natural: "100000000000000000000"}}, false, true},
		{"negative", umpirespb.SCALAR_KIND_INT32, signed("-2"), signed("-1"), false, true},
		{"float32 precision", umpirespb.SCALAR_KIND_FLOAT, &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: 0.1}}, &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: float64(float32(0.1))}}, true, false},
		{"zero", umpirespb.SCALAR_KIND_DOUBLE, &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: math.Copysign(0, -1)}}, &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: 0}}, true, false},
		{"NaN", umpirespb.SCALAR_KIND_DOUBLE, &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: math.NaN()}}, &umpirespb.Value{Value: &umpirespb.Value_FloatingPoint{FloatingPoint: math.NaN()}}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			typ := boundType(t, c, scalar(tc.kind))
			scope := map[Reference]Binding{{Kind: SlotReference, ID: "a"}: {Type: typ, Available: true}, {Kind: SlotReference, ID: "b"}: {Type: typ, Available: true}}
			resolver := func(ref Reference) *umpirespb.Value {
				if ref.ID == "a" {
					return tc.a
				}
				return tc.b
			}
			for _, eq := range []bool{false, true} {
				source := equal(slot("a"), slot("b"))
				want := tc.equal
				if !eq {
					source = &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Compare{Compare: &umpirespb.CompareExpression{Left: slot("a"), Right: slot("b"), Operator: umpirespb.COMPARISON_OPERATOR_LESS_THAN}}}
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
	typ := boundType(t, c, scalar(umpirespb.SCALAR_KIND_TEXT))
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "missing"}: {Type: typ}}
	for _, source := range []*umpirespb.ValueExpression{all(present(slot("missing")), equal(slot("missing"), literal(text("x")))), negate(anyOf(present(slot("missing")), literal(boolean(true))))} {
		e, err := c.BindExpression(source, nil, scope, DefaultLimits())
		require.NoError(t, err)
		value, _, err := e.Evaluate(context.Background(), func(Reference) *umpirespb.Value { return nil }, 100)
		require.NoError(t, err)
		require.False(t, value.GetBoolValue())
	}
}
func TestRuntimePathsAndPresence(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	source := &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{0x12, 3, 0x0a, 1, 'x', 0x1a, 3, 0x0a, 1, 'a', 0x1a, 3, 0x0a, 1, 'b', 0x22, 7, 0x0a, 3, 'k', 'e', 'y', 0x10, 7}}}}
	require.NoError(t, c.CheckLiteral(source, typ, DefaultLimits()))
	wildcard := fieldPath("items", "text")
	wildcard.Segments[0].Selector = &umpirespb.FieldPathSegment_Repeated{Repeated: &umpirespb.RepeatedWildcard{}}
	lookup := fieldPath("labels")
	lookup.Segments[0].Selector = &umpirespb.FieldPathSegment_MapKey{MapKey: &umpirespb.MapKeySelector{Key: text("key")}}
	absence := fieldPath("child", "optional_text")
	absence.Segments[1].Selector = &umpirespb.FieldPathSegment_Presence{Presence: &umpirespb.PresenceSelector{}}
	for _, tc := range []struct {
		name string
		path *umpirespb.FieldPath
		want *umpirespb.Value
	}{
		{"nested", fieldPath("child", "text"), text("x")},
		{"wildcard", wildcard, &umpirespb.Value{Value: &umpirespb.Value_ListValue{ListValue: &umpirespb.ValueList{Values: []*umpirespb.Value{text("a"), text("b")}}}}},
		{"map", lookup, signed("7")},
		{"presence", absence, boolean(false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expr := &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Path{Path: &umpirespb.PathExpression{Source: slot("source"), Path: tc.path}}}
			// The guard supplies the admission fact while the value itself exercises the path.
			e, err := c.BindConditionedExpression([]Condition{{Expression: present(expr), Matches: true}}, expr, nil, map[Reference]Binding{{Kind: SlotReference, ID: "source"}: {Type: typ, Available: true}}, DefaultLimits())
			require.NoError(t, err)
			value, work, err := e.Evaluate(context.Background(), func(Reference) *umpirespb.Value { return source }, 10000)
			require.NoError(t, err)
			require.True(t, proto.Equal(tc.want, value), "%v", value)
			_, _, err = e.Evaluate(context.Background(), func(Reference) *umpirespb.Value { return source }, work)
			require.NoError(t, err)
		})
	}
	empty := &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload"}}}
	expr := &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Path{Path: &umpirespb.PathExpression{Source: slot("source"), Path: absence}}}
	e, err := c.BindExpression(expr, nil, map[Reference]Binding{{Kind: SlotReference, ID: "source"}: {Type: typ, Available: true}}, DefaultLimits())
	require.NoError(t, err)
	value, _, err := e.Evaluate(context.Background(), func(Reference) *umpirespb.Value { return empty }, 1000)
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
			source := &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: tc.wire}}}
			require.NoError(t, c.CheckLiteral(source, typ, DefaultLimits()))
			path := fieldPath("items", "optional_text")
			path.Segments[0].Selector = &umpirespb.FieldPathSegment_Repeated{Repeated: &umpirespb.RepeatedWildcard{}}
			project := &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Path{Path: &umpirespb.PathExpression{Source: slot("source"), Path: path}}}
			scope := map[Reference]Binding{{Kind: SlotReference, ID: "source"}: {Type: typ, Available: true}}
			e, err := c.BindExpression(present(project), nil, scope, DefaultLimits())
			require.NoError(t, err)
			value, _, err := e.Evaluate(context.Background(), func(Reference) *umpirespb.Value { return source }, 1000)
			require.NoError(t, err)
			require.Equal(t, tc.present, value.GetBoolValue())
			path.Segments[1].Selector = &umpirespb.FieldPathSegment_Presence{Presence: &umpirespb.PresenceSelector{}}
			e, err = c.BindExpression(project, nil, scope, DefaultLimits())
			require.NoError(t, err)
			value, _, err = e.Evaluate(context.Background(), func(Reference) *umpirespb.Value { return source }, 1000)
			require.NoError(t, err)
			var expected []*umpirespb.Value
			for _, v := range tc.presence {
				expected = append(expected, boolean(v))
			}
			require.True(t, proto.Equal(&umpirespb.Value{Value: &umpirespb.Value_ListValue{ListValue: &umpirespb.ValueList{Values: expected}}}, value))
		})
	}
}
