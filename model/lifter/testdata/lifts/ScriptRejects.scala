// Script helpers and kit declarations the lifter must refuse, each at the line that declares it
// (fn-112.9). The lifter's tests lift them with the other rejected declarations and compare the
// diagnostics with expected/rejects.txt.
package fixture.scriptrejects

import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.Fault
import temporal.realize.*
import temporal.standaloneactivity.{
  activity,
  activityProtocol,
  control,
  Control,
  Inputs,
  ProtocolFact,
  Timeout
}
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.ActivityExecutionStatus

given Family = Family("fixture.scriptrejects")

private def realizing(items: Item*) = temporalRealization(
  machine = activityProtocol,
  operation = activity,
  roles = Vector(workflowService, taskQueue),
  scripts = Vector(controller(items*)),
  evidence = Vector.empty
)

private val stopWorker = Fault(taskQueue, FaultKind.workerStop)

/** A command no val declares has no name. */
val unnamedCommand: Realization = realizing(always(Fault(taskQueue, FaultKind.workerStop)))

/** A `perform` that binds no class lands nowhere. */
val performsNothing: Realization = realizing(perform())

/** An `onPath` that names no class is carried by no Case. */
val onNoPath: Realization = realizing(onPath()(stopWorker))

private val listed: StatusTable[ActivityExecutionStatus] = statusTable(
  ProtocolFact.statusPaused -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED
)

private val twice: StatusTable[ActivityExecutionStatus] = statusTable(
  ProtocolFact.statusPaused -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED,
  ProtocolFact.statusPaused -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED
)

private val described = Evidence.read(
  id = evidenceId(ProtocolFact.statusCompleted),
  records = ProtocolFact.statusCompleted,
  source = sourceId(ProtocolFact.statusCompleted),
  from = Recorded.single(
    METHOD_DESCRIBE_ACTIVITY_EXECUTION,
    Field[
      io.temporal.api.workflowservice.v1.DescribeActivityExecutionResponse,
      io.temporal.api.activity.v1.ActivityExecutionInfo
    ](_.getInfo)
  ),
  operation = Field[io.temporal.api.activity.v1.ActivityExecutionInfo, String](_.activityId),
  commitment = Commitment.reported
)

private def awaitStatus(table: StatusTable[ActivityExecutionStatus], fact: Fact) =
  await(described, workflowService)(
    Condition.equal(
      Field[io.temporal.api.activity.v1.ActivityExecutionInfo, ActivityExecutionStatus](_.status),
      Operand.enumValue(table(fact))
    )
  ) {
    field(_.activityId) := run
  }

private val awaitCompleted = awaitStatus(listed, ProtocolFact.statusCompleted)

/** A status the table lists no value for. */
val unlistedStatus: Realization = realizing(always(awaitCompleted))

private val awaitPaused = awaitStatus(twice, ProtocolFact.statusPaused)

/** A table that lists one fact twice. */
val statusTwice: Realization = realizing(always(awaitPaused))

private val inputInScope = rpc(workflowService, METHOD_PAUSE_ACTIVITY_EXECUTION) {
  Inputs.scheduleToStart := Timeout.expires
}

/** A line of a request scope that assigns no field of the request. */
val notAField: Realization = realizing(perform(control(Control.pause) -> inputInScope))
