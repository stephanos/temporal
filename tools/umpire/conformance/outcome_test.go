package conformance

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/check"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestPerformedOutcomesAreNotTemporalCheckedWithoutRejectionMetadata(t *testing.T) {
	realization := &umpirespb.Realization{Scripts: []*umpirespb.Script{{Items: []*umpirespb.Item{{Performs: []*umpirespb.Performance{{
		Step: &umpirespb.ActionClass{}, Command: &umpirespb.Command{Instruction: &umpirespb.Command_Rpc{Rpc: &umpirespb.Rpc{}}},
	}}}}}}}
	performed, err := compilePerformedOutcomes(nil, check.ClaimKey{}, realization, nil, nil)
	require.NoError(t, err)
	require.Empty(t, performed)
}

// A repeated terminate reaches the same realized RPC command twice: the first Model step is
// accepted, and the second is rejected because the activity is no longer found. Each Run result is
// checked against its particular Model step, using the code the realization exports.
func TestPerformedStepOutcomeMatchesItsRunInstructionResult(t *testing.T) {
	base := activityModel(t)
	source := proto.CloneOf(loweredActivity(t, proto.CloneOf(base), "terminate").source)
	pauseSource := loweredActivity(t, proto.CloneOf(base), "pauseResume").source
	var pauseInstruction *testpilotspb.InstructionNode
	for _, entrypoint := range pauseSource.GetProgram().GetEntrypoints() {
		if entrypoint.GetEntrypointId() != "controller" {
			continue
		}
		for _, instruction := range entrypoint.GetInstructions() {
			if instruction.GetInstructionId() == "pause-activity" {
				pauseInstruction = proto.CloneOf(instruction)
			}
		}
	}
	require.NotNil(t, pauseInstruction)
	var terminateAgain *testpilotspb.InstructionNode
	for _, entrypoint := range source.GetProgram().GetEntrypoints() {
		if entrypoint.GetEntrypointId() != "controller" {
			continue
		}
		for _, instruction := range entrypoint.GetInstructions() {
			if instruction.GetInstructionId() == "terminate-activity" {
				terminateAgain = proto.CloneOf(instruction)
				terminateAgain.InstructionId = "terminate-activity-2"
			}
		}
		require.NotNil(t, terminateAgain)
		entrypoint.Instructions = append(entrypoint.Instructions, terminateAgain, pauseInstruction)
	}
	m := proto.CloneOf(base)
	var pause *umpirespb.ActionClass
	var terminatedState *umpirespb.Expr
	for _, scenario := range m.GetScenarios() {
		if scenario.GetName() == "pausedThenCompleted" {
			pause = proto.CloneOf(scenario.GetActions()[1])
		}
		if scenario.GetName() == "terminatedWhileScheduled" {
			terminatedState = proto.CloneOf(scenario.GetStart())
			terminatedState.GetConstruct().GetArgs()[0].GetLiteral().GetEnum().Case = "terminated"
		}
	}
	require.NotNil(t, pause)
	require.NotNil(t, terminatedState)
	for _, scenario := range m.GetScenarios() {
		if scenario.GetName() != "terminatedWhileScheduled" {
			continue
		}
		scenario.Actions = append(scenario.Actions, proto.CloneOf(scenario.GetActions()[len(scenario.GetActions())-1]))
		scenario.Actions = append(scenario.Actions, pause)
	}
	for _, query := range m.GetQueries() {
		if query.GetName() == "terminate" {
			query.GetLimits().Steps += 2
			query.GetLimits().Actions += 2
			query.Total = wrapperspb.Int64(1440)
		}
	}
	for _, property := range m.GetProperties() {
		if property.GetMachine() == "activitySystem" && property.GetName() == "terminated" {
			property.When = &umpirespb.Property_WhenClass{WhenClass: pause}
		}
	}
	var rejectsNotFound *umpirespb.Expr
	for _, function := range m.GetFunctions() {
		if function.GetName() == "activityProduct.property.activityProduct.closedIsRejectedUniformly" {
			rejectsNotFound = proto.CloneOf(function.GetBody().GetBinary().GetRight().GetBinary().GetRight())
		}
	}
	require.NotNil(t, rejectsNotFound)
	rejectsNotFound.GetBinary().GetLeft().GetField().Base = &umpirespb.Expr{Kind: &umpirespb.Expr_Var{Var: "s"}}
	stateEquals := &umpirespb.Expr{Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{
		Op: umpirespb.Binary_OP_EQ,
		Left: &umpirespb.Expr{Kind: &umpirespb.Expr_Field{Field: &umpirespb.FieldAccess{
			Base: &umpirespb.Expr{Kind: &umpirespb.Expr_Var{Var: "s"}}, Field: "state",
		}}},
		Right: terminatedState,
	}}}
	for _, function := range m.GetFunctions() {
		if function.GetName() == "activitySystem.property.terminated" {
			function.Body = &umpirespb.Expr{Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{
				Op: umpirespb.Binary_OP_AND, Left: stateEquals, Right: rejectsNotFound,
			}}}
		}
	}
	factory, err := Prepare(m, check.ClaimKey{Family: activityFamily, Owner: activityMachine, Name: "terminate"}, source, generous)
	require.NoError(t, err)

	completed := func(instruction string, status testpilotspb.InstructionOutcomeStatus, code, detail string) *testpilotspb.RunEvent {
		return &testpilotspb.RunEvent{
			Sequence: 7,
			Kind:     testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED,
			Coordinates: &testpilotspb.RunEventCoordinates{
				EntrypointId: "controller", InstructionId: instruction, Attempt: 1,
			},
			Payload: &testpilotspb.RunEvent_Outcome{Outcome: &testpilotspb.InstructionOutcome{
				Status: status, ProtocolCode: code, Detail: detail,
			}},
		}
	}
	tests := map[string]struct {
		event            *testpilotspb.RunEvent
		wantExpectedCode string
		wantObservedCode string
	}{
		"an exact rejection code": {
			event: completed("terminate-activity-2", testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, strings.ToLower(codes.NotFound.String()), "message text is deliberately unrelated"),
		},
		"a different rejection code": {
			event:            completed("terminate-activity-2", testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, "already_exists", "not compared"),
			wantExpectedCode: "NOT_FOUND", wantObservedCode: "ALREADY_EXISTS",
		},
		"an accepted step with an error": {
			event:            completed("terminate-activity", testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, "not_found", "not compared"),
			wantExpectedCode: "OK", wantObservedCode: "NOT_FOUND",
		},
		"a rejected step with OK": {
			event:            completed("terminate-activity-2", testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, "ok", "not compared"),
			wantExpectedCode: "NOT_FOUND", wantObservedCode: "OK",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assessor := newAssessor(factory.plan)
			established, err := assessor.Observe(t.Context(), test.event)
			require.NoError(t, err)
			if test.wantExpectedCode == "" {
				require.Nil(t, established.Nonconformance)
				return
			}
			require.Equal(t, &testpilot.ConformanceAssessment{
				Status:                   testpilot.ConformanceNonconformant,
				SupportingEventSequences: []int64{7},
				Reason:                   "unexplained",
				Detail: "activitySystem, controller/" + test.event.GetCoordinates().GetInstructionId() +
					": expected gRPC code " + test.wantExpectedCode + ", observed " + test.wantObservedCode,
			}, established.Nonconformance)
			require.NotContains(t, established.Nonconformance.Detail, test.event.GetOutcome().GetDetail())
		})
	}

	// The expected code comes from this realization's metadata. Changing that sole source changes
	// the comparison without a Go-side rejection table.
	for _, entry := range realizationNamed(t, m, "standalone").GetRejectionCodes() {
		if entry.GetRejection() == umpirespb.RejectionCode_REJECTION_NOT_FOUND {
			entry.GrpcCode = "RESOURCE_EXHAUSTED"
		}
	}
	factory, err = Prepare(m, check.ClaimKey{Family: activityFamily, Owner: activityMachine, Name: "terminate"}, source, generous)
	require.NoError(t, err)
	assessor := newAssessor(factory.plan)
	established, err := assessor.Observe(t.Context(), completed("terminate-activity-2", testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, "not_found", "ignored"))
	require.NoError(t, err)
	require.Contains(t, established.Nonconformance.Detail, "expected gRPC code RESOURCE_EXHAUSTED, observed NOT_FOUND")
}
