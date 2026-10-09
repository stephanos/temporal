// The Temporal kit's literal and call scopes (model/temporal/realize/Syntax.scala) beside the core
// records they stand for (fn-133.1). `literalSugared` writes protobuf literals with
// `proto[M] { … }`, a call's response read with `read(…).into(…)`, a call that extends another's
// request with `extended`, and a message written out for a request field; `literalCored` writes each
// in its core form: `Proto[M](ProtoField.typed(…))`, `Instruction.rpc(…)(assign, reads)` and the
// assignment of each field the message sets. The lifter's tests lift both and require one IR of the
// two, but for positions, ids and names.
package fixture.literals

import framework.realize.*
import temporal.realize.*
import temporal.realize.WorkerInstruction.{AttemptFailure, NexusReply, WorkflowCommand}
import temporal.features.activity.standalone.activity
import temporal.features.activity.standalone.system.ActivitySystem as activitySystem
import io.temporal.api.command.v1.{Command as ApiCommand, ScheduleNexusOperationCommandAttributes}
import io.temporal.api.common.v1.Payload
import io.temporal.api.enums.v1.{CommandType, NexusHandlerErrorRetryBehavior}
import io.temporal.api.failure.v1.{ApplicationFailureInfo, Failure}
import io.temporal.api.history.v1.HistoryEvent
import io.temporal.api.nexus.v1.{Failure as NexusFailure, HandlerError, StartOperationResponse}
import io.temporal.api.workflowservice.v1.{
  GetWorkflowExecutionHistoryRequest,
  GetWorkflowExecutionHistoryResponse,
  StartActivityExecutionRequest,
  WorkflowServiceGrpc
}
import com.google.protobuf.ByteString
import com.google.protobuf.duration.Duration

private def literals(items: Item*) = temporalRealization(
  machine = activitySystem,
  operation = activity,
  roles = Vector(workflowService, nexusEndpoint),
  scripts = Vector(controller(items*)),
  evidence = Vector.empty,
  observations = Vector(historyEvent, correlated)
)

private val historyEvent = Observed[HistoryEvent]("history-event")

private val events = Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]](
  _.getHistory.events.map(event => event)
)

// ### With the scopes

// A scope passed in, applied where the literal writes it.
private def scheduling(deadlines: ProtoScope[ScheduleNexusOperationCommandAttributes] ?=> Unit) =
  WorkflowCommand(proto[ApiCommand] {
    field(_.commandType) := CommandType.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION
    field(_.getScheduleNexusOperationCommandAttributes) {
      field(_.endpoint) := nexusEndpoint
      field(_.service) := perCase("service")
      field(_.getInput) {
        field(_.metadata) := Map("encoding" -> "json/plain")
        field(_.data) := "\"request\""
      }
      deadlines
    }
  })

private val replied = NexusReply(proto[HandlerError] {
  field(_.errorType) := "INTERNAL"
  field(_.getFailure)(field(_.message) := "handler error")
  field(
    _.retryBehavior
  ) := NexusHandlerErrorRetryBehavior.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE
})

private val asynchronous = NexusReply(proto[StartOperationResponse](field(_.getAsyncSuccess) {}))

private val failed = AttemptFailure(proto[Failure] {
  field(_.message) := "attempt failed"
  field(_.getApplicationFailureInfo) := proto[ApplicationFailureInfo] {
    field(_.`type`) := "AttemptFailed"
    field(_.nonRetryable) := !true
  }
})

private val historyRead =
  rpc(workflowService, WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY) {
    field(_.namespace) := workerNamespace
    field(_.maximumPageSize) := Operand.integer(64)
  }

private val historyReadBack = historyRead.extended {
  read(events, Cardinality.each).into(historyEvent, Target.Lift(correlated.id))
}

private val started =
  rpc(workflowService, WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION) {
    field(_.getStartToCloseTimeout) := duration(5)
  }

val literalSugared: Realization = literals(
  everyCase(Command("scheduled", scheduling(()))),
  everyCase(Command("scheduled-by", scheduling(field(_.getScheduleToStartTimeout) := duration(2)))),
  everyCase(Command("replied", replied)),
  everyCase(Command("asynchronous", asynchronous)),
  everyCase(Command("failed", failed)),
  everyCase(Command("history-read-back", historyReadBack)),
  everyCase(Command("started", started))
)

// ### In the core forms

private def schedulingCore(
    deadlines: TypedProtoField[ScheduleNexusOperationCommandAttributes, ?]*
) = WorkflowCommand(
  Proto[ApiCommand](
    ProtoField.typed(
      Field[ApiCommand, CommandType](_.commandType),
      ProtoValue.enumValue(CommandType.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION)
    ),
    ProtoField.typed(
      Field[ApiCommand, ScheduleNexusOperationCommandAttributes](
        _.getScheduleNexusOperationCommandAttributes
      ),
      ProtoValue.message(
        Proto[ScheduleNexusOperationCommandAttributes](
          (Vector(
            ProtoField.typed(
              Field[ScheduleNexusOperationCommandAttributes, String](_.endpoint),
              ProtoValue.roleId(nexusEndpoint)
            ),
            ProtoField.typed(
              Field[ScheduleNexusOperationCommandAttributes, String](_.service),
              ProtoValue.named(perCase("service"))
            ),
            ProtoField.typed(
              Field[ScheduleNexusOperationCommandAttributes, Payload](_.getInput),
              ProtoValue.message(
                Proto[Payload](
                  ProtoField.typed(
                    Field[Payload, Map[String, ByteString]](_.metadata),
                    ProtoValue.mapping(ProtoEntry.typed("encoding", ProtoValue.utf8("json/plain")))
                  ),
                  ProtoField.typed(
                    Field[Payload, ByteString](_.data),
                    ProtoValue.utf8("\"request\"")
                  )
                )
              )
            )
          ) ++ deadlines)*
        )
      )
    )
  )
)

private val repliedCore = NexusReply(
  Proto[HandlerError](
    ProtoField.typed(Field[HandlerError, String](_.errorType), ProtoValue.text("INTERNAL")),
    ProtoField.typed(
      Field[HandlerError, NexusFailure](_.getFailure),
      ProtoValue.message(
        Proto[NexusFailure](
          ProtoField.typed(Field[NexusFailure, String](_.message), ProtoValue.text("handler error"))
        )
      )
    ),
    ProtoField.typed(
      Field[HandlerError, NexusHandlerErrorRetryBehavior](_.retryBehavior),
      ProtoValue.enumValue(
        NexusHandlerErrorRetryBehavior.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE
      )
    )
  )
)

private val asynchronousCore = NexusReply(
  Proto[StartOperationResponse](
    ProtoField.typed(
      Field[StartOperationResponse, StartOperationResponse.Async](_.getAsyncSuccess),
      ProtoValue.message(Proto[StartOperationResponse.Async]())
    )
  )
)

private val failedCore = AttemptFailure(
  Proto[Failure](
    ProtoField.typed(Field[Failure, String](_.message), ProtoValue.text("attempt failed")),
    ProtoField.typed(
      Field[Failure, ApplicationFailureInfo](_.getApplicationFailureInfo),
      ProtoValue.message(
        Proto[ApplicationFailureInfo](
          ProtoField.typed(
            Field[ApplicationFailureInfo, String](_.`type`),
            ProtoValue.text("AttemptFailed")
          ),
          ProtoField.typed(
            Field[ApplicationFailureInfo, Boolean](_.nonRetryable),
            ProtoValue.flag(false)
          )
        )
      )
    )
  )
)

private val historyReadBackCore =
  Instruction.rpc(workflowService, WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY)(
    Vector(
      Assignment.typed(
        Field[GetWorkflowExecutionHistoryRequest, String](_.namespace),
        workerNamespace
      ),
      Assignment.typed(
        Field[GetWorkflowExecutionHistoryRequest, Int](_.maximumPageSize),
        Operand.integer(64)
      )
    ),
    Vector(
      ResponseRead.typed(
        events,
        Cardinality.each,
        Vector(Target.Observe(historyEvent.id), Target.Lift(correlated.id))
      )
    )
  )

private val startedCore =
  Instruction.rpc(workflowService, WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION)(
    Vector(
      Assignment.typed(
        Field[StartActivityExecutionRequest, Long](_.getStartToCloseTimeout.seconds),
        Operand.number(5)
      )
    ),
    Vector.empty
  )

val literalCored: Realization = literals(
  everyCase(Command("scheduled", schedulingCore())),
  everyCase(
    Command(
      "scheduled-by",
      schedulingCore(
        ProtoField.typed(
          Field[ScheduleNexusOperationCommandAttributes, Duration](_.getScheduleToStartTimeout),
          ProtoValue.message(
            Proto[Duration](
              ProtoField.typed(Field[Duration, Long](_.seconds), ProtoValue.number(2))
            )
          )
        )
      )
    )
  ),
  everyCase(Command("replied", repliedCore)),
  everyCase(Command("asynchronous", asynchronousCore)),
  everyCase(Command("failed", failedCore)),
  everyCase(Command("history-read-back", historyReadBackCore)),
  everyCase(Command("started", startedCore))
)
