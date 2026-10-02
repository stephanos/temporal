package model

import (
	"regexp"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var payloadSegment = regexp.MustCompile(`^([a-z0-9_]+)(<([a-z0-9_]+)>)?$`)

// PayloadFields is the fields a path of a Run Event's payload names, in order: plain fields, and
// `oneof<member>` for one member of a oneof, each one value. With no descriptor it checks how the
// path is written and names no field.
func PayloadFields(of protoreflect.MessageDescriptor, path string) ([]protoreflect.FieldDescriptor, error) {
	if path == "" {
		return nil, mistype("reads an empty path")
	}
	segments := strings.Split(path, ".")
	parts := make([][]string, len(segments))
	for i, segment := range segments {
		if parts[i] = payloadSegment.FindStringSubmatch(segment); parts[i] == nil {
			return nil, mistype("reads %q of the path %s, and a guard reads a field or oneof<member>", segment, path)
		}
	}
	if of == nil {
		return nil, nil
	}
	fields := make([]protoreflect.FieldDescriptor, len(segments))
	for i, segment := range segments {
		name, member := protoreflect.Name(parts[i][1]), protoreflect.Name(parts[i][3])
		field := of.Fields().ByName(name)
		if member != "" {
			oneof := of.Oneofs().ByName(name)
			if oneof == nil || oneof.Fields().ByName(member) == nil {
				return nil, mistype("reads %s, and %s has no such member", segment, of.FullName())
			}
			field = oneof.Fields().ByName(member)
		}
		switch {
		case field == nil:
			return nil, mistype("reads %s, and %s has no field %s", path, of.FullName(), name)
		case field.IsList() || field.IsMap():
			return nil, mistype("reads %s, and %s holds several values", path, field.FullName())
		case i < len(segments)-1 && field.Message() == nil:
			return nil, mistype("reads %s, and %s is no message", path, field.FullName())
		default:
		}
		fields[i], of = field, field.Message()
	}
	return fields, nil
}

// payloadPath types a path of a Run Event's payload, as a guard reads one: a message, or one flag,
// text, enum value or signed integer.
func payloadPath(of Typed, path string) (Typed, error) {
	fields, err := PayloadFields(of.Message, path)
	if err != nil || fields == nil {
		return Typed{}, err
	}
	switch field := fields[len(fields)-1]; field.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return Typed{Shape: MessageShape, Message: field.Message()}, nil
	case protoreflect.BoolKind:
		return Typed{Shape: ConditionShape}, nil
	case protoreflect.StringKind:
		return Typed{Shape: TextShape}, nil
	case protoreflect.EnumKind:
		return Typed{Shape: EnumShape, Enum: field.Enum()}, nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return Typed{Shape: NumberShape}, nil
	default:
		return Typed{}, mistype("reads %s, which is of kind %s", field.FullName(), field.Kind())
	}
}

// GuardProblem is what is wrong with the guard of a Run Event source, or nil. It is the one check of
// a guard: admission makes it with no descriptor, the lowering to a Case with the descriptor of the
// payload, and the evaluation of the guard on a recorded Run with the descriptor of the payload it
// evaluates. A guard is made of the payload and of the flags, numbers, texts and enum values it writes
// out, is well typed, and is a condition.
func GuardProblem(guard *umpirespb.Operand, payload protoreflect.MessageDescriptor) error {
	if problem := overThePayload(guard); problem != "" {
		return &Mistype{Says: problem}
	}
	computes, err := TypeOf(guard, payload, payloadPath)
	if err != nil {
		return err
	}
	if computes.Shape != AnyShape && computes.Shape != ConditionShape {
		return mistype("is %s, and a guard is a condition", computes.Shape)
	}
	return nil
}

// payloadAlone ends what is wrong with a guard that reads something a recorded Run does not hold.
const payloadAlone = "; a Run Event's guard reads the event's payload alone"

// writtenInAGuard is what is wrong with a value a guard writes out, or empty. A guard is evaluated on
// a recorded Run, which holds no name a Case binds.
func writtenInAGuard(value *umpirespb.ProtoValue) string {
	switch value.GetKind().(type) {
	case *umpirespb.ProtoValue_Text, *umpirespb.ProtoValue_Flag, *umpirespb.ProtoValue_Number, *umpirespb.ProtoValue_EnumName:
		return ""
	case *umpirespb.ProtoValue_Named:
		return "writes out a name a Case binds" + payloadAlone
	default:
		return "writes out a value that is no text, flag, number or enum value"
	}
}

// overThePayload is what is wrong with what a Run Event's guard is made of, or empty: it reads the
// event's payload and what is written out, and nothing of the run.
func overThePayload(o *umpirespb.Operand) string {
	const alone = payloadAlone
	switch k := o.GetKind().(type) {
	case *umpirespb.Operand_Literal:
		return writtenInAGuard(k.Literal)
	case *umpirespb.Operand_Projected:
		return ""
	case *umpirespb.Operand_Run:
		return "reads the run's id" + alone
	case *umpirespb.Operand_Environment:
		return "reads the environment binding " + k.Environment + alone
	case *umpirespb.Operand_LearnedValue:
		return "reads the learned value " + k.LearnedValue + alone
	case *umpirespb.Operand_Path:
		return overThePayload(k.Path.GetOf())
	case *umpirespb.Operand_Present:
		return overThePayload(k.Present.GetOf())
	case *umpirespb.Operand_Equal:
		if left := overThePayload(k.Equal.GetLeft()); left != "" {
			return left
		}
		return overThePayload(k.Equal.GetRight())
	case *umpirespb.Operand_Greater:
		if left := overThePayload(k.Greater.GetLeft()); left != "" {
			return left
		}
		return overThePayload(k.Greater.GetRight())
	case *umpirespb.Operand_Not:
		return overThePayload(k.Not.GetOf())
	case *umpirespb.Operand_All:
		if len(k.All.GetOperands()) == 0 {
			return "is a conjunction of no operand"
		}
		for _, operand := range k.All.GetOperands() {
			if problem := overThePayload(operand); problem != "" {
				return problem
			}
		}
		return ""
	default:
		return "has an operand of no known kind"
	}
}
