package worker

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	commonnexus "go.temporal.io/server/common/nexus"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/types/known/anypb"
)

// The typed worker instructions carry Temporal API messages, and this Driver realizes each through
// the SDK call that produces it: a ScheduleNexusOperationCommandAttributes becomes
// workflow.ExecuteNexusOperation with the command's input and timeouts; a StartOperationResponse
// becomes the handler's return, synchronous with its payload or asynchronous through the completion
// authority the Driver publishes; a HandlerError becomes the error the handler returns with its type
// and retry behavior; and a completion Payload or Failure becomes the completion callback's body.
// Which fields of each message reach the SDK is the Driver-reach table in internal/execution, which
// preparation admits a Case against; the code here reads only the fields that table names realized.

// CommandTypes are the workflow command types this Driver realizes. DeriveProfile admits exactly
// these, so a Case carrying another command type rejects at preparation.
func CommandTypes() []enumspb.CommandType {
	return []enumspb.CommandType{enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION}
}

// scheduleNexusOperation is the schedule command an instruction carries, or nil for any other
// instruction.
func scheduleNexusOperation(instruction *testpilotspb.Instruction) *commandpb.ScheduleNexusOperationCommandAttributes {
	return instruction.GetWorkflowCommand().GetCommand().GetScheduleNexusOperationCommandAttributes()
}

// startsNexusOperation reports whether an instruction starts a Nexus operation the Driver must
// prepare a dispatch route for: the untyped start, or a schedule command.
func startsNexusOperation(instruction testpilot.InstructionPlan) bool {
	return instruction.Opcode() == testpilot.StartNexusOperation || scheduleNexusOperation(instruction.Source().GetInstruction()) != nil
}

// caseNexusHeaderKey carries a schedule command's Nexus header to the outbound interceptor, which
// merges the Driver's routing header into it.
type caseNexusHeaderKey struct{}

// scheduleNexus issues the schedule command: the operation's input is the carried payload, passed
// through the data converter unconverted, and its timeouts are the carried durations, the
// schedule-to-close one defaulting to the instruction's own timeout as the untyped start does.
func (i *workflowInterpreter) scheduleNexus(index int, instruction testpilot.InstructionPlan) error {
	source := instruction.Source()
	attributes := scheduleNexusOperation(source.GetInstruction())
	endpoint := i.session.definition.endpoints[attributes.GetEndpoint()]
	if attributes == nil || endpoint == "" {
		return ErrInvalid
	}
	operationCtx := workflow.WithValue(i.ctx, workflowSourceKey{}, source.GetInstructionId())
	operationCtx = workflow.WithValue(operationCtx, caseNexusHeaderKey{}, nexus.Header(maps.Clone(attributes.GetNexusHeader())))
	options := workflow.NexusOperationOptions{
		ScheduleToCloseTimeout: time.Duration(instruction.TimeoutMilliseconds()) * time.Millisecond,
		CancellationType:       workflow.NexusOperationCancellationTypeWaitRequested,
	}
	if attributes.GetScheduleToCloseTimeout() != nil {
		options.ScheduleToCloseTimeout = attributes.GetScheduleToCloseTimeout().AsDuration()
	}
	if attributes.GetScheduleToStartTimeout() != nil {
		options.ScheduleToStartTimeout = attributes.GetScheduleToStartTimeout().AsDuration()
	}
	if attributes.GetStartToCloseTimeout() != nil {
		options.StartToCloseTimeout = attributes.GetStartToCloseTimeout().AsDuration()
	}
	var input any
	if attributes.GetInput() != nil {
		input = converter.NewRawValue(attributes.GetInput())
	}
	future := workflow.NewNexusClient(endpoint, attributes.GetService()).ExecuteOperation(operationCtx, attributes.GetOperation(), input, options)
	i.futures[source.GetInstructionId()] = future
	i.typed[source.GetInstructionId()] = true
	var execution workflow.NexusOperationExecution
	err := future.GetNexusOperationExecution().Get(i.ctx, &execution)
	return i.state.Admit(context.Background(), index, outcomeForError(err))
}

// awaitedPayload reads a scheduled operation's result as the payload the handler answered, whole,
// as the Await's VALUE: an Any of the payload, or of an empty payload when the operation answered
// none.
func awaitedPayload(ctx workflow.Context, future workflow.NexusOperationFuture) (*testpilotspb.Value, error) {
	var raw converter.RawValue
	if err := future.Get(ctx, &raw); err != nil {
		return nil, err
	}
	payload := raw.Payload()
	if payload == nil {
		payload = &commonpb.Payload{}
	}
	packed, err := anypb.New(payload)
	if err != nil {
		return nil, err
	}
	return &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: packed}}, nil
}

// replyTyped answers the activation with a typed handler reply: a start response or a handler error.
func (s *Session) replyTyped(ctx context.Context, delivered delivery.Activation, reply *testpilotspb.NexusHandlerReply, options nexus.StartOperationOptions) (nexusResult, error) {
	switch typed := reply.GetReply().(type) {
	case *testpilotspb.NexusHandlerReply_Response:
		switch variant := typed.Response.GetVariant().(type) {
		case *nexuspb.StartOperationResponse_SyncSuccess:
			return nexusResult{kind: testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, raw: variant.SyncSuccess.GetPayload()}, nil
		case *nexuspb.StartOperationResponse_AsyncSuccess:
			token, err := s.publishCompletionAuthority(ctx, delivered, reply.GetHandleSlotId(), options)
			if err != nil {
				return nexusResult{}, err
			}
			return nexusResult{kind: testpilotspb.NEXUS_RESPONSE_KIND_ASYNCHRONOUS, token: token}, nil
		case *nexuspb.StartOperationResponse_Failure:
			failure, err := operationError(variant.Failure)
			if err != nil {
				return nexusResult{}, err
			}
			return nexusResult{kind: testpilotspb.NEXUS_RESPONSE_KIND_ERROR, replied: true}, failure
		default:
			return nexusResult{}, ErrInvalid
		}
	case *testpilotspb.NexusHandlerReply_Error:
		return nexusResult{kind: testpilotspb.NEXUS_RESPONSE_KIND_ERROR, replied: true}, handlerError(typed.Error)
	default:
		return nexusResult{}, ErrInvalid
	}
}

// handlerError is the error a handler returns for a carried HandlerError: its type, its failure's
// message and its retry behavior, each as the SDK spells it.
func handlerError(carried *nexuspb.HandlerError) *nexus.HandlerError {
	message := carried.GetFailure().GetMessage()
	if message == "" {
		message = "Nexus handler returned an error"
	}
	behavior := nexus.HandlerErrorRetryBehaviorUnspecified
	switch carried.GetRetryBehavior() {
	case enumspb.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE:
		behavior = nexus.HandlerErrorRetryBehaviorRetryable
	case enumspb.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE:
		behavior = nexus.HandlerErrorRetryBehaviorNonRetryable
	default:
		// Unspecified leaves the behavior to the error type, as the SDK does.
	}
	return &nexus.HandlerError{Type: nexus.HandlerErrorType(carried.GetErrorType()), Message: boundedText(message), RetryBehavior: behavior}
}

// operationError is the operation error a carried Failure ends an operation with: canceled when the
// failure says so and failed otherwise, carrying the failure converted as the Nexus SDK carries a
// Temporal failure.
func operationError(carried *failurepb.Failure) (*nexus.OperationError, error) {
	if carried == nil {
		return nil, ErrInvalid
	}
	converted, err := commonnexus.TemporalFailureToNexusFailure(carried)
	if err != nil {
		return nil, errors.Join(ErrInvalid, err)
	}
	state := nexus.OperationStateFailed
	if carried.GetCanceledFailureInfo() != nil {
		state = nexus.OperationStateCanceled
	}
	return &nexus.OperationError{State: state, Message: boundedText(carried.GetMessage()), Cause: &nexus.FailureError{Failure: converted}}, nil
}
