package ir

import (
	"context"
	"fmt"
	"math/bits"
	"strings"

	engine "cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"cel.dev/cel-go/common/types/traits"
	"cel.dev/cel-go/interpreter"
	celpb "cel.dev/expr"
	"google.golang.org/protobuf/proto"
)

func (e *Expression) Evaluate(ctx context.Context, resolve func(Reference) *celpb.Value, limit int64) (*celpb.Value, int64, error) {
	return e.evaluate(ctx, resolve, limit, false)
}
func (e *Expression) EvaluateExecution(ctx context.Context, resolve func(Reference) *celpb.Value, limit int64) (*celpb.Value, int64, error) {
	return e.evaluate(ctx, resolve, limit, true)
}
func (e *Expression) evaluate(ctx context.Context, resolve func(Reference) *celpb.Value, limit int64, copies bool) (*celpb.Value, int64, error) {
	if ctx == nil || resolve == nil || e == nil || e.checked == nil || limit <= 0 {
		return nil, 0, Invalid(Malformed, "expression", "context, prepared expression, resolver and positive work required")
	}
	r := &runtimeExpression{ctx: ctx, resolve: resolve, limit: limit, copyWork: copies}
	if err := r.charge(1); err != nil {
		return nil, r.work, err
	}
	program, err := e.environment.Program(e.checked, engine.CostLimit(uint64(limit-r.work)), engine.CostTracking(nil), engine.InterruptCheckFrequency(1))
	if err != nil {
		return nil, r.work, err
	}
	activation := &expressionActivation{expression: e, runtime: r, values: map[string]ref.Val{}}
	value, details, evalErr := program.ContextEval(ctx, activation)
	if details != nil && details.ActualCost() != nil {
		if err = r.charge(int64(*details.ActualCost())); err != nil {
			return nil, r.work, err
		}
	}
	if activation.failure != nil {
		return nil, r.work, activation.failure
	}
	if ctx.Err() != nil {
		return nil, r.work, ctx.Err()
	}
	if evalErr != nil {
		category := Unavailable
		if strings.Contains(evalErr.Error(), "cost limit") {
			category = LimitExceeded
		}
		return nil, r.work, Invalid(category, e.site.Path, evalErr.Error())
	}
	if err := r.chargeResult(value, e.typ); err != nil {
		return nil, r.work, err
	}
	result, err := fromCEL(value, e.typ)
	if err != nil {
		return nil, r.work, err
	}
	if err = r.charge(int64(proto.Size(result))); err != nil {
		return nil, r.work, err
	}
	return result, r.work, nil
}

func (r *runtimeExpression) chargeResult(value ref.Val, typ Type) error {
	if err := r.charge(1); err != nil {
		return err
	}
	if typ.cardinality == Repeated {
		list, ok := value.(traits.Lister)
		if !ok {
			return literalMismatch()
		}
		iterator := list.Iterator()
		for iterator.HasNext() == types.True {
			if err := r.chargeResult(iterator.Next(), typ.Element()); err != nil {
				return err
			}
		}
		return nil
	}
	if typ.cardinality == Map {
		mapping, ok := value.(traits.Mapper)
		if !ok {
			return literalMismatch()
		}
		iterator := mapping.Iterator()
		var count int64
		for iterator.HasNext() == types.True {
			key := iterator.Next()
			if err := r.chargeResult(key, typ.catalog.scalarType(typ.key)); err != nil {
				return err
			}
			if err := r.chargeResult(mapping.Get(key), typ.Element()); err != nil {
				return err
			}
			count++
		}
		return r.charge(count * int64(bits.Len64(uint64(count))+1))
	}
	if typ.message != nil {
		message, ok := value.Value().(proto.Message)
		if !ok {
			return literalMismatch()
		}
		return r.charge(2*int64(proto.Size(message)) + int64(len(typ.message.FullName())) + 16)
	}
	if typ.any {
		envelope, ok := value.(opaqueAny)
		if !ok {
			return literalMismatch()
		}
		return r.charge(int64(proto.Size(envelope.envelope)) + 16)
	}
	switch scalar := value.(type) {
	case types.String:
		return r.charge(int64(len(scalar)))
	case types.Bytes:
		return r.charge(int64(len(scalar)))
	}
	return nil
}

type runtimeExpression struct {
	copyWork    bool
	ctx         context.Context
	resolve     func(Reference) *celpb.Value
	limit, work int64
}

func (r *runtimeExpression) charge(n int64) error {
	if err := r.ctx.Err(); err != nil {
		return err
	}
	if n < 0 || n > r.limit-r.work {
		return Invalid(LimitExceeded, "expression", "runtime work ceiling exceeded")
	}
	r.work += n
	return nil
}
func boolValue(value bool) *celpb.Value {
	return &celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: value}}
}

type expressionActivation struct {
	expression *Expression
	runtime    *runtimeExpression
	values     map[string]ref.Val
	failure    error
}

func (a *expressionActivation) Parent() interpreter.Activation { return nil }
func (a *expressionActivation) ResolveName(name string) (any, bool) {
	binding, ok := a.expression.variables[name]
	if !ok {
		return nil, false
	}
	value, cached := a.values[name]
	if !cached {
		source, err := a.read(binding)
		if err == nil && source != nil {
			limits := a.expression.limits
			limits.Work = a.runtime.limit - a.runtime.work
			var snapshot *celpb.Value
			var work int64
			snapshot, work, err = SnapshotValue(a.runtime.ctx, source, binding.typ, limits)
			if chargeErr := a.runtime.charge(work); chargeErr != nil {
				err = chargeErr
			}
			if err == nil {
				value, err = toCEL(snapshot, binding.typ, a.expression.registry)
			}
		}
		if err != nil {
			a.failure = err
			value = types.NewErr("%v", err)
		} else if source == nil {
			value = types.OptionalNone
		} else if binding.optional {
			value = types.OptionalOf(value)
		}
		a.values[name] = value
	}
	return value, true
}
func (a *expressionActivation) read(binding *Expression) (*celpb.Value, error) {
	if err := a.runtime.charge(1); err != nil {
		return nil, err
	}
	switch binding.operator {
	case Literal:
		return binding.literal, nil
	case ReferenceValue:
		return a.runtime.resolve(binding.reference), nil
	case ReadPath:
		if binding.path == nil {
			return nil, Invalid(Malformed, "binding", "a bound path is required")
		}
		value, err := a.read(binding.children[0])
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, nil
		}
		return a.runtime.readPath(binding.path, value, binding.children[0].typ)
	default:
		return nil, fmt.Errorf("unsupported bound input %d", binding.operator)
	}
}
