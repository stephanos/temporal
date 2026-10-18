package framework.dsl

import java.util.Base64
import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir

class MachineHeadersSuite extends munit.FunSuite:
  private val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(Empty()))
  private val nothing = ir.TypeRef(ir.TypeRef.Ref.Named("scala.Nothing"))
  private val emptyField = ir.Field("child", Some(ir.TypeRef()))
  private def witness: Finite[Boolean] = new Finite[Boolean]:
    def values = throw AssertionError("catalog read")
  private val stateName = "fixture.HeadersState"
  private val stateRef = ir.TypeRef(ir.TypeRef.Ref.Named(stateName))
  private val stateShape = ir.Type(
    name = stateName,
    shape = ir.Type.Shape.Record(ir.Record(Vector(ir.Field("ready", Some(bool)))))
  )
  private def stateWitness: Data[State] = new Data[State]:
    def compare(left: State, right: State): Int = throw AssertionError("state compared")
  private def machine(
      state: Data[State] = stateWitness,
      outcome: Finite[Boolean] = witness,
      facts: Finite[Nothing] = Finite.of()
  ): Machine[Boolean, Nothing] =
    new Machine[Boolean, Nothing](using outcome, facts):
      override private[framework] val fs: Data[State] = state
      @scala.annotation.unused
      var ready = false
      override def name: String = throw AssertionError("native name getter read")
      override def end: Boolean = throw AssertionError("end read")
      def transitions: RuleBook[State, Boolean, Nothing] = throw AssertionError("transitions read")
      override private[framework] def refinementEvidence: Option[Refinement[State, ?]] =
        throw AssertionError("refinement read")
  private def retain(
      value: MachineDeclaration[?, ?, ?],
      index: Int,
      key: String,
      root: ir.TypeRef,
      types: Seq[ir.Type] = Nil
  ): Unit =
    value.retainHeaderSchema(
      index,
      key,
      Seq(Base64.getEncoder.encodeToString(root.toByteArray)),
      Seq(Base64.getEncoder.encodeToString(ir.Model(types = types).toByteArray)),
      Nil
    )
  private def scalar(value: MachineDeclaration[?, ?, ?]): Unit =
    retain(value, 0, stateName, stateRef, Seq(stateShape))
    retain(value, 1, "scala.Boolean", bool)
    retain(value, 2, "scala.Nothing", nothing)

  test(
    "native state headers retain their record while scalar fields stay blank and empty family is distinct"
  ):
    val state = new Data[State]:
      def compare(left: State, right: State): Int = throw AssertionError("state compared")
    val value = machine(state = state)
    assert(value.fs eq state)
    value.retainHeaderIdentity("", "declared")
    scalar(value)
    val actual = MachineHeaders.collect(value)
    assertEquals(actual.header, MachineHeaders.Header("", "declared", stateName, "", ""))
    assertEquals(actual.types, Vector(stateShape))
    assertEquals(MachineHeaders.collect(value), actual)

  test("missing declaration and partial selections refuse before getters or catalogs are read"):
    val unnamed = machine()
    assert(
      intercept[IllegalArgumentException](MachineHeaders.collect(unnamed)).getMessage
        .contains("declaration identity")
    )
    val value = machine()
    value.retainHeaderIdentity("fixture", "incomplete")
    retain(value, 0, "scala.Boolean", bool)
    assert(
      intercept[IllegalArgumentException](MachineHeaders.collect(value)).getMessage
        .contains("outcome")
    )

  test("identity and exact selection attachment is idempotent and rejects conflicts"):
    val value = machine()
    value.retainHeaderIdentity("fixture", "owned")
    value.retainHeaderIdentity("fixture", "owned")
    scalar(value)
    scalar(value)
    assert(
      intercept[IllegalArgumentException](value.retainHeaderIdentity("other", "owned")).getMessage
        .contains("conflicting")
    )
    assert(
      intercept[IllegalArgumentException](retain(value, 0, "wrong", bool)).getMessage
        .contains("conflicting")
    )
    assert(
      intercept[IllegalArgumentException](
        retain(machine(), 0, "wrong", ir.TypeRef(ir.TypeRef.Ref.Named("Actual")))
      ).getMessage.contains("Actual")
    )

  test("missing and conflicting selected type closures refuse with Machine type context"):
    val named = ir.TypeRef(ir.TypeRef.Ref.Named("fixture.State"))
    val shape = ir.Type(
      name = "fixture.State",
      shape = ir.Type.Shape.Record(
        ir.Record(
          Vector(ir.Field("child", Some(ir.TypeRef(ir.TypeRef.Ref.Named("fixture.Absent")))))
        )
      )
    )
    val value = machine()
    value.retainHeaderIdentity("fixture", "broken")
    retain(value, 0, "fixture.State", named, Seq(shape))
    retain(value, 1, "scala.Boolean", bool)
    retain(value, 2, "scala.Nothing", nothing)
    val missing = intercept[IllegalArgumentException](MachineHeaders.collect(value)).getMessage
    assert(missing.contains("broken") && missing.contains("fixture.Absent"), missing)
    val conflicting = machine()
    conflicting.retainHeaderIdentity("fixture", "conflicting")
    retain(conflicting, 0, "fixture.State", named, Seq(shape))
    val emptyShape = shape.copy(shape = ir.Type.Shape.Record(ir.Record()))
    retain(conflicting, 1, "fixture.State", named, Seq(emptyShape))
    retain(conflicting, 2, "scala.Nothing", nothing)
    assert(
      intercept[IllegalArgumentException](MachineHeaders.collect(conflicting)).getMessage
        .contains("conflicting definitions")
    )

  test("empty selected roots refuse instead of manufacturing scalar blank header fields"):
    val value = machine()
    value.retainHeaderIdentity("fixture", "emptyRoot")
    val failure =
      intercept[IllegalArgumentException](retain(value, 0, "missing", ir.TypeRef())).getMessage
    assert(failure.contains("emptyRoot") && failure.contains("type"), failure)

  test("a scalar schema key must match the canonical selected type"):
    val value = machine()
    value.retainHeaderIdentity("fixture", "wrongScalarKey")
    val failure =
      intercept[IllegalArgumentException](retain(value, 0, "other.Boolean", bool)).getMessage
    assert(failure.contains("wrongScalarKey") && failure.contains("other.Boolean"), failure)

  test("a selected Type without a shape refuses with Machine and type context"):
    val value = machine()
    value.retainHeaderIdentity("fixture", "emptyShape")
    retain(
      value,
      0,
      "fixture.State",
      ir.TypeRef(ir.TypeRef.Ref.Named("fixture.State")),
      Seq(ir.Type(name = "fixture.State"))
    )
    retain(value, 1, "scala.Boolean", bool)
    retain(value, 2, "scala.Nothing", nothing)
    val failure = intercept[IllegalArgumentException](MachineHeaders.collect(value)).getMessage
    assert(
      failure.contains("emptyShape") && failure.contains("fixture.State") && failure.contains(
        "shape"
      ),
      failure
    )

  for (label, fields) <- Seq(
      "missing field reference" -> Vector(ir.Field("child")),
      "empty field reference" -> Vector(emptyField)
    )
  do
    test(s"selected closure refuses $label with Machine and field context"):
      val value = machine()
      value.retainHeaderIdentity("fixture", "brokenField")
      retain(
        value,
        0,
        "fixture.State",
        ir.TypeRef(ir.TypeRef.Ref.Named("fixture.State")),
        Seq(ir.Type(name = "fixture.State", shape = ir.Type.Shape.Record(ir.Record(fields))))
      )
      retain(value, 1, "scala.Boolean", bool)
      retain(value, 2, "scala.Nothing", nothing)
      val failure = intercept[IllegalArgumentException](MachineHeaders.collect(value)).getMessage
      assert(failure.contains("brokenField") && failure.contains("child"), failure)
