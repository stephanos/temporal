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

import fixture.specimens.admission.{AdmissionFact, Message as Dispatched, dispatch, scheduledEmpty, staleAdmission, three}
import temporal.nexuscaller.{Reply, Timeout, handlerReply, nexusProtocol, schedule}
import temporal.standaloneactivity.{Control as ActivityControl, attemptStart, control}
import umpire.*
import umpire.realize.*
import umpire.realize.Instruction.*
import umpire.realize.Operand.*
import umpire.realize.ProtoValue.*

private val workflowService = "temporal.workflow-service"
private val workerRole = "temporal.worker"
private val taskQueue = "temporal.task-queue"
private val namespace = "temporal.worker.namespace"
private val queueResource = "temporal.task-queue.resource"
private val endpoint = "temporal.nexus-endpoint"

private val service = "/temporal.api.workflowservice.v1.WorkflowService/"

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
  Assignment("namespace", Environment(namespace)),
  Assignment("execution.workflow_id", Run),
  Assignment("execution.run_id", LearnedValue(run))
)

private val start = Command(
  "start-workflow",
  Rpc(
    workflowService,
    service + "StartWorkflowExecution",
    Vector(
      Assignment("namespace", Environment(namespace)),
      Assignment("workflow_id", Run),
      Assignment("workflow_type.name", Literal(Named(workflowType))),
      Assignment("task_queue.name", Environment(queueResource)),
      Assignment("request_id", Run)
    ),
    Vector(ResponseRead("run_id", Cardinality.one, Vector(Target.Bind(run))))
  )
)

/** One branch: polls the started run's history for the scheduled event. */
private val awaitScheduled = Command(
  "await-scheduled",
  Poll(
    "fixture.realizations.evidence.scheduled",
    workflowService,
    startedRun,
    Present(Path(Projected, "attributes<nexus_operation_scheduled_event_attributes>")),
    250
  ),
  after = Some(After("start-workflow"))
)

/** The other branch: waits for the started run to close. Neither branch waits for the other. */
private val awaitClose = Command(
  "await-close",
  Rpc(
    workflowService,
    service + "GetWorkflowExecutionHistory",
    startedRun ++ Vector(
      Assignment("wait_new_event", Literal(Flag(true))),
      Assignment(
        "history_event_filter_type",
        Literal(EnumName("HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT"))
      )
    )
  ),
  after = Some(After("start-workflow"))
)

/** Reads the history once both branches are done. */
private val history = Command(
  "history",
  Rpc(
    workflowService,
    service + "GetWorkflowExecutionHistory",
    startedRun,
    Vector(
      ResponseRead(
        "history.events[*]",
        Cardinality.each,
        Vector(Target.Observe("history-event"), Target.Lift("correlated-evidence"))
      )
    )
  ),
  after = Some(After("await-scheduled", "await-close"))
)

private val payload = Proto(
  "temporal.api.common.v1.Payload",
  ProtoField("metadata", Mapping(ProtoEntry("encoding", Utf8("json/plain")))),
  ProtoField("data", Utf8("\"done\""))
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
                  Proto(
                    "temporal.api.command.v1.Command",
                    ProtoField("command_type", EnumName("COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION")),
                    ProtoField(
                      "schedule_nexus_operation_command_attributes",
                      Message(
                        Proto(
                          "temporal.api.command.v1.ScheduleNexusOperationCommandAttributes",
                          ProtoField("endpoint", RoleId(endpoint)),
                          ProtoField("service", Text("fixture.service")),
                          ProtoField("operation", Text("probe"))
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
            Command("await-nexus-operation", AwaitCommand("start-nexus-operation"), regardless = true)
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
                  Proto(
                    "temporal.api.nexus.v1.StartOperationResponse",
                    ProtoField(
                      "sync_success",
                      Message(
                        Proto(
                          "temporal.api.nexus.v1.StartOperationResponse.Sync",
                          ProtoField("payload", Message(payload))
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
    Observed("history-event", "temporal.api.history.v1.HistoryEvent"),
    Observed("correlated-evidence", "temporal.server.api.testpilot.v1.CorrelatedEvidence")
  ),
  evidence = Vector(
    Evidence(
      id = "fixture.realizations.evidence.scheduled",
      records = "nexusOperationScheduled",
      source = "fixture.realizations.source.scheduled",
      from = Recorded.Read(service + "GetWorkflowExecutionHistory", "history.events[*]"),
      operation = "event_id",
      commitment = Commitment.reported
    ),
    Evidence(
      id = "fixture.realizations.evidence.completed",
      records = "nexusOperationCompleted",
      source = "fixture.realizations.source.history",
      from = Recorded.History("nexus_operation_completed_event_attributes"),
      operation = "attributes<nexus_operation_completed_event_attributes>.scheduled_event_id",
      commitment = Commitment.reported
    )
  ),
  cleanup = "cleanup"
)

// ### The activity specimen's held race

/** The dispatch the race holds, as the channel the specimen proposes for it. */
val dispatchChannel: Channel[Dispatched] =
  channel[Dispatched]("dispatchChannel", capacity = 1, order = Order.unordered, loss = Loss.reliable)

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

val pauseRaceQuery: Query = query("staleAdmission.pauseRace") find pausedWhileQueued in heldRace limits three

private def raceEvidence(records: String, commitment: Commitment) =
  Evidence(
    id = "fixture.realizations.race.evidence." + records,
    records = records,
    source = "fixture.realizations.race.source." + records,
    from = Recorded.Read(service + "ListActivityExecutions", "executions"),
    operation = "activity_id",
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
              Rpc(
                workflowService,
                service + "StartActivityExecution",
                Vector(
                  Assignment("namespace", Environment(namespace)),
                  Assignment("activity_id", Run)
                ),
                Vector(ResponseRead("run_id", Cardinality.one, Vector(Target.Bind(activityRun))))
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
                Rpc(
                  workflowService,
                  service + "PauseActivityExecution",
                  Vector(
                    Assignment("namespace", Environment(namespace)),
                    Assignment("activity_id", Run),
                    Assignment("run_id", LearnedValue(activityRun))
                  )
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
      Activation.Activity(Name("umpire-", fixture = true, suffix = "-activity"), workerRole, taskQueue),
      Vector(
        Item(performs =
          Vector(Performance(attemptStart, Command("run-attempt", Finish(Literal(Text("done"))))))
        )
      )
    )
  ),
  learned = Vector(Learned(activityRun, LearnedKind.text)),
  observations =
    Vector(Observed("correlated-evidence", "temporal.server.api.testpilot.v1.CorrelatedEvidence")),
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
  query("door.opens") find opensBecauseTheLatchGives in pushed limits Limits("one", steps = 1, actions = 1, search = 16)

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
                Rpc(
                  workflowService,
                  service + "StartWorkflowExecution",
                  Vector(
                    Assignment("namespace", Environment(namespace)),
                    Assignment("workflow_id", Run)
                  )
                )
              )
            )
          )
        ),
        Item(command =
          Some(
            Command(
              "history",
              Rpc(
                workflowService,
                service + "GetWorkflowExecutionHistory",
                Vector(
                  Assignment("namespace", Environment(namespace)),
                  Assignment("execution.workflow_id", Run)
                ),
                Vector(
                  ResponseRead(
                    "history.events[*]",
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
  observations =
    Vector(Observed("correlated-evidence", "temporal.server.api.testpilot.v1.CorrelatedEvidence")),
  evidence = Vector(
    Evidence(
      id = "fixture.realizations.door.evidence.opened",
      records = "doorOpened",
      source = "fixture.realizations.door.source.history",
      from = Recorded.History("workflow_execution_started_event_attributes"),
      operation = "event_id",
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
  case ErrandState.idle => List(Step(ErrandOutcome.accepted, ErrandState.queued, List(ErrandFact.errandListed)))
  case _                => Nil

/** The runner is handed the attempt. Nothing a caller reads says so. */
def deliverStep(s: ErrandState): List[ErrandStep] = s match
  case ErrandState.queued | ErrandState.retrying => List(Step(ErrandOutcome.accepted, ErrandState.held))
  case _                                         => Nil

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
  case ErrandFact.errandListed => "errandListed"
  case ErrandFact.errandClosed => "errandClosed"
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
  query("errand.retry") find closesOnCompletion in retriedOnce limits Limits("five", steps = 5, actions = 5, search = 4096)

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
  query("errand.withdrawn") find withdrawsOnCancel in canceledOnce limits Limits("three", steps = 3, actions = 3, search = 4096)

private val errandType = Name("umpire-", fixture = true, suffix = "-errand")

/**
 * One kind of the errand's evidence: an entry of the namespace's listing, once it reads so. A poll's
 * condition reads the entry alone, so the listing is the run's own only in a namespace that holds no
 * other activity.
 */
private def listed(records: String) =
  Evidence(
    id = "fixture.realizations.errand.evidence." + records,
    records = records,
    source = "fixture.realizations.errand.source." + records,
    from = Recorded.Read(service + "ListActivityExecutions", "executions"),
    operation = "activity_id",
    commitment = Commitment.reported
  )

private def awaitListed(id: String, records: String, reads: Operand) =
  Command(
    id,
    Poll(
      "fixture.realizations.errand.evidence." + records,
      workflowService,
      Vector(Assignment("namespace", Environment(namespace))),
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
                Rpc(
                  workflowService,
                  service + "StartActivityExecution",
                  Vector(
                    Assignment("namespace", Environment(namespace)),
                    Assignment("activity_id", Run),
                    Assignment("activity_type.name", Literal(Named(errandType))),
                    Assignment("task_queue.name", Environment(queueResource)),
                    Assignment("request_id", Run),
                    Assignment("start_to_close_timeout.seconds", Literal(Number(300)))
                  )
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
              "errandListed",
              All(
                Present(Path(Projected, "schedule_time")),
                Greater(Path(Projected, "state_transition_count"), Literal(Number(0))),
                Not(Equal(Path(Projected, "activity_id"), Literal(Text(""))))
              )
            )
          )
        ),
        Item(command =
          Some(
            awaitListed(
              "await-closed",
              "errandClosed",
              Equal(Path(Projected, "status"), Literal(EnumName("ACTIVITY_EXECUTION_STATUS_COMPLETED")))
            )
          )
        ),
        Item(
          command = Some(
            awaitListed(
              "await-withdrawn",
              "errandWithdrawn",
              Equal(Path(Projected, "status"), Literal(EnumName("ACTIVITY_EXECUTION_STATUS_CANCELED")))
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
                  Proto(
                    "temporal.api.failure.v1.Failure",
                    ProtoField("message", Text("not yet")),
                    ProtoField(
                      "application_failure_info",
                      Message(
                        Proto("temporal.api.failure.v1.ApplicationFailureInfo", ProtoField("type", Text("NotYet")))
                      )
                    )
                  )
                )
              )
            ),
            Performance(answer(ErrandAnswer.completed), Command("complete-attempt", Finish(Literal(Text("done"))))),
            Performance(answer(ErrandAnswer.canceled), Command("cancel-attempt", AttemptCanceled))
          )
        )
      )
    )
  ),
  observations =
    Vector(Observed("correlated-evidence", "temporal.server.api.testpilot.v1.CorrelatedEvidence")),
  evidence = Vector(listed("errandListed"), listed("errandClosed"), listed("errandWithdrawn")),
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
  query("tally.opens") find tallyOpened in tallied limits Limits("one", steps = 1, actions = 1, search = 16)

/** A realization whose evidence keeps fields and is read from one message. */
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
                Rpc(
                  workflowService,
                  service + "StartActivityExecution",
                  Vector(Assignment("namespace", Environment(namespace)), Assignment("activity_id", Run))
                )
              )
            )
          )
        ),
        Item(command =
          Some(
            Command(
              "await-opened",
              Poll(
                "fixture.realizations.tally.evidence.opened",
                workflowService,
                Vector(Assignment("namespace", Environment(namespace)), Assignment("activity_id", Run)),
                Equal(Path(Projected, "attempt"), Literal(Number(1))),
                250
              ),
              closes = Vector("fixture.realizations.tally.evidence.opened")
            )
          )
        )
      )
    )
  ),
  observations =
    Vector(Observed("correlated-evidence", "temporal.server.api.testpilot.v1.CorrelatedEvidence")),
  evidence = Vector(
    Evidence(
      id = "fixture.realizations.tally.evidence.opened",
      records = "doorOpened",
      source = "fixture.realizations.tally.source.describe",
      from = Recorded.Single(service + "DescribeActivityExecution", "info"),
      operation = "activity_id",
      commitment = Commitment.reported,
      fields = Vector(
        EvidenceField("run", "run_id", role = Some(FieldRole.operation)),
        EvidenceField("attempt", "attempt", role = Some(FieldRole.attempt)),
        EvidenceField("identity", "last_worker_identity", redacted = true)
      ),
      exhaustive = true
    ),
    // The start call's own answer, as the Run records it. A machine's evidence names each fact once,
    // so this second kind confirms a fact the door does not record and is off every path.
    Evidence(
      id = "fixture.realizations.tally.evidence.pushed",
      records = "doorPushed",
      source = "fixture.realizations.tally.source.pushed",
      from = Recorded.RunEvent(
        EventKind.instructionCompleted,
        "controller",
        "push-door",
        key = Run,
        guard = Some(Equal(Path(Projected, "status"), Literal(EnumName("INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"))))
      ),
      operation = "",
      commitment = Commitment.reported
    )
  )
)
