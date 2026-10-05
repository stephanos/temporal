package lower

// What a realization writes against protobuf descriptors: the messages it writes out, and the field
// paths its commands assign, read and key evidence by. A realization that names a type, a field, a
// value or a path its descriptors do not have is rejected here, at the declaration, before any Case
// exists.

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// instructionOutcomeMessage is the payload of the Run Events that are evidence: the outcome of an
// instruction, or of an activation it carries.
const instructionOutcomeMessage = "temporal.server.api.testpilot.v1.InstructionOutcome"

func locate(at *umpirespb.Position) string {
	if at.GetFile() == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", at.GetFile(), at.GetLine())
}

func errorAt(at *umpirespb.Position, format string, args ...any) error {
	return &umpiremodel.Error{Position: locate(at), Message: fmt.Sprintf(format, args...)}
}

// messageNamed is the descriptor of a protobuf message linked into this binary, by its full name.
func messageNamed(at *umpirespb.Position, name string) (protoreflect.MessageDescriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, errorAt(at, "no protobuf message %s", name)
	}
	md, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, errorAt(at, "%s is no protobuf message", name)
	}
	return md, nil
}

// methodNamed is the descriptor of a unary method, by its full name "/package.Service/Method".
func methodNamed(at *umpirespb.Position, name string) (protoreflect.MethodDescriptor, error) {
	service, method, ok := strings.Cut(strings.TrimPrefix(name, "/"), "/")
	if !ok || !strings.HasPrefix(name, "/") {
		return nil, errorAt(at, "%s is no method name of the form /package.Service/Method", name)
	}
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(service))
	if err != nil {
		return nil, errorAt(at, "no protobuf service %s", service)
	}
	sd, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, errorAt(at, "%s is no protobuf service", service)
	}
	md := sd.Methods().ByName(protoreflect.Name(method))
	switch {
	case md == nil:
		return nil, errorAt(at, "%s has no method %s", service, method)
	case md.IsStreamingClient() || md.IsStreamingServer():
		return nil, errorAt(at, "%s is a streaming method; a command calls a unary one", name)
	default:
		return md, nil
	}
}

// EvidenceElement is the message one piece of a kind of evidence of r is read as: the history event
// r's history read reads, the element or the one message of a read's response, or the payload of a
// Run Event. A poll's condition over the kind, and a Run Event's guard, read its fields.
func EvidenceElement(r *umpirespb.Realization, e *umpirespb.Evidence) (protoreflect.MessageDescriptor, error) {
	switch from := e.GetFrom().(type) {
	case *umpirespb.Evidence_History:
		return historyElement(r, e, from.History)
	case *umpirespb.Evidence_Read:
		return readFrom(e, from.Read, true)
	case *umpirespb.Evidence_Single:
		return readFrom(e, from.Single, false)
	case *umpirespb.Evidence_RunEvent:
		return messageNamed(e.GetPosition(), instructionOutcomeMessage)
	default:
		return nil, errorAt(e.GetPosition(), "evidence %s is recorded nowhere", e.GetId())
	}
}

// historyEvent is the message a realization's history is recorded as: what each read that lifts
// history evidence reads, by its method's response at its path. The realization's own reads declare
// it, so no message is assumed; nil where no read lifts history.
func historyEvent(r *umpirespb.Realization) (protoreflect.MessageDescriptor, error) {
	var found protoreflect.MessageDescriptor
	for _, c := range scriptCommands(r) {
		for _, read := range c.GetRpc().GetReads() {
			if !slices.ContainsFunc(read.GetTargets(), func(t *umpirespb.Target) bool { return t.GetLift() != "" }) {
				continue
			}
			md, err := readMessage(c, read)
			if err != nil {
				return nil, err
			}
			if found != nil && md.FullName() != found.FullName() {
				return nil, errorAt(c.GetPosition(), "command %s lifts history evidence from %s, a %s, and another read lifts it from a %s", c.GetId(),
					read.GetPath(), md.FullName(), found.FullName())
			}
			found = md
		}
	}
	return found, nil
}

// scriptCommands is every command of every script of r: each item's own and each it performs.
func scriptCommands(r *umpirespb.Realization) []*umpirespb.Command {
	var out []*umpirespb.Command
	for _, s := range r.GetScripts() {
		for _, item := range s.GetItems() {
			if item.GetCommand() != nil {
				out = append(out, item.GetCommand())
			}
			for _, p := range item.GetPerforms() {
				out = append(out, p.GetCommand())
			}
		}
	}
	return out
}

// readMessage is the message a call's read of its response reads at its path.
func readMessage(c *umpirespb.Command, read *umpirespb.ResponseRead) (protoreflect.MessageDescriptor, error) {
	at := c.GetPosition()
	method, err := methodNamed(at, c.GetRpc().GetMethod())
	if err != nil {
		return nil, err
	}
	end, err := walk(at, method.Output(), read.GetPath())
	if err != nil {
		return nil, err
	}
	if end.message() == nil {
		return nil, errorAt(at, "command %s lifts history evidence from %s, which reads no message", c.GetId(), read.GetPath())
	}
	return end.message(), nil
}

// historyElement is the history event a history kind of evidence is read as, which must have the
// oneof member `member` the kind names.
func historyElement(r *umpirespb.Realization, e *umpirespb.Evidence, member string) (protoreflect.MessageDescriptor, error) {
	event, err := historyEvent(r)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, errorAt(e.GetPosition(), "evidence %s is read from history, and no read of realization %s lifts history evidence", e.GetId(), r.GetName())
	}
	if fd := event.Fields().ByName(protoreflect.Name(member)); fd == nil || fd.ContainingOneof() == nil {
		return nil, errorAt(e.GetPosition(), "evidence %s: a history event has no attributes %s", e.GetId(), member)
	}
	return event, nil
}

// FieldAt is the field a path reaches in a message, as a realization's paths are read: a field, each
// element of a repeated field, or one member of a oneof.
func FieldAt(at *umpirespb.Position, md protoreflect.MessageDescriptor, path string) (protoreflect.FieldDescriptor, error) {
	end, err := walk(at, md, path)
	return end.field, err
}

var segment = regexp.MustCompile(`^([a-z0-9_]+)(\[\*\]|<([a-z0-9_]+)>)?$`)

// reached is where a field path ends: the field, and whether the path yields one value per element
// of a repeated field.
type reached struct {
	field  protoreflect.FieldDescriptor
	fanned bool
}

// message is the message type the path ends at, or nil when it ends at a scalar.
func (r reached) message() protoreflect.MessageDescriptor { return r.field.Message() }

// walk resolves a field path in a message: plain fields, `field[*]` over a repeated field, and
// `oneof<member>` for one member of a oneof. Any other selector of the path grammar is refused
// rather than read as something it is not.
func walk(at *umpirespb.Position, md protoreflect.MessageDescriptor, path string) (reached, error) {
	var out reached
	if path == "" {
		return out, errorAt(at, "an empty path names no field of %s", md.FullName())
	}
	segments := strings.Split(path, ".")
	for i, s := range segments {
		m := segment.FindStringSubmatch(s)
		if m == nil {
			return out, errorAt(at, "%q of the path %s is outside the paths a realization is checked for: a field, field[*] or oneof<member>", s, path)
		}
		name, member := protoreflect.Name(m[1]), protoreflect.Name(m[3])
		var fd protoreflect.FieldDescriptor
		if member != "" {
			oneof := md.Oneofs().ByName(name)
			if oneof == nil {
				return out, errorAt(at, "%s has no oneof %s", md.FullName(), name)
			}
			if fd = oneof.Fields().ByName(member); fd == nil {
				return out, errorAt(at, "%s of %s has no member %s", name, md.FullName(), member)
			}
		} else if fd = md.Fields().ByName(name); fd == nil {
			return out, errorAt(at, "%s has no field %s", md.FullName(), name)
		}
		each := m[2] == "[*]"
		switch {
		case each && !fd.IsList():
			return out, errorAt(at, "%s.%s is not repeated, and the path %s reads each element of it", md.FullName(), name, path)
		case !each && fd.IsList() && i < len(segments)-1:
			return out, errorAt(at, "%s.%s is repeated, and the path %s reads into it without [*]", md.FullName(), name, path)
		case fd.IsMap():
			return out, errorAt(at, "%s.%s is a map, which the path %s does not select an entry of", md.FullName(), name, path)
		default:
		}
		out.field, out.fanned = fd, out.fanned || each
		if i == len(segments)-1 {
			break
		}
		if md = fd.Message(); md == nil {
			return out, errorAt(at, "%s is no message, and the path %s reads a field of it", fd.FullName(), path)
		}
	}
	return out, nil
}

// writer writes out the protobuf messages of one realization for one Case.
type writer struct {
	fixture string
}

func (w *writer) name(n *umpirespb.Name) string {
	if n.GetFixture() {
		return n.GetPrefix() + w.fixture + n.GetSuffix()
	}
	return n.GetPrefix() + n.GetSuffix()
}

// into writes a message out as the concrete message dst, whose type it must name.
func (w *writer) into(p *umpirespb.Proto, dst proto.Message) error {
	written, err := w.message(p, dst.ProtoReflect().Descriptor())
	if err != nil {
		return err
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(written)
	if err != nil {
		return errorAt(p.GetPosition(), "%s: %v", p.GetMessage(), err)
	}
	if err := proto.Unmarshal(encoded, dst); err != nil {
		return errorAt(p.GetPosition(), "%s: %v", p.GetMessage(), err)
	}
	return nil
}

// message writes a message out. A field it does not name stays unset, so presence is the author's.
func (w *writer) message(p *umpirespb.Proto, want protoreflect.MessageDescriptor) (*dynamicpb.Message, error) {
	at := p.GetPosition()
	md, err := messageNamed(at, p.GetMessage())
	if err != nil {
		return nil, err
	}
	if want != nil && md.FullName() != want.FullName() {
		return nil, errorAt(at, "%s is written where a %s belongs", md.FullName(), want.FullName())
	}
	out := dynamicpb.NewMessage(md)
	for _, f := range p.GetFields() {
		fd := md.Fields().ByName(protoreflect.Name(f.GetName()))
		if fd == nil {
			return nil, errorAt(at, "%s has no field %s", md.FullName(), f.GetName())
		}
		if mapping, ok := f.GetValue().GetKind().(*umpirespb.ProtoValue_Mapping); ok {
			if !fd.IsMap() || fd.MapKey().Kind() != protoreflect.StringKind {
				return nil, errorAt(at, "%s is no map keyed by text, and a mapping is written into it", fd.FullName())
			}
			entries := out.Mutable(fd).Map()
			for _, e := range mapping.Mapping.GetEntries() {
				v, err := w.value(at, fd.MapValue(), e.GetValue())
				if err != nil {
					return nil, err
				}
				entries.Set(protoreflect.ValueOfString(e.GetKey()).MapKey(), v)
			}
			continue
		}
		if fd.IsList() || fd.IsMap() {
			return nil, errorAt(at, "%s holds several values, and one is written into it", fd.FullName())
		}
		v, err := w.value(at, fd, f.GetValue())
		if err != nil {
			return nil, err
		}
		out.Set(fd, v)
	}
	return out, nil
}

// value is one written value as the field's kind, which it must be of.
func (w *writer) value(at *umpirespb.Position, fd protoreflect.FieldDescriptor, v *umpirespb.ProtoValue) (protoreflect.Value, error) {
	crossed := func(written string) (protoreflect.Value, error) {
		return protoreflect.Value{}, errorAt(at, "%s is of kind %s, and %s is written into it", fd.FullName(), fd.Kind(), written)
	}
	text := func(s, written string) (protoreflect.Value, error) {
		if fd.Kind() != protoreflect.StringKind {
			return crossed(written)
		}
		return protoreflect.ValueOfString(s), nil
	}
	switch k := v.GetKind().(type) {
	case *umpirespb.ProtoValue_Text:
		return text(k.Text, "a text")
	case *umpirespb.ProtoValue_Named:
		return text(w.name(k.Named), "a name")
	case *umpirespb.ProtoValue_RoleId:
		return text(k.RoleId, "a role")
	case *umpirespb.ProtoValue_Flag:
		if fd.Kind() != protoreflect.BoolKind {
			return crossed("a flag")
		}
		return protoreflect.ValueOfBool(k.Flag), nil
	case *umpirespb.ProtoValue_Utf8:
		if fd.Kind() != protoreflect.BytesKind {
			return crossed("bytes")
		}
		return protoreflect.ValueOfBytes([]byte(k.Utf8)), nil
	case *umpirespb.ProtoValue_EnumName:
		if fd.Kind() != protoreflect.EnumKind {
			return crossed("an enum value")
		}
		value := fd.Enum().Values().ByName(protoreflect.Name(k.EnumName))
		if value == nil {
			return protoreflect.Value{}, errorAt(at, "%s has no value %s", fd.Enum().FullName(), k.EnumName)
		}
		return protoreflect.ValueOfEnum(value.Number()), nil
	case *umpirespb.ProtoValue_Number:
		return number(at, fd, k.Number)
	case *umpirespb.ProtoValue_Message:
		if fd.Message() == nil {
			return crossed("a message")
		}
		m, err := w.message(k.Message, fd.Message())
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfMessage(m), nil
	default:
		return crossed("a value of no kind")
	}
}

func number(at *umpirespb.Position, fd protoreflect.FieldDescriptor, n int64) (protoreflect.Value, error) {
	outside := func() (protoreflect.Value, error) {
		return protoreflect.Value{}, errorAt(at, "%d is outside what %s of kind %s holds", n, fd.FullName(), fd.Kind())
	}
	switch fd.Kind() {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		if n < math.MinInt32 || n > math.MaxInt32 {
			return outside()
		}
		return protoreflect.ValueOfInt32(int32(n)), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(n), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		if n < 0 || n > math.MaxUint32 {
			return outside()
		}
		return protoreflect.ValueOfUint32(uint32(n)), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if n < 0 {
			return outside()
		}
		return protoreflect.ValueOfUint64(uint64(n)), nil
	default:
		return protoreflect.Value{}, errorAt(at, "%s is of kind %s, and a number is written into it", fd.FullName(), fd.Kind())
	}
}
