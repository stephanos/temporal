package ir

import (
	"context"
	"math"
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestCELNominalEnumReferencesCannotCrossDescriptors(t *testing.T) {
	catalog := fixtureCatalog(t)
	scope := map[Reference]Binding{
		{Kind: SlotReference, ID: "fixture"}: {Type: boundType(t, catalog, named("fixture.State", true)), Available: true},
		{Kind: SlotReference, ID: "outcome"}: {Type: boundType(t, catalog, named("temporal.server.api.testpilot.v1.InstructionOutcomeStatus", true)), Available: true},
	}
	_, err := catalog.BindExpression(programSite, equal(slot("fixture"), slot("outcome")), nil, scope, DefaultLimits())
	require.Error(t, err)
}

func TestCELConditionalBranchesRetainDescriptorIdentity(t *testing.T) {
	catalog := fixtureCatalog(t)
	scope := map[Reference]Binding{
		{Kind: SlotReference, ID: "fixture"}: {Type: boundType(t, catalog, named("fixture.State", true)), Available: true},
		{Kind: SlotReference, ID: "outcome"}: {Type: boundType(t, catalog, named("temporal.server.api.testpilot.v1.InstructionOutcomeStatus", true)), Available: true},
	}
	source := equal(slot("fixture"), slot("outcome"))
	call := source.Cel.Expr.GetCallExpr()
	call.Function = "_?_:_"
	call.Args = append([]*celpb.Expr{{Id: 100, ExprKind: &celpb.Expr_ConstExpr{ConstExpr: &celpb.Constant{ConstantKind: &celpb.Constant_BoolValue{BoolValue: true}}}}}, call.Args...)
	_, err := catalog.BindExpression(programSite, source, nil, scope, DefaultLimits())
	require.Error(t, err)
}

func TestCELMessageSelectionRequiresTheAdmittedBindingPath(t *testing.T) {
	catalog := fixtureCatalog(t)
	typ := boundType(t, catalog, named("fixture.Payload", false))
	source := slot("message")
	source.Cel.Expr = &celpb.Expr{Id: 100, ExprKind: &celpb.Expr_SelectExpr{SelectExpr: &celpb.Expr_Select{Operand: source.Cel.Expr, Field: "text"}}}
	_, err := catalog.BindExpression(programSite, source, nil, map[Reference]Binding{{Kind: SlotReference, ID: "message"}: {Type: typ, Available: true}}, DefaultLimits())
	require.Equal(t, &Error{Category: Unsupported, Path: "program.guard.cel.expr[1]", Detail: "message projections require an admitted binding path"}, err)
}

func TestCELValueAdapterRoundTripsWithoutGlobalDescriptors(t *testing.T) {
	catalog := fixtureCatalog(t)
	for _, test := range []struct {
		name   string
		schema *testpilotspb.ValueType
		value  *celpb.Value
	}{
		{"signed", scalar(testpilotspb.SCALAR_KIND_INT64), &celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: math.MinInt64}}},
		{"unsigned", scalar(testpilotspb.SCALAR_KIND_UINT64), &celpb.Value{Kind: &celpb.Value_Uint64Value{Uint64Value: math.MaxUint64}}},
		{"bytes", scalar(testpilotspb.SCALAR_KIND_BYTES), &celpb.Value{Kind: &celpb.Value_BytesValue{BytesValue: []byte{0, 255}}}},
		{"enum", named("fixture.State", true), enumLiteral("READY")},
		{"unknown runtime enum", named("fixture.State", true), &celpb.Value{Kind: &celpb.Value_EnumValue{EnumValue: &celpb.EnumValue{Type: "fixture.State", Value: 99}}}},
		{"list", &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Repeated{Repeated: &testpilotspb.RepeatedType{Element: scalar(testpilotspb.SCALAR_KIND_INT64).GetSingular()}}}, &celpb.Value{Kind: &celpb.Value_ListValue{ListValue: &celpb.ListValue{Values: []*celpb.Value{signed("9223372036854775807")}}}}},
		{"catalog message with unknown wire", named("fixture.Payload", false), &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{10, 1, 'x', 0x78, 1}}}}},
		{"opaque Any", &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Any{Any: &emptypb.Empty{}}}}}, &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: &anypb.Any{TypeUrl: "private/Unregistered", Value: []byte{255}}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			typ := boundType(t, catalog, test.schema)
			reference := Reference{Kind: SlotReference, ID: "value"}
			bound, err := catalog.BindExpression(programSite, slot("value"), &typ, map[Reference]Binding{reference: {Type: typ, Available: true}}, DefaultLimits())
			require.NoError(t, err)
			actual, work, err := bound.Evaluate(t.Context(), func(Reference) *celpb.Value { return test.value }, 100000)
			require.NoError(t, err)
			require.True(t, proto.Equal(test.value, actual))
			require.Positive(t, work)
			if test.name == "unknown runtime enum" {
				comparison, err := catalog.BindExpression(programSite, equal(slot("value"), literal(enumLiteral("READY"))), nil, map[Reference]Binding{reference: {Type: typ, Available: true}}, DefaultLimits())
				require.NoError(t, err)
				matched, _, err := comparison.Evaluate(t.Context(), func(Reference) *celpb.Value { return test.value }, 100000)
				require.NoError(t, err)
				require.False(t, matched.GetBoolValue())
				require.EqualValues(t, 99, test.value.GetEnumValue().Value)
			}
		})
	}
}

func TestCELNativeDurationComparisonRetainsFullProtobufRange(t *testing.T) {
	catalog := fixtureCatalog(t)
	durationType := boundType(t, catalog, named("google.protobuf.Duration", false))
	value := func(seconds int64, nanos int32) *celpb.Value {
		envelope, err := anypb.New(&durationpb.Duration{Seconds: seconds, Nanos: nanos})
		require.NoError(t, err)
		return &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: envelope}}
	}
	reference := Reference{Kind: SlotReference, ID: "duration"}
	scope := map[Reference]Binding{reference: {Type: durationType, Available: true}}
	maximum := value(315576000000, 999999999)
	for _, test := range []struct {
		operator string
		right    *celpb.Value
		want     bool
	}{
		{"_==_", maximum, true},
		{"_>_", value(315576000000, 999999998), true},
		{"_<_", value(-315576000000, 0), false},
	} {
		bound, err := catalog.BindExpression(programSite, cel.Compare(test.operator, slot("duration"), literal(test.right)), nil, scope, DefaultLimits())
		require.NoError(t, err)
		actual, _, err := bound.Evaluate(t.Context(), func(Reference) *celpb.Value { return maximum }, 100000)
		require.NoError(t, err)
		require.Equal(t, test.want, actual.GetBoolValue())
	}
	_, err := catalog.BindExpression(programSite, cel.Compare("_+_", slot("duration"), literal(maximum)), nil, scope, DefaultLimits())
	require.ErrorContains(t, err, "unsupported CEL function")
}

func TestCELRestrictedAdmissionAndRuntimeOutcomes(t *testing.T) {
	catalog := fixtureCatalog(t)
	booleanType := boundType(t, catalog, scalar(testpilotspb.SCALAR_KIND_BOOLEAN))
	scope := map[Reference]Binding{{Kind: SlotReference, ID: "missing"}: {Type: booleanType}}
	bound, err := catalog.BindExpression(programSite, cel.All(slot("missing"), literal(boolean(false))), &booleanType, scope, DefaultLimits())
	require.NoError(t, err)
	result, _, err := bound.Evaluate(t.Context(), func(Reference) *celpb.Value { return nil }, 10000)
	require.NoError(t, err)
	require.False(t, result.GetBoolValue())
	bound, err = catalog.BindExpression(programSite, equal(slot("missing"), literal(boolean(true))), &booleanType, scope, DefaultLimits())
	require.NoError(t, err)
	_, _, err = bound.Evaluate(t.Context(), func(Reference) *celpb.Value { return nil }, 10000)
	require.ErrorContains(t, err, "optional.none() dereference")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err = bound.Evaluate(ctx, func(Reference) *celpb.Value { return nil }, 10000)
	require.ErrorIs(t, err, context.Canceled)
	_, _, err = bound.Evaluate(t.Context(), func(Reference) *celpb.Value { return boolean(true) }, 1)
	require.Error(t, err)
	forbidden := cel.Any(literal(boolean(true)), slot("unknown"))
	_, err = catalog.BindExpression(programSite, forbidden, &booleanType, scope, DefaultLimits())
	require.ErrorContains(t, err, "not declared")
	unknown := literal(boolean(true))
	unknown.Cel.Expr.ProtoReflect().SetUnknown([]byte{0x78, 1})
	_, err = catalog.BindExpression(programSite, unknown, nil, nil, DefaultLimits())
	require.Error(t, err)
}

func TestCELNativeCollectionASTRoundTripsAndMapMiss(t *testing.T) {
	catalog := fixtureCatalog(t)
	integer := func(id, value int64) *celpb.Expr {
		return &celpb.Expr{Id: id, ExprKind: &celpb.Expr_ConstExpr{ConstExpr: &celpb.Constant{ConstantKind: &celpb.Constant_Int64Value{Int64Value: value}}}}
	}
	textNode := func(id int64, value string) *celpb.Expr {
		return &celpb.Expr{Id: id, ExprKind: &celpb.Expr_ConstExpr{ConstExpr: &celpb.Constant{ConstantKind: &celpb.Constant_StringValue{StringValue: value}}}}
	}
	for _, test := range []struct {
		name string
		node *celpb.Expr
		want *celpb.Value
	}{
		{"list", &celpb.Expr{Id: 1, ExprKind: &celpb.Expr_ListExpr{ListExpr: &celpb.Expr_CreateList{Elements: []*celpb.Expr{integer(2, math.MinInt64), integer(3, math.MaxInt64)}}}}, &celpb.Value{Kind: &celpb.Value_ListValue{ListValue: &celpb.ListValue{Values: []*celpb.Value{signed("-9223372036854775808"), signed("9223372036854775807")}}}}},
		{"map", &celpb.Expr{Id: 1, ExprKind: &celpb.Expr_StructExpr{StructExpr: &celpb.Expr_CreateStruct{Entries: []*celpb.Expr_CreateStruct_Entry{
			{Id: 2, KeyKind: &celpb.Expr_CreateStruct_Entry_MapKey{MapKey: integer(3, 2)}, Value: textNode(4, "b")},
			{Id: 5, KeyKind: &celpb.Expr_CreateStruct_Entry_MapKey{MapKey: integer(6, 1)}, Value: textNode(7, "a")},
		}}}}, &celpb.Value{Kind: &celpb.Value_MapValue{MapValue: &celpb.MapValue{Entries: []*celpb.MapValue_Entry{{Key: signed("1"), Value: text("a")}, {Key: signed("2"), Value: text("b")}}}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := &testpilotspb.Expression{Cel: &celpb.ParsedExpr{Expr: test.node}}
			bound, err := catalog.BindExpression(programSite, source, nil, nil, DefaultLimits())
			require.NoError(t, err)
			actual, _, err := bound.Evaluate(t.Context(), func(Reference) *celpb.Value { return nil }, 10000)
			require.NoError(t, err)
			require.True(t, proto.Equal(test.want, actual))
			if test.name == "map" {
				missing := proto.CloneOf(source)
				missing.Cel.Expr = &celpb.Expr{Id: 100, ExprKind: &celpb.Expr_CallExpr{CallExpr: &celpb.Expr_Call{Function: "_[_]", Args: []*celpb.Expr{missing.Cel.Expr, integer(101, 3)}}}}
				bound, err = catalog.BindExpression(programSite, missing, nil, nil, DefaultLimits())
				require.NoError(t, err)
				_, _, err = bound.Evaluate(t.Context(), func(Reference) *celpb.Value { return nil }, 10000)
				require.ErrorContains(t, err, "no such key")
			}
		})
	}
}
