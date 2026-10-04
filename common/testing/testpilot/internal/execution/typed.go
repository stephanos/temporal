package execution

import (
	"strings"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
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
		realized:   []protoreflect.Name{"command_type", "schedule_nexus_operation_command_attributes", "schedule_activity_task_command_attributes"},
		unrealized: []protoreflect.Name{"user_metadata", "event_group_markers", "start_timer_command_attributes", "complete_workflow_execution_command_attributes", "fail_workflow_execution_command_attributes", "request_cancel_activity_task_command_attributes", "cancel_timer_command_attributes", "cancel_workflow_execution_command_attributes", "request_cancel_external_workflow_execution_command_attributes", "record_marker_command_attributes", "continue_as_new_workflow_execution_command_attributes", "start_child_workflow_execution_command_attributes", "signal_external_workflow_execution_command_attributes", "upsert_workflow_search_attributes_command_attributes", "protocol_message_command_attributes", "modify_workflow_properties_command_attributes", "request_cancel_nexus_operation_command_attributes"},
	},
	(&commandpb.ScheduleNexusOperationCommandAttributes{}).ProtoReflect().Descriptor().FullName(): {
		realized: []protoreflect.Name{"endpoint", "service", "operation", "input", "schedule_to_close_timeout", "nexus_header", "schedule_to_start_timeout", "start_to_close_timeout"},
	},
	(&commandpb.ScheduleActivityTaskCommandAttributes{}).ProtoReflect().Descriptor().FullName(): {
		realized: []protoreflect.Name{"activity_id", "activity_type", "task_queue", "input", "schedule_to_close_timeout", "schedule_to_start_timeout", "start_to_close_timeout", "heartbeat_timeout", "retry_policy"},
		// The SDK writes the header from its context propagators; the Driver requests no eager
		// execution, and sets no build id or priority.
		unrealized: []protoreflect.Name{"header", "request_eager_execution", "use_workflow_build_id", "priority"},
	},
	(&commonpb.ActivityType{}).ProtoReflect().Descriptor().FullName(): {
		realized: []protoreflect.Name{"name"},
	},
	(&taskqueuepb.TaskQueue{}).ProtoReflect().Descriptor().FullName(): {
		realized:   []protoreflect.Name{"name"},
		unrealized: []protoreflect.Name{"kind", "normal_name"},
	},
	(&commonpb.Payloads{}).ProtoReflect().Descriptor().FullName(): {
		realized: []protoreflect.Name{"payloads"},
	},
	(&commonpb.RetryPolicy{}).ProtoReflect().Descriptor().FullName(): {
		realized: []protoreflect.Name{"initial_interval", "backoff_coefficient", "maximum_interval", "maximum_attempts", "non_retryable_error_types"},
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
		return ir.Invalid(ir.Unsupported, path, "message the Driver cannot carry")
	}
	var err error
	reflection.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		fieldPath := path + "." + string(field.Name())
		for _, name := range row.unrealized {
			if field.Name() == name {
				err = ir.Invalid(ir.Unsupported, fieldPath, "field the Driver cannot set through the SDK")
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
		return ir.Invalid(ir.Malformed, path, "nil workflow command")
	}
	denoted := commandTypeOf(command)
	if denoted == enumspb.COMMAND_TYPE_UNSPECIFIED || command.GetCommandType() != denoted {
		return ir.Invalid(ir.Malformed, path+".command_type", "command type does not name the attributes the command carries")
	}
	if !a.commandTypes[denoted] {
		return ir.Invalid(ir.Unsupported, path+".command_type", "command type "+enumspb.CommandType_name[int32(denoted)]+" the Profile does not admit")
	}
	if err := checkReach(command, path); err != nil {
		return err
	}
	switch denoted {
	case enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION:
		return a.bindScheduleNexus(command.GetScheduleNexusOperationCommandAttributes(), path+".schedule_nexus_operation_command_attributes")
	case enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK:
		return a.bindScheduleActivity(command.GetScheduleActivityTaskCommandAttributes(), path+".schedule_activity_task_command_attributes")
	default:
		// The reach table already rejects every other attributes arm; a Profile that admits its type
		// still names a command no Driver realizes.
		return ir.Invalid(ir.Unsupported, path+".command_type", "command type "+enumspb.CommandType_name[int32(denoted)]+" the Driver cannot realize")
	}
}

// bindScheduleNexus admits a Nexus schedule: a valid service and operation on a declared endpoint
// role, header keys that name something, and timeouts within the Profile's ceiling.
func (a *admission) bindScheduleNexus(attributes *commandpb.ScheduleNexusOperationCommandAttributes, path string) error {
	if !ir.ValidID(attributes.GetService()) || !ir.ValidID(attributes.GetOperation()) {
		return ir.Invalid(ir.Malformed, path, "invalid Nexus service or operation")
	}
	if err := a.role(attributes.GetEndpoint(), testpilotspb.ROLE_KIND_ENDPOINT); err != nil {
		return err
	}
	for key := range attributes.GetNexusHeader() {
		if key == "" {
			return ir.Invalid(ir.Malformed, path+".nexus_header", "empty Nexus header key")
		}
	}
	return a.checkDurations(attributes, path, "schedule_to_close_timeout", "schedule_to_start_timeout", "start_to_close_timeout")
}

// bindScheduleActivity admits an activity schedule: a valid activity type on a declared task-queue
// role, which the SDK emits on every schedule, and timeouts and retry intervals within the
// Profile's ceiling.
func (a *admission) bindScheduleActivity(attributes *commandpb.ScheduleActivityTaskCommandAttributes, path string) error {
	if !ir.ValidID(attributes.GetActivityType().GetName()) {
		return ir.Invalid(ir.Malformed, path+".activity_type", "invalid activity type")
	}
	if err := a.role(attributes.GetTaskQueue().GetName(), testpilotspb.ROLE_KIND_TASK_QUEUE); err != nil {
		return err
	}
	if err := a.checkDurations(attributes, path, "schedule_to_close_timeout", "schedule_to_start_timeout", "start_to_close_timeout", "heartbeat_timeout"); err != nil {
		return err
	}
	if policy := attributes.GetRetryPolicy(); policy != nil {
		return a.checkDurations(policy, path+".retry_policy", "initial_interval", "maximum_interval")
	}
	return nil
}

// checkDurations admits each named duration field the message carries.
func (a *admission) checkDurations(message proto.Message, path string, names ...protoreflect.Name) error {
	reflection := message.ProtoReflect()
	for _, name := range names {
		field := reflection.Descriptor().Fields().ByName(name)
		if !reflection.Has(field) {
			continue
		}
		if err := a.checkDuration(reflection.Get(field).Message().Interface(), path+"."+string(name)); err != nil {
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
		return ir.Invalid(ir.Malformed, path, "invalid duration")
	}
	if duration.AsDuration().Milliseconds() > a.declaredLimits.MaxTotalDurationMilliseconds {
		return ir.Invalid(ir.LimitExceeded, path, "duration exceeds the Profile's total duration ceiling")
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
			return ir.Invalid(ir.Malformed, path+".response", "start response carries no variant")
		}
		async = typed.Response.GetAsyncSuccess() != nil
	case *testpilotspb.NexusHandlerReply_Error:
		carried, arm = typed.Error, "error"
		if typed.Error.GetErrorType() == "" {
			return ir.Invalid(ir.Malformed, path+".error.error_type", "handler error names no type")
		}
		if _, declared := enumspb.NexusHandlerErrorRetryBehavior_name[int32(typed.Error.GetRetryBehavior())]; !declared {
			return ir.Invalid(ir.Malformed, path+".error.retry_behavior", "undeclared retry behavior")
		}
	default:
		return ir.Invalid(ir.Malformed, path, "nil Nexus handler reply")
	}
	if err := checkReach(carried, path+"."+arm); err != nil {
		return err
	}
	if async {
		typ, exists := a.prepared.slots[reply.GetHandleSlotId()]
		if !exists || !typ.Opaque() {
			return ir.Invalid(ir.TypeMismatch, nodePath(g, n), "async response requires a handle Slot")
		}
		return a.addWriter(reply.GetHandleSlotId(), slotWriter{graph: g, node: i})
	}
	if reply.GetHandleSlotId() != "" {
		return ir.Invalid(ir.Unsupported, nodePath(g, n), "only async responses publish handles")
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
		return ir.Invalid(ir.TypeMismatch, nodePath(g, n), "completion requires a handle Slot")
	}
	switch typed := completion.GetResult().(type) {
	case *testpilotspb.NexusOperationCompletion_Payload:
		return checkReach(typed.Payload, path+".payload")
	case *testpilotspb.NexusOperationCompletion_Failure:
		return checkReach(typed.Failure, path+".failure")
	default:
		return ir.Invalid(ir.Malformed, path, "completion carries no result")
	}
}

// carriedCompletion is the message a typed completion delivers through its opaque handle: the payload
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

// startsNexusOperation reports whether an instruction starts a Nexus operation a reserved handler
// may answer: a workflow command scheduling one.
func startsNexusOperation(instruction *testpilotspb.Instruction) bool {
	return scheduledCommandType(instruction) == enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION
}

// scheduledActivityOf is the activity type and task-queue role an instruction's activity schedule
// names, or the zero key for any other instruction.
func scheduledActivityOf(instruction *testpilotspb.Instruction) activityKey {
	if scheduledCommandType(instruction) != enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK {
		return activityKey{}
	}
	attributes := instruction.GetWorkflowCommand().GetCommand().GetScheduleActivityTaskCommandAttributes()
	return activityKey{activityType: attributes.GetActivityType().GetName(), queueRole: attributes.GetTaskQueue().GetName()}
}

// startsAwaitable reports whether an instruction starts what an AwaitInstruction of the same
// entrypoint may await: a workflow command scheduling a Nexus operation or an activity.
func startsAwaitable(instruction *testpilotspb.Instruction) bool {
	switch scheduledCommandType(instruction) {
	case enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION, enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK:
		return true
	default:
		return false
	}
}

// scheduledCommandType is the command type a workflow command instruction issues, or UNSPECIFIED
// for any other instruction.
func scheduledCommandType(instruction *testpilotspb.Instruction) enumspb.CommandType {
	if InstructionOpcode(instruction) != contract.WorkflowCommand {
		return enumspb.COMMAND_TYPE_UNSPECIFIED
	}
	return commandTypeOf(instruction.GetWorkflowCommand().GetCommand())
}
