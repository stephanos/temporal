package activityrelations

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// secretPlaceholder replaces every present Secret value before storage. Secret claims project to
// presence only, so a placeholder of the same presence keeps every novelty key unchanged.
const secretPlaceholder = "secret"

type secretField struct {
	message protoreflect.FullName
	number  protoreflect.FieldNumber
}

// secrets is every request- and response-side Secret claim of every bound plan; a claim names a
// message type, so it applies wherever that type appears.
func secrets() map[secretField]bool {
	set := map[secretField]bool{}
	for _, plan := range Plans() {
		for _, field := range slices.Concat(plan.Secret, plan.RequestSecret) {
			set[secretField{protoreflect.FullName(field.Message), protoreflect.FieldNumber(field.Number)}] = true
		}
	}
	return set
}

// rawDigest keys raw-distinct counting. Its HMAC key lives only in one Collector and is never
// stored, so a stored digest cannot confirm a guessed Secret value.
type rawDigest struct{ key []byte }

func newRawDigest() (rawDigest, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return rawDigest{}, fmt.Errorf("ephemeral digest key: %w", err)
	}
	return rawDigest{key: key}, nil
}

func (d rawDigest) of(data []byte) string {
	mac := hmac.New(sha256.New, d.key)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// sanitize re-encodes a message with every Secret value replaced by a placeholder of the same
// presence.
func sanitize(message proto.Message, claims map[secretField]bool) ([]byte, error) {
	clone := proto.Clone(message)
	redact(clone.ProtoReflect(), claims)
	return proto.MarshalOptions{Deterministic: true}.Marshal(clone)
}

func redact(message protoreflect.Message, claims map[secretField]bool) {
	name := message.Descriptor().FullName()
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if claims[secretField{name, field.Number()}] {
			message.Set(field, placeholder(message, field))
			return true
		}
		switch {
		case field.IsMap():
			entry := field.MapValue()
			valueSecret := claims[secretField{field.Message().FullName(), 2}]
			value.Map().Range(func(key protoreflect.MapKey, item protoreflect.Value) bool {
				switch {
				case valueSecret:
					value.Map().Set(key, scalarPlaceholder(entry, item))
				case entry.Message() != nil:
					redact(item.Message(), claims)
				}
				return true
			})
		case field.IsList() && field.Message() != nil:
			for i := range value.List().Len() {
				redact(value.List().Get(i).Message(), claims)
			}
		case field.Message() != nil:
			redact(value.Message(), claims)
		}
		return true
	})
}

// placeholder keeps presence: one element for a nonempty list or map, an empty message for a
// present message, and the placeholder for a present scalar.
func placeholder(message protoreflect.Message, field protoreflect.FieldDescriptor) protoreflect.Value {
	value := message.NewField(field)
	switch {
	case field.IsList():
		element := value.List().NewElement()
		if field.Message() == nil {
			element = scalarPlaceholder(field, element)
		}
		value.List().Append(element)
	case field.IsMap():
		item := value.Map().NewValue()
		if field.MapValue().Message() == nil {
			item = scalarPlaceholder(field.MapValue(), item)
		}
		value.Map().Set(field.MapKey().Default().MapKey(), item)
	case field.Message() == nil:
		value = scalarPlaceholder(field, value)
	}
	return value
}

func scalarPlaceholder(field protoreflect.FieldDescriptor, zero protoreflect.Value) protoreflect.Value {
	switch field.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(secretPlaceholder)
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte(secretPlaceholder))
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(true)
	case protoreflect.EnumKind:
		return protoreflect.ValueOfEnum(1)
	case protoreflect.FloatKind:
		return protoreflect.ValueOfFloat32(1)
	case protoreflect.DoubleKind:
		return protoreflect.ValueOfFloat64(1)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(1)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(1)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(1)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(1)
	default:
		return zero
	}
}
