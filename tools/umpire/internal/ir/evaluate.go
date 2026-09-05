package ir

import (
	"cmp"
	"context"
	"math"
	"strconv"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// Evaluate reads already type-checked immutable values. A nil resolved value denotes absence.
// The returned value is independent of the resolver and prepared expression.
func (e *Expression) Evaluate(ctx context.Context, resolve func(Reference) *umpirespb.Value, limit int64) (*umpirespb.Value, int64, error) {
	return e.evaluate(ctx, resolve, limit, false)
}

// EvaluateExecution shares Evaluate's semantics and charges intermediate ownership copies.
// Evaluate retains the accounting units used by already-admitted Contract work bounds.
func (e *Expression) EvaluateExecution(ctx context.Context, resolve func(Reference) *umpirespb.Value, limit int64) (*umpirespb.Value, int64, error) {
	return e.evaluate(ctx, resolve, limit, true)
}
func (e *Expression) evaluate(ctx context.Context, resolve func(Reference) *umpirespb.Value, limit int64, copies bool) (*umpirespb.Value, int64, error) {
	r := runtimeExpression{ctx: ctx, resolve: resolve, limit: limit, copyWork: copies}
	if ctx == nil || resolve == nil || e == nil || limit <= 0 {
		return nil, 0, invalid(Malformed, "expression", "context, expression, resolver and positive work required")
	}
	v, err := r.eval(e)
	if err == nil && v == nil {
		err = invalid(Unavailable, "expression", "unguarded absent value")
	}
	if err == nil {
		if _, scalar := v.GetValue().(*umpirespb.Value_BoolValue); !scalar || copies {
			err = r.charge(int64(proto.Size(v)))
		}
	}
	if err != nil {
		return nil, r.work, err
	}
	return proto.CloneOf(v), r.work, nil
}

type runtimeExpression struct {
	copyWork    bool
	ctx         context.Context
	resolve     func(Reference) *umpirespb.Value
	limit, work int64
}

func (r *runtimeExpression) charge(n int64) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if n < 0 || n > r.limit-r.work {
		return invalid(LimitExceeded, "expression", "runtime work ceiling exceeded")
	}
	r.work += n
	return nil
}
func boolValue(v bool) *umpirespb.Value {
	return &umpirespb.Value{Value: &umpirespb.Value_BoolValue{BoolValue: v}}
}
func (r *runtimeExpression) eval(e *Expression) (*umpirespb.Value, error) {
	if err := r.charge(1); err != nil {
		return nil, err
	}
	switch e.operator {
	case Literal:
		if err := r.charge(int64(proto.Size(e.literal))); err != nil {
			return nil, err
		}
		return e.literal, nil
	case ReferenceValue:
		return r.resolve(e.reference), nil
	case Project:
		v, err := r.eval(e.children[0])
		if err != nil {
			return nil, err
		}
		return r.project(e.path, v, e.children[0].typ)
	case IsPresent:
		v, err := r.eval(e.children[0])
		return boolValue(v != nil), err
	case Not:
		v, err := r.eval(e.children[0])
		if err != nil {
			return nil, err
		}
		if v == nil {
			return nil, invalid(Unavailable, "expression", "absent boolean")
		}
		return boolValue(!v.GetBoolValue()), nil
	case All, Any:
		continuing := e.operator == All
		for _, child := range e.children {
			v, err := r.eval(child)
			if err != nil {
				return nil, err
			}
			if v == nil {
				return nil, invalid(Unavailable, "expression", "absent boolean")
			}
			if v.GetBoolValue() != continuing {
				return boolValue(!continuing), nil
			}
		}
		return boolValue(continuing), nil
	case Equals, Compare:
		return r.binary(e)
	default:
		return nil, invalid(Unsupported, "expression", "unknown prepared operator")
	}
}
func (r *runtimeExpression) binary(e *Expression) (*umpirespb.Value, error) {
	a, err := r.eval(e.children[0])
	if err != nil {
		return nil, err
	}
	b, err := r.eval(e.children[1])
	if err != nil {
		return nil, err
	}
	if a == nil || b == nil {
		return nil, invalid(Unavailable, "expression", "absent comparison operand")
	}
	if err := r.charge(int64(proto.Size(a)) + int64(proto.Size(b))); err != nil {
		return nil, err
	}
	if e.operator == Equals {
		same, err := r.equal(a, b, e.children[0].typ)
		return boolValue(same), err
	}
	ordering, unordered, err := compareValues(a, b, e.children[0].typ)
	if err != nil {
		return nil, err
	}
	if unordered {
		return boolValue(false), nil
	}
	switch e.comparison {
	case umpirespb.COMPARISON_OPERATOR_LESS_THAN:
		return boolValue(ordering < 0), nil
	case umpirespb.COMPARISON_OPERATOR_LESS_THAN_OR_EQUAL:
		return boolValue(ordering <= 0), nil
	case umpirespb.COMPARISON_OPERATOR_GREATER_THAN:
		return boolValue(ordering > 0), nil
	case umpirespb.COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL:
		return boolValue(ordering >= 0), nil
	default:
		return nil, invalid(Unsupported, "expression", "unknown comparison")
	}
}
func compareValues(a, b *umpirespb.Value, typ Type) (int, bool, error) {
	switch v := a.Value.(type) {
	case *umpirespb.Value_Natural:
		other := b.GetNatural()
		if len(v.Natural) != len(other) {
			return cmp.Compare(len(v.Natural), len(other)), false, nil
		}
		return cmp.Compare(v.Natural, other), false, nil
	case *umpirespb.Value_SignedInteger:
		x, err := strconv.ParseInt(v.SignedInteger, 10, 64)
		if err != nil {
			return 0, false, err
		}
		y, err := strconv.ParseInt(b.GetSignedInteger(), 10, 64)
		return cmp.Compare(x, y), false, err
	case *umpirespb.Value_UnsignedInteger:
		x, err := strconv.ParseUint(v.UnsignedInteger, 10, 64)
		if err != nil {
			return 0, false, err
		}
		y, err := strconv.ParseUint(b.GetUnsignedInteger(), 10, 64)
		return cmp.Compare(x, y), false, err
	case *umpirespb.Value_FloatingPoint:
		x, y := v.FloatingPoint, b.GetFloatingPoint()
		if typ.scalar == umpirespb.SCALAR_KIND_FLOAT {
			x, y = float64(float32(x)), float64(float32(y))
		}
		return cmp.Compare(x, y), math.IsNaN(x) || math.IsNaN(y), nil
	default:
		return 0, false, invalid(TypeMismatch, "expression", "ordered scalar required")
	}
}
func (r *runtimeExpression) equal(a, b *umpirespb.Value, typ Type) (bool, error) {
	if err := r.ctx.Err(); err != nil {
		return false, err
	}
	if typ.cardinality == Repeated {
		x, y := a.GetListValue().GetValues(), b.GetListValue().GetValues()
		if len(x) != len(y) {
			return false, nil
		}
		for i := range x {
			same, err := r.equal(x[i], y[i], typ.Element())
			if err != nil || !same {
				return same, err
			}
		}
		return true, nil
	}
	if typ.cardinality == Map {
		return r.equalMap(a, b, typ)
	}
	if typ.message != nil {
		if r.copyWork {
			if err := r.charge(2 * (int64(proto.Size(a)) + int64(proto.Size(b)))); err != nil {
				return false, err
			}
		}
		x, err := decodeMessage(a, typ.message)
		if err != nil {
			return false, err
		}
		y, err := decodeMessage(b, typ.message)
		if err != nil {
			return false, err
		}
		return proto.Equal(x.Interface(), y.Interface()), nil
	}
	if typ.scalar == umpirespb.SCALAR_KIND_FLOAT {
		x, y := float32(a.GetFloatingPoint()), float32(b.GetFloatingPoint())
		return x == y || math.IsNaN(float64(x)) && math.IsNaN(float64(y)), nil
	}
	return proto.Equal(a, b), nil
}

func (r *runtimeExpression) equalMap(a, b *umpirespb.Value, typ Type) (bool, error) {
	x, y := a.GetMapValue().GetEntries(), b.GetMapValue().GetEntries()
	if len(x) != len(y) {
		return false, nil
	}
	indexed := make(map[string]*umpirespb.Value, len(y))
	for _, entry := range y {
		if err := r.ctx.Err(); err != nil {
			return false, err
		}
		if r.copyWork {
			if err := r.charge(8*int64(proto.Size(entry.Key)) + 1); err != nil {
				return false, err
			}
		}
		indexed[entry.Key.String()] = entry.Value
	}
	for _, entry := range x {
		if r.copyWork {
			if err := r.charge(8*int64(proto.Size(entry.Key)) + 1); err != nil {
				return false, err
			}
		}
		other := indexed[entry.Key.String()]
		if other == nil {
			return false, nil
		}
		same, err := r.equal(entry.Value, other, typ.Element())
		if err != nil || !same {
			return same, err
		}
	}
	return true, nil
}
