package ir

import (
	"math"
	"math/bits"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// SnapshotValueWorkBound reserves native input work for an admitted value. storageBytes
// must bound the complete CEL Value envelope, including its encoded message payload and
// type URL; fanout is the admitted collection ceiling. Ordinary singular values retain
// their existing four-pass reservation. Messages additionally reserve packed-scalar
// traversal and deterministic protobuf field/map ordering, without raising runtime limits.
func SnapshotValueWorkBound(typ Type, storageBytes, fanout int64) (int64, error) {
	if typ.catalog == nil || storageBytes < 0 || fanout <= 0 || fanout > DefaultLimits().Fanout {
		return 0, Invalid(Malformed, "expression.work", "admitted type, nonnegative storage and admitted fan-out required")
	}
	factor := int64(4)
	if typ.message != nil || typ.cardinality != Singular {
		// A packed one-byte scalar costs eight scalar bytes plus its visit and wire
		// scan. Sixteen passes cover those costs, envelope traversal and ownership.
		factor = 16
	}
	if typ.message != nil {
		fields, mapFactor := messageOrderingFactors(typ.message, fanout)
		// Every populated field and map entry consumes at least one payload wire
		// byte, across the whole nested message graph. Field ordering therefore
		// costs at most storageBytes*(bits.Len(maxFields)+1). Map ordering uses
		// eight times (key bytes+1) for strings or eight times 32 for numeric keys;
		// string bytes and entry counts are each bounded by payload bytes.
		factor += int64(bits.Len(uint(fields))+1) + mapFactor
	}
	if typ.cardinality == Map {
		// Native CEL map output ordering compares one key per entry. Its entry
		// count is bounded by encoded storage and by the admitted fan-out.
		factor += int64(bits.Len64(uint64(fanout)) + 1)
	}
	if storageBytes > math.MaxInt64/factor {
		return 0, Invalid(LimitExceeded, "expression.work", "native value work reservation overflow")
	}
	return storageBytes * factor, nil
}

func messageOrderingFactors(root protoreflect.MessageDescriptor, fanout int64) (maxFields int, mapFactor int64) {
	pending := []protoreflect.MessageDescriptor{root}
	seen := make(map[protoreflect.FullName]bool)
	for len(pending) > 0 {
		message := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[message.FullName()] {
			continue
		}
		seen[message.FullName()] = true
		maxFields = max(maxFields, message.Fields().Len())
		for i := 0; i < message.Fields().Len(); i++ {
			field := message.Fields().Get(i)
			if field.IsMap() {
				keyFactor := int64(256)
				if field.MapKey().Kind() == protoreflect.StringKind {
					keyFactor = 16
				}
				mapFactor = max(mapFactor, keyFactor*int64(bits.Len64(uint64(fanout))+1))
			}
			if nested := field.Message(); nested != nil {
				pending = append(pending, nested)
			}
		}
	}
	return maxFields, mapFactor
}
