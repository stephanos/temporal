package framework.dsl

import io.temporal.server.api.umpire.v1 as ir

private[framework] object EvidenceLowering:
  private def line(fact: String, name: String, at: ir.Position): ir.MatchCase =
    ir.MatchCase(
      pattern = Some(
        ir.Pattern(ir.Pattern.Kind.Literal(ir.Value(ir.Value.Kind.Enum(ir.EnumValue(fact, name)))))
      ),
      body = Some(
        ir.Expr(
          position = Some(at),
          kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Text(name)))
        )
      )
    )

  def defaultEvidence(
      name: String,
      fact: ir.TypeRef,
      parameter: String,
      spelling: String,
      at: ir.Position,
      cases: Seq[(String, Boolean)],
      fail: String => Nothing
  ): ir.Function =
    for (c, _) <- cases.find(_._2) do
      fail(
        s"fact $c has fields, so the evidence of its name alone does not say what confirms it: " +
          s"declare `evidence { case $spelling.$c(...) => ... }`"
      )
    ir.Function(
      name = name,
      position = Some(at),
      params = Seq(ir.Param(parameter, Some(fact))),
      body = Some(
        ir.Expr(
          position = Some(at),
          kind = ir.Expr.Kind.Match(
            ir.Match(
              Some(ir.Expr(position = Some(at), kind = ir.Expr.Kind.Var(parameter))),
              cases.map(c => line(fact.getNamed, c._1, at))
            )
          )
        )
      )
    )

  def complete(
      fn: ir.Function,
      fact: ir.TypeRef,
      cases: => Seq[(String, Boolean)],
      at: ir.Position,
      fail: String => Nothing
  ): ir.Function = fn.getBody.kind match
    case ir.Expr.Kind.Match(m)
        if fn.params.size == 1 && m.getScrutinee.kind.`var`.contains(fn.params.head.name) =>
      val catalog = cases
      val index = catalog.map(_._1).zipWithIndex.toMap
      def covers(p: ir.Pattern): Map[String, Boolean] = p.kind match
        case ir.Pattern.Kind.Wildcard(_)        => catalog.map(_._1 -> true).toMap
        case ir.Pattern.Kind.Bind(b)            => covers(b.getPattern)
        case ir.Pattern.Kind.Alternatives(alts) =>
          alts.patterns
            .map(covers)
            .foldLeft(Map.empty[String, Boolean])((acc, c) =>
              c.foldLeft(acc) { case (acc, (k, all)) =>
                acc.updated(k, acc.getOrElse(k, false) || all)
              }
            )
        case ir.Pattern.Kind.Literal(v) if v.kind.isEnum => Map(v.getEnum.`case` -> true)
        case ir.Pattern.Kind.Case(c)                     =>
          def total(p: ir.Pattern): Boolean = p.kind match
            case ir.Pattern.Kind.Wildcard(_) => true
            case ir.Pattern.Kind.Bind(b)     => total(b.getPattern)
            case _                           => false
          Map(c.`case` -> c.fields.forall(total))
        case _ => Map.empty
      val covered = m.cases.map(line =>
        if line.guard.nonEmpty then covers(line.getPattern).map((k, _) => k -> false)
        else covers(line.getPattern)
      )
      val whole = covered.flatMap(_.collect { case (k, true) => k }).toSet
      val named = covered.flatMap(_.keys).toSet
      for (c, fields) <- catalog if fields && !whole(c) do
        fail(
          (if named(c) then s"evidence covers only some values of fact $c"
           else s"evidence names no line for fact $c") +
            ", which has fields: its name alone does not say what confirms it, so write a line for all of it"
        )
      val missing = catalog.collect { case (c, _) if !whole(c) => c }
      if missing.isEmpty then fn
      else
        val first = covered.map(_.keys.map(index).minOption.getOrElse(Int.MaxValue))
        val last = missing.map(c => c -> covered.lastIndexWhere(_.contains(c))).toMap
        val (lines, left) = m.cases.indices.foldLeft((Vector.empty[ir.MatchCase], missing)) {
          case ((lines, left), i) =>
            val (before, after) = left.partition(c => index(c) < first(i) && last(c) < i)
            (lines ++ before.map(line(fact.getNamed, _, at)) :+ m.cases(i), after)
        }
        fn.withBody(
          fn.getBody.withMatch(m.withCases(lines ++ left.map(line(fact.getNamed, _, at))))
        )
    case _ => fn
