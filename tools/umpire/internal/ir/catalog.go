// Package ir compiles the closed Case type and expression vocabulary without target I/O.
package ir

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type ErrorCategory string

const (
	Malformed     ErrorCategory = "malformed"
	Unknown       ErrorCategory = "unknown"
	TypeMismatch  ErrorCategory = "type_mismatch"
	Unavailable   ErrorCategory = "unavailable"
	Unsupported   ErrorCategory = "unsupported"
	LimitExceeded ErrorCategory = "limit_exceeded"
)

type Error struct {
	Category ErrorCategory
	Path     string
	Detail   string
}

func (e *Error) Error() string { return fmt.Sprintf("%s at %s: %s", e.Category, e.Path, e.Detail) }

func invalid(category ErrorCategory, path, detail string) error {
	if len(path) > 256 {
		path = path[:256]
	}
	return &Error{Category: category, Path: path, Detail: detail}
}

type Limits struct{ Depth, Work, Bytes, Fanout int64 }

func DefaultLimits() Limits { return Limits{Depth: 64, Work: 100_000, Bytes: 16 << 20, Fanout: 10_000} }

func (l Limits) validate() error {
	hard := DefaultLimits()
	if l.Depth <= 0 || l.Depth > hard.Depth || l.Work <= 0 || l.Work > hard.Work || l.Bytes <= 0 || l.Bytes > hard.Bytes || l.Fanout <= 0 || l.Fanout > hard.Fanout {
		return invalid(LimitExceeded, "$", "limits must be positive and within hard ceilings")
	}
	return nil
}

type budget struct {
	limits      Limits
	work, bytes int64
}

func (b *budget) charge(depth, work, bytes int64, path string) error {
	if depth > b.limits.Depth || work < 0 || bytes < 0 || work > b.limits.Work-b.work || bytes > b.limits.Bytes-b.bytes {
		return invalid(LimitExceeded, path, "depth, work, or byte ceiling exceeded")
	}
	b.work += work
	b.bytes += bytes
	return nil
}

func inspect(message protoreflect.Message, depth int64, b *budget, path string) error {
	if !message.IsValid() {
		return invalid(Malformed, path, "nil message")
	}
	if err := b.charge(depth, 1, 0, path); err != nil {
		return err
	}
	if len(message.GetUnknown()) != 0 {
		return invalid(Unknown, path, "unknown protobuf fields")
	}
	var result error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		result = inspectField(field, value, depth, b, path+"."+string(field.Name()))
		return result == nil
	})
	return result
}

func inspectField(field protoreflect.FieldDescriptor, value protoreflect.Value, depth int64, b *budget, path string) error {
	if field.IsMap() {
		if int64(value.Map().Len()) > b.limits.Fanout {
			return invalid(LimitExceeded, path, "map collection ceiling exceeded")
		}
		var result error
		value.Map().Range(func(key protoreflect.MapKey, item protoreflect.Value) bool {
			result = inspectValue(field.MapKey(), key.Value(), depth, b, path)
			if result == nil {
				result = inspectValue(field.MapValue(), item, depth, b, path)
			}
			return result == nil
		})
		return result
	}
	if field.IsList() {
		if int64(value.List().Len()) > b.limits.Fanout {
			return invalid(LimitExceeded, path, "repeated collection ceiling exceeded")
		}
		for i := 0; i < value.List().Len(); i++ {
			if err := inspectValue(field, value.List().Get(i), depth, b, path); err != nil {
				return err
			}
		}
		return nil
	}
	return inspectValue(field, value, depth, b, path)
}

func inspectValue(field protoreflect.FieldDescriptor, value protoreflect.Value, depth int64, b *budget, path string) error {
	if field.Message() != nil {
		return inspect(value.Message(), depth+1, b, path)
	}
	if field.Enum() != nil && field.Enum().Values().ByNumber(value.Enum()) == nil {
		return invalid(Unknown, path, "undefined enum value")
	}
	size := int64(8)
	if field.Kind() == protoreflect.StringKind {
		size = int64(len(value.String()))
	}
	if field.Kind() == protoreflect.BytesKind {
		size = int64(len(value.Bytes()))
	}
	return b.charge(depth, 1, size, path)
}

type Catalog struct {
	files    *protoregistry.Files
	identity string
}

func NewCatalog(source *descriptorpb.FileDescriptorSet) (*Catalog, error) {
	if source == nil {
		return nil, invalid(Malformed, "catalog", "descriptor set is required")
	}
	b := budget{limits: DefaultLimits()}
	if err := inspect(source.ProtoReflect(), 1, &b, "catalog"); err != nil {
		return nil, err
	}
	snapshot := proto.CloneOf(source)
	seen := make(map[string]bool, len(snapshot.File))
	for _, file := range snapshot.File {
		if file.GetName() == "" || seen[file.GetName()] {
			return nil, invalid(Malformed, "catalog", "missing or duplicate file name")
		}
		seen[file.GetName()] = true
	}
	files, err := protodesc.NewFiles(snapshot)
	if err != nil {
		return nil, invalid(Malformed, "catalog", "invalid descriptor graph")
	}
	for _, intrinsic := range intrinsicEnums() {
		if supplied, err := files.FindDescriptorByName(intrinsic.FullName()); err == nil {
			enumeration, ok := supplied.(protoreflect.EnumDescriptor)
			if !ok || !proto.Equal(protodesc.ToEnumDescriptorProto(enumeration), protodesc.ToEnumDescriptorProto(intrinsic)) {
				return nil, invalid(TypeMismatch, "catalog", "conflicting intrinsic enum definition")
			}
		}
	}
	slices.SortFunc(snapshot.File, func(a, b *descriptorpb.FileDescriptorProto) int { return strings.Compare(a.GetName(), b.GetName()) })
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(snapshot)
	if err != nil {
		return nil, invalid(Malformed, "catalog", "descriptor serialization failed")
	}
	sum := sha256.Sum256(encoded)
	return &Catalog{files: files, identity: fmt.Sprintf("%x", sum)}, nil
}

func (c *Catalog) Identity() string { return c.identity }

func (c *Catalog) Method(name string) (protoreflect.MethodDescriptor, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 3 || parts[0] != "" || !protoreflect.FullName(parts[1]).IsValid() || !protoreflect.Name(parts[2]).IsValid() {
		return nil, invalid(Malformed, "method", "expected /fully.qualified.Service/Method")
	}
	descriptor, err := c.files.FindDescriptorByName(protoreflect.FullName(parts[1] + "." + parts[2]))
	if err != nil {
		return nil, invalid(Unknown, "method", "method is not in the catalog")
	}
	method, ok := descriptor.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, invalid(TypeMismatch, "method", "descriptor is not a method")
	}
	if method.IsStreamingClient() || method.IsStreamingServer() {
		return nil, invalid(Unsupported, "method", "streaming methods are unsupported")
	}
	return method, nil
}

func missing(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return reflected.IsNil()
	default:
		return false
	}
}

func inspectSurface(message protoreflect.Message, b *budget, path string) error {
	logicalDepth := b.limits.Depth
	b.limits.Depth = 256
	err := inspect(message, 1, b, path)
	b.limits.Depth = logicalDepth
	return err
}

// CheckSurface bounds traversal before callers clone or otherwise walk untrusted Case data.
func CheckSurface(source proto.Message, limits Limits) error {
	if err := limits.validate(); err != nil {
		return err
	}
	if missing(source) {
		return invalid(Malformed, "$", "message is required")
	}
	b := budget{limits: limits}
	return inspectSurface(source.ProtoReflect(), &b, "$")
}

func intrinsicEnums() []protoreflect.EnumDescriptor {
	return []protoreflect.EnumDescriptor{umpirespb.InstructionOutcomeStatus(0).Descriptor(), umpirespb.RunEventKind(0).Descriptor()}
}
