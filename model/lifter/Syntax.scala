package umpire.lift

import io.temporal.server.api.umpire.v1 as ir
import io.temporal.server.api.umpire.v1.Expr.Kind as E

/**
 * The lifting of the framework's sugar (model/umpire/Syntax.scala). The lifter does not inline a
 * framework body, so each sugar form is matched here by its definition and lowered to the IR its core
 * form lifts to; the lifter's tests lift both spellings and require the same IR.
 */
private[lift] trait Syntax:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // Top-level definitions of a file are members of its package object.
  private val sugarOwner = "umpire.Syntax$package$"

  /** The sugar definition a term applies, if it applies one: its name and its argument lists. */
  private def sugarCall(t: Term): Option[(String, List[List[Term]])] = t match
    case Apply(fn, args)  => sugarCall(fn).map((n, as) => (n, as :+ args))
    case TypeApply(fn, _) => sugarCall(fn)
    case r: Ref if r.symbol.maybeOwner.fullName == sugarOwner => Some(r.symbol.name -> Nil)
    case _                                                    => None

  /** The type arguments a sugar call is applied to, innermost first. */
  private def typeArgs(t: Term): List[TypeRepr] = t match
    case Apply(fn, _)        => typeArgs(fn)
    case TypeApply(_, targs) => targs.map(_.tpe)
    case _                   => Nil

  /**
   * Hook: whether the term applies a sugar form, which `sugar` lifts. Core form: none of its own;
   * `lift` asks it of every term before `sugar` lowers one, as in `case _ if sugared(t) => sugar(t)`.
   */
  def sugared(t: Term): Boolean = sugarCall(t).nonEmpty

  /**
   * Hook: a sugar form lifted to the IR of its core form. Core form: `accept(s, f)` lifts as
   * `List(Step(Outcome.accepted, s, List(f)))`, `stay(s)` as `List(Step(Outcome.accepted, s))`,
   * `disabled` as `Nil`, `x.in(a, b)` as `List(a, b).contains(x)`, `a implies b` as `!a || b`,
   * `after.records(f)` as `after.facts.contains(f)` and a composition's `after.records(_.member, f)`
   * as `after.facts.contains("member_f")`.
   */
  def sugar(t: Term): ir.Expr = sugarCall(t) match
    case Some(("accept", List(List(state, facts), List(accepted)))) =>
      list(Seq(step(outcomeOf(accepted, "accept"), lift(state), lift(facts), text("", t), t)), t)
    case Some(("stay", List(List(state), List(accepted)))) =>
      list(Seq(step(outcomeOf(accepted, "stay"), lift(state), list(Nil, t), text("", t), t)), t)
    case Some(("disabled", Nil))                            => list(Nil, t)
    case Some(("in", List(List(value), List(first, rest)))) =>
      val member = typeArgs(t).headOption
      val members = rest match
        case Typed(Repeated(items, _), _) => first :: items
        case other                        =>
          fail(
            other,
            "in lists its members, so a list passed as `xs*` has no IR form: write them out"
          )
      binary(ir.Binary.Op.OP_CONTAINS, lift(value), list(members.map(lift(_, member)), t), t)
    case Some(("implies", List(List(a), List(b)))) =>
      binary(
        ir.Binary.Op.OP_OR,
        expr(t)(E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(lift(a))))),
        lift(b),
        t
      )
    // A composition's step records the member's fact under the composed key `<field>_<fact>`.
    case Some(("records", List(List(after), List(member, fact)))) =>
      val facts = expr(t)(E.Field(ir.FieldAccess(Some(lift(after)), "facts")))
      binary(ir.Binary.Op.OP_CONTAINS, text(composedFact(member, fact, t), t), facts, t)
    case Some(("records", List(List(after), List(fact)))) =>
      val facts = expr(t)(E.Field(ir.FieldAccess(Some(lift(after)), "facts")))
      binary(ir.Binary.Op.OP_CONTAINS, lift(fact), facts, t)
    case _ => fail(t, s"outside the liftable subset: ${t.show}")

  /**
   * The composed key of the fact a composition's `after.records(_.member, fact)` reads,
   * `<field>_<fact>`. The selector is one field of the composed state, and where a lifted
   * composition of that state exists, a member fills the field and records facts of the fact's type.
   */
  private def composedFact(member: Term, fact: Term, at: Term): String =
    val (field, state) = lambda(member)
      .collect { case (List(p), body) => (fieldPath(p, body), p.tpt.tpe) }
      .collect { case (Some(List(field)), tpe) => (field, instantiated(tpe).widen.dealias) }
      .getOrElse(
        fail(
          at,
          "records reads the facts of the member one field of the composed state names, such as " +
            s"`_.activity`, not ${member.show}"
        )
      )
    val value = literalValue(fact)
    val factType = value.kind match
      case ir.Value.Kind.Enum(e) => e.`type`
      case _                     => fact.tpe.widen.dealias.typeSymbol.fullName
    for c <- compositions.values if c.stateType == state.typeSymbol.fullName do
      val m = c.members
        .find(_.field == field)
        .getOrElse(
          fail(at, s"records reads the facts of $field, and no member of ${c.name} fills it")
        )
      for machine <- machineNamed(m.machine) if machine.factType != factType do
        val records = if machine.factType.isEmpty then "no" else machine.factType
        fail(
          at,
          s"records reads the $factType fact ${valueKey(value)} of $field, and its member " +
            s"${m.machine} records $records facts"
        )
    s"${field}_${valueKey(value)}"

  // The claim patterns: the words that begin one on a PropertyBuilder, and the word that finishes
  // each, a method of the class the beginning returns.
  private val finishers = Map("once" -> "keeps", "never" -> "from", "stays" -> "unless")
  private val finished =
    Map("umpire.Once" -> "keeps", "umpire.Never" -> "from", "umpire.Stays" -> "unless")

  /**
   * Hook: whether a declaration applies a claim pattern, which `pattern` folds. Core form: none of
   * its own; `fold` asks it beside `holds`, as in `case _ if patterned(t) => pattern(t, env, named)`.
   */
  def patterned(t: Term): Boolean = t match
    case _: Apply =>
      val sym = t.symbol
      (sym.maybeOwner.fullName == sugarOwner && finishers.contains(sym.name)) ||
      finished.get(sym.maybeOwner.fullName).contains(sym.name)
    case _ => false

  /** `builder.word(arg)`, a claim pattern begun: the word, the builder, its state type and `arg`. */
  private def begun(t: Term): Option[(String, Term, TypeRepr, Term)] = t match
    case Apply(Apply(TypeApply(fn, state :: _), List(builder)), List(arg))
        if fn.symbol.maybeOwner.fullName == sugarOwner && finishers.contains(fn.symbol.name) =>
      Some((fn.symbol.name, builder, state.tpe, arg))
    case _ => None

  /**
   * Hook: a claim pattern folded to the Property its `holds` or `holdsAcross` lambda declares, its
   * function synthesized from calls of the author's predicates, each lifted as `holds` lifts one.
   * Core form: `once(over).keeps(_.x)` folds as
   * `holdsAcross((before, after) => !over(before) || after.state.x == before.x)`, `never(to)` as
   * `holds(after => !to(after))`, `never(to).from(b)` as
   * `holdsAcross((before, after) => !b(before) || !to(after))`, `stays(p)` as
   * `holdsAcross((before, after) => !p(before) || p(after.state))` and `stays(p).unless(r)` as
   * `holdsAcross((before, after) => !p(before) || p(after.state) || r(after))`.
   */
  def pattern(t: Term, env: Map[Symbol, Decl], named: Option[Symbol]): Decl =
    // The word that finishes the pattern and its argument, if one does, and the call it finishes.
    val (start, finish) = t match
      case Apply(TypeApply(Select(r, w @ "keeps"), _), List(x)) => (r, Some(w -> x))
      case Apply(Select(r, w @ ("from" | "unless")), List(x))   => (r, Some(w -> x))
      case _                                                    => (t, None)
    val (word, builder, state, arg) = begun(start).getOrElse {
      val w = finish.fold(t.symbol.name)(_._1)
      val begins = finishers.collectFirst { case (b, f) if f == w => b }.getOrElse(w)
      val written = start match
        case r: Ref => r.symbol.name
        case other  => other.show
      fail(
        t,
        s"$w finishes the `$begins(...)` written directly before it, not $written: chain it onto " +
          "that call"
      )
    }
    if word == "once" && finish.isEmpty then
      fail(t, "once declares nothing until `.keeps(...)` finishes it, written directly after it")
    val (m, name) = fold(builder, env, named) match
      case Decl.PropertyOn(m, name, None)    => (m, name)
      case Decl.PropertyOn(_, name, Some(_)) =>
        fail(
          t,
          s"$word declares $name about every step, so it takes no when: write the restriction into " +
            "its predicates, or declare the Property with `holds`"
        )
      case other => fail(t, s"$word finishes a Property, not $other")
    val fn = s"$m.property.$name"
    def variable(n: String) = expr(t)(E.Var(n))
    def not(e: ir.Expr) = expr(t)(E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(e))))
    def or(l: ir.Expr, r: ir.Expr) = binary(ir.Binary.Op.OP_OR, l, r, t)
    def field(e: ir.Expr, path: List[String]) =
      path.foldLeft(e)((e, f) => expr(t)(E.Field(ir.FieldAccess(Some(e), f))))
    def call(f: String, on: ir.Expr) = expr(t)(E.Call(ir.Call(f, Seq(on))))
    val (before, after) = (variable("before"), variable("after"))
    val afterState = field(after, List("state"))
    // The author's predicates, each lifted as `holds` lifts one: by the def it names, or as a
    // function of its own named after the word that takes it.
    val begins = stepFunction(arg, fn, word)
    val body = (word, finish) match
      case ("once", Some((_, x))) =>
        val path = keptPath(x).getOrElse(
          fail(
            t,
            "a keeps projection is a field path, such as `_.phase` or `_.activity.phase`, or a def " +
              s"whose body is one, not ${x.show}"
          )
        )
        or(
          not(call(begins, before)),
          binary(ir.Binary.Op.OP_EQ, field(afterState, path), field(before, path), t)
        )
      case ("never", None)         => not(call(begins, after))
      case ("never", Some((w, b))) =>
        or(not(call(stepFunction(b, fn, w), before)), not(call(begins, after)))
      case ("stays", None)         => or(not(call(begins, before)), call(begins, afterState))
      case ("stays", Some((w, r))) =>
        // Left-nested, as `!p(before) || p(after.state) || r(after)` parses.
        or(
          or(not(call(begins, before)), call(begins, afterState)),
          call(stepFunction(r, fn, w), after)
        )
      case _ => fail(t, s"not a claim pattern: ${t.show}")
    val transition = !(word == "never" && finish.isEmpty)
    val params =
      (if transition then Seq(ir.Param("before", Some(typeRef(state, t)))) else Nil) :+ stepParam
    functions(fn) =
      ir.Function(name = fn, position = Some(pos(t)), params = params, body = Some(body))
    val p = ir.Property(
      machine = m,
      name = name,
      position = Some(pos(t)),
      holds = fn,
      transition = transition
    )
    register(properties, (m, name), p, t, s"Property $name of $m")
    Decl.Claim(claim(m, name))

  // The step a Property's function reads, its parameter `after`.
  private def stepParam = ir.Param("after", Some(named(stepType)))

  /**
   * The fields a `keeps` projection reads: a lambda's field path, or the one of the def it names,
   * forwards to or is bound to, whose body is a field path over its one parameter.
   */
  private def keptPath(x: Term): Option[List[String]] =
    def defPath(sym: Symbol): Option[List[String]] = defs.get(sym) match
      case Some(DefDef(_, List(TermParamClause(List(p))), _, Some(body))) => fieldPath(p, body)
      case _                                                              => None
    x match
      case Typed(e, _)                                 => keptPath(e)
      case Inlined(_, Nil, e)                          => keptPath(e)
      case r: Ref if boundFunctions.contains(r.symbol) => defPath(boundFunctions(r.symbol))
      case r: Ref if isFunction(r.symbol)              => defPath(r.symbol)
      case _                                           =>
        lambda(x).flatMap {
          case (List(p), body) =>
            fieldPath(p, body).orElse(body match
              case Apply(target, List(a: Ident))
                  if a.symbol == p.symbol && isFunction(target.symbol) =>
                defPath(target.symbol)
              case Apply(Select(f: Ref, "apply"), List(a: Ident))
                  if a.symbol == p.symbol && boundFunctions.contains(f.symbol) =>
                defPath(boundFunctions(f.symbol))
              case _ => None)
          case _ => None
        }

  /** The outcome a `given Accepted[O] = Accepted(o)` names: `o`. */
  private def outcomeOf(accepted: Term, form: String): ir.Expr =
    val declared = accepted match
      case r: Ref =>
        defs.get(resolveSymbol(r)) match
          case Some(ValDef(_, _, Some(rhs)))      => Some(rhs)
          case Some(DefDef(_, Nil, _, Some(rhs))) => Some(rhs)
          case _                                  => None
      case other => Some(other)
    declared.map(arguments) match
      case Some(Apply(fn, List(outcome)))
          if fn.symbol.name == "apply" &&
            fn.symbol.owner.companionClass.fullName == "umpire.Accepted" =>
        lift(outcome)
      case _ =>
        fail(
          accepted,
          s"$form answers the outcome a `given Accepted[O] = Accepted(o)` of the lifted sources names, " +
            s"not ${accepted.show}"
        )
