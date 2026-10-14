package framework.dsl

import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

class InboxOwnershipSuite extends munit.FunSuite:
  private def bytes(value: Array[Byte]): Seq[String] =
    Seq(Base64.getEncoder.encodeToString(value))
  private val message = ir.TypeRef(ir.TypeRef.Ref.IntRange(ir.IntRange(0, 2)))
  private val placeholder = ir.TypeRef(ir.TypeRef.Ref.Channel("fixture.Message"))
  private val field = ir.Field("box", Some(placeholder))
  private val record = ir.Type("fixture.State", shape = ir.Type.Shape.Record(ir.Record(Seq(field))))
  private def witness(
      shape: ir.Type = record,
      closure: Seq[ir.Type] = Nil,
      pending: ir.TypeRef = message
  ): Finite[Boolean] =
    val domain = new Finite[Boolean]:
      def values = throw new AssertionError("state catalog evaluated")
    domain.retainSchema(
      "fixture.State",
      bytes(ir.TypeRef(ir.TypeRef.Ref.Named("fixture.State")).toByteArray),
      bytes(ir.Model(types = shape +: closure).toByteArray),
      bytes(ir.Record(Seq(ir.Field("fixture.Message", Some(pending)))).toByteArray)
    )
  private def wire(
      key: String = "fixture.Message",
      id: String = "fixture.wire",
      ref: ir.TypeRef = message,
      closure: Seq[ir.Type] = Nil
  ): Channel[Boolean] =
    given Finite[Boolean] = new Finite[Boolean]:
      def values = throw new AssertionError("message catalog evaluated")
    val value = channel[Boolean](1, Order.fifo, Loss.reliable).declaredAt(id).retainName(id, "wire")
    value.retainInputSchema(
      key,
      bytes(ref.toByteArray),
      bytes(ir.Model(types = closure).toByteArray),
      Nil
    )
  private def bind(
      domain: Finite[Boolean],
      value: () => Channel[?],
      owner: String = "fixture.State",
      caseName: String = "",
      name: String = "box",
      key: String = "fixture.Message",
      id: String = "fixture.wire"
  ): Finite[Boolean] =
    domain.retainInbox("fixture.State", owner, caseName, name, key, id, value)

  test("exact field ownership preserves witness schema and memoizes the original receiver"):
    val domain = witness()
    val structural = domain.schema.get
    val value = wire()
    var calls = 0
    val same = bind(
      domain,
      () =>
        calls += 1; value
    )
    assert(same eq domain)
    bind(domain, () => throw new AssertionError("idempotent supplier replaced")): Unit
    assertEquals(calls, 0)
    for _ <- 0 until 2 do
      val result = Signature.stateSchema(domain, Some("fixture.State"), "state fixture")
      assertEquals(result.types.head.getRecord.fields.head.getType.getChannel, "fixture.wire")
      assertEquals(result.unresolvedInboxes, Map.empty[String, ir.TypeRef])
      assertEquals(domain.schema.get, structural)
    assertEquals(calls, 1)

  test("missing or extra typed field evidence refuses before returning a schema"):
    for (label, mutate, expected) <- Seq[(String, Finite[Boolean] => Unit, String)](
        ("missing", _ => (), "box"),
        ("owner", d => bind(d, () => wire(), owner = "fixture.Other"): Unit, "fixture.Other"),
        ("case", d => bind(d, () => wire(), caseName = "wrong"): Unit, "wrong"),
        ("field", d => bind(d, () => wire(), name = "wrong"): Unit, "wrong"),
        ("message", d => bind(d, () => wire(), key = "fixture.Other"): Unit, "message"),
        ("channel-key", d => bind(d, () => wire(key = "fixture.Other")): Unit, "message"),
        (
          "message-schema",
          d =>
            bind(d, () => wire(ref = ir.TypeRef(ir.TypeRef.Ref.IntRange(ir.IntRange(0, 3))))): Unit,
          "schema"
        ),
        ("channel-id", d => bind(d, () => wire(id = "fixture.other")): Unit, "identity"),
        ("null", d => bind(d, () => null): Unit, "initialized"),
        (
          "throwing",
          d => bind(d, () => throw IllegalStateException("premature")): Unit,
          "premature"
        )
      )
    do
      val domain = witness()
      mutate(domain)
      val error = intercept[IllegalArgumentException](
        Signature.stateSchema(domain, Some("fixture.State"), s"state $label")
      )
      assert(error.getMessage.contains(s"state $label"), error.getMessage)
      assert(error.getMessage.contains(expected), error.getMessage)
    val absent = new Finite[Boolean]:
      def values = throw new AssertionError("missing schema read values")
    for key <- Seq(None, Some("missing")) do
      assert(
        intercept[IllegalArgumentException](
          Signature.stateSchema(absent, key, "state missing")
        ).getMessage.contains("typed schema")
      )

  test("conflicting ownership and channel metadata refuse with field context"):
    val domain = bind(witness(), () => wire())
    val conflict =
      intercept[IllegalArgumentException](bind(domain, () => wire(), id = "fixture.other"))
    assert(conflict.getMessage.contains("fixture.State"))
    assert(conflict.getMessage.contains("box"))
    val unidentified = new Channel[Boolean]("", 1, Order.fifo, Loss.reliable, 0)(using Finite.of())
    val unschemed = new Channel[Boolean]("", 1, Order.fifo, Loss.reliable, 0)(using Finite.of())
      .declaredAt("fixture.wire")
    for (value, expected) <- Seq(unidentified -> "source identity", unschemed -> "message key") do
      val error = intercept[IllegalArgumentException](
        Signature.stateSchema(
          bind(witness(), () => value),
          Some("fixture.State"),
          "state missing channel"
        )
      )
      assert(error.getMessage.contains("box"), error.getMessage)
      assert(error.getMessage.contains(expected), error.getMessage)

  test("duplicate storage across record fields and enum cases remains refused"):
    val variants = Seq(
      record.copy(shape =
        ir.Type.Shape.Record(ir.Record(Seq(field, field.copy(name = "other"))))
      ) ->
        Seq("" -> "box", "" -> "other"),
      record.copy(shape =
        ir.Type.Shape.Enum(ir.Enum(Seq(ir.Case("one", Seq(field)), ir.Case("two", Seq(field)))))
      ) -> Seq("one" -> "box", "two" -> "box")
    )
    for (shape, coordinates) <- variants do
      val domain = witness(shape)
      val value = wire()
      coordinates.foreach((c, f) => bind(domain, () => value, caseName = c, name = f): Unit)
      val error = intercept[IllegalArgumentException](
        Signature.stateSchema(domain, Some("fixture.State"), "state duplicate")
      )
      assert(error.getMessage.contains("holds a channel in one field"), error.getMessage)

  test(
    "nested closure requires its own exact witness selection and rejects incomplete definitions"
  ):
    val nested = ir.Type(
      "fixture.State",
      shape = ir.Type.Shape.Record(
        ir.Record(Seq(ir.Field("inner", Some(ir.TypeRef(ir.TypeRef.Ref.Named("fixture.Inner"))))))
      )
    )
    val inner = record.copy(name = "fixture.Inner")
    val root = witness(nested, Seq(inner))
    val domain = new Finite[Boolean]:
      def values = throw new AssertionError("nested catalog evaluated")
    domain.retainSchema(
      "fixture.Inner",
      bytes(ir.TypeRef(ir.TypeRef.Ref.Named("fixture.Inner")).toByteArray),
      bytes(ir.Model(types = Seq(inner)).toByteArray),
      bytes(ir.Record(Seq(ir.Field("fixture.Message", Some(message)))).toByteArray)
    ): Unit
    domain.retainInbox(
      "fixture.Inner",
      "fixture.Inner",
      "",
      "box",
      "fixture.Message",
      "fixture.wire",
      () => wire()
    ): Unit
    assert(
      intercept[IllegalArgumentException](
        Signature.stateSchema(root, Some("fixture.State"), "state nested")
      ).getMessage.contains("fixture.Inner")
    )
    val resolved = Signature.stateTypes(
      Seq(root -> Some("fixture.State"), domain -> Some("fixture.Inner")),
      "state nested"
    )
    assertEquals(
      resolved.find(_.name == "fixture.Inner").get.getRecord.fields.head.getType.getChannel,
      "fixture.wire"
    )
    val incomplete = witness(nested)
    assert(
      intercept[IllegalArgumentException](
        Signature.stateSchema(incomplete, Some("fixture.State"), "state incomplete")
      ).getMessage.contains("fixture.Inner")
    )
    val named = ir.TypeRef(ir.TypeRef.Ref.Named("fixture.Message"))
    val messageType =
      ir.Type("fixture.Message", shape = ir.Type.Shape.Enum(ir.Enum(Seq(ir.Case("ping")))))
    val messageMissing =
      bind(witness(pending = named), () => wire(ref = named, closure = Seq(messageType)))
    val missingMessage = intercept[IllegalArgumentException](
      Signature.stateSchema(messageMissing, Some("fixture.State"), "state message closure")
    )
    assert(missingMessage.getMessage.contains("box"), missingMessage.getMessage)
    assert(missingMessage.getMessage.contains("incomplete"), missingMessage.getMessage)

  test("one selected witness cannot borrow another witness's same-type ownership"):
    val missing = witness()
    val complete = bind(witness(), () => wire())
    val error = intercept[IllegalArgumentException](
      Signature.stateTypes(
        Seq(missing -> Some("fixture.State"), complete -> Some("fixture.State")),
        "state exact witness"
      )
    )
    assert(error.getMessage.contains("box"), error.getMessage)

  test("an uninitialized supplier refuses and may resolve after initialization"):
    var available: Channel[?] = null
    var calls = 0
    val domain = bind(
      witness(),
      () =>
        calls += 1; available
    )
    assert(
      intercept[IllegalArgumentException](
        Signature.stateSchema(domain, Some("fixture.State"), "state early")
      ).getMessage.contains("initialized")
    )
    available = wire()
    for _ <- 0 until 2 do
      assertEquals(
        Signature
          .stateSchema(domain, Some("fixture.State"), "state ready")
          .types
          .head
          .getRecord
          .fields
          .head
          .getType
          .getChannel,
        "fixture.wire"
      )
    assertEquals(calls, 2)
