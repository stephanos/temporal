package framework.reify

import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir

private[framework] trait Types:
  self: BodyEngine =>
  import ctx.*
  import ctx.given
  import Equality.given
  import ctx.quotes.reflect.*

  // ### Types

  def isEnumCase(s: Symbol): Boolean = s.flags.is(Flags.Enum) && s.flags.is(Flags.Case)

  // The enum a case belongs to: the class whose companion object declares the case.
  def enumOf(caseSym: Symbol): Symbol = caseSym.owner.companionClass
  def fieldTypes(cls: Symbol): List[(String, TypeRepr)] =
    cls.caseFields.map(f => f.name -> cls.typeRef.memberType(f).widen)
  def isList(sym: Symbol): Boolean =
    sym == defn.RepeatedParamClass ||
      sym.fullName == "scala.collection.immutable.List" || sym.fullName == "scala.collection.immutable.Seq"
  def isNamed(tpe: TypeRepr, name: String): Boolean =
    tpe.widen.dealias.typeSymbol.fullName == name
  def messageType(inbox: TypeRepr): String =
    def identity(tpe: TypeRepr): String =
      tpe.dealias.widenTermRefByName match
        case ConstantType(IntConstant(n)) => n.toString
        case _                            =>
          val t = tpe.widen.dealias
          val args = t.typeArgs
          val name = t.typeSymbol.fullName
          if args.isEmpty then name else s"$name[${args.map(identity).mkString(",")}]"
    identity(inbox.widen.dealias.typeArgs.head)
  def named(name: String): ir.TypeRef = ir.TypeRef(ir.TypeRef.Ref.Named(name))
  def range(low: Long, high: Long): ir.TypeRef =
    ir.TypeRef(ir.TypeRef.Ref.IntRange(ir.IntRange(low = low, high = high)))

  def typeRef(declared: TypeRepr, at: Tree, owner: String = ""): ir.TypeRef =
    // A type parameter of a declaring function reads as the type its call applies it to.
    val tpe = instantiated(declared)
    // An opaque type is read before its alias is resolved: its range is its own. A transparent
    // alias, such as a named UpTo, is resolved first.
    val opaque = transparent(tpe).typeSymbol
    if opaque.fullName == upToType then range(0, upToBound(tpe, at))
    else if opaque.flags.is(Flags.Opaque) then
      val (lo, hi) = opaqueRange(opaque).getOrElse(
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
      if sym == defn.BooleanClass then ir.TypeRef(ir.TypeRef.Ref.Bool(Empty()))
      else if sym == defn.IntClass && owner.isEmpty then ir.TypeRef(ir.TypeRef.Ref.Int(Empty()))
      else if sym == defn.IntClass then
        fail(
          at,
          s"$owner.${at.symbol.name} is an Int field with no declared bound: write its type as UpTo[N] and " +
            "its values as UpTo(n)"
        )
      else if isList(sym) then ir.TypeRef(ir.TypeRef.Ref.List(typeRef(t.typeArgs.head, at, owner)))
      else if sym.fullName == "scala.Option" then named(optionType(t.typeArgs.head, at))
      else if sym.fullName == "framework.dsl.Inbox" then inboxRef(t, at, owner)
      else if sym.fullName == stepType then named(stepType)
      else
        declareType(sym, at)
        named(irTypeName(sym))

  def inboxRef(tpe: TypeRepr, at: Tree, owner: String): ir.TypeRef =
    if owner.isEmpty then
      fail(at, "an Inbox is held in a state field; a function reads it from the state")
    val channel = channelFields.getOrElse(
      (owner, messageType(tpe)),
      fail(
        at,
        s"an Inbox field of $owner names no " +
          "channel: give the state a `given Finite[Inbox[M]] = <channel>.contents` where its Finite is declared"
      )
    )
    ir.TypeRef(ir.TypeRef.Ref.Channel(channelOf(channel, at)))

  // A type with its transparent aliases resolved, stopping at an opaque type.
  def transparent(tpe: TypeRepr): TypeRepr =
    val shown = tpe.widen
    val sym = shown.typeSymbol
    if sym.isAliasType && !sym.flags.is(Flags.Opaque) then shown.dealias else shown

  // The framework's bounded counter, `UpTo[N]`: the values `0..N` of its field.
  val upToType = "framework.dsl.Domain$package$.UpTo"

  // The bound `N` of an `UpTo[N]`: a constant, at least 0.
  def upToBound(tpe: TypeRepr, at: Tree): Long =
    tpe.widen.dealias.typeArgs match
      case List(n) =>
        n.dealias.widenTermRefByName match
          case ConstantType(IntConstant(bound)) if bound >= 0 => bound.toLong
          case ConstantType(IntConstant(bound))               =>
            fail(
              at,
              s"UpTo[$bound] has no values: a counter counts from 0 to a bound of at least 0"
            )
          case other =>
            fail(at, s"UpTo bounds a counter by a constant, such as UpTo[2], not ${other.show}")
      case _ => fail(at, s"UpTo bounds a counter by a constant, such as UpTo[2], not ${tpe.show}")

  def declareType(sym: Symbol, at: Tree): Unit =
    lazy val name = irTypeName(sym)
    if sym != defn.NothingClass && !types.contains(name)
    then
      types(name) = ir.Type.defaultInstance // placeholder against recursion
      val position = Some(sourceTree(sym).map(pos).getOrElse(pos(at)))
      def field(cls: Symbol, n: String, ft: TypeRepr): ir.Field =
        // A list has no bound, so no finite type has one as a field.
        if isList(ft.dealias.widen.typeSymbol) then
          fail(
            sourceTree(sym).getOrElse(at),
            s"${sym.fullName}.$n is a list, which has no bound: a state " +
              "holds messages in a channel's Inbox"
          )
        val unbounded = ft.dealias.widen.typeSymbol == defn.IntClass && !ft.widen.typeSymbol.flags
          .is(Flags.Opaque)
        val declared =
          if unbounded
          then
            val candidates = cls.primaryConstructor.paramSymss.flatten ++ cls.caseFields
            candidates
              .filter(_.name == n)
              .flatMap(sourceTree)
              .find(_.pos.startLine >= 0)
              .getOrElse(at)
          else at
        val owner = if unbounded then cls.fullName else sym.fullName
        ir.Field(name = n, `type` = Some(typeRef(ft, declared, owner)))
      def fields(cls: Symbol): Seq[ir.Field] = fieldTypes(cls).map((n, ft) => field(cls, n, ft))
      val shape =
        if sym.flags.is(Flags.Enum) then
          checkRoles(sym, at)
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
      types(name) = ir.Type(name = name, position = position, shape = shape)

  // An optional value's type: an enum of `None` and `Some(value)`, one per type of value, named
  // after it, so a value keys as `None` or `Some-<key>` as the framework keys it.
  def optionType(arg: TypeRepr, at: Tree): String =
    val value = typeRef(arg, at)
    val argName = value.ref match
      case ir.TypeRef.Ref.Named(n)    => n
      case ir.TypeRef.Ref.Bool(_)     => "scala.Boolean"
      case ir.TypeRef.Ref.IntRange(r) =>
        val name = transparent(arg).typeSymbol.fullName
        if name == upToType then s"$name[${r.high}]" else name
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

  // The value type of an optional value's type: `Option[T]`, or `Some[T]`, also inside a union such
  // as the `Some[T] | None` of a match with both.
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

  // An Inbox field's channel comes from its state's given; a named opaque type's given is its range.
  def finiteOf(tpt: TypeTree): Option[TypeRepr] =
    if tpt.tpe.typeSymbol.name == "Finite" then tpt.tpe.typeArgs.headOption
    else None

  private lazy val schemaRetention =
    Symbol.requiredClass("framework.dsl.Finite").methodMember("retainSchema")

  private def finiteValue(value: Term): Term = value match
    case Apply(fn @ Select(receiver, _), _) if schemaRetention.contains(fn.symbol) =>
      finiteValue(receiver)
    case _ => value

  def readFinites(): Unit =
    for
      case ValDef(_, tpt, Some(rhs)) <- defs.values
      domain <- finiteOf(tpt)
    do
      finiteValue(rhs) match
        case Block(stats, _) =>
          val state = domain.dealias.typeSymbol.fullName
          for
            case ValDef(_, inbox, Some(value)) <- stats
            m <- finiteOf(inbox)
          do
            finiteValue(value) match
              case Select(channel, "contents") =>
                channelFields((state, messageType(m))) = resolveSymbol(channel)
              case _ => ()
        case Apply(Select(_, "upTo"), List(bound)) if domain.typeSymbol.flags.is(Flags.Opaque) =>
          opaqueRanges(domain.typeSymbol.fullName) = (0L, constInt(bound))
        case _ => ()
