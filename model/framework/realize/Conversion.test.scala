package framework.realize

import com.google.protobuf.duration.Duration
import io.temporal.server.api.umpire.v1 as ir

class RealizationConversionTest extends munit.FunSuite:
  private enum OwnedFact:
    case recorded
  private enum ForeignFact:
    case recorded

  test("fact ownership comes from the actual native machine catalog, not a case spelling"):
    val metadata = ir.Model(
      machines = Vector(ir.Machine(name = "machine", factType = "fixture.Fact")),
      types = Vector(
        ir.Type(
          name = "fixture.Fact",
          shape = ir.Type.Shape.Enum(ir.Enum(cases = Vector(ir.Case(name = "recorded"))))
        )
      )
    )
    val ctx =
      new Construction("fixture.realization", metadata, "machine", Some(Vector(OwnedFact.recorded)))
    assertEquals(ctx.fact(OwnedFact.recorded), "recorded")
    val failure = intercept[IllegalArgumentException](ctx.fact(ForeignFact.recorded))
    assert(failure.getMessage.contains("fixture.realization"))
    assert(failure.getMessage.contains("native machine fact catalog"))

  for (name, build) <- Vector[(String, () => Item)](
      "perform" -> (() => perform()),
      "onPath" -> (() => onPath()(Instruction.Hold("gate")))
    )
  do
    test(s"empty $name preserves the authored refusal"):
      val failure = intercept[IllegalArgumentException](build())
      assert(failure.getMessage.contains(name))
  test("action identity metadata is source-owned and preserves native declaration pointers"):
    val original = framework.dsl.action("step", framework.dsl.Actor("caller"))
    val declaration = original.decl
    original.declaredAt("fixture.caller.step"): Unit
    assert(original.decl eq declaration)
    assertEquals(declaration.productArity, 13)
    assertEquals(declaration.sourceIdentity, Some("fixture.caller.step"))
    val alias = original
    assert(alias.decl eq declaration)
    assertEquals(
      alias.on(framework.dsl.Entity("entity")).decl.sourceIdentity,
      Some("fixture.caller.step")
    )
    intercept[IllegalArgumentException](original.declaredAt("fixture.other.step"))

  test("action class conversion refuses lost entity binding metadata rather than dropping it"):
    val original = framework.dsl.action("step", framework.dsl.Actor("caller"))
    original.declaredAt("fixture.step"): Unit
    val bound = original.on(framework.dsl.Entity("entity"))
    val missing = new Construction(
      "fixture.realization",
      ir.Model(actions = Vector(ir.Action(id = "fixture.step"))),
      "machine"
    )
    val failure = intercept[IllegalArgumentException](missing.actionClass(bound))
    assert(failure.getMessage.contains("entity binding metadata"))
    val retained = new Construction(
      "fixture.realization",
      ir.Model(actions = Vector(ir.Action(id = "fixture.step", on = "entity"))),
      "machine"
    )
    assertEquals(retained.actionClass(bound).action, "fixture.step")

  test("channel source identity names its implicit actions without rebuilding native values"):
    val channel = framework.dsl.channel[Boolean](
      capacity = 2,
      order = framework.dsl.Order.fifo,
      loss = framework.dsl.Loss.reliable
    )
    val delivery = channel.deliver.decl
    val loss = channel.lose.decl
    val alias = channel
    channel.declaredAt("fixture.inbox"): Unit
    assert(alias eq channel)
    assert(channel.deliver.decl eq delivery)
    assert(channel.lose.decl eq loss)
    assertEquals(channel.sourceIdentity, Some("fixture.inbox"))
    assertEquals(delivery.sourceIdentity, Some("fixture.inbox.deliver"))
    assertEquals(loss.sourceIdentity, Some("fixture.inbox.lose"))
    val result = Actuator("gate", ControlKind.HoldDelivery(channel), "worker")
      .toIR(new Construction("fixture.realization", ir.Model(), "machine"))
    assertEquals(result.getHoldDelivery, "fixture.inbox")
    assertEquals(result.role, "worker")

  private object Machine extends framework.dsl.Machine[Boolean, Boolean, Boolean]:
    val init = false
    def end(s: Boolean): Boolean = s

    object rules extends framework.dsl.Rules

  test("a constructed realization preserves ordered commands and its header"):
    val first = Instruction.Hold("gate").named("pauseOrder")
    val second = Instruction.Release("gate").named("releaseOrder")
    val value = Realization(
      machine = Machine,
      producer = "example",
      producerVersion = "1",
      roles = Vector.empty,
      correlation = Correlation("p", "r", "o", "v", 8, 9, 10, 11, 12, 13),
      scripts =
        Vector(script("controller", Activation.Controller)(everyCase(first), everyCase(second)))
    )
    val result = value.toIR("sample.realization")
    assertEquals(result.id, "sample.realization")
    assertEquals(result.name, "realization")
    assertEquals(result.machine, "machine")
    assertEquals(result.producer, "example")
    assertEquals(result.producerVersion, "1")
    assertEquals(
      result.scripts.head.items.map(_.getCommand.id),
      Vector("pause-order", "release-order")
    )
    assertEquals(result.scripts.head.getController, com.google.protobuf.empty.Empty())
    assertEquals(result.getCorrelation.events, 8L)
    assertEquals(result.getCorrelation.eventSize, 13L)

  test("native named copies keep product, equality and unapply"):
    val original = Instruction.Hold("gate")
    val named = original.named("named")
    assertEquals(named.productArity, 1)
    assertEquals(named.productElement(0), "gate")
    assert(named.equals(original))
    val Instruction.Hold(control) = named
    assertEquals(control, "gate")
    assertEquals(named.copy().ref, None)

  test("command options copies retain the reference and rpc shared variants keep their base"):
    val original = command(Instruction.Hold("gate")).named("original")
    assertEquals(original.copy(regardless = true).ref, original.ref)
    val method = io.grpc.MethodDescriptor
      .newBuilder[Duration, Duration]()
      .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
      .setFullMethodName("example.Clock/Read")
      .setRequestMarshaller(new scalapb.grpc.Marshaller(Duration))
      .setResponseMarshaller(new scalapb.grpc.Marshaller(Duration))
      .build()
    val call = rpc("clock", method) {}.named("baseCall")
    val variant = call.withFields(field(_.seconds) := Operand.number(2))
    assertEquals(variant.commandId("fixture"), "base-call")
    assertEquals(variant.sharedName.flatMap(_.ref), call.ref)
    assertEquals(variant.assign.map(_.value.operand), Vector(Operand.Literal(ProtoValue.Number(2))))

  test("reference names are independent copies retaining the concrete instruction and options"):
    val original = Instruction.Hold("gate")
    val alias = original
    val named = original.named("pauseOrder")
    val renamed = alias.named("alias")
    assert(original eq alias)
    assertEquals(original.ref, None)
    assertEquals(named.ref.map(_.spelling), Some("pauseOrder"))
    assertEquals(renamed.ref.map(_.spelling), Some("alias"))
    assertEquals(named.control, "gate")
    assertEquals(everyCase(named).command.get.toIR("fixture.realization").id, "pause-order")
    assertEquals(everyCase(renamed).command.get.toIR("fixture.realization").id, "alias")

  test("typed protobuf conversion keeps fields and never invokes selectors"):
    var selected = 0
    val field = new Field[Duration, Long](
      _ =>
        selected += 1; 0L
      ,
      Some("seconds"),
      Some("google.protobuf.Duration"),
      None
    )
    val value = Proto[Duration](ProtoField.typed(field, ProtoValue.number(7)))
    val result = value.toIR("sample.duration")
    assertEquals(result.message, "google.protobuf.Duration")
    assertEquals(result.fields.map(_.name), Vector("seconds"))
    assertEquals(result.fields.map(_.getValue), Vector(ir.ProtoValue(ir.ProtoValue.Kind.Number(7))))
    assertEquals(selected, 0)

  test("opaque protobuf selectors refuse with their declaration"):
    val selector: Duration => Long = _.seconds
    val value = Proto[Duration](ProtoField.typed(Field(selector), ProtoValue.number(7)))
    val refused = intercept[IllegalArgumentException](value.toIR("sample.opaque"))
    assert(refused.getMessage.contains("sample.opaque"))
    assert(refused.getMessage.contains("selector"))

  test("generic instruction copies retain native shape and operands"):
    val values = Vector[Instruction](
      Instruction.Hold("gate"),
      Instruction.Release("gate"),
      Instruction.AwaitLearned("value"),
      Instruction.AwaitCommand("call"),
      Instruction.Finish(Operand.Literal(ProtoValue.Text("done")))
    )
    values.foreach: value =>
      val named = value.named("ref", "fixture.ref", "Model.scala:9")
      assert(named.equals(value))
      assertEquals(
        named.asInstanceOf[Product].productIterator.toList,
        value.asInstanceOf[Product].productIterator.toList
      )
      assertEquals(named.ref, Some(Reference("ref", "fixture.ref", "Model.scala:9")))
      assertEquals(value.ref, None)

  test("run-event evidence constructs keys, guards, fields and attempts"):
    val selected = Field[Duration, Long](_.seconds)
    val record = Recorded.runEvent[Duration](
      EventKind.diagnostic,
      "attempts",
      "finish",
      Operand.path(Operand.Projected.as[Duration], selected),
      Some(
        Condition.all(
          Condition.present(selected),
          Condition.not(Condition.equal(selected, Operand.number(3)))
        )
      ),
      Some(AttemptOf("attempts", 2))
    )
    val evidence = Evidence.runEvent(
      "record",
      "completed",
      "source",
      record,
      Commitment.durable,
      Vector(EvidenceField.typed("attempt", selected, Some(FieldRole.attempt), redacted = true)),
      exhaustive = true
    )
    val result = evidence.toIR(new Construction("fixture.run", ir.Model(), "machine"))
    assertEquals(result.id, "record")
    assertEquals(result.records, "completed")
    assertEquals(result.source, "source")
    assertEquals(result.getRunEvent.kind, ir.RunEventSource.Kind.KIND_DIAGNOSTIC)
    assertEquals(result.getRunEvent.script, "attempts")
    assertEquals(result.getRunEvent.command, "finish")
    assertEquals(result.getRunEvent.getKey.getPath.path, "seconds")
    assertEquals(result.getRunEvent.getGuard.getAll.operands.size, 2)
    assertEquals(
      result.getRunEvent.getGuard.getAll
        .operands(1)
        .getNot
        .getOf
        .getEqual
        .getRight
        .getLiteral
        .getNumber,
      3L
    )
    assertEquals(result.getRunEvent.getAttempt.number, 2L)
    assertEquals(result.fields.head.path, "seconds")
    assertEquals(result.fields.head.role, ir.EvidenceField.Role.ROLE_ATTEMPT)
    assert(result.fields.head.redacted)
    assert(result.exhaustive)

  test("scoped poll reads and an unconverted kit instruction refuse explicitly"):
    val method = io.grpc.MethodDescriptor
      .newBuilder[Duration, Duration]()
      .setType(io.grpc.MethodDescriptor.MethodType.UNARY)
      .setFullMethodName("example.Clock/Read")
      .setRequestMarshaller(new scalapb.grpc.Marshaller(Duration))
      .setResponseMarshaller(new scalapb.grpc.Marshaller(Duration))
      .build()
    val selected = Field[Duration, Long](_.seconds)
    val evidence = Evidence.read(
      "clock",
      "recorded",
      "source",
      Recorded.single(method, Field[Duration, Duration](identity)),
      selected,
      Commitment.reported
    )
    val poll = readUntil(evidence, "clock", Condition.present(selected), None) {
      read(selected, Cardinality.one).into(Target.Bind("value"))
    }
    val refused = intercept[IllegalArgumentException](poll.toIR("fixture.poll"))
    assert(refused.getMessage.contains("fixture.poll: scoped poll response reads"))
    object Opaque extends Instruction
    val unnamed =
      intercept[IllegalArgumentException](Opaque.named("opaque", "fixture.opaque", "Model.scala:9"))
    assert(unnamed.getMessage.contains("Model.scala:9: fixture.opaque"))
    val opaque = intercept[IllegalArgumentException](Opaque.toIR("fixture.opaque"))
    assert(opaque.getMessage.contains("fixture.opaque"))
