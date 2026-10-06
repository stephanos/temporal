// Realizations the lifter emits as written, and tools/umpire/lower lowers or refuses.
//
// Every machine runs on the door's types. `doorRealization` opens a door whose step says why.
// `learnedRun` binds the run id its start call returns and reads it from two branches. `pauseRace`
// declares what Testpilot cannot run yet, the hold of a channel's deliveries and a durable-commit
// observation, so lowering it names each gap instead of a Case. `errandRealization` runs a standalone
// activity whose first attempt fails and whose retry completes, on a machine whose facts a listing
// reports. `tallyRealization` declares evidence read from one message that keeps fields, one of them
// without its value, which no Case carries, and the Run's own record as evidence. The lifter's tests
// lift them and compare the IR with expected/realizations.json. They lift `heldByValue`, which names
// the monitors of a Query's expected Run by value, and `heldByName`, by their names, apart from them
// and require one expected Run of the two.
package fixture.realizations

import umpire.*
import DoorEntity.door
import ErrandEntity.errand
import umpire.realize.*, temporal.realize.{Role, RoleKind, WorkerActivation, WorkflowHistory}
import umpire.realize.Instruction.*, temporal.realize.WorkerInstruction.*
import umpire.realize.Operand.*
import umpire.realize.ProtoValue.*
import io.temporal.api.workflowservice.v1.*
import io.temporal.api.history.v1.*
import io.temporal.api.activity.v1.{ActivityExecutionInfo, ActivityExecutionListInfo}
import io.temporal.api.enums.v1.{ActivityExecutionStatus, HistoryEventFilterType}
import io.temporal.api.failure.v1.{ApplicationFailureInfo, Failure as ApiFailure}
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

private val roles = Vector(
  Role(workflowService, RoleKind.endpoint),
  Role(workerRole, RoleKind.worker, namespace = namespace),
  Role(taskQueue, RoleKind.taskQueue, namespace = namespace, resource = queueResource)
)

// ### A Property that reads why a step was taken

enum DoorState derives Finite:
  case closed, open

enum DoorOutcome derives Finite:
  case accepted

enum DoorFact derives Finite:
  case doorOpened

type DoorStep = Step[DoorState, DoorOutcome, DoorFact]

// The doors' entity, in an object of its own, which the machine objects read while they initialize.
object DoorEntity:
  val door: Entity = Entity(key = "doorId")

object doorkeeper extends Actor
val push = action(doorkeeper) on door

// A push opens a closed door, and says why; an open door takes no push.
def pushStep(s: DoorState): List[DoorStep] = s match
  case DoorState.closed =>
    List(Step(DoorOutcome.accepted, DoorState.open, List(DoorFact.doorOpened), "the latch gives"))
  case DoorState.open => Nil

def doorEvidence(f: DoorFact): String = f match
  case DoorFact.doorOpened => "doorOpened"

object Door extends Machine[DoorState, DoorOutcome, DoorFact]:
  val entity = door
  val init = DoorState.closed
  def end(doorState: State) = doorState == DoorState.open
  val evidence: DoorFact => String = doorEvidence

  object rules extends Bindings(push ~> pushStep)

// The step's explanation is part of the step a Property reads, as its outcome and facts are.
private val opensBecauseTheLatchGives =
  Door.property when push holds { s =>
    s.because == "the latch gives" && s.facts.contains(DoorFact.doorOpened)
  }

private val pushed = Door.scenario.starts(DoorState.closed).actions(push)

val doorOpens: Query =
  query("door.opens") find opensBecauseTheLatchGives in pushed limits Limits(
    "one",
    steps = 1,
    actions = 1,
    search = 16
  ) total 2

val doorRealization: Realization = Realization(
  machine = Door,
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
    Evidence.keyed(
      id = "fixture.realizations.door.evidence.opened",
      records = "doorOpened",
      source = "fixture.realizations.door.source.history",
      from = WorkflowHistory.event(
        Field[HistoryEvent, Option[WorkflowExecutionStartedEventAttributes]](
          _.attributes.workflowExecutionStartedEventAttributes
        )
      ),
      operation = Field[HistoryEvent, Long](_.eventId),
      commitment = Commitment.reported
    )
  )
)

// The bounds of a path of one push, which every door below is checked within.
private val onePush = Limits("one", steps = 1, actions = 1, search = 16)

// A small correlation, scoped by the section's own prefix.
private def doorCorrelation(scope: String) = Correlation(
  projection = s"fixture.realizations.$scope.projection",
  run = s"fixture.realizations.$scope.scope.run",
  operation = s"fixture.realizations.$scope.scope.door",
  observation = "correlated-evidence",
  events = 8,
  buffered = 8,
  keys = 2,
  support = 16,
  work = 1000,
  eventSize = 512
)

// ### A learned run id, read by two branches

// A door the start call opens. A machine runs under one realization, so it is a door of its own.
object Run extends Machine[DoorState, DoorOutcome, DoorFact]:
  val entity = door
  val init = DoorState.closed
  def end(doorState: State) = doorState == DoorState.open
  val evidence: DoorFact => String = doorEvidence

  object rules extends Bindings(push ~> pushStep)

private val runOpened = Run.property when push holds (_.facts.contains(DoorFact.doorOpened))

private val started = Run.scenario.starts(DoorState.closed).actions(push)

val runOpens: Query = query("run.opens") find runOpened in started limits onePush total 2

private val run = "workflow-run"
private val workflowType = Name("umpire-", fixture = true, suffix = "-workflow")

// The workflow a read names: the run's id, and the run id the start call returned.
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

private val runOpenedEvidence = Evidence.read(
  id = "fixture.realizations.run.evidence.opened",
  records = "doorOpened",
  source = "fixture.realizations.run.source.history",
  from = Recorded.read(
    WorkflowServiceGrpc.METHOD_GET_WORKFLOW_EXECUTION_HISTORY,
    Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]](
      _.getHistory.events.map(event => event)
    )
  ),
  operation = Field[HistoryEvent, Long](_.eventId),
  commitment = Commitment.reported
)

// The push: starts the workflow and binds the run id it returns, once.
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
      )
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

// One branch: polls the started run's history for its start.
private val awaitStarted = Command(
  "await-started",
  Instruction.readUntil(runOpenedEvidence, workflowService)(
    startedRun,
    Condition.present(
      Field[HistoryEvent, Option[WorkflowExecutionStartedEventAttributes]](
        _.attributes.workflowExecutionStartedEventAttributes
      )
    ),
    250
  ),
  after = Some(After("start-workflow"))
)

// The other branch: waits for the started run to close. Neither branch waits for the other.
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

// Reads the history once both branches are done.
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
  after = Some(After("await-started", "await-close"))
)

val learnedRun: Realization = Realization(
  machine = Run,
  producer = "fixture.realizations",
  producerVersion = "1",
  roles = roles,
  correlation = doorCorrelation("run"),
  scripts = Vector(
    Script(
      "controller",
      Activation.Controller,
      Vector(
        Item(performs = Vector(Performance(push, start))),
        Item(command = Some(awaitStarted)),
        Item(command = Some(awaitClose)),
        Item(command = Some(history))
      )
    )
  ),
  learned = Vector(Learned(run, LearnedKind.text)),
  observations = Vector(
    Observed[HistoryEvent]("history-event"),
    Observed[CorrelatedEvidence]("correlated-evidence")
  ),
  evidence = Vector(runOpenedEvidence),
  cleanup = "cleanup"
)

// ### A race no Driver holds

// The dispatch the race holds, as a channel: no Driver holds the deliveries of one.
val dispatchChannel: Channel[DoorFact] =
  channel[DoorFact](
    capacity = 1,
    order = Order.unordered,
    loss = Loss.reliable
  )

// A door the race pushes while the dispatch is held.
object Race extends Machine[DoorState, DoorOutcome, DoorFact]:
  val entity = door
  val init = DoorState.closed
  def end(doorState: State) = doorState == DoorState.open
  val evidence: DoorFact => String = doorEvidence

  object rules extends Bindings(push ~> pushStep)

private val pushedWhileHeld =
  Race.property when push holds (_.facts.contains(DoorFact.doorOpened))

private val heldRace = Race.scenario.starts(DoorState.closed).actions(push)

val pauseRaceQuery: Query =
  query("race.paused") find pushedWhileHeld in heldRace limits onePush total 2

private val activityRun = "activity-run"
private val holdDispatch = "hold-dispatch"

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
  machine = Race,
  producer = "fixture.realizations",
  producerVersion = "1",
  roles = roles,
  correlation = doorCorrelation("race"),
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
              push,
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
    )
  ),
  learned = Vector(Learned(activityRun, LearnedKind.text)),
  observations = Vector(Observed[CorrelatedEvidence]("correlated-evidence")),
  // The listing reports the door's fact. The dispatch only the server commits, and no public read
  // reports a commit: the listing is the nearest read back, and declaring it durable is the gap.
  evidence = Vector(
    raceEvidence("doorOpened", Commitment.reported),
    raceEvidence("dispatchEnqueued", Commitment.durable)
  ),
  controls = Vector(Actuator(holdDispatch, ControlKind.HoldDelivery(dispatchChannel)))
)

// ### Monitors a Query's expected Run names

// Whether a door was seen open, and whether it was seen to open again after that.
enum Seen derives Finite:
  case shut, opened, reopened

def seen(s: Seen, before: DoorState, after: DoorStep): Seen =
  if !after.facts.contains(DoorFact.doorOpened) then s
  else if s == Seen.shut then Seen.opened
  else Seen.reopened

// The same watch, read only after a step that opens the door.
val opensOnce: Monitor[DoorState, DoorOutcome, DoorFact, Seen] =
  monitor[DoorState, DoorOutcome, DoorFact, Seen](Seen.shut)(seen)(_ == Seen.reopened)
    .readAfter(after => after.facts.contains(DoorFact.doorOpened))

val staysOpen: Monitor[DoorState, DoorOutcome, DoorFact, Seen] =
  monitor[DoorState, DoorOutcome, DoorFact, Seen](Seen.shut)(seen)(_ == Seen.reopened)

// A door with authored monitors, which no realization above runs.
object Watched extends Machine[DoorState, DoorOutcome, DoorFact]:
  val entity = door
  val init = DoorState.closed
  def end(doorState: State) = doorState == DoorState.open
  val evidence: DoorFact => String = doorEvidence

  object monitors:
    val once = opensOnce
    val open = staysOpen

  object rules extends Bindings(push ~> pushStep)

private val watchedOpens =
  Watched.property when push holds (_.facts.contains(DoorFact.doorOpened))

private val watched = Watched.scenario.starts(DoorState.closed).actions(push)

// A Run's expected verdicts with each monitor named by value, beside the same verdicts with each
// named by its name: one expected Run.
val heldByValue: Query =
  (query find watchedOpens in watched limits onePush total 2).expect(
    RunExpectation(
      Conformance.conformant,
      PropertyOutcome.satisfied,
      PropertyOutcome.satisfied,
      Disposition.completed,
      Cleanup.succeeded,
      monitors = Vector(
        MonitorExpectation(opensOnce, PropertyOutcome.inconclusive, Some(Reason.neverEvaluated)),
        MonitorExpectation(staysOpen, PropertyOutcome.satisfied)
      )
    )
  )
val heldByName: Query =
  (query find watchedOpens in watched limits onePush total 2).expect(
    RunExpectation(
      Conformance.conformant,
      property = PropertyOutcome.satisfied,
      contract = PropertyOutcome.satisfied,
      disposition = Disposition.completed,
      cleanup = Cleanup.succeeded,
      monitors = Vector(
        MonitorExpectation("opensOnce", PropertyOutcome.inconclusive, Some(Reason.neverEvaluated)),
        MonitorExpectation("staysOpen", PropertyOutcome.satisfied)
      )
    )
  )

// ### An activity whose first attempt fails and whose retry completes

enum ErrandState derives Finite:
  case idle, queued, held, retrying, done, withdrawn

// How the worker answers one attempt.
enum ErrandAnswer derives Finite:
  case failed, completed, canceled

enum ErrandOutcome derives Finite:
  case accepted

// What a listing of the errand shows: that it exists, and that it closed completed or canceled.
enum ErrandFact derives Finite:
  case errandListed, errandClosed, errandWithdrawn

type ErrandStep = Step[ErrandState, ErrandOutcome, ErrandFact]

// The errand's entity, in an object of its own, as the doors' is.
object ErrandEntity:
  val errand: Entity = Entity(key = "errandId")

object requester extends Actor
object runner extends Actor
val request = action(requester).creates(errand)
val deliver = action(runner) on errand
val answer = action(runner).on(errand).input[ErrandAnswer]("answer")

def requestStep(s: ErrandState): List[ErrandStep] = s match
  case ErrandState.idle =>
    List(Step(ErrandOutcome.accepted, ErrandState.queued, List(ErrandFact.errandListed)))
  case _ => Nil

// The runner is handed the attempt. Nothing a caller reads says so.
def deliverStep(s: ErrandState): List[ErrandStep] = s match
  case ErrandState.queued | ErrandState.retrying =>
    List(Step(ErrandOutcome.accepted, ErrandState.held))
  case _ => Nil

// A failed attempt is retried and a caller reads nothing new; a completed one closes the errand, and
// a canceled one withdraws it.
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

object Errand extends Machine[ErrandState, ErrandOutcome, ErrandFact]:
  val entity = errand
  val init = ErrandState.idle
  def end(s: State) = s == ErrandState.done || s == ErrandState.withdrawn
  val evidence: ErrandFact => String = errandEvidence

  object rules
      extends Bindings(request ~> requestStep, deliver ~> deliverStep, answer ~> answerStep)

private val closesOnCompletion =
  Errand.property when answer(ErrandAnswer.completed) holds { s =>
    s.state == ErrandState.done && s.facts.contains(ErrandFact.errandClosed)
  }

private val retriedOnce = Errand.scenario
  .starts(ErrandState.idle)
  .actions(request, deliver, answer(ErrandAnswer.failed), deliver, answer(ErrandAnswer.completed))

val errandRetry: Query =
  query("errand.retry") find closesOnCompletion in retriedOnce limits Limits(
    "five",
    steps = 5,
    actions = 5,
    search = 4096
  ) total 30

// The one path that takes the worker's canceled answer, which Testpilot has no instruction for: its
// Query is blocked by that command, and the retry above, whose path does not take it, is not.
private val withdrawsOnCancel =
  Errand.property when answer(ErrandAnswer.canceled) holds { s =>
    s.state == ErrandState.withdrawn && s.facts.contains(ErrandFact.errandWithdrawn)
  }

private val canceledOnce = Errand.scenario
  .starts(ErrandState.idle)
  .actions(request, deliver, answer(ErrandAnswer.canceled))

val errandWithdrawn: Query =
  query("errand.withdrawn") find withdrawsOnCancel in canceledOnce limits Limits(
    "three",
    steps = 3,
    actions = 3,
    search = 4096
  ) total 18

private val errandType = Name("umpire-", fixture = true, suffix = "-errand")

// Why the first attempt fails.
private val notYet = Proto[ApplicationFailureInfo](
  ProtoField.typed(Field[ApplicationFailureInfo, String](_.`type`), ProtoValue.text("NotYet"))
)

// One kind of the errand's evidence: an entry of the namespace's listing, once it reads so. A poll's
// condition reads the entry alone, so the listing is the run's own only in a namespace that holds no
// other activity.
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
    Instruction.readUntil(evidence, workflowService)(
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
  machine = Errand,
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
      WorkerActivation.Activity(errandType, workerRole, taskQueue, starts = Vector(deliver)),
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
                      ProtoValue.message(notYet)
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

// A second door, for a realization of its own: a machine runs under one.
object Tally extends Machine[DoorState, DoorOutcome, DoorFact]:
  val entity = door
  val init = DoorState.closed
  def end(doorState: State) = doorState == DoorState.open
  val evidence: DoorFact => String = doorEvidence

  object rules extends Bindings(push ~> pushStep)

private val tallyOpened =
  Tally.property when push holds (_.facts.contains(DoorFact.doorOpened))

private val tallied = Tally.scenario.starts(DoorState.closed).actions(push)

val tallyOpens: Query =
  query("tally.opens") find tallyOpened in tallied limits Limits(
    "one",
    steps = 1,
    actions = 1,
    search = 16
  ) total 2

// A realization whose evidence keeps fields and is read from one message.
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
  machine = Tally,
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
              Instruction.readUntil(tallyOpenedEvidence, workflowService)(
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
