package umpire.lift

import scala.collection.mutable
import scala.jdk.CollectionConverters.*
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
    val b = ir.Composition.newBuilder().setPosition(pos(rhs))
    val replaced = mutable.ArrayBuffer.empty[(String, String)]
    def move(t: Term): ir.SyncMove =
      val (member, a) = arrow(t)
      ir.SyncMove
        .newBuilder()
        .setMember(constString(member))
        .setAction(actions(action(a)).getName)
        .build()
    def walk(t: Term): Unit = t match
      case Apply(Select(inner, "sync"), List(name, first, second)) =>
        walk(inner)
        b.addSyncs(
          ir.Sync
            .newBuilder()
            .setName(constString(name))
            .setFirst(move(first))
            .setSecond(move(second))
        )
      case Apply(Select(inner, "ends"), List(p))                 => walk(inner); b.setEnds(lift(p))
      case Apply(Select(inner, "replaces"), List(field, opaque)) =>
        walk(inner)
        replaced += constString(field) -> machineOf(resolveSymbol(opaque), opaque).getName
      case Apply(
            Apply(Apply(TypeApply(Ident("compose"), List(s)), List(family, name)), List(members)),
            _
          ) =>
        b.setFamily(constString(family))
          .setName(constString(name))
          .setStateType(typeRef(s.tpe, t).getNamed)
        for m <- varargs(members) do
          val (field, member) = arrow(m)
          b.addMembers(
            ir.Member
              .newBuilder()
              .setField(constString(field))
              .setMachine(machineOf(resolveSymbol(member), member).getName)
          )
      case other => fail(other, s"not a part of a composition declaration: ${other.show}")
    walk(rhs)
    val fields = b.getMembersList.asScala.map(_.getField)
    for
      s <- b.getSyncsList.asScala; m <- Seq(s.getFirst, s.getSecond)
      if !fields.contains(m.getMember)
    do fail(rhs, s"sync ${s.getName} names ${m.getMember}, which is not a member of ${b.getName}")
    for (field, opaque) <- replaced do
      val i = fields.indexOf(field)
      if i < 0 then fail(rhs, s"$field replaces $opaque, and no member fills $field")
      val member = b.getMembers(i).getMachine
      if !machineNamed(member).exists(_.getRefines.getProduct == opaque) then
        fail(rhs, s"$field replaces $opaque, and its member $member does not refine $opaque")
      b.setMembers(i, b.getMembers(i).toBuilder.setReplaces(opaque))
    b.build()
