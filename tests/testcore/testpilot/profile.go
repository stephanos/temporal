package testpilot

import (
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
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
		{Context: testpilotspb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 1},
	}
	if nexusHandlers > 0 {
		shapes = append(shapes, testpilot.ReservationCarrierShape{
			Context: testpilotspb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: nexusHandlers,
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
func nexusCapabilities() []testpilot.Capability {
	return []testpilot.Capability{
		testpilot.InvokeRPC, testpilot.AwaitSlot, testpilot.CompleteNexusOperation,
		testpilot.StartNexusOperation, testpilot.Await, testpilot.Finish, testpilot.RespondNexus,
	}
}

// caseProfile assembles one ProfileSpec from the parts a Case chose, carrying the Case's own
// declared Program and Contract limits rather than a shared ceiling.
func caseProfile(
	identity string,
	catalog *testpilot.Catalog,
	source *testpilotspb.Case,
	roles []testpilot.RolePolicy,
	capabilities []testpilot.Capability,
	bindings []testpilot.EnvironmentBinding,
) testpilot.ProfileSpec {
	return testpilot.ProfileSpec{
		Identity:            identity,
		Catalog:             catalog,
		Roles:               roles,
		Capabilities:        capabilities,
		EnvironmentBindings: bindings,
		ProgramLimits:       proto.CloneOf(source.GetProgram().GetLimits()),
		ContractLimits:      proto.CloneOf(source.GetContract().GetLimits()),
	}
}
