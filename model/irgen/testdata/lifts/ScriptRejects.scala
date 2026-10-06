// Script helpers and kit declarations the lifter must refuse, each at the line that declares it
// (fn-112.9). The lifter's tests lift them with the other rejected declarations and compare the
// diagnostics with expected/rejects.txt.
package fixture.scriptrejects

import umpire.*
import umpire.realize.*
import temporal.realize.WorkerInstruction.{AttemptFailure, Fault}
import temporal.realize.*
import temporal.features.activity.Timeout
import temporal.features.activity.standalone.{activity, client, scheduleToStart, Control}
import temporal.features.activity.standalone.system.Fact as ActivityFact
import temporal.features.activity.standalone.system.ActivitySystem as activitySystem
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.ActivityExecutionStatus

private def realizing(items: Item*) = temporalRealization(
  machine = activitySystem,
  operation = activity,
  roles = Vector(workflowService, taskQueue),
  scripts = Vector(controller(items*)),
  evidence = Vector.empty
)

private val stopWorker = Fault(taskQueue, FaultKind.workerStop)

// A command no val declares has no name.
val unnamedCommand: Realization = realizing(everyCase(Fault(taskQueue, FaultKind.workerStop)))

// A `perform` that binds no class lands nowhere.
val performsNothing: Realization = realizing(perform())

// An `onPath` that names no class is carried by no Case.
val onNoPath: Realization = realizing(onPath()(stopWorker))

private val listed: StatusTable[ActivityExecutionStatus] = statusTable(
  ActivityFact.statusPaused -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED
)

private val twice: StatusTable[ActivityExecutionStatus] = statusTable(
  ActivityFact.statusPaused -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED,
  ActivityFact.statusPaused -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED
)

private val described = Evidence.read(
  id = evidenceId(ActivityFact.statusCompleted),
  records = ActivityFact.statusCompleted,
  source = sourceId(ActivityFact.statusCompleted),
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

private val awaitCompleted = awaitStatus(listed, ActivityFact.statusCompleted)

// A status the table lists no value for.
val unlistedStatus: Realization = realizing(everyCase(awaitCompleted))

private val awaitPaused = awaitStatus(twice, ActivityFact.statusPaused)

// A table that lists one fact twice.
val statusTwice: Realization = realizing(everyCase(awaitPaused))

private val inputInScope = rpc(workflowService, METHOD_PAUSE_ACTIVITY_EXECUTION) {
  scheduleToStart := Timeout.expires
}

// A line of a request scope that assigns no field of the request.
val notAField: Realization = realizing(perform(client.control(Control.pause) -> inputInScope))

private val inputInPoll = await(described, workflowService)(
  Condition.equal(
    Field[io.temporal.api.activity.v1.ActivityExecutionInfo, ActivityExecutionStatus](_.status),
    Operand.enumValue(ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED)
  )
) {
  scheduleToStart := Timeout.expires
}

// A line of a `readUntil`'s request scope that assigns no field of the request.
val notAPolledField: Realization = realizing(everyCase(inputInPoll))

// Evidence of a case of another enum than the facts the machine records.
val foreignFact: Realization = temporalRealization(
  machine = activitySystem,
  operation = activity,
  roles = Vector(workflowService, taskQueue),
  scripts = Vector(controller(everyCase(stopWorker))),
  evidence = Vector(answered(Control.pause, stopWorker))
)

private val failedTwice = AttemptFailure(proto[io.temporal.api.failure.v1.Failure] {
  field(_.message) := "attempt failed"
  field(_.message) := "attempt failed again"
})

// A protobuf literal that sets one field twice (fn-133.1).
val literalTwice: Realization = realizing(everyCase(failedTwice))
