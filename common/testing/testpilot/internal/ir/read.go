package ir

import (
	"context"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

// Read retains absent wildcard branches: any absent element makes the value absent, while
// an explicit presence selector returns one aligned boolean for every element.
func (p *Path) Read(ctx context.Context, source proto.Message, limits Limits) (*testpilotspb.Value, int64, error) {
	b, err := runtimeBudget(ctx, limits)
	if err != nil {
		return nil, 0, err
	}
	if p == nil || p.source.message == nil {
		return nil, 0, invalid(TypeMismatch, "path", "message source required")
	}
	snapshot, err := snapshotMessage(source, p.source.message, limits.Bytes, b)
	if err != nil {
		return nil, b.work, err
	}
	r := runtimeExpression{ctx: ctx, limit: limits.Work - b.work, copyWork: true}
	var values []*testpilotspb.Value
	if len(p.steps) == 0 {
		if err = r.charge(2*int64(proto.Size(snapshot)) + 1); err == nil {
			var wire []byte
			wire, err = proto.MarshalOptions{Deterministic: true}.Marshal(snapshot)
			values = []*testpilotspb.Value{{Value: &testpilotspb.Value_MessageValue{MessageValue: &anypb.Any{TypeUrl: "type.googleapis.com/" + string(p.source.message.FullName()), Value: wire}}}}
		}
	} else {
		values, err = r.readBranches(snapshot.ProtoReflect(), p, 0, make([]int64, len(p.steps)), min(p.limit, limits.Fanout))
	}
	work := b.work + r.work
	if err != nil {
		return nil, work, err
	}
	for _, v := range values {
		if v == nil {
			return nil, work, nil
		}
	}
	var result *testpilotspb.Value
	if p.fanout {
		result = &testpilotspb.Value{Value: &testpilotspb.Value_ListValue{ListValue: &testpilotspb.ValueList{Values: values}}}
	} else if len(values) > 0 {
		result = values[0]
	}
	if err = r.charge(int64(proto.Size(result))); err != nil {
		return nil, b.work + r.work, err
	}
	return result, b.work + r.work, nil
}
func (r *runtimeExpression) readBranches(message protoreflect.Message, p *Path, index int, counts []int64, fanout int64) ([]*testpilotspb.Value, error) {
	if err := r.charge(1); err != nil {
		return nil, err
	}
	step := p.steps[index]
	if index == len(p.steps)-1 {
		if message == nil {
			if step.Selector == Presence {
				return []*testpilotspb.Value{boolValue(false)}, nil
			}
			return []*testpilotspb.Value{nil}, nil
		}
		remaining := fanout - counts[index]
		values, err := r.selectField(message, step, remaining)
		if values == nil && err == nil {
			values = []*testpilotspb.Value{nil}
		}
		if int64(len(values)) > remaining {
			return nil, invalid(LimitExceeded, "path", "fan-out ceiling exceeded")
		}
		counts[index] += int64(len(values))
		return values, err
	}
	children, err := r.messageChildren(message, step, fanout-counts[index])
	if err != nil {
		return nil, err
	}
	if int64(len(children)) > fanout-counts[index] {
		return nil, invalid(LimitExceeded, "path", "fan-out ceiling exceeded")
	}
	counts[index] += int64(len(children))
	var result []*testpilotspb.Value
	for _, child := range children {
		values, err := r.readBranches(child, p, index+1, counts, fanout)
		if err != nil {
			return nil, err
		}
		result = append(result, values...)
	}
	return result, nil
}

func (r *runtimeExpression) messageChildren(message protoreflect.Message, step PathStep, remaining int64) ([]protoreflect.Message, error) {
	var children []protoreflect.Message
	if message == nil || step.Field.HasPresence() && !message.Has(step.Field) {
		children = []protoreflect.Message{nil}
	} else {
		value := message.Get(step.Field)
		switch step.Selector {
		case Wildcard:
			list := value.List()
			if int64(list.Len()) > remaining {
				return nil, invalid(LimitExceeded, "path", "fan-out ceiling exceeded")
			}
			if err := r.charge(int64(list.Len())); err != nil {
				return nil, err
			}
			children = make([]protoreflect.Message, list.Len())
			for i := range children {
				children[i] = list.Get(i).Message()
			}
		case MapKey:
			key, err := mapKey(step.Key, step.Field.MapKey())
			if err != nil {
				return nil, err
			}
			children = []protoreflect.Message{nil}
			if value.Map().Has(key) {
				children[0] = value.Map().Get(key).Message()
			}
		case Field, Oneof:
			children = []protoreflect.Message{value.Message()}
		case Presence:
			return nil, invalid(Unsupported, "path", "nonterminal presence")
		default:
			return nil, invalid(Unsupported, "path", "unknown selector")
		}
	}
	return children, nil
}

// ReadValue reads one bound Path out of an already-projected message Value, with the same runtime
// path semantics a Contract expression reads an Observation with. A nil result denotes absence.
func ReadValue(ctx context.Context, value *testpilotspb.Value, typ Type, path *Path, limits Limits) (*testpilotspb.Value, int64, error) {
	check := limits
	check.Work = DefaultLimits().Work
	if err := check.validate(); err != nil {
		return nil, 0, err
	}
	if ctx == nil || value == nil || path == nil || typ.Message() == nil || limits.Work <= 0 {
		return nil, 0, invalid(Malformed, "path", "context, message value, path and positive work required")
	}
	r := runtimeExpression{ctx: ctx, limit: limits.Work, copyWork: true}
	result, err := r.project(path, value, typ)
	if err != nil || result == nil {
		return nil, r.work, err
	}
	return proto.CloneOf(result), r.work, nil
}

// SameMessage reports exact structural identity of two message descriptors, including every nested
// message they reach. Two catalogs may name the same message differently, so a declared type is
// admitted against a compiled-in one only through this check.
func SameMessage(a, b protoreflect.MessageDescriptor) bool {
	return sameMessage(a, b, map[protoreflect.FullName]bool{})
}
func sameMessage(a, b protoreflect.MessageDescriptor, seen map[protoreflect.FullName]bool) bool {
	if a == nil || b == nil || a.FullName() != b.FullName() || !proto.Equal(protodesc.ToDescriptorProto(a), protodesc.ToDescriptorProto(b)) {
		return false
	}
	if seen[a.FullName()] {
		return true
	}
	seen[a.FullName()] = true
	for i := 0; i < a.Fields().Len(); i++ {
		x, y := a.Fields().Get(i), b.Fields().Get(i)
		if x.Message() != nil && !sameMessage(x.Message(), y.Message(), seen) {
			return false
		}
	}
	return true
}
