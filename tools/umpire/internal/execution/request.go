package execution

import (
	"context"

	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
)

func (a *activationValues) request(ctx context.Context, c Coordinate, limit int64) (proto.Message, bool, int64, error) {
	n, err := a.instruction(c)
	if err != nil {
		return nil, false, 0, err
	}
	w, err := a.newWork(ctx, limit)
	if err != nil {
		return nil, false, 0, err
	}
	if n.opcode != InvokeRPC {
		return nil, false, 0, invalid(ir.TypeMismatch, "request", "RPC instruction required")
	}
	if n.guard != nil {
		guard, err := a.evaluate(w, n.guard)
		if err != nil {
			return nil, false, w.work, err
		}
		if !guard.GetBoolValue() {
			return nil, false, w.work, nil
		}
	}
	writes := make([]ir.Write, 0, len(n.assignments))
	for _, assignment := range n.assignments {
		value, err := a.evaluate(w, assignment.value)
		if err != nil {
			return nil, false, w.work, err
		}
		writes = append(writes, ir.Write{Path: assignment.target, Value: value})
	}
	request, work, err := ir.BuildRequest(ctx, n.method.Input(), writes, w.remaining(a.store.program.source.Limits.MaxRequestBytes))
	w.work += work
	return request, err == nil, w.work, err
}
