package temporal.realize

import framework.dsl.*
import framework.realize.*
import io.temporal.api.common.v1.{Payload, Payloads}
import io.temporal.api.failure.v1.{ApplicationFailureInfo, Failure}
import io.temporal.api.history.v1.HistoryEvent
import io.temporal.api.enums.v1.ActivityExecutionStatus
import temporal.server.api.testpilot.v1.{ActivityAttempt, InstructionOutcome}
import io.temporal.api.workflowservice.v1.{
  DescribeActivityExecutionResponse,
  StartActivityExecutionRequest
}
import io.temporal.api.workflowservice.v1.WorkflowServiceGrpc.*

private enum ConstructionFact:
  case completed, failed, canceled, timedOut

private enum ConstructionTimeout:
  case unset, expires

private object ConstructionMachine extends Machine[Boolean, Boolean, ConstructionFact]:
  val init = false
  def end(s: Boolean): Boolean = s

  object rules extends Rules

class TemporalRealizationConstructionTest extends munit.FunSuite:
  test("nested proto scopes retain message identities, paths and literal types"):
    val failure = proto[Failure] {
      field(_.message) := "failed"
      field(_.getApplicationFailureInfo) {
        field(_.`type`) := "Retryable"
        field(_.nonRetryable) := false
      }
    }
    assertEquals(failure.message, "temporal.api.failure.v1.Failure")
    assertEquals(
      failure.fields.map(_.field.path),
      Vector(Some("message"), Some("failure_info<application_failure_info>"))
    )
    failure.fields(1).value.value match
      case info: TypedProto[?] =>
        assertEquals(info.message, "temporal.api.failure.v1.ApplicationFailureInfo")
        assertEquals(info.fields.map(_.field.path), Vector(Some("type"), Some("non_retryable")))
        assertEquals(
          info.fields.map(_.value.value),
          Vector(ProtoValue.Text("Retryable"), ProtoValue.Flag(false))
        )
      case other => fail(s"nested literal discarded its fields: $other")
    assertEquals(
      Field[Failure, Boolean](_.getApplicationFailureInfo.nonRetryable).path,
      Some("failure_info<application_failure_info>.non_retryable")
    )

  test("request message assignments retain their typed protobuf values"):
    val request = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {
      field(_.getStartToCloseTimeout) := duration(7)
    }
    assertEquals(request.assign.head.target.path, Some("start_to_close_timeout"))
    assertEquals(
      request.requestMessage,
      "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
    )
    assertEquals(
      request.responseMessage,
      "temporal.api.workflowservice.v1.StartActivityExecutionResponse"
    )
    request.assign.head.value.operand match
      case Operand.Message(value) =>
        assertEquals(value.message, "google.protobuf.Duration")
        assertEquals(value.fields.map(_.value.value), Vector(ProtoValue.Number(7)))
      case other => fail(s"request message assignment discarded its literal: $other")

  test("scoped rpc captures schema names with exactly one evaluation of role, method and scope"):
    var roles = 0
    var methods = 0
    var scopes = 0
    def role: Role =
      roles += 1
      workflowService
    def method =
      methods += 1
      METHOD_START_ACTIVITY_EXECUTION
    val call = rpc(role, method) {
      scopes += 1
      field(_.namespace) := Operand.text("selected")
    }
    assertEquals((roles, methods, scopes), (1, 1, 1))
    assertEquals(call.assign.head.value.operand, Operand.Literal(ProtoValue.Text("selected")))
    assertEquals(
      call.requestMessage,
      "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
    )
    assertEquals(
      call.responseMessage,
      "temporal.api.workflowservice.v1.StartActivityExecutionResponse"
    )

  test("ordinary generic rpc helpers retain names through standard companion evidence"):
    def generic[Req <: scalapb.GeneratedMessage, Rsp <: scalapb.GeneratedMessage](
        method: io.grpc.MethodDescriptor[Req, Rsp]
    )(using scalapb.GeneratedMessageCompanion[Req], scalapb.GeneratedMessageCompanion[Rsp]) =
      rpc(workflowService, method) {}
    val call = generic(METHOD_START_ACTIVITY_EXECUTION)
    assertEquals(
      call.requestMessage,
      "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
    )
    assertEquals(
      call.responseMessage,
      "temporal.api.workflowservice.v1.StartActivityExecutionResponse"
    )

  test("literal conversion retains bytes, maps, roles, names and message sequences"):
    val payload = proto[Payload] {
      field(_.metadata) := Map("encoding" -> "json/plain")
      field(_.data) := "body"
    }
    payload.fields.head.value.value match
      case entries: Vector[?] =>
        entries.head match
          case entry: TypedProtoEntry =>
            assertEquals(entry.key, "encoding")
            assertEquals(entry.value.value, ProtoValue.Utf8("json/plain"))
          case other => fail(s"mapping discarded entry $other")
      case other => fail(s"mapping discarded values $other")
    assertEquals(payload.fields(1).value.value, ProtoValue.Utf8("body"))
    val payloads =
      Proto[Payloads](ProtoField.typed(Field(_.payloads), ProtoValue.messages(payload)))
    payloads.fields.head.value.value match
      case values: Vector[?] => assertEquals(values.size, 1); assert(values.head.equals(payload))
      case other             => fail(s"messages discarded values $other")
    val request = proto[StartActivityExecutionRequest] {
      field(_.getTaskQueue.name) := taskQueue
      field(_.getActivityType.name) := perCase("work")
    }
    assertEquals(
      request.fields.map(_.value.value),
      Vector(
        ProtoValue.RoleId("temporal.task-queue"),
        ProtoValue.Named(Name("umpire-", fixture = true, suffix = "-work"))
      )
    )
    val core = ProtoValue.text("literal")
    val info = proto[ApplicationFailureInfo](field(_.`type`) := core)
    assert(info.fields.head.value eq core)

  test("read evidence retains operation selector and all field metadata"):
    val selection =
      Field[DescribeActivityExecutionResponse, io.temporal.api.activity.v1.ActivityExecutionInfo](
        _.getInfo
      )
    val operation = Field[io.temporal.api.activity.v1.ActivityExecutionInfo, String](_.activityId)
    val field = EvidenceField.typed(
      "attempt",
      Field[io.temporal.api.activity.v1.ActivityExecutionInfo, Int](_.attempt),
      role = Some(FieldRole.attempt),
      redacted = true
    )
    val read = Recorded.single(METHOD_DESCRIBE_ACTIVITY_EXECUTION, selection)
    read.recorded match
      case value: Recorded.TypedSingle[?, ?, ?] =>
        assertEquals(
          value.requestMessage,
          "temporal.api.workflowservice.v1.DescribeActivityExecutionRequest"
        )
        assertEquals(
          value.responseMessage,
          "temporal.api.workflowservice.v1.DescribeActivityExecutionResponse"
        )
        assertEquals(value.projectedMessage, "temporal.api.activity.v1.ActivityExecutionInfo")
      case other => fail(s"Recorded.single discarded protobuf type arguments: $other")
    val evidence = Evidence.read(
      "completed",
      ConstructionFact.completed,
      "description",
      read,
      operation,
      Commitment.durable,
      fields = Vector(field),
      exhaustive = true
    )
    assertEquals(evidence.operation.path, Some("activity_id"))
    assertEquals(
      evidence.fields.map(f => (f.id, f.path.path, f.role, f.redacted)),
      Vector(("attempt", Some("attempt"), Some(FieldRole.attempt), true))
    )
    assert(evidence.evidence.from eq read.recorded)
    assertEquals(evidence.evidence.id, "completed")
    assertEquals(evidence.evidence.source, "description")
    assertEquals(evidence.evidence.commitment, Commitment.durable)
    assert(evidence.evidence.exhaustive)

  test("DescribedStatus retains machine, call base and selectors"):
    val calls = RequestBase(workflowService, "namespace" -> workerNamespace, "activity_id" -> run)
    val described = DescribedStatus(
      ConstructionMachine,
      calls,
      METHOD_DESCRIBE_ACTIVITY_EXECUTION,
      Field(_.getInfo),
      Field(_.activityId),
      Field(_.status)
    )(
      ConstructionFact.completed -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_COMPLETED,
      ConstructionFact.failed -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_FAILED,
      ConstructionFact.canceled -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_CANCELED,
      ConstructionFact.timedOut -> ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_TIMED_OUT
    )
    assert(described.machine eq ConstructionMachine)
    assert(described.calls eq calls)
    assert(described.method eq METHOD_DESCRIBE_ACTIVITY_EXECUTION)
    assertEquals(described.info.path, Some("info"))
    assertEquals(described.operation.path, Some("activity_id"))
    assertEquals(described.status.path, Some("status"))
    assertEquals(described.entries.size, 4)
    assertEquals(
      calls.fields.map((path, value) => path -> value.operand),
      Seq(
        "namespace" -> Operand.Environment("temporal.worker.namespace"),
        "activity_id" -> Operand.Run
      )
    )
    var evaluations = 0
    def completed =
      evaluations += 1
      ConstructionFact.completed
    described.await(completed) match
      case poll: Instruction.TypedPoll[?, ?] =>
        assertEquals(poll.until.operator, ConditionOperator.equal)
        assertEquals(
          poll.until.value.map(_.operand).collect { case Operand.Literal(value) =>
            value.productIterator.toVector
          },
          Some(Vector("ACTIVITY_EXECUTION_STATUS_COMPLETED"))
        )
        assert(poll.role.equals(calls))
        assertEquals(poll.ref.map(_.spelling), Some("await-completed"))
      case other => fail(s"described await discarded arguments $other")
    assertEquals(evaluations, 1)
    assertEquals(
      Vector(
        ConstructionFact.failed,
        ConstructionFact.canceled,
        ConstructionFact.timedOut
      ).map(described.await).map(_.ref.map(_.spelling)),
      Vector(Some("await-failed"), Some("await-canceled"), Some("await-timed-out"))
    )

  test("deadline declarations retain action, call, value, unset and field selections"):
    val timeout = input[ConstructionTimeout](default = ConstructionTimeout.unset)
    val action = framework.dsl.action("start", Actor("caller")).input(timeout)
    val call = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {}
    val fallback = command(call)
    val value = duration(3)
    val field = timeout.sets[StartActivityExecutionRequest](_.getStartToCloseTimeout)
    val item =
      deadlines[StartActivityExecutionRequest](action, call, value, Some(timeout -> fallback))(
        field
      )
    item.binding match
      case Some(binding: DeadlineBindings[?]) =>
        assert(binding.action.equals(action))
        assert(binding.call.equals(call))
        assert(binding.value eq value)
        assertEquals(binding.unset, Some(timeout -> fallback))
        assertEquals(binding.fields.map(_.selection.path), Vector(Some("start_to_close_timeout")))
        assert(binding.fields.head.input eq timeout)
        assertEquals(
          field.field(StartActivityExecutionRequest()),
          com.google.protobuf.duration.Duration()
        )
      case other => fail(s"deadlines discarded declaration $other")

  test("empty deadline field lists retain call identity independently of the unused M parameter"):
    val action = framework.dsl.action("start", Actor("caller"))
    val call = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {}
    val value = duration(3)
    val request = deadlines[StartActivityExecutionRequest](action, call, value)()
    val other = deadlines[Failure](action, call, value)()
    for item <- Vector(request, other) do
      item.binding match
        case Some(binding: DeadlineBindings[?]) =>
          assert(binding.action.equals(action))
          assert(binding.call.equals(call))
          assert(binding.value eq value)
          assertEquals(binding.fields.size, 0)
          assertEquals(binding.unset, None)
          binding.call match
            case rpc: Instruction.TypedRpc[?, ?] =>
              assertEquals(
                rpc.requestMessage,
                "temporal.api.workflowservice.v1.StartActivityExecutionRequest"
              )
              assertEquals(
                rpc.responseMessage,
                "temporal.api.workflowservice.v1.StartActivityExecutionResponse"
              )
            case other => fail(s"empty deadline list lost its call identity: $other")
        case other => fail(s"empty deadline list discarded its declaration: $other")

  test("HistoryKind captures indirect selectors while retaining the native function"):
    val kind =
      HistoryKind(ConstructionFact.completed, _.attributes.activityTaskScheduledEventAttributes)
    assertEquals(kind.selection.path, Some("attributes<activity_task_scheduled_event_attributes>"))
    assertEquals(kind.selection.rootMessage, Some("temporal.api.history.v1.HistoryEvent"))
    assertEquals(
      kind.selection.selectedMessage,
      Some("temporal.api.history.v1.ActivityTaskScheduledEventAttributes")
    )
    assertEquals(kind.attributes(HistoryEvent()), None)
    val module = HistoryEvidence("scheduled_event_id", "status")(kind)
    assertEquals(module.key, "scheduled_event_id")
    assertEquals(module.factPrefix, "status")
    assertEquals(module.kinds.size, 1)
    val evidence = module.evidence.head
    assertEquals(
      evidence.operation.flatMap(_.path),
      Some("attributes<activity_task_scheduled_event_attributes>")
    )
    evidence.evidence.from match
      case history: WorkflowHistory.TypedHistory[?, ?] =>
        assertEquals(history.operationKey, Some("scheduled_event_id"))
      case other => fail(s"history discarded its operation key: $other")
    HistoryEvidence("event_id", "status")(kind).evidence.head.evidence.from match
      case history: WorkflowHistory.TypedHistory[?, ?] =>
        assertEquals(history.operationKey, Some("event_id"))
      case other => fail(s"history discarded its second operation key: $other")

  test("selector factory arguments are evaluated once and native functions stay callable"):
    var historyFactories = 0
    def attributes: HistoryEvent => Option[scalapb.GeneratedMessage] =
      historyFactories += 1
      _.attributes.activityTaskScheduledEventAttributes
    val history = HistoryKind(ConstructionFact.completed, attributes)
    assertEquals(historyFactories, 1)
    assertEquals(history.attributes(HistoryEvent()), None)
    assertEquals(history.selection.path, None)
    var deadlineFactories = 0
    def timeoutField: StartActivityExecutionRequest => com.google.protobuf.duration.Duration =
      deadlineFactories += 1
      _.getStartToCloseTimeout
    val timeout = input[ConstructionTimeout](default = ConstructionTimeout.unset)
    val deadline = timeout.sets(timeoutField)
    assertEquals(deadlineFactories, 1)
    assertEquals(
      deadline.field(StartActivityExecutionRequest()),
      com.google.protobuf.duration.Duration()
    )
    assertEquals(deadline.selection.path, None)

  test("repeated and optional field paths follow descriptors without evaluating selectors"):
    assertEquals(
      Field[Payloads, Seq[com.google.protobuf.ByteString]](_.payloads.map(_.data)).path,
      Some("payloads[*].data")
    )
    assertEquals(
      Field[InstructionOutcome, Option[String]](_.activityAttempt.map(_.deliveryId)).path,
      Some("activity_attempt.delivery_id")
    )
    assertEquals(
      Field[InstructionOutcome, String](_.getActivityAttempt.deliveryId).path,
      Some("activity_attempt.delivery_id")
    )

  test("typed enum literals and widened numbers retain their declared values"):
    val info = proto[io.temporal.api.activity.v1.ActivityExecutionInfo] {
      field(_.status) := ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_COMPLETED
    }
    info.fields.head.value.value match
      case value: ProtoValue =>
        assertEquals(value.productIterator.toVector, Vector("ACTIVITY_EXECUTION_STATUS_COMPLETED"))
      case other => fail(s"enum literal discarded its name: $other")
    val value = proto[com.google.protobuf.duration.Duration](field(_.seconds) := 6)
    assertEquals(value.fields.head.value.value, ProtoValue.Number(6))
    assertEquals(
      Operand
        .enumValue(ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_COMPLETED)
        .operand
        .toString,
      "Literal(EnumName(ACTIVITY_EXECUTION_STATUS_COMPLETED))"
    )
    assertEquals(
      ProtoValue
        .enumValue(ActivityExecutionStatus.ACTIVITY_EXECUTION_STATUS_COMPLETED)
        .value
        .toString,
      "EnumName(ACTIVITY_EXECUTION_STATUS_COMPLETED)"
    )

  test("recorded repeated reads retain their method and complete field selection"):
    val selection =
      Field[io.temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryResponse, Seq[
        HistoryEvent
      ]](_.getHistory.events)
    val recorded = Recorded.read(METHOD_GET_WORKFLOW_EXECUTION_HISTORY, selection)
    recorded.recorded match
      case read: Recorded.TypedRead[?, ?, ?] =>
        assertEquals(
          read.requestMessage,
          "temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryRequest"
        )
        assertEquals(
          read.responseMessage,
          "temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryResponse"
        )
        assertEquals(read.projectedMessage, "temporal.api.history.v1.HistoryEvent")
        assert(read.method eq METHOD_GET_WORKFLOW_EXECUTION_HISTORY)
        assertEquals(read.path.path, Some("history.events"))
        assert(read.path eq selection)
      case other => fail(s"Recorded.read discarded arguments: $other")

  test("answered and delivered evidence retain facts, commands, attempts and confirmations"):
    val call = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {}
    val attempts =
      script("attempts", WorkerActivation.Activity(perCase("job"), caseWorker, taskQueue))()
    val action = framework.dsl.action("start", Actor("caller"))()
    val taking = Taking(action, 2)
    val answered = answeredAs("answer", ConstructionFact.completed, call, taking)
    assertEquals(answered.evidence.records, ConstructionFact.completed)
    assertEquals(answered.evidence.confirms, Vector(Taking(action, 2)))
    assert(answered.evidence.id.endsWith(".evidence.answer"))
    val simple = temporal.realize.answered(ConstructionFact.completed, call)
    assertEquals(simple.evidence.records, ConstructionFact.completed)
    val delivery = delivered(ConstructionFact.completed, attempts, 3, call, taking)
    delivery.evidence.from match
      case event: Recorded.TypedRunEvent[?] =>
        assertEquals(event.message, "temporal.server.api.testpilot.v1.InstructionOutcome")
        assertEquals(event.kind, EventKind.diagnostic)
        assertEquals(event.script, "controller")
        assert(event.command.equals(call))
        assertEquals(event.attempt.map(_.number), Some(3L))
        assert(event.attempt.get.script.equals(attempts))
        assertEquals(event.guard.map(_.operator), Some(ConditionOperator.all))
        assertEquals(
          event.guard.get.children.map(_.operator),
          Vector(ConditionOperator.present, ConditionOperator.not)
        )
      case other => fail(s"delivered discarded event metadata: $other")
    assertEquals(
      delivery.fields.map(f => (f.id, f.path.path, f.role)),
      Vector(
        ("attempt", Some("activity_attempt.sdk_attempt"), Some(FieldRole.attempt)),
        ("delivery", Some("activity_attempt.delivery_id"), Some(FieldRole.delivery)),
        ("activityRun", Some("activity_attempt.activity_run_id"), None)
      )
    )
    val field = Field[InstructionOutcome, Int](_.getActivityAttempt.sdkAttempt)
    assertEquals(attemptField(field).role, Some(FieldRole.attempt))
    assertEquals(
      deliveryField(Field(_.getActivityAttempt.deliveryId)).role,
      Some(FieldRole.delivery)
    )
    assertEquals(activityRunField(Field(_.getActivityAttempt.activityRunId)).id, "activityRun")
    assertEquals(
      Condition
        .present(Field[InstructionOutcome, Option[ActivityAttempt]](_.activityAttempt))
        .field
        .map(_.path),
      Some(Some("activity_attempt"))
    )

  test("kit literal helpers retain every supplied value"):
    assertEquals(duration(9).fields.map(_.value.value), Vector(ProtoValue.Number(9)))
    val payload = jsonPayload("hello")
    assertEquals(payload.fields.map(_.field.path), Vector(Some("metadata"), Some("data")))
    assertEquals(payload.fields(1).value.value, ProtoValue.Utf8("\"hello\""))
    val failure = applicationFailure("Retryable", "failed", retryable = false)
    assertEquals(failure.fields.head.value.value, ProtoValue.Text("failed"))
    failure.fields(1).value.value match
      case info: TypedProto[?] =>
        assertEquals(
          info.fields.map(_.value.value),
          Vector(ProtoValue.Text("Retryable"), ProtoValue.Flag(true))
        )
      case other => fail(s"applicationFailure discarded its metadata: $other")
    assertEquals(perCase("job"), Name("umpire-", fixture = true, suffix = "-job"))

  test("kit instruction helpers preserve typed references and worker options"):
    val call = rpc(workflowService, METHOD_START_ACTIVITY_EXECUTION) {}
    val learned = Learned("result", LearnedKind.handle)
    val publication = ActivityPublication("pending")
    val payloads = Proto[Payloads]()
    val failure = applicationFailure("failure", "message", retryable = true)
    val actuator = Actuator(
      "held",
      ControlKind.HoldDispatched(framework.dsl.action("start", Actor("caller"))()),
      taskQueue
    )
    assertEquals(finish("done"), Instruction.Finish(Operand.Literal(ProtoValue.Text("done"))))
    assertEquals(
      fault(caseWorker, FaultKind.workerStop),
      WorkerInstruction.Fault(caseWorker, FaultKind.workerStop)
    )
    assertEquals(hold(actuator), Instruction.Hold(actuator))
    assertEquals(release(actuator), Instruction.Release(actuator))
    assertEquals(awaitLearned(learned), Instruction.AwaitLearned(learned))
    assertEquals(awaitCommand(call), Instruction.AwaitCommand(call))
    assertEquals(attemptFailure(failure), WorkerInstruction.AttemptFailure(failure))
    assertEquals(attemptCanceled, WorkerInstruction.AttemptCanceled)
    assertEquals(attemptWithheld, WorkerInstruction.AttemptWithheld(WithholdingMode.context, None))
    assertEquals(
      attemptPending,
      WorkerInstruction.AttemptWithheld(WithholdingMode.sdkPending, None)
    )
    assertEquals(
      attemptPending(call),
      WorkerInstruction.AttemptWithheld(WithholdingMode.sdkPending, Some(call))
    )
    assertEquals(
      awaitActivityPublication(publication),
      WorkerInstruction.AwaitActivityPublication(publication)
    )
    assertEquals(attemptHeartbeat(payloads), WorkerInstruction.AttemptHeartbeat(payloads))
    assertEquals(workflowCommand(payloads), WorkerInstruction.WorkflowCommand(payloads))
    assertEquals(nexusReply(payloads, learned), WorkerInstruction.NexusReply(payloads, learned))
    assertEquals(
      nexusCompletion(learned, payloads),
      WorkerInstruction.NexusCompletion(learned, payloads)
    )

  test("behavior factories retain methods, causes, visibility and elapsed bounds"):
    val bound = WaitBound(Elapsed.milliseconds(250), Elapsed.milliseconds(5000))
    val visibility = METHOD_START_ACTIVITY_EXECUTION.visibleTo(
      METHOD_DESCRIBE_ACTIVITY_EXECUTION,
      Visible.eventually(bound)
    )
    assert(visibility.write.equals(METHOD_START_ACTIVITY_EXECUTION))
    assert(visibility.read eq METHOD_DESCRIBE_ACTIVITY_EXECUTION)
    assertEquals(
      visibility.when,
      Visible.eventually(WaitBound(Elapsed.milliseconds(250), Elapsed.milliseconds(5000)))
    )
    val cause = CauseKind.delivery.visibleTo(METHOD_DESCRIBE_ACTIVITY_EXECUTION, Visible.atOnce)
    assertEquals(cause.write, CauseKind.delivery)
    assert(cause.read eq METHOD_DESCRIBE_ACTIVITY_EXECUTION)
    assertEquals(cause.when, Visible.atOnce)
    val bounded = CauseKind.timer.boundedBy(bound)
    assertEquals(bounded.kind, CauseKind.timer)
    assertEquals(bounded.bound.interval, com.google.protobuf.duration.Duration(nanos = 250000000))
    assertEquals(bounded.bound.atMost, com.google.protobuf.duration.Duration(seconds = 5))

  test("realization and section helpers retain every authored runtime reference"):
    val instruction = finish("done")
    val item = everyCase(instruction)
    val script = controller(item)
    val operation = Entity("job", key = "id")
    val setting = RequiredSetting("feature", "true")
    val learned = Learned("value", LearnedKind.text)
    val observed = Observed[Payload]("payload")
    val action = framework.dsl.action("start", Actor("caller"))()
    val step = ServerStep(
      action,
      CauseKind.timer,
      Some(Elapsed.milliseconds(2000)),
      TimeoutBasis.startToClose
    )
    val actuator = Actuator("held", ControlKind.HoldDispatched(action), caseWorker)
    val behavior = ApiBehavior(
      Vector.empty,
      Vector.empty,
      Some(AttemptNumbering(2, true)),
      Some(InstructionLimit(Elapsed.milliseconds(5000), 3)),
      runOrderIsCausal = true
    )
    val realization = temporalRealization(
      ConstructionMachine,
      operation,
      Vector(caseWorker),
      Vector(script),
      Vector.empty,
      Vector(learned),
      Vector(observed),
      Vector(actuator),
      Vector(setting),
      Vector(step),
      behavior
    )
    assert(realization.machine eq ConstructionMachine)
    assertEquals(realization.roles, Vector(caseWorker))
    assertEquals(realization.scripts, Vector(script))
    assertEquals(realization.learned, Vector(learned))
    assertEquals(realization.observations, Vector(observed))
    assertEquals(realization.controls, Vector(actuator))
    assertEquals(realization.requiredSettings, Vector(setting))
    assertEquals(realization.serverSteps, Vector(step))
    assertEquals(realization.behavior, Some(behavior))
    assert(realization.correlation.operation.endsWith(".scope.job"))
    assertEquals(realization.correlation.observation, "correlated-evidence")
    assertEquals(script.id, "controller")
    assertEquals(script.items, Vector(item))
    val inserted = inserting(instruction)(item, item)
    val replaced = replacing(instruction)(item)
    assertEquals(
      (inserted.step, inserted.replaces, inserted.items),
      (instruction, false, Seq(item, item))
    )
    assertEquals((replaced.step, replaced.replaces, replaced.items), (instruction, true, Seq(item)))
    val base = new Realizes(
      ConstructionMachine,
      Vector(learned),
      Vector(observed),
      Vector(setting),
      Seq(script)
    ) {}
    val derived = new DerivesFrom(base, ConstructionMachine, Seq(item)) {}
    assert(derived.base eq base)
    assert(derived.machine eq ConstructionMachine)
    assertEquals(derived.runs, Seq(item))
    assertEquals(base.learned, Vector(learned))
    assertEquals(base.observations, Vector(observed))
    assertEquals(base.requiredSettings, Vector(setting))
    assertEquals(base.runs, Seq(script))
    assertEquals(new Controller(item) {}.items, Seq(item))
    assertEquals(new Workers(script) {}.scripts, Seq(script))
    assertEquals(new ServerSteps(step) {}.steps, Seq(step))
    assertEquals(new Controls(actuator) {}.actuators, Seq(actuator))
    assertEquals(new Changes(inserted, replaced) {}.changes, Seq(inserted, replaced))
    assert(everyValue(ConstructionFact.completed).equals(ConstructionFact.completed))
