package framework.dsl

import io.temporal.server.api.umpire.v1 as ir
import io.temporal.server.api.umpire.v1.Expr.Kind as E

private[framework] object RuleLowering:
  final case class Row(
      cls: Option[Seq[ir.Value]],
      guard: ir.Expr,
      effect: ir.Expr,
      position: Option[ir.Position]
  )

  def stateName(inputs: Seq[ir.Param], fail: String => Nothing): String =
    Iterator("s", "state", "s0")
      .find(n => !inputs.exists(_.name == n))
      .getOrElse(fail("rule inputs exhaust the state parameter names s, state and s0"))

  def valuesOf(
      t: ir.TypeRef,
      types: collection.Map[String, ir.Type],
      fail: String => Nothing
  ): Seq[ir.Value] = t.ref match
    case ir.TypeRef.Ref.Bool(_)     => Seq(false, true).map(b => ir.Value(ir.Value.Kind.Bool(b)))
    case ir.TypeRef.Ref.IntRange(r) => (r.low to r.high).map(i => ir.Value(ir.Value.Kind.Int(i)))
    case ir.TypeRef.Ref.Named(n)    =>
      def product(fields: Seq[ir.Field]): Seq[Seq[ir.Value]] =
        fields.foldLeft(Seq(Seq.empty[ir.Value]))((vs, f) =>
          for v <- vs; x <- valuesOf(f.getType, types, fail) yield v :+ x
        )
      types.get(n).map(_.shape) match
        case Some(ir.Type.Shape.Enum(e)) =>
          e.cases.flatMap(c =>
            product(c.fields).map(fs => ir.Value(ir.Value.Kind.Enum(ir.EnumValue(n, c.name, fs))))
          )
        case Some(ir.Type.Shape.Record(r)) =>
          product(r.fields).map(fs => ir.Value(ir.Value.Kind.Record(ir.RecordValue(n, fs))))
        case _ => fail(s"$n has no values the rules can name")
    case _ => fail("an input of a rule's action has no finite values")

  def patternOf(v: ir.Value): ir.Pattern = v.kind match
    case ir.Value.Kind.Enum(e) if e.fields.nonEmpty =>
      ir.Pattern(
        ir.Pattern.Kind.Case(
          ir.CasePattern(`type` = e.`type`, `case` = e.`case`, fields = e.fields.map(patternOf))
        )
      )
    case _ => ir.Pattern(ir.Pattern.Kind.Literal(v))

  def lowered(
      machine: String,
      state: ir.TypeRef,
      action: String,
      inputs: Seq[ir.Param],
      rules: Seq[Row],
      at: Option[ir.Position],
      types: collection.Map[String, ir.Type],
      fail: String => Nothing
  ): ir.Function =
    def expr(position: Option[ir.Position])(kind: E) = ir.Expr(position = position, kind = kind)
    def chain(rs: Seq[Row]): ir.Expr =
      rs.foldRight(expr(at)(E.List(ir.ListOf())))((r, rest) =>
        expr(r.position)(E.If(ir.If(Some(r.guard), Some(r.effect), Some(rest))))
      )
    def byInputs(i: Int, fixed: Seq[ir.Value]): ir.Expr =
      if i == inputs.size then chain(rules.filter(_.cls.forall(_.equals(fixed))))
      else
        val input = inputs(i)
        val cases = valuesOf(input.getType, types, fail).map(v =>
          ir.MatchCase(pattern = Some(patternOf(v)), body = Some(byInputs(i + 1, fixed :+ v)))
        )
        expr(at)(E.Match(ir.Match(Some(expr(at)(E.Var(input.name))), cases)))
    ir.Function(
      name = s"$machine.rules.$action",
      position = at,
      params = ir.Param(stateName(inputs, fail), Some(state)) +: inputs,
      body = Some(if rules.exists(_.cls.nonEmpty) then byInputs(0, Nil) else chain(rules))
    )
