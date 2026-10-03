// Realizations the lifter emits as written, and tools/umpire/lower lowers or refuses.
//
// `learnedRun` runs the Nexus caller's synchronous completion with a run id the start call binds and
// two branches that each read it. `pauseRace` is the activity specimen's proposed race scenario
// (specimens/activity.md, E8 and E10): it declares what Testpilot cannot run yet, a held delivery, a
// durable-commit observation and a machine with authored monitors, so lowering it names each gap
// instead of a Case. `errandRealization` runs a standalone activity whose first attempt fails and
// whose retry completes, on a machine whose facts a listing reports. `tallyRealization` declares
// evidence read from one message that keeps fields, one of them without its value, which no Case
// carries, and the Run's own record as evidence. The lifter's tests
// lift them and compare the IR with expected/realizations.json.
package fixture.realizations

import fixture.specimens.admission.{
  dispatch,
  scheduledEmpty,
  staleAdmission,
  three,
  AdmissionFact,
  Message as Dispatched
}
import temporal.nexuscaller.{handlerReply, nexusProtocol, schedule, Reply, Timeout}
import temporal.standaloneactivity.{attemptStart, control, Control as ActivityControl}
import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.*
import umpire.realize.Operand.*
import umpire.realize.ProtoValue.*
import io.temporal.api.workflowservice.v1.*
import io.temporal.api.history.v1.*
import io.temporal.api.activity.v1.{ActivityExecutionInfo, ActivityExecutionListInfo}
import io.temporal.api.common.v1.Payload
import io.temporal.api.command.v1.{Command as ApiCommand, ScheduleNexusOperationCommandAttributes}
import io.temporal.api.enums.v1.{ActivityExecutionStatus, CommandType, HistoryEventFilterType}
import io.temporal.api.failure.v1.{ApplicationFailureInfo, Failure as ApiFailure}
import io.temporal.api.nexus.v1.StartOperationResponse
import temporal.server.api.testpilot.v1.{
  CorrelatedEvidence,
  InstructionOutcome,
  InstructionOutcomeStatus
}

private val workflowService = "temporal.workflow-service"
private val workerRole = "temporal.worker"
private val taskQueue = "temporal.task-queue"
private val namespace = "temporal.worker.namespace"
private val queueResource = "temporal.task-queue.resource"
private val endpoint = "temporal.nexus-endpoint"

private val roles = Vector(
  Role(workflowService, RoleKind.endpoint),
  Role(workerRole, RoleKind.worker, namespace = namespace),
  Role(taskQueue, RoleKind.taskQueue, namespace = namespace, resource = queueResource)
)

// ### A learned run id, read by two branches

private val run = "workflow-run"
private val workflowType = Name("umpire-", fixture = true, suffix = "-workflow")

/** The workflow a read names: the run's id, and the run id the start call returned. */
private val startedRun = Vector(
  Assignment.typed(
    Field[GetWorkflowExecutionHistoryRequest, String](_.namespace),
    Operand.environment[String](namespace)
  ),
  Assignment.typed(
    Field[GetWorkflowExecutionHistoryRequest, String](_.getExecution.workflowId),
    Operand.run()
  ),
  Assignment.typed(
    Field[GetWorkflowExecutionHistoryRequest, String](_.getExecution.runId),
    Operand.learnedValue[String](run)
  )
)

private val scheduledEvidence = Evidence.read(
  id = "fixture.realizations.evidence.scheduled",
  records = "nexusOperationScheduled",
  source = "fixture.realizations.source.scheduled",
  from = Recorded.read(
    WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY,
    Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]](
      _.getHistory.events.map(event => event)
    )
  ),
  operation = Field[HistoryEvent, Long](_.eventId),
  commitment = Commitment.reported
)

private val completedEvidence = Evidence.history(
  id = "fixture.realizations.evidence.completed",
  records = "nexusOperationCompleted",
  source = "fixture.realizations.source.history",
  from = Recorded.history(
    Field[HistoryEvent, Option[NexusOperationCompletedEventAttributes]](
      _.attributes.nexusOperationCompletedEventAttributes
    )
  ),
  operation =
    Field[HistoryEvent, Long](_.getNexusOperationCompletedEventAttributes.scheduledEventId),
  commitment = Commitment.reported
)

private val start = Command(
  "start-workflow",
  Instruction.rpc(workflowService, WorkflowServiceGrpc.METHOD_START_WORKFLOW_EXECUTION)(
    Vector(
      Assignment.typed(
        Field[StartWorkflowExecutionRequest, String](_.namespace),
        Operand.environment[String](namespace)
      ),
      Assignment.typed(Field[StartWorkflowExecutionRequest, String](_.workflowId), Operand.run()),
      Assignment.typed(
        Field[StartWorkflowExecutionRequest, String](_.getWorkflowType.name),
        Operand.named(workflowType)
      ),
      Assignment.typed(
        Field[StartWorkflowExecutionRequest, String](_.getTaskQueue.name),
        Operand.environment[String](queueResource)
      ),
      Assignment.typed(Field[StartWorkflowExecutionRequest, String](_.requestId), Operand.run())
    ),
    Vector(
      ResponseRead.typed(
        Field[StartWorkflowExecutionResponse, String](_.runId),
        Cardinality.one,
        Vector(Target.Bind(run))
      )
    )
  )
)

/** One branch: polls the started run's history for the scheduled event. */
private val awaitScheduled = Command(
  "await-scheduled",
  Instruction.poll(scheduledEvidence, workflowService)(
    startedRun,
    Condition.present(
      Field[HistoryEvent, Option[NexusOperationScheduledEventAttributes]](
        _.attributes.nexusOperationScheduledEventAttributes
      )
    ),
    250
  ),
  after = Some(After("start-workflow"))
)

/** The other branch: waits for the started run to close. Neither branch waits for the other. */
private val awaitClose = Command(
  "await-close",
  Instruction.rpc(workflowService, WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY)(
    startedRun ++ Vector(
      Assignment.typed(
        Field[GetWorkflowExecutionHistoryRequest, Boolean](_.waitNewEvent),
        Operand.flag(true)
      ),
      Assignment.typed(
        Field[GetWorkflowExecutionHistoryRequest, HistoryEventFilterType](_.historyEventFilterType),
        Operand.enumValue(HistoryEventFilterType.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT)
      )
    ),
    Vector.empty
  ),
  after = Some(After("start-workflow"))
)

/** Reads the history once both branches are done. */
private val history = Command(
  "history",
  Instruction.rpc(workflowService, WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY)(
    startedRun,
    Vector(
      ResponseRead.typed(
        Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]](
          _.getHistory.events.map(event => event)
        ),
        Cardinality.each,
        Vector(Target.Observe("history-event"), Target.Lift("correlated-evidence"))
      )
    )
  ),
  after = Some(After("await-scheduled", "await-close"))
)

private val payload = Proto[Payload](
  ProtoField.typed(
    Field[Payload, Map[String, com.google.protobuf.ByteString]](_.metadata),
    ProtoValue.mapping(ProtoEntry.typed("encoding", ProtoValue.utf8("json/plain")))
  ),
  ProtoField.typed(
    Field[Payload, com.google.protobuf.ByteString](_.data),
    ProtoValue.utf8("\"done\"")
  )
)

val learnedRun: Realization = Realization(
  name = "learnedRun",
  machine = nexusProtocol,
  producer = "fixture.realizations",
  producerVersion = "1",
  roles = roles :+ Role(endpoint, RoleKind.endpoint, resource = "temporal.nexus-endpoint.resource"),
  correlation = Correlation(
    projection = "fixture.realizations.projection",
    run = "fixture.realizations.scope.run",
    operation = "fixture.realizations.scope.operation",
    observation = "correlated-evidence",
    events = 32,
    buffered = 16,
    keys = 8,
    support = 128,
    work = 1000000,
    eventSize = 512
  ),
  scripts = Vector(
    Script(
      "controller",
      Activation.Controller,
      Vector(
        Item(command = Some(start)),
        Item(command = Some(awaitScheduled)),
        Item(command = Some(awaitClose)),
        Item(command = Some(history))
      )
    ),
    Script(
      "workflow",
      Activation.Workflow(workflowType, workerRole, taskQueue),
      Vector(
        Item(performs =
          Vector(
            Performance(
              schedule(Timeout.unset, Timeout.unset, Timeout.unset),
              Command(
                "start-nexus-operation",
                WorkflowCommand(
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
                          ProtoField.typed(
                            Field[ScheduleNexusOperationCommandAttributes, String](_.endpoint),
                            ProtoValue.roleId(endpoint)
                          ),
                          ProtoField.typed(
                            Field[ScheduleNexusOperationCommandAttributes, String](_.service),
                            ProtoValue.text("fixture.service")
                          ),
                          ProtoField.typed(
                            Field[ScheduleNexusOperationCommandAttributes, String](_.operation),
                            ProtoValue.text("probe")
                          )
                        )
                      )
                    )
                  )
                )
              )
            )
          )
        ),
        Item(command =
          Some(
            Command(
              "await-nexus-operation",
              AwaitCommand("start-nexus-operation"),
              regardless = true
            )
          )
        ),
        Item(command =
          Some(Command("finish-workflow", Finish(Literal(Text("done"))), regardless = true))
        )
      )
    ),
    Script(
      "handler",
      Activation.NexusHandler("fixture.service", "probe", workerRole, taskQueue),
      Vector(
        Item(performs =
          Vector(
            Performance(
              handlerReply(Reply.syncSuccess),
              Command(
                "respond-sync",
                NexusReply(
                  Proto[StartOperationResponse](
                    ProtoField.typed(
                      Field[StartOperationResponse, StartOperationResponse.Sync](_.getSyncSuccess),
                      ProtoValue.message(
                        Proto[StartOperationResponse.Sync](
                          ProtoField.typed(
                            Field[StartOperationResponse.Sync, Payload](_.getPayload),
                            ProtoValue.message(payload)
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
      )
    )
  ),
  learned = Vector(Learned(run, LearnedKind.text)),
  observations = Vector(
    Observed[HistoryEvent]("history-event"),
    Observed[CorrelatedEvidence]("correlated-evidence")
  ),
  evidence = Vector(scheduledEvidence, completedEvidence),
  cleanup = "cleanup"
)

// ### The activity specimen's held race

/** The dispatch the race holds, as the channel the specimen proposes for it. */
val dispatchChannel: Channel[Dispatched] =
  channel[Dispatched](
    "dispatchChannel",
    capacity = 1,
    order = Order.unordered,
    loss = Loss.reliable
  )

private val activityRun = "activity-run"
private val holdDispatch = "hold-dispatch"

// The pinned path stops at the pause. The stale design's admission after it has two results, the
// message consumed and the message kept for redelivery, that record the same facts, so no kind of
// evidence tells them apart and a producer refuses a path through it as ambiguous.
private val pausedWhileQueued =
  staleAdmission.property("pausedWhileQueued") when control(ActivityControl.pause) holds
    (_.facts.contains(AdmissionFact.statusPaused))

private val heldRace = staleAdmission
  .scenario("heldRace")
  .starts(scheduledEmpty)
  .actions(dispatch, control(ActivityControl.pause))

val pauseRaceQuery: Query =
  query("staleAdmission.pauseRace") find pausedWhileQueued in heldRace limits three

private def raceEvidence(records: String, commitment: Commitment) =
  Evidence.read(
    id = "fixture.realizations.race.evidence." + records,
    records = records,
    source = "fixture.realizations.race.source." + records,
    from = Recorded.read(
      WorkflowServiceGrpc.METHOD_LIST_ACTIVITY_EXECUTIONS,
      Field[ListActivityExecutionsResponse, Seq[ActivityExecutionListInfo]](_.executions)
    ),
    operation = Field[ActivityExecutionListInfo, String](_.activityId),
    commitment = commitment
  )

val pauseRace: Realization = Realization(
  name = "pauseRace",
  machine = staleAdmission,
  producer = "fixture.realizations",
  producerVersion = "1",
  roles = roles,
  correlation = Correlation(
    projection = "fixture.realizations.race.projection",
    run = "fixture.realizations.race.scope.run",
    operation = "fixture.realizations.race.scope.activity",
    observation = "correlated-evidence",
    events = 32,
    buffered = 16,
    keys = 8,
    support = 128,
    work = 1000000,
    eventSize = 512
  ),
  scripts = Vector(
    Script(
      "controller",
      Activation.Controller,
      Vector(
        Item(command = Some(Command("hold-dispatch-before-start", Hold(holdDispatch)))),
        Item(command =
          Some(
            Command(
              "start-activity",
              Instruction.rpc(workflowService, WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION)(
                Vector(
                  Assignment.typed(
                    Field[StartActivityExecutionRequest, String](_.namespace),
                    Operand.environment[String](namespace)
                  ),
                  Assignment.typed(
                    Field[StartActivityExecutionRequest, String](_.activityId),
                    Operand.run()
                  )
                ),
                Vector(
                  ResponseRead.typed(
                    Field[StartActivityExecutionResponse, String](_.runId),
                    Cardinality.one,
                    Vector(Target.Bind(activityRun))
                  )
                )
              )
            )
          )
        ),
        Item(performs =
          Vector(
            Performance(
              control(ActivityControl.pause),
              Command(
                "pause-activity",
                Instruction
                  .rpc(workflowService, WorkflowServiceGrpc.METHOD_PAUSE_ACTIVITY_EXECUTION)(
                    Vector(
                      Assignment.typed(
                        Field[PauseActivityExecutionRequest, String](_.namespace),
                        Operand.environment[String](namespace)
                      ),
                      Assignment.typed(
                        Field[PauseActivityExecutionRequest, String](_.activityId),
                        Operand.run()
                      ),
                      Assignment.typed(
                        Field[PauseActivityExecutionRequest, String](_.runId),
                        Operand.learnedValue[String](activityRun)
                      )
                    ),
                    Vector.empty
                  )
              )
            )
          )
        ),
        Item(command = Some(Command("release-dispatch", Release(holdDispatch))))
      )
    ),
    Script(
      "activity",
      Activation
        .Activity(Name("umpire-", fixture = true, suffix = "-activity"), workerRole, taskQueue),
      Vector(
        Item(performs =
          Vector(Performance(attemptStart, Command("run-attempt", Finish(Literal(Text("done"))))))
        )
      )
    )
  ),
  learned = Vector(Learned(activityRun, LearnedKind.text)),
  observations = Vector(Observed[CorrelatedEvidence]("correlated-evidence")),
  // Every kind the path records is declared: the statuses a caller can read back, and the dispatch and
  // the admission, which only the server commits. No public read reports a commit. The listing is the
  // nearest read back, which the commit observation the specimen proposes replaces.
  evidence = Vector(
    raceEvidence("statusPaused", Commitment.reported),
    raceEvidence("statusStarted", Commitment.reported),
    raceEvidence("dispatchEnqueued", Commitment.durable),
    raceEvidence("attemptAdmitted", Commitment.durable)
  ),
  controls = Vector(Control(holdDispatch, ControlKind.HoldDelivery(dispatchChannel)))
)

// ### A Property that reads why a step was taken

enum DoorState derives Finite:
  case closed, open

enum DoorOutcome derives Finite:
  case accepted

enum DoorFact derives Finite:
  case doorOpened

type DoorStep = Step[DoorState, DoorOutcome, DoorFact]

val doorFamily: umpire.Family = umpire.Family("fixture.realizations.door")
val doorkeeper: Party = Party("doorkeeper")
val door: Entity = Entity("door", key = "doorId")
val push = action("push", doorkeeper) on door

/** A push opens a closed door, and says why; an open door takes no push. */
def pushStep(s: DoorState): List[DoorStep] = s match
  case DoorState.closed =>
    List(Step(DoorOutcome.accepted, DoorState.open, List(DoorFact.doorOpened), "the latch gives"))
  case DoorState.open => Nil

def doorEvidence(f: DoorFact): String = f match
  case DoorFact.doorOpened => "doorOpened"

val doorMachine: Machine[DoorState, DoorOutcome, DoorFact] =
  machine[DoorState, DoorOutcome, DoorFact](doorFamily, "door") {
    forEntity(door)
    starts(DoorState.closed)
    ends(_ == DoorState.open)
    evidence(doorEvidence)
    steps(push ~> pushStep)
  }

/** The step's explanation is part of the step a Property reads, as its outcome and facts are. */
private val opensBecauseTheLatchGives =
  doorMachine.property("opensBecauseTheLatchGives") when push holds { s =>
    s.because == "the latch gives" && s.facts.contains(DoorFact.doorOpened)
  }

private val pushed = doorMachine.scenario("pushed").starts(DoorState.closed).actions(push)

val doorOpens: Query =
  query("door.opens") find opensBecauseTheLatchGives in pushed limits Limits(
    "one",
    steps = 1,
    actions = 1,
    search = 16
  )

val doorRealization: Realization = Realization(
  name = "doorRealization",
  machine = doorMachine,
  producer = "fixture.realizations",
  producerVersion = "1",
  roles = roles,
  correlation = Correlation(
    projection = "fixture.realizations.door.projection",
    run = "fixture.realizations.door.scope.run",
    operation = "fixture.realizations.door.scope.door",
    observation = "correlated-evidence",
    events = 8,
    buffered = 8,
    keys = 2,
    support = 16,
    work = 1000,
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
              push,
              Command(
                "push-door",
                Instruction.rpc(
                  workflowService,
                  WorkflowServiceGrpc.METHOD_START_WORKFLOW_EXECUTION
                )(
                  Vector(
                    Assignment.typed(
                      Field[StartWorkflowExecutionRequest, String](_.namespace),
                      Operand.environment[String](namespace)
                    ),
                    Assignment.typed(
                      Field[StartWorkflowExecutionRequest, String](_.workflowId),
                      Operand.run()
                    )
                  ),
                  Vector.empty
                )
              )
            )
          )
        ),
        Item(command =
          Some(
            Command(
              "history",
              Instruction
                .rpc(workflowService, WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY)(
                  Vector(
                    Assignment.typed(
                      Field[GetWorkflowExecutionHistoryRequest, String](_.namespace),
                      Operand.environment[String](namespace)
                    ),
                    Assignment.typed(
                      Field[GetWorkflowExecutionHistoryRequest, String](_.getExecution.workflowId),
                      Operand.run()
                    )
                  ),
                  Vector(
                    ResponseRead.typed(
                      Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]](
                        _.getHistory.events.map(event => event)
                      ),
                      Cardinality.each,
                      Vector(Target.Lift("correlated-evidence"))
                    )
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
    Evidence.history(
      id = "fixture.realizations.door.evidence.opened",
      records = "doorOpened",
      source = "fixture.realizations.door.source.history",
      from = Recorded.history(
        Field[HistoryEvent, Option[WorkflowExecutionStartedEventAttributes]](
          _.attributes.workflowExecutionStartedEventAttributes
        )
      ),
      operation = Field[HistoryEvent, Long](_.eventId),
      commitment = Commitment.reported
    )
  )
)

// ### An activity whose first attempt fails and whose retry completes

enum ErrandState derives Finite:
  case idle, queued, held, retrying, done, withdrawn

/** How the worker answers one attempt. */
enum ErrandAnswer derives Finite:
  case failed, completed, canceled

enum ErrandOutcome derives Finite:
  case accepted

/** What a listing of the errand shows: that it exists, and that it closed completed or canceled. */
enum ErrandFact derives Finite:
  case errandListed, errandClosed, errandWithdrawn

type ErrandStep = Step[ErrandState, ErrandOutcome, ErrandFact]

val errandFamily: umpire.Family = umpire.Family("fixture.realizations.errand")
val requester: Party = Party("requester")
val runner: Party = Party("runner")
val errand: Entity = Entity("errand", key = "errandId")
val request = action("request", requester).creates(errand)
val deliver = action("deliver", runner) on errand
val answer = action("answer", runner).on(errand).input[ErrandAnswer]("answer")

def requestStep(s: ErrandState): List[ErrandStep] = s match
  case ErrandState.idle =>
    List(Step(ErrandOutcome.accepted, ErrandState.queued, List(ErrandFact.errandListed)))
  case _ => Nil

/** The runner is handed the attempt. Nothing a caller reads says so. */
def deliverStep(s: ErrandState): List[ErrandStep] = s match
  case ErrandState.queued | ErrandState.retrying =>
    List(Step(ErrandOutcome.accepted, ErrandState.held))
  case _ => Nil

/**
 * A failed attempt is retried and a caller reads nothing new; a completed one closes the errand, and
 * a canceled one withdraws it.
 */
def answerStep(s: ErrandState, a: ErrandAnswer): List[ErrandStep] =
  if s != ErrandState.held then Nil
  else
    a match
      case ErrandAnswer.failed    => List(Step(ErrandOutcome.accepted, ErrandState.retrying))
      case ErrandAnswer.completed =>
        List(Step(ErrandOutcome.accepted, ErrandState.done, List(ErrandFact.errandClosed)))
      case ErrandAnswer.canceled =>
        List(Step(ErrandOutcome.accepted, ErrandState.withdrawn, List(ErrandFact.errandWithdrawn)))

def errandEvidence(f: ErrandFact): String = f match
  case ErrandFact.errandListed    => "errandListed"
  case ErrandFact.errandClosed    => "errandClosed"
  case ErrandFact.errandWithdrawn => "errandWithdrawn"

val errandMachine: Machine[ErrandState, ErrandOutcome, ErrandFact] =
  machine[ErrandState, ErrandOutcome, ErrandFact](errandFamily, "errand") {
    forEntity(errand)
    starts(ErrandState.idle)
    ends(s => s == ErrandState.done || s == ErrandState.withdrawn)
    evidence(errandEvidence)
    steps(request ~> requestStep, deliver ~> deliverStep, answer ~> answerStep)
  }

private val closesOnCompletion =
  errandMachine.property("closesOnCompletion") when answer(ErrandAnswer.completed) holds { s =>
    s.state == ErrandState.done && s.facts.contains(ErrandFact.errandClosed)
  }

private val retriedOnce = errandMachine
  .scenario("retriedOnce")
  .starts(ErrandState.idle)
  .actions(request, deliver, answer(ErrandAnswer.failed), deliver, answer(ErrandAnswer.completed))

val errandRetry: Query =
  query("errand.retry") find closesOnCompletion in retriedOnce limits Limits(
    "five",
    steps = 5,
    actions = 5,
    search = 4096
  )

// The one path that takes the worker's canceled answer, which Testpilot has no instruction for: its
// Query is blocked by that command, and the retry above, whose path does not take it, is not.
private val withdrawsOnCancel =
  errandMachine.property("withdrawsOnCancel") when answer(ErrandAnswer.canceled) holds { s =>
    s.state == ErrandState.withdrawn && s.facts.contains(ErrandFact.errandWithdrawn)
  }

private val canceledOnce = errandMachine
  .scenario("canceledOnce")
  .starts(ErrandState.idle)
  .actions(request, deliver, answer(ErrandAnswer.canceled))

val errandWithdrawn: Query =
  query("errand.withdrawn") find withdrawsOnCancel in canceledOnce limits Limits(
    "three",
    steps = 3,
    actions = 3,
    search = 4096
  )

private val errandType = Name("umpire-", fixture = true, suffix = "-errand")

/**
 * One kind of the errand's evidence: an entry of the namespace's listing, once it reads so. A poll's
 * condition reads the entry alone, so the listing is the run's own only in a namespace that holds no
 * other activity.
 */
private def listed(records: String) =
  Evidence.read(
    id = "fixture.realizations.errand.evidence." + records,
    records = records,
    source = "fixture.realizations.errand.source." + records,
    from = Recorded.read(
      WorkflowServiceGrpc.METHOD_LIST_ACTIVITY_EXECUTIONS,
      Field[ListActivityExecutionsResponse, Seq[ActivityExecutionListInfo]](_.executions)
    ),
    operation = Field[ActivityExecutionListInfo, String](_.activityId),
    commitment = Commitment.reported
  )

private val errandListed = listed("errandListed")
private val errandClosed = listed("errandClosed")
private val errandWithdrawnEvidence = listed("errandWithdrawn")

private def awaitListed(
    id: String,
    evidence: EvidenceRef[ListActivityExecutionsRequest, ActivityExecutionListInfo],
    reads: Condition[ActivityExecutionListInfo]
) =
  Command(
    id,
    Instruction.poll(evidence, workflowService)(
      Vector(
        Assignment.typed(
          Field[ListActivityExecutionsRequest, String](_.namespace),
          Operand.environment[String](namespace)
        )
      ),
      reads,
      250
    )
  )

val errandRealization: Realization = Realization(
  name = "errandRealization",
  machine = errandMachine,
  producer = "fixture.realizations",
  producerVersion = "1",
  roles = roles,
  correlation = Correlation(
    projection = "fixture.realizations.errand.projection",
    run = "fixture.realizations.errand.scope.run",
    operation = "fixture.realizations.errand.scope.errand",
    observation = "correlated-evidence",
    events = 8,
    buffered = 8,
    keys = 2,
    support = 16,
    work = 1000,
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
              request,
              Command(
                "start-activity",
                Instruction.rpc(
                  workflowService,
                  WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
                )(
                  Vector(
                    Assignment.typed(
                      Field[StartActivityExecutionRequest, String](_.namespace),
                      Operand.environment[String](namespace)
                    ),
                    Assignment.typed(
                      Field[StartActivityExecutionRequest, String](_.activityId),
                      Operand.run()
                    ),
                    Assignment.typed(
                      Field[StartActivityExecutionRequest, String](_.getActivityType.name),
                      Operand.named(errandType)
                    ),
                    Assignment.typed(
                      Field[StartActivityExecutionRequest, String](_.getTaskQueue.name),
                      Operand.environment[String](queueResource)
                    ),
                    Assignment.typed(
                      Field[StartActivityExecutionRequest, String](_.requestId),
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
          )
        ),
        // The entry is the errand's once it has a schedule time, has made a transition and names an activity.
        Item(command =
          Some(
            awaitListed(
              "await-listed",
              errandListed,
              Condition.all(
                Condition.present(
                  Field[ActivityExecutionListInfo, Option[com.google.protobuf.timestamp.Timestamp]](
                    _.scheduleTime
                  )
                ),
                Condition.greater(
                  Field[ActivityExecutionListInfo, Long](_.stateTransitionCount),
                  Operand.number(0L)
                ),
                Condition.not(
                  Condition.equal(
                    Field[ActivityExecutionListInfo, String](_.activityId),
                    Operand.text("")
                  )
                )
              )
            )
          )
        ),
        Item(command =
          Some(
            awaitListed(
              "await-closed",
              errandClosed,
              Condition.equal(
                Field[ActivityExecutionListInfo, ActivityExecutionStatus](_.status),
                Operand.enumValue(ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_COMPLETED)
              )
            )
          )
        ),
        Item(
          command = Some(
            awaitListed(
              "await-withdrawn",
              errandWithdrawnEvidence,
              Condition.equal(
                Field[ActivityExecutionListInfo, ActivityExecutionStatus](_.status),
                Operand.enumValue(ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_CANCELED)
              )
            )
          ),
          when = Vector(answer(ErrandAnswer.canceled))
        )
      )
    ),
    Script(
      "errand",
      Activation.Activity(errandType, workerRole, taskQueue, starts = Vector(deliver)),
      Vector(
        Item(performs =
          Vector(
            Performance(
              answer(ErrandAnswer.failed),
              Command(
                "fail-attempt",
                AttemptFailure(
                  Proto[ApiFailure](
                    ProtoField
                      .typed(Field[ApiFailure, String](_.message), ProtoValue.text("not yet")),
                    ProtoField.typed(
                      Field[ApiFailure, ApplicationFailureInfo](_.getApplicationFailureInfo),
                      ProtoValue.message(
                        Proto[ApplicationFailureInfo](
                          ProtoField.typed(
                            Field[ApplicationFailureInfo, String](_.`type`),
                            ProtoValue.text("NotYet")
                          )
                        )
                      )
                    )
                  )
                )
              )
            ),
            Performance(
              answer(ErrandAnswer.completed),
              Command("complete-attempt", Finish(Literal(Text("done"))))
            ),
            Performance(answer(ErrandAnswer.canceled), Command("cancel-attempt", AttemptCanceled))
          )
        )
      )
    )
  ),
  observations = Vector(Observed[CorrelatedEvidence]("correlated-evidence")),
  evidence = Vector(errandListed, errandClosed, errandWithdrawnEvidence),
  cleanup = "cleanup"
)

// ### What no Case carries yet

/** A second door, for a realization of its own: a machine runs under one. */
val tallyMachine: Machine[DoorState, DoorOutcome, DoorFact] =
  machine[DoorState, DoorOutcome, DoorFact](doorFamily, "tally") {
    forEntity(door)
    starts(DoorState.closed)
    ends(_ == DoorState.open)
    evidence(doorEvidence)
    steps(push ~> pushStep)
  }

private val tallyOpened =
  tallyMachine.property("tallyOpened") when push holds (_.facts.contains(DoorFact.doorOpened))

private val tallied = tallyMachine.scenario("tallied").starts(DoorState.closed).actions(push)

val tallyOpens: Query =
  query("tally.opens") find tallyOpened in tallied limits Limits(
    "one",
    steps = 1,
    actions = 1,
    search = 16
  )

/** A realization whose evidence keeps fields and is read from one message. */
private val tallyOpenedEvidence = Evidence.read(
  id = "fixture.realizations.tally.evidence.opened",
  records = "doorOpened",
  source = "fixture.realizations.tally.source.describe",
  from = Recorded.single(
    WorkflowServiceGrpc.METHOD_DESCRIBE_ACTIVITY_EXECUTION,
    Field[DescribeActivityExecutionResponse, ActivityExecutionInfo](_.getInfo)
  ),
  operation = Field[ActivityExecutionInfo, String](_.activityId),
  commitment = Commitment.reported,
  fields = Vector(
    EvidenceField.typed(
      "run",
      Field[ActivityExecutionInfo, String](_.runId),
      role = Some(FieldRole.operation)
    ),
    EvidenceField.typed(
      "attempt",
      Field[ActivityExecutionInfo, Int](_.attempt),
      role = Some(FieldRole.attempt)
    ),
    EvidenceField.typed(
      "identity",
      Field[ActivityExecutionInfo, String](_.lastWorkerIdentity),
      redacted = true
    )
  ),
  exhaustive = true
)

val tallyRealization: Realization = Realization(
  name = "tallyRealization",
  machine = tallyMachine,
  producer = "fixture.realizations",
  producerVersion = "1",
  roles = roles,
  correlation = Correlation(
    projection = "fixture.realizations.tally.projection",
    run = "fixture.realizations.tally.scope.run",
    operation = "fixture.realizations.tally.scope.door",
    observation = "correlated-evidence",
    events = 8,
    buffered = 8,
    keys = 2,
    support = 16,
    work = 1000,
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
              push,
              Command(
                "push-door",
                Instruction.rpc(
                  workflowService,
                  WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
                )(
                  Vector(
                    Assignment.typed(
                      Field[StartActivityExecutionRequest, String](_.namespace),
                      Operand.environment[String](namespace)
                    ),
                    Assignment.typed(
                      Field[StartActivityExecutionRequest, String](_.activityId),
                      Operand.run()
                    )
                  ),
                  Vector.empty
                )
              )
            )
          )
        ),
        Item(command =
          Some(
            Command(
              "await-opened",
              Instruction.poll(tallyOpenedEvidence, workflowService)(
                Vector(
                  Assignment.typed(
                    Field[DescribeActivityExecutionRequest, String](_.namespace),
                    Operand.environment[String](namespace)
                  ),
                  Assignment.typed(
                    Field[DescribeActivityExecutionRequest, String](_.activityId),
                    Operand.run()
                  )
                ),
                Condition.equal(Field[ActivityExecutionInfo, Int](_.attempt), Operand.integer(1)),
                250
              ),
              closes = Vector("fixture.realizations.tally.evidence.opened")
            )
          )
        )
      )
    )
  ),
  observations = Vector(Observed[CorrelatedEvidence]("correlated-evidence")),
  evidence = Vector(
    tallyOpenedEvidence,
    // The start call's own answer, as the Run records it. A machine's evidence names each fact once,
    // so this second kind confirms a fact the door does not record and is off every path.
    Evidence.runEvent(
      id = "fixture.realizations.tally.evidence.pushed",
      records = "doorPushed",
      source = "fixture.realizations.tally.source.pushed",
      from = Recorded.runEvent[InstructionOutcome](
        EventKind.instructionCompleted,
        "controller",
        "push-door",
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
  )
)
