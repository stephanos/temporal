package model

// A Run Event's guard reads the event's payload by paths. These cases read a path and a guard against
// a descriptor, as the lowering and the evaluation of a guard do, and without one, as admission does.

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestPayloadFieldsNamesTheFieldsOfAPath(t *testing.T) {
	evidence := (&umpirespb.Evidence{}).ProtoReflect().Descriptor()
	position := evidence.Fields().ByName("position")
	runEvent := evidence.Oneofs().ByName("from").Fields().ByName("run_event")
	for name, test := range map[string]struct {
		path string
		want []protoreflect.FieldDescriptor
	}{
		"a field":              {"id", []protoreflect.FieldDescriptor{evidence.Fields().ByName("id")}},
		"a field of a message": {"position.line", []protoreflect.FieldDescriptor{position, position.Message().Fields().ByName("line")}},
		"a member of a oneof":  {"from<run_event>.kind", []protoreflect.FieldDescriptor{runEvent, runEvent.Message().Fields().ByName("kind")}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := PayloadFields(evidence, test.path)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestPayloadFieldsRefusesAPathThePayloadDoesNotHave(t *testing.T) {
	evidence := (&umpirespb.Evidence{}).ProtoReflect().Descriptor()
	for name, test := range map[string]struct{ path, says string }{
		"no path":               {"", "reads an empty path"},
		"a fan-out":             {"fields[*]", `reads "fields[*]" of the path fields[*], and a guard reads a field or oneof<member>`},
		"no such field":         {"nope", fmt.Sprintf("reads nope, and %s has no field nope", evidence.FullName())},
		"a oneof by its name":   {"from", fmt.Sprintf("reads from, and %s has no field from", evidence.FullName())},
		"no such member":        {"from<nope>", fmt.Sprintf("reads from<nope>, and %s has no such member", evidence.FullName())},
		"several values":        {"fields", fmt.Sprintf("reads fields, and %s holds several values", evidence.Fields().ByName("fields").FullName())},
		"a field of no message": {"id.first", fmt.Sprintf("reads id.first, and %s is no message", evidence.Fields().ByName("id").FullName())},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := PayloadFields(evidence, test.path)
			require.Nil(t, got)
			require.Equal(t, &Mistype{Says: test.says}, err)
		})
	}
}

// With no descriptor only how the path is written is checked, and it names no field.
func TestPayloadFieldsWithoutADescriptorReadsHowThePathIsWritten(t *testing.T) {
	got, err := PayloadFields(nil, "any.oneof<member>.field")
	require.NoError(t, err)
	require.Nil(t, got)

	got, err = PayloadFields(nil, "items[*].name")
	require.Nil(t, got)
	require.Equal(t, &Mistype{Says: `reads "items[*]" of the path items[*].name, and a guard reads a field or oneof<member>`}, err)
}

func TestGuardProblemTypesAGuardAgainstThePayload(t *testing.T) {
	evidence := (&umpirespb.Evidence{}).ProtoReflect().Descriptor()
	model := (&umpirespb.Model{}).ProtoReflect().Descriptor()
	commitment := evidence.Fields().ByName("commitment").Enum()
	present := func(of *umpirespb.Operand) *umpirespb.Operand {
		return &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: of}}}
	}
	enumName := func(name string) *umpirespb.Operand {
		return opLiteral(&umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_EnumName{EnumName: name}})
	}
	for name, test := range map[string]struct {
		guard   *umpirespb.Operand
		payload protoreflect.MessageDescriptor
		says    string
	}{
		"a flag of the payload":            {admPayload("exhaustive"), evidence, ""},
		"an enum value the enum has":       {opEqual(admPayload("commitment"), enumName("COMMITMENT_DURABLE")), evidence, ""},
		"an order of a number":             {admGreater(admPayload("position.line"), opNumber), evidence, ""},
		"a path, with no descriptor":       {admPayload("exhaustive"), nil, ""},
		"a text":                           {admPayload("id"), evidence, "is a text, and a guard is a condition"},
		"a number":                         {admPayload("position.line"), evidence, "is a number, and a guard is a condition"},
		"an enum value":                    {admPayload("commitment"), evidence, "is an enum value, and a guard is a condition"},
		"a message":                        {admPayload("position"), evidence, "is a message, and a guard is a condition"},
		"an enum value the enum lacks":     {opEqual(admPayload("commitment"), enumName("NOPE")), evidence, fmt.Sprintf("compares a value of %s with NOPE, which it does not have", commitment.FullName())},
		"a field of a kind no guard reads": {present(admPayload("version")), model, fmt.Sprintf("reads %s, which is of kind uint32", model.Fields().ByName("version").FullName())},
		"a field the payload lacks":        {present(admPayload("nope")), evidence, fmt.Sprintf("reads nope, and %s has no field nope", evidence.FullName())},
		"the run's id":                     {present(&umpirespb.Operand{Kind: &umpirespb.Operand_Run{Run: &umpirespb.Empty{}}}), evidence, "reads the run's id; a Run Event's guard reads the event's payload alone"},
		"a conjunction of nothing":         {opAll(), evidence, "is a conjunction of no operand"},
	} {
		t.Run(name, func(t *testing.T) {
			err := GuardProblem(test.guard, test.payload)
			if test.says == "" {
				require.NoError(t, err)
				return
			}
			require.Equal(t, &Mistype{Says: test.says}, err)
		})
	}
}
