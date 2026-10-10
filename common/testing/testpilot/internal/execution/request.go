package execution

import (
	"context"

	celpb "cel.dev/expr"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

// evaluateGuarded evaluates the node's guard and, when it holds, the node's input, through evaluate,
// which charges both to one work bucket. A false guard disables the node and leaves the input
// unevaluated; a node without an input yields nil.
func (n *node) evaluateGuarded(evaluate func(*ir.Expression) (*celpb.Value, error)) (*celpb.Value, bool, error) {
	if n.guard != nil {
		guard, err := evaluate(n.guard)
		if err != nil || !guard.GetBoolValue() {
			return nil, false, err
		}
	}
	if n.input == nil {
		return nil, true, nil
	}
	input, err := evaluate(n.input)
	if err != nil {
		return nil, false, err
	}
	return input, true, nil
}

func (a *activationValues) request(ctx context.Context, c contract.Coordinate, limit int64) (proto.Message, bool, int64, error) {
	n, err := a.instruction(c)
	if err != nil {
		return nil, false, 0, err
	}
	w, err := a.newWork(ctx, limit)
	if err != nil {
		return nil, false, 0, err
	}
	if n.opcode != contract.InvokeRPC && n.opcode != contract.ReadEvidence {
		return nil, false, 0, ir.Invalid(ir.TypeMismatch, "request", "RPC instruction required")
	}
	if _, enabled, err := n.evaluateGuarded(func(e *ir.Expression) (*celpb.Value, error) { return a.evaluate(w, e) }); err != nil || !enabled {
		return nil, false, w.work, err
	}
	writes := make([]ir.Write, 0, len(n.assignments))
	for _, assignment := range n.assignments {
		value, err := a.evaluate(w, assignment.value)
		if err != nil {
			return nil, false, w.work, err
		}
		writes = append(writes, ir.Write{Path: assignment.target, Value: value})
	}
	request, work, err := ir.BuildRequest(ctx, n.method.Input(), writes, w.remaining(a.store.program.limits.MaxRequestBytes))
	w.work += work
	return request, err == nil, w.work, err
}
