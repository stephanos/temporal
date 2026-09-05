package ir

import (
	"context"
	"math"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func runtimeBudget(ctx context.Context, limits Limits) (*budget, error) {
	check := limits
	check.Work = DefaultLimits().Work
	if err := check.validate(); err != nil {
		return nil, err
	}
	if ctx == nil || limits.Work <= 0 {
		return nil, invalid(Malformed, "value", "context and positive runtime work required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	limits.Bytes = math.MaxInt64
	return &budget{ctx: ctx, limits: limits}, nil
}

// SnapshotValue validates before copying; work includes traversal, payload decoding and ownership.
func SnapshotValue(ctx context.Context, value *umpirespb.Value, typ Type, limits Limits) (*umpirespb.Value, int64, error) {
	b, err := runtimeBudget(ctx, limits)
	if err != nil {
		return nil, 0, err
	}
	if err = validateValue(value, typ, b); err == nil {
		if int64(proto.Size(value)) > limits.Bytes {
			err = invalid(LimitExceeded, "value", "encoded value exceeds byte ceiling")
		} else {
			err = b.charge(1, int64(proto.Size(value)), 0, "value")
		}
	}
	if err != nil {
		return nil, b.work, err
	}
	return proto.CloneOf(value), b.work, nil
}
func validateValue(value *umpirespb.Value, typ Type, b *budget) error {
	if value == nil || typ.catalog == nil || typ.opaque {
		return invalid(TypeMismatch, "value", "ordinary typed value required")
	}
	if err := inspectSurface(value.ProtoReflect(), b, "value"); err != nil {
		return err
	}
	return typ.catalog.checkLiteral(value, typ, b, 1)
}

// SnapshotMessage decodes into the pinned descriptor, including when the Host uses generated types.
func SnapshotMessage(ctx context.Context, source proto.Message, descriptor protoreflect.MessageDescriptor, limits Limits) (proto.Message, int64, error) {
	b, err := runtimeBudget(ctx, limits)
	if err != nil {
		return nil, 0, err
	}
	result, err := snapshotMessage(source, descriptor, limits.Bytes, b)
	return result, b.work, err
}
func snapshotMessage(source proto.Message, descriptor protoreflect.MessageDescriptor, bytes int64, b *budget) (proto.Message, error) {
	if missing(source) || descriptor == nil || source.ProtoReflect().Descriptor().FullName() != descriptor.FullName() {
		return nil, invalid(TypeMismatch, "message", "wrong protobuf response type")
	}
	if err := compatibleMessage(source.ProtoReflect().Descriptor(), descriptor, b); err != nil {
		return nil, err
	}
	if err := inspect(source.ProtoReflect(), 1, b, "message"); err != nil {
		return nil, err
	}
	size := int64(proto.Size(source))
	if size > bytes {
		return nil, invalid(LimitExceeded, "message", "encoded message exceeds byte ceiling")
	}
	if err := b.charge(1, 2*size, 0, "message"); err != nil {
		return nil, err
	}
	wire, err := proto.MarshalOptions{Deterministic: true}.Marshal(source)
	if err != nil {
		return nil, err
	}
	if err = scanMessage(wire, descriptor, b, 1); err != nil {
		return nil, err
	}
	result := dynamicpb.NewMessage(descriptor)
	if err = (proto.UnmarshalOptions{RecursionLimit: int(b.limits.Depth)}).Unmarshal(wire, result); err != nil {
		return nil, err
	}
	if err = inspect(result.ProtoReflect(), 1, b, "message"); err != nil {
		return nil, err
	}
	return result, nil
}
