package execution

import (
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
)

func (a *admission) bindActivityAttempts() error {
	for _, g := range a.prepared.graphs {
		if g.context != contract.ActivityEntrypoint {
			continue
		}
		var group []int
		for position, index := range g.order {
			n := g.nodes[index]
			if position == 0 && len(n.dependencies) != 0 || position > 0 && (len(n.dependencies) != 1 || n.dependencies[0] != g.order[position-1]) {
				return ir.Invalid(ir.Unsupported, nodePath(g, n), "activity attempts require a linear instruction sequence")
			}
			if n.opcode == contract.ActivityHeartbeat && len(group) != 0 {
				return ir.Invalid(ir.Unsupported, nodePath(g, n), "an activity attempt admits at most one heartbeat prefix")
			}
			group = append(group, index)
			switch n.opcode {
			case contract.Finish, contract.ActivityAttemptFailure, contract.ActivityAttemptCancellation, contract.ActivityAttemptWithholding:
				g.activityAttempts = append(g.activityAttempts, group)
				group = nil
			case contract.ActivityHeartbeat:
			default:
				return ir.Invalid(ir.Unsupported, nodePath(g, n), "unsupported activity attempt instruction")
			}
		}
		if len(group) != 0 {
			return ir.Invalid(ir.Malformed, nodePath(g, g.nodes[group[0]]), "an activity attempt requires one terminal disposition")
		}
	}
	return nil
}
