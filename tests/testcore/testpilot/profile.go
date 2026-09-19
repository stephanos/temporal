package testpilot

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
)

const (
	startWorkflowExecutionMethod      = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	getWorkflowExecutionHistoryMethod = "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"
)

// workflowServiceRole authorizes the two WorkflowService methods every Case in this package uses
// on one endpoint role, and reserves one workflow entrypoint plus nexusHandlers Nexus handler
// entrypoints on its StartWorkflowExecution. A Case that reserves no Nexus handler passes zero and
// gets no Nexus handler shape at all.
func workflowServiceRole(nexusHandlers int64) testpilot.RolePolicy {
	shapes := []testpilot.ReservationCarrierShape{
		{Kind: testpilot.WorkflowEntrypoint, MaximumCount: 1},
	}
	if nexusHandlers > 0 {
		shapes = append(shapes, testpilot.ReservationCarrierShape{
			Kind: testpilot.NexusHandlerEntrypoint, MaximumCount: nexusHandlers,
		})
	}
	return testpilot.RolePolicy{
		ID: "temporal.workflow-service", Kind: testpilotspb.ROLE_KIND_ENDPOINT,
		Methods: []string{startWorkflowExecutionMethod, getWorkflowExecutionHistoryMethod},
		ReservationCarriers: []testpilot.ReservationCarrierPolicy{{
			Method: startWorkflowExecutionMethod, Shapes: shapes,
		}},
	}
}

// nexusCapabilities is the instruction set a workflow-owned Nexus Case needs. The unary Case
// declares its own narrower set instead of borrowing this one.
func nexusCapabilities() []testpilot.Opcode {
	return []testpilot.Opcode{
		testpilot.InvokeRPC, testpilot.AwaitSlot, testpilot.CompleteNexusOperation,
		testpilot.StartNexusOperation, testpilot.Await, testpilot.Finish, testpilot.RespondNexus,
	}
}

// realizedNexusCapabilities is the instruction set a Case the Nexus realization produces needs:
// the typed worker instructions in place of the untyped Nexus ones (fn-85 R10), which the typed
// Nexus example keeps until fn-86 removes them.
func realizedNexusCapabilities() []testpilot.Opcode {
	return []testpilot.Opcode{
		testpilot.InvokeRPC, testpilot.AwaitSlot, testpilot.Await, testpilot.Finish,
		testpilot.WorkflowCommand, testpilot.NexusHandlerReply, testpilot.NexusOperationCompletion,
	}
}

// caseProfile assembles one ProfileSpec from the parts a Case chose, under the Temporal default
// resource ceilings and instruction limits every Temporal Profile shares.
func caseProfile(
	identity string,
	catalog *testpilot.Catalog,
	roles []testpilot.RolePolicy,
	opcodes []testpilot.Opcode,
	bindings []testpilot.EnvironmentBinding,
) testpilot.ProfileSpec {
	programLimits, contractLimits, correlatedLimits := temporal.DefaultCeilings()
	return testpilot.ProfileSpec{
		Identity:            identity,
		Catalog:             catalog,
		Roles:               roles,
		Opcodes:             opcodes,
		EnvironmentBindings: bindings,
		ProgramLimits:       programLimits,
		ContractLimits:      contractLimits,
		CorrelatedLimits:    correlatedLimits,
		InstructionDefaults: temporal.DefaultInstructionLimits(),
	}
}
