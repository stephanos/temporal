package ir

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRawProjectionPreservesWildcardAbsenceAndBudgets(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	for _, tc := range []struct {
		name    string
		wire    []byte
		present bool
		flags   []bool
	}{
		{"empty", nil, true, nil}, {"all present", []byte{0x1a, 2, 0x52, 0, 0x1a, 3, 0x52, 1, 'y'}, true, []bool{true, true}},
		{"mixed", []byte{0x1a, 2, 0x52, 0, 0x1a, 0}, false, []bool{true, false}}, {"all missing", []byte{0x1a, 0, 0x1a, 0}, false, []bool{false, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := dynamicpb.NewMessage(typ.Message())
			require.NoError(t, proto.Unmarshal(tc.wire, source))
			path := fieldPath("items", "optional_text")
			path.Segments[0].Selector = &umpirespb.FieldPathSegment_Repeated{Repeated: &umpirespb.RepeatedWildcard{}}
			for _, presence := range []bool{false, true} {
				if presence {
					path.Segments[1].Selector = &umpirespb.FieldPathSegment_Presence{Presence: &umpirespb.PresenceSelector{}}
				}
				p, err := c.BindPath(typ, path, DefaultLimits())
				require.NoError(t, err)
				got, work, err := p.Read(context.Background(), source, DefaultLimits())
				require.NoError(t, err)
				if presence {
					var flags []bool
					for _, v := range got.GetListValue().GetValues() {
						flags = append(flags, v.GetBoolValue())
					}
					require.Equal(t, tc.flags, flags)
				} else {
					require.Equal(t, tc.present, got != nil)
				}
				exact := DefaultLimits()
				exact.Work = work
				_, _, err = p.Read(context.Background(), source, exact)
				require.NoError(t, err)
				exact.Work--
				_, _, err = p.Read(context.Background(), source, exact)
				require.Error(t, err)
				_, _, err = p.Read(context.Background(), &timestamppb.Timestamp{}, DefaultLimits())
				require.Error(t, err)
			}
		})
	}
}

func TestRawResponseWorkIsIndependentOfBindingCap(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	source := dynamicpb.NewMessage(typ.Message())
	source.Set(typ.Message().Fields().ByName("text"), protoreflect.ValueOfString(strings.Repeat("x", 1<<18)))
	path, err := c.BindPath(typ, fieldPath("state"), DefaultLimits())
	require.NoError(t, err)
	limits := DefaultLimits()
	limits.Bytes = int64(proto.Size(source))
	limits.Work = 1 << 24
	value, work, err := path.Read(context.Background(), source, limits)
	require.NoError(t, err)
	require.Equal(t, int32(0), value.GetEnumValue().Number)
	require.Greater(t, work, DefaultLimits().Work)
	limits.Work = work
	_, _, err = path.Read(context.Background(), source, limits)
	require.NoError(t, err)
	limits.Work--
	_, _, err = path.Read(context.Background(), source, limits)
	require.Error(t, err)
	limits.Work = work
	limits.Bytes--
	_, _, err = path.Read(context.Background(), source, limits)
	require.Error(t, err)
}

func TestCompiledRequestProjectsUnrelatedResponse(t *testing.T) {
	c := fixtureCatalog(t)
	input := boundType(t, c, named("fixture.Payload", false))
	output := boundType(t, c, named("google.protobuf.Timestamp", false))
	write, err := c.BindPath(input, fieldPath("child", "text"), DefaultLimits())
	require.NoError(t, err)
	request, _, err := BuildRequest(context.Background(), input.Message(), []Write{{Path: write, Value: text("request")}}, DefaultLimits())
	require.NoError(t, err)
	read, err := c.BindPath(output, fieldPath("seconds"), DefaultLimits())
	require.NoError(t, err)
	value, _, err := read.Read(context.Background(), &timestamppb.Timestamp{Seconds: 123}, DefaultLimits())
	require.NoError(t, err)
	require.True(t, proto.Equal(signed("123"), value))
	_, _, err = read.Read(context.Background(), request, DefaultLimits())
	require.Error(t, err)
}

type cancelDuringWork struct {
	context.Context
	remaining int
}

func (c *cancelDuringWork) Err() error {
	c.remaining--
	if c.remaining <= 0 {
		return context.Canceled
	}
	return nil
}

func TestValueRuntimeCancellationAndMalformedInputs(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	path, err := c.BindPath(typ, fieldPath("text"), DefaultLimits())
	require.NoError(t, err)
	source := dynamicpb.NewMessage(typ.Message())
	source.Set(typ.Message().Fields().ByName("text"), protoreflect.ValueOfString("text"))
	for _, ctx := range []context.Context{nil, &cancelDuringWork{Context: context.Background(), remaining: 1}, &cancelDuringWork{Context: context.Background(), remaining: 5}} {
		result, _, err := path.Read(ctx, source, DefaultLimits())
		require.Error(t, err)
		require.Nil(t, result)
	}
	textType := boundType(t, c, scalar(umpirespb.SCALAR_KIND_TEXT))
	for _, value := range []*umpirespb.Value{nil, {}, {Value: (*umpirespb.Value_Text)(nil)}, {Value: &umpirespb.Value_Text{Text: string([]byte{0xff})}}} {
		result, _, err := SnapshotValue(context.Background(), value, textType, DefaultLimits())
		require.Error(t, err)
		require.Nil(t, result)
	}
	opaque := boundType(t, c, &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_OpaqueCapability{OpaqueCapability: &umpirespb.OpaqueCapabilityType{}}}}})
	_, _, err = SnapshotValue(context.Background(), text("secret"), opaque, DefaultLimits())
	require.Error(t, err)
}

func TestExecutionExpressionAccountsNestedCopies(t *testing.T) {
	c := fixtureCatalog(t)
	typ := boundType(t, c, named("fixture.Payload", false))
	source := &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/fixture.Payload", Value: []byte{0x12, 5, 0x12, 3, 0x0a, 1, 'x'}}}}
	path := &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Path{Path: &umpirespb.PathExpression{Source: slot("input"), Path: fieldPath("child", "child", "text")}}}
	e, err := c.BindConditionedExpression([]Condition{{Expression: present(path), Matches: true}}, path, nil, map[Reference]Binding{{Kind: SlotReference, ID: "input"}: {Type: typ, Available: true}}, DefaultLimits())
	require.NoError(t, err)
	resolve := func(Reference) *umpirespb.Value { return source }
	legacy, oldWork, err := e.Evaluate(context.Background(), resolve, 100000)
	require.NoError(t, err)
	value, work, err := e.EvaluateExecution(context.Background(), resolve, 100000)
	require.NoError(t, err)
	require.True(t, proto.Equal(legacy, value))
	require.Greater(t, work, oldWork)
	_, _, err = e.EvaluateExecution(context.Background(), resolve, work)
	require.NoError(t, err)
	_, _, err = e.EvaluateExecution(context.Background(), resolve, work-1)
	require.Error(t, err)
	source.GetMessageValue().Value = nil
	_, _, legacyErr := e.Evaluate(context.Background(), resolve, 100000)
	_, _, err = e.EvaluateExecution(context.Background(), resolve, 100000)
	require.Error(t, legacyErr)
	require.Error(t, err)
}

func TestResponseRejectsSameNameDivergentDescriptors(t *testing.T) {
	pinned := fixtureCatalog(t)
	typ := boundType(t, pinned, named("fixture.Payload", false))
	for _, tc := range []struct {
		name   string
		mutate func(*descriptorpb.FileDescriptorSet)
		reject bool
	}{
		{"equivalent independent", func(*descriptorpb.FileDescriptorSet) {}, false},
		{"wire compatible field", func(s *descriptorpb.FileDescriptorSet) {
			s.File[2].MessageType[0].Field[0].Type = descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum()
		}, true},
		{"nested enum", func(s *descriptorpb.FileDescriptorSet) { s.File[2].EnumType[0].Value[1].Number = proto.Int32(2) }, true},
		{"nested message", func(s *descriptorpb.FileDescriptorSet) {
			s.File[1].MessageType[0].Field[0].Type = descriptorpb.FieldDescriptorProto_TYPE_SINT64.Enum()
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := catalogFixture()
			tc.mutate(source)
			c, err := NewCatalog(source)
			require.NoError(t, err)
			actual := boundType(t, c, named("fixture.Payload", false))
			message := dynamicpb.NewMessage(actual.Message())
			snapshot, work, err := SnapshotMessage(context.Background(), message, typ.Message(), DefaultLimits())
			if tc.reject {
				require.Error(t, err)
				require.Nil(t, snapshot)
				return
			}
			require.NoError(t, err)
			require.NotSame(t, actual.Message(), typ.Message())
			limits := DefaultLimits()
			limits.Work = work
			_, _, err = SnapshotMessage(context.Background(), message, typ.Message(), limits)
			require.NoError(t, err)
			limits.Work--
			_, _, err = SnapshotMessage(context.Background(), message, typ.Message(), limits)
			require.Error(t, err)
		})
	}
}

func TestResponseEquivalentRepeatedBytes(t *testing.T) {
	set := catalogFixture()
	field := set.File[2].MessageType[0].Field[0]
	field.Type = descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum()
	field.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
	pinned, err := NewCatalog(set)
	require.NoError(t, err)
	source, err := NewCatalog(proto.Clone(set).(*descriptorpb.FileDescriptorSet))
	require.NoError(t, err)
	target := boundType(t, pinned, named("fixture.Payload", false)).Message()
	actual := boundType(t, source, named("fixture.Payload", false)).Message()
	for _, populated := range []bool{false, true} {
		message := dynamicpb.NewMessage(actual)
		if populated {
			message.Mutable(actual.Fields().Get(0)).List().Append(protoreflect.ValueOfBytes([]byte("value")))
		}
		snapshot, _, err := SnapshotMessage(context.Background(), message, target, DefaultLimits())
		require.NoError(t, err)
		before, err := proto.Marshal(message)
		require.NoError(t, err)
		after, err := proto.Marshal(snapshot)
		require.NoError(t, err)
		require.Equal(t, before, after)
	}
}
