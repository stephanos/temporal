package temporal.realize

import framework.dsl.*
import framework.realize.*
import io.temporal.api.common.v1.Payload
import io.temporal.api.history.v1.HistoryEvent
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*
import io.temporal.server.api.umpire.v1 as ir
import scalapb.GeneratedMessage

private object ConversionFixture:
  enum ConversionFact:
    case completed

  enum ForeignFact:
    case completed

  enum Timeout:
    case unset, expires

  object ConversionMachine extends Machine[Boolean, Boolean, ConversionFact]:
    val init = false
    def end(state: Boolean): Boolean = state

    object rules extends Rules

  val conversionComplete =
    finish("done").named(
      "complete",
      "fixture.complete",
      "ConstructedTemporal.test.scala:17"
    )
  val conversionFault =
    fault(caseWorker, FaultKind.workerStop)
      .named(
        "stopWorker",
        "fixture.stopWorker",
        "ConstructedTemporal.test.scala:20"
      )
  val conversionReplacement =
    finish("replacement")
      .named(
        "replacement",
        "fixture.replacement",
        "ConstructedTemporal.test.scala:23"
      )
  val conversionDeliveryAction =
    val action = framework.dsl.action("delivery", Actor("server"))
    val retained = classOf[framework.dsl.Action[?]]
      .getMethod("declaredAt", classOf[String])
      .invoke(action, "fixture.delivery")
    require(retained.eq(action))
    action
  val conversionDelivery = conversionDeliveryAction()

  val deadlineInput = input[Timeout](default = Timeout.unset)
  val deadlineAction =
    val action =
      framework.dsl.action("deadline", Actor("server")).input(deadlineInput)
    val retained = classOf[framework.dsl.Action[?]]
      .getMethod("declaredAt", classOf[String])
      .invoke(action, "fixture.deadline")
    require(retained.eq(action))
    action
  val deadlineCall = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {}
    .named(
      "deadlineCall",
      "fixture.deadlineCall",
      "ConstructedTemporal.test.scala:47"
    )
  val messageAssignmentCall =
    rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {
      field(_.getStartToCloseTimeout) := duration(3)
    }.named(
      "messageAssignmentCall",
      "fixture.messageAssignmentCall",
      "ConstructedTemporal.test.scala:51"
    )
  val deadlineDeclaration = deadlines[StartActivityExecutionRequest](
    deadlineAction,
    deadlineCall,
    duration(deadlineSeconds)
  )(deadlineInput.sets(_.getStartToCloseTimeout))

  val settlementCarrier = finish("carrier")
    .named(
      "settlementCarrier",
      "fixture.settlementCarrier",
      "ConstructedTemporal.test.scala:55"
    )
  val settlementAnswer = rpc(
    workflowService,
    METHOD_RESPOND_ACTIVITY_TASK_COMPLETED_BY_ID
  ) {}.named(
    "settlementAnswer",
    "fixture.settlementAnswer",
    "ConstructedTemporal.test.scala:59"
  )
  val settlementDone = finish("settled")
    .named(
      "settlementDone",
      "fixture.settlementDone",
      "ConstructedTemporal.test.scala:61"
    )
  val settlementCleanup = command(finish("cleanup"))
    .named(
      "settlementCleanup",
      "fixture.settlementCleanup",
      "ConstructedTemporal.test.scala:63"
    )
  val resetHeld = finish("held")
    .named(
      "resetHeld",
      "fixture.resetHeld",
      "ConstructedTemporal.test.scala:65"
    )
  val resetRequest = rpc(workflowService, METHOD_RESET_ACTIVITY_EXECUTION) {}
    .named(
      "resetRequest",
      "fixture.resetRequest",
      "ConstructedTemporal.test.scala:67"
    )
  val scheduledSettlement = ActivityExternalSettlement.Scheduled(
    settlementCarrier,
    settlementAnswer,
    settlementDone,
    settlementCleanup
  )

  val historyAttributes = Field[HistoryEvent, Option[GeneratedMessage]](
    _.attributes.activityTaskScheduledEventAttributes
  )
  val historyEvidence = HistoryEvidence("scheduled_event_id", "")(
    HistoryKind(
      ConversionFact.completed,
      _.attributes.activityTaskScheduledEventAttributes
    )
  ).evidence.head
  val foreignHistoryEvidence = Evidence.keyed(
    id = "foreign-history",
    records = ForeignFact.completed,
    source = "history-source",
    from = WorkflowHistory.keyedEvent(historyAttributes, "scheduled_event_id"),
    operation = historyAttributes,
    commitment = Commitment.reported,
    exhaustive = true
  )

  val conversionAttempts =
    script(
      "attempts",
      WorkerActivation.Activity(
        perCase("work"),
        caseWorker,
        taskQueue,
        starts = Vector(conversionDelivery)
      )
    )(everyCase(conversionFault))

  object ConversionBase
      extends Realizes(
        ConversionMachine,
        learned = Vector(Learned("result", LearnedKind.text)),
        observations = Vector(Observed[Payload]("record")),
        requiredSettings = Vector(RequiredSetting("feature.enabled", "true"))
      ):

    object controller extends Controller(everyCase(conversionComplete))

    object workers extends Workers(conversionAttempts)

    object serverSteps
        extends ServerSteps(
          ServerStep(
            conversionDelivery,
            CauseKind.timer,
            Some(timerDeadline),
            TimeoutBasis.startToClose
          )
        )

  object ConversionDerived extends DerivesFrom(ConversionBase, ConversionMachine):

    object changes
        extends Changes(
          replacing(conversionComplete)(everyCase(conversionReplacement))
        )

  object DeadlineRealization extends Realizes(ConversionMachine):

    object controller extends Controller(deadlineDeclaration)

  object MessageAssignmentRealization extends Realizes(ConversionMachine):

    object controller extends Controller(everyCase(messageAssignmentCall))

  object SettlementRealization extends Realizes(ConversionMachine):

    object serverSteps
        extends ServerSteps(
          scheduledSettlement,
          ActivityResetSettlement(
            carrier = settlementCarrier,
            activity = conversionAttempts,
            attempt = 1,
            pending = ActivityPublication("pending-reset"),
            held = resetHeld,
            resetRequest = resetRequest,
            timer = conversionDelivery,
            freshAttempt = 2,
            settlement = settlementDone,
            cleanup = settlementCleanup
          )
        )

  object HistoryRealization extends Realizes(ConversionMachine):

    object evidence extends Evidences(historyEvidence)

  object ForeignHistoryRealization extends Realizes(ConversionMachine):

    object evidence extends Evidences(foreignHistoryEvidence)

class TemporalRealizationConversionTest extends munit.FunSuite:
  import ConversionFixture.*

  private def deliveryIdentity: String =
    ConversionBase.dependencies.actions.head match
      case ConstructedRealization.ActionDependency.Declaration(identity) =>
        identity
      case other => fail(s"delivery dependency converted as $other")

  private def context(declaration: String) = ConstructedRealization.Context(
    declaration,
    ir.Model(
      machines = Vector(
        ir.Machine(
          name = ConversionMachine.name,
          entity = "activity",
          factType = "fixture.ConversionFact"
        )
      ),
      actions = Vector(ir.Action(id = deliveryIdentity, name = "delivery")),
      types = Vector(
        ir.Type(
          name = "fixture.ConversionFact",
          shape = ir.Type.Shape.Enum(
            ir.Enum(cases = Vector(ir.Case(name = "completed")))
          )
        )
      )
    ),
    Some("fixture")
  )

  private def deadlineContext = context("fixture.DeadlineRealization").copy(
    metadata = context("fixture.DeadlineRealization").metadata.copy(
      actions = Vector(
        ir.Action(
          id = "fixture.deadline",
          name = "deadline",
          inputs = Vector(
            ir.Param(
              "deadlineInput",
              Some(ir.TypeRef(ir.TypeRef.Ref.Named("fixture.Timeout")))
            )
          )
        )
      ),
      types = context("fixture.DeadlineRealization").metadata.types :+ ir.Type(
        name = "fixture.Timeout",
        shape = ir.Type.Shape.Enum(
          ir.Enum(cases = Vector(ir.Case(name = "unset"), ir.Case(name = "expires")))
        )
      )
    )
  )

  test(
    "constructed conversion retains Temporal sections, roles, settings and behavior"
  ):
    val converted = ConversionBase.toIR(context("fixture.ConversionBase"))

    assertEquals(converted.id, "fixture.ConversionBase")
    assertEquals(converted.name, "conversionBase")
    assertEquals(converted.machine, ConversionMachine.name)
    assertEquals(converted.producer, "fixture.testpilot")
    assertEquals(converted.learned.map(_.id), Vector("result"))
    assertEquals(converted.observations.map(_.id), Vector("record"))
    assertEquals(
      converted.requiredSettings,
      Vector(ir.RequiredSetting("feature.enabled", "true"))
    )
    assert(converted.behavior.nonEmpty)
    assert(converted.getBehavior.visibility.nonEmpty)
    assert(
      converted.getBehavior.causes.exists(
        _.kind.equals(ir.CauseKind.CAUSE_KIND_TIMER)
      )
    )
    assert(
      converted.getBehavior.visibility.exists(
        _.write.equals(
          ir.Visibility.Write.Cause(ir.CauseKind.CAUSE_KIND_ACTIVITY_HEARTBEAT)
        )
      )
    )
    assertEquals(converted.scripts.map(_.id), Vector("controller", "attempts"))
    assertEquals(
      converted.roles.map(_.id),
      Vector(
        "temporal.workflow-service",
        "temporal.worker",
        "temporal.task-queue"
      )
    )
    assertEquals(converted.serverSteps.size, 1)
    assertEquals(converted.serverSteps.head.getStep.action, deliveryIdentity)
    assertEquals(converted.serverSteps.head.kind, ir.CauseKind.CAUSE_KIND_TIMER)
    assertEquals(converted.serverSteps.head.deadline, Some(timerDeadline))
    assertEquals(
      converted.serverSteps.head.timeoutBasis,
      ir.TimeoutBasis.TIMEOUT_BASIS_START_TO_CLOSE
    )
    converted.scripts(1).items.head.getCommand.instruction match
      case ir.Command.Instruction.Fault(value) =>
        assertEquals(value.role, "temporal.worker")
        assertEquals(value.kind, ir.Fault.Kind.KIND_WORKER_STOP)
      case other => fail(s"worker instruction converted as $other")

  test(
    "dependencies retain exact Temporal action identities once in traversal order"
  ):
    assertEquals(
      ConversionBase.dependencies.actions,
      Vector(
        ConstructedRealization.ActionDependency.Declaration(
          "fixture.delivery"
        )
      )
    )
    assertEquals(ConversionBase.dependencies.channels, Vector.empty)

  test(
    "deadline conversion expands native Timeout classes and request assignments"
  ):
    val converted = DeadlineRealization.toIR(deadlineContext)
    val performs = converted.scripts.head.items.head.performs

    assertEquals(performs.size, 2)
    assertEquals(
      performs.map(_.getStep.inputs.head.getEnum.`case`).toSet,
      Set("unset", "expires")
    )
    assertEquals(performs.map(_.getCommand.id).toSet, Set("deadline-call"))
    val assigned = performs.flatMap(_.getCommand.instruction match
      case ir.Command.Instruction.Rpc(call) => call.assign.map(_.target)
      case other                            => fail(s"deadline command converted as $other"))
    assertEquals(assigned, Vector("start_to_close_timeout.seconds"))

  test(
    "message-valued request sugar expands leaves in an ordinary RPC"
  ):
    val converted = MessageAssignmentRealization.toIR(
      context("fixture.MessageAssignmentRealization")
    )
    val assignments =
      converted.scripts.head.items.head.getCommand.instruction match
        case ir.Command.Instruction.Rpc(call) => call.assign
        case other                            => fail(s"message assignment converted as $other")

    assertEquals(
      assignments.map(_.target),
      Vector("start_to_close_timeout.seconds")
    )
    assertEquals(
      assignments.head.getValue.kind,
      ir.Operand.Kind.Literal(ir.ProtoValue(ir.ProtoValue.Kind.Number(3)))
    )

  test(
    "external and reset settlements retain command, script and timer identities"
  ):
    val converted =
      SettlementRealization.toIR(context("fixture.SettlementRealization"))

    assertEquals(converted.externalSettlements.size, 1)
    assertEquals(
      converted.externalSettlements.head.carrier,
      "settlement-carrier"
    )
    assertEquals(converted.externalSettlements.head.answer, "settlement-answer")
    assertEquals(
      converted.externalSettlements.head.settlement,
      "settlement-done"
    )
    assertEquals(
      converted.externalSettlements.head.getCleanup.id,
      "settlement-cleanup"
    )
    assertEquals(converted.resetSettlements.size, 1)
    assertEquals(converted.resetSettlements.head.activity, "attempts")
    assertEquals(converted.resetSettlements.head.pending, "pending-reset")
    assertEquals(
      converted.resetSettlements.head.getTimer.action,
      deliveryIdentity
    )
    assertEquals(converted.resetSettlements.head.freshAttempt, 2L)

  test("history evidence retains the owned fact and native timing selections"):
    val converted =
      HistoryRealization.toIR(context("fixture.HistoryRealization"))
    val evidence = converted.evidence.head

    assertEquals(evidence.id, "fixture.evidence.completed")
    assertEquals(evidence.source, "fixture.source.history")
    assertEquals(evidence.records, "completed")
    assertEquals(
      evidence.from,
      ir.Evidence.From.History("activity_task_scheduled_event_attributes")
    )
    assertEquals(
      evidence.operation,
      "attributes<activity_task_scheduled_event_attributes>.scheduled_event_id"
    )
    assert(evidence.exhaustive)

  test("history evidence refuses a same-spelled fact from a foreign machine"):
    val failure = intercept[IllegalArgumentException](
      ForeignHistoryRealization.toIR(
        context("fixture.ForeignHistoryRealization")
      )
    )

    assert(
      failure.getMessage.startsWith("fixture.ForeignHistoryRealization:"),
      failure.getMessage
    )
    assert(
      failure.getMessage.contains("native machine fact catalog"),
      failure.getMessage
    )

  test(
    "derived conversion replaces base controller items and inherits its header"
  ):
    assertEquals(
      ConversionBase
        .toIR(context("fixture.ConversionBase"))
        .scripts
        .head
        .items
        .map(_.getCommand.id),
      Vector("complete")
    )
    val converted = ConversionDerived.toIR(context("fixture.ConversionDerived"))

    assertEquals(converted.name, "conversionDerived")
    assertEquals(
      converted.scripts.head.items.map(_.getCommand.id),
      Vector("replacement")
    )
    assertEquals(converted.learned.map(_.id), Vector("result"))
    assertEquals(
      converted.requiredSettings,
      Vector(ir.RequiredSetting("feature.enabled", "true"))
    )

  test("conversion refuses missing family provenance at the declaration"):
    val missing = context("fixture.Missing").copy(sourcePackage = None)
    val failure =
      intercept[IllegalArgumentException](ConversionBase.toIR(missing))

    assertEquals(
      failure.getMessage,
      "fixture.Missing: family has no declaring package context"
    )

  test("conversion refuses an unknown section at the declaration"):
    class Invalid extends Realizes(ConversionMachine):
      object surprise extends Controller()
    val invalid = new Invalid
    assertEquals(invalid.surprise.items, Seq.empty)

    val failure = intercept[IllegalArgumentException](
      invalid.toIR(context("fixture.Invalid"))
    )
    assert(
      failure.getMessage.startsWith("fixture.Invalid:"),
      failure.getMessage
    )
    assert(failure.getMessage.contains("surprise"), failure.getMessage)
