package execution

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

// activityFixture is a controller whose carrier call starts one activity, and the activity
// entrypoint the lowering of an activity script produces: one Finish that completes the attempt.
func activityFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	source, _, policy := fixture(t)
	// An attempt may end in a Temporal failure, so the catalog knows that message beside the
	// fixture's own service.
	descriptors := testsupport.DescriptorClosure(failurepb.File_temporal_api_failure_v1_message_proto)
	descriptors.File = append(descriptors.File, &descriptorpb.FileDescriptorProto{Name: proto.String("admission.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("text"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}}}},
		Service:     []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Service"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Call"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload")}}}},
	})
	catalog, err := ir.NewCatalog(descriptors)
	require.NoError(t, err)
	policy.CatalogIdentity = catalog.Identity()
	policy.EnvironmentBindings = append(policy.EnvironmentBindings, contract.EnvironmentBinding{ID: "namespace", Value: "namespace"}, contract.EnvironmentBinding{ID: "queue", Value: "queue"})
	policy.Roles[0].ReservationCarriers = []contract.ReservationCarrierPolicy{{Method: "/example.Service/Call", Shapes: []contract.ReservationCarrierShape{{Kind: contract.ActivityEntrypoint, MaximumCount: 1}}}}
	source.Program.Roles = append(source.Program.Roles, &testpilotspb.Role{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"}, &testpilotspb.Role{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "queue"})
	source.Program.Entrypoints = append(source.Program.Entrypoints, &testpilotspb.Entrypoint{
		EntrypointId: "activity",
		Activation:   &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{ActivityType: "activity-type", WorkerRoleId: "worker", TaskQueueRoleId: "queue", AttemptNumbering: &testpilotspb.AttemptNumbering{First: 1, OneRun: true}}},
		Instructions: []*testpilotspb.InstructionNode{activityNode("run-attempt", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: textLiteral("done")}}})},
	})
	return source, catalog, policy
}

// failing is the instruction that ends its attempt with the failure.
func failing(id string, failure *failurepb.Failure) *testpilotspb.InstructionNode {
	return activityNode(id, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptFailure{ActivityAttemptFailure: &testpilotspb.ActivityAttemptFailure{Failure: failure}}})
}

// finishing is a Finish whose result is the message, carried whole.
func finishing(t *testing.T, id string, message proto.Message) *testpilotspb.InstructionNode {
	t.Helper()
	carried, err := anypb.New(message)
	require.NoError(t, err)
	return activityNode(id, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: carried}}}}}}})
}

func retryableFailure() *failurepb.Failure {
	return &failurepb.Failure{Message: "not yet", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "transient"}}}
}

// retried makes the activity script two attempts: the first fails retryably and the second
// completes.
func retried(source *testpilotspb.Case, policy *Profile) {
	activity := source.Program.Entrypoints[1]
	activity.Instructions = []*testpilotspb.InstructionNode{failing("first-attempt", retryableFailure()), activity.Instructions[0]}
	policy.Opcodes = append(policy.Opcodes, contract.ActivityAttemptFailure)
}

func activityNode(id string, instruction *testpilotspb.Instruction) *testpilotspb.InstructionNode {
	node := rpcNode(id)
	node.Instruction = instruction
	return node
}

// An activity entrypoint that carries no script asks for nothing, so a Profile whose carriers admit
// no activity activation still admits it, reserved by no one.
func TestPrepareAdmitsAnActivityEntrypointWithoutAScriptUncarried(t *testing.T) {
	source, catalog, policy := activityFixture(t)
	source.Program.Entrypoints[1].Instructions = nil
	// It declares no attempt, so even a carrier that admits activities reserves nothing for it.
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	_, carried := prepared.ReservationCarrier("controller", "call")
	require.False(t, carried)

	policy.Roles[0].ReservationCarriers = nil
	prepared, err = Prepare(source, catalog, policy)
	require.NoError(t, err)
	_, carried = prepared.ReservationCarrier("controller", "call")
	require.False(t, carried)
}

// Only an activity needs its carrier to be admitted: a workflow or Nexus-handler script no carrier
// reserves prepares as it did, and is refused, if at all, by the Driver that would deliver it.
func TestPrepareStillAdmitsAnUncarriedWorkflowScript(t *testing.T) {
	source, catalog, policy := handleFixture(t)
	policy.Roles[0].ReservationCarriers = nil
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	_, carried := prepared.ReservationCarrier("controller", "call")
	require.False(t, carried)
}

func TestPrepareAdmitsAnActivityScript(t *testing.T) {
	source, catalog, policy := activityFixture(t)
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)

	plan, carried := prepared.ReservationCarrier("controller", "call")
	require.True(t, carried)
	require.Equal(t, contract.ReservationCarrierPlan{
		EndpointRoleID: "endpoint",
		Method:         "/example.Service/Call",
		Reservations:   []contract.ReservationTopology{{EntrypointID: "activity", Kind: contract.ActivityEntrypoint, Count: 1}},
	}, plan)

	activity := prepared.Entrypoints()[1]
	require.Equal(t, contract.ActivityEntrypoint, activity.Kind())
	require.Equal(t, []int{0}, activity.Order())
	finish := activity.Instructions()[0]
	require.Equal(t, contract.Finish, finish.Opcode())
	result, enabled, _, err := finish.EvaluateInput(context.Background(), func(ir.Reference) *testpilotspb.Value { return nil }, activity.RuntimeWorkLimit())
	require.NoError(t, err)
	require.True(t, enabled)
	require.True(t, proto.Equal(textValue("done"), result))
}

// An activity entrypoint's attempts are judged by how the Case declares the server numbers them, so
// one that declares no numbering, or a first attempt number below 1, is refused at the entrypoint.
func TestPrepareRejectsAnActivityEntrypointThatNumbersNoAttempts(t *testing.T) {
	for name, numbering := range map[string]*testpilotspb.AttemptNumbering{
		"no numbering":    nil,
		"numbered from 0": {OneRun: true},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := activityFixture(t)
			source.Program.Entrypoints[1].GetActivity().AttemptNumbering = numbering
			_, err := Prepare(source, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, ir.Error{Category: ir.Malformed, Path: "activity",
				Detail: "an activity entrypoint requires the numbering of its attempts, from a positive first"}, *diagnostic)
		})
	}
}

// The script declares the activity's attempts, one per instruction, and the carrier reserves an
// activation for each, within what the Profile's carrier admits.
func TestPrepareReservesOneActivationPerDeclaredAttempt(t *testing.T) {
	source, catalog, policy := activityFixture(t)
	retried(source, &policy)
	_, err := Prepare(source, catalog, policy)
	var rejected *ir.Error
	require.ErrorAs(t, err, &rejected)
	require.Equal(t, &ir.Error{Category: ir.Unsupported, Path: "controller.call", Detail: "carrier reservation shape or cardinality is unauthorized"}, rejected)

	policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = 2
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	plan, carried := prepared.ReservationCarrier("controller", "call")
	require.True(t, carried)
	require.Equal(t, contract.ReservationCarrierPlan{
		EndpointRoleID: "endpoint",
		Method:         "/example.Service/Call",
		Reservations:   []contract.ReservationTopology{{EntrypointID: "activity", Kind: contract.ActivityEntrypoint, Count: 2}},
	}, plan)
	require.Equal(t, []int{0, 1}, prepared.Entrypoints()[1].Order())
}

// An attempt fails only through the instruction that says so, with an application failure the SDK
// can carry: any other failure rejects at preparation, naming the field, and so does the instruction
// anywhere but an activity entrypoint or under a Profile that does not authorize it.
func TestPrepareAdmitsAnAttemptFailureOnlyAsAnApplicationFailureOfAnActivity(t *testing.T) {
	const instruction = "program.entrypoints[activity].instructions[run-attempt].instruction.activity_attempt_failure"
	unsupported := func(path string) *ir.Error {
		return &ir.Error{Category: ir.Unsupported, Path: path, Detail: "unsupported instruction context or Driver capability"}
	}
	for name, test := range map[string]struct {
		failure *failurepb.Failure
		mutate  func(*testpilotspb.Case, *Profile)
		want    *ir.Error
	}{
		"a retryable application failure":     {failure: retryableFailure()},
		"a non-retryable application failure": {failure: &failurepb.Failure{Message: "refused", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "refusal", NonRetryable: true}}}},
		"a failure with no info":              {failure: &failurepb.Failure{Message: "failed"}},
		"no failure": {
			want: &ir.Error{Category: ir.Malformed, Path: instruction + ".failure", Detail: "an attempt failure carries the failure"},
		},
		"a timeout": {
			failure: &failurepb.Failure{Message: "late", FailureInfo: &failurepb.Failure_TimeoutFailureInfo{TimeoutFailureInfo: &failurepb.TimeoutFailureInfo{}}},
			want:    &ir.Error{Category: ir.Unsupported, Path: instruction + ".failure.timeout_failure_info", Detail: "an activity attempt fails with an application failure"},
		},
		"a cancellation": {
			failure: &failurepb.Failure{Message: "stopped", FailureInfo: &failurepb.Failure_CanceledFailureInfo{CanceledFailureInfo: &failurepb.CanceledFailureInfo{}}},
			want:    &ir.Error{Category: ir.Unsupported, Path: instruction + ".failure.canceled_failure_info", Detail: "an activity attempt fails with an application failure"},
		},
		"encoded attributes": {
			failure: &failurepb.Failure{Message: "opaque", EncodedAttributes: &commonpb.Payload{Data: []byte("x")}},
			want:    &ir.Error{Category: ir.Unsupported, Path: instruction + ".failure.encoded_attributes", Detail: "field the Driver cannot set through the SDK"},
		},
		"a Profile without the capability": {
			failure: retryableFailure(),
			mutate: func(_ *testpilotspb.Case, policy *Profile) {
				policy.Opcodes = policy.Opcodes[:len(policy.Opcodes)-1]
			},
			want: &ir.Error{Category: ir.Unsupported, Path: "activity.run-attempt", Detail: "instruction activity_attempt_failure the Profile does not authorize"},
		},
		"in a workflow": {
			failure: retryableFailure(),
			mutate: func(source *testpilotspb.Case, policy *Profile) {
				policy.Roles[0].ReservationCarriers[0].Shapes[0].Kind = contract.WorkflowEntrypoint
				source.Program.Entrypoints[1].Activation = &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
			},
			want: unsupported("activity.run-attempt"),
		},
		"in a Nexus handler": {
			failure: retryableFailure(),
			mutate: func(source *testpilotspb.Case, _ *Profile) {
				source.Program.Entrypoints[1].Activation = &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
			},
			want: unsupported("activity.run-attempt"),
		},
		"in a controller": {
			failure: retryableFailure(),
			mutate: func(source *testpilotspb.Case, _ *Profile) {
				node := failing("fail", retryableFailure())
				node.After = runsAfter("controller")
				source.Program.Entrypoints[0].Instructions = append(source.Program.Entrypoints[0].Instructions, node)
			},
			want: unsupported("controller.fail"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := activityFixture(t)
			policy.Opcodes = append(policy.Opcodes, contract.ActivityAttemptFailure)
			source.Program.Entrypoints[1].Instructions[0] = failing("run-attempt", test.failure)
			if test.mutate != nil {
				test.mutate(source, &policy)
			}
			_, err := Prepare(source, catalog, policy)
			if test.want == nil {
				require.NoError(t, err)
				return
			}
			var rejected *ir.Error
			require.ErrorAs(t, err, &rejected)
			require.Equal(t, test.want, rejected)
		})
	}
}

// A Finish completes its attempt with whatever its result is. A result that happens to be a
// Temporal failure message, of any kind, is a value the activity completes with, never a failed
// attempt: failing is the other instruction's.
func TestActivityFinishCompletesWithAFailureMessageAsItsResult(t *testing.T) {
	for name, failure := range map[string]*failurepb.Failure{
		"an application failure": retryableFailure(),
		"a timeout":              {Message: "late", FailureInfo: &failurepb.Failure_TimeoutFailureInfo{TimeoutFailureInfo: &failurepb.TimeoutFailureInfo{}}},
		"encoded attributes":     {Message: "opaque", EncodedAttributes: &commonpb.Payload{Data: []byte("x")}},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := activityFixture(t)
			source.Program.Entrypoints[1].Instructions[0] = finishing(t, "run-attempt", failure)
			prepared, err := Prepare(source, catalog, policy)
			require.NoError(t, err)
			require.Equal(t, contract.Finish, prepared.Entrypoints()[1].Instructions()[0].Opcode())
		})
	}
}

// A workflow's Finish completes the workflow with whatever its result is, as it always did, a
// failure message included.
func TestWorkflowFinishStillCompletesWithAFailureValue(t *testing.T) {
	source, catalog, policy := activityFixture(t)
	policy.Roles[0].ReservationCarriers[0].Shapes[0].Kind = contract.WorkflowEntrypoint
	finish := finishing(t, "finish", &failurepb.Failure{Message: "late", FailureInfo: &failurepb.Failure_TimeoutFailureInfo{TimeoutFailureInfo: &failurepb.TimeoutFailureInfo{}}})
	source.Program.Entrypoints[1] = &testpilotspb.Entrypoint{
		EntrypointId: "workflow",
		Activation:   &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}},
		Instructions: []*testpilotspb.InstructionNode{finish},
	}
	_, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
}

// An activity attempt reports as every worker instruction does: a status, an SDK failure code and a
// detail, and no protocol code, which only a controller's protocol effect carries.
func TestActivityFinishOutcomeIsAWorkerOutcome(t *testing.T) {
	source, catalog, policy := activityFixture(t)
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	activity := prepared.Entrypoints()[1]
	finish := activity.Instructions()[0]
	for name, test := range map[string]struct {
		outcome  *testpilotspb.InstructionOutcome
		admitted bool
	}{
		"succeeded":     {&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, true},
		"sdk failure":   {&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, SdkFailureCode: "sdk_failure", Detail: "failed"}, true},
		"canceled":      {&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED, SdkFailureCode: "canceled"}, true},
		"protocol code": {&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, ProtocolCode: "unavailable"}, false},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := finish.ValidateOutcome(context.Background(), test.outcome, activity.RuntimeWorkLimit())
			if test.admitted {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}

func TestPrepareRejectsAnActivityItCannotActivate(t *testing.T) {
	unsupported := func(instruction string) *ir.Error {
		return &ir.Error{Category: ir.Unsupported, Path: "activity." + instruction, Detail: "unsupported instruction context or Driver capability"}
	}
	uncarried := &ir.Error{Category: ir.Unavailable, Path: "activity", Detail: "no reservation carrier of the Profile activates the activity entrypoint"}
	for name, test := range map[string]struct {
		mutate func(*testpilotspb.Case, *Profile)
		want   *ir.Error
	}{
		"the Profile lacks the Finish capability": {
			mutate: func(_ *testpilotspb.Case, policy *Profile) {
				policy.Opcodes = []contract.Opcode{contract.InvokeRPC}
			},
			want: &ir.Error{Category: ir.Unsupported, Path: "activity.run-attempt", Detail: "instruction finish the Profile does not authorize"},
		},
		"the Profile names no carrier": {
			mutate: func(_ *testpilotspb.Case, policy *Profile) { policy.Roles[0].ReservationCarriers = nil },
			want:   uncarried,
		},
		"the carrier admits no activity activation": {
			mutate: func(_ *testpilotspb.Case, policy *Profile) {
				policy.Roles[0].ReservationCarriers[0].Shapes[0].Kind = contract.WorkflowEntrypoint
			},
			want: uncarried,
		},
		"only the cleanup invokes the carrier": {
			mutate: func(source *testpilotspb.Case, _ *Profile) {
				source.Program.Cleanup.Instructions = source.Program.Entrypoints[0].Instructions
				source.Program.Entrypoints[0].Instructions = nil
			},
			want: uncarried,
		},
		"a controller activation is no carrier shape": {
			mutate: func(_ *testpilotspb.Case, policy *Profile) {
				policy.Roles[0].ReservationCarriers[0].Shapes[0].Kind = contract.ControllerEntrypoint
			},
			want: &ir.Error{Category: ir.Unsupported, Path: "policy.reservation_carriers", Detail: "carrier shape has an unsupported activation context"},
		},
		"an RPC": {
			mutate: func(source *testpilotspb.Case, _ *Profile) {
				source.Program.Entrypoints[1].Instructions[0] = rpcNode("run-attempt")
			},
			want: unsupported("run-attempt"),
		},
		"a workflow command": {
			mutate: func(source *testpilotspb.Case, _ *Profile) {
				source.Program.Entrypoints[1].Instructions[0] = scheduleNode("run-attempt")
			},
			want: unsupported("run-attempt"),
		},
		"a Nexus handler reply": {
			mutate: func(source *testpilotspb.Case, _ *Profile) {
				source.Program.Entrypoints[1].Instructions[0] = replyNode("run-attempt", syncReply())
			},
			want: unsupported("run-attempt"),
		},
		"an await of an instruction": {
			mutate: func(source *testpilotspb.Case, _ *Profile) {
				source.Program.Entrypoints[1].Instructions[0] = activityNode("run-attempt", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: &testpilotspb.InstructionReference{EntrypointId: "activity", InstructionId: "run-attempt"}}}})
			},
			want: unsupported("run-attempt"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := activityFixture(t)
			test.mutate(source, &policy)
			_, err := Prepare(source, catalog, policy)
			var rejected *ir.Error
			require.ErrorAs(t, err, &rejected)
			require.Equal(t, test.want, rejected)
		})
	}
}

// Admitting a Finish in an activity entrypoint admits it nowhere else new: a controller and a Nexus
// handler still run none.
func TestFinishStillRunsOnlyWhereAScriptEnds(t *testing.T) {
	finish := func() *testpilotspb.InstructionNode {
		return activityNode("finish", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: textLiteral("done")}}})
	}
	for name, test := range map[string]struct {
		mutate func(*testpilotspb.Case)
		path   string
	}{
		"a controller": {
			mutate: func(source *testpilotspb.Case) {
				node := finish()
				node.After = runsAfter("controller")
				source.Program.Entrypoints[0].Instructions = append(source.Program.Entrypoints[0].Instructions, node)
			},
			path: "controller.finish",
		},
		"a Nexus handler": {
			mutate: func(source *testpilotspb.Case) {
				source.Program.Entrypoints[1].Activation = &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				source.Program.Entrypoints[1].Instructions = []*testpilotspb.InstructionNode{finish()}
			},
			path: "activity.finish",
		},
		"the cleanup": {
			mutate: func(source *testpilotspb.Case) {
				source.Program.Cleanup.Instructions = []*testpilotspb.InstructionNode{finish()}
			},
			path: "cleanup.finish",
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := activityFixture(t)
			test.mutate(source)
			_, err := Prepare(source, catalog, policy)
			var rejected *ir.Error
			require.ErrorAs(t, err, &rejected)
			require.Equal(t, &ir.Error{Category: ir.Unsupported, Path: test.path, Detail: "unsupported instruction context or Driver capability"}, rejected)
		})
	}
}

// attempted is the reservation outcome of one activity attempt with its typed identities and what
// the worker offered Temporal.
func attempted(status testpilotspb.InstructionOutcomeStatus, attempt int32, deliveryID string, response testpilotspb.ActivityAttemptResponse) *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{Status: status, ActivityAttempt: &testpilotspb.ActivityAttempt{ActivityRunId: "activity-run", SdkAttempt: attempt, DeliveryId: deliveryID, Response: response}}
}

// inRun is the outcome naming another activity run.
func inRun(activityRunID string, outcome *testpilotspb.InstructionOutcome) *testpilotspb.InstructionOutcome {
	crossed := proto.CloneOf(outcome)
	crossed.ActivityAttempt.ActivityRunId = activityRunID
	return crossed
}

// recordedAttempt is one reservation event of a Run, whole but for what the clock decides.
type recordedAttempt struct {
	Source  string
	Causes  []string
	Outcome string
}

// What each attempt of an activity was reaches the Run as a typed fact, in attempt order however
// the reservations settle: the reservation's event, under the attempt's ordinal and the
// coordinates of the instruction that carried the start, carries the activity run, the SDK
// attempt, the delivery and the answer the worker offered. A position no attempt was delivered
// for, because the activity closed after an earlier attempt, is recorded as that, names no SDK
// attempt, and is caused by the attempt before it as well as by the start. An attempt the worker
// refused is recorded too, and then the Run is incomplete: the record is never replaced by the
// failure it caused. An outcome no Driver may report fails the Run and is not recorded.
func TestRunRecordsEachActivityAttemptAsATypedFact(t *testing.T) {
	succeeded, failed, canceled := testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED
	retryable := attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE)
	refused := attempted(failed, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED)
	refused.SdkFailureCode, refused.Detail = "umpire_worker", "context canceled"
	notNeeded := attempted(canceled, 0, "", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED)
	const started, first, second, third = "scheduler.g0.n0.a1.started", "scheduler.g0.n0.a1.r0.i0", "scheduler.g0.n0.a1.r0.i1", "scheduler.g0.n0.a1.r0.i2"
	fact := func(source string, outcome *testpilotspb.InstructionOutcome, causes ...string) recordedAttempt {
		return recordedAttempt{Source: source, Causes: append([]string{started}, causes...), Outcome: outcome.String()}
	}
	for name, test := range map[string]struct {
		outcomes   []*testpilotspb.InstructionOutcome
		recorded   []recordedAttempt
		incomplete string
	}{
		"a failed attempt and its retry": {
			outcomes: []*testpilotspb.InstructionOutcome{retryable, attempted(succeeded, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)},
			recorded: []recordedAttempt{fact(first, retryable), fact(second, attempted(succeeded, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED))},
		},
		"an attempt that ends the activity and the positions no longer needed": {
			outcomes: []*testpilotspb.InstructionOutcome{attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_NON_RETRYABLE), notNeeded, notNeeded},
			recorded: []recordedAttempt{
				fact(first, attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_NON_RETRYABLE)),
				fact(second, notNeeded, first), fact(third, notNeeded, first),
			},
		},
		// The worker's completion never reached the server, which then issued the next attempt.
		"an offered completion and the attempt after it": {
			outcomes: []*testpilotspb.InstructionOutcome{attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED), attempted(succeeded, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED), notNeeded},
			recorded: []recordedAttempt{
				fact(first, attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)),
				fact(second, attempted(succeeded, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)),
				fact(third, notNeeded, second),
			},
		},
		// The worker's canceled answer is an offer like any other: it is recorded, and the positions
		// after it are not needed only because the server then reported the activity closed.
		"an offered cancellation and the position no longer needed": {
			outcomes: []*testpilotspb.InstructionOutcome{attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_CANCELED), notNeeded},
			recorded: []recordedAttempt{
				fact(first, attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_CANCELED)),
				fact(second, notNeeded, first),
			},
		},
		"a canceled attempt that claims it offered a cancellation": {
			outcomes:   []*testpilotspb.InstructionOutcome{retryable, attempted(canceled, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_CANCELED)},
			recorded:   []recordedAttempt{fact(first, retryable)},
			incomplete: "activation_failed",
		},
		"an attempt the worker refused": {
			outcomes:   []*testpilotspb.InstructionOutcome{retryable, refused},
			recorded:   []recordedAttempt{fact(first, retryable), fact(second, refused)},
			incomplete: "activation_failed",
		},
		// A reservation canceled with no attempt fact is one the Run released, not one the Driver
		// found unneeded: it fails the activation as it always did, and records nothing.
		"an attempt never delivered": {
			outcomes:   []*testpilotspb.InstructionOutcome{retryable, {Status: canceled}},
			recorded:   []recordedAttempt{fact(first, retryable)},
			incomplete: "activation_failed",
		},
		"a failed attempt that claims it was not needed": {
			outcomes:   []*testpilotspb.InstructionOutcome{retryable, attempted(failed, 0, "", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED)},
			recorded:   []recordedAttempt{fact(first, retryable)},
			incomplete: "activation_failed",
		},
		"a canceled attempt that claims a completion": {
			outcomes:   []*testpilotspb.InstructionOutcome{retryable, attempted(canceled, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)},
			recorded:   []recordedAttempt{fact(first, retryable)},
			incomplete: "activation_failed",
		},
		"an attempt that names no response": {
			outcomes:   []*testpilotspb.InstructionOutcome{retryable, attempted(succeeded, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_UNSPECIFIED)},
			recorded:   []recordedAttempt{fact(first, retryable)},
			incomplete: "activation_failed",
		},
		"an attempt under another position's number": {
			outcomes:   []*testpilotspb.InstructionOutcome{retryable, attempted(succeeded, 3, "delivery-3", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)},
			recorded:   []recordedAttempt{fact(first, retryable)},
			incomplete: "activation_failed",
		},
		// Every attempt of one activity names the activity run the first recorded attempt named.
		"a retry that names another activity run": {
			outcomes:   []*testpilotspb.InstructionOutcome{retryable, inRun("another-run", attempted(succeeded, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED))},
			recorded:   []recordedAttempt{fact(first, retryable)},
			incomplete: "activation_failed",
		},
		"a position not needed in another activity run": {
			outcomes:   []*testpilotspb.InstructionOutcome{attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED), inRun("another-run", notNeeded)},
			recorded:   []recordedAttempt{fact(first, attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED))},
			incomplete: "activation_failed",
		},
		"a first position not needed": {
			outcomes:   []*testpilotspb.InstructionOutcome{notNeeded, notNeeded},
			incomplete: "activation_failed",
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := activityFixture(t)
			retried(source, &policy)
			if len(test.outcomes) == 3 {
				script := source.Program.Entrypoints[1]
				script.Instructions = append(script.Instructions, finishing(t, "third-attempt", &failurepb.Failure{Message: "late"}))
			}
			policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = int64(len(test.outcomes))
			prepared, err := Prepare(source, catalog, policy)
			require.NoError(t, err)
			var requests []contract.ReservationRequest
			// The first attempt settles last, once the start itself is recorded complete, and every
			// later reservation is settled from the outset: the Run still records them in order.
			startCompleted := make(chan struct{})
			monitor := schedulerMonitor{observe: func(event *testpilotspb.RunEvent) Decision {
				if event.GetSourceId() == "scheduler.g0.n0.a1.completed" {
					close(startCompleted)
				}
				return Continue
			}}
			host := &testsupport.Session{}
			host.OnReserve = func(_ context.Context, request contract.ReservationRequest) ([]contract.ReservationHandle, error) {
				requests = append(requests, request)
				var handles []contract.ReservationHandle
				// The Driver hands the reservations back in no particular order.
				for ordinal := len(test.outcomes) - 1; ordinal >= 0; ordinal-- {
					outcome := test.outcomes[ordinal]
					handles = append(handles, &testsupport.Reservation{
						ID:         contract.ReservationIdentity{Origin: request.Origin, EntrypointID: request.EntrypointID, Ordinal: int64(ordinal), ID: fmt.Sprintf("reservation-%d", ordinal+1)},
						Activation: contract.Coordinate{RunID: request.Origin.RunID, EntrypointID: request.EntrypointID, ActivationID: fmt.Sprintf("reservation-%d", ordinal+1), Attempt: request.Origin.Attempt},
						Effect: &testsupport.Effect{OnWait: func(ctx context.Context) (contract.EffectResult, error) {
							if ordinal != 0 {
								return contract.EffectResult{Outcome: outcome}, nil
							}
							select {
							case <-ctx.Done():
								return contract.EffectResult{}, ctx.Err()
							case <-startCompleted:
								return contract.EffectResult{Outcome: outcome}, nil
							}
						}},
					})
				}
				return handles, nil
			}
			host.OnInvokeRPC = func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) { return effectResponse(prepared, "ok"), nil }}, nil
			}
			s, err := newScheduler(prepared, "run", "case", host, monitor, time.Now)
			require.NoError(t, err)
			err = s.execute(context.Background())

			origin := contract.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "call", Attempt: 1}
			require.Equal(t, []contract.ReservationRequest{{Origin: origin, EntrypointID: "activity", Count: int64(len(test.outcomes))}}, requests)
			var recorded []recordedAttempt
			for _, event := range s.recorder.run.Events {
				if event.Kind == testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC && event.GetOutcome() != nil {
					require.True(t, proto.Equal(eventCoordinates(origin), event.Coordinates), event.Coordinates)
					recorded = append(recorded, recordedAttempt{Source: event.SourceId, Causes: event.CausalSourceIds, Outcome: event.GetOutcome().String()})
				}
			}
			require.Equal(t, test.recorded, recorded)
			failures := diagnosticCodes(s.recorder.run)
			if test.incomplete == "" {
				require.NoError(t, err)
				require.False(t, s.recorder.incomplete)
				require.Empty(t, failures)
				return
			}
			require.Error(t, err)
			require.True(t, s.recorder.incomplete)
			require.Equal(t, []string{test.incomplete}, failures)
		})
	}
}

// The kind the outcome is judged under is the reserved entrypoint's own: a canceled reservation of
// a workflow that performs nothing is recorded, and the same reservation claiming to be an activity
// position that was not needed fails the Run.
func TestRunRejectsAnActivityAttemptReportedForAnotherKindOfEntrypoint(t *testing.T) {
	for name, test := range map[string]struct {
		outcome  *testpilotspb.InstructionOutcome
		recorded bool
	}{
		"a released workflow reservation":      {&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED}, true},
		"one that claims an activity position": {attempted(testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED, 0, "", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED), false},
	} {
		t.Run(name, func(t *testing.T) {
			c, catalog, policy := fixture(t)
			addWorker(c, &policy)
			second := proto.CloneOf(c.Program.Entrypoints[1])
			second.EntrypointId = "workflow_second"
			c.Program.Entrypoints = append(c.Program.Entrypoints, second)
			p, err := Prepare(c, catalog, policy)
			require.NoError(t, err)
			host := &testsupport.Session{}
			host.OnReserve = func(_ context.Context, r contract.ReservationRequest) ([]contract.ReservationHandle, error) {
				return []contract.ReservationHandle{&testsupport.Reservation{ID: contract.ReservationIdentity{Origin: r.Origin, EntrypointID: r.EntrypointID, ID: "reservation." + r.EntrypointID}, Activation: contract.Coordinate{RunID: r.Origin.RunID, EntrypointID: r.EntrypointID, ActivationID: "actual-" + r.EntrypointID}, Effect: &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
				}}}}, nil
			}
			host.OnInvokeRPC = func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) { return effectResponse(p, "ok"), nil }}, nil
			}
			s, err := newScheduler(p, "run", "case", host, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			require.NoError(t, s.execute(context.Background()))
			origin := contract.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller.0", InstructionID: "call", Attempt: 1}
			// An attempt of the same carrier is already recorded, so only the kind can reject.
			s.attemptFacts["scheduler.g0.n0.a1.r1"] = priorAttempt{source: "scheduler.g0.n0.a1.r1.i0", activityRunID: "activity-run"}
			decision, err := s.publishCompletion(context.Background(), schedulerCompletion{
				reservation: &scheduledReservation{identity: contract.ReservationIdentity{Origin: origin, EntrypointID: "workflow_second", Ordinal: 1, ID: "reservation.workflow_second"}, source: "scheduler.g0.n0.a1.r1.i1", cause: "scheduler.g0.n0.a1.started"},
				result:      contract.EffectResult{Outcome: test.outcome},
			})
			if test.recorded {
				require.NoError(t, err)
				require.Equal(t, Continue, decision)
				return
			}
			require.Error(t, err)
			require.Equal(t, Stop, decision)
			require.Equal(t, []string{"activation_failed"}, diagnosticCodes(s.recorder.run))
		})
	}
}
