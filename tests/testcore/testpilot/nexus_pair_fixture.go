package testpilot

import (
	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

// NexusPairFixture is the pair Model's one Query: two instances of the caller Model's operation in
// one workflow, each answered asynchronously by its own handler and completed by the caller.
const NexusPairFixture = "nexusPairTests-bothComplete"

// The two operations the pair schedules, named after the realization's operation and the
// instance, and the one monitor rule per instance the field relation lowers to.
const (
	NexusPairService         = "umpire.case.service"
	NexusPairFirstOperation  = "complete-1"
	NexusPairSecondOperation = "complete-2"
	NexusPairFirstRuleID     = "relation-1"
	NexusPairSecondRuleID    = "relation-2"
)

// NexusPairProfile is the hand-written Profile the pair Case prepares under: the caller
// realization's five roles with two Nexus handler activations reserved by the workflow start, one
// per instance, the typed instructions and the read the realization polls with, and the four
// bindings the roles reference.
func NexusPairProfile(catalog *testpilot.Catalog, environment NexusCallerEnvironment) testpilot.ProfileSpec {
	profile := caseProfile("nexus-pair-profile", catalog,
		[]testpilot.RolePolicy{
			workflowServiceRole(2),
			{ID: "temporal.worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.handler-task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		realizedNexusCapabilities(),
		[]testpilot.EnvironmentBinding{
			{ID: NexusCallerWorkerNamespaceBindingID, Value: environment.Namespace},
			{ID: NexusCallerTaskQueueBindingID, Value: environment.TaskQueue},
			{ID: NexusCallerHandlerTaskQueueBindingID, Value: environment.HandlerTaskQueue},
			{ID: NexusCallerEndpointBindingID, Value: environment.NexusEndpoint},
		})
	profile.Opcodes = append(profile.Opcodes, testpilot.ReadEvidence)
	profile.CommandTypes = []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION}
	return profile
}
