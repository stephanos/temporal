package execution

import (
	"strings"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
)

// The typed worker instructions carry Temporal API messages. Admission reads each carried message
// against the Driver-reach table below: a field the Temporal Driver realizes through its SDK is
// carried, a field it cannot set rejects at preparation naming the field, and the table's
// completeness test requires every field of every carried message to be named by one of the two
// lists, so an API regeneration that adds a field is reviewed here before a Case can carry it. The
// table lives with admission rather than with the Driver because preparation runs without a Driver;
// the Driver's interpreter reads only the fields it names as realized.

// messageReach is one carried message's row: the fields the Driver realizes and the fields it does
// not. A realized message-typed field whose message has its own row is read against that row.
type messageReach struct {
	realized, unrealized []protoreflect.Name
}

// driverReach is the Driver-reach table, keyed by carried message.
var driverReach = map[protoreflect.FullName]messageReach{
	(&commandpb.Command{}).ProtoReflect().Descriptor().FullName(): {
		realized:   []protoreflect.Name{"command_type", "schedule_nexus_operation_command_attributes"},
		unrealized: []protoreflect.Name{"user_metadata", "event_group_markers", "schedule_activity_task_command_attributes", "start_timer_command_attributes", "complete_workflow_execution_command_attributes", "fail_workflow_execution_command_attributes", "request_cancel_activity_task_command_attributes", "cancel_timer_command_attributes", "cancel_workflow_execution_command_attributes", "request_cancel_external_workflow_execution_command_attributes", "record_marker_command_attributes", "continue_as_new_workflow_execution_command_attributes", "start_child_workflow_execution_command_attributes", "signal_external_workflow_execution_command_attributes", "upsert_workflow_search_attributes_command_attributes", "protocol_message_command_attributes", "modify_workflow_properties_command_attributes", "request_cancel_nexus_operation_command_attributes"},
	},
	(&commandpb.ScheduleNexusOperationCommandAttributes{}).ProtoReflect().Descriptor().FullName(): {
		realized: []protoreflect.Name{"endpoint", "service", "operation", "input", "schedule_to_close_timeout", "nexus_header", "schedule_to_start_timeout", "start_to_close_timeout"},
	},
	(&nexuspb.StartOperationResponse{}).ProtoReflect().Descriptor().FullName(): {
		realized:   []protoreflect.Name{"sync_success", "async_success", "failure"},
		unrealized: []protoreflect.Name{"operation_error"},
	},
	(&nexuspb.StartOperationResponse_Sync{}).ProtoReflect().Descriptor().FullName(): {
		realized:   []protoreflect.Name{"payload"},
		unrealized: []protoreflect.Name{"links"},
	},
	(&nexuspb.StartOperationResponse_Async{}).ProtoReflect().Descriptor().FullName(): {
		// The Driver issues the operation token, so a reply carries none.
		unrealized: []protoreflect.Name{"operation_id", "links", "operation_token"},
	},
	(&nexuspb.HandlerError{}).ProtoReflect().Descriptor().FullName(): {
		realized: []protoreflect.Name{"error_type", "failure", "retry_behavior"},
	},
	(&nexuspb.Failure{}).ProtoReflect().Descriptor().FullName(): {
		realized:   []protoreflect.Name{"message"},
		unrealized: []protoreflect.Name{"stack_trace", "metadata", "details", "cause"},
	},
	(&commonpb.Payload{}).ProtoReflect().Descriptor().FullName(): {
		realized: []protoreflect.Name{"metadata", "data", "external_payloads"},
	},
	(&failurepb.Failure{}).ProtoReflect().Descriptor().FullName(): {
		realized:   []protoreflect.Name{"message", "source", "stack_trace", "cause", "application_failure_info", "timeout_failure_info", "canceled_failure_info", "terminated_failure_info", "server_failure_info", "reset_workflow_failure_info", "activity_failure_info", "child_workflow_execution_failure_info", "nexus_operation_execution_failure_info", "nexus_handler_failure_info"},
		unrealized: []protoreflect.Name{"encoded_attributes"},
	},
}

// checkReach rejects a populated field of message the Driver does not realize, naming the field at
// its path, and reads each populated message-typed field the table has a row for the same way.
func checkReach(message proto.Message, path string) error {
	reflection := message.ProtoReflect()
	row, known := driverReach[reflection.Descriptor().FullName()]
	if !known {
		return invalid(ir.Unsupported, path, "message the Driver cannot carry")
	}
	var err error
	reflection.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldPath := path + "." + string(field.Name())
		for _, name := range row.unrealized {
			if field.Name() == name {
				err = invalid(ir.Unsupported, fieldPath, "field the Driver cannot set through the SDK")
				return false
			}
		}
		if field.Kind() == protoreflect.MessageKind && !field.IsList() && !field.IsMap() {
			if _, nested := driverReach[field.Message().FullName()]; nested {
				err = checkReach(value.Message().Interface(), fieldPath)
				return err == nil
			}
		}
		return true
	})
	return err
}

// commandTypeOf is the command type the attributes arm of command denotes, or UNSPECIFIED when the
// command carries no attributes: the arm `<name>_command_attributes` denotes `COMMAND_TYPE_<NAME>`.
func commandTypeOf(command *commandpb.Command) enumspb.CommandType {
	reflection := command.ProtoReflect()
	arm := reflection.WhichOneof(reflection.Descriptor().Oneofs().ByName("attributes"))
	if arm == nil {
		return enumspb.COMMAND_TYPE_UNSPECIFIED
	}
	name := "COMMAND_TYPE_" + strings.ToUpper(strings.TrimSuffix(string(arm.Name()), "_command_attributes"))
	return enumspb.CommandType(enumspb.CommandType_value[name])
}

// bindWorkflowCommand admits a workflow command: its type is one the Profile admits and the one its
// attributes denote, and its attributes are within the Driver's reach, name declared roles, and
// carry durations within the Profile's ceiling.
func (a *admission) bindWorkflowCommand(g *graph, n *node) error {
	command := n.source.Instruction.GetWorkflowCommand().GetCommand()
	path := expressionPath(g, n, "instruction.workflow_command.command")
	if command == nil {
		return invalid(ir.Malformed, path, "nil workflow command")
	}
	denoted := commandTypeOf(command)
	if denoted == enumspb.COMMAND_TYPE_UNSPECIFIED || command.GetCommandType() != denoted {
		return invalid(ir.Malformed, path+".command_type", "command type does not name the attributes the command carries")
	}
	if !a.commandTypes[denoted] {
		return invalid(ir.Unsupported, path+".command_type", "command type the Profile does not admit")
	}
	if err := checkReach(command, path); err != nil {
		return err
	}
	attributes := command.GetScheduleNexusOperationCommandAttributes()
	attributesPath := path + ".schedule_nexus_operation_command_attributes"
	if !validID(attributes.GetService()) || !validID(attributes.GetOperation()) {
		return invalid(ir.Malformed, attributesPath, "invalid Nexus service or operation")
	}
	if err := a.role(attributes.GetEndpoint(), testpilotspb.ROLE_KIND_ENDPOINT); err != nil {
		return err
	}
	for key := range attributes.GetNexusHeader() {
		if key == "" {
			return invalid(ir.Malformed, attributesPath+".nexus_header", "empty Nexus header key")
		}
	}
	reflection := attributes.ProtoReflect()
	for _, name := range []protoreflect.Name{"schedule_to_close_timeout", "schedule_to_start_timeout", "start_to_close_timeout"} {
		field := reflection.Descriptor().Fields().ByName(name)
		if !reflection.Has(field) {
			continue
		}
		if err := a.checkDuration(reflection.Get(field).Message().Interface(), attributesPath+"."+string(name)); err != nil {
			return err
		}
	}
	return nil
}

// checkDuration admits one carried timeout: a valid, positive duration within the Profile's total
// duration ceiling. A duration the protobuf library cannot represent is malformed; one over the
// ceiling exceeds a limit.
func (a *admission) checkDuration(message proto.Message, path string) error {
	duration, ok := message.(*durationpb.Duration)
	if !ok || duration.CheckValid() != nil || duration.AsDuration() <= 0 {
		return invalid(ir.Malformed, path, "invalid duration")
	}
	if duration.AsDuration().Milliseconds() > a.prepared.limits.MaxTotalDurationMilliseconds {
		return invalid(ir.LimitExceeded, path, "duration exceeds the Profile's total duration ceiling")
	}
	return nil
}

// bindNexusHandlerReply admits a handler reply: a start response or a handler error within the
// Driver's reach, where only an asynchronous response publishes a handle, into a declared handle Slot.
func (a *admission) bindNexusHandlerReply(g *graph, i int, n *node) error {
	reply := n.source.Instruction.GetNexusHandlerReply()
	path := expressionPath(g, n, "instruction.nexus_handler_reply")
	var carried proto.Message
	var arm string
	async := false
	switch typed := reply.GetReply().(type) {
	case *testpilotspb.NexusHandlerReply_Response:
		carried, arm = typed.Response, "response"
		if typed.Response.GetVariant() == nil {
			return invalid(ir.Malformed, path+".response", "start response carries no variant")
		}
		async = typed.Response.GetAsyncSuccess() != nil
	case *testpilotspb.NexusHandlerReply_Error:
		carried, arm = typed.Error, "error"
		if typed.Error.GetErrorType() == "" {
			return invalid(ir.Malformed, path+".error.error_type", "handler error names no type")
		}
		if _, declared := enumspb.NexusHandlerErrorRetryBehavior_name[int32(typed.Error.GetRetryBehavior())]; !declared {
			return invalid(ir.Malformed, path+".error.retry_behavior", "undeclared retry behavior")
		}
	default:
		return invalid(ir.Malformed, path, "nil Nexus handler reply")
	}
	if err := checkReach(carried, path+"."+arm); err != nil {
		return err
	}
	if async {
		typ, exists := a.prepared.slots[reply.GetHandleSlotId()]
		if !exists || !typ.Opaque() {
			return invalid(ir.TypeMismatch, nodePath(g, n), "async response requires a handle Slot")
		}
		return a.addWriter(reply.GetHandleSlotId(), slotWriter{graph: g, node: i})
	}
	if reply.GetHandleSlotId() != "" {
		return invalid(ir.Unsupported, nodePath(g, n), "only async responses publish handles")
	}
	return nil
}

// bindNexusOperationCompletion admits a typed completion: a handle Slot to consume and a payload
// or failure within the Driver's reach.
func (a *admission) bindNexusOperationCompletion(g *graph, n *node) error {
	completion := n.source.Instruction.GetNexusOperationCompletion()
	path := expressionPath(g, n, "instruction.nexus_operation_completion")
	typ, exists := a.prepared.slots[completion.GetHandleSlotId()]
	if !exists || !typ.Opaque() {
		return invalid(ir.TypeMismatch, nodePath(g, n), "completion requires a handle Slot")
	}
	switch typed := completion.GetResult().(type) {
	case *testpilotspb.NexusOperationCompletion_Payload:
		return checkReach(typed.Payload, path+".payload")
	case *testpilotspb.NexusOperationCompletion_Failure:
		return checkReach(typed.Failure, path+".failure")
	default:
		return invalid(ir.Malformed, path, "completion carries no result")
	}
}

// carriedCompletion is the message a typed completion delivers through its capability: the payload
// or the failure the instruction carries.
func carriedCompletion(completion *testpilotspb.NexusOperationCompletion) proto.Message {
	switch typed := completion.GetResult().(type) {
	case *testpilotspb.NexusOperationCompletion_Payload:
		return typed.Payload
	case *testpilotspb.NexusOperationCompletion_Failure:
		return typed.Failure
	default:
		return nil
	}
}

// nexusOperationOf is the service and operation an instruction starts: the schedule command's.
func nexusOperationOf(instruction *testpilotspb.Instruction) nexusOperation {
	attributes := instruction.GetWorkflowCommand().GetCommand().GetScheduleNexusOperationCommandAttributes()
	return nexusOperation{service: attributes.GetService(), operation: attributes.GetOperation()}
}

// startsNexusOperation reports whether an instruction starts a Nexus operation an AwaitInstruction
// of the same entrypoint may await: a workflow command scheduling one.
func startsNexusOperation(instruction *testpilotspb.Instruction) bool {
	switch InstructionOpcode(instruction) {
	case contract.WorkflowCommand:
		return commandTypeOf(instruction.GetWorkflowCommand().GetCommand()) == enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION
	default:
		return false
	}
}
