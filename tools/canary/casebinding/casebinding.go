// Package casebinding is the production canary's fixed Case: the Lean renderer's canonical bytes
// for the one admitted canary Case the canary runs, and the hand-authored Driver Profile it is
// prepared under. The Profile is an authorization snapshot (QLF-01), so it is written out here as
// literals rather than derived at run time: a change to Testpilot's derivation or its default
// ceilings can never silently widen what the production credential may do. Only the environment's
// coordinates are filled in when the canary binds.
package casebinding

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"

	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/umpire/recordedrun"
)

// The pinned canary Case, `make canary-gen-case` writes it and `make canary-check-case` diffs a
// fresh render against it.
//
//go:embed testdata/nexusCallerCanary-syncCompletion-case.json
var pinned []byte

// The workflow-service methods the Case's endpoint role invokes: the public ones that start its
// workflow and read its history, and nothing else.
const (
	startWorkflowExecution      = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	getWorkflowExecutionHistory = "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"
)

// Case is the pinned Case's canonical bytes.
func Case() []byte { return bytes.Clone(pinned) }

// Identity is the pinned Case's identity, as fn-26 and the recorded Run name it.
func Identity() (string, error) { return recordedrun.CaseIdentity(pinned) }

// ProfileSpec is the canary's hand-authored Driver Profile for the pinned Case under the given
// environment's coordinates: the roles, methods, reservation carriers, opcodes and command types
// the Case needs, and the Temporal Profile's ceilings and instruction defaults, all as literals.
func ProfileSpec(canary *policy.Policy, catalog *testpilot.Catalog, environment testpilotdriver.Environment) testpilot.ProfileSpec {
	return testpilot.ProfileSpec{
		Identity: canary.CaseProfile,
		Catalog:  catalog,
		Roles: []testpilot.RolePolicy{
			{
				ID: "temporal.workflow-service", Kind: testpilotspb.ROLE_KIND_ENDPOINT,
				Methods: []string{startWorkflowExecution, getWorkflowExecutionHistory},
				ReservationCarriers: []testpilot.ReservationCarrierPolicy{{
					Method: startWorkflowExecution,
					Shapes: []testpilot.ReservationCarrierShape{
						{Kind: testpilot.WorkflowEntrypoint, MaximumCount: 1},
						{Kind: testpilot.NexusHandlerEntrypoint, MaximumCount: 1},
					},
				}},
			},
			{ID: "temporal.worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.handler-task-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
		},
		Opcodes: []testpilot.Opcode{
			testpilot.InvokeRPC, testpilot.Await, testpilot.Finish, testpilot.WorkflowCommand,
			testpilot.NexusHandlerReply, testpilot.ReadEvidence,
		},
		CommandTypes: []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION},
		EnvironmentBindings: []testpilot.EnvironmentBinding{
			{ID: "temporal.worker.namespace", Value: environment.Namespace},
			{ID: "temporal.task-queue.resource", Value: environment.TaskQueue},
			{ID: "temporal.handler-task-queue.resource", Value: environment.HandlerTaskQueue},
			{ID: "temporal.nexus-endpoint.resource", Value: environment.NexusEndpoint},
		},
		ProgramLimits: &testpilotspb.ProgramLimits{
			MaxEntrypoints: 4, MaxNodes: 16, MaxEdges: 24, MaxActivations: 8, MaxAttempts: 16,
			MaxRunEvents: 512, MaxExpressionDepth: 12, MaxPathFanout: 32,
			MaxRequestBytes: 32768, MaxResponseBytes: 8192,
			MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 20000,
			MaxInstructionEmittedEvents: 128, MaxInstructionResponseBytes: 8192,
		},
		ContractLimits: &testpilotspb.ContractLimits{
			MaxRules: 4, MaxStates: 16, MaxTransitions: 64, MaxExpressionDepth: 12,
			MaxWorkPerEvent: 4000000, MaxTotalWork: 1000000000, MaxCaptures: 64, MaxCaptureBytes: 65536,
		},
		CorrelatedLimits: &testpilotspb.CorrelatedLimits{
			MaxEvents: 64, MaxBuffered: 32, MaxKeys: 8, MaxSupport: 256, MaxProjectionWork: 1000000000,
			MaxEventBytes: 512, MaxSemanticTransitions: 32, MaxObligations: 16, MaxObligationWork: 100000000,
			MaxCaptures: 16, MaxCorrelationDepth: 2,
		},
		InstructionDefaults: testpilot.InstructionDefaults{TimeoutMilliseconds: 10000, MaxAttempts: 1},
	}
}

// Bound is the pinned Case prepared for one environment: its source, its identity, the Profile it
// was prepared under, and the prepared Case the controller runs.
type Bound struct {
	Source       *testpilotspb.Case
	CaseIdentity string
	Profile      testpilot.ProfileSpec
	Prepared     *testpilot.PreparedCase
}

// Bind checks the pinned Case is the policy's and prepares it under the canary's Profile with the
// environment's coordinates and the tree's catalog, with no connection. Its Profile name is the
// policy's, whatever the environment names.
func Bind(canary *policy.Policy, environment testpilotdriver.Environment) (*Bound, error) {
	if canary == nil {
		return nil, errors.New("a canary policy is required")
	}
	identity, err := Identity()
	if err != nil {
		return nil, fmt.Errorf("the pinned canary Case has no identity: %w", err)
	}
	if identity != canary.CaseIdentity {
		return nil, fmt.Errorf("the pinned canary Case is %s, the policy's is %s", identity, canary.CaseIdentity)
	}
	source, err := testpilot.DecodeCaseProtoJSON(pinned)
	if err != nil {
		return nil, fmt.Errorf("decode the pinned canary Case: %w", err)
	}
	catalog, err := testpilotdriver.NewWorkflowServiceCatalog()
	if err != nil {
		return nil, fmt.Errorf("build the method catalog: %w", err)
	}
	profile := ProfileSpec(canary, catalog, environment)
	prepared, err := testpilot.Prepare(source, profile)
	if err != nil {
		return nil, fmt.Errorf("prepare the pinned canary Case: %w", err)
	}
	return &Bound{Source: source, CaseIdentity: identity, Profile: profile, Prepared: prepared}, nil
}
