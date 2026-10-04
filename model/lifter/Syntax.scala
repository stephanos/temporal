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
   * `disabled` as `Nil`, `x.in(a, b)` as `List(a, b).contains(x)`, `a implies b` as `!a || b` and
   * `after.records(f)` as `after.facts.contains(f)`.
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
    case Some(("records", List(List(after), List(fact)))) =>
      val facts = expr(t)(E.Field(ir.FieldAccess(Some(lift(after)), "facts")))
      binary(ir.Binary.Op.OP_CONTAINS, lift(fact), facts, t)
    case _ => fail(t, s"outside the liftable subset: ${t.show}")

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
