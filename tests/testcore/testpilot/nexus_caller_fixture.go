package testpilot

import (
	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

const (
	NexusCallerWorkerNamespaceBindingID = "temporal.worker.namespace"
	NexusCallerTaskQueueBindingID       = "temporal.task-queue.resource"
	NexusCallerEndpointBindingID        = "temporal.nexus-endpoint.resource"
)

// NexusCallerAsyncCompletionFixture is the caller Model's Query 2 fixture, the async reply then
// succeeded callback, which the hand-written Profile below is the derivation oracle for.
const NexusCallerAsyncCompletionFixture = "nexusCallerTests-asyncCompletion"

type NexusCallerEnvironment struct {
	Namespace     string
	TaskQueue     string
	NexusEndpoint string
}

// NexusCallerProfile is the hand-written Profile every caller-set Case prepares under: the four
// roles the realization declares, the Opcodes its scaffolding and its typed instructions need, and
// the three bindings the roles reference.
func NexusCallerProfile(catalog *testpilot.Catalog, environment NexusCallerEnvironment) testpilot.ProfileSpec {
	profile := caseProfile("nexus-caller-profile", catalog,
		[]testpilot.RolePolicy{
			workflowServiceRole(1),
			{ID: "temporal.worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		realizedNexusCapabilities(),
		[]testpilot.EnvironmentBinding{
			{ID: NexusCallerWorkerNamespaceBindingID, Value: environment.Namespace},
			{ID: NexusCallerTaskQueueBindingID, Value: environment.TaskQueue},
			{ID: NexusCallerEndpointBindingID, Value: environment.NexusEndpoint},
		})
	// The realization schedules the operation through a workflow command, which the Profile
	// admits per command type.
	profile.CommandTypes = []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION}
	return profile
}
