package execution_test

import (
	"context"
	"fmt"
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// The SDK transport tests establish invocation. This fixture establishes that recording its typed
// local outcome and replaying it offline preserve the Contract's conclusion and bounded cleanup.
func TestActivityHeartbeatGroupRecordReplaysItsVerdictAndSupport(t *testing.T) {
	const startMethod = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution"
	for _, pending := range []bool{false, true} {
		t.Run(fmt.Sprintf("pending=%t", pending), func(t *testing.T) {
			prepared := facadetest.RuntimeCase(t, facadetest.SyncReply, nil, func(profile *testpilot.ProfileSpec) {
				profile.Opcodes = append(profile.Opcodes, testpilot.ActivityHeartbeat, testpilot.ActivityAttemptWithholding)
				profile.Roles[0].Methods = []string{startMethod}
				profile.Roles[0].ReservationCarriers = []testpilot.ReservationCarrierPolicy{{Method: startMethod, Shapes: []testpilot.ReservationCarrierShape{{Kind: testpilot.ActivityEntrypoint, MaximumCount: 2}}}}
			}, func(program *testpilotspb.Program) {
				controller := program.Entrypoints[0]
				invoke := controller.Instructions[0].Instruction.GetInvokeRpc()
				invoke.Method = startMethod
				invoke.RequestAssignments = append(invoke.RequestAssignments,
					&testpilotspb.RequestAssignment{Target: "activity_id", Value: facadetest.Text("activity")},
					&testpilotspb.RequestAssignment{Target: "activity_type.name", Value: facadetest.Text("activity-type")},
					&testpilotspb.RequestAssignment{Target: "heartbeat_timeout.seconds", Value: cel.Literal(&celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: 1}})})
				entry := &testpilotspb.Entrypoint{EntrypointId: "activity", Activation: &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue", AttemptNumbering: &testpilotspb.AttemptNumbering{First: 1, OneRun: true}}}}
				entry.Instructions = append(entry.Instructions, &testpilotspb.InstructionNode{InstructionId: "heartbeat", Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityHeartbeat{ActivityHeartbeat: &testpilotspb.ActivityHeartbeat{Details: &commonpb.Payloads{Payloads: []*commonpb.Payload{facadetest.Payload("beat")}}}}}})
				if pending {
					entry.Instructions = append(entry.Instructions, &testpilotspb.InstructionNode{InstructionId: "pending", Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptWithholding{ActivityAttemptWithholding: &testpilotspb.ActivityAttemptWithholding{Mode: testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING}}}})
				}
				entry.Instructions = append(entry.Instructions, &testpilotspb.InstructionNode{InstructionId: "finish", Limits: facadetest.Bounds(), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: facadetest.Text("done")}}}})
				program.Entrypoints = []*testpilotspb.Entrypoint{controller, entry}
			})
			driver := &facadetest.Driver{DriverIdentity: prepared.Identity(), OnOpen: func(_ context.Context, _ string, _ testpilot.PreparedProgram) (testpilot.Session, error) {
				return &testsupport.Session{
					OnReserve: func(_ context.Context, request testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
						var handles []testpilot.ReservationHandle
						for ordinal := int64(0); ordinal < request.Count; ordinal++ {
							response := testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED
							if pending && ordinal == 0 {
								response = testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING
							}
							outcome := &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ActivityAttempt: &testpilotspb.ActivityAttempt{ActivityRunId: "activity-run", SdkAttempt: int32(ordinal + 1), DeliveryId: fmt.Sprintf("delivery-%d", ordinal+1), Response: response, HeartbeatInvoked: ordinal == 0}}
							handles = append(handles, &testsupport.Reservation{ID: testpilot.ReservationIdentity{Origin: request.Origin, EntrypointID: request.EntrypointID, Ordinal: ordinal, ID: fmt.Sprintf("reservation-%d", ordinal)}, Activation: testpilot.Coordinate{RunID: request.Origin.RunID, EntrypointID: request.EntrypointID, ActivationID: fmt.Sprintf("reservation-%d", ordinal), Attempt: request.Origin.Attempt}, Effect: testsupport.Completed(testpilot.EffectResult{Outcome: outcome})})
						}
						return handles, nil
					},
					OnInvokeRPC: func(_ context.Context, _ testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message) (testpilot.EffectHandle, error) {
						return testsupport.Completed(testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: dynamicpb.NewMessage(method.Output())}), nil
					},
					OnClose: func(ctx context.Context) error {
						_, bounded := ctx.Deadline()
						require.True(t, bounded, "cleanup must retain its executor deadline")
						return ctx.Err()
					},
				}, nil
			}}
			run, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
			require.Len(t, verdict.GetRules(), 1)
			require.Equal(t, "closed", verdict.GetRules()[0].GetTerminalStateId())
			require.Equal(t, []int64{run.GetEvents()[len(run.GetEvents())-1].GetSequence()}, verdict.GetSupportingEventSequences())
			require.Equal(t, testpilotspb.CLEANUP_STATUS_SUCCEEDED, run.GetCleanup().GetStatus())
			require.Equal(t, 1, driver.Closed())
			var attempts []*testpilotspb.ActivityAttempt
			for _, event := range run.GetEvents() {
				if attempt := event.GetOutcome().GetActivityAttempt(); attempt != nil {
					attempts = append(attempts, attempt)
				}
			}
			expected := 1
			if pending {
				expected = 2
			}
			require.Len(t, attempts, expected)
			require.True(t, attempts[0].GetHeartbeatInvoked())
			if pending {
				require.Equal(t, testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING, attempts[0].GetResponse())
				require.Equal(t, int32(2), attempts[1].GetSdkAttempt())
				require.Equal(t, "delivery-2", attempts[1].GetDeliveryId())
				require.False(t, attempts[1].GetHeartbeatInvoked())
			}
			encoded, err := proto.Marshal(run)
			require.NoError(t, err)
			recorded := &testpilotspb.Run{}
			require.NoError(t, proto.Unmarshal(encoded, recorded))
			protorequire.ProtoEqual(t, run, recorded)
			replayed, evaluation, err := prepared.Evaluate(t.Context(), recorded)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, verdict, replayed)
			protorequire.ProtoEqual(t, run.GetVerdict(), replayed)
			require.Empty(t, evaluation.Violations)
		})
	}
}
