package execution

import (
	"fmt"

	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
)

// deriveReservations gives each reservation carrier its reservations. An ordinary controller
// instruction invoking a method the Profile names as a carrier on that endpoint reserves one activation
// of each workflow or Nexus-handler entrypoint whose kind the carrier's shapes admit. One entrypoint's
// activation has one carrier, so two instructions that could both carry it reject.
func (a *admission) deriveReservations() error {
	carriedBy := map[string]*node{}
	for _, controller := range a.prepared.graphs {
		if controller.cleanup || controller.context != contract.ControllerEntrypoint {
			continue
		}
		for _, carrierNode := range controller.nodes {
			if carrierNode.opcode != contract.InvokeRPC {
				continue
			}
			rpc := carrierNode.source.Instruction.GetInvokeRpc()
			carrier, ok := a.carriers[rpc.EndpointRoleId][rpc.Method]
			if !ok {
				continue
			}
			admitted := map[contract.EntrypointKind]bool{}
			for _, shape := range carrier.Shapes {
				admitted[shape.Kind] = true
			}
			for _, target := range a.prepared.graphs {
				if target.cleanup || !admitted[target.context] {
					continue
				}
				if err := a.charge(1); err != nil {
					return err
				}
				if previous, claimed := carriedBy[target.id]; claimed {
					return invalid(ir.Unsupported, nodePath(controller, carrierNode), fmt.Sprintf("instructions %s and %s both carry the reservation of entrypoint %s", previous.source.InstructionId, carrierNode.source.InstructionId, target.id))
				}
				carriedBy[target.id] = carrierNode
				carrierNode.reservations = append(carrierNode.reservations, contract.ReservationTopology{EntrypointID: target.id, Kind: target.context, Count: 1})
			}
		}
	}
	return nil
}

func (a *admission) bindReservationCarriers() error {
	for _, graph := range a.prepared.graphs {
		for _, node := range graph.nodes {
			if len(node.reservations) == 0 {
				continue
			}
			rpc := node.source.Instruction.GetInvokeRpc()
			carrier := a.carriers[rpc.EndpointRoleId][rpc.Method]
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

func (a *admission) checkCarrierShape(controller *graph, node *node, carrier contract.ReservationCarrierPolicy) error {
	maximum := make(map[contract.EntrypointKind]int64, len(carrier.Shapes))
	for _, shape := range carrier.Shapes {
		maximum[shape.Kind] = shape.MaximumCount
	}
	counts := map[contract.EntrypointKind]int64{}
	for _, reservation := range node.reservations {
		count := counts[reservation.Kind]
		if allowed := maximum[reservation.Kind]; reservation.Count > allowed-count {
			return invalid(ir.Unsupported, nodePath(controller, node), "carrier reservation shape or cardinality is unauthorized")
		}
		counts[reservation.Kind] = count + reservation.Count
	}
	return nil
}

func (a *admission) compileCarrierTopology(controller *graph, node *node) (contract.ReservationCarrierPlan, error) {
	rpc := node.source.Instruction.GetInvokeRpc()
	reservations, handlers, handlerIndex, err := a.carrierReservations(controller, node)
	if err != nil {
		return contract.ReservationCarrierPlan{}, err
	}
	plan := contract.ReservationCarrierPlan{EndpointRoleID: rpc.EndpointRoleId, Method: rpc.Method, Reservations: reservations}
	handlerOrdinals := make(map[string]int64, len(handlers))
	for _, reservation := range node.reservations {
		workflow := a.graphIndex[reservation.EntrypointID]
		if workflow.context != contract.WorkflowEntrypoint {
			continue
		}
		if err := a.appendWorkflowRoutes(controller, node, workflow, reservation.Count, handlerIndex, handlerOrdinals, &plan); err != nil {
			return contract.ReservationCarrierPlan{}, err
		}
	}
	for _, handler := range handlers {
		if err := a.charge(1); err != nil {
			return contract.ReservationCarrierPlan{}, err
		}
		if handlerOrdinals[handler.graph.id] != handler.count {
			return contract.ReservationCarrierPlan{}, invalid(ir.Malformed, nodePath(controller, node), "reserved Nexus handler count does not match potential sources")
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

func (a *admission) carrierReservations(controller *graph, node *node) ([]contract.ReservationTopology, []reservedHandler, map[nexusOperation]reservedHandler, error) {
	reservations := make([]contract.ReservationTopology, 0, len(node.reservations))
	var handlers []reservedHandler
	handlerIndex := make(map[nexusOperation]reservedHandler)
	for _, reservation := range node.reservations {
		if err := a.charge(1); err != nil {
			return nil, nil, nil, err
		}
		target := a.graphIndex[reservation.EntrypointID]
		reservations = append(reservations, reservation)
		if target.context == contract.NexusHandlerEntrypoint {
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

func (a *admission) appendWorkflowRoutes(controller *graph, node *node, workflow *graph, count int64, handlers map[nexusOperation]reservedHandler, ordinals map[string]int64, plan *contract.ReservationCarrierPlan) error {
	for _, index := range workflow.order {
		source := workflow.nodes[index]
		if !startsNexusOperation(source.source.Instruction) {
			continue
		}
		if err := a.charge(1); err != nil {
			return err
		}
		started := nexusOperationOf(source.source.Instruction)
		handler, ok := handlers[started]
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
			plan.Routes = append(plan.Routes, contract.ReservationRoute{WorkflowEntrypointID: workflow.id, WorkflowOrdinal: workflowOrdinal, SourceInstructionID: source.source.InstructionId, HandlerEntrypointID: handler.graph.id, HandlerOrdinal: handlerOrdinal})
			ordinals[handler.graph.id] = handlerOrdinal + 1
		}
	}
	return nil
}
