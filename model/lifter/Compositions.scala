package umpire.lift

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir

private[lift] trait Compositions:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Compositions

  /** `"field" -> value`. */
  def arrow(t: Term): (Term, Term) = t match
    case Apply(
          TypeApply(Select(Apply(TypeApply(Ident("ArrowAssoc"), _), List(k)), "->"), _),
          List(v)
        ) =>
      (k, v)
    case other => fail(other, s"expected `\"member\" -> value`, not ${other.show}")

  def compositionOf(sym: Symbol, at: Tree): ir.Composition =
    compositions.getOrElseUpdate(
      sym.fullName,
      composition(valDef(sym, at, "a composition").rhs.get)
    )

  /**
   * A composition, from `compose[S](family, name)(members*)` and the syncs, ends and replacements
   * chained onto it.
   */
  def composition(rhs: Term): ir.Composition =
    val replaced = mutable.ArrayBuffer.empty[(String, String)]
    def move(t: Term): ir.SyncMove =
      val (member, a) = arrow(t)
      ir.SyncMove(constString(member), actions(action(a)).name)
    def walk(t: Term): ir.Composition = t match
      case Apply(Select(inner, "sync"), List(name, first, second)) =>
        walk(inner).addSyncs(ir.Sync(constString(name), Some(move(first)), Some(move(second))))
      case Apply(Select(inner, "ends"), List(p))                 => walk(inner).withEnds(lift(p))
      case Apply(Select(inner, "replaces"), List(field, opaque)) =>
        val c = walk(inner)
        replaced += constString(field) -> machineOf(resolveSymbol(opaque), opaque).name
        c
      case Apply(
            Apply(Apply(TypeApply(Ident("compose"), List(s)), List(family, name)), List(members)),
            _
          ) =>
        ir.Composition(
          position = Some(pos(rhs)),
          family = constString(family),
          name = constString(name),
          stateType = typeRef(s.tpe, t).getNamed,
          members = varargs(members).map { m =>
            val (field, member) = arrow(m)
            ir.Member(constString(field), machineOf(resolveSymbol(member), member).name)
          }
        )
      case other => fail(other, s"not a part of a composition declaration: ${other.show}")
    val c = walk(rhs)
    val fields = c.members.map(_.field)
    for s <- c.syncs; m <- Seq(s.getFirst, s.getSecond) if !fields.contains(m.member) do
      fail(rhs, s"sync ${s.name} names ${m.member}, which is not a member of ${c.name}")
    c.withMembers(replaced.foldLeft(c.members) { case (members, (field, opaque)) =>
      val i = fields.indexOf(field)
      if i < 0 then fail(rhs, s"$field replaces $opaque, and no member fills $field")
      val member = members(i).machine
      if !machineNamed(member).exists(_.getRefines.product == opaque) then
        fail(rhs, s"$field replaces $opaque, and its member $member does not refine $opaque")
      members.updated(i, members(i).withReplaces(opaque))
    })
