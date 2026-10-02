package conformance

import (
	"errors"
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// This file reads the Run's own record as evidence: whether a Run Event is an occurrence of the
// evidence a realization declares with a Run Event source. The source names a kind of event, the
// command that records it, and a guard over the event's payload, the instruction outcome.
//
// What a well-formed guard is, admission and the lowering to a Case decide by goir.GuardProblem, and
// so does this file, against the descriptor of the payload it evaluates: a guard one of the three
// refuses is refused by all, in the same words, whatever the event holds. A guard that is well formed
// is then evaluated, and the one thing that can keep it from a value is the event: it compares,
// negates or joins a value the payload does not hold. Either way a guard that cannot be evaluated is
// an error, and is never read as a guard that does not hold.

// GuardError is a Run Event source's guard that cannot be evaluated on a Run Event.
type GuardError struct {
	Event   int64
	Message string
}

func (e *GuardError) Error() string {
	return fmt.Sprintf("run event %d: the guard %s", e.Event, e.Message)
}

var eventKinds = map[umpirespb.RunEventSource_Kind]testpilotspb.RunEventKind{
	umpirespb.RunEventSource_KIND_INSTRUCTION_COMPLETED: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED,
	umpirespb.RunEventSource_KIND_INSTRUCTION_TIMED_OUT: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT,
	umpirespb.RunEventSource_KIND_DIAGNOSTIC:            testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC,
}

// admits reports whether a Run Event is an occurrence of the evidence a Run Event source declares: an
// event of the source's kind, recorded for the source's command, that carries an instruction outcome
// the source's guard holds of, and, for a source that is the record of an attempt, the outcome of the
// attempt of that number. A source with no guard takes every such event.
func admits(source *umpirespb.RunEventSource, event *testpilotspb.RunEvent) (bool, error) {
	refused := func(says string) (bool, error) {
		return false, &GuardError{Event: event.GetSequence(), Message: says}
	}
	kind, known := eventKinds[source.GetKind()]
	if !known {
		return refused("is of a Run Event source of no known kind")
	}
	at := event.GetCoordinates()
	if event.GetKind() != kind || at.GetEntrypointId() != source.GetScript() || at.GetInstructionId() != source.GetCommand() || event.GetOutcome() == nil {
		return false, nil
	}
	// The record of an attempt is of the attempt its source names, by the number the Run records it
	// under; an outcome that records no attempt is the record of none. The script the source names is
	// not read here: a Run records an attempt at the command that carries it and under no script's
	// name, so the lowering to a Case refuses a realization whose command carries two activities.
	if of := source.GetAttempt(); of != nil && int64(event.GetOutcome().GetActivityAttempt().GetSdkAttempt()) != of.GetNumber() {
		return false, nil
	}
	if source.GetGuard() == nil {
		return true, nil
	}
	payload := event.GetOutcome().ProtoReflect()
	if err := umpiremodel.GuardProblem(source.GetGuard(), payload.Descriptor()); err != nil {
		return refused(err.Error())
	}
	held, err := evaluate(source.GetGuard(), payload)
	switch {
	case err != nil:
		return refused(err.Error())
	case held.absent:
		return refused("is an absent value, and a guard is a condition")
	default:
		return held.flag, nil
	}
}

// value is one value a well-formed guard computes. Its type is the one goir.TypeOf gave it, so only
// the part of that type is set: a flag, a number, a text or the name of an enum value, or a message.
type value struct {
	// absent is a value the payload does not hold: a message that is unset, a oneof member that is not
	// the one set, or a field of either.
	absent  bool
	flag    bool
	number  int64
	text    string
	message protoreflect.Message
}

// evaluate computes an operand of a well-formed guard over a Run Event's payload.
func evaluate(o *umpirespb.Operand, payload protoreflect.Message) (value, error) {
	switch k := o.GetKind().(type) {
	case *umpirespb.Operand_Literal:
		return value{flag: k.Literal.GetFlag(), number: k.Literal.GetNumber(), text: k.Literal.GetText() + k.Literal.GetEnumName()}, nil
	case *umpirespb.Operand_Projected:
		return value{message: payload}, nil
	case *umpirespb.Operand_Path:
		of, err := evaluate(k.Path.GetOf(), payload)
		if err != nil || of.absent {
			return of, err
		}
		return valueAt(of.message, k.Path.GetPath())
	case *umpirespb.Operand_Present:
		of, err := evaluate(k.Present.GetOf(), payload)
		// A scalar the payload holds is present whatever its value: presence does not tell a zero or an
		// empty text from one that was set. A guard that means a positive number says so.
		return value{flag: !of.absent}, err
	case *umpirespb.Operand_Equal:
		left, right, err := sides(k.Equal.GetLeft(), k.Equal.GetRight(), payload, "compares")
		return value{flag: left.flag == right.flag && left.number == right.number && left.text == right.text}, err
	case *umpirespb.Operand_Greater:
		left, right, err := sides(k.Greater.GetLeft(), k.Greater.GetRight(), payload, "orders")
		return value{flag: left.number > right.number}, err
	case *umpirespb.Operand_Not:
		of, err := needed(k.Not.GetOf(), payload, "negates")
		return value{flag: !of.flag}, err
	case *umpirespb.Operand_All:
		for _, operand := range k.All.GetOperands() {
			if joined, err := needed(operand, payload, "joins"); err != nil || !joined.flag {
				return joined, err
			}
		}
		return value{flag: true}, nil
	default:
		return value{}, errors.New("has an operand of no known kind")
	}
}

// needed evaluates an operand whose value another operand needs, which is an error where the payload
// does not hold it.
func needed(o *umpirespb.Operand, payload protoreflect.Message, verb string) (value, error) {
	of, err := evaluate(o, payload)
	if err == nil && of.absent {
		err = fmt.Errorf("%s an absent value", verb)
	}
	return of, err
}

// sides evaluates the two operands of a comparison.
func sides(left, right *umpirespb.Operand, payload protoreflect.Message, verb string) (l, r value, err error) {
	if l, err = needed(left, payload, verb); err != nil {
		return l, r, err
	}
	r, err = needed(right, payload, verb)
	return l, r, err
}

// valueAt is the value at a path of a message the payload holds. A message on the way that is unset,
// and a oneof member that is not the one set, leave the value absent; a scalar that is no member of a
// oneof is held always, at its default where nothing set it.
func valueAt(at protoreflect.Message, path string) (value, error) {
	fields, err := umpiremodel.PayloadFields(at.Descriptor(), path)
	if err != nil {
		return value{}, err
	}
	last := fields[len(fields)-1]
	// The last segment is told by its position: a recursive message can name one field twice.
	for i, field := range fields {
		if field.HasPresence() && !at.Has(field) {
			return value{absent: true}, nil
		}
		if i < len(fields)-1 {
			at = at.Get(field).Message()
		}
	}
	switch read := at.Get(last); last.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return value{message: read.Message()}, nil
	case protoreflect.BoolKind:
		return value{flag: read.Bool()}, nil
	case protoreflect.StringKind:
		return value{text: read.String()}, nil
	case protoreflect.EnumKind:
		name := last.Enum().Values().ByNumber(read.Enum())
		if name == nil {
			return value{}, fmt.Errorf("reads %s, whose value %d its enum does not name", last.FullName(), read.Enum())
		}
		return value{text: string(name.Name())}, nil
	default:
		return value{number: read.Int()}, nil
	}
}
