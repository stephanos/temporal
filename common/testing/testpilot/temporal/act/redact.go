package act

import (
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

// redact clears every named field path inside m, looking through packed Anys, and reports
// whether it removed anything. A path is a message's full name followed by its field path.
func redact(m protoreflect.Message, paths []string) bool {
	if len(paths) == 0 || !m.IsValid() {
		return false
	}
	removed := false
	name := string(m.Descriptor().FullName())
	for _, path := range paths {
		if field, ok := strings.CutPrefix(path, name+"."); ok {
			removed = clearPath(m, strings.Split(field, ".")) || removed
		}
	}
	if packed, ok := m.Interface().(*anypb.Any); ok {
		inner, err := packed.UnmarshalNew()
		if err != nil || !redact(inner.ProtoReflect(), paths) {
			return removed
		}
		data, err := proto.MarshalOptions{Deterministic: true}.Marshal(inner)
		if err != nil {
			return removed
		}
		packed.Value = data
		return true
	}
	m.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch {
		case fd.IsList() && fd.Message() != nil:
			list := value.List()
			for i := range list.Len() {
				removed = redact(list.Get(i).Message(), paths) || removed
			}
		case fd.IsMap() && fd.MapValue().Message() != nil:
			value.Map().Range(func(_ protoreflect.MapKey, entry protoreflect.Value) bool {
				removed = redact(entry.Message(), paths) || removed
				return true
			})
		case fd.Message() != nil && !fd.IsList() && !fd.IsMap():
			removed = redact(value.Message(), paths) || removed
		default:
		}
		return true
	})
	return removed
}

// clearPath removes one field path below m.
func clearPath(m protoreflect.Message, path []string) bool {
	fd := m.Descriptor().Fields().ByName(protoreflect.Name(path[0]))
	if fd == nil || !m.Has(fd) {
		return false
	}
	if len(path) == 1 {
		m.Clear(fd)
		return true
	}
	if fd.Message() == nil || fd.IsList() || fd.IsMap() {
		return false
	}
	return clearPath(m.Mutable(fd).Message(), path[1:])
}
