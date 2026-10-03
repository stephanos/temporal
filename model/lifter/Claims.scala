package umpire.lift

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir

/**
 * What a declaration folds to at lift time: a name or text the IR uses, or a declaration still being
 * built by the calls chained onto it.
 */
private enum Decl:
  case Text(value: String)

  /** A machine or a composition, by name. */
  case Model(name: String)
  case Items(items: List[Decl])
  case PropertyOn(machine: String, name: String, when: Option[Either[ir.ActionClass, String]])
  case ScenarioOn(machine: String, name: String, start: Option[ir.Expr])

  /** A declared Property or Scenario. */
  case Claim(ref: ir.ClaimRef)
  case QueryNamed(name: String)
  case QueryOn(name: String, form: ir.Query.Form, property: ir.ClaimRef)
  case QueryIn(
      name: String,
      form: ir.Query.Form,
      property: ir.ClaimRef,
      scenario: ir.ClaimRef,
      through: Option[String]
  )

  /** How a Scenario reads a Property: the machine whose refinement it reads through, if any. */
  case Reading(through: Option[String])
  case Bounds(limits: ir.Limits)

  /** A declared Query or progress claim. */
  case Declared(name: String)

private[lift] trait Claims:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Claims: Properties, Scenarios, Queries and progress, folded with their arguments bound

  def modelName(d: Decl, at: Tree): String = d match
    case Decl.Model(name) => name
    case other            => fail(at, s"expected a machine or a composition, got $other")
  def textOf(d: Decl, at: Tree): String = d match
    case Decl.Text(s) => s
    case other        => fail(at, s"expected a string, got $other")
  def claimOf(d: Decl, at: Tree): ir.ClaimRef = d match
    case Decl.Claim(ref) => ref
    case other           => fail(at, s"expected a Property or a Scenario, got $other")
  def claim(machine: String, name: String): ir.ClaimRef = ir.ClaimRef(machine, name)

  /** One class of an action: the action bare, or applied to one value per input. */
  def classOf(t: Term): ir.ActionClass = t match
    case Typed(e, _)                                                    => classOf(e)
    case Apply(Apply(fn, List(a)), values) if fn.symbol.name == "apply" =>
      ir.ActionClass(action(a), values.map(literalValue))
    case ref => ir.ActionClass(action(ref))

  /**
   * Keeps the first declaration under a key and refuses a different second one, which would share
   * its Definition ID.
   */
  def register[K, V](
      into: mutable.LinkedHashMap[K, V],
      key: K,
      value: V,
      at: Tree,
      what: String
  ): Unit =
    into.get(key) match
      case Some(existing) if existing != value =>
        fail(at, s"$what is declared twice, and both would share one Definition ID")
      case _ => into(key) = value

  def limitsOf(t: Term): ir.Limits = arguments(t) match
    case Apply(Select(companion, "apply"), List(name, steps, actions, search))
        if companion.tpe.typeSymbol.companionClass.fullName == "umpire.Limits" =>
      val l = ir.Limits(
        constString(name),
        constInt(steps).toInt,
        constInt(actions).toInt,
        constInt(search).toInt
      )
      for
        (bound, v) <- Seq("steps" -> l.steps, "actions" -> l.actions, "search" -> l.search)
        if v < 0
      do fail(t, s"limits ${l.name} declare $v $bound; a bound is at least 0")
      l
    case other =>
      fail(
        other,
        s"Limits are declared by `Limits(name, steps, actions, search)`, not ${other.show}"
      )

  def query(
      name: String,
      form: ir.Query.Form,
      p: ir.ClaimRef,
      s: ir.ClaimRef,
      through: Option[String],
      limits: ir.Limits,
      at: Tree
  ): Decl =
    through match
      case Some(m) =>
        if m != s.machine then
          fail(
            at,
            s"$name reads its Property through the refinement of $m, and its Scenario runs on ${s.machine}"
          )
        val refined = machineNamed(m).map(_.getRefines.product).filter(_.nonEmpty)
        if !refined.contains(p.machine) then
          fail(
            at,
            s"$name reads ${p.name}, a Property of ${p.machine}, through the refinement of $m, which " +
              s"refines ${refined.getOrElse("nothing")}"
          )
      case None =>
        if p.machine != s.machine then
          fail(
            at,
            s"$name pairs ${p.name}, a Property of ${p.machine}, with ${s.name}, a Scenario of " +
              s"${s.machine}, and reads it through no refinement"
          )
    val q = ir.Query(
      name = name,
      position = Some(pos(at)),
      form = form,
      property = Some(p),
      scenario = Some(s),
      through = through.nonEmpty,
      limits = Some(limits)
    )
    register(queries, name, q, at, s"Query $name")
    Decl.Declared(name)

  /**
   * What a declaration of the lifted sources folds to, with the helper function parameters bound
   * in `env`.
   */
  def fold(t: Term, env: Map[Symbol, Decl]): Decl = t match
    case Typed(e, _)                                => fold(e, env)
    case Inlined(_, Nil, e)                         => fold(e, env)
    case NamedArg(_, e)                             => fold(e, env)
    case Block(stats, _: Apply) if synthetic(stats) => fold(arguments(t), env)
    case Block(stats, e)                            =>
      val inner = stats.foldLeft(env) {
        case (acc, v @ ValDef(_, _, Some(rhs))) => acc + (v.symbol -> fold(rhs, acc))
        case (_, other)                         => fail(other, s"not a declaration: ${other.show}")
      }
      fold(e, inner)
    case Literal(StringConstant(s))       => Decl.Text(s)
    case r: Ref if env.contains(r.symbol) => env(r.symbol)
    // `s"..."`, with each argument folded to its text.
    case Apply(Select(Apply(Select(sc, "apply"), List(parts)), "s"), List(args))
        if sc.symbol.fullName == "scala.StringContext" =>
      val texts = varargs(args).map(a => textOf(fold(a, env), a))
      Decl.Text(varargs(parts).map(constString).zipAll(texts, "", "").map(_ + _).mkString)
    case Select(m, "name")
        if isNamed(m.tpe, "umpire.Machine") || isNamed(m.tpe, "umpire.Composition") =>
      Decl.Text(modelName(fold(m, env), m))
    case Apply(TypeApply(Select(Ident("Vector" | "List"), "apply"), _), List(items)) =>
      Decl.Items(varargs(items).map(fold(_, env)))

    case Apply(Apply(TypeApply(Ident("property"), _), List(m)), List(name)) =>
      Decl.PropertyOn(modelName(fold(m, env), m), textOf(fold(name, env), name), None)
    case Apply(Select(b, "when"), List(c)) =>
      fold(b, env) match
        case p: Decl.PropertyOn => p.copy(when = Some(Left(classOf(c))))
        case other              => fail(t, s"when restricts a Property, not $other")
    case Apply(Select(b, "whenAction"), List(a)) =>
      fold(b, env) match
        case p: Decl.PropertyOn => p.copy(when = Some(Right(constString(a))))
        case other              => fail(t, s"whenAction restricts a Property, not $other")
    case Apply(Select(b, op @ ("holds" | "holdsAcross")), List(f)) =>
      fold(b, env) match
        case Decl.PropertyOn(m, name, when) =>
          val p = ir.Property(
            machine = m,
            name = name,
            position = Some(pos(t)),
            holds = stepFunction(f, s"$m.property", name),
            transition = op == "holdsAcross",
            when = when.fold(ir.Property.When.Empty)(
              _.fold(ir.Property.When.WhenClass(_), ir.Property.When.WhenAction(_))
            )
          )
          register(properties, (m, name), p, t, s"Property $name of $m")
          Decl.Claim(claim(m, name))
        case other => fail(t, s"$op finishes a Property, not $other")

    case Apply(Apply(TypeApply(Ident("scenario"), _), List(m)), List(name)) =>
      Decl.ScenarioOn(modelName(fold(m, env), m), textOf(fold(name, env), name), None)
    case Apply(Select(b, "starts"), List(s)) =>
      fold(b, env) match
        case sc: Decl.ScenarioOn => sc.copy(start = Some(lift(s)))
        case other               => fail(t, s"starts begins a Scenario, not $other")
    case Apply(Select(b, op @ ("actions" | "actionKeys")), List(items)) =>
      scenario(fold(b, env), t) { s =>
        if op == "actions" then s.addAllActions(varargs(items).map(classOf))
        else s.addAllKeys(varargs(items).map(constString))
      }
    case Select(b, "free") => scenario(fold(b, env), t)(_.withFree(true))

    case Apply(Ident("query"), List(name)) => Decl.QueryNamed(textOf(fold(name, env), name))
    case Apply(TypeApply(Select(q, form @ ("find" | "verify")), _), List(p)) =>
      fold(q, env) match
        case Decl.QueryNamed(name) =>
          val f = if form == "find" then ir.Query.Form.FORM_FIND else ir.Query.Form.FORM_VERIFY
          Decl.QueryOn(name, f, claimOf(fold(p, env), p))
        case other => fail(t, s"$form asks a Query, not $other")
    case Apply(Apply(TypeApply(Select(q, "in"), _), List(s)), List(reads)) =>
      (fold(q, env), fold(reads, env)) match
        case (Decl.QueryOn(name, form, p), Decl.Reading(through)) =>
          Decl.QueryIn(name, form, p, claimOf(fold(s, env), s), through)
        case (other, _) => fail(t, s"in gives a Query its Scenario, not $other")
    case Apply(Select(q, "explore"), List(space)) =>
      fold(q, env) match
        case Decl.Declared(name) if queries.contains(name) =>
          queries(name) =
            queries(name).withExploration(emit(ir.Exploration, Bound(space, Map.empty)))
          Decl.Declared(name)
        case other => fail(t, s"explore declares a Query's finite variations, not $other")
    case Apply(Select(q, "expect"), List(expected)) =>
      fold(q, env) match
        case Decl.Declared(name) if queries.contains(name) =>
          queries(name) =
            queries(name).withExpectedRun(emit(ir.RunExpectation, Bound(expected, Map.empty)))
          Decl.Declared(name)
        case other => fail(t, s"expect declares a Query's live assessment, not $other")
    case Apply(Select(q, "limits"), List(l)) =>
      fold(q, env) match
        case Decl.QueryIn(name, form, p, s, through) =>
          val limits = fold(l, env) match
            case Decl.Bounds(limits) => limits
            case other               => fail(l, s"expected Limits, got $other")
          query(name, form, p, s, through, limits, t)
        case other => fail(t, s"limits bounds a Query, not $other")
    case Apply(TypeApply(Select(Ident("Reads"), "through"), _), List(m, _)) =>
      Decl.Reading(Some(modelName(fold(m, env), m)))
    case TypeApply(Ident("identity"), _) if t.symbol.owner.fullName.startsWith("umpire.Reads") =>
      Decl.Reading(None)
    case Apply(Select(companion, "apply"), _)
        if companion.tpe.typeSymbol.companionClass.fullName == "umpire.Limits" =>
      Decl.Bounds(limitsOf(t))

    case Apply(
          Apply(Apply(TypeApply(Ident("leadsTo"), _), List(m)), List(name)),
          List(from, to, within, under)
        ) =>
      val (machine, n) = (modelName(fold(m, env), m), textOf(fold(name, env), name))
      val steps = constInt(within)
      if steps < 1 then
        fail(t, s"progress $n bounds itself within $steps steps; a bound is at least one step")
      val p = ir.Progress(
        machine = machine,
        name = n,
        position = Some(pos(t)),
        from = stepFunction(from, s"$machine.progress.$n", "from"),
        to = stepFunction(to, s"$machine.progress.$n", "to"),
        within = steps.toInt,
        assumptions = varargs(under).map(a => assumptionOf(resolveSymbol(a), a))
      )
      register(progress, (machine, n), p, t, s"progress $n of $machine")
      Decl.Declared(n)

    // A value declared elsewhere, folded once: a machine or composition by its name, anything else
    // by its definition.
    case r: Ref if !isFunction(r.symbol) && defs.contains(resolveSymbol(r)) =>
      val sym = resolveSymbol(r)
      folded.getOrElseUpdate(
        sym,
        valDef(sym, r, "a declaration") match
          case d if isNamed(d.tpt.tpe, "umpire.Machine")     => Decl.Model(machineOf(sym, r).name)
          case d if isNamed(d.tpt.tpe, "umpire.Composition") =>
            Decl.Model(compositionOf(sym, r).name)
          case d => fold(d.rhs.get, Map.empty)
      )
    // A helper function of the lifted sources that declares: its body, with its arguments bound.
    case Apply(fn, args) if isFunction(fn.symbol) =>
      defs(fn.symbol) match
        case d: DefDef =>
          val params = d.termParamss.flatMap(_.params).map(_.symbol)
          fold(d.rhs.get, params.zip(args.map(fold(_, env))).toMap)
        case _ => fail(t, s"${fn.symbol.fullName} is not a function of the lifted sources")
    case other => fail(other, s"not a declaration the IR carries: ${other.show}")

  def scenario(d: Decl, at: Tree)(f: ir.Scenario => ir.Scenario): Decl = d match
    case Decl.ScenarioOn(m, name, start) =>
      val s = ir.Scenario(
        machine = m,
        name = name,
        position = Some(pos(at)),
        start = Some(start.getOrElse(fail(at, s"Scenario $name of $m names no start")))
      )
      register(scenarios, (m, name), f(s), at, s"Scenario $name of $m")
      Decl.Claim(claim(m, name))
    case other => fail(at, s"expected a Scenario, got $other")
