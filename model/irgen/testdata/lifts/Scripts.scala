// The script helpers (model/umpire/realize/Scripts.scala) and the Temporal kit (model/temporal/realize)
// beside the core records they stand for.
//
// `helpers` writes a realization with the helpers and the kit: commands named after their vals,
// declarations referred to by value, a fact by its case, a status read from a table, a call extended
// with `withFields`, a read written with the kit's `await`, and a command written out under its id
// around a call. `records` writes the same realization as core records, every id written out.
// `sugaredRequest` and `coredRequest` declare one request both ways: `field(_.name) := operand`
// (model/temporal/realize/Syntax.scala) and its core form `Assignment.typed(Field[Req, V](_.name),
// operand)`, in the scope of the same call. The lifter's tests lift them and require each pair's IR
// to be equal apart from positions, ids and names.
package fixture.scripts

import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.Release, temporal.realize.WorkerInstruction.Fault
import temporal.realize.*
import temporal.features.standaloneactivity.{activity, caller, worker, Control, ProtocolFact}
import temporal.features.standaloneactivity.system.ActivityProtocol as activityProtocol
import io.temporal.api.workflowservice.v1.*
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.api.enums.v1.ActivityExecutionStatus
import io.temporal.api.activity.v1.ActivityExecutionInfo
import temporal.server.api.testpilot.v1.{
  CorrelatedEvidence,
  InstructionOutcome,
  InstructionOutcomeStatus
}

given Family = Family("fixture.scripts")

// ### With the helpers and the kit

private val statusPaused = Evidence.read(
  id = evidenceId(ProtocolFact.statusPaused),
  records = ProtocolFact.statusPaused,
  source = sourceId(ProtocolFact.statusPaused),
  from = Recorded.single(
    METHOD_DESCRIBE_ACTIVITY_EXECUTION,
    Field[DescribeActivityExecutionResponse, ActivityExecutionInfo](_.getInfo)
  ),
  operation = Field[ActivityExecutionInfo, String](_.activityId),
  commitment = Commitment.reported
)

private val statuses = statusTable(
  ProtocolFact.statusPaused -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED,
  ProtocolFact.statusTimedOut -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_TIMED_OUT
)

private val stopWorker = Fault(taskQueue, FaultKind.workerStop)

private val pauseActivity = rpc(workflowService, METHOD_PAUSE_ACTIVITY_EXECUTION) {
  field(_.namespace) := workerNamespace
  field(_.activityId) := run
}

private val startActivity = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {
  field(_.namespace) := workerNamespace
  field(_.activityId) := run
}

private val awaitPaused = await(statusPaused, workflowService)(
  Condition.equal(
    Field[ActivityExecutionInfo, ActivityExecutionStatus](_.status),
    Operand.enumValue(statuses(ProtocolFact.statusPaused))
  )
) {
  field(_.namespace) := workerNamespace
  field(_.activityId) := run
}

private val dispatchHold =
  Actuator("hold-dispatch", ControlKind.HoldDispatched(worker.attemptStart), taskQueue)

private val releaseDispatch =
  command(Release(dispatchHold), regardless = true, closes = Vector(statusPaused))

val helpers: Realization = temporalRealization(
  machine = activityProtocol,
  operation = activity,
  roles = Vector(workflowService, caseWorker, taskQueue),
  scripts = Vector(
    controller(
      perform(caller.control(Control.pause) -> pauseActivity),
      onPath(caller.control(Control.pause))(awaitPaused),
      everyCase(startActivity.withFields {
        field(_.getStartToCloseTimeout.seconds) := unreachedDeadline
      }),
      perform(
        worker.attemptStart -> releaseDispatch,
        caller.control(Control.terminate) -> stopWorker
      ),
      // A command written out under its own id, around a call written in its scope.
      perform(
        caller.control(Control.unpause) -> Command(
          "unpause-written-out",
          rpc(workflowService, METHOD_UNPAUSE_ACTIVITY_EXECUTION) {
            field(_.namespace) := workerNamespace
          }
        )
      )
    )
  ),
  evidence = Vector(statusPaused, answered(ProtocolFact.statusScheduled, startActivity)),
  controls = Vector(dispatchHold)
)

// ### As core records

private val describedCore = Evidence.read(
  id = "fixture.scripts.evidence.statusPaused",
  records = "statusPaused",
  source = "fixture.scripts.source.statusPaused",
  from = Recorded.single(
    WorkflowServiceGrpc.METHOD_DESCRIBE_ACTIVITY_EXECUTION,
    Field[DescribeActivityExecutionResponse, ActivityExecutionInfo](_.getInfo)
  ),
  operation = Field[ActivityExecutionInfo, String](_.activityId),
  commitment = Commitment.reported
)

val records: Realization = Realization(
  machine = activityProtocol,
  producer = "fixture.scripts.testpilot",
  producerVersion = "1",
  roles = Vector(
    Role("temporal.workflow-service", RoleKind.endpoint),
    Role("temporal.worker", RoleKind.worker, namespace = "temporal.worker.namespace"),
    Role(
      "temporal.task-queue",
      RoleKind.taskQueue,
      namespace = "temporal.worker.namespace",
      resource = "temporal.task-queue.resource"
    )
  ),
  correlation = Correlation(
    projection = "fixture.scripts.projection",
    run = "fixture.scripts.scope.run",
    operation = "fixture.scripts.scope.activity",
    observation = "correlated-evidence",
    events = 32,
    buffered = 16,
    keys = 8,
    support = 128,
    work = 1000000000,
    eventSize = 512
  ),
  scripts = Vector(
    Script(
      "controller",
      Activation.Controller,
      Vector(
        Item(performs =
          Vector(
            Performance(
              caller.control(Control.pause),
              Command(
                "pause-activity",
                Instruction.rpc(
                  "temporal.workflow-service",
                  WorkflowServiceGrpc.METHOD_PAUSE_ACTIVITY_EXECUTION
                )(
                  Vector(
                    Assignment.typed(
                      Field[PauseActivityExecutionRequest, String](_.namespace),
                      Operand.environment[String]("temporal.worker.namespace")
                    ),
                    Assignment.typed(
                      Field[PauseActivityExecutionRequest, String](_.activityId),
                      Operand.run()
                    )
                  ),
                  Vector.empty
                )
              )
            )
          )
        ),
        Item(
          command = Some(
            Command(
              "await-paused",
              Instruction.readUntil(describedCore, "temporal.workflow-service")(
                Vector(
                  Assignment.typed(
                    Field[DescribeActivityExecutionRequest, String](_.namespace),
                    Operand.environment[String]("temporal.worker.namespace")
                  ),
                  Assignment.typed(
                    Field[DescribeActivityExecutionRequest, String](_.activityId),
                    Operand.run()
                  )
                ),
                Condition.equal(
                  Field[ActivityExecutionInfo, ActivityExecutionStatus](_.status),
                  Operand.enumValue(ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_PAUSED)
                ),
                0
              )
            )
          ),
          when = Vector(caller.control(Control.pause))
        ),
        Item(command =
          Some(
            Command(
              "start-activity",
              Instruction.rpc(
                "temporal.workflow-service",
                WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
              )(
                Vector(
                  Assignment.typed(
                    Field[StartActivityExecutionRequest, String](_.namespace),
                    Operand.environment[String]("temporal.worker.namespace")
                  ),
                  Assignment.typed(
                    Field[StartActivityExecutionRequest, String](_.activityId),
                    Operand.run()
                  ),
                  Assignment.typed(
                    Field[StartActivityExecutionRequest, Long](_.getStartToCloseTimeout.seconds),
                    Operand.number(300L)
                  )
                ),
                Vector.empty
              )
            )
          )
        ),
        Item(performs =
          Vector(
            Performance(
              worker.attemptStart,
              Command(
                "release-dispatch",
                Instruction.Release("hold-dispatch"),
                regardless = true,
                closes = Vector("fixture.scripts.evidence.statusPaused")
              )
            ),
            Performance(
              caller.control(Control.terminate),
              Command("stop-worker", Fault("temporal.task-queue", FaultKind.workerStop))
            )
          )
        ),
        Item(performs =
          Vector(
            Performance(
              caller.control(Control.unpause),
              Command(
                "unpause-written-out",
                Instruction.rpc(
                  "temporal.workflow-service",
                  WorkflowServiceGrpc.METHOD_UNPAUSE_ACTIVITY_EXECUTION
                )(
                  Vector(
                    Assignment.typed(
                      Field[UnpauseActivityExecutionRequest, String](_.namespace),
                      Operand.environment[String]("temporal.worker.namespace")
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
  ),
  observations = Vector(Observed[CorrelatedEvidence]("correlated-evidence")),
  evidence = Vector(
    describedCore,
    Evidence.runEvent(
      id = "fixture.scripts.evidence.statusScheduled",
      records = "statusScheduled",
      source = "fixture.scripts.source.record",
      from = Recorded.runEvent[InstructionOutcome](
        EventKind.instructionCompleted,
        "controller",
        "start-activity",
        key = Operand.runKey(),
        guard = Some(
          Condition.equal(
            Field[InstructionOutcome, InstructionOutcomeStatus](_.status),
            Operand.enumValue(InstructionOutcomeStatus.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)
          )
        )
      ),
      commitment = Commitment.reported
    )
  ),
  controls = Vector(
    Actuator(
      "hold-dispatch",
      ControlKind.HoldDispatched(worker.attemptStart),
      role = "temporal.task-queue"
    )
  ),
  cleanup = "cleanup",
  behavior = Some(temporalBehavior)
)

// ### One request field, both ways

private def oneRequest(start: Instruction) = temporalRealization(
  machine = activityProtocol,
  operation = activity,
  roles = Vector(workflowService),
  scripts = Vector(controller(everyCase(start))),
  evidence = Vector.empty
)

private val startSugared = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {
  field(_.getTaskQueue.name) := taskQueueName
  field(_.getScheduleToStartTimeout.seconds) := deadline
}

private val startCored = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {
  Assignment.typed(Field[StartActivityExecutionRequest, String](_.getTaskQueue.name), taskQueueName)
  Assignment.typed(
    Field[StartActivityExecutionRequest, Long](_.getScheduleToStartTimeout.seconds),
    deadline
  )
}

val sugaredRequest: Realization = oneRequest(startSugared)

val coredRequest: Realization = oneRequest(startCored)
