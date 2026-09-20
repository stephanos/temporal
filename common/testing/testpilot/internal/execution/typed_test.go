package execution

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	sdkpb "go.temporal.io/api/sdk/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/durationpb"
)

func scheduleCommand() *commandpb.Command {
	return &commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
		Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
			Endpoint: "endpoint", Service: "service", Operation: "operation",
			Input:                  &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(`"request"`)},
			ScheduleToCloseTimeout: durationpb.New(5 * time.Second),
			NexusHeader:            map[string]string{"x-case": "1"},
		}},
	}
}

func scheduleNode(id string) *testpilotspb.InstructionNode {
	node := rpcNode(id)
	node.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotspb.WorkflowCommand{Command: scheduleCommand()}}}
	return node
}

func replyNode(id string, reply *testpilotspb.NexusHandlerReply) *testpilotspb.InstructionNode {
	node := rpcNode(id)
	node.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusHandlerReply{NexusHandlerReply: reply}}
	return node
}

func asyncReply(slot string) *testpilotspb.NexusHandlerReply {
	return &testpilotspb.NexusHandlerReply{HandleSlotId: slot, Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_AsyncSuccess{AsyncSuccess: &nexuspb.StartOperationResponse_Async{}}}}}
}

func syncReply() *testpilotspb.NexusHandlerReply {
	return &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_SyncSuccess{SyncSuccess: &nexuspb.StartOperationResponse_Sync{Payload: &commonpb.Payload{Data: []byte(`"done"`)}}}}}}
}

func completionNode(id string, completion *testpilotspb.NexusOperationCompletion) *testpilotspb.InstructionNode {
	node := rpcNode(id)
	node.Instruction = &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusOperationCompletion{NexusOperationCompletion: completion}}
	return node
}

func payloadCompletion(slot string) *testpilotspb.NexusOperationCompletion {
	return &testpilotspb.NexusOperationCompletion{HandleSlotId: slot, Result: &testpilotspb.NexusOperationCompletion_Payload{Payload: &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(`"done"`)}}}
}

func TestPrepareAdmitsTypedInstructions(t *testing.T) {
	c, catalog, p := handleFixture(t)
	prepared, err := Prepare(c, catalog, p)
	require.NoError(t, err)
	// The Await of a scheduled command yields the handler's payload whole, as an Any.
	await := prepared.Entrypoints()[1].Instructions()[1]
	value, ok := await.OutcomeType(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)
	require.True(t, ok)
	require.NotNil(t, value.GetSingular().GetAny())
	// A typed completion is a controller protocol effect, so it produces a protocol code.
	_, ok = prepared.Entrypoints()[0].Instructions()[2].OutcomeType(testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE)
	require.True(t, ok)
}

// The four rejections R10 names each land in an existing category at the field they name: an
// invalid duration, a field the Driver cannot set, a reply the activation does not admit, and a
// command type the Profile does not admit.
func TestPrepareRejectsTypedInstructions(t *testing.T) {
	schedule := func(c *testpilotspb.Case) *commandpb.Command {
		return c.Program.Entrypoints[1].Instructions[0].Instruction.GetWorkflowCommand().GetCommand()
	}
	for _, tc := range []struct {
		name     string
		mutate   func(*testpilotspb.Case, *Profile)
		category ir.ErrorCategory
		path     string
	}{
		{"command type the Profile does not admit", func(_ *testpilotspb.Case, p *Profile) { p.CommandTypes = nil }, ir.Unsupported,
			"program.entrypoints[workflow].instructions[start].instruction.workflow_command.command.command_type"},
		{"command type disagrees with its attributes", func(c *testpilotspb.Case, _ *Profile) {
			schedule(c).CommandType = enumspb.COMMAND_TYPE_START_TIMER
		}, ir.Malformed, "program.entrypoints[workflow].instructions[start].instruction.workflow_command.command.command_type"},
		{"command with no attributes", func(c *testpilotspb.Case, _ *Profile) { schedule(c).Attributes = nil }, ir.Malformed,
			"program.entrypoints[workflow].instructions[start].instruction.workflow_command.command.command_type"},
		{"unsettable command field", func(c *testpilotspb.Case, _ *Profile) {
			schedule(c).UserMetadata = &sdkpb.UserMetadata{}
		}, ir.Unsupported, "program.entrypoints[workflow].instructions[start].instruction.workflow_command.command.user_metadata"},
		{"negative duration", func(c *testpilotspb.Case, _ *Profile) {
			schedule(c).GetScheduleNexusOperationCommandAttributes().ScheduleToStartTimeout = durationpb.New(-time.Second)
		}, ir.Malformed, "program.entrypoints[workflow].instructions[start].instruction.workflow_command.command.schedule_nexus_operation_command_attributes.schedule_to_start_timeout"},
		{"zero duration", func(c *testpilotspb.Case, _ *Profile) {
			schedule(c).GetScheduleNexusOperationCommandAttributes().StartToCloseTimeout = durationpb.New(0)
		}, ir.Malformed, "program.entrypoints[workflow].instructions[start].instruction.workflow_command.command.schedule_nexus_operation_command_attributes.start_to_close_timeout"},
		{"duration over the Profile ceiling", func(c *testpilotspb.Case, _ *Profile) {
			schedule(c).GetScheduleNexusOperationCommandAttributes().ScheduleToCloseTimeout = durationpb.New(31 * time.Second)
		}, ir.LimitExceeded, "program.entrypoints[workflow].instructions[start].instruction.workflow_command.command.schedule_nexus_operation_command_attributes.schedule_to_close_timeout"},
		{"unknown endpoint role", func(c *testpilotspb.Case, _ *Profile) {
			schedule(c).GetScheduleNexusOperationCommandAttributes().Endpoint = "missing"
		}, ir.Unknown, "role"},
		{"empty header key", func(c *testpilotspb.Case, _ *Profile) {
			schedule(c).GetScheduleNexusOperationCommandAttributes().NexusHeader = map[string]string{"": "1"}
		}, ir.Malformed, "program.entrypoints[workflow].instructions[start].instruction.workflow_command.command.schedule_nexus_operation_command_attributes.nexus_header"},
		{"unauthorized command opcode", func(_ *testpilotspb.Case, p *Profile) {
			p.Opcodes = slices.DeleteFunc(p.Opcodes, func(opcode contract.Opcode) bool { return opcode == contract.WorkflowCommand })
		}, ir.Unsupported, "workflow.start"},
		{"sync reply publishing a handle", func(c *testpilotspb.Case, _ *Profile) {
			reply := syncReply()
			reply.HandleSlotId = "handle"
			c.Program.Entrypoints[2].Instructions[0] = replyNode("respond", reply)
		}, ir.Unsupported, "handler.respond"},
		{"async reply without a handle Slot", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[2].Instructions[0] = replyNode("respond", asyncReply("missing"))
		}, ir.TypeMismatch, "handler.respond"},
		{"reply carrying the Driver's operation token", func(c *testpilotspb.Case, _ *Profile) {
			reply := asyncReply("handle")
			reply.GetResponse().GetAsyncSuccess().OperationToken = "mine"
			c.Program.Entrypoints[2].Instructions[0] = replyNode("respond", reply)
		}, ir.Unsupported, "program.entrypoints[handler].instructions[respond].instruction.nexus_handler_reply.response.async_success.operation_token"},
		{"reply through the deprecated error arm", func(c *testpilotspb.Case, _ *Profile) {
			reply := &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Response{Response: &nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_OperationError{OperationError: &nexuspb.UnsuccessfulOperationError{}}}}} //nolint:staticcheck // the deprecated arm is what the reach table must reject
			c.Program.Entrypoints[2].Instructions[0] = replyNode("respond", reply)
		}, ir.Unsupported, "program.entrypoints[handler].instructions[respond].instruction.nexus_handler_reply.response.operation_error"},
		{"handler error without a type", func(c *testpilotspb.Case, _ *Profile) {
			reply := &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Error{Error: &nexuspb.HandlerError{}}}
			c.Program.Entrypoints[2].Instructions[0] = replyNode("respond", reply)
		}, ir.Malformed, "program.entrypoints[handler].instructions[respond].instruction.nexus_handler_reply.error.error_type"},
		{"handler error with an unsettable failure field", func(c *testpilotspb.Case, _ *Profile) {
			reply := &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Error{Error: &nexuspb.HandlerError{ErrorType: "BAD_REQUEST", Failure: &nexuspb.Failure{Message: "bad", Details: []byte("{}")}}}}
			c.Program.Entrypoints[2].Instructions[0] = replyNode("respond", reply)
		}, ir.Unsupported, "program.entrypoints[handler].instructions[respond].instruction.nexus_handler_reply.error.failure.details"},
		{"empty reply", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[2].Instructions[0] = replyNode("respond", &testpilotspb.NexusHandlerReply{})
		}, ir.Malformed, "program.entrypoints[handler].instructions[respond].instruction.nexus_handler_reply"},
		{"completion without a result", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[2].Instruction.GetNexusOperationCompletion().Result = nil
		}, ir.Malformed, "program.entrypoints[controller].instructions[complete].instruction.nexus_operation_completion"},
		{"completion with an unsettable failure field", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[2].Instruction.GetNexusOperationCompletion().Result = &testpilotspb.NexusOperationCompletion_Failure{Failure: &failurepb.Failure{Message: "failed", EncodedAttributes: &commonpb.Payload{}}}
		}, ir.Unsupported, "program.entrypoints[controller].instructions[complete].instruction.nexus_operation_completion.failure.encoded_attributes"},
		{"completion without a handle Slot", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Entrypoints[0].Instructions[2].Instruction.GetNexusOperationCompletion().HandleSlotId = "missing"
		}, ir.TypeMismatch, "controller.complete"},
		{"duplicate command type policy", func(_ *testpilotspb.Case, p *Profile) {
			p.CommandTypes = append(p.CommandTypes, enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION)
		}, ir.Malformed, "policy.command_types"},
		{"unspecified command type policy", func(_ *testpilotspb.Case, p *Profile) {
			p.CommandTypes = append(p.CommandTypes, enumspb.COMMAND_TYPE_UNSPECIFIED)
		}, ir.Malformed, "policy.command_types"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, catalog, p := handleFixture(t)
			tc.mutate(c, &p)
			_, err := Prepare(c, catalog, p)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, tc.category, diagnostic.Category, diagnostic.Detail)
			require.Equal(t, tc.path, diagnostic.Path, diagnostic.Detail)
		})
	}
}

// The Driver-reach table names every field of every message it carries: an API regeneration that
// adds a field fails here until the row says whether the Driver realizes it, and a row cannot name
// a field the message does not declare or name one twice.
func TestDriverReachTableNamesEveryField(t *testing.T) {
	require.NotEmpty(t, driverReach)
	for fullName, row := range driverReach {
		t.Run(string(fullName), func(t *testing.T) {
			descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName(fullName)
			require.NoError(t, err)
			message, ok := descriptor.(protoreflect.MessageDescriptor)
			require.True(t, ok)
			named := map[protoreflect.Name]int{}
			for _, name := range row.realized {
				named[name]++
			}
			for _, name := range row.unrealized {
				named[name]++
			}
			for name, count := range named {
				require.Equal(t, 1, count, "%s named twice", name)
				require.NotNil(t, message.Fields().ByName(name), "%s is not a field of %s", name, fullName)
			}
			for index := range message.Fields().Len() {
				field := message.Fields().Get(index)
				require.Contains(t, named, field.Name(), "%s.%s is in neither list", fullName, field.Name())
			}
		})
	}
}

// commandTypeOf derives the command type from the attributes arm alone, so every declared arm
// denotes the command type of its own name and a command with no arm denotes none.
func TestCommandTypeOfNamesEveryAttributesArm(t *testing.T) {
	oneof := (&commandpb.Command{}).ProtoReflect().Descriptor().Oneofs().ByName("attributes")
	require.NotNil(t, oneof)
	for index := range oneof.Fields().Len() {
		field := oneof.Fields().Get(index)
		command := &commandpb.Command{}
		command.ProtoReflect().Mutable(field)
		expected := "COMMAND_TYPE_" + strings.ToUpper(strings.TrimSuffix(string(field.Name()), "_command_attributes"))
		require.Equal(t, expected, enumspb.CommandType_name[int32(commandTypeOf(command))], field.Name())
		require.NotEqual(t, enumspb.COMMAND_TYPE_UNSPECIFIED, commandTypeOf(command), field.Name())
	}
	require.Equal(t, enumspb.COMMAND_TYPE_UNSPECIFIED, commandTypeOf(&commandpb.Command{}))
	require.Equal(t, enumspb.COMMAND_TYPE_UNSPECIFIED, commandTypeOf(nil))
}

func TestPrepareTypedInstructionsKeepTheirContexts(t *testing.T) {
	for _, tc := range []struct {
		name        string
		instruction *testpilotspb.Instruction
		expected    contract.EntrypointKind
	}{
		{"command", scheduleNode("x").Instruction, contract.WorkflowEntrypoint},
		{"reply", replyNode("x", syncReply()).Instruction, contract.NexusHandlerEntrypoint},
		{"completion", completionNode("x", payloadCompletion("handle")).Instruction, contract.ControllerEntrypoint},
	} {
		for _, context := range []contract.EntrypointKind{contract.ControllerEntrypoint, contract.WorkflowEntrypoint, contract.NexusHandlerEntrypoint} {
			if context == tc.expected {
				continue
			}
			t.Run(fmt.Sprintf("%s in kind %d", tc.name, context), func(t *testing.T) {
				c, catalog, p := fixture(t)
				addWorker(c, &p)
				c.Program.Slots = []*testpilotspb.Slot{handleSlot("handle")}
				g := c.Program.Entrypoints[0]
				g.Instructions[0].Instruction = tc.instruction
				switch context {
				case contract.WorkflowEntrypoint:
					g.Activation = &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				case contract.NexusHandlerEntrypoint:
					g.Activation = &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{Service: "service", Operation: "operation", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
				default:
				}
				_, err := Prepare(c, catalog, p)
				var diagnostic *ir.Error
				require.ErrorAs(t, err, &diagnostic)
				require.Equal(t, ir.Unsupported, diagnostic.Category, diagnostic.Detail)
				require.Equal(t, "unsupported instruction context or Driver capability", diagnostic.Detail)
			})
		}
	}
}

func TestCarriedCompletionIsThePayloadOrFailure(t *testing.T) {
	payload := payloadCompletion("handle")
	require.True(t, proto.Equal(payload.GetPayload(), carriedCompletion(payload)))
	failure := &testpilotspb.NexusOperationCompletion{HandleSlotId: "handle", Result: &testpilotspb.NexusOperationCompletion_Failure{Failure: &failurepb.Failure{Message: "failed"}}}
	require.True(t, proto.Equal(failure.GetFailure(), carriedCompletion(failure)))
	require.Nil(t, carriedCompletion(&testpilotspb.NexusOperationCompletion{}))
}
