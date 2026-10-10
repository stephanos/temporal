package runtimecel

import (
	"errors"
	"fmt"
	"strings"

	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/realization"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Lower translates only the runtime operand subset; finite Model expressions remain independent.
func Lower(o *umpirespb.Operand, projected protoreflect.MessageDescriptor, name func(*umpirespb.Name) string) (expression *testpilotspb.Expression, err error) {
	defer func() {
		var located *interp.Error
		if err != nil && o.GetPosition().GetFile() != "" && (!errors.As(err, &located) || located.Position == "") {
			err = interp.ErrorAt(o.GetPosition(), "%s", err)
		}
	}()
	lower := func(child *umpirespb.Operand) (*testpilotspb.Expression, error) { return Lower(child, projected, name) }
	sides := func(left, right *umpirespb.Operand, operator string) (*testpilotspb.Expression, error) {
		l, err := lower(left)
		if err != nil {
			return nil, err
		}
		r, err := lower(right)
		if err != nil {
			return nil, err
		}
		return cel.Compare(operator, l, r), nil
	}
	switch k := o.GetKind().(type) {
	case *umpirespb.Operand_Literal:
		var value *celpb.Value
		switch v := k.Literal.GetKind().(type) {
		case *umpirespb.ProtoValue_Text:
			value = &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: v.Text}}
		case *umpirespb.ProtoValue_Flag:
			value = &celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: v.Flag}}
		case *umpirespb.ProtoValue_Number:
			value = &celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: v.Number}}
		case *umpirespb.ProtoValue_Named:
			if name == nil {
				return nil, fmt.Errorf("a literal name is unresolved")
			}
			value = &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: name(v.Named)}}
		case *umpirespb.ProtoValue_EnumName:
			var err error
			value, err = Enum(v.EnumName)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("a literal operand is a text, a flag, a number, an enum value or a name")
		}
		return cel.Literal(value), nil
	case *umpirespb.Operand_Environment:
		return cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: k.Environment}}), nil
	case *umpirespb.Operand_LearnedValue:
		return cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_SlotId{SlotId: k.LearnedValue}}), nil
	case *umpirespb.Operand_Run:
		return cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Run{Run: &emptypb.Empty{}}}), nil
	case *umpirespb.Operand_Projected:
		return cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &emptypb.Empty{}}}), nil
	case *umpirespb.Operand_Path:
		if k.Path.GetOf().GetProjected() != nil && projected != nil {
			if _, err := realization.PayloadFields(projected, k.Path.GetPath()); err != nil {
				return nil, err
			}
		}
		of, err := lower(k.Path.GetOf())
		if err != nil {
			return nil, err
		}
		return cel.Path(of, k.Path.GetPath()), nil
	case *umpirespb.Operand_Present:
		of, err := lower(k.Present.GetOf())
		if err != nil {
			return nil, err
		}
		return cel.Present(of), nil
	case *umpirespb.Operand_Equal:
		return sides(k.Equal.GetLeft(), k.Equal.GetRight(), "_==_")
	case *umpirespb.Operand_Greater:
		return sides(k.Greater.GetLeft(), k.Greater.GetRight(), "_>_")
	case *umpirespb.Operand_Not:
		of, err := lower(k.Not.GetOf())
		if err != nil {
			return nil, err
		}
		return cel.Not(of), nil
	case *umpirespb.Operand_All:
		if len(k.All.GetOperands()) == 0 {
			return nil, fmt.Errorf("is a conjunction of no operand")
		}
		var operands []*testpilotspb.Expression
		for _, operand := range k.All.GetOperands() {
			child, err := lower(operand)
			if err != nil {
				return nil, err
			}
			operands = append(operands, child)
		}
		return cel.All(operands...), nil
	default:
		return nil, fmt.Errorf("an operand of no known kind")
	}
}

// Enum resolves authored names to descriptor-exact CEL enum values at lowering time.
func Enum(name string) (*celpb.Value, error) {
	var found protoreflect.EnumValueDescriptor
	var ambiguous bool
	var enums func(protoreflect.EnumDescriptors)
	enums = func(es protoreflect.EnumDescriptors) {
		for i := 0; i < es.Len(); i++ {
			e := es.Get(i)
			short := name[strings.LastIndex(name, ".")+1:]
			v := e.Values().ByName(protoreflect.Name(short))
			if v == nil || strings.Contains(name, ".") && name != string(v.FullName()) {
				continue
			}
			if found != nil && found.FullName() != v.FullName() {
				ambiguous = true
			}
			found = v
		}
	}
	var messages func(protoreflect.MessageDescriptors)
	messages = func(ms protoreflect.MessageDescriptors) {
		for i := 0; i < ms.Len(); i++ {
			enums(ms.Get(i).Enums())
			messages(ms.Get(i).Messages())
		}
	}
	protoregistry.GlobalFiles.RangeFiles(func(f protoreflect.FileDescriptor) bool { enums(f.Enums()); messages(f.Messages()); return true })
	if found == nil || ambiguous {
		return nil, fmt.Errorf("enum value %s is unresolved or ambiguous", name)
	}
	return &celpb.Value{Kind: &celpb.Value_EnumValue{EnumValue: &celpb.EnumValue{Type: string(found.Parent().FullName()), Value: int32(found.Number())}}}, nil
}
