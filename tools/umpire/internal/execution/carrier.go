package execution

import (
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
)

func (a *admission) bindReservationCarriers() error {
	for _, graph := range a.prepared.graphs {
		for _, node := range graph.nodes {
			if len(node.source.ActivationReservations) == 0 {
				continue
			}
			if node.opcode != InvokeRPC {
				return invalid(ir.Unsupported, nodePath(graph, node), "activation reservations require an authorized carrier RPC")
			}
			rpc := node.source.Instruction.GetInvokeRpc()
			carrier, ok := a.carriers[rpc.EndpointRoleId][rpc.Method]
			if !ok {
				return invalid(ir.Unsupported, nodePath(graph, node), "reservation-bearing RPC has no carrier authority")
			}
			if err := a.checkCarrierShape(graph, node, carrier); err != nil {
				return err
			}
			plan, err := a.compileCarrierTopology(graph, node)
			if err != nil {
				return err
			}
			a.prepared.carriers[carrierCoordinate{entrypointID: graph.id, instructionID: node.source.InstructionId}] = plan
		}
	}
	return nil
}

func (a *admission) checkCarrierShape(controller *graph, node *node, carrier ReservationCarrierPolicy) error {
	maximum := make(map[umpirespb.EntrypointContext]int64, len(carrier.Shapes))
	for _, shape := range carrier.Shapes {
		maximum[shape.Context] = shape.MaximumCount
	}
	counts := map[umpirespb.EntrypointContext]int64{}
	for _, reservation := range node.source.ActivationReservations {
		target := a.graphIndex[reservation.EntrypointId]
		if target == nil {
			return invalid(ir.Unknown, nodePath(controller, node), "carrier reservation target is missing")
		}
		count := counts[target.context]
		if allowed := maximum[target.context]; allowed == 0 || reservation.Count > allowed-count {
			return invalid(ir.Unsupported, nodePath(controller, node), "carrier reservation shape or cardinality is unauthorized")
		}
		counts[target.context] = count + reservation.Count
	}
	return nil
}

func (a *admission) compileCarrierTopology(controller *graph, node *node) (ReservationCarrierPlan, error) {
	rpc := node.source.Instruction.GetInvokeRpc()
	reservations, handlers, handlerIndex, err := a.carrierReservations(controller, node)
	if err != nil {
		return ReservationCarrierPlan{}, err
	}
	plan := ReservationCarrierPlan{EndpointRoleID: rpc.EndpointRoleId, Method: rpc.Method, Reservations: reservations}
	handlerOrdinals := make(map[string]int64, len(handlers))
	for _, reservation := range node.source.ActivationReservations {
		workflow := a.graphIndex[reservation.EntrypointId]
		if workflow.context != umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW {
			continue
		}
		if err := a.appendWorkflowRoutes(controller, node, workflow, reservation.Count, handlerIndex, handlerOrdinals, &plan); err != nil {
			return ReservationCarrierPlan{}, err
		}
	}
	for _, handler := range handlers {
		if err := a.charge(1); err != nil {
			return ReservationCarrierPlan{}, err
		}
		if handlerOrdinals[handler.graph.id] != handler.count {
			return ReservationCarrierPlan{}, invalid(ir.Malformed, nodePath(controller, node), "reserved Nexus handler count does not match potential sources")
		}
	}
	return plan, nil
}

type reservedHandler struct {
	graph *graph
	count int64
}

type nexusOperation struct {
	service   string
	operation string
}

func (a *admission) carrierReservations(controller *graph, node *node) ([]ReservationTopology, []reservedHandler, map[nexusOperation]reservedHandler, error) {
	reservations := make([]ReservationTopology, 0, len(node.source.ActivationReservations))
	var handlers []reservedHandler
	handlerIndex := make(map[nexusOperation]reservedHandler)
	for _, reservation := range node.source.ActivationReservations {
		if err := a.charge(1); err != nil {
			return nil, nil, nil, err
		}
		target := a.graphIndex[reservation.EntrypointId]
		reservations = append(reservations, ReservationTopology{EntrypointID: target.id, Context: target.context, Count: reservation.Count})
		if target.context == umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER {
			if err := a.charge(1); err != nil {
				return nil, nil, nil, err
			}
			handler := reservedHandler{graph: target, count: reservation.Count}
			binding := target.activation.GetNexusHandler()
			operation := nexusOperation{service: binding.Service, operation: binding.Operation}
			if _, exists := handlerIndex[operation]; exists {
				return nil, nil, nil, invalid(ir.Malformed, nodePath(controller, node), "ambiguous reserved Nexus handler mapping")
			}
			handlerIndex[operation] = handler
			handlers = append(handlers, handler)
		}
	}
	return reservations, handlers, handlerIndex, nil
}

func (a *admission) appendWorkflowRoutes(controller *graph, node *node, workflow *graph, count int64, handlers map[nexusOperation]reservedHandler, ordinals map[string]int64, plan *ReservationCarrierPlan) error {
	for _, index := range workflow.order {
		source := workflow.nodes[index]
		if source.opcode != StartNexusOperation {
			continue
		}
		if err := a.charge(1); err != nil {
			return err
		}
		start := source.source.Instruction.GetStartNexusOperation()
		handler, ok := handlers[nexusOperation{service: start.Service, operation: start.Operation}]
		if !ok {
			return invalid(ir.Unavailable, nodePath(controller, node), "missing or crossed reserved Nexus handler mapping")
		}
		for workflowOrdinal := int64(0); workflowOrdinal < count; workflowOrdinal++ {
			handlerOrdinal := ordinals[handler.graph.id]
			if handlerOrdinal >= handler.count {
				return invalid(ir.LimitExceeded, nodePath(controller, node), "reserved Nexus handler count does not match potential sources")
			}
			if err := a.charge(1); err != nil {
				return err
			}
			plan.Routes = append(plan.Routes, ReservationRoute{WorkflowEntrypointID: workflow.id, WorkflowOrdinal: workflowOrdinal, SourceInstructionID: source.source.InstructionId, HandlerEntrypointID: handler.graph.id, HandlerOrdinal: handlerOrdinal})
			ordinals[handler.graph.id] = handlerOrdinal + 1
		}
	}
	return nil
}
