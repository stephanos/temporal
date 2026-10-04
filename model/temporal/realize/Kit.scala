/* The shared Temporal realization kit: what every Temporal realization says alike, written once.
 *
 * A Temporal Case runs against a deployment the run sets up: the frontend's WorkflowService, the
 * Case's own worker and the task queues it polls, each a role with the environment bindings the run
 * supplies for it. Its evidence is lifted into one correlated record, keyed by the run and by the
 * operation, and checked within one window. Its controller is one script, and a read it waits on
 * polls at the kit's one interval. What a feature says differently is a parameter: its family, which
 * names its Definition IDs, the roles it addresses, the operation its evidence is keyed by, its
 * scripts and its evidence.
 *
 * The standalone activity and the Nexus caller realizations build on it. Everything here is a
 * declaration the lifter reads, by value, into the realizations that use it.
 */
package temporal.realize

import umpire.{Entity, Family, Machine}
import umpire.realize.*
import temporal.server.api.testpilot.v1.{
  ActivityAttempt,
  CorrelatedEvidence,
  InstructionOutcome,
  InstructionOutcomeStatus
}

// ### Roles, and the environment bindings a run supplies for them

private val workerNamespaceBinding = "temporal.worker.namespace"
private val taskQueueBinding = "temporal.task-queue.resource"
private val handlerTaskQueueBinding = "temporal.handler-task-queue.resource"
private val nexusEndpointBinding = "temporal.nexus-endpoint.resource"

/** The frontend's WorkflowService: the endpoint every call of a controller is made on. */
val workflowService: Role = Role("temporal.workflow-service", RoleKind.endpoint)

/** The Case's own worker, in the namespace the run supplies. */
val caseWorker: Role = Role("temporal.worker", RoleKind.worker, namespace = workerNamespaceBinding)

/** The task queue the Case's worker polls, in the namespace the run supplies. */
val taskQueue: Role = Role(
  "temporal.task-queue",
  RoleKind.taskQueue,
  namespace = workerNamespaceBinding,
  resource = taskQueueBinding
)

/** The task queue a Nexus handler's worker polls, apart from the caller's. */
val handlerTaskQueue: Role = Role(
  "temporal.handler-task-queue",
  RoleKind.taskQueue,
  namespace = workerNamespaceBinding,
  resource = handlerTaskQueueBinding
)

/** The Nexus endpoint a caller schedules its operations on. */
val nexusEndpoint: Role =
  Role("temporal.nexus-endpoint", RoleKind.endpoint, resource = nexusEndpointBinding)

/** The namespace the run supplies, as a request names it. */
val workerNamespace: TypedOperand[String] = Operand.environment(workerNamespaceBinding)

/** The name of the task queue the Case's worker polls. */
val taskQueueName: TypedOperand[String] = Operand.environment(taskQueueBinding)

/** The run's own id. A Case starts one operation, under that id. */
val run: TypedOperand[String] = Operand.run()

/** A name each Case gets its own copy of, such as the activity or workflow type it runs. */
def perCase(kind: String): Name = Name("umpire-", fixture = true, suffix = "-" + kind)

// ### Deadlines a request sets

/** The deadline, in seconds, a path whose timer fires sets: one a Case lives to see expire. */
val deadlineSeconds: Long = 2

/**
 * A deadline, in seconds, no Case lives to see: what a request carries where the server refuses one
 * that sets none and the path sets none.
 */
val unreachedDeadlineSeconds: Long = 300

/** `deadlineSeconds` as a request field's value. */
val deadline: TypedOperand[Long] = Operand.number(deadlineSeconds)

/** `deadlineSeconds` in milliseconds, as a timer step's deadline names it. */
val deadlineMs: Long = deadlineSeconds * 1000

/** `unreachedDeadlineSeconds` as a request field's value. */
val unreachedDeadline: TypedOperand[Long] = Operand.number(unreachedDeadlineSeconds)

// ### Evidence: Definition IDs, the correlated record and the window

/** The Definition ID of one kind of evidence of the family: `<family>.evidence.<kind>`. */
def evidenceId(kind: Fact)(using family: Family): String = family.root + ".evidence." + kind

/** The Definition ID of one source evidence is counted in: `<family>.source.<name>`. */
def sourceId(name: Fact)(using family: Family): String = family.root + ".source." + name

/**
 * The Run's own record. The Run numbers what it records, so every kind read from it counts in this
 * one source, in the order the Run recorded it.
 */
def runRecord(using Family): String = sourceId("record")

/** The observation every Temporal realization lifts its evidence into, one correlated record. */
val correlatedEvidence: String = "correlated-evidence"

/** The correlated record as a realization observes it. */
val correlated: Observed = Observed[CorrelatedEvidence](correlatedEvidence)

/**
 * The fields that scope a run's evidence to its run and to the `operation`, the entity the machine
 * is of (Definition IDs `<family>.projection`, `<family>.scope.run` and
 * `<family>.scope.<entity>`), and the window a check of it keeps: every Temporal Case keeps the same.
 */
def correlation(operation: Entity)(using family: Family): Correlation = Correlation(
  projection = family.root + ".projection",
  run = family.root + ".scope.run",
  operation = family.root + ".scope." + operation.name,
  observation = correlatedEvidence,
  events = 32,
  buffered = 16,
  keys = 8,
  support = 128,
  work = 1000000000,
  eventSize = 512
)

/**
 * How `machine`'s find Queries run on Temporal: `roles`, `evidence` keyed by `operation`, the
 * `requiredSettings`, the `serverSteps` no command performs, the kit's correlation and cleanup, and
 * the API `behavior` Behavior.scala declares, which only a lifter fixture replaces; its val's name,
 * `<family>.testpilot`'s.
 */
def temporalRealization(
    machine: Machine[?, ?, ?],
    operation: Entity,
    roles: Vector[Role],
    scripts: Vector[Script],
    evidence: Vector[Evidence | EvidenceRef[?, ?] | TypedEvidence[?]],
    learned: Vector[Learned] = Vector.empty,
    observations: Vector[Observed] = Vector(correlated),
    controls: Vector[Actuator] = Vector.empty,
    requiredSettings: Vector[RequiredSetting] = Vector.empty,
    serverSteps: Vector[ServerStep] = Vector.empty,
    behavior: ApiBehavior = temporalBehavior
)(using family: Family): Realization = Realization(
  machine = machine,
  producer = family.root + ".testpilot",
  producerVersion = "1",
  roles = roles,
  correlation = correlation(operation),
  scripts = scripts,
  learned = learned,
  observations = observations,
  evidence = evidence,
  controls = controls,
  cleanup = "cleanup",
  requiredSettings = requiredSettings,
  behavior = Some(behavior),
  serverSteps = serverSteps
)

// ### The controller and its reads

/** The id of a Case's controller script, the one script the Run records the calls of. */
val controllerScript: String = "controller"

/** The controller's script: the calls and controls of a Case, in the order its path makes them. */
def controller(items: Item*): Script = script(controllerScript, Activation.Controller)(items*)

/** How often a read polls until it sees what it waits for. No call site writes an interval. */
private val pollIntervalMs: Long = 250

/**
 * Reads `evidence` on `role` until an element satisfies `until`, the request's fields assigned in
 * the scope it opens: the one form a realization waits in, so the interval it polls at is the kit's.
 */
def await[Req, Projected](evidence: EvidenceRef[Req, Projected], role: Role)(
    until: Condition[Projected]
)(assign: RequestScope[Req] ?=> Unit): Instruction =
  poll(evidence, role, until, pollIntervalMs)(assign)

/** That a command of the controller succeeded, as the Run records its outcome. */
val succeeded: Condition[InstructionOutcome] = Condition.equal(
  Field[InstructionOutcome, InstructionOutcomeStatus](_.status),
  Operand.enumValue(InstructionOutcomeStatus.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)
)

/**
 * What the server answered a call of the controller, where it succeeded: the Run's record of the
 * call's completion, the kind `kind` that confirms `records`. The run starts one operation, under its
 * own id, so the run is the key.
 */
def answeredAs(kind: Fact, records: Fact, call: Command | Instruction, confirms: Taking*)(using
    Family
): TypedEvidence[InstructionOutcome] = Evidence.runEvent(
  id = evidenceId(kind),
  records = records,
  source = runRecord,
  from = Recorded.runEvent[InstructionOutcome](
    EventKind.instructionCompleted,
    controllerScript,
    call,
    key = Operand.runKey(),
    guard = Some(succeeded)
  ),
  commitment = Commitment.reported,
  confirms = Vector(confirms*)
)

/** `answeredAs` for the kind named after the fact it confirms. */
def answered(fact: Fact, call: Command | Instruction, confirms: Taking*)(using
    Family
): TypedEvidence[InstructionOutcome] = answeredAs(fact, fact, call, confirms*)

/**
 * What the Case's worker reports of one attempt it was delivered: the Run's record of an activation
 * the call carries, declared the record of the attempt of the script `attempts` that the server
 * numbers `number`, counted from 1. The Run records an attempt once it is answered, so this evidence
 * reaches a Run with that answer, and the declaration is what says so and which record it is. A
 * delivered attempt has a delivery; the record of a position no attempt was delivered for has an
 * empty one, which is a value a record holds like any other, so the guard asks for a delivery that is
 * not empty. One record is one piece of evidence, so each attempt has a kind of its own. What the
 * worker then offered the server is not read: an offer is not the server's acceptance.
 */
def delivered(
    fact: Fact,
    attempts: Script,
    number: Long,
    call: Command | Instruction,
    confirms: Taking*
)(using Family): TypedEvidence[InstructionOutcome] = Evidence.runEvent(
  id = evidenceId(fact),
  records = fact,
  source = runRecord,
  from = Recorded.runEvent[InstructionOutcome](
    EventKind.diagnostic,
    controllerScript,
    call,
    key = Operand.runKey(),
    guard = Some(
      Condition.all(
        Condition.present(Field[InstructionOutcome, Option[ActivityAttempt]](_.activityAttempt)),
        Condition.not(
          Condition.equal(
            Field[InstructionOutcome, String](_.getActivityAttempt.deliveryId),
            Operand.text("")
          )
        )
      )
    ),
    attempt = Some(AttemptOf(attempts, number))
  ),
  commitment = Commitment.reported,
  fields = Vector(
    attemptField(Field(_.getActivityAttempt.sdkAttempt)),
    deliveryField(Field(_.getActivityAttempt.deliveryId)),
    activityRunField(Field(_.getActivityAttempt.activityRunId))
  ),
  confirms = Vector(confirms*)
)

// ### The fields a Run's record of an attempt carries, read at the record's own path

/** The attempt's number, as the server counts attempts. */
def attemptField(path: Field[InstructionOutcome, Int]) =
  EvidenceField.typed("attempt", path, role = Some(FieldRole.attempt))

/** The stamp of the delivery, which tells two deliveries of one attempt apart. */
def deliveryField(path: Field[InstructionOutcome, String]) =
  EvidenceField.typed("delivery", path, role = Some(FieldRole.delivery))

/** The activity run the attempt belongs to. */
def activityRunField(path: Field[InstructionOutcome, String]) =
  EvidenceField.typed("activityRun", path)

// ### The endpoint a standalone Nexus operation names

/** The name of the Nexus endpoint the run creates for the Case, as a start request names it. */
val nexusEndpointName: TypedOperand[String] = Operand.environment(nexusEndpointBinding)
