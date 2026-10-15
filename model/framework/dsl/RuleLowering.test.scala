package framework.dsl

import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir

class RuleLoweringSuite extends munit.FunSuite:
  private val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(Empty()))
  private val at = Some(ir.Position("Rules.scala", 12))
  private def literal(value: Boolean) = ir.Value(ir.Value.Kind.Bool(value))
  private def expr(value: Boolean) = ir.Expr(kind = ir.Expr.Kind.Literal(literal(value)))
  private def fail(message: String): Nothing = throw IllegalArgumentException(message)

  test("class lowering retains every nested domain leaf and ordered whole-action participation"):
    val rows = Seq(
      RuleLowering.Row(Some(Seq(literal(false), literal(true))), expr(false), expr(true), at),
      RuleLowering.Row(None, expr(true), expr(false), at)
    )
    val result = RuleLowering.lowered(
      "m",
      bool,
      "go",
      Seq(ir.Param("s", Some(bool)), ir.Param("state", Some(bool))),
      rows,
      at,
      Map.empty,
      fail
    )
    assertEquals(result.name, "m.rules.go")
    assertEquals(result.params.map(_.name), Seq("s0", "s", "state"))
    val outer = result.getBody.getMatch
    assertEquals(outer.getScrutinee.getVar, "s")
    assertEquals(outer.cases.map(_.getPattern.getLiteral), Seq(literal(false), literal(true)))
    val inner = outer.cases.head.getBody.getMatch
    assertEquals(inner.getScrutinee.getVar, "state")
    assertEquals(inner.cases.map(_.getPattern.getLiteral), Seq(literal(false), literal(true)))
    assertEquals(inner.cases.head.getBody.getIf.getCondition, expr(true))
    val selected = inner.cases(1).getBody.getIf
    assertEquals(selected.getCondition, expr(false))
    assertEquals(selected.getElse.getIf.getCondition, expr(true))
    assertEquals(selected.getElse.getIf.getElse.getList.items, Nil)
    assertEquals(result.position, at)
    assertEquals(outer.cases.head.getBody.position, at)

  test("domain enumeration preserves range, enum fields and record products"):
    val pair = ir.Type(
      name = "Pair",
      shape =
        ir.Type.Shape.Record(ir.Record(Seq(ir.Field("a", Some(bool)), ir.Field("b", Some(bool)))))
    )
    val mode = ir.Type(
      name = "Mode",
      shape = ir.Type.Shape.Enum(
        ir.Enum(Seq(ir.Case("plain"), ir.Case("tagged", Seq(ir.Field("flag", Some(bool))))))
      )
    )
    val types = Map("Pair" -> pair, "Mode" -> mode)
    val values = RuleLowering.valuesOf(ir.TypeRef(ir.TypeRef.Ref.Named("Mode")), types, fail)
    assertEquals(
      values.map(v => v.getEnum.`case` -> v.getEnum.fields),
      Seq("plain" -> Nil, "tagged" -> Seq(literal(false)), "tagged" -> Seq(literal(true)))
    )
    assertEquals(RuleLowering.patternOf(values(1)).getCase.fields.head.getLiteral, literal(false))
    val records = RuleLowering.valuesOf(ir.TypeRef(ir.TypeRef.Ref.Named("Pair")), types, fail)
    assertEquals(
      records.map(_.getRecord.fields),
      Seq(
        Seq(literal(false), literal(false)),
        Seq(literal(false), literal(true)),
        Seq(literal(true), literal(false)),
        Seq(literal(true), literal(true))
      )
    )
    assert(RuleLowering.patternOf(records.head).kind.isLiteral)
    val range = ir.TypeRef(ir.TypeRef.Ref.IntRange(ir.IntRange(0, 1)))
    assertEquals(RuleLowering.valuesOf(range, types, fail).map(_.getInt), Seq(0L, 1L))

  test("missing finite domains and exhausted state names refuse contextually"):
    interceptMessage[IllegalArgumentException]("Missing has no values the rules can name") {
      RuleLowering.valuesOf(ir.TypeRef(ir.TypeRef.Ref.Named("Missing")), Map.empty, fail)
    }: Unit
    val inputs = Seq("s", "state", "s0").map(ir.Param(_, Some(bool)))
    intercept[IllegalArgumentException](RuleLowering.stateName(inputs, fail))
