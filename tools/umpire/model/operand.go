package model

import (
	"fmt"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Shape is the type of the value an operand computes.
type Shape string

const (
	// AnyShape is a value its reader does not type: a path read where no descriptor is.
	AnyShape       Shape = ""
	ConditionShape Shape = "a condition"
	NumberShape    Shape = "a number"
	TextShape      Shape = "a text"
	EnumShape      Shape = "an enum value"
	MessageShape   Shape = "a message"
	SeveralShape   Shape = "several values"
	OtherShape     Shape = "a value that is no text, flag, integer or enum value"
)

// Typed is what an operand computes: its shape and, where a descriptor was read, the enum an enum
// value is of and the message a message is. An enum value written out has its name and no enum.
type Typed struct {
	Shape   Shape
	Enum    protoreflect.EnumDescriptor
	Name    string
	Message protoreflect.MessageDescriptor
}

// Mistype is what is wrong with the types of an operand, as the rest of a sentence about it.
type Mistype struct{ Says string }

func (m *Mistype) Error() string { return m.Says }

func mistype(format string, args ...any) error { return &Mistype{Says: fmt.Sprintf(format, args...)} }

// Paths types the value at a path of a value that is a message, by its descriptor where one was read,
// or of a value of any shape.
type Paths func(of Typed, path string) (Typed, error)

// TypeOf is the one reading of the types of an operand, which admission, the lowering to a Case and
// the evaluation of a guard all make, so that what is well typed for one is well typed for all. An
// order is of numbers, a negation and a conjunction of conditions, a comparison of two values of one
// type that is no message, and a path of a message. projected is the message the projected value is,
// or nil where no descriptor is read, and paths types a path; with no paths every path is of any
// shape. An operand of no known kind is of any shape: what a command and a guard may be made of is
// checked where each is admitted.
func TypeOf(o *umpirespb.Operand, projected protoreflect.MessageDescriptor, paths Paths) (Typed, error) {
	of := func(operands ...*umpirespb.Operand) ([]Typed, error) {
		out := make([]Typed, len(operands))
		for i, operand := range operands {
			var err error
			if out[i], err = TypeOf(operand, projected, paths); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	condition := Typed{Shape: ConditionShape}
	switch k := o.GetKind().(type) {
	case *umpirespb.Operand_Literal:
		return writtenType(k.Literal), nil
	case *umpirespb.Operand_Environment, *umpirespb.Operand_Run, *umpirespb.Operand_LearnedValue:
		return Typed{Shape: TextShape}, nil
	case *umpirespb.Operand_Projected:
		return Typed{Shape: MessageShape, Message: projected}, nil
	case *umpirespb.Operand_Path:
		from, err := of(k.Path.GetOf())
		switch {
		case err != nil:
			return Typed{}, err
		case from[0].Shape != AnyShape && from[0].Shape != MessageShape:
			return Typed{}, mistype("reads %s of %s, which is no message", k.Path.GetPath(), from[0].Shape)
		case paths == nil:
			return Typed{}, nil
		default:
			return paths(from[0], k.Path.GetPath())
		}
	case *umpirespb.Operand_Present:
		_, err := of(k.Present.GetOf())
		return condition, err
	case *umpirespb.Operand_Equal:
		sides, err := of(k.Equal.GetLeft(), k.Equal.GetRight())
		if err != nil {
			return Typed{}, err
		}
		return condition, compared(sides[0], sides[1])
	case *umpirespb.Operand_Greater:
		sides, err := of(k.Greater.GetLeft(), k.Greater.GetRight())
		return condition, every(sides, err, NumberShape, "orders", "numbers are ordered")
	case *umpirespb.Operand_Not:
		operands, err := of(k.Not.GetOf())
		return condition, every(operands, err, ConditionShape, "negates", "a condition is negated")
	case *umpirespb.Operand_All:
		operands, err := of(k.All.GetOperands()...)
		return condition, every(operands, err, ConditionShape, "joins", "conditions are joined")
	default:
		return Typed{}, nil
	}
}

// every is what is wrong with the first operand that is of neither one shape nor any, after what was
// wrong with typing them.
func every(operands []Typed, err error, want Shape, verb, only string) error {
	if err != nil {
		return err
	}
	for _, operand := range operands {
		if operand.Shape != AnyShape && operand.Shape != want {
			return mistype("%s %s, and only %s", verb, operand.Shape, only)
		}
	}
	return nil
}

// compared is what is wrong with comparing two values, or nil: each is one flag, number, text or
// enum value, they are of one type, and two enum values are of one enum, a written one being a name
// the other's enum has.
func compared(left, right Typed) error {
	for _, side := range []Typed{left, right} {
		if side.Shape == MessageShape || side.Shape == SeveralShape || side.Shape == OtherShape {
			return mistype("compares %s", side.Shape)
		}
	}
	switch {
	case left.Shape == AnyShape || right.Shape == AnyShape:
		return nil
	case left.Shape != right.Shape:
		return mistype("compares %s with %s", left.Shape, right.Shape)
	case left.Shape != EnumShape:
		return nil
	case left.Enum == nil && right.Enum == nil:
		return mistype("compares two enum values it writes out")
	case left.Enum != nil && right.Enum != nil && left.Enum.FullName() != right.Enum.FullName():
		return mistype("compares a value of %s with one of %s", left.Enum.FullName(), right.Enum.FullName())
	case left.Enum == nil && right.Enum.Values().ByName(protoreflect.Name(left.Name)) == nil:
		return mistype("compares a value of %s with %s, which it does not have", right.Enum.FullName(), left.Name)
	case right.Enum == nil && left.Enum.Values().ByName(protoreflect.Name(right.Name)) == nil:
		return mistype("compares a value of %s with %s, which it does not have", left.Enum.FullName(), right.Name)
	default:
		return nil
	}
}

// writtenType is the type of a value an operand writes out.
func writtenType(value *umpirespb.ProtoValue) Typed {
	switch v := value.GetKind().(type) {
	case *umpirespb.ProtoValue_Text, *umpirespb.ProtoValue_Named:
		return Typed{Shape: TextShape}
	case *umpirespb.ProtoValue_Flag:
		return Typed{Shape: ConditionShape}
	case *umpirespb.ProtoValue_Number:
		return Typed{Shape: NumberShape}
	case *umpirespb.ProtoValue_EnumName:
		return Typed{Shape: EnumShape, Name: v.EnumName}
	default:
		return Typed{Shape: OtherShape}
	}
}
