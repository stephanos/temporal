package fixture.typed

import io.temporal.api.workflowservice.v1.{
  DescribeWorkflowExecutionResponse,
  GetWorkflowExecutionHistoryRequest,
  ListWorkflowExecutionsRequest,
  ListWorkflowExecutionsResponse,
  StartActivityExecutionRequest,
  StartActivityExecutionResponse,
  WorkflowServiceGrpc
}
import io.temporal.api.workflow.v1.WorkflowExecutionInfo
import io.temporal.api.history.v1.HistoryEvent
import io.temporal.api.common.v1.WorkflowExecution
import io.temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryResponse
import temporal.server.api.testpilot.v1.{CorrelatedEvidence, InstructionOutcome}
import temporal.server.api.testpilot.v1.DeliveryAdmissionDecision
import umpire.*
import umpire.realize.*

val one = action("one", Party("caller")).schema[StartActivityExecutionRequest]
val two = action("two", Party("caller"))
  .schema[StartActivityExecutionResponse]
  .schema[StartActivityExecutionRequest]

val oldOne = action("one", Party("caller"))
  .schema("temporal.api.workflowservice.v1.StartActivityExecutionRequest")
val oldTwo = action("two", Party("caller"))
  .schema(
    "temporal.api.workflowservice.v1.StartActivityExecutionResponse",
    "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
  )

enum State derives Finite:
  case idle

enum Result derives Finite:
  case accepted

def step(s: State): List[Step[State, Result, Nothing]] = List(
  Step(Result.accepted, s)
)

val typedMachine: Machine[State, Result, Nothing] =
  machine[State, Result, Nothing](Family("fixture.typed"), "typed") {
    starts(State.idle)
    ends(_ => true)
    steps(one ~> step, two ~> step)
  }

val oldMachine: Machine[State, Result, Nothing] =
  machine[State, Result, Nothing](Family("fixture.typed"), "old") {
    starts(State.idle)
    ends(_ => true)
    steps(oldOne ~> step, oldTwo ~> step)
  }

val call = Instruction.rpc(
  "endpoint",
  WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
)(
  Vector(
    Assignment.typed(
      Field[StartActivityExecutionRequest, String](_.namespace),
      Operand.run()
    ),
    Assignment.typed(
      Field[StartActivityExecutionRequest, String](_.getTaskQueue.name),
      Operand.environment[String]("queue")
    )
  ),
  Vector(
    ResponseRead.typed(
      Field[StartActivityExecutionResponse, String](_.runId),
      Cardinality.one,
      Vector(Target.Bind("run"))
    )
  )
)

val listed = Recorded.read(
  WorkflowServiceGrpc.METHOD_LIST_WORKFLOW_EXECUTIONS,
  Field[ListWorkflowExecutionsResponse, Seq[WorkflowExecutionInfo]](
    _.executions.map(execution => execution)
  )
)
val evidence = Evidence.read(
  id = "fixture.typed.evidence.listed",
  records = "listed",
  source = "fixture.typed.source.listed",
  from = listed,
  operation = Field[WorkflowExecutionInfo, String](_.getExecution.workflowId),
  commitment = Commitment.reported
)
val described = Recorded.single(
  WorkflowServiceGrpc.METHOD_DESCRIBE_WORKFLOW_EXECUTION,
  Field[DescribeWorkflowExecutionResponse, WorkflowExecutionInfo](
    _.getWorkflowExecutionInfo
  )
)
val singleEvidence = Evidence.read(
  id = "fixture.typed.evidence.described",
  records = "described",
  source = "fixture.typed.source.described",
  from = described,
  operation = Field[WorkflowExecutionInfo, String](_.getExecution.workflowId),
  commitment = Commitment.reported
)
val poll = Instruction.poll(evidence, "endpoint")(
  Vector(
    Assignment.typed(
      Field[ListWorkflowExecutionsRequest, String](_.namespace),
      Operand.run()
    )
  ),
  Condition.equal(
    Field[WorkflowExecutionInfo, String](_.getExecution.workflowId),
    Operand.text("workflow")
  ),
  250
)

val historyCall = Instruction.rpc(
  "endpoint",
  WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY
)(
  Vector(
    Assignment.typed(
      Field[GetWorkflowExecutionHistoryRequest, String](_.namespace),
      Operand.run()
    )
  ),
  Vector(
    ResponseRead.typed(
      Field[GetWorkflowExecutionHistoryResponse, Seq[Long]](
        _.getHistory.events.map(_.eventId)
      ),
      Cardinality.each,
      Vector.empty
    )
  )
)
val historySource = Recorded.history(
  Field[HistoryEvent, Option[
    io.temporal.api.history.v1.NexusOperationScheduledEventAttributes
  ]](
    _.attributes.nexusOperationScheduledEventAttributes
  )
)
val historyEvidence = Evidence.history(
  "fixture.typed.evidence.history",
  "history",
  "fixture.typed.source.history",
  historySource,
  Field[HistoryEvent, Long](_.eventId),
  Commitment.reported,
  fields = Vector(
    EvidenceField.typed(
      "scheduled",
      Field[HistoryEvent, String](
        _.getNexusOperationScheduledEventAttributes.requestId
      )
    )
  )
)
private def decision(value: DeliveryAdmissionDecision): TypedOperand[DeliveryAdmissionDecision] =
  Operand.enumValue(value)
val eventSource = Recorded.runEvent[InstructionOutcome](
  EventKind.instructionCompleted,
  "controller",
  "call",
  Operand.path(
    Operand.Projected.as[InstructionOutcome],
    Field[InstructionOutcome, String](_.getDeliveryAdmission.activityId)
  ),
  guard = Some(
    Condition.all(
      Condition.equal(
        Field[InstructionOutcome, DeliveryAdmissionDecision](
          _.getDeliveryAdmission.decision
        ),
        decision(
          DeliveryAdmissionDecision.DELIVERY_ADMISSION_DECISION_ADMITTED
        )
      ),
      Condition.greater(
        Field[InstructionOutcome, Int](_.getDeliveryAdmission.attempt),
        Operand.integer(0)
      )
    )
  )
)
val eventEvidence = Evidence.runEvent(
  "fixture.typed.evidence.event",
  "admitted",
  "fixture.typed.source.event",
  eventSource,
  Commitment.durable,
  fields = Vector(
    EvidenceField.typed(
      "delivery",
      Field[InstructionOutcome, Option[String]](
        _.deliveryAdmission.map(_.deliveryId)
      )
    )
  )
)
val runEventEvidence = Evidence.runEvent(
  "fixture.typed.evidence.run",
  "run",
  "fixture.typed.source.run",
  Recorded.runEvent[InstructionOutcome](
    EventKind.instructionCompleted,
    "controller",
    "call",
    Operand.runKey()
  ),
  Commitment.reported
)

private val correlation =
  Correlation("projection", "run", "operation", "observation", 1, 1, 1, 1, 1, 1)
private val roles = Vector(Role("endpoint", RoleKind.endpoint))
private val method = "/temporal.api.workflowservice.v1.WorkflowService/"
private val oldCall = Instruction.Rpc(
  "endpoint",
  method + "StartActivityExecution",
  Vector(
    Assignment("namespace", Operand.Run),
    Assignment("task_queue.name", Operand.Environment("queue"))
  ),
  Vector(ResponseRead("run_id", Cardinality.one, Vector(Target.Bind("run"))))
)
private val oldEvidence = Evidence(
  "fixture.typed.evidence.listed",
  "listed",
  "fixture.typed.source.listed",
  Recorded.Read(method + "ListWorkflowExecutions", "executions[*]"),
  "execution.workflow_id",
  Commitment.reported
)
private val oldSingleEvidence = Evidence(
  "fixture.typed.evidence.described",
  "described",
  "fixture.typed.source.described",
  Recorded
    .Single(method + "DescribeWorkflowExecution", "workflow_execution_info"),
  "execution.workflow_id",
  Commitment.reported
)
private val oldPoll = Instruction.Poll(
  "fixture.typed.evidence.listed",
  "endpoint",
  Vector(Assignment("namespace", Operand.Run)),
  Operand.Equal(
    Operand.Path(Operand.Projected, "execution.workflow_id"),
    Operand.Literal(ProtoValue.Text("workflow"))
  ),
  250
)
private val oldHistoryCall = Instruction.Rpc(
  "endpoint",
  method + "GetWorkflowExecutionHistory",
  Vector(Assignment("namespace", Operand.Run)),
  Vector(
    ResponseRead("history.events[*].event_id", Cardinality.each, Vector.empty)
  )
)
private val oldHistoryEvidence = Evidence(
  "fixture.typed.evidence.history",
  "history",
  "fixture.typed.source.history",
  Recorded.History("nexus_operation_scheduled_event_attributes"),
  "event_id",
  Commitment.reported,
  fields = Vector(
    EvidenceField(
      "scheduled",
      "attributes<nexus_operation_scheduled_event_attributes>.request_id"
    )
  )
)
private val oldEventEvidence = Evidence(
  "fixture.typed.evidence.event",
  "admitted",
  "fixture.typed.source.event",
  Recorded.RunEvent(
    EventKind.instructionCompleted,
    "controller",
    "call",
    Operand.Path(Operand.Projected, "delivery_admission.activity_id"),
    guard = Some(
      Operand.All(
        Operand.Equal(
          Operand.Path(Operand.Projected, "delivery_admission.decision"),
          Operand.Literal(
            ProtoValue.EnumName("DELIVERY_ADMISSION_DECISION_ADMITTED")
          )
        ),
        Operand.Greater(
          Operand.Path(Operand.Projected, "delivery_admission.attempt"),
          Operand.Literal(ProtoValue.Number(0))
        )
      )
    )
  ),
  "",
  Commitment.durable,
  fields = Vector(EvidenceField("delivery", "delivery_admission.delivery_id"))
)
private val oldRunEventEvidence = Evidence(
  "fixture.typed.evidence.run",
  "run",
  "fixture.typed.source.run",
  Recorded.RunEvent(
    EventKind.instructionCompleted,
    "controller",
    "call",
    Operand.Run
  ),
  "",
  Commitment.reported
)

val typedPayload = Proto[io.temporal.api.common.v1.Payload](
  ProtoField.typed(
    Field[io.temporal.api.common.v1.Payload, Map[String, com.google.protobuf.ByteString]](
      _.metadata
    ),
    ProtoValue.mapping(ProtoEntry.typed("encoding", ProtoValue.utf8("json/plain")))
  ),
  ProtoField.typed(
    Field[io.temporal.api.common.v1.Payload, com.google.protobuf.ByteString](_.data),
    ProtoValue.utf8("\"done\"")
  )
)
private val oldPayload = Proto(
  "temporal.api.common.v1.Payload",
  ProtoField("metadata", ProtoValue.Mapping(ProtoEntry("encoding", ProtoValue.Utf8("json/plain")))),
  ProtoField("data", ProtoValue.Utf8("\"done\""))
)
private def typedFailure(nonRetryable: Boolean) =
  Proto[io.temporal.api.failure.v1.Failure](
    ProtoField.typed(
      Field[io.temporal.api.failure.v1.Failure, String](_.message),
      ProtoValue.text("failed")
    ),
    ProtoField.typed(
      Field[io.temporal.api.failure.v1.Failure, io.temporal.api.failure.v1.ApplicationFailureInfo](
        _.getApplicationFailureInfo
      ),
      ProtoValue.message(
        Proto[io.temporal.api.failure.v1.ApplicationFailureInfo](
          ProtoField.typed(
            Field[io.temporal.api.failure.v1.ApplicationFailureInfo, String](_.`type`),
            ProtoValue.text("Expected")
          ),
          ProtoField.typed(
            Field[io.temporal.api.failure.v1.ApplicationFailureInfo, Boolean](_.nonRetryable),
            ProtoValue.flag(nonRetryable)
          )
        )
      )
    )
  )
private val oldFailure = Proto(
  "temporal.api.failure.v1.Failure",
  ProtoField("message", ProtoValue.Text("failed")),
  ProtoField(
    "application_failure_info",
    ProtoValue.Message(
      Proto(
        "temporal.api.failure.v1.ApplicationFailureInfo",
        ProtoField("type", ProtoValue.Text("Expected")),
        ProtoField("non_retryable", ProtoValue.Flag(true))
      )
    )
  )
)
private def scheduleAttributes(
    fields: Vector[
      TypedProtoField[io.temporal.api.command.v1.ScheduleNexusOperationCommandAttributes, ?]
    ]
) =
  Proto[io.temporal.api.command.v1.ScheduleNexusOperationCommandAttributes](fields*)
private def seconds(value: Long) = ProtoValue.number(value)
private def typedCommand(value: io.temporal.api.enums.v1.CommandType) =
  Proto[io.temporal.api.command.v1.Command](
    ProtoField.typed(
      Field[io.temporal.api.command.v1.Command, io.temporal.api.enums.v1.CommandType](
        _.commandType
      ),
      ProtoValue.enumValue(value)
    ),
    ProtoField.typed(
      Field[
        io.temporal.api.command.v1.Command,
        io.temporal.api.command.v1.ScheduleNexusOperationCommandAttributes
      ](
        _.getScheduleNexusOperationCommandAttributes
      ),
      ProtoValue.message(
        scheduleAttributes(
          Vector(
            ProtoField.typed(
              Field[io.temporal.api.command.v1.ScheduleNexusOperationCommandAttributes, String](
                _.endpoint
              ),
              ProtoValue.roleId("endpoint")
            ),
            ProtoField.typed(
              Field[
                io.temporal.api.command.v1.ScheduleNexusOperationCommandAttributes,
                com.google.protobuf.duration.Duration
              ](_.getScheduleToStartTimeout),
              ProtoValue.message(
                Proto[com.google.protobuf.duration.Duration](
                  ProtoField.typed(
                    Field[com.google.protobuf.duration.Duration, Long](_.seconds),
                    seconds(2L)
                  )
                )
              )
            )
          )
        )
      )
    )
  )
private val oldCommand = Proto(
  "temporal.api.command.v1.Command",
  ProtoField("command_type", ProtoValue.EnumName("COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION")),
  ProtoField(
    "schedule_nexus_operation_command_attributes",
    ProtoValue.Message(
      Proto(
        "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes",
        ProtoField("endpoint", ProtoValue.RoleId("endpoint")),
        ProtoField(
          "schedule_to_start_timeout",
          ProtoValue.Message(
            Proto(
              "google.protobuf.Duration",
              ProtoField("seconds", ProtoValue.Number(2))
            )
          )
        )
      )
    )
  )
)
private def typedResponse(
    variant: TypedProtoField[io.temporal.api.nexus.v1.StartOperationResponse, ?]
) = Proto[io.temporal.api.nexus.v1.StartOperationResponse](variant)
private val responseVariant = ProtoField.typed(
  Field[
    io.temporal.api.nexus.v1.StartOperationResponse,
    io.temporal.api.nexus.v1.StartOperationResponse.Async
  ](_.getAsyncSuccess),
  ProtoValue.message(Proto[io.temporal.api.nexus.v1.StartOperationResponse.Async]())
)
private val oldResponse = Proto(
  "temporal.api.nexus.v1.StartOperationResponse",
  ProtoField(
    "async_success",
    ProtoValue.Message(Proto("temporal.api.nexus.v1.StartOperationResponse.Async"))
  )
)

val typedRealization = Realization(
  "same",
  typedMachine,
  "producer",
  "v1",
  roles,
  correlation,
  Vector(
    Script(
      "controller",
      Activation.Controller,
      Vector(
        Item(command = Some(Command("call", call))),
        Item(command = Some(Command("poll", poll))),
        Item(command = Some(Command("history", historyCall))),
        Item(command = Some(Command("payload", Instruction.NexusReply(typedPayload)))),
        Item(command = Some(Command("failure", Instruction.AttemptFailure(typedFailure(true))))),
        Item(command =
          Some(
            Command(
              "command",
              Instruction.WorkflowCommand(
                typedCommand(
                  io.temporal.api.enums.v1.CommandType.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION
                )
              )
            )
          )
        ),
        Item(command =
          Some(
            Command(
              "response",
              Instruction.NexusReply(
                typedResponse(responseVariant)
              )
            )
          )
        )
      )
    )
  ),
  observations = Vector(Observed[CorrelatedEvidence]("observed")),
  evidence = Vector(
    evidence,
    singleEvidence,
    historyEvidence,
    eventEvidence,
    runEventEvidence
  )
)

val oldRealization = Realization(
  "same",
  typedMachine,
  "producer",
  "v1",
  roles,
  correlation,
  Vector(
    Script(
      "controller",
      Activation.Controller,
      Vector(
        Item(command = Some(Command("call", oldCall))),
        Item(command = Some(Command("poll", oldPoll))),
        Item(command = Some(Command("history", oldHistoryCall))),
        Item(command = Some(Command("payload", Instruction.NexusReply(oldPayload)))),
        Item(command = Some(Command("failure", Instruction.AttemptFailure(oldFailure)))),
        Item(command = Some(Command("command", Instruction.WorkflowCommand(oldCommand)))),
        Item(command = Some(Command("response", Instruction.NexusReply(oldResponse))))
      )
    )
  ),
  observations =
    Vector(Observed("observed", "temporal.server.api.testpilot.v1.CorrelatedEvidence")),
  evidence = Vector(
    oldEvidence,
    oldSingleEvidence,
    oldHistoryEvidence,
    oldEventEvidence,
    oldRunEventEvidence
  )
)

val mapped = Recorded.read(
  WorkflowServiceGrpc.METHOD_LIST_WORKFLOW_EXECUTIONS,
  Field[ListWorkflowExecutionsResponse, Seq[WorkflowExecution]](
    _.executions.map(_.getExecution)
  )
)
val mappedEvidence = Evidence.read(
  "fixture.typed.evidence.mapped",
  "mapped",
  "fixture.typed.source.mapped",
  mapped,
  Field[WorkflowExecution, String](_.workflowId),
  Commitment.reported
)
val invalidMappedRealization = Realization(
  "invalidMapped",
  typedMachine,
  "producer",
  "v1",
  roles,
  correlation,
  Vector.empty,
  evidence = Vector(mappedEvidence)
)

val unknownEnumRealization = Realization(
  "unknownEnum",
  typedMachine,
  "producer",
  "v1",
  roles,
  correlation,
  Vector(
    Script(
      "controller",
      Activation.Controller,
      Vector(
        Item(command =
          Some(
            Command(
              "unknown",
              Instruction.WorkflowCommand(
                typedCommand(
                  io.temporal.api.enums.v1.CommandType.Unrecognized(999)
                )
              )
            )
          )
        )
      )
    )
  )
)

private def boundLongOperand(value: Long): TypedOperand[Long] = Operand.number(value)
val typedLongRealization = Realization(
  "long",
  typedMachine,
  "producer",
  "v1",
  roles,
  correlation,
  Vector(
    Script(
      "controller",
      Activation.Controller,
      Vector(
        Item(command =
          Some(
            Command(
              "long",
              Instruction.rpc("endpoint", WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION)(
                Vector(
                  Assignment.typed(
                    Field[StartActivityExecutionRequest, Long](_.getStartToCloseTimeout.seconds),
                    Operand.number(300L)
                  ),
                  Assignment.typed(
                    Field[StartActivityExecutionRequest, Long](_.getScheduleToStartTimeout.seconds),
                    boundLongOperand(2L)
                  )
                ),
                Vector.empty
              )
            )
          )
        )
      )
    )
  )
)
val oldLongRealization = Realization(
  "long",
  typedMachine,
  "producer",
  "v1",
  roles,
  correlation,
  Vector(
    Script(
      "controller",
      Activation.Controller,
      Vector(
        Item(command =
          Some(
            Command(
              "long",
              Instruction.Rpc(
                "endpoint",
                method + "StartActivityExecution",
                Vector(
                  Assignment(
                    "start_to_close_timeout.seconds",
                    Operand.Literal(ProtoValue.Number(300))
                  ),
                  Assignment(
                    "schedule_to_start_timeout.seconds",
                    Operand.Literal(ProtoValue.Number(2))
                  )
                )
              )
            )
          )
        )
      )
    )
  )
)
