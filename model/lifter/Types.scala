package umpire.lift

import io.temporal.server.api.umpire.v1 as ir

private[lift] trait Types:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Types

  def isEnumCase(s: Symbol): Boolean = s.flags.is(Flags.Enum) && s.flags.is(Flags.Case)

  /** The enum a case belongs to: the class whose companion object declares the case. */
  def enumOf(caseSym: Symbol): Symbol = caseSym.owner.companionClass
  def fieldTypes(cls: Symbol): List[(String, TypeRepr)] =
    cls.caseFields.map(f => f.name -> cls.typeRef.memberType(f).widen)
  def isList(sym: Symbol): Boolean = sym == defn.RepeatedParamClass ||
    sym.fullName == "scala.collection.immutable.List" || sym.fullName == "scala.collection.immutable.Seq"
  def isNamed(tpe: TypeRepr, name: String): Boolean =
    tpe.widen.dealias.typeSymbol.fullName == name
  def messageType(inbox: TypeRepr): String =
    inbox.widen.dealias.typeArgs.head.dealias.typeSymbol.fullName
  def named(name: String): ir.TypeRef = ir.TypeRef.newBuilder().setNamed(name).build()

  def typeRef(tpe: TypeRepr, at: Tree, owner: String = ""): ir.TypeRef =
    // An opaque type is read before its alias is resolved: its range is its own.
    val opaque = tpe.widen.typeSymbol
    if opaque.flags.is(Flags.Opaque) then
      val (lo, hi) = opaqueRanges.getOrElse(
        opaque.fullName,
        fail(
          at,
          s"${opaque.fullName} has no range: give it a " +
            "`given Finite[...] = Finite.upTo(...)` beside the type"
        )
      )
      ir.TypeRef.newBuilder().setIntRange(ir.IntRange.newBuilder().setLow(lo).setHigh(hi)).build()
    else
      val t = tpe.dealias.widen
      val sym = t.typeSymbol
      if sym == defn.BooleanClass then
        ir.TypeRef.newBuilder().setBool(ir.Empty.getDefaultInstance).build()
      else if sym == defn.IntClass && owner.isEmpty then
        ir.TypeRef.newBuilder().setInt(ir.Empty.getDefaultInstance).build()
      else if sym == defn.IntClass then
        val (lo, hi) = intRanges.getOrElse(
          owner,
          fail(
            at,
            s"an Int field of $owner has no range: give the state a " +
              "`given Finite[Int] = Finite.upTo(...)` where its Finite is derived"
          )
        )
        ir.TypeRef
          .newBuilder()
          .setIntRange(ir.IntRange.newBuilder().setLow(lo).setHigh(hi))
          .build()
      else if isList(sym) then
        ir.TypeRef.newBuilder().setList(typeRef(t.typeArgs.head, at, owner)).build()
      else if sym.fullName == "scala.Option" then named(optionType(t.typeArgs.head, at))
      else if sym.fullName == "umpire.Inbox" then
        if owner.isEmpty then
          fail(at, "an Inbox is held in a state field; a function reads it from the state")
        val channel = channelFields.getOrElse(
          (owner, messageType(t)),
          fail(
            at,
            s"an Inbox field of $owner names no " +
              "channel: give the state a `given Finite[Inbox[M]] = <channel>.contents` where its Finite is declared"
          )
        )
        ir.TypeRef.newBuilder().setChannel(channelOf(channel, at)).build()
      else if sym.fullName == stepType then named(stepType)
      else
        declareType(sym, at)
        named(sym.fullName)

  def declareType(sym: Symbol, at: Tree): Unit =
    if !types.contains(sym.fullName) && sym != defn.NothingClass then
      types(sym.fullName) = ir.Type.getDefaultInstance // placeholder against recursion
      val b = ir.Type
        .newBuilder()
        .setName(sym.fullName)
        .setPosition(scala.util.Try(pos(sym.tree)).getOrElse(pos(at)))
      def field(n: String, ft: TypeRepr): ir.Field =
        // A list has no bound, so no finite type has one as a field.
        if isList(ft.dealias.widen.typeSymbol) then
          fail(
            scala.util.Try(sym.tree).getOrElse(at),
            s"${sym.fullName}.$n is a list, which has no bound: a state " +
              "holds messages in a channel's Inbox"
          )
        ir.Field.newBuilder().setName(n).setType(typeRef(ft, at, sym.fullName)).build()
      if sym.flags.is(Flags.Enum) then
        val e = ir.Enum.newBuilder()
        for c <- sym.children do
          val cb = ir.Case.newBuilder().setName(c.name)
          if c.isClassDef then for (n, ft) <- fieldTypes(c) do cb.addFields(field(n, ft))
          e.addCases(cb)
        b.setEnum(e)
      else if sym.flags.is(Flags.Case) then
        val r = ir.Record.newBuilder()
        for (n, ft) <- fieldTypes(sym) do r.addFields(field(n, ft))
        b.setRecord(r)
      else
        fail(
          at,
          s"${sym.fullName} is neither an enum nor a case class, so it has no finite catalog"
        )
      types(sym.fullName) = b.build()

  /**
   * An optional value's type: an enum of `None` and `Some(value)`, one per type of value, named
   * after it, so a value keys as `None` or `Some-<key>` as the framework keys it.
   */
  def optionType(arg: TypeRepr, at: Tree): String =
    val value = typeRef(arg, at)
    val argName = value.getRefCase match
      case ir.TypeRef.RefCase.NAMED     => value.getNamed
      case ir.TypeRef.RefCase.BOOL      => "scala.Boolean"
      case ir.TypeRef.RefCase.INT_RANGE => arg.widen.typeSymbol.fullName
      case _                            =>
        fail(
          at,
          s"an optional ${arg.show} has no finite catalog: give its value an enum, a record, a " +
            "Boolean or an opaque type with a range"
        )
    val name = s"scala.Option[$argName]"
    if !types.contains(name) then
      types(name) = ir.Type
        .newBuilder()
        .setName(name)
        .setPosition(pos(at))
        .setEnum(
          ir.Enum
            .newBuilder()
            .addCases(ir.Case.newBuilder().setName("None"))
            .addCases(
              ir.Case
                .newBuilder()
                .setName("Some")
                .addFields(ir.Field.newBuilder().setName("value").setType(value))
            )
        )
        .build()
    name

  /**
   * The value type of an optional value's type: `Option[T]`, or `Some[T]`, also inside a union such
   * as the `Some[T] | None` of a match with both.
   */
  def optionArg(tpe: TypeRepr, at: Tree): TypeRepr =
    def arg(t: TypeRepr): Option[TypeRepr] = t.widen.dealias match
      case AppliedType(o, List(a))
          if o.typeSymbol.fullName == "scala.Option" || o.typeSymbol.fullName == "scala.Some" =>
        Some(a)
      case OrType(l, r) => arg(l).orElse(arg(r))
      case _            => None
    arg(tpe).getOrElse(
      fail(
        at,
        s"None has no type here: ${tpe.widen.show} is not optional, so write `None: Option[T]`"
      )
    )

  // The `given Finite[S]` blocks: an Int field's range is the `Finite.upTo(bound)` in scope there.
  // An Inbox field's channel is the one whose contents are in scope there, and an opaque type's own
  // given is its range.
  def finiteOf(tpt: TypeTree): Option[TypeRepr] =
    if tpt.tpe.typeSymbol.name == "Finite" then tpt.tpe.typeArgs.headOption else None
  def readFinites(): Unit =
    for d <- defs.values do
      d match
        case ValDef(_, tpt, Some(Block(stats, _))) if finiteOf(tpt).nonEmpty =>
          val state = finiteOf(tpt).get.dealias.typeSymbol.fullName
          for case ValDef(_, _, Some(Apply(Select(_, "upTo"), List(bound)))) <- stats do
            intRanges(state) = (0L, constInt(bound))
          for
            case ValDef(_, inbox, Some(Select(channel, "contents"))) <- stats; m <- finiteOf(inbox)
          do channelFields((state, messageType(m))) = resolveSymbol(channel)
        case ValDef(_, tpt, Some(Apply(Select(_, "upTo"), List(bound))))
            if finiteOf(tpt).exists(_.typeSymbol.flags.is(Flags.Opaque)) =>
          opaqueRanges(finiteOf(tpt).get.typeSymbol.fullName) = (0L, constInt(bound))
        case _ => ()
