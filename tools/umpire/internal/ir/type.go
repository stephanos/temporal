package ir

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type Cardinality uint8

const (
	Singular Cardinality = iota + 1
	Repeated
	Map
)

type Type struct {
	schema      *umpirespb.ValueType
	catalog     *Catalog
	cardinality Cardinality
	scalar      umpirespb.ScalarKind
	message     protoreflect.MessageDescriptor
	enumeration protoreflect.EnumDescriptor
	opaque      bool
	any         bool
	key         umpirespb.ScalarKind
}

func (t Type) Schema() *umpirespb.ValueType            { return proto.CloneOf(t.schema) }
func (t Type) Cardinality() Cardinality                { return t.cardinality }
func (t Type) Scalar() umpirespb.ScalarKind            { return t.scalar }
func (t Type) Message() protoreflect.MessageDescriptor { return t.message }
func (t Type) Enum() protoreflect.EnumDescriptor       { return t.enumeration }
func (t Type) Opaque() bool                            { return t.opaque }
func (t Type) Any() bool                               { return t.any }
func (t Type) MapKey() umpirespb.ScalarKind            { return t.key }
func (t Type) Equal(other Type) bool {
	return t.schema != nil && other.schema != nil && proto.Equal(t.schema, other.schema) && t.catalog.identity == other.catalog.identity
}
func (t Type) Element() Type {
	if t.cardinality == Singular || t.schema == nil {
		return t
	}
	result := t
	var singular *umpirespb.SingularType
	if t.cardinality == Repeated {
		singular = t.schema.GetRepeated().GetElement()
	} else {
		singular = t.schema.GetMap().GetValue()
	}
	result.schema = &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: proto.CloneOf(singular)}}
	result.cardinality = Singular
	result.key = umpirespb.SCALAR_KIND_UNSPECIFIED
	return result
}

func (c *Catalog) BindType(schema *umpirespb.ValueType) (Type, error) {
	if schema == nil || missing(schema.Shape) {
		return Type{}, invalid(Malformed, "type", "type is required")
	}
	b := budget{limits: DefaultLimits()}
	if err := inspect(schema.ProtoReflect(), 1, &b, "type"); err != nil {
		return Type{}, err
	}
	result := Type{schema: proto.CloneOf(schema), catalog: c}
	var singular *umpirespb.SingularType
	switch shape := schema.Shape.(type) {
	case *umpirespb.ValueType_Singular:
		result.cardinality = Singular
		singular = shape.Singular
	case *umpirespb.ValueType_Repeated:
		result.cardinality = Repeated
		singular = shape.Repeated.GetElement()
	case *umpirespb.ValueType_Map:
		result.cardinality = Map
		singular = shape.Map.GetValue()
		result.key = shape.Map.GetKey().GetKind()
		if !mapKeyKind(result.key) {
			return Type{}, invalid(TypeMismatch, "type", "invalid protobuf map key kind")
		}
	default:
		return Type{}, invalid(Malformed, "type", "missing type shape")
	}
	if singular == nil || missing(singular.Type) {
		return Type{}, invalid(Malformed, "type", "missing singular type")
	}
	if err := c.bindSingular(singular, &result); err != nil {
		return Type{}, err
	}

	if result.opaque && result.cardinality != Singular {
		return Type{}, invalid(TypeMismatch, "type", "capabilities cannot occupy collections")
	}
	return result, nil
}

func (c *Catalog) bindSingular(singular *umpirespb.SingularType, result *Type) error {
	switch value := singular.Type.(type) {
	case *umpirespb.SingularType_Scalar:
		result.scalar = value.Scalar.GetKind()
		if result.scalar < umpirespb.SCALAR_KIND_TEXT || result.scalar > umpirespb.SCALAR_KIND_DOUBLE {
			return invalid(Unknown, "type", "unknown scalar kind")
		}
	case *umpirespb.SingularType_Enumeration:
		var descriptor protoreflect.Descriptor
		var err error
		intrinsic := umpirespb.InstructionOutcomeStatus(0).Descriptor()
		if value.Enumeration.GetProtobufType() == string(intrinsic.FullName()) {
			descriptor = intrinsic
		} else {
			descriptor, err = c.files.FindDescriptorByName(protoreflect.FullName(value.Enumeration.GetProtobufType()))
		}
		if err != nil {
			return invalid(Unknown, "type", "unknown enumeration")
		}
		var ok bool
		result.enumeration, ok = descriptor.(protoreflect.EnumDescriptor)
		if !ok {
			return invalid(TypeMismatch, "type", "expected enumeration descriptor")
		}
	case *umpirespb.SingularType_Message:
		descriptor, err := c.files.FindDescriptorByName(protoreflect.FullName(value.Message.GetProtobufType()))
		if err != nil {
			return invalid(Unknown, "type", "unknown message")
		}
		var ok bool
		result.message, ok = descriptor.(protoreflect.MessageDescriptor)
		if !ok || result.message.IsMapEntry() {
			return invalid(TypeMismatch, "type", "expected ordinary message descriptor")
		}
		if result.message.FullName() == "google.protobuf.Any" {
			return invalid(TypeMismatch, "type", "Any requires the explicit Any type")
		}
	case *umpirespb.SingularType_Any:
		result.any = true
	case *umpirespb.SingularType_OpaqueCapability:
		result.opaque = true
	default:
		return invalid(Malformed, "type", "missing singular type variant")
	}
	return nil
}

func mapKeyKind(kind umpirespb.ScalarKind) bool {
	switch kind {
	case umpirespb.SCALAR_KIND_TEXT, umpirespb.SCALAR_KIND_BOOLEAN,
		umpirespb.SCALAR_KIND_INT32, umpirespb.SCALAR_KIND_INT64, umpirespb.SCALAR_KIND_UINT32, umpirespb.SCALAR_KIND_UINT64,
		umpirespb.SCALAR_KIND_SINT32, umpirespb.SCALAR_KIND_SINT64, umpirespb.SCALAR_KIND_FIXED32, umpirespb.SCALAR_KIND_FIXED64,
		umpirespb.SCALAR_KIND_SFIXED32, umpirespb.SCALAR_KIND_SFIXED64:
		return true
	default:
		return false
	}
}

func (c *Catalog) scalarType(kind umpirespb.ScalarKind) Type {
	return Type{catalog: c, cardinality: Singular, scalar: kind, schema: &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Scalar{Scalar: &umpirespb.ScalarType{Kind: kind}}}}}}
}

func (c *Catalog) CheckLiteral(value *umpirespb.Value, typ Type, limits Limits) error {
	if err := limits.validate(); err != nil {
		return err
	}
	if value == nil || missing(value.Value) {
		return invalid(Malformed, "literal", "literal is required")
	}
	if !c.owns(typ) {
		return invalid(TypeMismatch, "literal", "type does not belong to this catalog")
	}
	b := budget{limits: limits}
	if err := inspectSurface(value.ProtoReflect(), &b, "literal"); err != nil {
		return err
	}
	return c.checkLiteral(value, typ, &b, 1)
}

func (c *Catalog) checkLiteral(value *umpirespb.Value, typ Type, b *budget, depth int64) error {
	if value == nil || missing(value.Value) || typ.schema == nil {
		return invalid(Malformed, "literal", "missing literal or type")
	}
	if err := b.charge(depth, 1, 0, "literal"); err != nil {
		return err
	}
	if typ.opaque {
		return invalid(Unsupported, "literal", "capability literals are not representable")
	}
	if typ.cardinality == Repeated {
		return c.checkList(value, typ, b, depth)
	}
	if typ.cardinality == Map {
		return c.checkMap(value, typ, b, depth)
	}
	if typ.enumeration != nil {
		return checkEnum(value, typ)
	}
	if typ.message != nil || typ.any {
		return c.checkMessage(value, typ, b, depth)
	}
	return checkScalar(value, typ.scalar)
}

func literalMismatch() error {
	return invalid(TypeMismatch, "literal", "literal does not match its declared type")
}

func (c *Catalog) checkList(value *umpirespb.Value, typ Type, b *budget, depth int64) error {
	list, ok := value.Value.(*umpirespb.Value_ListValue)
	if !ok || list.ListValue == nil {
		return literalMismatch()
	}
	if int64(len(list.ListValue.Values)) > b.limits.Fanout {
		return invalid(LimitExceeded, "literal", "collection ceiling exceeded")
	}
	for _, item := range list.ListValue.Values {
		if err := c.checkLiteral(item, typ.Element(), b, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) checkMap(value *umpirespb.Value, typ Type, b *budget, depth int64) error {
	entries, ok := value.Value.(*umpirespb.Value_MapValue)
	if !ok || entries.MapValue == nil {
		return literalMismatch()
	}
	if int64(len(entries.MapValue.Entries)) > b.limits.Fanout {
		return invalid(LimitExceeded, "literal", "collection ceiling exceeded")
	}
	seen := make(map[string]bool, len(entries.MapValue.Entries))
	for _, entry := range entries.MapValue.Entries {
		if entry == nil {
			return invalid(Malformed, "literal", "nil map entry")
		}
		if err := c.checkLiteral(entry.Key, c.scalarType(typ.key), b, depth+1); err != nil {
			return err
		}
		key := entry.Key.String()
		if seen[key] {
			return invalid(Malformed, "literal", "duplicate map key")
		}
		seen[key] = true
		if err := c.checkLiteral(entry.Value, typ.Element(), b, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func checkEnum(value *umpirespb.Value, typ Type) error {
	item, ok := value.Value.(*umpirespb.Value_EnumValue)
	if !ok || item.EnumValue == nil {
		return literalMismatch()
	}
	if typ.enumeration.Values().ByNumber(protoreflect.EnumNumber(item.EnumValue.Number)) == nil {
		return invalid(Unknown, "literal", "undefined enum number")
	}
	return nil
}

func (c *Catalog) checkMessage(value *umpirespb.Value, typ Type, b *budget, depth int64) error {
	item, ok := value.Value.(*umpirespb.Value_MessageValue)
	if !ok || item.MessageValue == nil {
		return literalMismatch()
	}
	envelope := item.MessageValue
	slash := strings.LastIndexByte(envelope.TypeUrl, '/')
	if slash < 0 || !protoreflect.FullName(envelope.TypeUrl[slash+1:]).IsValid() {
		return invalid(Malformed, "literal", "invalid message type URL")
	}
	if typ.any {
		return nil
	}
	if envelope.TypeUrl[slash+1:] != string(typ.message.FullName()) {
		return literalMismatch()
	}
	if err := scanMessage(envelope.Value, typ.message, b, depth+1); err != nil {
		return err
	}
	message := dynamicpb.NewMessage(typ.message)
	if err := (proto.UnmarshalOptions{RecursionLimit: int(b.limits.Depth)}).Unmarshal(envelope.Value, message); err != nil {
		return invalid(Malformed, "literal", "invalid message wire payload")
	}
	return inspect(message.ProtoReflect(), depth+1, b, "literal.message")
}

func checkScalar(value *umpirespb.Value, kind umpirespb.ScalarKind) error {
	switch kind {
	case umpirespb.SCALAR_KIND_TEXT:
		item, ok := value.Value.(*umpirespb.Value_Text)
		if !ok || !utf8.ValidString(item.Text) {
			return literalMismatch()
		}
	case umpirespb.SCALAR_KIND_BYTES:
		if _, ok := value.Value.(*umpirespb.Value_BytesValue); !ok {
			return literalMismatch()
		}
	case umpirespb.SCALAR_KIND_BOOLEAN:
		if _, ok := value.Value.(*umpirespb.Value_BoolValue); !ok {
			return literalMismatch()
		}
	case umpirespb.SCALAR_KIND_NATURAL:
		item, ok := value.Value.(*umpirespb.Value_Natural)
		if !ok || !canonicalUnsigned(item.Natural) {
			return literalMismatch()
		}
	case umpirespb.SCALAR_KIND_INT32, umpirespb.SCALAR_KIND_INT64, umpirespb.SCALAR_KIND_SINT32, umpirespb.SCALAR_KIND_SINT64, umpirespb.SCALAR_KIND_SFIXED32, umpirespb.SCALAR_KIND_SFIXED64, umpirespb.SCALAR_KIND_UINT32, umpirespb.SCALAR_KIND_UINT64, umpirespb.SCALAR_KIND_FIXED32, umpirespb.SCALAR_KIND_FIXED64:
		return checkInteger(value, kind)
	case umpirespb.SCALAR_KIND_FLOAT, umpirespb.SCALAR_KIND_DOUBLE:
		item, ok := value.Value.(*umpirespb.Value_FloatingPoint)
		if !ok {
			return literalMismatch()
		}
		if kind == umpirespb.SCALAR_KIND_FLOAT && !math.IsInf(item.FloatingPoint, 0) && math.Abs(item.FloatingPoint) > math.MaxFloat32 {
			return literalMismatch()
		}
	default:
		return literalMismatch()
	}
	return nil
}

func checkInteger(value *umpirespb.Value, kind umpirespb.ScalarKind) error {
	switch kind {
	case umpirespb.SCALAR_KIND_INT32, umpirespb.SCALAR_KIND_INT64, umpirespb.SCALAR_KIND_SINT32, umpirespb.SCALAR_KIND_SINT64, umpirespb.SCALAR_KIND_SFIXED32, umpirespb.SCALAR_KIND_SFIXED64:
		item, ok := value.Value.(*umpirespb.Value_SignedInteger)
		if !ok {
			return literalMismatch()
		}
		bits := 64
		if kind == umpirespb.SCALAR_KIND_INT32 || kind == umpirespb.SCALAR_KIND_SINT32 || kind == umpirespb.SCALAR_KIND_SFIXED32 {
			bits = 32
		}
		parsed, err := strconv.ParseInt(item.SignedInteger, 10, bits)
		if err != nil || strconv.FormatInt(parsed, 10) != item.SignedInteger {
			return literalMismatch()
		}
	case umpirespb.SCALAR_KIND_UINT32, umpirespb.SCALAR_KIND_UINT64, umpirespb.SCALAR_KIND_FIXED32, umpirespb.SCALAR_KIND_FIXED64:
		item, ok := value.Value.(*umpirespb.Value_UnsignedInteger)
		if !ok {
			return literalMismatch()
		}
		bits := 64
		if kind == umpirespb.SCALAR_KIND_UINT32 || kind == umpirespb.SCALAR_KIND_FIXED32 {
			bits = 32
		}
		parsed, err := strconv.ParseUint(item.UnsignedInteger, 10, bits)
		if err != nil || strconv.FormatUint(parsed, 10) != item.UnsignedInteger {
			return literalMismatch()
		}
	default:
		return literalMismatch()
	}
	return nil
}

func canonicalUnsigned(value string) bool {
	if value == "" || (len(value) > 1 && value[0] == '0') {
		return false
	}
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

// Scan before unmarshaling so tiny repeated messages cannot allocate beyond the work ceiling.
func scanMessage(data []byte, descriptor protoreflect.MessageDescriptor, b *budget, depth int64) error {
	_, err := scanFields(data, descriptor, b, depth, 0)
	return err
}

func scanFields(data []byte, descriptor protoreflect.MessageDescriptor, b *budget, depth int64, end protowire.Number) ([]byte, error) {
	if err := b.charge(depth, 0, 0, "literal.message"); err != nil {
		return nil, err
	}
	counts := map[protoreflect.FieldNumber]int64{}
	for len(data) > 0 {
		if err := b.charge(depth, 1, 0, "literal.message"); err != nil {
			return nil, err
		}
		number, wireType, tagBytes := protowire.ConsumeTag(data)
		if tagBytes < 0 {
			return nil, invalid(Malformed, "literal.message", "invalid wire tag")
		}
		data = data[tagBytes:]
		if wireType == protowire.EndGroupType {
			if end == 0 || number != end {
				return nil, invalid(Malformed, "literal.message", "mismatched group terminator")
			}
			return data, nil
		}
		field := descriptor.Fields().ByNumber(number)
		if field == nil {
			return nil, invalid(Unknown, "literal.message", "unknown message field")
		}
		consumed, count, err := scanField(data, field, wireType, b, depth)
		if err != nil {
			return nil, err
		}
		if field.IsList() || field.IsMap() {
			if count > b.limits.Fanout-counts[number] {
				return nil, invalid(LimitExceeded, "literal.message", "collection ceiling exceeded")
			}
			counts[number] += count
		}
		data = data[consumed:]
	}
	if end != 0 {
		return nil, invalid(Malformed, "literal.message", "missing group terminator")
	}
	return nil, nil
}

func scanField(data []byte, field protoreflect.FieldDescriptor, wireType protowire.Type, b *budget, depth int64) (int, int64, error) {
	if wireType == protowire.StartGroupType {
		if field.Kind() != protoreflect.GroupKind {
			return 0, 0, invalid(TypeMismatch, "literal.message", "group wire type requires a group descriptor")
		}
		remaining, err := scanFields(data, field.Message(), b, depth+1, field.Number())
		return len(data) - len(remaining), 1, err
	}
	consumed := protowire.ConsumeFieldValue(field.Number(), wireType, data)
	if consumed < 0 {
		return 0, 0, invalid(Malformed, "literal.message", "invalid wire field")
	}
	if wireType != protowire.BytesType {
		return consumed, 1, nil
	}
	payload, n := protowire.ConsumeBytes(data)
	if n < 0 {
		return 0, 0, invalid(Malformed, "literal.message", "invalid length-delimited field")
	}
	if field.Message() != nil {
		return consumed, 1, scanMessage(payload, field.Message(), b, depth+1)
	}
	if field.IsList() && field.Kind() != protoreflect.StringKind && field.Kind() != protoreflect.BytesKind {
		count, err := scanPacked(payload, field.Kind(), b, depth)
		return consumed, count, err
	}
	return consumed, 1, nil
}

func scanPacked(data []byte, kind protoreflect.Kind, b *budget, depth int64) (int64, error) {
	var count int64
	for len(data) > 0 {
		if err := b.charge(depth, 1, 0, "literal.message"); err != nil {
			return 0, err
		}
		var consumed int
		switch kind {
		case protoreflect.Fixed32Kind, protoreflect.Sfixed32Kind, protoreflect.FloatKind:
			_, consumed = protowire.ConsumeFixed32(data)
		case protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind, protoreflect.DoubleKind:
			_, consumed = protowire.ConsumeFixed64(data)
		default:
			_, consumed = protowire.ConsumeVarint(data)
		}
		if consumed < 0 {
			return 0, invalid(Malformed, "literal.message", "invalid packed value")
		}
		count++
		if count > b.limits.Fanout {
			return 0, invalid(LimitExceeded, "literal.message", "packed collection ceiling exceeded")
		}
		data = data[consumed:]
	}
	return count, nil
}
