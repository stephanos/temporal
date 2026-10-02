package testpilot

// What a realization writes against protobuf descriptors: the messages it writes out, and the field
// paths its commands assign, read and key evidence by. A realization that names a type, a field, a
// value or a path its descriptors do not have is rejected here, at the declaration, before any Case
// exists.

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/model/scalav2/goir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	historyEventMessage = "temporal.api.history.v1.HistoryEvent"
	// instructionOutcomeMessage is the payload of the Run Events that are evidence: the outcome of an
	// instruction, or of an activation it carries.
	instructionOutcomeMessage = "temporal.server.api.testpilot.v1.InstructionOutcome"
)

func locate(at *modelirspb.Position) string {
	if at.GetFile() == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", at.GetFile(), at.GetLine())
}

func errorAt(at *modelirspb.Position, format string, args ...any) error {
	return &goir.Error{Position: locate(at), Message: fmt.Sprintf(format, args...)}
}

// messageNamed is the descriptor of a protobuf message linked into this binary, by its full name.
func messageNamed(at *modelirspb.Position, name string) (protoreflect.MessageDescriptor, error) {
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
func methodNamed(at *modelirspb.Position, name string) (protoreflect.MethodDescriptor, error) {
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
func walk(at *modelirspb.Position, md protoreflect.MessageDescriptor, path string) (reached, error) {
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

func (w *writer) name(n *modelirspb.Name) string {
	if n.GetFixture() {
		return n.GetPrefix() + w.fixture + n.GetSuffix()
	}
	return n.GetPrefix() + n.GetSuffix()
}

// into writes a message out as the concrete message dst, whose type it must name.
func (w *writer) into(p *modelirspb.Proto, dst proto.Message) error {
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
func (w *writer) message(p *modelirspb.Proto, want protoreflect.MessageDescriptor) (*dynamicpb.Message, error) {
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
		if mapping, ok := f.GetValue().GetKind().(*modelirspb.ProtoValue_Mapping); ok {
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
func (w *writer) value(at *modelirspb.Position, fd protoreflect.FieldDescriptor, v *modelirspb.ProtoValue) (protoreflect.Value, error) {
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
	case *modelirspb.ProtoValue_Text:
		return text(k.Text, "a text")
	case *modelirspb.ProtoValue_Named:
		return text(w.name(k.Named), "a name")
	case *modelirspb.ProtoValue_RoleId:
		return text(k.RoleId, "a role")
	case *modelirspb.ProtoValue_Flag:
		if fd.Kind() != protoreflect.BoolKind {
			return crossed("a flag")
		}
		return protoreflect.ValueOfBool(k.Flag), nil
	case *modelirspb.ProtoValue_Utf8:
		if fd.Kind() != protoreflect.BytesKind {
			return crossed("bytes")
		}
		return protoreflect.ValueOfBytes([]byte(k.Utf8)), nil
	case *modelirspb.ProtoValue_EnumName:
		if fd.Kind() != protoreflect.EnumKind {
			return crossed("an enum value")
		}
		value := fd.Enum().Values().ByName(protoreflect.Name(k.EnumName))
		if value == nil {
			return protoreflect.Value{}, errorAt(at, "%s has no value %s", fd.Enum().FullName(), k.EnumName)
		}
		return protoreflect.ValueOfEnum(value.Number()), nil
	case *modelirspb.ProtoValue_Number:
		return number(at, fd, k.Number)
	case *modelirspb.ProtoValue_Message:
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

func number(at *modelirspb.Position, fd protoreflect.FieldDescriptor, n int64) (protoreflect.Value, error) {
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
