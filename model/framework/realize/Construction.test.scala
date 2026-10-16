package framework.realize

import com.google.protobuf.duration.Duration
import io.grpc.MethodDescriptor
import scalapb.grpc.Marshaller

class RealizationConstructionTest extends munit.FunSuite:
  private val method = MethodDescriptor
    .newBuilder[Duration, Duration]()
    .setType(MethodDescriptor.MethodType.UNARY)
    .setFullMethodName("example.Clock/Read")
    .setRequestMarshaller(new Marshaller(Duration))
    .setResponseMarshaller(new Marshaller(Duration))
    .build()

  test("rpc retains scoped assignments and response reads in author order"):
    val call = rpc("clock", method) {
      field(_.seconds) := Operand.number(7)
      field(_.nanos) := Operand.integer(12)
      read(Field[Duration, Long](_.seconds), Cardinality.one)
        .into(Observed[Duration]("elapsed"), Target.Bind("seconds"))
    }
    assertEquals(
      call.assign.map(_.value.operand),
      Vector(
        Operand.Literal(ProtoValue.Number(7)),
        Operand.Literal(ProtoValue.Number(12))
      )
    )
    assertEquals(
      call.reads.map(_.targets),
      Vector(
        Vector(Target.Observe("elapsed"), Target.Bind("seconds"))
      )
    )
    val next = rpc("clock", method)(field(_.seconds) := Operand.number(9))
    assertEquals(
      next.assign.map(_.value.operand),
      Vector(Operand.Literal(ProtoValue.Number(9)))
    )
    assertEquals(next.reads.size, 0)

  test(
    "withFields and extended append scope data without changing the original"
  ):
    val call = rpc("clock", method)(field(_.seconds) := Operand.number(7))
    val variant = call.withFields(field(_.nanos) := Operand.integer(12))
    val own = call.extended {
      field(_.nanos) := Operand.integer(30)
      read(Field[Duration, Int](_.nanos), Cardinality.one)
        .into(Target.Bind("fraction"))
    }
    assertEquals(call.assign.size, 1)
    assertEquals(
      variant.assign.map(_.value.operand),
      Vector(
        Operand.Literal(ProtoValue.Number(7)),
        Operand.Literal(ProtoValue.Number(12))
      )
    )
    assertEquals(
      own.assign.map(_.value.operand),
      Vector(
        Operand.Literal(ProtoValue.Number(7)),
        Operand.Literal(ProtoValue.Number(30))
      )
    )
    assertEquals(
      own.reads.map(_.targets),
      Vector(Vector(Target.Bind("fraction")))
    )
    assertEquals(variant.sharedName, Some(call))
    assertEquals(own.sharedName, None)
    assert(!(variant eq call))
    assert(!(own eq call))

  test(
    "withFields appends response reads and extended owns its identity after a shared variant"
  ):
    val call = rpc("clock", method) {
      read(Field[Duration, Long](_.seconds), Cardinality.one)
        .into(Target.Bind("seconds"))
    }
    val variant = call.withFields {
      read(Field[Duration, Int](_.nanos), Cardinality.one)
        .into(Target.Bind("nanos"))
    }
    val own = variant.extended {}
    assertEquals(
      call.reads.map(_.targets),
      Vector(Vector(Target.Bind("seconds")))
    )
    assertEquals(
      variant.reads.map(_.targets),
      Vector(Vector(Target.Bind("seconds")), Vector(Target.Bind("nanos")))
    )
    assertEquals(
      own.reads.map(_.targets),
      Vector(Vector(Target.Bind("seconds")), Vector(Target.Bind("nanos")))
    )
    assertEquals(variant.sharedName, Some(call))
    assertEquals(own.sharedName, None)

  test("proto retains scalar fields and repeated construction has no leakage"):
    val value = proto[Duration] {
      field(_.seconds) := 5L
      field(_.nanos) := 40
    }
    assertEquals(
      value.fields.map(_.value.value),
      Vector(ProtoValue.Number(5), ProtoValue.Number(40))
    )
    val next = proto[Duration](field(_.seconds) := 2L)
    assertEquals(next.fields.map(_.value.value), Vector(ProtoValue.Number(2)))

    val duplicate = intercept[IllegalArgumentException](
      proto[Duration] {
        field(_.seconds) := 1L
        field(_.seconds) := 2L
      }
    )
    assert(duplicate.getMessage.contains("seconds"))
    assert(duplicate.getMessage.contains("twice"))

  test("readUntil retains its request assignments and original operands"):
    val evidence = Evidence.read(
      "elapsed",
      "recorded",
      "clock",
      Recorded.single(method, Field[Duration, Duration](identity)),
      Field[Duration, Long](_.seconds),
      Commitment.reported
    )
    val condition =
      Condition.equal(Field[Duration, Long](_.seconds), Operand.number(8))
    val poll =
      readUntil(evidence, "clock", condition, Some(Duration(seconds = 1))) {
        field(_.seconds) := Operand.number(6)
        read(Field[Duration, Long](_.seconds), Cardinality.one)
          .into(Target.Bind("elapsed"))
      }
    assertEquals(
      poll.assign.map(_.value.operand),
      Vector(Operand.Literal(ProtoValue.Number(6)))
    )
    assert(poll.evidence eq evidence)
    assert(poll.until eq condition)
    assertEquals(poll.interval, Some(Duration(seconds = 1)))
    assertEquals(
      poll.reads.map(_.targets),
      Vector(Vector(Target.Bind("elapsed")))
    )

  test("conditions retain distinct operators and ordered operands"):
    val field = Field[Duration, Long](_.seconds)
    val operand = Operand.number(4)
    val present = Condition.present(field)
    val equal = Condition.equal(field, operand)
    val greater = Condition.greater(field, operand)
    val not = Condition.not(equal)
    val all = Condition.all(present, greater, not)
    assertEquals(
      Vector(
        present.operator,
        equal.operator,
        greater.operator,
        not.operator,
        all.operator
      ),
      Vector(
        ConditionOperator.present,
        ConditionOperator.equal,
        ConditionOperator.greater,
        ConditionOperator.not,
        ConditionOperator.all
      )
    )
    assertEquals(all.children, Vector(present, greater, not))
    assertEquals(not.children, Vector(equal))
    assertEquals(equal.field, Some(field))
    assertEquals(greater.value, Some(operand))

  test(
    "Field retains descriptor paths and native selection without executing it"
  ):
    var selected = false
    val captured = Field[Duration, Long] { value =>
      selected = true; value.seconds
    }
    assert(!selected)
    assertEquals(captured.select(Duration(seconds = 7)), 7L)
    assert(selected)
    assertEquals(Field[Duration, Long](_.seconds).path, Some("seconds"))
    assertEquals(Field[Duration, Int](_.nanos).path, Some("nanos"))

  test("Proto retains its protobuf identity"):
    val value =
      Proto[Duration](ProtoField.typed(Field(_.seconds), ProtoValue.number(2)))
    assertEquals(value.message, "google.protobuf.Duration")
    assertEquals(
      proto[Duration](field(_.seconds) := 3L).message,
      "google.protobuf.Duration"
    )

  test(
    "ordinary generic protobuf helpers use the threaded companion for field paths"
  ):
    def build[M <: Duration](using
        scalapb.GeneratedMessageCompanion[M]
    ): TypedProto[M] =
      proto[M](field[M, Long](_.seconds) := 7L)
    val value = build[Duration]
    assertEquals(value.message, "google.protobuf.Duration")
    assertEquals(value.fields.head.field.path, Some("seconds"))
    assertEquals(
      value.fields.head.field.rootMessage,
      Some("google.protobuf.Duration")
    )
    assertEquals(value.fields.head.value.value, ProtoValue.Number(7))

  test(
    "protobuf factory type arguments retain root, request, response and empty element names"
  ):
    val field = Field[Duration, Long](_.seconds)
    assertEquals(field.rootMessage, Some("google.protobuf.Duration"))
    assertEquals(field.selectedMessage, None)
    val opaque: Duration => Long = _.seconds
    assertEquals(Field(opaque).rootMessage, Some("google.protobuf.Duration"))
    val call = rpc("clock", method) {}
    assertEquals(call.requestMessage, "google.protobuf.Duration")
    assertEquals(call.responseMessage, "google.protobuf.Duration")
    val recorded =
      Recorded.single(method, Field[Duration, Duration](value => value))
    recorded.recorded match
      case value: Recorded.TypedSingle[?, ?, ?] =>
        assertEquals(value.requestMessage, "google.protobuf.Duration")
        assertEquals(value.responseMessage, "google.protobuf.Duration")
        assertEquals(value.projectedMessage, "google.protobuf.Duration")
      case other => fail(s"Recorded.single discarded type arguments: $other")
    val event = Recorded
      .runEvent[Duration](
        EventKind.instructionCompleted,
        "script",
        "call",
        Operand.runKey()
      )
    event.recorded match
      case value: Recorded.TypedRunEvent[?] =>
        assertEquals(value.message, "google.protobuf.Duration")
      case other => fail(s"Recorded.runEvent discarded its root: $other")
    assertEquals(
      ProtoValue.messages[Duration]().message,
      Some("google.protobuf.Duration")
    )
    assertEquals(
      Operand.Projected.as[Duration].message,
      "google.protobuf.Duration"
    )

  test("aliasOf retains its target"):
    val instruction =
      Instruction.Finish(Operand.Literal(ProtoValue.Text("done")))
    val target =
      command(instruction, after = Some(After("prepare")), regardless = true)
    val alias = aliasOf(target)(Instruction.AwaitCommand(target))
    assertEquals(alias.alias, Some(target))

  test("onPath keeps action inputs safely"):
    val instruction =
      Instruction.Finish(Operand.Literal(ProtoValue.Text("done")))
    val action = framework.dsl.action("clock", framework.dsl.Actor("caller"))
    val item = onPath(action)(instruction)
    assertEquals(item.when, Vector(action))
    assertEquals(item.command.map(_.instruction), Some(instruction))

  test(
    "operand factories retain literal, environment, learned and projected inputs"
  ):
    assertEquals(
      Vector(
        Operand.text("text").operand,
        Operand.flag(true).operand,
        Operand.number(18).operand,
        Operand.integer(3).operand,
        Operand.named(Name("pre", true, "post")).operand,
        Operand.run().operand,
        Operand.environment[String]("binding").operand,
        Operand.learnedValue[String]("learned").operand
      ),
      Vector(
        Operand.Literal(ProtoValue.Text("text")),
        Operand.Literal(ProtoValue.Flag(true)),
        Operand.Literal(ProtoValue.Number(18)),
        Operand.Literal(ProtoValue.Number(3)),
        Operand.Literal(ProtoValue.Named(Name("pre", true, "post"))),
        Operand.Run,
        Operand.Environment("binding"),
        Operand.LearnedValue("learned")
      )
    )
    assertEquals(Operand.runKey().operand, Operand.Run)
    val path = Operand.path(
      Operand.Projected.as[Duration],
      Field[Duration, Long](_.seconds)
    )
    assertEquals(path.origin.operand, Operand.Projected)
    assertEquals(path.field.path, Some("seconds"))

  test("protobuf value factories retain each literal kind and ordered entries"):
    assertEquals(
      Vector(
        ProtoValue.text("text").value,
        ProtoValue.flag(false).value,
        ProtoValue.number(8).value,
        ProtoValue.integer(2).value,
        ProtoValue.utf8("bytes").value,
        ProtoValue.roleId("clock").value,
        ProtoValue.named(Name("n", false, "s")).value
      ),
      Vector(
        ProtoValue.Text("text"),
        ProtoValue.Flag(false),
        ProtoValue.Number(8),
        ProtoValue.Number(2),
        ProtoValue.Utf8("bytes"),
        ProtoValue.RoleId("clock"),
        ProtoValue.Named(Name("n", false, "s"))
      )
    )
    val first = ProtoEntry.typed("one", ProtoValue.utf8("1"))
    val second = ProtoEntry.typed("two", ProtoValue.utf8("2"))
    assertEquals(ProtoValue.mapping(first, second).value, Vector(first, second))
    assertEquals(first.key, "one")
    assertEquals(first.value.value, ProtoValue.Utf8("1"))
    val proto = Proto[Duration]()
    assert(ProtoValue.message(proto).value.equals(proto))
    assertEquals(ProtoValue.messages(proto, proto).value, Vector(proto, proto))

  test(
    "core assignment, response read and instruction factories retain typed arguments"
  ):
    val path = Field[Duration, Long](_.seconds)
    val value = Operand.number(8)
    val assignment = Assignment.typed(path, value)
    assert(!assignment.expandsMessageFields)
    val read = ResponseRead.typed(
      path,
      Cardinality.each,
      Vector(Target.Observe("seen"), Target.Lift("lifted"))
    )
    assert(assignment.target eq path)
    assert(assignment.value eq value)
    assert(read.path eq path)
    assertEquals(read.cardinality, Cardinality.each)
    assertEquals(
      read.targets,
      Vector(Target.Observe("seen"), Target.Lift("lifted"))
    )
    Instruction.rpc("clock", method)(Vector(assignment), Vector(read)) match
      case call: Instruction.TypedRpc[?, ?] =>
        assertEquals(call.role, "clock")
        assert(call.method eq method)
        assertEquals(
          call.assign.map(_.value.operand),
          Vector(Operand.Literal(ProtoValue.Number(8)))
        )
        assert(call.assign.head.equals(assignment))
        assertEquals(call.reads, Vector(read))
      case other => fail(s"Instruction.rpc discarded its declaration: $other")

  test("request message sugar retains its leaf-expansion intent"):
    val scope = RequestScope[Duration]()
    val message = proto[Duration](field(_.seconds) := 8L)
    RequestField(Field[Duration, Duration](identity), scope) := message

    assert(scope.assignments.head.expandsMessageFields)
    assertEquals(scope.assignments.head.value.operand, Operand.Message(message))

  test(
    "recorded and evidence factories retain events, attempts, keys, guards and fields"
  ):
    val path = Field[Duration, Long](_.seconds)
    val selected = Recorded.single(method, Field[Duration, Duration](identity))
    selected.recorded match
      case single: Recorded.TypedSingle[?, ?, ?] =>
        assert(single.method eq method)
      case other => fail(s"Recorded.single discarded method: $other")
    val condition = Condition.greater(path, Operand.number(1))
    val attempt = AttemptOf("attempts", 2)
    val key = Operand.path(Operand.Projected.as[Duration], path)
    val event = Recorded.runEvent[Duration](
      EventKind.diagnostic,
      "script",
      "command",
      key,
      Some(condition),
      Some(attempt)
    )
    event.recorded match
      case recorded: Recorded.TypedRunEvent[?] =>
        assertEquals(recorded.kind, EventKind.diagnostic)
        assertEquals(recorded.script, "script")
        assertEquals(recorded.command, "command")
        assert(recorded.key.equals(key))
        assert(recorded.guard.get eq condition)
        assertEquals(recorded.attempt, Some(AttemptOf("attempts", 2)))
      case other => fail(s"Recorded.runEvent discarded arguments: $other")
    val field = EvidenceField.typed(
      "number",
      path,
      Some(FieldRole.delivery),
      redacted = true
    )
    val step = framework.dsl.action("clock", framework.dsl.Actor("caller"))()
    val taking = Taking(step, 3)
    val evidence = Evidence.runEvent(
      "kind",
      "fact",
      "source",
      event,
      Commitment.durable,
      Vector(field),
      exhaustive = true,
      Vector(taking)
    )
    assertEquals(evidence.operation, None)
    assertEquals(evidence.fields, Vector(field))
    assertEquals(evidence.evidence.records, "fact")
    assert(evidence.evidence.from eq event.recorded)
    assertEquals(evidence.evidence.confirms, Vector(Taking(step, 3)))
    assertEquals(evidence.evidence.commitment, Commitment.durable)
    assert(evidence.evidence.exhaustive)
    val keyed = Evidence.keyed(
      "keyed",
      "fact",
      "source",
      new KeyedRef[Duration](selected.recorded),
      path,
      Commitment.reported,
      Vector(field),
      exhaustive = true,
      Vector(taking)
    )
    assertEquals(keyed.operation, Some(path))
    assertEquals(keyed.fields, Vector(field))
    assert(keyed.evidence.from eq selected.recorded)

  test("command and script factories retain options, order and class bindings"):
    val instruction = Instruction.AwaitLearned("value")
    val expected = command(
      instruction,
      Some(After("first", "second")),
      Some(Duration(seconds = 5)),
      regardless = true,
      Vector("kind")
    )
    assertEquals(expected.after.map(_.commands), Some(Seq("first", "second")))
    assertEquals(expected.timeout, Some(Duration(seconds = 5)))
    assert(expected.regardless)
    assertEquals(expected.closes, Vector("kind"))
    val step = framework.dsl.action("clock", framework.dsl.Actor("caller"))()
    val performance = perform(step -> expected, step -> instruction)
    assertEquals(performance.performs.map(_.step), Vector(step, step))
    assertEquals(
      performance.performs.map(_.command.instruction),
      Vector(instruction, instruction)
    )
    assert(performance.performs.head.command eq expected)
    val every = everyCase(expected)
    assertEquals(every.command, Some(expected))
    val declared = script("script", Activation.Controller)(performance, every)
    assertEquals(declared.id, "script")
    assertEquals(declared.activation, Activation.Controller)
    assertEquals(declared.items, Vector(performance, every))
    val selected = onPath(step)(instruction)
    assertEquals(selected.when, Vector(step))

  test(
    "status and exploration factories retain declared values and sequence order"
  ):
    val table = statusTable("first" -> 4, "second" -> 2)
    assertEquals(table.entries, Vector("first" -> 4, "second" -> 2))
    assertEquals(table("second"), 2)
    val step = framework.dsl.action("clock", framework.dsl.Actor("caller"))()
    val first = Alternative("first", 3, Vector(step))
    val second = Alternative("second", 1, Vector.empty)
    val variation = Variation(2, Vector(first, second))
    val exploration =
      Exploration("explore", Vector(variation), 5, 7, dropPrefix = true)
    assertEquals(exploration.name, "explore")
    assertEquals(exploration.variations.head.index, 2)
    assertEquals(
      exploration.variations.head.choices.map(a => (a.name, a.priority)),
      Vector("first" -> 3, "second" -> 1)
    )
    assertEquals(first.actions, Vector(step))
    assertEquals(
      (exploration.runs, exploration.edits, exploration.dropPrefix),
      (5, 7, true)
    )
    assertEquals(
      Elapsed.milliseconds(1234),
      Duration(seconds = 1, nanos = 234000000)
    )
