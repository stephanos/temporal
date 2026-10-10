// Package ir compiles the closed Case type and expression vocabulary without target I/O.
package ir

import (
	"cmp"
	"context"
	"crypto/sha256"
	"fmt"
	"iter"
	"math/bits"
	"slices"
	"strings"

	celpb "cel.dev/expr"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
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

type Limits struct{ Depth, Work, Bytes, Fanout int64 }

func DefaultLimits() Limits { return Limits{Depth: 64, Work: 100_000, Bytes: 16 << 20, Fanout: 10_000} }

// validateStructure bounds everything but the work ceiling, which a runtime caller supplies as its
// own remaining budget rather than as an admission bound.
func (l Limits) validateStructure() error {
	structure := l
	structure.Work = DefaultLimits().Work
	return structure.validate()
}
func (l Limits) validate() error {
	hard := DefaultLimits()
	if l.Depth <= 0 || l.Depth > hard.Depth || l.Work <= 0 || l.Work > hard.Work || l.Bytes <= 0 || l.Bytes > hard.Bytes || l.Fanout <= 0 || l.Fanout > hard.Fanout {
		return Invalid(LimitExceeded, "$", "limits must be positive and within hard ceilings")
	}
	return nil
}

type budget struct {
	ctx         context.Context
	limits      Limits
	work, bytes int64
	expand      Expansion
	payload     bool
	descriptors bool
	surfaceWork int64
}

func (b *budget) charge(depth, work, bytes int64, path string) error {
	if b.ctx != nil {
		if err := b.ctx.Err(); err != nil {
			return err
		}
		if bytes > b.limits.Work-work {
			return Invalid(LimitExceeded, path, "runtime work ceiling exceeded")
		}
		work += bytes
	}
	if depth > b.limits.Depth || work < 0 || bytes < 0 || work > b.limits.Work-b.work || bytes > b.limits.Bytes-b.bytes {
		return Invalid(LimitExceeded, path, "depth, work, or byte ceiling exceeded")
	}
	b.work += work
	b.bytes += bytes
	return nil
}

func inspect(message protoreflect.Message, depth int64, b *budget, path string) error {
	if !message.IsValid() {
		return Invalid(Malformed, path, "nil message")
	}
	if err := b.charge(depth, 1, 0, path); err != nil {
		return err
	}
	if len(message.GetUnknown()) != 0 && !b.payload {
		return Invalid(Unknown, path, "unknown protobuf fields")
	}

	if b.ctx == nil {
		var result error
		message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
			result = inspectField(field, value, depth, b, path+"."+string(field.Name()))
			return result == nil
		})
		return result
	}
	type entry struct {
		field protoreflect.FieldDescriptor
		value protoreflect.Value
	}
	var entries []entry
	var result error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		result = b.charge(depth, 1, 0, path)
		if result != nil {
			return false
		}
		entries = append(entries, entry{field: field, value: value})
		return true
	})
	if result != nil {
		return result
	}
	if err := b.charge(depth, int64(len(entries))*int64(bits.Len(uint(len(entries)))+1), 0, path); err != nil {
		return err
	}
	slices.SortFunc(entries, func(a, b entry) int { return cmp.Compare(a.field.Number(), b.field.Number()) })
	for _, entry := range entries {
		if err := inspectField(entry.field, entry.value, depth, b, path+"."+string(entry.field.Name())); err != nil {
			return err
		}
	}
	return nil
}

func inspectField(field protoreflect.FieldDescriptor, value protoreflect.Value, depth int64, b *budget, path string) error {
	if field.IsExtension() && !b.payload && !b.descriptors {
		return Invalid(Unsupported, path, "protobuf extensions are unsupported")
	}
	if field.IsMap() {
		return inspectMap(field, value, depth, b, path)
	}
	if field.IsList() {
		if b.expand != nil && field.Message() != nil {
			return inspectExpandedList(field, value.List(), depth, b, path)
		}
		if int64(value.List().Len()) > b.limits.Fanout {
			return Invalid(LimitExceeded, path, "repeated collection ceiling exceeded")
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

// inspectExpandedList inspects a repeated message field as the list b.expand writes it out as. The
// written-out length is bounded before any element is inspected, as the written-out message's own
// check bounds it.
func inspectExpandedList(field protoreflect.FieldDescriptor, list protoreflect.List, depth int64, b *budget, path string) error {
	var length int64
	for i := 0; i < list.Len(); i++ {
		count := int64(1)
		if expanded, _, ok := b.expand(field, list.Get(i).Message()); ok {
			count = int64(expanded)
		}
		if length += count; length > b.limits.Fanout {
			return Invalid(LimitExceeded, path, "repeated collection ceiling exceeded")
		}
	}
	for i := 0; i < list.Len(); i++ {
		_, elements, ok := b.expand(field, list.Get(i).Message())
		if !ok {
			if err := inspectValue(field, list.Get(i), depth, b, path); err != nil {
				return err
			}
			continue
		}
		for element := range elements {
			if err := inspect(element.ProtoReflect(), depth+1, b, path); err != nil {
				return err
			}
		}
	}
	return nil
}

func inspectValue(field protoreflect.FieldDescriptor, value protoreflect.Value, depth int64, b *budget, path string) error {
	if field.Message() != nil {
		return inspect(value.Message(), depth+1, b, path)
	}
	if field.Enum() != nil && field.Enum().Values().ByNumber(value.Enum()) == nil && !b.payload {
		return Invalid(Unknown, path, "undefined enum value")
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
		return nil, Invalid(Malformed, "catalog", "descriptor set is required")
	}
	b := budget{limits: DefaultLimits(), descriptors: true}
	if err := inspect(source.ProtoReflect(), 1, &b, "catalog"); err != nil {
		return nil, err
	}
	snapshot := proto.CloneOf(source)
	seen := make(map[string]bool, len(snapshot.File))
	for _, file := range snapshot.File {
		if file.GetName() == "" || seen[file.GetName()] {
			return nil, Invalid(Malformed, "catalog", "missing or duplicate file name")
		}
		seen[file.GetName()] = true
	}
	files, err := protodesc.NewFiles(snapshot)
	if err != nil {
		return nil, Invalid(Malformed, "catalog", "invalid descriptor graph")
	}
	for _, intrinsic := range intrinsicEnums() {
		if supplied, err := files.FindDescriptorByName(intrinsic.FullName()); err == nil {
			enumeration, ok := supplied.(protoreflect.EnumDescriptor)
			if !ok || !proto.Equal(protodesc.ToEnumDescriptorProto(enumeration), protodesc.ToEnumDescriptorProto(intrinsic)) {
				return nil, Invalid(TypeMismatch, "catalog", "conflicting intrinsic enum definition")
			}
		}
	}
	intrinsicDuration := (&durationpb.Duration{}).ProtoReflect().Descriptor()
	if supplied, err := files.FindDescriptorByName(intrinsicDuration.FullName()); err == nil {
		message, ok := supplied.(protoreflect.MessageDescriptor)
		if !ok || !SameMessage(message, intrinsicDuration) {
			return nil, Invalid(TypeMismatch, "catalog", "conflicting intrinsic duration definition")
		}
	}
	slices.SortFunc(snapshot.File, func(a, b *descriptorpb.FileDescriptorProto) int { return strings.Compare(a.GetName(), b.GetName()) })
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(snapshot)
	if err != nil {
		return nil, Invalid(Malformed, "catalog", "descriptor serialization failed")
	}
	sum := sha256.Sum256(encoded)
	return &Catalog{files: files, identity: fmt.Sprintf("%x", sum)}, nil
}

func (c *Catalog) Identity() string { return c.identity }

func (c *Catalog) Method(name string) (protoreflect.MethodDescriptor, error) {
	parts := strings.Split(name, "/")
	if len(parts) != 3 || parts[0] != "" || !protoreflect.FullName(parts[1]).IsValid() || !protoreflect.Name(parts[2]).IsValid() {
		return nil, Invalid(Malformed, "method", "expected /fully.qualified.Service/Method")
	}
	descriptor, err := c.files.FindDescriptorByName(protoreflect.FullName(parts[1] + "." + parts[2]))
	if err != nil {
		return nil, Invalid(Unknown, "method", "method is not in the catalog")
	}
	method, ok := descriptor.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, Invalid(TypeMismatch, "method", "descriptor is not a method")
	}
	if method.IsStreamingClient() || method.IsStreamingServer() {
		return nil, Invalid(Unsupported, "method", "streaming methods are unsupported")
	}
	return method, nil
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
	if IsNil(source) {
		return Invalid(Malformed, "$", "message is required")
	}
	b := budget{limits: limits}
	return inspectSurface(source.ProtoReflect(), &b, "$")
}

// Expansion writes out one element of a repeated message field as the count messages elements
// yields, or reports false when the element stands for itself.
type Expansion func(field protoreflect.FieldDescriptor, element protoreflect.Message) (count int, elements iter.Seq[proto.Message], ok bool)

// CheckExpandedSurface is CheckSurface over the message source stands for once expand writes out its
// repeated message fields' elements, without building that message. Callers check source's own
// surface first.
func CheckExpandedSurface(source proto.Message, limits Limits, expand Expansion) error {
	if err := limits.validate(); err != nil {
		return err
	}
	if IsNil(source) || expand == nil {
		return Invalid(Malformed, "$", "message and expansion are required")
	}
	b := budget{limits: limits, expand: expand}
	return inspectSurface(source.ProtoReflect(), &b, "$")
}

func intrinsicEnums() []protoreflect.EnumDescriptor {
	return []protoreflect.EnumDescriptor{testpilotspb.InstructionOutcomeStatus(0).Descriptor(), testpilotspb.RunEventKind(0).Descriptor(), testpilotspb.FaultKind(0).Descriptor()}
}

// MaxRunEventKind is the highest admitted Run Event kind. Recording, Monitor observation,
// Contract transition admission and capture iteration all bound themselves by it, so a new kind
// is admitted in one place rather than four.
const MaxRunEventKind = testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED

// RunEventPayload is the payload arm Run Events of one kind may carry.
type RunEventPayload struct {
	// Arm is the member of the RunEvent payload oneof, empty when the kind carries no payload.
	Arm protoreflect.Name
	// Required is true when every event of the kind carries the arm.
	Required bool
}

// RunEventPayloadOf is the one kind-to-payload table. Recording rejects an event that disagrees
// with it, and Contract preparation declares the arms a transition's kinds may carry from it.
// The INSTRUCTION_COMPLETED events a response read emits carry Observations and no outcome, so no
// outcome arm is required.
func RunEventPayloadOf(kind testpilotspb.RunEventKind) RunEventPayload {
	switch kind {
	case testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT, testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC:
		return RunEventPayload{Arm: "outcome"}
	case testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED:
		return RunEventPayload{Arm: "fault_injected", Required: true}
	default:
		return RunEventPayload{}
	}
}

// CheckRunEventPayload rejects an event carrying a payload arm its kind cannot carry, or lacking
// the arm its kind requires.
func CheckRunEventPayload(event *testpilotspb.RunEvent) error {
	message := event.ProtoReflect()
	carried := message.WhichOneof(message.Descriptor().Oneofs().ByName("payload"))
	expected := RunEventPayloadOf(event.GetKind())
	kind := fmt.Sprint(int32(event.GetKind()))
	if value := event.GetKind().Descriptor().Values().ByNumber(event.GetKind().Number()); value != nil {
		kind = string(value.Name())
	}
	switch {
	case carried != nil && carried.Name() != expected.Arm:
		return Invalid(Malformed, "run_event.payload", fmt.Sprintf("%s cannot carry the %s payload", kind, carried.Name()))
	case carried == nil && expected.Required:
		return Invalid(Malformed, "run_event.payload", fmt.Sprintf("%s requires the %s payload", kind, expected.Arm))
	default:
		return nil
	}
}

// RunEventPayloadType is the type of one payload arm, or false when arm is not a payload member.
func (c *Catalog) RunEventPayloadType(arm protoreflect.Name) (Type, bool) {
	field := (*testpilotspb.RunEvent)(nil).ProtoReflect().Descriptor().Oneofs().ByName("payload").Fields().ByName(arm)
	if field == nil {
		return Type{}, false
	}
	return c.fieldType(field), true
}

// RunEventPayloadValue reads the payload arm of event as an expression value, encoded like any
// message a path reads. It is nil when the event carries another arm or none.
func RunEventPayloadValue(event *testpilotspb.RunEvent, arm protoreflect.Name) *celpb.Value {
	message := event.ProtoReflect()
	carried := message.WhichOneof(message.Descriptor().Oneofs().ByName("payload"))
	if carried == nil || carried.Name() != arm {
		return nil
	}
	payload := message.Get(carried).Message().Interface()
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(payload)
	if err != nil {
		return nil
	}
	return &celpb.Value{Kind: &celpb.Value_ObjectValue{ObjectValue: &anypb.Any{TypeUrl: "type.googleapis.com/" + string(carried.Message().FullName()), Value: encoded}}}
}

func inspectMap(field protoreflect.FieldDescriptor, value protoreflect.Value, depth int64, b *budget, path string) error {
	if int64(value.Map().Len()) > b.limits.Fanout {
		return Invalid(LimitExceeded, path, "map collection ceiling exceeded")
	}

	if b.ctx == nil {
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
	items := value.Map()
	if err := b.charge(depth, int64(items.Len()), 0, path); err != nil {
		return err
	}
	keys := make([]protoreflect.MapKey, 0, items.Len())
	var orderingWork int64
	var orderingExceeded bool
	items.Range(func(key protoreflect.MapKey, _ protoreflect.Value) bool {
		keys = append(keys, key)
		keyBytes := int64(32)
		if field.MapKey().Kind() == protoreflect.StringKind {
			keyBytes = int64(len(key.String())) + 1
		}
		factor := 8 * int64(bits.Len(uint(items.Len()))+1)
		if keyBytes > (b.limits.Work-orderingWork)/factor {
			orderingExceeded = true
		} else if !orderingExceeded {
			orderingWork += keyBytes * factor
		}
		return true
	})
	if orderingExceeded {
		return Invalid(LimitExceeded, path, "runtime work ceiling exceeded")
	}
	if err := b.charge(depth, orderingWork, 0, path); err != nil {
		return err
	}
	slices.SortFunc(keys, func(a, b protoreflect.MapKey) int { return compareMapKeys(field.MapKey().Kind(), a, b) })
	for _, key := range keys {
		if err := inspectValue(field.MapKey(), key.Value(), depth, b, path); err != nil {
			return err
		}
		if err := inspectValue(field.MapValue(), items.Get(key), depth, b, path); err != nil {
			return err
		}
	}
	return nil
}

func compareMapKeys(kind protoreflect.Kind, a, b protoreflect.MapKey) int {
	switch kind {
	case protoreflect.BoolKind:
		if a.Bool() == b.Bool() {
			return 0
		}
		if a.Bool() {
			return 1
		}
		return -1
	case protoreflect.StringKind:
		return strings.Compare(a.String(), b.String())
	case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Sint32Kind, protoreflect.Sint64Kind, protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind:
		return cmp.Compare(a.Int(), b.Int())
	default:
		return cmp.Compare(a.Uint(), b.Uint())
	}
}
