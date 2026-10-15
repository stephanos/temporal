package framework.dsl

import java.util.Base64
import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir

class MachineEndpointsSuite extends munit.FunSuite:
  private val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(Empty()))
  private val literal = ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Bool(true))))
  private def end(tpe: ir.TypeRef = bool, body: ir.Expr = literal): ir.Expr =
    ir.Expr(kind = ir.Expr.Kind.Lambda(ir.Lambda(Seq(ir.Param("written", Some(tpe))), Some(body))))
  private def machine: Machine[Boolean, Boolean, Nothing] =
    new Machine[Boolean, Boolean, Nothing]:
      def init: Boolean = throw AssertionError("init read")
      def end(s: Boolean): Boolean = throw AssertionError("end read")
      def rules: RuleBook[Boolean, Boolean, Nothing] = throw AssertionError("rules read")
      override def name: String = throw AssertionError("public name read")
  private def chunks(bytes: Array[Byte]): Seq[String] = Seq(Base64.getEncoder.encodeToString(bytes))
  private def retained(
      source: Machine[?, ?, ?],
      starts: Seq[ir.Expr] = Seq(literal),
      ends: ir.Expr = end(),
      model: ir.Model = ir.Model()
  ): Unit =
    source.retainHeaderIdentity("fixture", "source")
    source.retainEndpoints(
      source.getClass.getName,
      "fixture",
      "source",
      chunks(ir.ListOf(starts).toByteArray),
      chunks(ends.toByteArray),
      chunks(model.toByteArray)
    )

  test(
    "ordered roots and reachable requires bodies types and external identities stay exact and inert"
  ):
    val nested = ir.Type("Nested", shape = ir.Type.Shape.Enum(ir.Enum(Seq(ir.Case("value")))))
    val record = ir.Type(
      "Record",
      position = Some(ir.Position("types.scala", 3)),
      shape = ir.Type.Shape.Record(
        ir.Record(
          Seq(
            ir.Field("nested", Some(ir.TypeRef(ir.TypeRef.Ref.Named("Nested")))),
            ir.Field("box", Some(ir.TypeRef(ir.TypeRef.Ref.Channel("fixture.wire"))))
          )
        )
      )
    )
    val guard = ir.Function(
      "guard",
      position = Some(ir.Position("helpers.scala", 12)),
      params = Seq(ir.Param("s", Some(bool))),
      body = Some(literal)
    )
    val helper = ir.Function(
      "helper",
      position = Some(ir.Position("helpers.scala", 16)),
      params = Seq(ir.Param("arg", Some(bool))),
      body = Some(ir.Expr(kind = ir.Expr.Kind.Var("arg"))),
      requires = Some(
        ir.Expr(kind =
          ir.Expr.Kind.Call(
            ir.Call("guard", Seq(ir.Expr(kind = ir.Expr.Kind.Hole("fixture.unknown"))))
          )
        )
      )
    )
    val falseValue = literal.copy(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Bool(false))))
    val ends = end(
      ir.TypeRef(ir.TypeRef.Ref.Named("Record")),
      ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("helper", Seq(literal))))
    )
    val source = machine
    retained(
      source,
      Seq(literal, falseValue),
      ends,
      ir.Model(
        functions = Seq(helper, guard, ir.Function("unused")),
        types = Seq(record, nested, ir.Type("Unused"))
      )
    )
    val value = MachineEndpoints.collect(source)
    assertEquals(value.starts, Vector(literal, falseValue))
    assertEquals(value.end, ends)
    assertEquals(value.functions, Vector(guard, helper))
    assertEquals(value.types, Vector(nested, record))
    assertEquals(value.channels, Vector("fixture.wire"))
    assertEquals(value.holes, Vector("fixture.unknown"))
    assertEquals(MachineEndpoints.collect(source), value)

  test(
    "malformed roots expression children parameters type closure and callees refuse contextually"
  ):
    val call = ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("helper")))
    val function =
      ir.Function("helper", params = Seq(ir.Param("arg", Some(bool))), body = Some(literal))
    val named = ir.TypeRef(ir.TypeRef.Ref.Named("Absent"))
    val cases = Seq(
      (Seq.empty[ir.Expr], end(), ir.Model(), "starts"),
      (Seq(literal), literal, ir.Model(), "state Lambda"),
      (Seq(ir.Expr()), end(), ir.Model(), "expression"),
      (
        Seq(literal),
        end(body =
          ir.Expr(kind =
            ir.Expr.Kind.Binary(ir.Binary(op = ir.Binary.Op.OP_EQ, left = Some(literal)))
          )
        ),
        ir.Model(),
        "right operand"
      ),
      (
        Seq(literal),
        end(body = ir.Expr(kind = ir.Expr.Kind.Lambda(ir.Lambda()))),
        ir.Model(),
        "lambda body"
      ),
      (Seq(literal), end(ir.TypeRef()), ir.Model(), "type reference"),
      (Seq(literal), end(named), ir.Model(), "type Absent"),
      (Seq(literal), end(named), ir.Model(types = Seq(ir.Type("Absent"))), "shape"),
      (
        Seq(literal),
        end(named),
        ir.Model(types = Seq(ir.Type("Absent", shape = ir.Type.Shape.Enum(ir.Enum())))),
        "enum cases"
      ),
      (
        Seq(literal),
        end(named),
        ir.Model(types =
          Seq(ir.Type("Absent", shape = ir.Type.Shape.Record(ir.Record(Seq(ir.Field("missing"))))))
        ),
        "field missing"
      ),
      (Seq(literal), end(body = call), ir.Model(), "callee/root helper"),
      (Seq(literal), end(body = call), ir.Model(functions = Seq(function.clearBody)), "body"),
      (
        Seq(literal),
        end(body = call),
        ir.Model(functions = Seq(function.copy(params = Seq(ir.Param("arg"))))),
        "parameter arg"
      ),
      (Seq(literal), end(body = ir.Expr(kind = ir.Expr.Kind.Var("absent"))), ir.Model(), "binding"),
      (
        Seq(literal),
        end(body = ir.Expr(kind = ir.Expr.Kind.Hole(""))),
        ir.Model(),
        "external hole"
      ),
      (Seq(literal), end(ir.TypeRef(ir.TypeRef.Ref.Channel(""))), ir.Model(), "external channel")
    )
    cases.foreach { (starts, ends, model, reason) =>
      val error = intercept[IllegalArgumentException](retained(machine, starts, ends, model))
      assert(error.getMessage.contains("Machine fixture.source endpoints"), error.getMessage)
      assert(error.getMessage.contains(reason), error.getMessage)
    }

  test("pending missing stale malformed and conflicting retained provenance never escapes"):
    val pending = machine
    assert(
      intercept[IllegalArgumentException](MachineEndpoints.collect(pending)).getMessage
        .contains("registration")
    )
    pending.retainHeaderIdentity("fixture", "source")
    assert(
      intercept[IllegalArgumentException](MachineEndpoints.collect(pending)).getMessage
        .contains("metadata")
    )
    val args = (
      chunks(ir.ListOf(Seq(literal)).toByteArray),
      chunks(end().toByteArray),
      chunks(ir.Model().toByteArray)
    )
    for owner <- Seq("", "wrong.Owner") do
      intercept[IllegalArgumentException](
        pending.retainEndpoints(owner, "fixture", "source", args._1, args._2, args._3)
      ): Unit
    intercept[IllegalArgumentException](
      pending.retainEndpoints(
        pending.getClass.getName,
        "wrong",
        "source",
        args._1,
        args._2,
        args._3
      )
    ): Unit
    intercept[IllegalArgumentException](
      pending.retainEndpoints(
        pending.getClass.getName,
        "fixture",
        "source",
        Seq("!"),
        args._2,
        args._3
      )
    ): Unit
    retained(pending)
    retained(pending)
    intercept[IllegalArgumentException](
      retained(
        pending,
        Seq(literal),
        end(body = ir.Expr(kind = ir.Expr.Kind.Hole("fixture.changed")))
      )
    ): Unit
    val original = pending.endpointMetadata
    val stale = new Machine[Boolean, Boolean, Nothing]:
      def init = false
      def end(s: Boolean) = s
      object rules extends Bindings()
      override private[framework] def endpointMetadata = original
    stale.retainHeaderIdentity("fixture", "source")
    assert(
      intercept[IllegalArgumentException](MachineEndpoints.collect(stale)).getMessage
        .contains("stale endpoint source")
    )
    val root = ir.Function("helper", body = Some(literal))
    val changed = root.copy(body = Some(ir.Expr(kind = ir.Expr.Kind.Hole("fixture.changed"))))
    val shape = ir.Type("Record", shape = ir.Type.Shape.Record(ir.Record()))
    val differentShape = shape.copy(shape = ir.Type.Shape.Enum(ir.Enum()))
    for model <- Seq(
        ir.Model(functions = Seq(root, changed)),
        ir.Model(types = Seq(shape, differentShape)),
        ir.Model(machines = Seq(ir.Machine(name = "foreign")))
      )
    do intercept[IllegalArgumentException](retained(machine, model = model)): Unit
