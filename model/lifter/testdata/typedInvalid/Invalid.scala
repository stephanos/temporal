//> using scala 3.9.0
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:imports
//> using jar ../../../gen/model-scala.jar
//> using jar ../../../gen/api-scalapb.jar
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
package fixture.typedInvalid

import io.temporal.api.workflowservice.v1.{
  ListWorkflowExecutionsRequest,
  ListWorkflowExecutionsResponse,
  StartActivityExecutionRequest,
  StartActivityExecutionResponse,
  WorkflowServiceGrpc
}
import io.temporal.api.workflow.v1.WorkflowExecutionInfo
import io.temporal.api.history.v1.HistoryEvent
import temporal.server.api.testpilot.v1.InstructionOutcome
import umpire.*
import umpire.realize.*

val nonMessage = action("bad", Party("caller")).schema[String]
val wrongRequest = Instruction.rpc(
  "endpoint",
  WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
)(
  Vector(
    Assignment.typed(
      Field[StartActivityExecutionResponse, String](_.runId),
      Operand.run()
    )
  ),
  Vector.empty
)
val wrongResponse =
  Instruction.rpc(
    "endpoint",
    WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
  )(
    Vector.empty,
    Vector(
      ResponseRead.typed(
        Field[StartActivityExecutionRequest, String](_.namespace),
        Cardinality.one,
        Vector.empty
      )
    )
  )
val listed = Recorded.read(
  WorkflowServiceGrpc.METHOD_LIST_WORKFLOW_EXECUTIONS,
  Field[ListWorkflowExecutionsResponse, Seq[WorkflowExecutionInfo]](
    _.executions
  )
)
val evidence = Evidence.read(
  "evidence",
  "listed",
  "source",
  listed,
  Field[WorkflowExecutionInfo, String](_.getExecution.workflowId),
  Commitment.reported
)
val wrongPollRequest = Instruction.poll(evidence, "endpoint")(
  Vector(
    Assignment.typed(
      Field[StartActivityExecutionResponse, String](_.runId),
      Operand.run()
    )
  ),
  Condition.present(
    Field[WorkflowExecutionInfo, String](_.getExecution.workflowId)
  )
)
val wrongPollProjection = Instruction.poll(evidence, "endpoint")(
  Vector(
    Assignment.typed(
      Field[ListWorkflowExecutionsRequest, String](_.namespace),
      Operand.run()
    )
  ),
  Condition.present(Field[StartActivityExecutionResponse, String](_.runId))
)
val forgedRecorded =
  RecordedRef[
    StartActivityExecutionRequest,
    StartActivityExecutionResponse,
    WorkflowExecutionInfo
  ](
    Recorded.Read("method", "path")
  )
val forgedEvidence =
  EvidenceRef[StartActivityExecutionRequest, WorkflowExecutionInfo](
    Evidence(
      "e",
      "record",
      "source",
      Recorded.Read("method", "path"),
      "operation",
      Commitment.reported
    )
  )
val directWrongRpc = Instruction.TypedRpc(
  "endpoint",
  WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION,
  Vector(
    Assignment.typed(
      Field[StartActivityExecutionResponse, String](_.runId),
      Operand.run()
    )
  ),
  Vector.empty
)
val directWrongPoll = Instruction.TypedPoll(
  evidence,
  "endpoint",
  Vector(
    Assignment.typed(
      Field[StartActivityExecutionResponse, String](_.runId),
      Operand.run()
    )
  ),
  Condition.present(
    Field[WorkflowExecutionInfo, String](_.getExecution.workflowId)
  )
)
val directWrongRead = Recorded.TypedRead(
  WorkflowServiceGrpc.METHOD_LIST_WORKFLOW_EXECUTIONS,
  Field[StartActivityExecutionResponse, Seq[WorkflowExecutionInfo]](_ => Seq.empty)
)
val directWrongSingle = Recorded.TypedSingle(
  WorkflowServiceGrpc.METHOD_DESCRIBE_WORKFLOW_EXECUTION,
  Field[StartActivityExecutionRequest, WorkflowExecutionInfo](_ =>
    throw new IllegalStateException()
  )
)
val misspelledField = Field[StartActivityExecutionRequest, String](_.namespac)
val wrongFieldValue = Field[StartActivityExecutionRequest, Boolean](_.namespace)
val wrongLiteral = Assignment.typed(
  Field[StartActivityExecutionRequest, String](_.namespace),
  Operand.flag(true)
)
val wrongSymbolic = Assignment.typed(
  Field[StartActivityExecutionRequest, Boolean](_.requestEagerExecution),
  Operand.environment[String]("namespace")
)
val wrongOneof = Field[HistoryEvent, String](_.attributes.noSuchEventAttributes)
val historyKey = Operand.path(
  Operand.Projected.as[HistoryEvent],
  Field[HistoryEvent, Long](_.eventId)
)
val wrongRunEventKey = Recorded.runEvent[InstructionOutcome](
  EventKind.instructionCompleted,
  "controller",
  "call",
  historyKey
)
val forgedProjectedPath = new ProjectedPath[InstructionOutcome, String]()
val wrongPollRight = Condition.equal(
  Field[WorkflowExecutionInfo, String](_.getExecution.workflowId),
  Operand.path(
    Operand.Projected.as[StartActivityExecutionResponse],
    Field[StartActivityExecutionResponse, String](_.runId)
  )
)
val nonMessageObserved = Observed[String]("bad")
