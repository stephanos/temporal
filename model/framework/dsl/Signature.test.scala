package framework.dsl

import java.util.Base64
import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir

class SignatureTest extends munit.FunSuite:
  private def bytes(value: Array[Byte]): Seq[String] =
    Seq(Base64.getEncoder.encodeToString(value))
  private val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(Empty()))
  private def retain[A](value: ScalarAction[A]): ScalarAction[A] =
    value.retainInputSchema(
      0,
      "scala.Boolean",
      bytes(bool.toByteArray),
      bytes(ir.Model().toByteArray),
      Nil
    ): Unit
    value

  test("signature shape uses retained identity without evaluating values and survives every copy"):
    given Finite[Boolean] = new Finite[Boolean]:
      def values = throw new AssertionError("signature enumerated values")
    val original = retain(action("explicit", Actor("caller")).input[Boolean](name = "flag"))
      .declaredAt("fixture.caller.original")
    original.decl.boundAt("fixture.bound")
    val record = Entity("record")
    val copies = List(original.results("Answer"), original.on(record), original.creates(record))
    for (copy, index) <- copies.zipWithIndex do
      assertEquals(copy.toIR.id, "fixture.caller.original")
      assertEquals(copy.toIR.name, "explicit")
      assertEquals(copy.toIR.inputs, Seq(ir.Param("flag", Some(bool))))
      assertEquals(copy.decl.sourceBinding, if index == 0 then Some("fixture.bound") else None)
    assertEquals(Signature.collect(List(original), Nil).types, Seq.empty)

  test(
    "a new entity binding gets its own source while signature-only copies retain the prior binding"
  ):
    given Finite[Boolean] = Finite.of(false, true)
    val original = retain(action("update", Actor("caller")).input[Boolean](name = "flag"))
      .sourceAt("fixture.original")
    val first = original.on(Entity("first")).sourceAt("fixture.first")
    val second = first.on(Entity("second")).results("Answer").sourceAt("fixture.second")
    assertEquals(first.decl.sourceBinding, Some("fixture.first"))
    assertEquals(second.decl.sourceBinding, Some("fixture.second"))
    assertEquals(second.decl.sourceIdentity, Some("fixture.original"))
    assertEquals(first.results("Answer").decl.sourceBinding, Some("fixture.first"))

  test("missing metadata and unresolved Inbox references refuse with action and input context"):
    val missing = action("send", Actor("caller"))
      .input[Boolean](name = "message")(using Finite.of(false, true))
      .declaredAt("fixture.send")
    val error = intercept[IllegalArgumentException](missing.toIR)
    assert(error.getMessage.contains("fixture.send"), error.getMessage)
    assert(error.getMessage.contains("message"), error.getMessage)
    val inbox = ir.TypeRef(ir.TypeRef.Ref.Channel("scala.Boolean"))
    missing.retainInputSchema(
      0,
      "inbox",
      bytes(inbox.toByteArray),
      bytes(ir.Model().toByteArray),
      bytes(ir.Record(fields = Seq(ir.Field("scala.Boolean", Some(bool)))).toByteArray)
    ): Unit
    assert(
      intercept[IllegalArgumentException](missing.toIR).getMessage.contains("unresolved Inbox")
    )

  test("required and false defaults remain separate metadata"):
    given Finite[Boolean] = Finite.of(false, true)
    val send =
      retain(action("send", Actor("caller")).input[Boolean](name = "flag", default = false))
    send.declaredAt("fixture.send"): Unit
    assertEquals(
      send.inputMetadata.map(_.default),
      Vector(Some(ir.Value(ir.Value.Kind.Bool(false))))
    )
    assertEquals(send().values, List(false))
    val requiredAction =
      retain(action("required", Actor("caller")).input[Boolean](name = "required"))
        .declaredAt("fixture.required")
    assertEquals(requiredAction.inputMetadata.map(_.default), Vector(None))

  test("missing and conflicting type definitions identify the declaration and input"):
    val named = ir.TypeRef(ir.TypeRef.Ref.Named("fixture.Mode"))
    val first = ir.Type(
      name = "fixture.Mode",
      shape = ir.Type.Shape.Enum(ir.Enum(cases = Seq(ir.Case("first"))))
    )
    val second = first.copy(shape = ir.Type.Shape.Enum(ir.Enum(cases = Seq(ir.Case("second")))))
    for definitions <- Seq(Seq.empty, Seq(first, second)) do
      val send = action("send", Actor("caller"))
        .input[Boolean](name = "mode")(using Finite.of(false, true))
        .declaredAt("fixture.send")
      send.retainInputSchema(
        0,
        "fixture.Mode",
        bytes(named.toByteArray),
        bytes(ir.Model(types = definitions).toByteArray),
        Nil
      ): Unit
      val refused = intercept[IllegalArgumentException](send.toIR)
      assert(refused.getMessage.contains("fixture.send input mode"), refused.getMessage)
      assert(refused.getMessage.contains("fixture.Mode"), refused.getMessage)
    val send = retain(
      action("send", Actor("caller")).input[Boolean](name = "flag")(using Finite.of(false, true))
    ).declaredAt("fixture.send")
    val conflict = intercept[IllegalArgumentException](
      send.retainInputSchema(
        0,
        "scala.Boolean",
        bytes(named.toByteArray),
        bytes(ir.Model(types = Seq(first)).toByteArray),
        Nil
      )
    )
    assert(conflict.getMessage.contains("fixture.send input flag"), conflict.getMessage)

  test("channel identity, explicit names and conflicting definitions are checked without contents"):
    given Finite[Boolean] = Finite.of(false, true)
    def wire(capacity: Int): Channel[Boolean] =
      val value = new Channel[Boolean]("explicit", capacity, Order.fifo, Loss.lossy, 1)
        .declaredAt("fixture.wire")
        .retainName("fixture.wire", "wire")
      value.retainInputSchema(
        "scala.Boolean",
        bytes(bool.toByteArray),
        bytes(ir.Model().toByteArray),
        Nil
      )
    val first = wire(1)
    assertEquals(first.toIR.name, "explicit")
    assertEquals(first.deliver.toIR.name, "explicitDelivery")
    assertEquals(first.lose.toIR.name, "explicitLoss")
    assert(first.sourceAt("fixture.alias") eq first)
    assertEquals(first.toIR.id, "fixture.wire")
    intercept[IllegalArgumentException](first.declaredAt("fixture.invalid")): Unit
    assertEquals(
      Signature.collect(List(first.deliver, first.deliver), List(first, first)).actions.size,
      1
    )
    assert(
      intercept[IllegalArgumentException](Signature.collect(Nil, List(first, wire(2)))).getMessage
        .contains("fixture.wire")
    )

  test("separately requested channel actions refuse unbounded messages with their channel context"):
    given Finite[Int] = Finite.upTo(2)
    val wire = new Channel[Int]("", 1, Order.fifo, Loss.lossy, 0)
      .declaredAt("fixture.wire")
      .retainName("fixture.wire", "wire")
    val integer = ir.TypeRef(ir.TypeRef.Ref.Int(Empty()))
    wire.retainInputSchema(
      "scala.Int",
      bytes(integer.toByteArray),
      bytes(ir.Model().toByteArray),
      Nil
    ): Unit
    for convert <- Seq[() => Any](() => wire.toIR, () => wire.deliver.toIR, () => wire.lose.toIR) do
      val refused = intercept[IllegalArgumentException](convert(): Unit)
      assert(refused.getMessage.contains("fixture.wire"), refused.getMessage)
      assert(refused.getMessage.contains("Int with no declared bound"), refused.getMessage)
