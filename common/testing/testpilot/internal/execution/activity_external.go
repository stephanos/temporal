package execution

import (
	"fmt"

	"go.temporal.io/api/workflowservice/v1"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
)

type activityExternalSettlement struct {
	source                                             *testpilotspb.ActivityExternalSettlement
	controller                                         *graph
	carrier, held, answer, cancel, settlement, cleanup *node
	runSlot                                            string
}

func externalMethod(n *node) string {
	if n == nil || n.method == nil {
		return ""
	}
	return string(n.method.FullName())
}

func externalByID(n *node) bool {
	switch externalMethod(n) {
	case "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCompletedById", "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskFailedById", "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCanceledById":
		return true
	}
	return false
}

func externalExactMethod(n *node) bool {
	if n == nil || n.method == nil {
		return false
	}
	expected := workflowservice.File_temporal_api_workflowservice_v1_service_proto.Services().ByName("WorkflowService").Methods().ByName(n.method.Name())
	return expected != nil && expected.FullName() == n.method.FullName() && !n.method.IsStreamingClient() && !n.method.IsStreamingServer() && ir.SameMessage(expected.Input(), n.method.Input()) && ir.SameMessage(expected.Output(), n.method.Output())
}

func externalSuccessExpression(ref *testpilotspb.InstructionReference) *testpilotspb.Expression {
	status := cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{Instruction: ref, Field: testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS}}})
	literal := cel.Literal(cel.Enum(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED))
	return cel.All(cel.Present(status), cel.Compare("_==_", status, literal))
}

func externalCleanupGuard(b *activityExternalSettlement) bool {
	expected := cel.All(externalSuccessExpression(b.source.Carrier), cel.Not(externalSuccessExpression(b.source.Settlement)))
	return proto.Equal(b.cleanup.source.Guard, expected)
}

func externalAssignmentsOf(n *node) []*testpilotspb.RequestAssignment {
	if n.opcode == contract.ReadEvidence {
		return n.source.Instruction.GetReadEvidence().GetRequestAssignments()
	}
	return n.source.Instruction.GetInvokeRpc().GetRequestAssignments()
}

func externalAssignment(n *node, field string) *testpilotspb.Expression {
	for _, assignment := range externalAssignmentsOf(n) {
		if assignment.GetTarget() == field {
			return assignment.GetValue()
		}
	}
	return nil
}

func (a *admission) bindActivityExternalSettlements() error {
	claimed := map[*node]bool{}
	for i, declaration := range a.prepared.source.GetActivityExternalSettlements() {
		if err := a.charge(9); err != nil {
			return err
		}
		path := fmt.Sprintf("program.activity_external_settlements[%d]", i)
		invalid := func(detail string) error { return ir.Invalid(ir.Malformed, path, detail) }
		resolve := func(ref *testpilotspb.InstructionReference) (*graph, *node) {
			if ref == nil {
				return nil, nil
			}
			g := a.graphIndex[ref.GetEntrypointId()]
			if g == nil {
				return nil, nil
			}
			index, ok := g.index[ref.GetInstructionId()]
			if !ok {
				return nil, nil
			}
			return g, g.nodes[index]
		}
		g, carrier := resolve(declaration.GetCarrier())
		if carrier == nil || g.cleanup || g.context != contract.ControllerEntrypoint || carrier.opcode != contract.InvokeRPC || externalMethod(carrier) != "temporal.api.workflowservice.v1.WorkflowService.StartActivityExecution" {
			return invalid("external settlement requires an exact standalone activity carrier")
		}
		basis := &activityExternalSettlement{source: declaration, controller: g, carrier: carrier}
		for _, read := range carrier.source.Instruction.GetInvokeRpc().GetResponseReads() {
			if read.GetPath() == "run_id" && len(read.GetTargets()) == 1 {
				basis.runSlot = read.GetTargets()[0].GetSlotId()
			}
		}
		if basis.runSlot == "" {
			return invalid("carrier must learn the actual activity run_id into one Slot")
		}
		for _, binding := range []struct {
			ref     *testpilotspb.InstructionReference
			target  **node
			cleanup bool
		}{
			{declaration.GetAnswer(), &basis.answer, false}, {declaration.GetSettlement(), &basis.settlement, false}, {declaration.GetCleanup(), &basis.cleanup, true},
		} {
			owner, n := resolve(binding.ref)
			if n == nil || binding.cleanup != owner.cleanup || !binding.cleanup && owner != g || n.timeoutMilliseconds <= 0 || n.maxAttempts <= 0 || claimed[n] {
				return invalid("missing, duplicate or unbounded settlement/answer/cleanup reference")
			}
			*binding.target = n
			claimed[n] = true
		}
		method := externalMethod(basis.answer)
		if basis.answer.opcode != contract.InvokeRPC || method != "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskFailedById" && method != "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCanceledById" && method != "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCompletedById" {
			return invalid("exact typed activity ByID answer required")
		}
		if basis.cleanup.opcode != contract.InvokeRPC || externalMethod(basis.cleanup) != "temporal.api.workflowservice.v1.WorkflowService.TerminateActivityExecution" {
			return invalid("bounded Cleanup must terminate this standalone activity")
		}
		if basis.settlement.opcode != contract.ReadEvidence || externalMethod(basis.settlement) != "temporal.api.workflowservice.v1.WorkflowService.DescribeActivityExecution" || basis.settlement.source.Instruction.GetReadEvidence().GetUntil() == nil {
			return invalid("bounded terminal Describe settlement required")
		}
		if declaration.GetActivityEntrypointId() == "" {
			if method != "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCompletedById" || declaration.GetPendingSlotId() != "" || declaration.GetHeld() != nil || declaration.GetRequestCancel() != nil || declaration.GetReservationOrdinal() != 0 {
				return invalid("controller-only completion cannot claim a worker publication")
			}
		} else {
			if method == "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCompletedById" {
				return invalid("worker-held external answer must fail or cancel")
			}
			owner, held := resolve(declaration.GetHeld())
			basis.held = held
			if held == nil || owner != g || held.opcode != contract.ReadEvidence || externalMethod(held) != "temporal.api.workflowservice.v1.WorkflowService.DescribeActivityExecution" || held.source.Instruction.GetReadEvidence().GetUntil() == nil || held.timeoutMilliseconds <= 0 || held.maxAttempts <= 0 || claimed[held] {
				return invalid("bounded held-state Describe read required")
			}
			claimed[held] = true
			activity := a.graphIndex[declaration.GetActivityEntrypointId()]
			ordinal := declaration.GetReservationOrdinal()
			if activity == nil || activity.context != contract.ActivityEntrypoint || ordinal < 0 || ordinal >= int64(len(activity.activityAttempts)) {
				return invalid("external settlement needs an actual activity reservation ordinal")
			}
			group := activity.activityAttempts[ordinal]
			terminal := activity.nodes[group[len(group)-1]].source.Instruction.GetActivityAttemptWithholding()
			if terminal == nil || terminal.GetMode() != testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING || !proto.Equal(terminal.GetExternalSettlement(), declaration.GetAnswer()) {
				return invalid("SDK_PENDING terminal must explicitly name its external answer basis")
			}
			typ, exists := a.prepared.slots[declaration.GetPendingSlotId()]
			if !exists || typ.Cardinality() != ir.Singular || !ir.SameMessage(typ.Message(), (&testpilotspb.ActivityAttempt{}).ProtoReflect().Descriptor()) {
				return invalid("publication Slot must contain the exact ActivityAttempt message")
			}
			if err := a.addWriter(declaration.GetPendingSlotId(), slotWriter{graph: g, node: g.index[carrier.source.InstructionId], asynchronous: true}); err != nil {
				return err
			}
			if method == "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCanceledById" {
				owner, cancel := resolve(declaration.GetRequestCancel())
				basis.cancel = cancel
				if cancel == nil || owner != g || cancel.opcode != contract.InvokeRPC || externalMethod(cancel) != "temporal.api.workflowservice.v1.WorkflowService.RequestCancelActivityExecution" || claimed[cancel] {
					return invalid("canceled ByID requires a preceding typed RequestCancel")
				}
				claimed[cancel] = true
				if !expressionLiteral(externalAssignment(basis.settlement, "include_outcome")).GetBoolValue() {
					return invalid("canceled settlement must include the service outcome")
				}
			} else if declaration.GetRequestCancel() != nil {
				return invalid("failed ByID cannot claim a cancellation request")
			}
		}
		for _, n := range []*node{basis.held, basis.cancel, basis.answer, basis.settlement, basis.cleanup} {
			if n == nil {
				continue
			}
			for _, field := range []string{"namespace", "activity_id"} {
				if externalAssignment(carrier, field) == nil || !proto.Equal(externalAssignment(carrier, field), externalAssignment(n, field)) {
					return invalid("namespace name and activity ID must be those of the carrier")
				}
			}
			if expressionReference(externalAssignment(n, "run_id")).GetSlotId() != basis.runSlot || externalAssignment(n, "workflow_id") != nil {
				return invalid("standalone request needs learned actual run_id and no workflow_id")
			}
		}
		for _, n := range []*node{basis.carrier, basis.held, basis.cancel, basis.answer, basis.settlement, basis.cleanup} {
			if n != nil && !externalExactMethod(n) {
				return invalid("external settlement requires exact WorkflowService method and request/response descriptors")
			}
		}
		a.prepared.external = append(a.prepared.external, basis)
	}
	for _, g := range a.prepared.graphs {
		for _, n := range g.nodes {
			if externalByID(n) && externalAssignment(n, "workflow_id") == nil && !claimed[n] {
				return ir.Invalid(ir.Malformed, nodePath(g, n), "standalone ByID answer requires an explicit external settlement basis")
			}
			if ref := n.source.Instruction.GetActivityAttemptWithholding().GetExternalSettlement(); ref != nil {
				found := false
				for _, b := range a.prepared.external {
					if b.source.GetActivityEntrypointId() == g.id && proto.Equal(b.source.GetAnswer(), ref) {
						found = true
					}
				}
				if !found {
					return ir.Invalid(ir.Malformed, nodePath(g, n), "withholding names no admitted external settlement")
				}
			}
		}
	}
	return nil
}

func externalRequires(g *graph, n, predecessor *node, seen map[*node]bool) bool {
	if n == nil || predecessor == nil || seen[n] {
		return false
	}
	seen[n] = true
	for id := range successFacts(n.guard) {
		index, ok := g.index[id]
		if !ok || !n.ancestors[index] {
			continue
		}
		previous := g.nodes[index]
		if previous == predecessor || externalRequires(g, previous, predecessor, seen) {
			return true
		}
	}
	return false
}

func (a *admission) checkActivityExternalSettlements() error {
	for _, b := range a.prepared.external {
		invalid := func(detail string) error { return ir.Invalid(ir.Malformed, "activity_external_settlement", detail) }
		if !externalCleanupGuard(b) {
			return invalid("Cleanup must follow successful carrier unless terminal Describe already succeeded")
		}
		if !externalRequires(b.controller, b.settlement, b.answer, map[*node]bool{}) {
			return invalid("terminal settlement must follow a successful ByID answer")
		}
		if b.held == nil {
			for _, g := range a.prepared.graphs {
				if !g.cleanup && g.context != contract.ControllerEntrypoint {
					return invalid("scheduled completion requires controller-only entrypoints, not an empty worker script")
				}
			}
			if len(b.carrier.reservations) != 0 {
				return invalid("controller-only completion cannot reserve worker activations")
			}
			if !externalRequires(b.controller, b.answer, b.carrier, map[*node]bool{}) {
				return invalid("scheduled completion must follow its successful carrier")
			}
			continue
		}
		carried := false
		for _, reservation := range b.carrier.reservations {
			if reservation.EntrypointID == b.source.ActivityEntrypointId && b.source.ReservationOrdinal < reservation.Count {
				carried = true
			}
		}
		if !carried {
			return invalid("external basis does not match the carrier reservation")
		}
		var awaited *node
		for _, n := range b.controller.nodes {
			if n.opcode == contract.AwaitSlot && n.source.Instruction.GetAwaitSlot().GetSlotId() == b.source.PendingSlotId {
				if awaited != nil {
					return invalid("publication is awaited twice")
				}
				awaited = n
			}
		}
		first := b.held
		if b.cancel != nil {
			first = b.cancel
			if !externalRequires(b.controller, b.held, b.cancel, map[*node]bool{}) {
				return invalid("held cancellation read must follow RequestCancel")
			}
		}
		if !externalRequires(b.controller, first, awaited, map[*node]bool{}) || !externalRequires(b.controller, b.answer, b.held, map[*node]bool{}) {
			return invalid("external answer requires successful publication and held-state reads in order")
		}
	}
	return nil
}
