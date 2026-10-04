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
  def named(name: String): ir.TypeRef = ir.TypeRef(ir.TypeRef.Ref.Named(name))
  def range(low: Long, high: Long): ir.TypeRef =
    ir.TypeRef(ir.TypeRef.Ref.IntRange(ir.IntRange(low = low, high = high)))

  def typeRef(declared: TypeRepr, at: Tree, owner: String = ""): ir.TypeRef =
    // A type parameter of a declaring function reads as the type its call applies it to.
    val tpe = instantiated(declared)
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
      range(lo, hi)
    else
      val t = tpe.dealias.widen
      val sym = t.typeSymbol
      if sym == defn.BooleanClass then ir.TypeRef(ir.TypeRef.Ref.Bool(ir.Empty()))
      else if sym == defn.IntClass && owner.isEmpty then ir.TypeRef(ir.TypeRef.Ref.Int(ir.Empty()))
      else if sym == defn.IntClass then
        val (lo, hi) = intRanges.getOrElse(
          owner,
          fail(
            at,
            s"an Int field of $owner has no range: give the state a " +
              "`given Finite[Int] = Finite.upTo(...)` where its Finite is derived"
          )
        )
        range(lo, hi)
      else if isList(sym) then ir.TypeRef(ir.TypeRef.Ref.List(typeRef(t.typeArgs.head, at, owner)))
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
        ir.TypeRef(ir.TypeRef.Ref.Channel(channelOf(channel, at)))
      else if sym.fullName == stepType then named(stepType)
      else
        declareType(sym, at)
        named(sym.fullName)

  def declareType(sym: Symbol, at: Tree): Unit =
    if !types.contains(sym.fullName) && sym != defn.NothingClass then
      types(sym.fullName) = ir.Type.defaultInstance // placeholder against recursion
      val position = Some(scala.util.Try(pos(sym.tree)).getOrElse(pos(at)))
      def field(n: String, ft: TypeRepr): ir.Field =
        // A list has no bound, so no finite type has one as a field.
        if isList(ft.dealias.widen.typeSymbol) then
          fail(
            scala.util.Try(sym.tree).getOrElse(at),
            s"${sym.fullName}.$n is a list, which has no bound: a state " +
              "holds messages in a channel's Inbox"
          )
        ir.Field(name = n, `type` = Some(typeRef(ft, at, sym.fullName)))
      def fields(cls: Symbol): Seq[ir.Field] = fieldTypes(cls).map(field)
      val shape =
        if sym.flags.is(Flags.Enum) then
          val cases = sym.children.map(c =>
            ir.Case(name = c.name, fields = if c.isClassDef then fields(c) else Nil)
          )
          ir.Type.Shape.Enum(ir.Enum(cases))
        else if sym.flags.is(Flags.Case) then ir.Type.Shape.Record(ir.Record(fields(sym)))
        else
          fail(
            at,
            s"${sym.fullName} is neither an enum nor a case class, so it has no finite catalog"
          )
      types(sym.fullName) = ir.Type(name = sym.fullName, position = position, shape = shape)

  /**
   * An optional value's type: an enum of `None` and `Some(value)`, one per type of value, named
   * after it, so a value keys as `None` or `Some-<key>` as the framework keys it.
   */
  def optionType(arg: TypeRepr, at: Tree): String =
    val value = typeRef(arg, at)
    val argName = value.ref match
      case ir.TypeRef.Ref.Named(n)    => n
      case ir.TypeRef.Ref.Bool(_)     => "scala.Boolean"
      case ir.TypeRef.Ref.IntRange(_) => arg.widen.typeSymbol.fullName
      case ir.TypeRef.Ref.Int(_) | ir.TypeRef.Ref.List(_) | ir.TypeRef.Ref.Channel(_) |
          ir.TypeRef.Ref.Empty =>
        fail(
          at,
          s"an optional ${arg.show} has no finite catalog: give its value an enum, a record, a " +
            "Boolean or an opaque type with a range"
        )
    val name = s"scala.Option[$argName]"
    if !types.contains(name) then
      val cases = Seq(
        ir.Case(name = "None"),
        ir.Case(name = "Some", fields = Seq(ir.Field(name = "value", `type` = Some(value))))
      )
      types(name) =
        ir.Type(name = name, position = Some(pos(at)), shape = ir.Type.Shape.Enum(ir.Enum(cases)))
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
