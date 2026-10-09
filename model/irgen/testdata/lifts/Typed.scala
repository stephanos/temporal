// Typed selectors the live Models do not write: a scalar mapped out of a repeated field, an
// evidence field read through `Option.map`, a Long operand bound through a helper's parameter, a
// mapped read ending at a singular message, and an unknown generated enum value. The live IR
// (model/ir) pins schemas, methods, payloads, failures, commands and history evidence.
package fixture.typed

import io.temporal.api.command.v1.Command as WorkflowCommandProto
import io.temporal.api.common.v1.WorkflowExecution
import io.temporal.api.enums.v1.CommandType
import io.temporal.api.workflowservice.v1.{
  GetWorkflowExecutionHistoryRequest,
  GetWorkflowExecutionHistoryResponse,
  ListWorkflowExecutionsResponse,
  StartActivityExecutionRequest,
  StartActivityExecutionResponse,
  WorkflowServiceGrpc
}
import temporal.server.api.testpilot.v1.InstructionOutcome
import framework.*
import framework.realize.*, temporal.realize.{Role, RoleKind, WorkerInstruction}

val one = action(Actor("caller"))

enum State derives Finite:
  case idle

enum Result derives Finite:
  case accepted

def step(s: State): List[Step[State, Result, Nothing]] = List(
  Step(Result.accepted, s)
)

object Typed extends Machine[State, Result, Nothing]:
  val init = State.idle
  def end(state: State) = true

  object rules extends Bindings(one ~> step)

// The value reaches `Operand.number` through the parameter, not as a literal at the call.
private def seconds(value: Long): TypedOperand[Long] = Operand.number(value)

val call = Instruction.rpc(
  "endpoint",
  WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
)(
  Vector(
    Assignment.typed(
      Field[StartActivityExecutionRequest, Long](_.getHeartbeatTimeout.seconds),
      seconds(5L)
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

val historyCall = Instruction.rpc(
  "endpoint",
  WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY
)(
  Vector(
    Assignment.typed(
      Field[GetWorkflowExecutionHistoryRequest, String](_.getExecution.workflowId),
      Operand.text("workflow")
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

// `Option.map` selects into the optional message without indexing it as a repeated field.
val eventEvidence = Evidence.runEvent(
  "fixture.typed.evidence.event",
  "admitted",
  "fixture.typed.source.event",
  Recorded.runEvent[InstructionOutcome](
    EventKind.instructionCompleted,
    "controller",
    "call",
    Operand.runKey()
  ),
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

private val correlation =
  Correlation("projection", "run", "operation", "observation", 1, 1, 1, 1, 1, 1)
private val roles = Vector(Role("endpoint", RoleKind.endpoint))

val typedRealization = Realization(
  "same",
  Typed,
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
        Item(command = Some(Command("history", historyCall)))
      )
    )
  ),
  evidence = Vector(eventEvidence)
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
  Typed,
  "producer",
  "v1",
  roles,
  correlation,
  Vector.empty,
  evidence = Vector(mappedEvidence)
)

val unknownEnumRealization = Realization(
  "unknownEnum",
  Typed,
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
              WorkerInstruction.WorkflowCommand(
                Proto[WorkflowCommandProto](
                  ProtoField.typed(
                    Field[WorkflowCommandProto, CommandType](_.commandType),
                    // Widened past the compile-time refusal, so the lifter's is the one that runs.
                    ProtoValue.enumValue[CommandType, CommandType](CommandType.Unrecognized(999))
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
