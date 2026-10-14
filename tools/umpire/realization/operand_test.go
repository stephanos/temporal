package realization

// TypeOf is the one reading of an operand's types. These cases read it alone, with the descriptor and
// the paths a caller gives it.

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func opLiteral(value *umpirespb.ProtoValue) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: value}}
}

func opEqual(left, right *umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{Left: left, Right: right}}}
}

func opAll(operands ...*umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_All{All: &umpirespb.All{Operands: operands}}}
}

var (
	opText     = admWritten(&umpirespb.ProtoValue_Text{Text: "a"})
	opNumber   = admWritten(&umpirespb.ProtoValue_Number{Number: 1})
	opFlag     = opLiteral(&umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Flag{Flag: true}})
	opEnumName = opLiteral(&umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_EnumName{EnumName: "COMMITMENT_DURABLE"}})
	opPayload  = &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &emptypb.Empty{}}}
)

func TestTypeOfShapesWhatAnOperandComputes(t *testing.T) {
	evidence := (&umpirespb.Evidence{}).ProtoReflect().Descriptor()
	for name, test := range map[string]struct {
		operand *umpirespb.Operand
		want    Typed
	}{
		"a text written out":       {opText, Typed{Shape: TextShape}},
		"a name a Case binds":      {opLiteral(&umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Named{Named: &umpirespb.Name{Prefix: "n"}}}), Typed{Shape: TextShape}},
		"a flag written out":       {opFlag, Typed{Shape: ConditionShape}},
		"a number written out":     {opNumber, Typed{Shape: NumberShape}},
		"an enum value written":    {opEnumName, Typed{Shape: EnumShape, Name: "COMMITMENT_DURABLE"}},
		"a value of no kind":       {opLiteral(&umpirespb.ProtoValue{}), Typed{Shape: OtherShape}},
		"an environment binding":   {&umpirespb.Operand{Kind: &umpirespb.Operand_Environment{Environment: "namespace"}}, Typed{Shape: TextShape}},
		"the run's id":             {&umpirespb.Operand{Kind: &umpirespb.Operand_Run{Run: &emptypb.Empty{}}}, Typed{Shape: TextShape}},
		"a learned value":          {&umpirespb.Operand{Kind: &umpirespb.Operand_LearnedValue{LearnedValue: "id"}}, Typed{Shape: TextShape}},
		"the projected value":      {opPayload, Typed{Shape: MessageShape, Message: evidence}},
		"a path, with no paths":    {admPayload("id"), Typed{}},
		"a presence":               {&umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: admPayload("id")}}}, Typed{Shape: ConditionShape}},
		"a comparison of one type": {opEqual(opText, opText), Typed{Shape: ConditionShape}},
		"an order of numbers":      {admGreater(opNumber, opNumber), Typed{Shape: ConditionShape}},
		"a negation":               {admNot(opFlag), Typed{Shape: ConditionShape}},
		"a conjunction":            {opAll(opFlag, admNot(opFlag)), Typed{Shape: ConditionShape}},
		"an operand of no kind":    {&umpirespb.Operand{}, Typed{}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := TypeOf(test.operand, evidence, nil)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestTypeOfSaysWhatIsMistyped(t *testing.T) {
	for name, test := range map[string]struct {
		operand *umpirespb.Operand
		says    string
	}{
		"a path of a text":               {&umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: "first", Of: opText}}}, "reads first of a text, which is no message"},
		"a comparison of two types":      {opEqual(opText, opNumber), "compares a text with a number"},
		"a comparison of a message":      {opEqual(opPayload, opText), "compares a message"},
		"two enum values written out":    {opEqual(opEnumName, opEnumName), "compares two enum values it writes out"},
		"an order of a text":             {admGreater(opNumber, opText), "orders a text, and only numbers are ordered"},
		"a negation of a number":         {admNot(opNumber), "negates a number, and only a condition is negated"},
		"a conjunction with a text":      {opAll(opFlag, opText), "joins a text, and only conditions are joined"},
		"a mistype under a conjunction":  {opAll(opFlag, admNot(opNumber)), "negates a number, and only a condition is negated"},
		"a mistype under a presence":     {&umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: admGreater(opText, opNumber)}}}, "orders a text, and only numbers are ordered"},
		"the first of two mistyped ends": {opEqual(admNot(opText), admNot(opNumber)), "negates a text, and only a condition is negated"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := TypeOf(test.operand, nil, nil)
			require.Equal(t, &Mistype{Says: test.says}, err)
		})
	}
}

// What a path is of is the caller's to say: TypeOf asks its paths with the type it read the path of.
func TestTypeOfAsksItsPathsForAPath(t *testing.T) {
	evidence := (&umpirespb.Evidence{}).ProtoReflect().Descriptor()
	type asked struct {
		of   Typed
		path string
	}
	var calls []asked
	paths := func(of Typed, path string) (Typed, error) {
		calls = append(calls, asked{of, path})
		return Typed{Shape: NumberShape}, nil
	}
	got, err := TypeOf(admGreater(admPayload("position.line"), opNumber), evidence, paths)
	require.NoError(t, err)
	require.Equal(t, Typed{Shape: ConditionShape}, got)
	require.Equal(t, []asked{{Typed{Shape: MessageShape, Message: evidence}, "position.line"}}, calls)
}

// A reader that types paths may find several values at one, where a path fans out over a repeated
// field. Several values are no operand of a comparison or an order, and have no fields.
func TestSeveralValuesAreNoOperand(t *testing.T) {
	several := func(Typed, string) (Typed, error) { return Typed{Shape: SeveralShape}, nil }
	path := admPayload("items[*].name")
	for name, test := range map[string]struct {
		operand *umpirespb.Operand
		says    string
	}{
		"compared": {&umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{Left: admWritten(&umpirespb.ProtoValue_Text{Text: "a"}), Right: path}}}, "compares several values"},
		"ordered":  {admGreater(path, admWritten(&umpirespb.ProtoValue_Number{Number: 0})), "orders several values, and only numbers are ordered"},
		"read":     {&umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: "first", Of: path}}}, "reads first of several values, which is no message"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := TypeOf(test.operand, nil, several)
			require.Equal(t, &Mistype{Says: test.says}, err)
		})
	}
}
